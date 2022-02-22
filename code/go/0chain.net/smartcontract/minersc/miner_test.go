package minersc_test

import (
	"strconv"
	"testing"

	"0chain.net/chaincore/block"
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/mocks"
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
	const (
		mockDeletedMinerId               = "mock deleted miner id"
		mockRoundNumber                  = 5
		x10                state.Balance = 10 * 1000 * 1000 * 1000
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
		for i := 0; i < p.activePools; i++ {
			id := "active pool " + strconv.Itoa(i)
			mn.Active[id] = sci.NewDelegatePool()
			mn.Active[id].ID = id
			mn.Active[id].TokenLockInterface = &ViewChangeLock{}
		}
		for i, amount := range p.pendingPools {
			id := "pending pool " + strconv.Itoa(i)
			delegateId := "delegate " + strconv.Itoa(i)
			mn.Pending[id] = sci.NewDelegatePool()
			mn.Pending[id].SetBalance(state.Balance(amount) * x10)
			balances.On("AddTransfer", &state.Transfer{
				ClientID:   msc.ID,
				ToClientID: delegateId,
				Amount:     state.Balance(amount) * x10,
			}).Return(nil).Once()
			mn.Pending[id].ID = id
			mn.Pending[id].TokenLockInterface = &ViewChangeLock{}
			mn.Pending[id].DelegateID = delegateId
			un := NewUserNode()
			un.ID = delegateId
			un.Pools = map[datastore.Key][]datastore.Key{mn.ID: {id}}
			balances.On("GetTrieNode", un.GetKey(), mock.MatchedBy(func(n *UserNode) bool {
				*n = *un
				return true
			})).Return(nil).Once()
			balances.On("DeleteTrieNode", un.GetKey()).Return("", nil).Once()
		}

		balances.On("GetTrieNode", mn.GetKey(), mock.MatchedBy(func(n *MinerNode) bool {
			*n = *mn
			return true
		})).Return(nil).Once()

		balances.On(
			"InsertTrieNode",
			mn.GetKey(),
			mock.MatchedBy(func(mn *MinerNode) bool {
				return 0 == len(mn.Pending) &&
					len(mn.Deleting) == p.activePools &&
					len(mn.Active) == p.activePools &&
					mn.ID == mockDeletedMinerId
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
