package storagesc

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"strconv"
	"testing"
	"time"

	"0chain.net/smartcontract/stakepool/spenum"

	"github.com/0chain/common/core/currency"

	"0chain.net/chaincore/chain/state"
	"0chain.net/smartcontract/partitions"
	"0chain.net/smartcontract/stakepool"
	"github.com/0chain/common/core/util"
	"github.com/stretchr/testify/require"
)

func TestStorageSmartContract_blobberBlockRewards(t *testing.T) {
	type params struct {
		numBlobbers       int
		wp                []currency.Coin
		rp                []currency.Coin
		totalData         []float64
		dataRead          []float64
		successChallenges []int
		delegatesBal      [][]currency.Coin
		serviceCharge     []float64
	}

	type result struct {
		blobberRewards          []currency.Coin
		blobberDelegatesRewards [][]currency.Coin
	}

	setupRewards := func(t *testing.T, p params, balances state.StateContextI, sc *StorageSmartContract) state.StateContextI {
		conf := setConfig(t, balances)
		allBR, err := getActivePassedBlobberRewardsPartitions(balances, conf.BlockReward.TriggerPeriod)
		require.NoError(t, err)
		for i := 0; i < p.numBlobbers; i++ {
			bID := "blobber" + strconv.Itoa(i)
			err = allBR.Add(balances, &BlobberRewardNode{
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
			sp.Settings.ServiceChargeRatio = p.serviceCharge[i]
			for j, bal := range p.delegatesBal[i] {
				dID := "delegate" + strconv.Itoa(j)
				dp := new(stakepool.DelegatePool)
				dp.Balance = bal
				dp.DelegateID = dID
				sp.Pools[dID] = dp
			}
			_, err = balances.InsertTrieNode(stakePoolKey(spenum.Blobber, bID), sp)
			require.NoError(t, err)
		}
		err = allBR.Save(balances)
		require.NoError(t, err)
		return balances
	}

	compareResult := func(t *testing.T, p params, r result, balances state.StateContextI, ssc *StorageSmartContract) {
		conf, err := ssc.getConfig(balances, false)
		require.NoError(t, err)
		for i := 0; i < p.numBlobbers; i++ {
			bID := "blobber" + strconv.Itoa(i)
			sp, err := ssc.getStakePool(spenum.Blobber, bID, balances)
			require.NoError(t, err)

			fmt.Println("Expected Blobber ", i, " Reward : ", r.blobberRewards[i], " vs Actual Reward : ", sp.Reward)

			//require.EqualValues(t, r.blobberRewards[i], sp.Reward)

			for j := range p.delegatesBal[i] {
				key := "delegate" + strconv.Itoa(j)
				fmt.Println("Expected Blobber ", i, " Delegate ", j, " Reward : ", r.blobberDelegatesRewards[i][j], " vs Actual Reward : ", sp.Pools[key].Reward)
				//require.EqualValues(t, r.blobberDelegatesRewards[i][j], sp.Pools[key].Reward)
			}
		}
		_, err = balances.DeleteTrieNode(
			BlobberRewardKey(
				GetPreviousRewardRound(balances.GetBlock().Round, conf.BlockReward.TriggerPeriod)),
		)
		require.NoError(t, err)

		fmt.Println("\n-------------------------------------------------------------")
	}

	var tests []struct {
		name    string
		wantErr bool
		params  params
		result  result
	}

	writePrices := [][]float64{{1, 1}, {1, 3}}
	readPrices := [][]float64{{0, 1}, {0, 0}, {1, 1}}
	totalReads := [][]float64{{1, 1}, {1, 3}}
	totalData := [][]float64{{1, 3}, {1, 1}}
	challenges := [][]float64{{1000, 1300}, {1000, 1000}}

	for _, readPrice := range readPrices {
		for _, writePrice := range writePrices {
			for _, readData := range totalReads {
				for _, writeData := range totalData {
					for _, challenge := range challenges {

						blobber1Weight := calculateWeight(writePrice[0], readPrice[0], writeData[0], readData[0], 4, challenge[0])
						blobber2Weight := calculateWeight(writePrice[1], readPrice[1], writeData[1], readData[1], 2, challenge[1])

						fmt.Println("Blobber 1 Weight : ", blobber1Weight)
						fmt.Println("Blobber 2 Weight : ", blobber2Weight)

						blobber1ExpectedReward := blobber1Weight * 500 / (blobber1Weight + blobber2Weight)
						blobber2ExpectedReward := 500 - blobber1ExpectedReward
						fmt.Println("blobber1ExpectedReward", blobber1ExpectedReward)
						fmt.Println("blobber2ExpectedReward", blobber2ExpectedReward)

						wp1, _ := currency.Float64ToCoin(writePrice[0])
						wp2, _ := currency.Float64ToCoin(writePrice[1])
						rp1, _ := currency.Float64ToCoin(readPrice[0])
						rp2, _ := currency.Float64ToCoin(readPrice[1])

						br1, _ := currency.Float64ToCoin(blobber1ExpectedReward * 0.1)
						br2, _ := currency.Float64ToCoin(blobber2ExpectedReward * 0.1)
						blobber1ExpectedReward = blobber1ExpectedReward - blobber1ExpectedReward*0.1
						blobber2ExpectedReward = blobber2ExpectedReward - blobber2ExpectedReward*0.1

						b1d1, _ := currency.Float64ToCoin(blobber1ExpectedReward * 0.25)
						b1d2, _ := currency.Float64ToCoin(blobber1ExpectedReward * 0.75)
						b2d1, _ := currency.Float64ToCoin(blobber2ExpectedReward * 0.5)
						b2d2, _ := currency.Float64ToCoin(blobber2ExpectedReward * 0.5)

						fmt.Println("br1", br1)
						fmt.Println("br2", br2)
						fmt.Println("b1d1", b1d1)
						fmt.Println("b1d2", b1d2)
						fmt.Println("b2d1", b2d1)
						fmt.Println("b2d2", b2d2)

						tests = append(tests, struct {
							name    string
							wantErr bool
							params  params
							result  result
						}{
							name: "blobber_block_rewards",
							params: params{
								numBlobbers:       2,
								wp:                []currency.Coin{wp1, wp2},
								rp:                []currency.Coin{rp1, rp2},
								totalData:         []float64{writeData[0], writeData[1]},
								dataRead:          []float64{readData[0], readData[1]},
								successChallenges: []int{int(challenge[0]), int(challenge[1])},
								delegatesBal:      [][]currency.Coin{{1, 3}, {1, 1}},
								serviceCharge:     []float64{.1, .1},
							}, result: result{
								blobberRewards:          []currency.Coin{br1, br2},
								blobberDelegatesRewards: [][]currency.Coin{{b1d1, b1d2}, {b2d1, b2d2}},
							},
						})
					}
				}
			}
		}
	}

	//tests := []struct {
	//	name    string
	//	wantErr bool
	//	params  params
	//	result  result
	//}{
	//	{
	//		name: "1_blobber",
	//		params: params{
	//			numBlobbers:       1,
	//			wp:                []currency.Coin{2},
	//			rp:                []currency.Coin{1},
	//			totalData:         []float64{10},
	//			dataRead:          []float64{2},
	//			successChallenges: []int{10},
	//			delegatesBal:      [][]currency.Coin{{1, 0, 3}},
	//			serviceCharge:     []float64{.1},
	//		},
	//		result: result{
	//			blobberRewards:          []currency.Coin{50},
	//			blobberDelegatesRewards: [][]currency.Coin{{113, 0, 337}},
	//		},
	//	},
	//	{
	//		name: "2_blobber",
	//		params: params{
	//			numBlobbers:       2,
	//			wp:                []currency.Coin{3, 1},
	//			rp:                []currency.Coin{1, 0},
	//			totalData:         []float64{10, 50},
	//			dataRead:          []float64{2, 15},
	//			successChallenges: []int{5, 2},
	//			delegatesBal:      [][]currency.Coin{{1, 0, 3}, {1, 6, 3}},
	//			serviceCharge:     []float64{.1, .1},
	//		},
	//		result: result{
	//			blobberRewards:          []currency.Coin{18, 31},
	//			blobberDelegatesRewards: [][]currency.Coin{{43, 0, 124}, {29, 170, 85}},
	//		},
	//	},
	//}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			balances := newTestBalances(t, true)
			ssc := newTestStorageSC()
			setupRewards(t, tt.params, balances, ssc)
			err := ssc.blobberBlockRewards(balances)
			require.EqualValues(t, tt.wantErr, err != nil)
			compareResult(t, tt.params, tt.result, balances, ssc)
			require.EqualValues(t, true, false)
		})
	}
}

