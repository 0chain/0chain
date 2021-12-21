package chain_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"0chain.net/smartcontract/zcnsc"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/smartcontract"
	"0chain.net/chaincore/state"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/logging"
	"0chain.net/core/memorystore"
	"0chain.net/core/util"
	"0chain.net/core/viper"
	"0chain.net/smartcontract/faucetsc"
	"0chain.net/smartcontract/interestpoolsc"
	"0chain.net/smartcontract/minersc"
	"0chain.net/smartcontract/multisigsc"
	"0chain.net/smartcontract/setupsc"
	"0chain.net/smartcontract/storagesc"
	"0chain.net/smartcontract/vestingsc"
)

func init() {
	config.SetupDefaultConfig()
	viper.Set("development.smart_contract.faucet", true)
	viper.Set("development.smart_contract.interest", true)
	viper.Set("development.smart_contract.miner", true)
	viper.Set("development.smart_contract.storage", true)
	viper.Set("development.smart_contract.vesting", true)
	viper.Set("development.smart_contract.zcn", true)
	viper.Set("development.smart_contract.multisig", true)
	config.SmartContractConfig = viper.New()
	config.SmartContractConfig.Set("smart_contracts.faucetsc.ownerId", "1746b06bb09f55ee01b33b5e2e055d6cc7a900cb57c0a3a5eaabb8a0e7745802")
	config.SmartContractConfig.Set("smart_contracts.minersc.ownerId", "1746b06bb09f55ee01b33b5e2e055d6cc7a900cb57c0a3a5eaabb8a0e7745802")
	config.SmartContractConfig.Set("smart_contracts.interestpoolsc.ownerId", "1746b06bb09f55ee01b33b5e2e055d6cc7a900cb57c0a3a5eaabb8a0e7745802")
	config.SmartContractConfig.Set("smart_contracts.vestingsc.ownerId", "1746b06bb09f55ee01b33b5e2e055d6cc7a900cb57c0a3a5eaabb8a0e7745802")
	config.SmartContractConfig.Set("smart_contracts.storagesc.ownerId", "1746b06bb09f55ee01b33b5e2e055d6cc7a900cb57c0a3a5eaabb8a0e7745802")

	setupsc.SetupSmartContracts()
	logging.InitLogging("development")
	common.ConfigRateLimits()
	block.SetupEntity(memorystore.GetStorageProvider())
}

