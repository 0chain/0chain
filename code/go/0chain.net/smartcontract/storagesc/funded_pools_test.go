package storagesc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/mocks"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/core/util"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestAddToFundedPools(t *testing.T) {
	const (
		mockClient     = "mock client"
		mockNewPool    = "mock new pool"
		mockExistingId = "mock existing id"
	)

	type parameters struct {
		client, newPool string
		existing        []string
	}
	var setExpectations = func(
		t *testing.T, p parameters,
	) (*StorageSmartContract, cstate.StateContextI) {
		var balances = &mocks.StateContextI{}
		var ssc = &StorageSmartContract{
			SmartContract: sci.NewSC(ADDRESS),
		}
		if len(p.existing) != 0 {
			var existingPools fundedPools = p.existing
			balances.On(
				"GetTrieNode",
				fundedPoolsKey(ssc.ID, p.client),
			).Return(&existingPools, nil).Once()
		} else {
			balances.On(
				"GetTrieNode",
				fundedPoolsKey(ssc.ID, p.client),
			).Return(nil, util.ErrValueNotPresent).Once()
		}

		balances.On(
			"InsertTrieNode",
			fundedPoolsKey(ssc.ID, p.client),
			mock.MatchedBy(func(fp *fundedPools) bool {
				if len(p.existing) != len(*fp)-1 {
					return false
				}
				for i, id := range p.existing {
					if id != (*fp)[i] {
						return false
					}
				}
				return p.newPool == (*fp)[len(*fp)-1]
			}),
		).Return("", nil).Once()

		return ssc, balances
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
			name: "ok",
			parameters: parameters{
				client:   mockClient,
				newPool:  mockNewPool,
				existing: []string{mockExistingId},
			},
		},
		{
			name: "ok_no_existing",
			parameters: parameters{
				client:  mockClient,
				newPool: mockNewPool,
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ssc, balances := setExpectations(t, tt.parameters)

			err := ssc.addToFundedPools(tt.parameters.client, tt.parameters.newPool, balances)

			require.EqualValues(t, tt.want.error, err != nil)
			if err != nil {
				require.EqualValues(t, tt.want.errorMsg, err.Error())
				return
			}
			require.True(t, mock.AssertExpectationsForObjects(t, balances))
		})
	}
}

func TestIsFundedPool(t *testing.T) {
	const (
		mockClient     = "mock client"
		mockPoolId     = "mock pool id"
		mockExistingId = "mock existing id"
	)

	type parameters struct {
		client, poolId string
		existing       []string
	}
	var setExpectations = func(
		t *testing.T, p parameters,
	) (*StorageSmartContract, cstate.StateContextI) {
		var balances = &mocks.StateContextI{}
		var ssc = &StorageSmartContract{

			SmartContract: sci.NewSC(ADDRESS),
		}

		if len(p.existing) != 0 {
			var existingPools fundedPools = p.existing
			balances.On(
				"GetTrieNode",
				fundedPoolsKey(ssc.ID, p.client),
			).Return(&existingPools, nil).Once()
		} else {
			balances.On(
				"GetTrieNode",
				fundedPoolsKey(ssc.ID, p.client),
			).Return(nil, util.ErrValueNotPresent).Once()
		}

		return ssc, balances
	}

	type want struct {
		error    bool
		errorMsg string
		ok       bool
	}
	tests := []struct {
		name       string
		parameters parameters
		want       want
	}{
		{
			name: "ok_not_found",
			parameters: parameters{
				client:   mockClient,
				poolId:   mockPoolId,
				existing: []string{mockExistingId},
			},
		},
		{
			name: "ok_found",
			parameters: parameters{
				client: mockClient,
				poolId: mockPoolId,
				existing: []string{
					mockExistingId, mockPoolId,
				},
			},
			want: want{
				ok: true,
			},
		},
		{
			name: "ok_not_found_no_existing_funding_pool",
			parameters: parameters{
				client: mockClient,
				poolId: mockPoolId,
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ssc, balances := setExpectations(t, tt.parameters)

			ok, err := ssc.isFundedPool(tt.parameters.client, tt.parameters.poolId, balances)

			require.EqualValues(t, tt.want.error, err != nil)
			if err != nil {
				require.EqualValues(t, tt.want.errorMsg, err.Error())
				return
			}
			require.EqualValues(t, tt.want.ok, ok)
			require.True(t, mock.AssertExpectationsForObjects(t, balances))
		})
	}
}