func prepareState(n, partSize int) (state.StateContextI, func()) {
	dir, err := os.MkdirTemp("", "part_state")
	if err != nil {
		panic(err)
	}

	pdb, _ := util.NewPNodeDB(dir, dir+"/log")

	clean := func() {
		pdb.Flush()
		pdb.Close()
		_ = os.RemoveAll(dir)
	}

	mpt := util.NewMerklePatriciaTrie(pdb, 0, nil)
	sctx := state.NewStateContext(nil,
		mpt, nil, nil, nil,
		nil, nil, nil, nil)

	part, err := partitions.CreateIfNotExists(sctx, "brn_test", partSize)
	if err != nil {
		panic(err)
	}

	for i := 0; i < n; i++ {
		br := BlobberRewardNode{
			ID:                strconv.Itoa(i),
			SuccessChallenges: i,
			WritePrice:        currency.Coin(i),
			ReadPrice:         currency.Coin(i),
			TotalData:         float64(i * 100),
			DataRead:          float64(i),
		}
		//bs[i] = b
		if err := part.Add(sctx, &br); err != nil {
			panic(err)
		}
	}

	if err := part.Save(sctx); err != nil {
		panic(err)
	}

	return sctx, clean
}

func BenchmarkPartitionsGetItem(b *testing.B) {
	ps, clean := prepareState(100, 100)
	defer clean()

	part, err := partitions.GetPartitions(ps, "brn_test")
	require.NoError(b, err)

	id := strconv.Itoa(10)
	for i := 0; i < b.N; i++ {
		var br BlobberRewardNode
		_ = part.Get(ps, id, &br)
	}
}

