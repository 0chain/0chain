package minersc_test

import (
	"strconv"
	"testing"

	"0chain.net/chaincore/currency"

	"0chain.net/smartcontract/stakepool"
	"0chain.net/smartcontract/stakepool/spenum"

	"0chain.net/chaincore/block"
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/chain/state/mocks"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
	. "0chain.net/smartcontract/minersc"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestDeleteMiner(t *testing.T) {
	t.Skip("delete_miner unused and to be reworked as kill_provider")
	const (
		mockDeletedMinerId               = "mock deleted miner id"
		mockRoundNumber                  = 5
		x10                currency.Coin = 10 * 1000 * 1000 * 1000
	)
	type parameters struct {
		pendingPools []int
		activePools  int
	}
	type args struct {
		msc       *MinerSmartContract
		inputData []byte
		gn        *GlobalNode
		balances  cstate.StateContextI
	}
	var setExpectations = func(
		t *testing.T, p parameters,
	) args {
		var balances = &mocks.StateContextI{}
		var msc = &MinerSmartContract{
			SmartContract: sci.NewSC(ADDRESS),
		}
		mn := NewMinerNode()
		mn.ID = mockDeletedMinerId
		mn.Settings.DelegateWallet = mockDeletedMinerId
		for i := 0; i < p.activePools; i++ {
			id := "active pool " + strconv.Itoa(i)
			var dp stakepool.DelegatePool
			dp.Status = spenum.Active
			mn.Pools[id] = &dp
		}
		for i, amount := range p.pendingPools {
			id := "pending pool " + strconv.Itoa(i)
			delegateId := "delegate " + strconv.Itoa(i)
			var dp stakepool.DelegatePool
			dp.Status = spenum.Pending
			dp.Balance = currency.Coin(amount) * x10
			dp.DelegateID = delegateId
			balances.On("AddTransfer", &state.Transfer{
				ClientID:   msc.ID,
				ToClientID: delegateId,
				Amount:     dp.Balance,
			}).Return(nil).Once()

			var un stakepool.UserStakePools
			un.Pools = map[datastore.Key][]datastore.Key{mn.ID: {id}}
			balances.On("GetTrieNode", stakepool.UserStakePoolsKey(spenum.Miner, id), mock.MatchedBy(func(n *stakepool.UserStakePools) bool {
				return true
			})).Return(nil).Once()
			balances.On("DeleteTrieNode", stakepool.UserStakePoolsKey(spenum.Miner, id)).Return("", nil).Once()
		}

		balances.On(
			"DeleteTrieNode",
			stakepool.UserStakePoolsKey(spenum.Miner, mn.Settings.DelegateWallet),
		).Return("", nil).Once()

		balances.On("GetTrieNode", mn.GetKey(), mock.MatchedBy(func(n *MinerNode) bool {
			*n = *mn
			return true
		})).Return(nil).Once()

		balances.On(
			"InsertTrieNode",
			mn.GetKey(),
			mock.MatchedBy(func(mn *MinerNode) bool {
				return 0 == len(mn.Pools) && mn.ID == mockDeletedMinerId
			}),
		).Return("", nil).Once()

		pn := &PhaseNode{}
		balances.On("GetTrieNode", pn.GetKey(), mock.AnythingOfType("*minersc.PhaseNode")).Return(util.ErrValueNotPresent).Once()
		mockBlock := &block.Block{}
		mockBlock.Round = mockRoundNumber
		balances.On("GetBlock").Return(mockBlock).Twice()
		balances.On("GetTrieNode", DKGMinersKey, mock.AnythingOfType("*minersc.DKGMinerNodes")).Return(util.ErrValueNotPresent).Once()

		mnInput := &MinerNode{
			SimpleNode: &SimpleNode{
				ID: mockDeletedMinerId,
			},
		}
		return args{
			msc:       msc,
			inputData: mnInput.Encode(),
			gn:        &GlobalNode{},
			balances:  balances,
		}
	}

	type want struct {
		error    bool
		errorMsg string
	}
	tests := []struct {
		name       string
		parameters parameters
		want       want
	}{
		{
			name: "ok_1_pool",
			parameters: parameters{
				pendingPools: []int{1},
				activePools:  1,
			},
		},
		{
			name: "ok_10_pools",
			parameters: parameters{
				pendingPools: []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
				activePools:  10,
			},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			args := setExpectations(t, test.parameters)

			_, err := args.msc.DeleteMiner(
				&transaction.Transaction{},
				args.inputData,
				args.gn,
				args.balances,
			)
			require.EqualValues(t, test.want.error, err != nil)
			if err != nil {
				require.EqualValues(t, test.want.errorMsg, err.Error())
				return
			}
			require.True(t, mock.AssertExpectationsForObjects(t, args.balances))
		})
	}
}
