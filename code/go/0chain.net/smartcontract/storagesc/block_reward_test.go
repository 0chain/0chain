package storagesc

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"strconv"
	"testing"
	"time"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/transaction"
	"0chain.net/smartcontract/stakepool/spenum"

	"github.com/0chain/common/core/currency"
	"github.com/0chain/common/core/statecache"

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

	getZeta := func(wp, rp float64) float64 {

		i := float64(1)
		k := float64(0.9)
		mu := float64(0.2)

		if wp == 0 {
			return 0
		}

		return i - (k * (rp / (rp + (mu * wp))))
	}

	getGamma := func(X, R float64) float64 {

		A := float64(10)
		B := float64(9)
		alpha := float64(0.2)

		if X == 0 {
			return 0
		}

		factor := math.Abs((alpha*X - R) / (alpha*X + R))

		return A - B*factor
	}

	calculateWeight := func(wp, rp, X, R, stakes, challenges float64) float64 {

		zeta := getZeta(wp, rp)
		gamma := getGamma(X, R)

		return (zeta*gamma + 1) * stakes * challenges
	}

	calculateExpectedRewards := func(p params) result {
		if p.numBlobbers != len(p.wp) {
			return result{}
		}

		wp1, _ := p.wp[0].Float64()
		rp1, _ := p.rp[0].Float64()
		totalData1 := p.totalData[0]
		dataRead1 := p.dataRead[0]
		successChallenges1 := p.successChallenges[0]
		delegatesBal1 := p.delegatesBal[0]
		serviceCharge1 := p.serviceCharge[0]

		totalStakes1 := 0.0
		for _, bal := range delegatesBal1 {
			totalStakes1 += float64(bal)
		}

		blobber1Weight := calculateWeight(wp1, rp1, totalData1, dataRead1, totalStakes1, float64(successChallenges1))

		wp2, _ := p.wp[1].Float64()
		rp2, _ := p.rp[1].Float64()
		totalData2 := p.totalData[1]
		dataRead2 := p.dataRead[1]
		successChallenges2 := p.successChallenges[1]
		delegatesBal2 := p.delegatesBal[1]
		serviceCharge2 := p.serviceCharge[1]

		totalStakes2 := 0.0
		for _, bal := range delegatesBal2 {
			totalStakes2 += float64(bal)
		}

		blobber2Weight := calculateWeight(wp2, rp2, totalData2, dataRead2, totalStakes2, float64(successChallenges2))

		totalWeight := blobber1Weight + blobber2Weight

		blobber1TotalReward := (blobber1Weight / totalWeight) * 18000000000
		blobber2TotalReward := (blobber2Weight / totalWeight) * 18000000000

		blobber1Reward := blobber1TotalReward * serviceCharge1
		blobber2Reward := blobber2TotalReward * serviceCharge2

		blobber1DelegatesTotalReward := blobber1TotalReward - blobber1Reward
		blobber2DelegatesTotalReward := blobber2TotalReward - blobber2Reward

		blobber1DelegatesRewards := make([]currency.Coin, len(delegatesBal1))
		blobber2DelegatesRewards := make([]currency.Coin, len(delegatesBal2))

		for i, bal := range delegatesBal1 {
			blobber1DelegatesRewards[i], _ = currency.Float64ToCoin(blobber1DelegatesTotalReward * (float64(bal) / totalStakes1))
		}

		for i, bal := range delegatesBal2 {
			blobber2DelegatesRewards[i], _ = currency.Float64ToCoin(blobber2DelegatesTotalReward * (float64(bal) / totalStakes2))
		}

		blobber1RewardInCoin, _ := currency.Float64ToCoin(blobber1Reward)
		blobber2RewardInCoin, _ := currency.Float64ToCoin(blobber2Reward)

		return result{
			blobberRewards: []currency.Coin{
				blobber1RewardInCoin,
				blobber2RewardInCoin,
			},
			blobberDelegatesRewards: [][]currency.Coin{
				blobber1DelegatesRewards,
				blobber2DelegatesRewards,
			},
		}
	}

	calculateExpectedRewards(params{
		numBlobbers: 2,
		wp:          []currency.Coin{1, 2, 3},
	})

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
		_, err := ssc.getConfig(balances, false)
		require.NoError(t, err)
		for i := 0; i < p.numBlobbers; i++ {
			bID := "blobber" + strconv.Itoa(i)
			sp, err := ssc.getStakePool(spenum.Blobber, bID, balances)
			require.NoError(t, err)

			message := fmt.Sprintf("Expected Blobber %d Actual Reward : %v", i, sp.Reward)

			resultBlobberReward, _ := r.blobberRewards[i].Float64()
			actualBlobberReward, _ := sp.Reward.Float64()

			require.InDelta(t, resultBlobberReward, actualBlobberReward, errDelta, message)

			for j := range p.delegatesBal[i] {
				key := "delegate" + strconv.Itoa(j)
				message := fmt.Sprintf("Expected Blobber %d Delegate %d  Actual Reward : %v", i, j, sp.Pools[key].Reward)

				resultDelegateReward, _ := r.blobberDelegatesRewards[i][j].Float64()
				actualDelegateReward, _ := sp.Pools[key].Reward.Float64()

				require.InDelta(t, resultDelegateReward, actualDelegateReward, errDelta, message)
			}
		}
		require.NoError(t, err)
	}

	type TestCase struct {
		name    string
		params  params
		result  result
		wantErr bool
	}

	var tests []TestCase

	// When there is only one blobber
	caseParams := params{
		numBlobbers:       1,
		wp:                []currency.Coin{2},
		rp:                []currency.Coin{1},
		totalData:         []float64{10},
		dataRead:          []float64{2},
		successChallenges: []int{10},
		delegatesBal:      [][]currency.Coin{{1, 0, 3}},
		serviceCharge:     []float64{.1},
	}
	caseResult := result{
		blobberRewards:          []currency.Coin{1800000000},
		blobberDelegatesRewards: [][]currency.Coin{{4050000000, 0, 12150000000}},
	}
	tests = append(tests, TestCase{
		name:    "Only one blobber in the partition",
		params:  caseParams,
		result:  caseResult,
		wantErr: false,
	})

	// 2 Blobbers with 3 delegates each same data
	caseParams = params{
		numBlobbers:       2,
		wp:                []currency.Coin{1000000000, 1000000000},
		rp:                []currency.Coin{1000000000, 1000000000},
		totalData:         []float64{10, 10},
		dataRead:          []float64{10, 10},
		successChallenges: []int{40, 40},
		delegatesBal:      [][]currency.Coin{{2, 2, 2}, {2, 2, 2}},
		serviceCharge:     []float64{.1, .1},
	}
	caseResult = calculateExpectedRewards(caseParams)
	tests = append(tests, TestCase{
		name:    "2 Blobbers with 3 delegates each same data",
		params:  caseParams,
		result:  caseResult,
		wantErr: false,
	})

	// 2 Blobbers with 3 delegates with write price {1000000000, 9000000000}
	caseParams = params{
		numBlobbers:       2,
		wp:                []currency.Coin{1000000000, 9000000000},
		rp:                []currency.Coin{1000000000, 1000000000},
		totalData:         []float64{10, 10},
		dataRead:          []float64{10, 10},
		successChallenges: []int{40, 40},
		delegatesBal:      [][]currency.Coin{{2, 2, 2}, {2, 2, 2}},
		serviceCharge:     []float64{.1, .1},
	}
	caseResult = calculateExpectedRewards(caseParams)
	tests = append(tests, TestCase{
		name:    "2 Blobbers with 3 delegates with write price {1000000000, 9000000000}",
		params:  caseParams,
		result:  caseResult,
		wantErr: false,
	})

	// 2 Blobbers with 3 delegates with read price to verify free reads {0, 1000000000}
	caseParams = params{
		numBlobbers:       2,
		wp:                []currency.Coin{1000000000, 1000000000},
		rp:                []currency.Coin{0, 1000000000},
		totalData:         []float64{10, 10},
		dataRead:          []float64{10, 10},
		successChallenges: []int{40, 40},
		delegatesBal:      [][]currency.Coin{{2, 2, 2}, {2, 2, 2}},
		serviceCharge:     []float64{.1, .1},
	}
	caseResult = calculateExpectedRewards(caseParams)
	tests = append(tests, TestCase{
		name:    "2 Blobbers with 3 delegates with read price {0, 1000000000}",
		params:  caseParams,
		result:  caseResult,
		wantErr: false,
	})

	// 2 Blobbers with 3 delegates with different read price {1000000000, 9000000000}
	caseParams = params{
		numBlobbers:       2,
		wp:                []currency.Coin{1000000000, 1000000000},
		rp:                []currency.Coin{1000000000, 9000000000},
		totalData:         []float64{10, 10},
		dataRead:          []float64{10, 10},
		successChallenges: []int{40, 40},
		delegatesBal:      [][]currency.Coin{{2, 2, 2}, {2, 2, 2}},
		serviceCharge:     []float64{.1, .1},
	}
	caseResult = calculateExpectedRewards(caseParams)
	tests = append(tests, TestCase{
		name:    "2 Blobbers with 3 delegates with read price {1000000000, 9000000000}",
		params:  caseParams,
		result:  caseResult,
		wantErr: false,
	})

	// 2 Blobbers with 3 delegates with different total data {10, 90}
	caseParams = params{
		numBlobbers:       2,
		wp:                []currency.Coin{1000000000, 1000000000},
		rp:                []currency.Coin{0, 0},
		totalData:         []float64{10, 90},
		dataRead:          []float64{10, 10},
		successChallenges: []int{40, 40},
		delegatesBal:      [][]currency.Coin{{2, 2, 2}, {2, 2, 2}},
		serviceCharge:     []float64{.1, .1},
	}
	caseResult = calculateExpectedRewards(caseParams)
	tests = append(tests, TestCase{
		name:    "2 Blobbers with 3 delegates with different total data {10, 90}",
		params:  caseParams,
		result:  caseResult,
		wantErr: false,
	})

	// 2 Blobbers with 3 delegates with different data read {10, 90}
	caseParams = params{
		numBlobbers:       2,
		wp:                []currency.Coin{1000000000, 1000000000},
		rp:                []currency.Coin{0, 0},
		totalData:         []float64{10, 10},
		dataRead:          []float64{10, 90},
		successChallenges: []int{40, 40},
		delegatesBal:      [][]currency.Coin{{2, 2, 2}, {2, 2, 2}},
		serviceCharge:     []float64{.1, .1},
	}
	caseResult = calculateExpectedRewards(caseParams)
	tests = append(tests, TestCase{
		name:    "2 Blobbers with 3 delegates with different data read {10, 90}",
		params:  caseParams,
		result:  caseResult,
		wantErr: false,
	})

	// 2 Blobbers with 3 delegates with different success challenges {40, 90}
	caseParams = params{
		numBlobbers:       2,
		wp:                []currency.Coin{1000000000, 1000000000},
		rp:                []currency.Coin{0, 0},
		totalData:         []float64{10, 10},
		dataRead:          []float64{10, 10},
		successChallenges: []int{40, 90},
		delegatesBal:      [][]currency.Coin{{2, 2, 2}, {2, 2, 2}},
		serviceCharge:     []float64{.1, .1},
	}
	caseResult = calculateExpectedRewards(caseParams)
	tests = append(tests, TestCase{
		name:    "2 Blobbers with 3 delegates with different success challenges {40, 90}",
		params:  caseParams,
		result:  caseResult,
		wantErr: false,
	})

	// 2 Blobbers with 3 delegates with different success challenges {40, 0}
	caseParams = params{
		numBlobbers:       2,
		wp:                []currency.Coin{1000000000, 1000000000},
		rp:                []currency.Coin{0, 0},
		totalData:         []float64{10, 10},
		dataRead:          []float64{10, 10},
		successChallenges: []int{40, 0},
		delegatesBal:      [][]currency.Coin{{2, 2, 2}, {2, 2, 2}},
		serviceCharge:     []float64{.1, .1},
	}
	caseResult = calculateExpectedRewards(caseParams)
	tests = append(tests, TestCase{
		name:    "2 Blobbers with 3 delegates with different success challenges {40, 0}",
		params:  caseParams,
		result:  caseResult,
		wantErr: false,
	})

	// 2 Blobbers with 3 delegates with different delegates balances {{1, 0, 3}, {2, 6, 7}}
	caseParams = params{
		numBlobbers:       2,
		wp:                []currency.Coin{1000000000, 1000000000},
		rp:                []currency.Coin{0, 0},
		totalData:         []float64{10, 10},
		dataRead:          []float64{10, 10},
		successChallenges: []int{40, 40},
		delegatesBal:      [][]currency.Coin{{1, 0, 3}, {2, 6, 7}},
		serviceCharge:     []float64{.1, .1},
	}
	caseResult = calculateExpectedRewards(caseParams)
	tests = append(tests, TestCase{
		name:    "2 Blobbers with 3 delegates with different delegates balances {{1, 0, 3}, {2, 6, 7}}",
		params:  caseParams,
		result:  caseResult,
		wantErr: false,
	})

	// 2 Blobbers with 3 delegates with different service charge {.1, .3}
	caseParams = params{
		numBlobbers:       2,
		wp:                []currency.Coin{1000000000, 1000000000},
		rp:                []currency.Coin{0, 0},
		totalData:         []float64{10, 10},
		dataRead:          []float64{10, 10},
		successChallenges: []int{40, 40},
		delegatesBal:      [][]currency.Coin{{2, 2, 2}, {2, 2, 2}},
		serviceCharge:     []float64{.1, .3},
	}
	caseResult = calculateExpectedRewards(caseParams)
	tests = append(tests, TestCase{
		name:    "2 Blobbers with 3 delegates with different service charge {.1, .3}",
		params:  caseParams,
		result:  caseResult,
		wantErr: false,
	})

	// 2 Blobbers with 3 delegates with different service charge {1, 0}
	caseParams = params{
		numBlobbers:       2,
		wp:                []currency.Coin{1000000000, 1000000000},
		rp:                []currency.Coin{0, 0},
		totalData:         []float64{10, 10},
		dataRead:          []float64{10, 10},
		successChallenges: []int{40, 40},
		delegatesBal:      [][]currency.Coin{{2, 2, 2}, {2, 2, 2}},
		serviceCharge:     []float64{1, 0},
	}
	caseResult = calculateExpectedRewards(caseParams)
	tests = append(tests, TestCase{
		name:    "2 Blobbers with 3 delegates with different service charge {1, 0}",
		params:  caseParams,
		result:  caseResult,
		wantErr: false,
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			balances := newTestBalances(t, true)
			bk := &block.Block{}
			bk.Round = 210
			balances.setBlock(t, bk)

			in := BlobberBlockRewardsInput{Round: balances.block.Round}
			marshal, err2 := json.Marshal(in)
			if err2 != nil {
				t.Error(err2)
			}
			ssc := newTestStorageSC()
			setupRewards(t, tt.params, balances, ssc)
			err := ssc.blobberBlockRewards(&transaction.Transaction{}, marshal, balances)
			require.EqualValues(t, tt.wantErr, err != nil)
			compareResult(t, tt.params, tt.result, balances, ssc)
		})
	}
}

