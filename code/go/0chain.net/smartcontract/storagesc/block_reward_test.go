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

						totalReward := 500.0

						totalExpectedReward, _ := currency.Float64ToCoin(totalReward)

						blobber1Weight := calculateWeight(writePrice[0], readPrice[0], writeData[0], readData[0], 4, challenge[0])
						blobber2Weight := calculateWeight(writePrice[1], readPrice[1], writeData[1], readData[1], 2, challenge[1])
						totalWeight := blobber1Weight + blobber2Weight

						fmt.Println("Blobber 1 Weight : ", blobber1Weight, " vs Blobber 2 Weight : ", blobber2Weight, " vs Total Weight : ", totalWeight)

						blobber1ExpectedReward, _ := currency.Float64ToCoin(totalReward * (blobber1Weight / totalWeight))
						blobber2ExpectedReward, _ := currency.MinusCoin(totalExpectedReward, blobber1ExpectedReward)

						fmt.Println("Blobber 1 Expected Reward : ", blobber1ExpectedReward, " vs Blobber 2 Expected Reward : ", blobber2ExpectedReward)

						blobber1Reward, _ := currency.MultFloat64(blobber1ExpectedReward, 0.1)
						blobber2Reward, _ := currency.MultFloat64(blobber2ExpectedReward, 0.1)
						blobber1ExpectedReward, _ = currency.MinusCoin(blobber1ExpectedReward, blobber1Reward)
						blobber2ExpectedReward, _ = currency.MinusCoin(blobber2ExpectedReward, blobber2Reward)

						blobber1Delegate1Reward, _ := currency.MultFloat64(blobber1ExpectedReward, 0.25)
						blobber1Delegate2Reward, _ := currency.MinusCoin(blobber1ExpectedReward, blobber1Delegate1Reward)

						blobber2Delegate1Reward, _ := currency.MultFloat64(blobber2ExpectedReward, 0.5)
						blobber2Delegate2Reward, _ := currency.MinusCoin(blobber2ExpectedReward, blobber2Delegate1Reward)

						wp1, _ := currency.Float64ToCoin(writePrice[0])
						wp2, _ := currency.Float64ToCoin(writePrice[1])
						rp1, _ := currency.Float64ToCoin(readPrice[0])
						rp2, _ := currency.Float64ToCoin(readPrice[1])

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
								blobberRewards:          []currency.Coin{blobber1Reward, blobber2Reward},
								blobberDelegatesRewards: [][]currency.Coin{{blobber1Delegate1Reward, blobber1Delegate2Reward}, {blobber2Delegate1Reward, blobber2Delegate2Reward}},
							},
						})
					}
				}
			}
		}
	}

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
	B := float64(9)
	alpha := float64(0.2)

	if X == 0 {
		return 0
	}

	factor := math.Abs((alpha*X - R) / (alpha*X + R))

	//fmt.Println("factor", factor)
	return A - B*factor
}

func calculateWeight(wp, rp, X, R, stakes, challenges float64) float64 {

	//fmt.Println("wp", wp, "rp", rp, "X", X, "R", R, "stakes", stakes, "challenges", challenges)

	zeta := getZeta(wp, rp)
	gamma := getGamma(X, R)

	//fmt.Println("zeta", zeta)
	//fmt.Println("gamma", gamma)
	//fmt.Println("stakes", stakes)
	//fmt.Println("challenges", challenges)

	return (zeta*gamma + 1) * stakes * challenges
}