func TestChain_HandleSCRest_Status(t *testing.T) {
	const (
		clientID     = "client id"
		blobberID    = "blobber_id"
		allocationID = "allocation_id"
	)

	lfb := block.NewBlock("", 1)
	lfb.ClientState = util.NewMerklePatriciaTrie(util.NewMemoryNodeDB(), 1, nil)
	serverChain := chain.NewChainFromConfig()
	serverChain.LatestFinalizedBlock = lfb

	type (
		sortedBlobbers []*storagesc.StorageNode
		storageNodes   struct {
			Nodes sortedBlobbers
		}

		stakePoolConfig struct {
			MinLock          int64         `json:"min_lock"`
			InterestRate     float64       `json:"interest_rate"`
			InterestInterval time.Duration `json:"interest_interval"`
		}
		scConfig struct {
			TimeUnit                        time.Duration    `json:"time_unit"`
			MaxMint                         state.Balance    `json:"max_mint"`
			Minted                          state.Balance    `json:"minted"`
			MinAllocSize                    int64            `json:"min_alloc_size"`
			MinAllocDuration                time.Duration    `json:"min_alloc_duration"`
			MaxChallengeCompletionTime      time.Duration    `json:"max_challenge_completion_time"`
			MinOfferDuration                time.Duration    `json:"min_offer_duration"`
			MinBlobberCapacity              int64            `json:"min_blobber_capacity"`
			ValidatorReward                 float64          `json:"validator_reward"`
			BlobberSlash                    float64          `json:"blobber_slash"`
			MaxReadPrice                    state.Balance    `json:"max_read_price"`
			MaxWritePrice                   state.Balance    `json:"max_write_price"`
			FailedChallengesToCancel        int              `json:"failed_challenges_to_cancel"`
			FailedChallengesToRevokeMinLock int              `json:"failed_challenges_to_revoke_min_lock"`
			ChallengeEnabled                bool             `json:"challenge_enabled"`
			MaxChallengesPerGeneration      int              `json:"max_challenges_per_generation"`
			ChallengeGenerationRate         float64          `json:"challenge_rate_per_mb_min"`
			MinStake                        state.Balance    `json:"min_stake"`
			MaxStake                        state.Balance    `json:"max_stake"`
			MaxDelegates                    int              `json:"max_delegates"`
			MaxCharge                       float64          `json:"max_charge"`
			StakePool                       *stakePoolConfig `json:"stakepool"`
		}

		userStakePools struct {
			Pools map[datastore.Key][]datastore.Key `json:"pools"`
		}
	)

	type args struct {
		w *httptest.ResponseRecorder
		r *http.Request
	}
	tests := []struct {
		name           string
		chain          *chain.Chain
		args           args
		wantStatus     int
		setValidConfig bool
	}{
		{
			name:  "Faucet_/personalPeriodicLimit_Empty_Global_Node_404",
			chain: serverChain,
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					tar := fmt.Sprintf("%v%v%v", "/v1/screst/", faucetsc.ADDRESS, "/personalPeriodicLimit")
					req := httptest.NewRequest(http.MethodGet, tar, nil)

					return req
				}(),
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "Faucet_/personalPeriodicLimit_Decoding_Global_Node_Err_500",
			chain: func() *chain.Chain {
				gv := util.SecureSerializableValue{Buffer: []byte("}{")}

				lfb := block.NewBlock("", 1)
				lfb.ClientState = util.NewMerklePatriciaTrie(util.NewMemoryNodeDB(), 1, nil)
				k := encryption.Hash(faucetsc.ADDRESS + faucetsc.ADDRESS)
				if _, err := lfb.ClientState.Insert(util.Path(k), &gv); err != nil {
					t.Fatal(err)
				}

				ch := chain.NewChainFromConfig()
				ch.LatestFinalizedBlock = lfb

				return ch
			}(),
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					tar := fmt.Sprintf("%v%v%v", "/v1/screst/", faucetsc.ADDRESS, "/personalPeriodicLimit")
					req := httptest.NewRequest(http.MethodGet, tar, nil)

					return req
				}(),
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "Faucet_/personalPeriodicLimit_Empty_User_Node_404",
			chain: func() *chain.Chain {
				gn := &faucetsc.GlobalNode{ID: faucetsc.ADDRESS}
				blob, err := json.Marshal(gn)
				if err != nil {
					t.Fatal(err)
				}
				gv := util.SecureSerializableValue{Buffer: blob}

				lfb := block.NewBlock("", 1)
				lfb.ClientState = util.NewMerklePatriciaTrie(util.NewMemoryNodeDB(), 1, nil)
				k := encryption.Hash(gn.ID + gn.ID)
				if _, err := lfb.ClientState.Insert(util.Path(k), &gv); err != nil {
					t.Fatal(err)
				}

				ch := chain.NewChainFromConfig()
				ch.LatestFinalizedBlock = lfb

				return ch
			}(),
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					tar := fmt.Sprintf("%v%v%v", "/v1/screst/", faucetsc.ADDRESS, "/personalPeriodicLimit")
					req := httptest.NewRequest(http.MethodGet, tar, nil)

					return req
				}(),
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "Faucet_/personalPeriodicLimit_Decoding_User_Node_Err_500",
			chain: func() *chain.Chain {
				gn := &faucetsc.GlobalNode{ID: faucetsc.ADDRESS}
				blob, err := json.Marshal(gn)
				if err != nil {
					t.Fatal(err)
				}
				gv := util.SecureSerializableValue{Buffer: blob}
				gk := encryption.Hash(faucetsc.ADDRESS + faucetsc.ADDRESS)

				uv := util.SecureSerializableValue{Buffer: []byte("}{")}
				uk := encryption.Hash(faucetsc.ADDRESS + clientID)

				lfb := block.NewBlock("", 1)
				lfb.ClientState = util.NewMerklePatriciaTrie(util.NewMemoryNodeDB(), 1, nil)
				if _, err := lfb.ClientState.Insert(util.Path(gk), &gv); err != nil {
					t.Fatal(err)
				}
				if _, err := lfb.ClientState.Insert(util.Path(uk), &uv); err != nil {
					t.Fatal(err)
				}

				ch := chain.NewChainFromConfig()
				ch.LatestFinalizedBlock = lfb

				return ch
			}(),
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					tar := fmt.Sprintf("%v%v%v", "/v1/screst/", faucetsc.ADDRESS, "/personalPeriodicLimit")
					u, err := url.Parse(tar)
					if err != nil {
						t.Fatal(err)
					}
					q := u.Query()
					q.Set("client_id", clientID)
					u.RawQuery = q.Encode()

					return httptest.NewRequest(http.MethodGet, u.String(), nil)
				}(),
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:  "Faucet_/globalPeriodicLimit_Empty_Global_Node_404",
			chain: serverChain,
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					tar := fmt.Sprintf("%v%v%v", "/v1/screst/", faucetsc.ADDRESS, "/globalPeriodicLimit")
					req := httptest.NewRequest(http.MethodGet, tar, nil)

					return req
				}(),
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "Faucet_/globalPeriodicLimit_Decoding_Global_Node_Err_500",
			chain: func() *chain.Chain {
				v := util.SecureSerializableValue{Buffer: []byte("}{")}
				k := encryption.Hash(faucetsc.ADDRESS + faucetsc.ADDRESS)

				lfb := block.NewBlock("", 1)
				lfb.ClientState = util.NewMerklePatriciaTrie(util.NewMemoryNodeDB(), 1, nil)
				if _, err := lfb.ClientState.Insert(util.Path(k), &v); err != nil {
					t.Fatal(err)
				}

				ch := chain.NewChainFromConfig()
				ch.LatestFinalizedBlock = lfb

				return ch
			}(),
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					tar := fmt.Sprintf("%v%v%v", "/v1/screst/", faucetsc.ADDRESS, "/globalPeriodicLimit")
					req := httptest.NewRequest(http.MethodGet, tar, nil)

					return req
				}(),
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:  "Faucet_/pourAmount_Empty_Global_Node_404",
			chain: serverChain,
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					tar := fmt.Sprintf("%v%v%v", "/v1/screst/", faucetsc.ADDRESS, "/pourAmount")
					req := httptest.NewRequest(http.MethodGet, tar, nil)

					return req
				}(),
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "Faucet_/pourAmount_Decoding_Global_Node_500",
			chain: func() *chain.Chain {
				v := util.SecureSerializableValue{Buffer: []byte("}{")}
				k := encryption.Hash(faucetsc.ADDRESS + faucetsc.ADDRESS)

				lfb := block.NewBlock("", 1)
				lfb.ClientState = util.NewMerklePatriciaTrie(util.NewMemoryNodeDB(), 1, nil)
				if _, err := lfb.ClientState.Insert(util.Path(k), &v); err != nil {
					t.Fatal(err)
				}

				ch := chain.NewChainFromConfig()
				ch.LatestFinalizedBlock = lfb

				return ch
			}(),
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					tar := fmt.Sprintf("%v%v%v", "/v1/screst/", faucetsc.ADDRESS, "/pourAmount")
					req := httptest.NewRequest(http.MethodGet, tar, nil)

					return req
				}(),
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:  "Interestpool_/getPoolsStats_Empty_User_Nodes_404",
			chain: serverChain,
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					tar := fmt.Sprintf("%v%v%v", "/v1/screst/", interestpoolsc.ADDRESS, "/getPoolsStats")
					req := httptest.NewRequest(http.MethodGet, tar, nil)

					return req
				}(),
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:  "Minersc_/getNodepool_Decode_Miner_From_Params_Err_400",
			chain: serverChain,
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					tar := fmt.Sprintf("%v%v%v", "/v1/screst/", minersc.ADDRESS, "/getNodepool")
					req := httptest.NewRequest(http.MethodGet, tar, nil)

					return req
				}(),
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:  "Minersc_/getNodepool_Miner_Does_Not_Exist_404",
			chain: serverChain,
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					tar := fmt.Sprintf("%v%v%v", "/v1/screst/", minersc.ADDRESS, "/getNodepool")
					u, err := url.Parse(tar)
					if err != nil {
						t.Fatal(err)
					}
					q := u.Query()
					q.Set("n2n_host", "n2n host")
					q.Set("id", "id")
					u.RawQuery = q.Encode()

					req := httptest.NewRequest(http.MethodGet, u.String(), nil)

					return req
				}(),
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "Minersc_/getUserPools_No_User_Node_500",
			chain: func() *chain.Chain {
				gv := util.SecureSerializableValue{Buffer: []byte("}{")}

				lfb := block.NewBlock("", 1)
				lfb.ClientState = util.NewMerklePatriciaTrie(util.NewMemoryNodeDB(), 1, nil)
				k := encryption.Hash(minersc.ADDRESS + clientID)
				if _, err := lfb.ClientState.Insert(util.Path(k), &gv); err != nil {
					t.Fatal(err)
				}

				ch := chain.NewChainFromConfig()
				ch.LatestFinalizedBlock = lfb

				return ch
			}(),
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					tar := fmt.Sprintf("%v%v%v", "/v1/screst/", minersc.ADDRESS, "/getUserPools")
					u, err := url.Parse(tar)
					if err != nil {
						t.Fatal(err)
					}
					q := u.Query()
					q.Set("client_id", clientID)
					u.RawQuery = q.Encode()

					req := httptest.NewRequest(http.MethodGet, u.String(), nil)

					return req
				}(),
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "Minersc_/getSharderList_Decoding_User_Node_Err_500",
			chain: func() *chain.Chain {
				gv := util.SecureSerializableValue{Buffer: []byte("}{")}

				lfb := block.NewBlock("", 1)
				lfb.ClientState = util.NewMerklePatriciaTrie(util.NewMemoryNodeDB(), 1, nil)
				k := encryption.Hash(minersc.AllShardersKey)
				if _, err := lfb.ClientState.Insert(util.Path(k), &gv); err != nil {
					t.Fatal(err)
				}

				ch := chain.NewChainFromConfig()
				ch.LatestFinalizedBlock = lfb

				return ch
			}(),
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					tar := fmt.Sprintf("%v%v%v", "/v1/screst/", minersc.ADDRESS, "/getSharderList")
					req := httptest.NewRequest(http.MethodGet, tar, nil)

					return req
				}(),
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "Minersc_/nodePoolStat_Decoding_User_Node_Err_500",
			chain: func() *chain.Chain {
				gv := util.SecureSerializableValue{Buffer: []byte("}{")}

				lfb := block.NewBlock("", 1)
				lfb.ClientState = util.NewMerklePatriciaTrie(util.NewMemoryNodeDB(), 1, nil)
				k := encryption.Hash(minersc.ADDRESS)
				if _, err := lfb.ClientState.Insert(util.Path(k), &gv); err != nil {
					t.Fatal(err)
				}

				ch := chain.NewChainFromConfig()
				ch.LatestFinalizedBlock = lfb

				return ch
			}(),
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					tar := fmt.Sprintf("%v%v%v", "/v1/screst/", minersc.ADDRESS, "/nodePoolStat")
					req := httptest.NewRequest(http.MethodGet, tar, nil)

					return req
				}(),
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "Minersc_/configs_Err_500",
			chain: func() *chain.Chain {
				gv := util.SecureSerializableValue{Buffer: []byte("}{")}

				lfb := block.NewBlock("", 1)
				lfb.ClientState = util.NewMerklePatriciaTrie(util.NewMemoryNodeDB(), 1, nil)
				k := encryption.Hash(minersc.GlobalNodeKey)
				if _, err := lfb.ClientState.Insert(util.Path(k), &gv); err != nil {
					t.Fatal(err)
				}

				ch := chain.NewChainFromConfig()
				ch.LatestFinalizedBlock = lfb

				return ch
			}(),
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					tar := fmt.Sprintf("%v%v%v", "/v1/screst/", minersc.ADDRESS, "/configs")
					req := httptest.NewRequest(http.MethodGet, tar, nil)

					return req
				}(),
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "Minersc_/getMinerList_DEcoding_User_Node_Err_500",
			chain: func() *chain.Chain {
				gv := util.SecureSerializableValue{Buffer: []byte("}{")}

				lfb := block.NewBlock("", 1)
				lfb.ClientState = util.NewMerklePatriciaTrie(util.NewMemoryNodeDB(), 1, nil)
				k := encryption.Hash(minersc.AllMinersKey)
				if _, err := lfb.ClientState.Insert(util.Path(k), &gv); err != nil {
					t.Fatal(err)
				}

				ch := chain.NewChainFromConfig()
				ch.LatestFinalizedBlock = lfb

				return ch
			}(),
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					tar := fmt.Sprintf("%v%v%v", "/v1/screst/", minersc.ADDRESS, "/getMinerList")
					req := httptest.NewRequest(http.MethodGet, tar, nil)

					return req
				}(),
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "Minersc_/getSharderKeepList_Decoding_Err_500",
			chain: func() *chain.Chain {
				gv := util.SecureSerializableValue{Buffer: []byte("}{")}

				lfb := block.NewBlock("", 1)
				lfb.ClientState = util.NewMerklePatriciaTrie(util.NewMemoryNodeDB(), 1, nil)
				k := encryption.Hash(minersc.ShardersKeepKey)
				if _, err := lfb.ClientState.Insert(util.Path(k), &gv); err != nil {
					t.Fatal(err)
				}

				ch := chain.NewChainFromConfig()
				ch.LatestFinalizedBlock = lfb

				return ch
			}(),
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					tar := fmt.Sprintf("%v%v%v", "/v1/screst/", minersc.ADDRESS, "/getSharderKeepList")
					req := httptest.NewRequest(http.MethodGet, tar, nil)

					return req
				}(),
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "Minersc_/getDkgList_Decoding_Err_500",
			chain: func() *chain.Chain {
				gv := util.SecureSerializableValue{Buffer: []byte("}{")}

				lfb := block.NewBlock("", 1)
				lfb.ClientState = util.NewMerklePatriciaTrie(util.NewMemoryNodeDB(), 1, nil)
				k := encryption.Hash(minersc.DKGMinersKey)
				if _, err := lfb.ClientState.Insert(util.Path(k), &gv); err != nil {
					t.Fatal(err)
				}

				ch := chain.NewChainFromConfig()
				ch.LatestFinalizedBlock = lfb

				return ch
			}(),
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					tar := fmt.Sprintf("%v%v%v", "/v1/screst/", minersc.ADDRESS, "/getDkgList")
					req := httptest.NewRequest(http.MethodGet, tar, nil)

					return req
				}(),
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "Minersc_/nodeStat_Decoding_Err_500",
			chain: func() *chain.Chain {
				gv := util.SecureSerializableValue{Buffer: []byte("}{")}

				lfb := block.NewBlock("", 1)
				lfb.ClientState = util.NewMerklePatriciaTrie(util.NewMemoryNodeDB(), 1, nil)
				k := encryption.Hash(minersc.ADDRESS + clientID)
				if _, err := lfb.ClientState.Insert(util.Path(k), &gv); err != nil {
					t.Fatal(err)
				}

				ch := chain.NewChainFromConfig()
				ch.LatestFinalizedBlock = lfb

				return ch
			}(),
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					tar := fmt.Sprintf("%v%v%v", "/v1/screst/", minersc.ADDRESS, "/nodeStat")
					u, err := url.Parse(tar)
					if err != nil {
						t.Fatal(err)
					}
					q := u.Query()
					q.Set("id", clientID)
					u.RawQuery = q.Encode()

					req := httptest.NewRequest(http.MethodGet, u.String(), nil)

					return req
				}(),
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "Minersc_/getUserPools_Fail_Retrieving_Miners_Node_404",
			chain: func() *chain.Chain {
				un := minersc.UserNode{
					ID: clientID,
					Pools: map[datastore.Key][]datastore.Key{
						"key": {},
					},
				}
				blob, err := json.Marshal(un)
				if err != nil {
					t.Fatal(err)
				}
				gv := util.SecureSerializableValue{Buffer: blob}

				lfb := block.NewBlock("", 1)
				lfb.ClientState = util.NewMerklePatriciaTrie(util.NewMemoryNodeDB(), 1, nil)
				k := encryption.Hash(minersc.ADDRESS + clientID)
				if _, err := lfb.ClientState.Insert(util.Path(k), &gv); err != nil {
					t.Fatal(err)
				}

				ch := chain.NewChainFromConfig()
				ch.LatestFinalizedBlock = lfb

				return ch
			}(),
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					tar := fmt.Sprintf("%v%v%v", "/v1/screst/", minersc.ADDRESS, "/getUserPools")
					u, err := url.Parse(tar)
					if err != nil {
						t.Fatal(err)
					}
					q := u.Query()
					q.Set("client_id", clientID)
					u.RawQuery = q.Encode()

					req := httptest.NewRequest(http.MethodGet, u.String(), nil)

					return req
				}(),
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "Minersc_/getUserPools_Decoding_Miners_Node_Err_500",
			chain: func() *chain.Chain {
				minerID := "miner id"

				un := minersc.UserNode{
					ID: clientID,
					Pools: map[datastore.Key][]datastore.Key{
						minerID: {},
					},
				}
				blob, err := json.Marshal(un)
				if err != nil {
					t.Fatal(err)
				}
				gv := util.SecureSerializableValue{Buffer: blob}
				gk := encryption.Hash(minersc.ADDRESS + clientID)

				mv := util.SecureSerializableValue{Buffer: []byte("}{")}
				mk := encryption.Hash(minersc.ADDRESS + minerID)

				lfb := block.NewBlock("", 1)
				lfb.ClientState = util.NewMerklePatriciaTrie(util.NewMemoryNodeDB(), 1, nil)
				if _, err := lfb.ClientState.Insert(util.Path(gk), &gv); err != nil {
					t.Fatal(err)
				}
				if _, err := lfb.ClientState.Insert(util.Path(mk), &mv); err != nil {
					t.Fatal(err)
				}

				ch := chain.NewChainFromConfig()
				ch.LatestFinalizedBlock = lfb

				return ch
			}(),
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					tar := fmt.Sprintf("%v%v%v", "/v1/screst/", minersc.ADDRESS, "/getUserPools")
					u, err := url.Parse(tar)
					if err != nil {
						t.Fatal(err)
					}
					q := u.Query()
					q.Set("client_id", clientID)
					u.RawQuery = q.Encode()

					req := httptest.NewRequest(http.MethodGet, u.String(), nil)

					return req
				}(),
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:  "Minersc_/getMpksList_Empty_Miners_Mpks_404",
			chain: serverChain,
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					tar := fmt.Sprintf("%v%v%v", "/v1/screst/", minersc.ADDRESS, "/getMpksList")
					req := httptest.NewRequest(http.MethodGet, tar, nil)

					return req
				}(),
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "Minersc_/getMpksList_Decoding_Err_500",
			chain: func() *chain.Chain {
				gv := util.SecureSerializableValue{Buffer: []byte("}{")}

				lfb := block.NewBlock("", 1)
				lfb.ClientState = util.NewMerklePatriciaTrie(util.NewMemoryNodeDB(), 1, nil)
				k := encryption.Hash(minersc.MinersMPKKey)
				if _, err := lfb.ClientState.Insert(util.Path(k), &gv); err != nil {
					t.Fatal(err)
				}

				ch := chain.NewChainFromConfig()
				ch.LatestFinalizedBlock = lfb

				return ch
			}(),
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					tar := fmt.Sprintf("%v%v%v", "/v1/screst/", minersc.ADDRESS, "/getMpksList")
					req := httptest.NewRequest(http.MethodGet, tar, nil)

					return req
				}(),
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:  "Minersc_/getGroupShareOrSigns_Empty_SOS_404",
			chain: serverChain,
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					tar := fmt.Sprintf("%v%v%v", "/v1/screst/", minersc.ADDRESS, "/getGroupShareOrSigns")
					req := httptest.NewRequest(http.MethodGet, tar, nil)

					return req
				}(),
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "Minersc_/getGroupShareOrSigns_Decoding_Err_500",
			chain: func() *chain.Chain {
				gv := util.SecureSerializableValue{Buffer: []byte("}{")}

				lfb := block.NewBlock("", 1)
				lfb.ClientState = util.NewMerklePatriciaTrie(util.NewMemoryNodeDB(), 1, nil)
				k := encryption.Hash(minersc.GroupShareOrSignsKey)
				if _, err := lfb.ClientState.Insert(util.Path(k), &gv); err != nil {
					t.Fatal(err)
				}

				ch := chain.NewChainFromConfig()
				ch.LatestFinalizedBlock = lfb

				return ch
			}(),
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					tar := fmt.Sprintf("%v%v%v", "/v1/screst/", minersc.ADDRESS, "/getGroupShareOrSigns")
					req := httptest.NewRequest(http.MethodGet, tar, nil)

					return req
				}(),
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:  "Minersc_/getMagicBlock_Empty_Magic_Block_404",
			chain: serverChain,
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					tar := fmt.Sprintf("%v%v%v", "/v1/screst/", minersc.ADDRESS, "/getMagicBlock")
					req := httptest.NewRequest(http.MethodGet, tar, nil)

					return req
				}(),
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "Minersc_/getMagicBlock_Decoding_Err_500",
			chain: func() *chain.Chain {
				gv := util.SecureSerializableValue{Buffer: []byte("}{")}

				lfb := block.NewBlock("", 1)
				lfb.ClientState = util.NewMerklePatriciaTrie(util.NewMemoryNodeDB(), 1, nil)
				k := encryption.Hash(minersc.MagicBlockKey)
				if _, err := lfb.ClientState.Insert(util.Path(k), &gv); err != nil {
					t.Fatal(err)
				}

				ch := chain.NewChainFromConfig()
				ch.LatestFinalizedBlock = lfb

				return ch
			}(),
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					tar := fmt.Sprintf("%v%v%v", "/v1/screst/", minersc.ADDRESS, "/getMagicBlock")
					req := httptest.NewRequest(http.MethodGet, tar, nil)

					return req
				}(),
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "Minersc_/nodePoolStat_Not_Found_404",
			chain: func() *chain.Chain {
				mn := minersc.NewMinerNode()
				blob, err := json.Marshal(mn)
				if err != nil {
					t.Fatal(err)
				}

				gv := util.SecureSerializableValue{Buffer: blob}

				lfb := block.NewBlock("", 1)
				lfb.ClientState = util.NewMerklePatriciaTrie(util.NewMemoryNodeDB(), 1, nil)
				k := encryption.Hash(minersc.ADDRESS + clientID)
				if _, err := lfb.ClientState.Insert(util.Path(k), &gv); err != nil {
					t.Fatal(err)
				}

				ch := chain.NewChainFromConfig()
				ch.LatestFinalizedBlock = lfb

				return ch
			}(),
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					tar := fmt.Sprintf("%v%v%v", "/v1/screst/", minersc.ADDRESS, "/nodePoolStat")
					u, err := url.Parse(tar)
					if err != nil {
						t.Fatal(err)
					}
					q := u.Query()
					q.Set("id", clientID)
					u.RawQuery = q.Encode()

					req := httptest.NewRequest(http.MethodGet, u.String(), nil)

					return req
				}(),
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "Storagesc_/getConfig_500",
			chain: func() *chain.Chain {
				gv := util.SecureSerializableValue{Buffer: []byte("}{")}

				lfb := block.NewBlock("", 1)
				lfb.ClientState = util.NewMerklePatriciaTrie(util.NewMemoryNodeDB(), 1, nil)
				k := encryption.Hash(storagesc.ADDRESS + ":configurations")
				if _, err := lfb.ClientState.Insert(util.Path(k), &gv); err != nil {
					t.Fatal(err)
				}

				ch := chain.NewChainFromConfig()
				ch.LatestFinalizedBlock = lfb

				return ch
			}(),
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					tar := fmt.Sprintf("%v%v%v", "/v1/screst/", storagesc.ADDRESS, "/getConfig")
					u, err := url.Parse(tar)
					if err != nil {
						t.Fatal(err)
					}
					q := u.Query()
					q.Set("id", clientID)
					u.RawQuery = q.Encode()

					req := httptest.NewRequest(http.MethodGet, u.String(), nil)

					return req
				}(),
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:  "Storagesc_/getConfig_500",
			chain: serverChain,
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					tar := fmt.Sprintf("%v%v%v", "/v1/screst/", storagesc.ADDRESS, "/getConfig")
					req := httptest.NewRequest(http.MethodGet, tar, nil)

					return req
				}(),
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "Storagesc_/latestreadmarker_500",
			chain: func() *chain.Chain {
				gv := util.SecureSerializableValue{Buffer: []byte("}{")}

				lfb := block.NewBlock("", 1)
				lfb.ClientState = util.NewMerklePatriciaTrie(util.NewMemoryNodeDB(), 1, nil)
				k := encryption.Hash(storagesc.ADDRESS + encryption.Hash(":"))
				if _, err := lfb.ClientState.Insert(util.Path(k), &gv); err != nil {
					t.Fatal(err)
				}

				ch := chain.NewChainFromConfig()
				ch.LatestFinalizedBlock = lfb

				return ch
			}(),
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					tar := fmt.Sprintf("%v%v%v", "/v1/screst/", storagesc.ADDRESS, "/latestreadmarker")
					req := httptest.NewRequest(http.MethodGet, tar, nil)

					return req
				}(),
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:  "Storagesc_/allocation_No_Allocation_404",
			chain: serverChain,
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					tar := fmt.Sprintf("%v%v%v", "/v1/screst/", storagesc.ADDRESS, "/allocation")
					req := httptest.NewRequest(http.MethodGet, tar, nil)

					return req
				}(),
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "Storagesc_/allocation_Decoding_Err_500",
			chain: func() *chain.Chain {
				gv := util.SecureSerializableValue{Buffer: []byte("}{")}

				lfb := block.NewBlock("", 1)
				lfb.ClientState = util.NewMerklePatriciaTrie(util.NewMemoryNodeDB(), 1, nil)
				k := encryption.Hash(storagesc.ADDRESS)
				if _, err := lfb.ClientState.Insert(util.Path(k), &gv); err != nil {
					t.Fatal(err)
				}

				ch := chain.NewChainFromConfig()
				ch.LatestFinalizedBlock = lfb

				return ch
			}(),
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					tar := fmt.Sprintf("%v%v%v", "/v1/screst/", storagesc.ADDRESS, "/allocation")
					req := httptest.NewRequest(http.MethodGet, tar, nil)

					return req
				}(),
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "Storagesc_/allocations_Get_List_Err_500",
			chain: func() *chain.Chain {
				gv := util.SecureSerializableValue{Buffer: []byte("}{")}

				lfb := block.NewBlock("", 1)
				lfb.ClientState = util.NewMerklePatriciaTrie(util.NewMemoryNodeDB(), 1, nil)
				k := encryption.Hash(storagesc.ADDRESS)
				if _, err := lfb.ClientState.Insert(util.Path(k), &gv); err != nil {
					t.Fatal(err)
				}

				ch := chain.NewChainFromConfig()
				ch.LatestFinalizedBlock = lfb

				return ch
			}(),
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					tar := fmt.Sprintf("%v%v%v", "/v1/screst/", storagesc.ADDRESS, "/allocations")
					req := httptest.NewRequest(http.MethodGet, tar, nil)

					return req
				}(),
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:  "Storagesc_/allocation_min_lock_500",
			chain: serverChain,
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					tar := fmt.Sprintf("%v%v%v", "/v1/screst/", storagesc.ADDRESS, "/allocation_min_lock")
					req := httptest.NewRequest(http.MethodGet, tar, nil)

					return req
				}(),
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:  "Storagesc_/allocation_min_lock_Invalid_Config_Err_500",
			chain: serverChain,
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					type newAllocationRequest struct {
						DataShards                 int                  `json:"data_shards"`
						ParityShards               int                  `json:"parity_shards"`
						Size                       int64                `json:"size"`
						Expiration                 common.Timestamp     `json:"expiration_date"`
						Owner                      string               `json:"owner_id"`
						OwnerPublicKey             string               `json:"owner_public_key"`
						PreferredBlobbers          []string             `json:"preferred_blobbers"`
						ReadPriceRange             storagesc.PriceRange `json:"read_price_range"`
						WritePriceRange            storagesc.PriceRange `json:"write_price_range"`
						MaxChallengeCompletionTime time.Duration        `json:"max_challenge_completion_time"`
					}
					allocReq := &newAllocationRequest{}
					blob, err := json.Marshal(allocReq)
					if err != nil {
						t.Fatal(err)
					}

					tar := fmt.Sprintf("%v%v%v", "/v1/screst/", storagesc.ADDRESS, "/allocation_min_lock")
					u, err := url.Parse(tar)
					if err != nil {
						t.Fatal(err)
					}
					q := u.Query()
					q.Set("allocation_data", string(blob))
					u.RawQuery = q.Encode()

					req := httptest.NewRequest(http.MethodGet, u.String(), nil)

					return req
				}(),
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "Storagesc_/allocation_min_lock_Invalid_Config_500",
			chain: func() *chain.Chain {
				sn := storageNodes{
					Nodes: []*storagesc.StorageNode{
						{},
					},
				}
				blob, err := json.Marshal(sn)
				if err != nil {
					t.Fatal(err)
				}
				gv := util.SecureSerializableValue{Buffer: blob}

				lfb := block.NewBlock("", 1)
				lfb.ClientState = util.NewMerklePatriciaTrie(util.NewMemoryNodeDB(), 1, nil)
				k := encryption.Hash(storagesc.ALL_BLOBBERS_KEY)
				if _, err := lfb.ClientState.Insert(util.Path(k), &gv); err != nil {
					t.Fatal(err)
				}

				ch := chain.NewChainFromConfig()
				ch.LatestFinalizedBlock = lfb

				return ch
			}(),
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					type newAllocationRequest struct {
						DataShards                 int                  `json:"data_shards"`
						ParityShards               int                  `json:"parity_shards"`
						Size                       int64                `json:"size"`
						Expiration                 common.Timestamp     `json:"expiration_date"`
						Owner                      string               `json:"owner_id"`
						OwnerPublicKey             string               `json:"owner_public_key"`
						PreferredBlobbers          []string             `json:"preferred_blobbers"`
						ReadPriceRange             storagesc.PriceRange `json:"read_price_range"`
						WritePriceRange            storagesc.PriceRange `json:"write_price_range"`
						MaxChallengeCompletionTime time.Duration        `json:"max_challenge_completion_time"`
					}
					allocReq := &newAllocationRequest{
						Size:           2048,
						Expiration:     common.Timestamp(time.Now().Add(time.Hour).Unix()),
						DataShards:     1,
						OwnerPublicKey: "owners public key",
						Owner:          "owner",
					}
					blob, err := json.Marshal(allocReq)
					if err != nil {
						t.Fatal(err)
					}

					tar := fmt.Sprintf("%v%v%v", "/v1/screst/", storagesc.ADDRESS, "/allocation_min_lock")
					u, err := url.Parse(tar)
					if err != nil {
						t.Fatal(err)
					}
					q := u.Query()
					q.Set("allocation_data", string(blob))
					u.RawQuery = q.Encode()

					req := httptest.NewRequest(http.MethodGet, u.String(), nil)

					return req
				}(),
			},
			setValidConfig: true,
			wantStatus:     http.StatusInternalServerError,
		},
		{
			name:  "Storagesc_/openchallenges_404",
			chain: serverChain,
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					tar := fmt.Sprintf("%v%v%v", "/v1/screst/", storagesc.ADDRESS, "/openchallenges")
					req := httptest.NewRequest(http.MethodGet, tar, nil)

					return req
				}(),
			},
			setValidConfig: true,
			wantStatus:     http.StatusNotFound,
		},
		{
			name: "Storagesc_/openchallenges_404",
			chain: func() *chain.Chain {
				gv := util.SecureSerializableValue{Buffer: []byte("}{")}

				lfb := block.NewBlock("", 1)
				lfb.ClientState = util.NewMerklePatriciaTrie(util.NewMemoryNodeDB(), 1, nil)
				k := encryption.Hash(storagesc.ADDRESS + ":blobberchallenge:")
				if _, err := lfb.ClientState.Insert(util.Path(k), &gv); err != nil {
					t.Fatal(err)
				}

				ch := chain.NewChainFromConfig()
				ch.LatestFinalizedBlock = lfb

				return ch
			}(),
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					tar := fmt.Sprintf("%v%v%v", "/v1/screst/", storagesc.ADDRESS, "/openchallenges")
					req := httptest.NewRequest(http.MethodGet, tar, nil)

					return req
				}(),
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:  "Storagesc_/getchallenge_404",
			chain: serverChain,
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					tar := fmt.Sprintf("%v%v%v", "/v1/screst/", storagesc.ADDRESS, "/getchallenge")
					req := httptest.NewRequest(http.MethodGet, tar, nil)

					return req
				}(),
			},
			setValidConfig: true,
			wantStatus:     http.StatusNotFound,
		},
		{
			name: "Storagesc_/getchallenge_500",
			chain: func() *chain.Chain {
				gv := util.SecureSerializableValue{Buffer: []byte("}{")}

				lfb := block.NewBlock("", 1)
				lfb.ClientState = util.NewMerklePatriciaTrie(util.NewMemoryNodeDB(), 1, nil)
				k := encryption.Hash(storagesc.ADDRESS + ":blobberchallenge:")
				if _, err := lfb.ClientState.Insert(util.Path(k), &gv); err != nil {
					t.Fatal(err)
				}

				ch := chain.NewChainFromConfig()
				ch.LatestFinalizedBlock = lfb

				return ch
			}(),
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					tar := fmt.Sprintf("%v%v%v", "/v1/screst/", storagesc.ADDRESS, "/getchallenge")
					req := httptest.NewRequest(http.MethodGet, tar, nil)

					return req
				}(),
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "Storagesc_/getblobbers_500",
			chain: func() *chain.Chain {
				gv := util.SecureSerializableValue{Buffer: []byte("}{")}

				lfb := block.NewBlock("", 1)
				lfb.ClientState = util.NewMerklePatriciaTrie(util.NewMemoryNodeDB(), 1, nil)
				k := encryption.Hash(storagesc.ALL_BLOBBERS_KEY)
				if _, err := lfb.ClientState.Insert(util.Path(k), &gv); err != nil {
					t.Fatal(err)
				}

				ch := chain.NewChainFromConfig()
				ch.LatestFinalizedBlock = lfb

				return ch
			}(),
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					tar := fmt.Sprintf("%v%v%v", "/v1/screst/", storagesc.ADDRESS, "/getblobbers")
					req := httptest.NewRequest(http.MethodGet, tar, nil)

					return req
				}(),
			},
			setValidConfig: true,
			wantStatus:     http.StatusInternalServerError,
		},
		{
			name:  "Storagesc_/getBlobber_400",
			chain: serverChain,
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					tar := fmt.Sprintf("%v%v%v", "/v1/screst/", storagesc.ADDRESS, "/getBlobber")
					req := httptest.NewRequest(http.MethodGet, tar, nil)

					return req
				}(),
			},
			setValidConfig: true,
			wantStatus:     http.StatusBadRequest,
		},
		{
			name:  "Storagesc_/getBlobber_404",
			chain: serverChain,
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					tar := fmt.Sprintf("%v%v%v", "/v1/screst/", storagesc.ADDRESS, "/getBlobber")
					u, err := url.Parse(tar)
					if err != nil {
						t.Fatal(err)
					}
					q := u.Query()
					q.Set("blobber_id", "blobber_id")
					u.RawQuery = q.Encode()

					req := httptest.NewRequest(http.MethodGet, u.String(), nil)

					return req
				}(),
			},
			setValidConfig: true,
			wantStatus:     http.StatusNotFound,
		},
		{
			name:  "Storagesc_/getReadPoolStat_404",
			chain: serverChain,
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					tar := fmt.Sprintf("%v%v%v", "/v1/screst/", storagesc.ADDRESS, "/getReadPoolStat")
					req := httptest.NewRequest(http.MethodGet, tar, nil)

					return req
				}(),
			},
			setValidConfig: true,
			wantStatus:     http.StatusNotFound,
		},
		{
			name:  "Storagesc_/getReadPoolAllocBlobberStat_404",
			chain: serverChain,
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					tar := fmt.Sprintf("%v%v%v", "/v1/screst/", storagesc.ADDRESS, "/getReadPoolAllocBlobberStat")
					req := httptest.NewRequest(http.MethodGet, tar, nil)

					return req
				}(),
			},
			setValidConfig: true,
			wantStatus:     http.StatusNotFound,
		},
		{
			name:  "Storagesc_/getWritePoolStat_404",
			chain: serverChain,
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					tar := fmt.Sprintf("%v%v%v", "/v1/screst/", storagesc.ADDRESS, "/getWritePoolStat")
					req := httptest.NewRequest(http.MethodGet, tar, nil)

					return req
				}(),
			},
			setValidConfig: true,
			wantStatus:     http.StatusNotFound,
		},
		{
			name:  "Storagesc_/getWritePoolAllocBlobberStat_404",
			chain: serverChain,
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					tar := fmt.Sprintf("%v%v%v", "/v1/screst/", storagesc.ADDRESS, "/getWritePoolAllocBlobberStat")
					req := httptest.NewRequest(http.MethodGet, tar, nil)

					return req
				}(),
			},
			setValidConfig: true,
			wantStatus:     http.StatusNotFound,
		},
		{
			name:  "Storagesc_/getStakePoolStat_No_Config_404",
			chain: serverChain,
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					tar := fmt.Sprintf("%v%v%v", "/v1/screst/", storagesc.ADDRESS, "/getStakePoolStat")
					req := httptest.NewRequest(http.MethodGet, tar, nil)

					return req
				}(),
			},
			setValidConfig: true,
			wantStatus:     http.StatusNotFound,
		},
		{
			name: "Storagesc_/getStakePoolStat_No_Blobber_404",
			chain: func() *chain.Chain {
				blob, err := json.Marshal(&scConfig{})
				if err != nil {
					t.Fatal(err)
				}
				gv := util.SecureSerializableValue{Buffer: blob}

				lfb := block.NewBlock("", 1)
				lfb.ClientState = util.NewMerklePatriciaTrie(util.NewMemoryNodeDB(), 1, nil)
				k := encryption.Hash(storagesc.ADDRESS + ":configurations")
				if _, err := lfb.ClientState.Insert(util.Path(k), &gv); err != nil {
					t.Fatal(err)
				}

				ch := chain.NewChainFromConfig()
				ch.LatestFinalizedBlock = lfb

				return ch
			}(),
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					tar := fmt.Sprintf("%v%v%v", "/v1/screst/", storagesc.ADDRESS, "/getStakePoolStat")
					req := httptest.NewRequest(http.MethodGet, tar, nil)

					return req
				}(),
			},
			setValidConfig: true,
			wantStatus:     http.StatusNotFound,
		},
		{
			name: "Storagesc_/getStakePoolStat_No_Stake_Pool_404",
			chain: func() *chain.Chain {
				blob, err := json.Marshal(&scConfig{})
				if err != nil {
					t.Fatal(err)
				}
				v := util.SecureSerializableValue{Buffer: blob}

				lfb := block.NewBlock("", 1)
				lfb.ClientState = util.NewMerklePatriciaTrie(util.NewMemoryNodeDB(), 1, nil)
				k := encryption.Hash(storagesc.ADDRESS + ":configurations")
				if _, err := lfb.ClientState.Insert(util.Path(k), &v); err != nil {
					t.Fatal(err)
				}

				bl := storagesc.StorageNode{}
				blob, err = json.Marshal(bl)
				if err != nil {
					t.Fatal(err)
				}
				v2 := util.SecureSerializableValue{Buffer: blob}
				k2 := encryption.Hash(storagesc.ADDRESS + blobberID)
				if _, err := lfb.ClientState.Insert(util.Path(k2), &v2); err != nil {
					t.Fatal(err)
				}

				ch := chain.NewChainFromConfig()
				ch.LatestFinalizedBlock = lfb

				return ch
			}(),
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					tar := fmt.Sprintf("%v%v%v", "/v1/screst/", storagesc.ADDRESS, "/getStakePoolStat")
					u, err := url.Parse(tar)
					if err != nil {
						t.Fatal(err)
					}
					q := u.Query()
					q.Set("blobber_id", blobberID)
					u.RawQuery = q.Encode()

					req := httptest.NewRequest(http.MethodGet, u.String(), nil)

					return req
				}(),
			},
			setValidConfig: true,
			wantStatus:     http.StatusNotFound,
		},
		{
			name:  "Storagesc_/getUserStakePoolStat_No_Config_404",
			chain: serverChain,
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					tar := fmt.Sprintf("%v%v%v", "/v1/screst/", storagesc.ADDRESS, "/getUserStakePoolStat")
					req := httptest.NewRequest(http.MethodGet, tar, nil)

					return req
				}(),
			},
			setValidConfig: true,
			wantStatus:     http.StatusNotFound,
		},
		{
			name: "Storagesc_/getUserStakePoolStat_No_User_Stake_Pool_404",
			chain: func() *chain.Chain {
				conf := &scConfig{
					StakePool: &stakePoolConfig{},
				}
				blob, err := json.Marshal(conf)
				if err != nil {
					t.Fatal(err)
				}
				gv := util.SecureSerializableValue{Buffer: blob}

				lfb := block.NewBlock("", 1)
				lfb.ClientState = util.NewMerklePatriciaTrie(util.NewMemoryNodeDB(), 1, nil)
				k := encryption.Hash(storagesc.ADDRESS + ":configurations")
				if _, err := lfb.ClientState.Insert(util.Path(k), &gv); err != nil {
					t.Fatal(err)
				}

				ch := chain.NewChainFromConfig()
				ch.LatestFinalizedBlock = lfb

				return ch
			}(),
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					tar := fmt.Sprintf("%v%v%v", "/v1/screst/", storagesc.ADDRESS, "/getUserStakePoolStat")
					req := httptest.NewRequest(http.MethodGet, tar, nil)

					return req
				}(),
			},
			setValidConfig: true,
			wantStatus:     http.StatusNotFound,
		},
		{
			name: "Storagesc_/getUserStakePoolStat_No_Stake_Pool_404",
			chain: func() *chain.Chain {
				conf := &scConfig{
					StakePool: &stakePoolConfig{},
				}
				blob, err := json.Marshal(conf)
				if err != nil {
					t.Fatal(err)
				}
				v := util.SecureSerializableValue{Buffer: blob}

				lfb := block.NewBlock("", 1)
				lfb.ClientState = util.NewMerklePatriciaTrie(util.NewMemoryNodeDB(), 1, nil)
				k := encryption.Hash(storagesc.ADDRESS + ":configurations")
				if _, err := lfb.ClientState.Insert(util.Path(k), &v); err != nil {
					t.Fatal(err)
				}

				sp := &userStakePools{
					Pools: map[datastore.Key][]datastore.Key{
						"key": {"key"},
					},
				}
				blob, err = json.Marshal(sp)
				if err != nil {
					t.Fatal(err)
				}
				v2 := util.SecureSerializableValue{Buffer: blob}
				k2 := encryption.Hash(storagesc.ADDRESS + ":stakepool:userpools:")
				if _, err := lfb.ClientState.Insert(util.Path(k2), &v2); err != nil {
					t.Fatal(err)
				}

				ch := chain.NewChainFromConfig()
				ch.LatestFinalizedBlock = lfb

				return ch
			}(),
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					tar := fmt.Sprintf("%v%v%v", "/v1/screst/", storagesc.ADDRESS, "/getUserStakePoolStat")
					req := httptest.NewRequest(http.MethodGet, tar, nil)

					return req
				}(),
			},
			setValidConfig: true,
			wantStatus:     http.StatusNotFound,
		},
		{
			name:  "Storagesc_/getChallengePoolStat_400",
			chain: serverChain,
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					tar := fmt.Sprintf("%v%v%v", "/v1/screst/", storagesc.ADDRESS, "/getChallengePoolStat")
					req := httptest.NewRequest(http.MethodGet, tar, nil)

					return req
				}(),
			},
			setValidConfig: true,
			wantStatus:     http.StatusBadRequest,
		},
		{
			name: "Storagesc_/getChallengePoolStat_de_Allocation_500",
			chain: func() *chain.Chain {
				v := util.SecureSerializableValue{Buffer: []byte("}{")}

				lfb := block.NewBlock("", 1)
				lfb.ClientState = util.NewMerklePatriciaTrie(util.NewMemoryNodeDB(), 1, nil)
				k := encryption.Hash(storagesc.ADDRESS + allocationID)
				if _, err := lfb.ClientState.Insert(util.Path(k), &v); err != nil {
					t.Fatal(err)
				}

				ch := chain.NewChainFromConfig()
				ch.LatestFinalizedBlock = lfb

				return ch
			}(),
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					tar := fmt.Sprintf("%v%v%v", "/v1/screst/", storagesc.ADDRESS, "/getChallengePoolStat")
					u, err := url.Parse(tar)
					if err != nil {
						t.Fatal(err)
					}
					q := u.Query()
					q.Set("allocation_id", allocationID)
					u.RawQuery = q.Encode()

					req := httptest.NewRequest(http.MethodGet, u.String(), nil)

					return req
				}(),
			},
			setValidConfig: true,
			wantStatus:     http.StatusInternalServerError,
		},
		{
			name: "Storagesc_/getChallengePoolStat_No_Challenge_Pool_404",
			chain: func() *chain.Chain {
				sa := &storagesc.StorageAllocation{}
				blob, err := json.Marshal(sa)
				if err != nil {
					t.Fatal(err)
				}

				v := util.SecureSerializableValue{Buffer: blob}

				lfb := block.NewBlock("", 1)
				lfb.ClientState = util.NewMerklePatriciaTrie(util.NewMemoryNodeDB(), 1, nil)
				k := encryption.Hash(storagesc.ADDRESS + allocationID)
				if _, err := lfb.ClientState.Insert(util.Path(k), &v); err != nil {
					t.Fatal(err)
				}

				ch := chain.NewChainFromConfig()
				ch.LatestFinalizedBlock = lfb

				return ch
			}(),
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					tar := fmt.Sprintf("%v%v%v", "/v1/screst/", storagesc.ADDRESS, "/getChallengePoolStat")
					u, err := url.Parse(tar)
					if err != nil {
						t.Fatal(err)
					}
					q := u.Query()
					q.Set("allocation_id", allocationID)
					u.RawQuery = q.Encode()

					req := httptest.NewRequest(http.MethodGet, u.String(), nil)

					return req
				}(),
			},
			setValidConfig: true,
			wantStatus:     http.StatusNotFound,
		},
		{
			name:  "Vestingsc_/getConfig_500",
			chain: serverChain,
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					tar := fmt.Sprintf("%v%v%v", "/v1/screst/", vestingsc.ADDRESS, "/getConfig")
					req := httptest.NewRequest(http.MethodGet, tar, nil)

					return req
				}(),
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:  "Vestingsc_/getPoolInfo_404",
			chain: serverChain,
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					tar := fmt.Sprintf("%v%v%v", "/v1/screst/", vestingsc.ADDRESS, "/getPoolInfo")
					req := httptest.NewRequest(http.MethodGet, tar, nil)

					return req
				}(),
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "Vestingsc_/getClientPools_500",
			chain: func() *chain.Chain {
				v := util.SecureSerializableValue{Buffer: []byte("}{")}

				lfb := block.NewBlock("", 1)
				lfb.ClientState = util.NewMerklePatriciaTrie(util.NewMemoryNodeDB(), 1, nil)
				k := encryption.Hash(vestingsc.ADDRESS + ":clientvestingpools:")
				if _, err := lfb.ClientState.Insert(util.Path(k), &v); err != nil {
					t.Fatal(err)
				}

				ch := chain.NewChainFromConfig()
				ch.LatestFinalizedBlock = lfb

				return ch
			}(),
			args: args{
				w: httptest.NewRecorder(),
				r: func() *http.Request {
					tar := fmt.Sprintf("%v%v%v", "/v1/screst/", vestingsc.ADDRESS, "/getClientPools")
					req := httptest.NewRequest(http.MethodGet, tar, nil)

					return req
				}(),
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name,
			func(t *testing.T) {
				if test.setValidConfig {
					config.SmartContractConfig.Set("smart_contracts.storagesc.max_challenge_completion_time", 1000)
					config.SmartContractConfig.Set("smart_contracts.vestingsc.min_duration", time.Second*5)
				} else {
					config.SmartContractConfig.Set("smart_contracts.storagesc.max_challenge_completion_time", -1)
					config.SmartContractConfig.Set("smart_contracts.vestingsc.min_duration", 0)
				}

				test.chain.HandleSCRest(test.args.w, test.args.r)
				d, err := ioutil.ReadAll(test.args.w.Result().Body)
				require.NoError(t, err)
				assert.Equal(t, test.wantStatus, test.args.w.Result().StatusCode, string(d))
			},
		)
	}
}