func prepareState(n, partSize int, sctx state.StateContextI) func() {
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

	// mpt := util.NewMerklePatriciaTrie(pdb, 0, nil, statecache.NewEmpty())
	// sctx = state.NewStateContext(nil,
	// mpt, nil, nil, nil,
	// nil, nil, nil, nil)

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

	return clean
}

func BenchmarkPartitionsGetItem(t *testing.B) {
	balances := newTestBalances(t, false)
	clean := prepareState(100, 100, balances)
	defer clean()

	part, err := partitions.GetPartitions(balances, "brn_test")
	require.NoError(t, err)

	id := strconv.Itoa(10)
	for i := 0; i < t.N; i++ {
		var br BlobberRewardNode
		_, _ = part.Get(balances, id, &br)
	}
}

func BenchmarkGetRandomItems(t *testing.B) {
	seed := rand.NewSource(time.Now().Unix())
	r := rand.New(seed)
	balances := newTestBalances(t, false)

	part, err := partitions.GetPartitions(balances, "brn_test")
	require.NoError(t, err)
	//ids := make([]string, 100)
	//for i := 0; i < 100; i++ {
	//	ids[i] = strconv.Itoa(i)
	//}

	for i := 0; i < t.N; i++ {
		var bs []BlobberRewardNode
		_ = part.GetRandomItems(balances, r, &bs)
	}
}

