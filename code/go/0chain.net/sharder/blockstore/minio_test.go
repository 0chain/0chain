package blockstore

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	"0chain.net/core/viper"
	"github.com/stretchr/testify/require"
)

func getMinioViperConfig(t *testing.T, strategy string) *viper.Viper {
	v := viper.New()
	coldStorageConfig := `
strategy: %s
cloud_storages:
  - storage_service_url: "s3.amazonaws.com"
    access_id: "ABCD"
	secret_access_key: "EFGH"
	bucket_name: "bkt1"
	allowed_block_numbers: "10^15"
	allowed_block_size: "10^5"
	
  - storage_service_url: "s3.amazonaws.com"
    access_id: "ABCD"
	secret_access_key: "EFGH"
	bucket_name: "bkt2"
	allowed_block_numbers: "%10^15"
	allowed_block_size: "10^5"
`
	v.SetConfigType("yaml")
	err := v.ReadConfig(bytes.NewBuffer([]byte(coldStorageConfig)))
	require.Nil(t, err)
	return v
}

func TestInitCold(t *testing.T) {
	t.Run("Undefined mode should panic", func(t *testing.T) {
		require.Panics(t, func() {
			initCold(getMinioViperConfig(t, "round_robin"), "")
		})
	})

	t.Run("Strategy other than round_robin should panic", func(t *testing.T) {
		require.Panics(t, func() {
			initCold(getMinioViperConfig(t, ""), "start")
		})
	})
}
func TestSelectNextStorage(t *testing.T) {
	t.Run("Next storage must have been selected", func(t *testing.T) {
		cT := initCold(getMinioViperConfig(t, ""), "start")
		cT.Mu.Lock()
		cT.Mu.Unlock()
		for i := 1; i < 10; i++ {
			ind := i % len(cT.ColdStorages)
			bktName := fmt.Sprintf("lpobkt%d", ind)
			select {
			case selectedColdStorage := <-cT.SelectedStorageChan:
				require.NotNil(t, selectedColdStorage)
				mc := selectedColdStorage.coldStorage.(*minioClient)
				require.Equal(t, bktName, mc.bucketName)
			default:
				t.Fail()
			}
			cT.SelectNextStorage(cT.ColdStorages, cT.PrevInd)
		}
	})
}

type mockMinioClient struct {
	mb func() (string, error)
}

func (mc *mockMinioClient) moveBlock(hash, blockPath string) (string, error) {
	return mc.mb()
}
func (mc *mockMinioClient) getBlock(hash string) ([]byte, error) {
	return nil, nil
}

func TestMoveBlock(t *testing.T) {
	type input struct {
		name                    string
		cTier                   *coldTier
		minioClients            []*mockMinioClient
		selectedColdStorageChan chan selectedColdStorage
		wantErr                 bool
		setup                   func(t *testing.T, in *input)
		furtherTets             func(t *testing.T, in *input)
	}

	tests := []input{
		{
			name: "Should select next cold storage",
			cTier: &coldTier{
				Mu: make(Mutex, 1),
			},
			minioClients: []*mockMinioClient{
				{
					mb: func() (string, error) { return "minio:bkt1/hash", nil },
				}, {
					mb: func() (string, error) { return "minio:bkt2/hash", nil },
				},
			},
			setup: func(t *testing.T, in *input) {
				in.cTier.SelectNextStorage = getColdRBStrategyFunc(in.cTier, in.selectedColdStorageChan)
				for _, m := range in.minioClients {
					in.cTier.ColdStorages = append(in.cTier.ColdStorages, m)
				}
				in.cTier.SelectNextStorage(in.cTier.ColdStorages, in.cTier.PrevInd)
			},
			furtherTets: func(t *testing.T, in *input) {
				prevInd := in.cTier.PrevInd
				selectedStorage := <-in.cTier.SelectedStorageChan
				require.NotEqual(t, prevInd, selectedStorage.prevInd)
			},
		}, {
			name: "Moving block should return error",
			cTier: &coldTier{
				Mu: make(Mutex, 1),
			},
			minioClients: []*mockMinioClient{
				{
					mb: func() (string, error) { return "", errors.New("minio error") },
				},
			},
			setup: func(t *testing.T, in *input) {
				for _, m := range in.minioClients {
					in.cTier.ColdStorages = append(in.cTier.ColdStorages, m)
				}
				in.cTier.SelectNextStorage(in.cTier.ColdStorages, in.cTier.PrevInd)
			},
			wantErr: true,
		},
		{
			name: "Failed cold storage should be removed from list of able storages",
			cTier: &coldTier{
				Mu: make(Mutex, 1),
			},
			minioClients: []*mockMinioClient{
				{
					mb: func() (string, error) { return "", errors.New("minio error") },
				},
				{
					mb: func() (string, error) { return "/path", nil },
				},
			},
			setup: func(t *testing.T, in *input) {
				in.cTier.SelectNextStorage = getColdRBStrategyFunc(in.cTier, in.selectedColdStorageChan)
				for _, m := range in.minioClients {
					in.cTier.ColdStorages = append(in.cTier.ColdStorages, m)
				}
				in.cTier.SelectNextStorage(in.cTier.ColdStorages, in.cTier.PrevInd)
			},
			furtherTets: func(t *testing.T, in *input) {
				for i := 0; i < 10; i++ {
					selectedColdStorage := <-in.selectedColdStorageChan
					require.Equal(t, len(in.cTier.ColdStorages)-1, selectedColdStorage.prevInd)
					in.cTier.SelectNextStorage(in.cTier.ColdStorages, in.cTier.PrevInd)
				}
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			if test.setup != nil {
				test.setup(t, &test)
			}

			_, err := test.cTier.moveBlock("hash", "/a/b")
			if test.wantErr {
				require.NotNil(t, err)
			} else {
				require.Nil(t, err)
			}

			if test.furtherTets != nil {
				test.furtherTets(t, &test)
			}
		})
	}
}
