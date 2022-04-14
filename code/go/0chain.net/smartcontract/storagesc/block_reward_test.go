package storagesc

import (
	"math/rand"
	"os"
	"strconv"
	"testing"
	"time"

	"0chain.net/chaincore/chain/state"
	bstate "0chain.net/chaincore/state"
	"0chain.net/core/util"
	"0chain.net/smartcontract/partitions"
	"0chain.net/smartcontract/stakepool"
	"github.com/stretchr/testify/require"
)

func TestStorageSmartContract_blobberBlockRewards(t *testing.T) {
	balances := newTestBalances(t, true)
	type params struct {
		numBlobbers       int
		wp                []bstate.Balance
		rp                []bstate.Balance
		totalData         []float64
		dataRead          []float64
		successChallenges []int
		delegatesBal      [][]bstate.Balance
		serviceCharge     []float64
	}

	type result struct {
		blobberRewards          []bstate.Balance
		blobberDelegatesRewards [][]bstate.Balance
	}

	setupRewards := func(t *testing.T, p params, sc *StorageSmartContract) {
		conf := setConfig(t, balances)
		allBR, err := getActivePassedBlobberRewardsPartitions(balances, conf.BlockReward.TriggerPeriod)
		require.NoError(t, err)
		for i := 0; i < p.numBlobbers; i++ {
			bID := "blobber" + strconv.Itoa(i)
			_, err = allBR.AddItem(balances, &BlobberRewardNode{
				ID:                bID,
				SuccessChallenges: p.successChallenges[i],
				WritePrice:        p.wp[i],
				ReadPrice:         p.rp[i],
				TotalData:         p.totalData[i],
				DataRead:          p.dataRead[i],
			})
			require.NoError(t, err)

			sp := newStakePool()
			sp.Settings.DelegateWallet = bID
			sp.Settings.ServiceCharge = p.serviceCharge[i]
			for j, bal := range p.delegatesBal[i] {
				dp := new(stakepool.DelegatePool)
				dp.Balance = bal
				sp.Pools["delegate"+strconv.Itoa(j)] = dp
			}
			_, err = balances.InsertTrieNode(stakePoolKey(sc.ID, bID), sp)
			require.NoError(t, err)
		}
		err = allBR.Save(balances)
		require.NoError(t, err)
	}

	compareResult := func(t *testing.T, p params, r result, ssc *StorageSmartContract) {
		conf, err := ssc.getConfig(balances, false)
		require.NoError(t, err)
		for i := 0; i < p.numBlobbers; i++ {
			bID := "blobber" + strconv.Itoa(i)
			sp, err := ssc.getStakePool(bID, balances)
			require.NoError(t, err)

			require.EqualValues(t, r.blobberRewards[i], sp.Reward)

			for j := range p.delegatesBal[i] {
				key := "delegate" + strconv.Itoa(j)
				require.EqualValues(t, r.blobberDelegatesRewards[i][j], sp.Pools[key].Reward)
			}
		}
		_, err = balances.DeleteTrieNode(
			BlobberRewardKey(
				GetPreviousRewardRound(balances.GetBlock().Round, conf.BlockReward.TriggerPeriod)),
		)
		require.NoError(t, err)
	}

	tests := []struct {
		name    string
		wantErr bool
		params  params
		result  result
	}{
		{
			name: "1_blobber",
			params: params{
				numBlobbers:       1,
				wp:                []bstate.Balance{2},
				rp:                []bstate.Balance{1},
				totalData:         []float64{10},
				dataRead:          []float64{2},
				successChallenges: []int{10},
				delegatesBal:      [][]bstate.Balance{{1, 0, 3}},
				serviceCharge:     []float64{.1},
			},
			result: result{
				blobberRewards:          []bstate.Balance{50},
				blobberDelegatesRewards: [][]bstate.Balance{{112, 0, 337}},
			},
		},
		{
			name: "2_blobber",
			params: params{
				numBlobbers:       2,
				wp:                []bstate.Balance{3, 1},
				rp:                []bstate.Balance{1, 0},
				totalData:         []float64{10, 50},
				dataRead:          []float64{2, 15},
				successChallenges: []int{5, 2},
				delegatesBal:      [][]bstate.Balance{{1, 0, 3}, {1, 6, 3}},
				serviceCharge:     []float64{.1, .1},
			},
			result: result{
				blobberRewards:          []bstate.Balance{15, 34},
				blobberDelegatesRewards: [][]bstate.Balance{{35, 0, 107}, {30, 183, 91}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ssc := newTestStorageSC()
			setupRewards(t, tt.params, ssc)
			err := ssc.blobberBlockRewards(balances)
			require.EqualValues(t, tt.wantErr, err != nil)
			compareResult(t, tt.params, tt.result, ssc)

		})
	}
}

func prepareState(n int) (state.StateContextI, func()) {
	dir, err := os.MkdirTemp("", "part_state")
	if err != nil {
		panic(err)
	}

	pdb, _ := util.NewPNodeDB(dir, dir+"/log")

	clean := func() {
		pdb.Close()
		_ = os.RemoveAll(dir)
	}

	mpt := util.NewMerklePatriciaTrie(pdb, 0, nil)
	sctx := state.NewStateContext(nil,
		mpt, nil, nil, nil,
		nil, nil, nil)

	part, err := partitions.CreateIfNotExists(sctx, "brn_test", n)
	if err != nil {
		panic(err)
	}

	for i := 0; i < n; i++ {
		br := BlobberRewardNode{
			ID:                strconv.Itoa(i),
			SuccessChallenges: i,
			WritePrice:        bstate.Balance(i),
			ReadPrice:         bstate.Balance(i),
			TotalData:         float64(i * 100),
			DataRead:          float64(i),
		}
		//bs[i] = b
		if _, err := part.AddItem(sctx, &br); err != nil {
			panic(err)
		}
	}

	if err := part.Save(sctx); err != nil {
		panic(err)
	}

	return sctx, clean
}

func BenchmarkPartitionsGetItem(b *testing.B) {
	ps, clean := prepareState(100)
	defer clean()

	part, err := partitions.GetPartitions(ps, "brn_test")
	require.NoError(b, err)

	id := strconv.Itoa(10)
	for i := 0; i < b.N; i++ {
		var br BlobberRewardNode
		_ = part.GetItem(ps, 0, id, &br)
	}
}

func BenchmarkGetRandomItems(b *testing.B) {
	seed := rand.NewSource(time.Now().Unix())
	r := rand.New(seed)
	ps, clean := prepareState(100)
	defer clean()

	part, err := partitions.GetPartitions(ps, "brn_test")
	require.NoError(b, err)
	ids := make([]string, 100)
	for i := 0; i < 100; i++ {
		ids[i] = strconv.Itoa(i)
	}

	for i := 0; i < b.N; i++ {
		var bs []BlobberRewardNode
		_ = part.GetRandomItems(ps, r, &bs)
	}
}

func BenchmarkGetUpdateItem(b *testing.B) {
	ps, clean := prepareState(100)
	defer clean()

	part, err := partitions.GetPartitions(ps, "brn_test")
	require.NoError(b, err)

	for i := 0; i < b.N; i++ {
		_ = part.UpdateItem(ps, 0, &BlobberRewardNode{ID: "100"})
	}

	_ = part.Save(ps)
}