func BenchmarkGetRandomItems(b *testing.B) {
	seed := rand.NewSource(time.Now().Unix())
	r := rand.New(seed)
	ps, clean := prepareState(100, 100)
	defer clean()

	part, err := partitions.GetPartitions(ps, "brn_test")
	require.NoError(b, err)
	//ids := make([]string, 100)
	//for i := 0; i < 100; i++ {
	//	ids[i] = strconv.Itoa(i)
	//}

	for i := 0; i < b.N; i++ {
		var bs []BlobberRewardNode
		_ = part.GetRandomItems(ps, r, &bs)
	}
}

func TestPartitionRandomItems(t *testing.T) {
	seed := rand.NewSource(time.Now().Unix())
	r := rand.New(seed)
	ps, clean := prepareState(6, 10)
	defer clean()

	part, err := partitions.GetPartitions(ps, "brn_test")
	require.NoError(t, err)
	ids := make([]string, 100)
	for i := 0; i < 100; i++ {
		ids[i] = strconv.Itoa(i)
	}

	//for i := 0; i < b.N; i++ {
	var bs []BlobberRewardNode
	_ = part.GetRandomItems(ps, r, &bs)
	//}

	for i := 0; i < 6; i++ {
		t.Logf("%v", bs[i])
	}
}

func BenchmarkGetUpdateItem(b *testing.B) {
	ps, clean := prepareState(100, 100)
	defer clean()

	part, err := partitions.GetPartitions(ps, "brn_test")
	require.NoError(b, err)

	for i := 0; i < b.N; i++ {
		_ = part.UpdateItem(ps, &BlobberRewardNode{ID: "100"})
	}

	_ = part.Save(ps)
}

func prepareMPTState(t *testing.T) (state.StateContextI, func()) {
	dir, err := os.MkdirTemp("", "part_state")
	if err != nil {
		panic(err)
	}

	pdb, _ := util.NewPNodeDB(dir, dir+"/log")

	clean := func() {
		pdb.Flush()
		pdb.Close()
		_ = os.RemoveAll(dir)
	}

	mpt := util.NewMerklePatriciaTrie(pdb, 0, nil)
	return state.NewStateContext(nil,
		mpt, nil, nil, nil,
		nil, nil, nil, nil), clean
}

func TestAddBlobberChallengeItems(t *testing.T) {
	state, clean := prepareMPTState(t)
	defer clean()

	_, err := partitions.CreateIfNotExists(state, ALL_CHALLENGE_READY_BLOBBERS_KEY, allChallengeReadyBlobbersPartitionSize)
	require.NoError(t, err)

	p, err := partitionsChallengeReadyBlobbers(state)
	require.NoError(t, err)

	err = p.Add(state, &ChallengeReadyBlobber{BlobberID: "blobber_id_1"})
	require.NoError(t, err)
	err = p.Save(state)
	require.NoError(t, err)

	p, err = partitionsChallengeReadyBlobbers(state)
	require.NoError(t, err)
	s, err := p.Size(state)
	require.NoError(t, err)
	require.Equal(t, 1, s)

	err = p.Add(state, &ChallengeReadyBlobber{BlobberID: "blobber_id_2"})
	require.NoError(t, err)

	err = p.Save(state)
	require.NoError(t, err)

	p, err = partitionsChallengeReadyBlobbers(state)
	require.NoError(t, err)

	s, err = p.Size(state)
	require.NoError(t, err)

	require.Equal(t, 2, s)
}

func getZeta(wp, rp float64) float64 {

	i := float64(1)
	k := float64(0.9)
	mu := float64(0.2)

	if wp == 0 {
		return 0
	}

	return i - (k * (rp / (rp + (mu * wp))))
}

func getGamma(X, R float64) float64 {

	A := float64(10)
	B := float64(1)
	alpha := float64(0.2)

	if X == 0 {
		return 0
	}

	factor := math.Abs((alpha*X - R) / (alpha*X + R))
	return A - B*factor
}

func calculateWeight(wp, rp, X, R, stakes, challenges float64) float64 {

	zeta := getZeta(wp, rp)
	gamma := getGamma(X, R)

	fmt.Println("zeta", zeta)
	fmt.Println("gamma", gamma)
	fmt.Println("stakes", stakes)
	fmt.Println("challenges", challenges)

	return (zeta*gamma + 1) * stakes * challenges
}