func TestGetSCRestOutput(t *testing.T) {
	chain := chain.NewChainFromConfig()

	getRequest := func(adress string) *http.Request {
		tar := fmt.Sprintf("%v%v", "/v1/screst/", adress)
		req := httptest.NewRequest(http.MethodGet, tar, nil)
		return req
	}

	tests := []struct {
		name    string
		address string
		empty   bool
	}{
		{
			name:    "faucetsc",
			address: faucetsc.ADDRESS,
		},
		{
			name:    "faucetsc",
			address: interestpoolsc.ADDRESS,
		},
		{
			name:    "miner",
			address: minersc.ADDRESS,
		},
		{
			name:    "miner",
			address: minersc.ADDRESS,
		},
		{
			name:    "multisig",
			address: multisigsc.Address,
		},
		{
			name:    "storage",
			address: storagesc.ADDRESS,
		},
		{
			name:    "vesting",
			address: vestingsc.ADDRESS,
		},
		{
			name:    "zcn",
			address: zcnsc.ADDRESS,
		},
		{
			name:    "invalid",
			address: "not_an_address",
			empty:   true,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			chain.HandleSCRest(w, getRequest(test.address))

			body := w.Body.String()
			sc := smartcontract.ContractMap[test.address]
			if test.empty {
				require.EqualValues(t, body, "")
				return
			}
			restPoints := sc.GetRestPoints()
			require.EqualValues(t, len(restPoints), strings.Count(body, "/v1/screst/*/"))
		})
	}
}
