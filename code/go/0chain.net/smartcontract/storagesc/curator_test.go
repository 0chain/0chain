package storagesc

import (
	"encoding/json"
	"testing"

	chainState "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/mocks"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/transaction"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestAddCurator(t *testing.T) {
	t.Parallel()
	const (
		mockClientId     = "mock client id"
		mockCuratorId    = "mock curator id"
		mockAllocationId = "mock allocation id"
	)
	type args struct {
		ssc      *StorageSmartContract
		txn      *transaction.Transaction
		input    []byte
		balances chainState.StateContextI
	}
	type parameters struct {
		clientId         string
		info             curatorInput
		existingCurators []string
	}
	type want struct {
		err    bool
		errMsg string
	}
	var setExpectations = func(t *testing.T, name string, p parameters, want want) args {
		var balances = &mocks.StateContextI{}
		var txn = &transaction.Transaction{
			ClientID: p.clientId,
		}
		var ssc = &StorageSmartContract{
			SmartContract: sci.NewSC(ADDRESS),
		}
		input, err := json.Marshal(p.info)
		require.NoError(t, err)

		var sa = StorageAllocation{
			ID:    p.info.AllocationId,
			Owner: p.clientId,
		}
		for _, curator := range p.existingCurators {
			sa.Curators = append(sa.Curators, curator)
		}
		balances.On("GetTrieNode", sa.GetKey(ssc.ID)).Return(&sa, nil).Once()

		balances.On(
			"InsertTrieNode",
			sa.GetKey(ssc.ID),
			mock.MatchedBy(func(sa *StorageAllocation) bool {
				if len(sa.Curators) < 1 {
					return false
				}
				for i, curator := range p.existingCurators {
					if curator != sa.Curators[i] {
						return false
					}
				}
				if p.info.CuratorId != sa.Curators[len(sa.Curators)-1] {
					return false
				}
				return sa.ID == p.info.AllocationId && sa.Owner == p.clientId
			})).Return("", nil).Once()

		return args{ssc, txn, input, balances}
	}

	testCases := []struct {
		name       string
		parameters parameters
		want       want
	}{
		{
			name: "ok",
			parameters: parameters{
				clientId: mockClientId,
				info: curatorInput{
					CuratorId:    mockCuratorId,
					AllocationId: mockAllocationId,
				},
			},
		},
	}
	for _, test := range testCases {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			args := setExpectations(t, test.name, test.parameters, test.want)

			_, err := args.ssc.addCurator(args.txn, args.input, args.balances)

			require.EqualValues(t, test.want.err, err != nil)
			if err != nil {
				require.EqualValues(t, test.want.errMsg, err.Error())
				return
			}
			require.True(t, mock.AssertExpectationsForObjects(t, args.balances))
		})
	}
}

func TestRemoveCurator(t *testing.T) {
	t.Parallel()
	const (
		mockClientId     = "mock client id"
		mockCuratorId    = "mock curator id"
		mockCuratorId2   = "mock curator id 2"
		mockCuratorId3   = "mock curator id 3"
		mockAllocationId = "mock allocation id"
	)
	type args struct {
		ssc      *StorageSmartContract
		txn      *transaction.Transaction
		input    []byte
		balances chainState.StateContextI
	}
	type parameters struct {
		clientId         string
		info             curatorInput
		existingCurators []string
	}
	type want struct {
		err    bool
		errMsg string
	}
	var setExpectations = func(t *testing.T, name string, p parameters, want want) args {
		var balances = &mocks.StateContextI{}
		var txn = &transaction.Transaction{
			ClientID: p.clientId,
		}
		var ssc = &StorageSmartContract{

			SmartContract: sci.NewSC(ADDRESS),
		}
		input, err := json.Marshal(p.info)
		require.NoError(t, err)

		var sa = StorageAllocation{
			ID:    p.info.AllocationId,
			Owner: p.clientId,
		}
		for _, curator := range p.existingCurators {
			sa.Curators = append(sa.Curators, curator)
		}
		balances.On("GetTrieNode", sa.GetKey(ssc.ID)).Return(&sa, nil).Once()

		balances.On(
			"InsertTrieNode",
			sa.GetKey(ssc.ID),
			mock.MatchedBy(func(sa *StorageAllocation) bool {
				if len(sa.Curators) != len(p.existingCurators)-1 {
					return false
				}
				for _, curator := range sa.Curators {
					if p.info.CuratorId == curator {
						return false
					}
				}
				return sa.ID == p.info.AllocationId && sa.Owner == p.clientId
			})).Return("", nil).Once()

		return args{ssc, txn, input, balances}
	}

	testCases := []struct {
		name       string
		parameters parameters
		want       want
	}{
		{
			name: "ok_1_curators",
			parameters: parameters{
				clientId: mockClientId,
				info: curatorInput{
					CuratorId:    mockCuratorId,
					AllocationId: mockAllocationId,
				},
				existingCurators: []string{mockCuratorId2, mockCuratorId, mockCuratorId3},
			},
		},
		{
			name: "ok_3_curators",
			parameters: parameters{
				clientId: mockClientId,
				info: curatorInput{
					CuratorId:    mockCuratorId,
					AllocationId: mockAllocationId,
				},
				existingCurators: []string{mockCuratorId},
			},
		},
		{
			name: "error_not_curator",
			parameters: parameters{
				clientId: mockClientId,
				info: curatorInput{
					CuratorId:    mockCuratorId,
					AllocationId: mockAllocationId,
				},
				existingCurators: []string{mockCuratorId2, mockCuratorId3},
			},
			want: want{
				err:    true,
				errMsg: "remove_curator_failed: cannot find curator: " + mockCuratorId,
			},
		},
	}
	for _, test := range testCases {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			args := setExpectations(t, test.name, test.parameters, test.want)

			_, err := args.ssc.removeCurator(args.txn, args.input, args.balances)

			require.EqualValues(t, test.want.err, err != nil)
			if err != nil {
				require.EqualValues(t, test.want.errMsg, err.Error())
				return
			}
			require.True(t, mock.AssertExpectationsForObjects(t, args.balances))
		})
	}
}
