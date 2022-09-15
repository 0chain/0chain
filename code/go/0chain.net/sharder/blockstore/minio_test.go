package blockstore

import (
	"bytes"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/0chain/common/core/logging"

	"0chain.net/core/viper"
	"github.com/stretchr/testify/require"
)

func init() {
	logging.InitLogging("debug", ".")
}

func getMinioViperConfig(t *testing.T, strategy string) *viper.Viper {
	v := viper.New()
	coldStorageConfigF := `
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
    allowed_block_numbers: "10^15"
    allowed_block_size: "10^5"
`
	coldStorageConfig := fmt.Sprintf(coldStorageConfigF, strategy)
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
	timeout := time.After(time.Minute * 3)
	doneCh := make(chan bool)
	go func(t *testing.T) {
		select {
		case <-timeout:
			fmt.Println("Test timed out")
			t.Fail()
		case <-doneCh:
		}
	}(t)

	type input struct {
		name                string
		cTier               *coldTier
		minioClients        []*mockMinioClient
		storageSelectorChan chan selectedColdStorage
		wantErr             bool
		setup               func(t *testing.T, in *input)
		furtherTets         func(t *testing.T, in *input)
	}

	tests := []*input{
		{
			name: "Should select next cold storage",
			cTier: &coldTier{
				Mu: make(Mutex, 1),
			},
			storageSelectorChan: make(chan selectedColdStorage, 1),
			minioClients: []*mockMinioClient{
				{
					mb: func() (string, error) { return "minio:bkt1/hash", nil },
				}, {
					mb: func() (string, error) { return "minio:bkt2/hash", nil },
				},
			},
			setup: func(t *testing.T, in *input) {
				in.cTier.SelectNextStorage = getColdRBStrategyFunc(in.cTier, in.storageSelectorChan)
				in.cTier.StorageSelectorChan = in.storageSelectorChan
				for _, m := range in.minioClients {
					in.cTier.ColdStorages = append(in.cTier.ColdStorages, m)
				}
				go in.cTier.SelectNextStorage(in.cTier.ColdStorages, in.cTier.PrevInd)
			},
			furtherTets: func(t *testing.T, in *input) {
				prevInd := in.cTier.PrevInd
				selectedStorage := <-in.cTier.StorageSelectorChan
				require.NotEqual(t, prevInd, selectedStorage.prevInd)
			},
		}, {
			name: "Moving block should return error",
			cTier: &coldTier{
				Mu: make(Mutex, 1),
			},
			storageSelectorChan: make(chan selectedColdStorage, 1),
			minioClients: []*mockMinioClient{
				{
					mb: func() (string, error) { return "", errors.New("minio error") },
				},
			},
			setup: func(t *testing.T, in *input) {
				in.cTier.SelectNextStorage = getColdRBStrategyFunc(in.cTier, in.storageSelectorChan)
				in.cTier.StorageSelectorChan = in.storageSelectorChan
				for _, m := range in.minioClients {
					in.cTier.ColdStorages = append(in.cTier.ColdStorages, m)
				}
				go in.cTier.SelectNextStorage(in.cTier.ColdStorages, in.cTier.PrevInd)
			},
			wantErr: true,
		},
		{
			name: "Failed cold storage should be removed from list of able storages",
			cTier: &coldTier{
				Mu: make(Mutex, 1),
			},
			storageSelectorChan: make(chan selectedColdStorage, 1),
			minioClients: []*mockMinioClient{
				{
					mb: func() (string, error) { return "", errors.New("minio error") },
				},
				{
					mb: func() (string, error) { return "/path", nil },
				},
			},
			setup: func(t *testing.T, in *input) {
				in.cTier.SelectNextStorage = getColdRBStrategyFunc(in.cTier, in.storageSelectorChan)
				in.cTier.StorageSelectorChan = in.storageSelectorChan
				for _, m := range in.minioClients {
					in.cTier.ColdStorages = append(in.cTier.ColdStorages, m)
				}
				go in.cTier.SelectNextStorage(in.cTier.ColdStorages, in.cTier.PrevInd)
			},
			furtherTets: func(t *testing.T, in *input) {
				for i := 0; i < 10; i++ {
					selectedColdStorage := <-in.cTier.StorageSelectorChan
					require.Equal(t, 0, selectedColdStorage.prevInd)
					go in.cTier.SelectNextStorage(in.cTier.ColdStorages, in.cTier.PrevInd)
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Logf("running test: %s", test.name)
			if test.setup != nil {
				test.setup(t, test)
			}

			_, err := test.cTier.moveBlock("hash", "/a/b")
			if test.wantErr {
				require.NotNil(t, err)
			} else {
				require.Nil(t, err)
			}

			if test.furtherTets != nil {
				test.furtherTets(t, test)
			}
		})
	}

	doneCh <- true
}
