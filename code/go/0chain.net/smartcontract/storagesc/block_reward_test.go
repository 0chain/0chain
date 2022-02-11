package storagesc

import (
	"0chain.net/chaincore/state"
	"0chain.net/smartcontract/partitions"
	"0chain.net/smartcontract/stakepool"
	"github.com/stretchr/testify/require"
	"strconv"
	"testing"
)

func TestStorageSmartContract_blobberBlockRewards(t *testing.T) {
	balances := newTestBalances(t, true)
	type params struct {
		numBlobbers       int
		wp                []state.Balance
		rp                []state.Balance
		totalData         []float64
		dataRead          []float64
		successChallenges []int
		delegatesBal      [][]state.Balance
	}

	type result struct {
		blobberRewards          []state.Balance
		blobberDelegatesRewards [][]state.Balance
	}

	setupRewards := func(t *testing.T, p params, sc *StorageSmartContract) {
		setConfig(t, balances)
		allBR, err := getActivePassedBlobbersList(balances)
		require.NoError(t, err)
		for i := 0; i < p.numBlobbers; i++ {
			bID := "blobber" + strconv.Itoa(i)
			_, err = allBR.Add(&partitions.BlobberRewardNode{
				Id:                bID,
				SuccessChallenges: p.successChallenges[i],
				WritePrice:        p.wp[i],
				ReadPrice:         p.rp[i],
				TotalData:         p.totalData[i],
				DataRead:          p.dataRead[i],
			}, balances)
			require.NoError(t, err)

			sp := newStakePool()
			sp.Settings.DelegateWallet = bID
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

		_, err := balances.DeleteTrieNode(ACTIVE_PASSED_BLOBBERS_KEY)
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
				wp:                []state.Balance{2},
				rp:                []state.Balance{1},
				totalData:         []float64{10},
				dataRead:          []float64{2},
				successChallenges: []int{10},
				delegatesBal:      [][]state.Balance{{1, 0, 3}},
			},
			result: result{
				blobberRewards:          []state.Balance{500},
				blobberDelegatesRewards: [][]state.Balance{{125, 0, 375}},
			},
		},
		{
			name: "2_blobber",
			params: params{
				numBlobbers:       2,
				wp:                []state.Balance{3, 1},
				rp:                []state.Balance{1, 0},
				totalData:         []float64{10, 50},
				dataRead:          []float64{2, 15},
				successChallenges: []int{5, 2},
				delegatesBal:      [][]state.Balance{{1, 0, 3}, {1, 6, 3}},
			},
			result: result{
				blobberRewards:          []state.Balance{178, 321},
				blobberDelegatesRewards: [][]state.Balance{{44, 0, 133}, {32, 192, 96}},
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