func TestPartitionRandomItems(t *testing.T) {
	seed := rand.NewSource(time.Now().Unix())
	r := rand.New(seed)
	balances := newTestBalances(t, false)
	clean := prepareState(6, 10, balances)
	defer clean()

	part, err := partitions.GetPartitions(balances, "brn_test")
	require.NoError(t, err)
	ids := make([]string, 100)
	for i := 0; i < 100; i++ {
		ids[i] = strconv.Itoa(i)
	}

	//for i := 0; i < b.N; i++ {
	var bs []BlobberRewardNode
	_ = part.GetRandomItems(balances, r, &bs)
	//}

	for i := 0; i < 6; i++ {
		t.Logf("%v", bs[i])
	}
}

func BenchmarkGetUpdateItem(b *testing.B) {
	balances := newTestBalances(b, false)
	clean := prepareState(100, 100, balances)
	defer clean()

	part, err := partitions.GetPartitions(balances, "brn_test")
	require.NoError(b, err)

	for i := 0; i < b.N; i++ {
		_ = part.UpdateItem(balances, &BlobberRewardNode{ID: "100"})
	}

	_ = part.Save(balances)
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

	mpt := util.NewMerklePatriciaTrie(pdb, 0, nil, statecache.NewEmpty())
	b := block.Block{}
	return state.NewStateContext(&b,
		mpt, nil, nil, nil,
		nil, nil, nil, nil, nil, nil), clean
}

