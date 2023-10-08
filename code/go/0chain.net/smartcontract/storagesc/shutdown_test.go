package storagesc

import (
	"testing"

	"0chain.net/chaincore/transaction"
	"0chain.net/smartcontract/provider"
	"0chain.net/smartcontract/stakepool"
	"0chain.net/smartcontract/stakepool/spenum"
	"github.com/0chain/common/core/util"
	"github.com/stretchr/testify/require"
)

func TestStorageSmartContract_shutdownBlobber(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		blobberSavedSize  int64
		stakePool         *stakePool
		input             []byte
		clientID          string
		expectedStateFunc func(t *testing.T, state *testBalances)
	}{
		{
			name:             "both blobber and stake pools are empty",
			blobberSavedSize: 0,
			stakePool:        newStakePool(),
			input:            []byte(`{"provider_id":"blobber_id"}`),
			clientID:         "blobber_id",
			expectedStateFunc: func(t *testing.T, state *testBalances) {
				var b StorageNode
				b.ID = "blobber_id"
				err := state.GetTrieNode(b.GetKey(), &b)
				require.Error(t, util.ErrValueNotPresent, err)

				var s stakepool.StakePool
				err = state.GetTrieNode(stakePoolKey(spenum.Blobber, "blobber_id"), &s)
				require.Error(t, util.ErrValueNotPresent, err)
			},
		},
		{
			name:             "blobber is not empty",
			blobberSavedSize: 10,
			stakePool:        newStakePool(),
			input:            []byte(`{"provider_id":"blobber_id"}`),
			clientID:         "blobber_id",
			expectedStateFunc: func(t *testing.T, state *testBalances) {
				var b StorageNode
				b.ID = "blobber_id"
				err := state.GetTrieNode(b.GetKey(), &b)
				require.NoError(t, err)

				var s stakepool.StakePool
				err = state.GetTrieNode(stakePoolKey(spenum.Blobber, "blobber_id"), &s)
				require.NoError(t, err)
			},
		},
		{
			name:             "stake pool is not empty",
			blobberSavedSize: 0,
			stakePool: &stakePool{
				StakePool: &stakepool.StakePool{
					Pools: map[string]*stakepool.DelegatePool{
						"stake_pool_1": &stakepool.DelegatePool{},
						"stake_pool_2": &stakepool.DelegatePool{},
					},
				},
			},
			input:    []byte(`{"provider_id":"blobber_id"}`),
			clientID: "blobber_id",
			expectedStateFunc: func(t *testing.T, state *testBalances) {
				var b StorageNode
				b.ID = "blobber_id"
				b.ProviderType = spenum.Blobber
				err := state.GetTrieNode(b.GetKey(), &b)
				require.NoError(t, err)

				var sp stakePool
				err = state.GetTrieNode(stakePoolKey(spenum.Blobber, "blobber_id"), &sp)
				require.NoError(t, err)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ssc := &StorageSmartContract{}

			balances := newTestBalances(t, false)
			balances.txn = &transaction.Transaction{
				ClientID: tc.clientID,
			}

			conf := &Config{
				OwnerId:   tc.clientID,
				StakePool: &stakePoolConfig{},
			}

			mustSave(t, scConfigKey(ADDRESS), conf, balances)

			// Create a blobber and a stake pool
			blobber := &StorageNode{
				Provider: provider.Provider{
					ID:           "blobber_id",
					ProviderType: spenum.Blobber,
				},
				SavedData: tc.blobberSavedSize,
			}
			balances.InsertTrieNode(blobber.GetKey(), blobber)
			balances.InsertTrieNode(stakePoolKey(spenum.Blobber, blobber.ID), tc.stakePool)

			// Call the shutdown method
			_, err := ssc.shutdownBlobber(balances.txn, tc.input, balances)
			require.NoError(t, err)

			tc.expectedStateFunc(t, balances)
		})
	}
}
