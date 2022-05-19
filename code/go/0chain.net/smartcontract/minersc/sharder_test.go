package minersc_test

import (
	"strconv"
	"testing"

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

func TestDeleteSharder(t *testing.T) {
	t.Skip("delete_sharder is unused and to be reworked as kill_provider")
	const (
		mockDeletedSharderId               = "mock deleted sharder id"
		mockRoundNumber                    = 5
		x10                  state.Balance = 10 * 1000 * 1000 * 1000
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
		mn.ID = mockDeletedSharderId
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
			dp.Balance = state.Balance(amount) * x10
			dp.DelegateID = delegateId
			balances.On("AddTransfer", &state.Transfer{
				ClientID:   msc.ID,
				ToClientID: delegateId,
				Amount:     dp.Balance,
			}).Return(nil).Once()

			var un stakepool.UserStakePools
			un.Pools = map[datastore.Key][]datastore.Key{mn.ID: {id}}
			balances.On("GetTrieNode", stakepool.UserStakePoolsKey(spenum.Sharder, id),
				mock.MatchedBy(func(n *stakepool.UserStakePools) bool {
					return true
				})).Return(nil).Once()
			balances.On("DeleteTrieNode", stakepool.UserStakePoolsKey(spenum.Sharder, id)).Return("", nil).Once()
		}

		balances.On("GetTrieNode", GetSharderKey(mockDeletedSharderId),
			mock.MatchedBy(func(n *MinerNode) bool {
				*n = *mn
				return true
			})).Return(nil).Once()
		balances.On(
			"InsertTrieNode",
			mn.GetKey(),
			mock.MatchedBy(func(mn *MinerNode) bool {
				return 0 == len(mn.Pools) && mn.ID == mockDeletedSharderId
			}),
		).Return("", nil).Once()

		pn := &PhaseNode{}
		balances.On("GetTrieNode", pn.GetKey(), mock.AnythingOfType("*minersc.PhaseNode")).Return(util.ErrValueNotPresent).Once()
		mockBlock := &block.Block{}
		mockBlock.Round = mockRoundNumber
		balances.On("GetBlock").Return(mockBlock).Twice()
		balances.On("GetTrieNode", ShardersKeepKey, mock.AnythingOfType("*minersc.MinerNodes")).Return(util.ErrValueNotPresent).Once()
		balances.On("InsertTrieNode", ShardersKeepKey, &MinerNodes{}).Return("", nil).Once()

		mnInput := &MinerNode{
			SimpleNode: &SimpleNode{
				ID: mockDeletedSharderId,
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

			_, err := args.msc.DeleteSharder(
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