func TestAddBlobberChallengeItems(t *testing.T) {
	state, clean := prepareMPTState(t)
	defer clean()

	_, err := partitions.CreateIfNotExists(state, ALL_CHALLENGE_READY_BLOBBERS_KEY, allChallengeReadyBlobbersPartitionSize)
	require.NoError(t, err)

	p, _, err := partitionsChallengeReadyBlobbers(state)
	require.NoError(t, err)

	err = p.Add(state, &ChallengeReadyBlobber{BlobberID: "blobber_id_1"})
	require.NoError(t, err)
	err = p.Save(state)
	require.NoError(t, err)

	p, _, err = partitionsChallengeReadyBlobbers(state)
	require.NoError(t, err)
	s, err := p.Size(state)
	require.NoError(t, err)
	require.Equal(t, 1, s)

	err = p.Add(state, &ChallengeReadyBlobber{BlobberID: "blobber_id_2"})
	require.NoError(t, err)

	err = p.Save(state)
	require.NoError(t, err)

	p, _, err = partitionsChallengeReadyBlobbers(state)
	require.NoError(t, err)

	s, err = p.Size(state)
	require.NoError(t, err)

	require.Equal(t, 2, s)
}

func TestGetBlockReward(t *testing.T) {

	br, _ := currency.Float64ToCoin(800000)
	brChangePeriod := int64(10000)
	brChangeRatio := 0.1

	type result struct {
		reward currency.Coin
		err    error
	}

	compareResult := func(t *testing.T, expected, actual result) {
		require.Equal(t, expected.err, actual.err)
		require.Equal(t, expected.reward, actual.reward)
	}

	calculateResult := func(currentRound int64) currency.Coin {
		changeBalance := 1 - brChangeRatio
		changePeriods := currentRound / brChangePeriod

		factor := math.Pow(changeBalance, float64(changePeriods))
		result, _ := currency.MultFloat64(br, factor)
		return result
	}

	tests := []struct {
		name         string
		currentRound int64
		result       result
	}{
		{
			name:         "Test 1",
			currentRound: 500,
			result: result{
				reward: calculateResult(500),
				err:    nil,
			},
		},
		{
			name:         "Test 1",
			currentRound: 23420,
			result: result{
				reward: calculateResult(23420),
				err:    nil,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			reward, err := getBlockReward(br, test.currentRound, brChangePeriod, brChangeRatio)
			compareResult(t, test.result, result{reward, err})
		})
	}
}
