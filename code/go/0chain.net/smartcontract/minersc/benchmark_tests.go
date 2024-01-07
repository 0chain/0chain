package minersc

import (
	cstate "0chain.net/smartcontract/common"
	"encoding/json"
	"strings"
	"testing"

	sc "0chain.net/core/config"
	"0chain.net/smartcontract/provider"

	"0chain.net/core/common"

	"0chain.net/smartcontract/stakepool"
	"0chain.net/smartcontract/stakepool/spenum"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/smartcontract"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/threshold/bls"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/viper"
	bk "0chain.net/smartcontract/benchmark"
)

const (
	owner = "1746b06bb09f55ee01b33b5e2e055d6cc7a900cb57c0a3a5eaabb8a0e7745802"
)

type BenchTest struct {
	name     string
	endpoint func(
		*transaction.Transaction,
		[]byte,
		*GlobalNode,
		cstate.StateContextI,
	) (string, error)
	txn   *transaction.Transaction
	input []byte
}

func (bt BenchTest) Name() string {
	return bt.name
}

func (bt BenchTest) Transaction() *transaction.Transaction {
	return &transaction.Transaction{
		HashIDField: datastore.HashIDField{
			Hash: bt.txn.Hash,
		},
		ClientID:     bt.txn.ClientID,
		ToClientID:   bt.txn.ToClientID,
		Value:        bt.txn.Value,
		CreationDate: bt.txn.CreationDate,
	}
}

func (bt BenchTest) Run(balances cstate.TimedQueryStateContext, b *testing.B) error {
	b.StopTimer()
	if bt.name == "miner.shareSignsOrShares" {
		var pn = PhaseNode{
			Phase:        Publish,
			StartRound:   1,
			CurrentRound: 2,
			Restarts:     0,
		}
		_, err := balances.InsertTrieNode(pn.GetKey(), &pn)
		if err != nil {
			panic(err)
		}
	}
	b.StartTimer()

	gn, err := getGlobalNode(balances)
	if err != nil {
		panic(err)
	}
	_, err = bt.endpoint(bt.Transaction(), bt.input, gn, balances)

	return err
}

func BenchmarkTests(
	data bk.BenchData, _ bk.SignatureScheme,
) bk.TestSuite {
	creationTimeRaw := viper.GetInt64("MptCreationTime")
	creationTime := common.Now()
	if creationTimeRaw != 0 {
		creationTime = common.Timestamp(creationTimeRaw)
	}

	var msc = MinerSmartContract{
		SmartContract: sci.NewSC(ADDRESS),
	}
	msc.setSC(msc.SmartContract, &smartcontract.BCContext{})
	var tests = []BenchTest{
		{
			name:     "miner.add_miner",
			endpoint: msc.AddMiner,
			txn:      &transaction.Transaction{CreationDate: creationTime},
			input: (&MinerNode{
				SimpleNode: &SimpleNode{
					Provider: provider.Provider{
						ID:           encryption.Hash("magic_block_miner_1"),
						ProviderType: spenum.Miner,
					},
					PublicKey: "miner's public key",
					N2NHost:   "new n2n_host",
					Host:      "new host",
					Port:      1234,
				},
				StakePool: &stakepool.StakePool{
					Pools: make(map[string]*stakepool.DelegatePool),
					Settings: stakepool.Settings{
						ServiceChargeRatio: viper.GetFloat64(bk.MinerMaxCharge),
						MaxNumDelegates:    viper.GetInt(bk.MinerMaxDelegates),
					},
				},
			}).Encode(),
		},
		{
			name:     "miner.add_sharder",
			endpoint: msc.AddSharder,
			txn:      &transaction.Transaction{CreationDate: creationTime},
			input: (&MinerNode{
				SimpleNode: &SimpleNode{
					Provider: provider.Provider{
						ID:           data.InactiveSharder,
						ProviderType: spenum.Sharder,
					},
					PublicKey: data.InactiveSharderPK,
					N2NHost:   "new n2n_host",
					Host:      "new host",
					Port:      1234,
				},
				StakePool: &stakepool.StakePool{
					Pools: make(map[string]*stakepool.DelegatePool),
					Settings: stakepool.Settings{
						ServiceChargeRatio: viper.GetFloat64(bk.MinerMaxCharge),
						MaxNumDelegates:    viper.GetInt(bk.MinerMaxDelegates),
					},
				},
			}).Encode(),
		},
		{
			name:     "miner.update_globals",
			endpoint: msc.minerHealthCheck,
			txn: &transaction.Transaction{
				ClientID:     data.Miners[0],
				CreationDate: creationTime,
			},
			input: nil,
		},
		{
			name:     "miner.miner_health_check",
			endpoint: msc.minerHealthCheck,
			txn: &transaction.Transaction{
				ClientID:     data.Miners[0],
				CreationDate: creationTime,
			},
			input: nil,
		},
		{
			name:     "miner.sharder_health_check",
			endpoint: msc.sharderHealthCheck,
			txn: &transaction.Transaction{
				ClientID:     data.Sharders[0],
				CreationDate: creationTime,
			},
			input: nil,
		},
		{
			name: "miner.payFees",
			endpoint: func(t *transaction.Transaction,
				input []byte, gn *GlobalNode, balances cstate.StateContextI) (
				resp string, err error) {
				p := &PayFeesInput{Round: balances.GetBlock().Round}
				marshal, err := json.Marshal(p)
				if err != nil {
					return "", err
				}
				return msc.payFees(t, marshal, gn, balances)
			},
			txn: &transaction.Transaction{
				ClientID:     data.Miners[0],
				ToClientID:   ADDRESS,
				CreationDate: creationTime,
			},
		},
		{
			name: "storage.kill_miner",
			input: (&provider.ProviderRequest{
				ID: data.Miners[0],
			}).Encode(),
			endpoint: msc.killMiner,
			txn: &transaction.Transaction{
				ClientID:     viper.GetString(bk.MinerMOwner),
				CreationDate: creationTime,
			},
		},
		{
			name: "storage.kill_sharder",
			input: (&provider.ProviderRequest{
				ID: data.Sharders[0],
			}).Encode(),
			endpoint: msc.killSharder,
			txn: &transaction.Transaction{
				ClientID:     viper.GetString(bk.MinerMOwner),
				CreationDate: creationTime,
			},
		},
		{
			name:     "miner.contributeMpk",
			endpoint: msc.contributeMpk,
			txn: &transaction.Transaction{
				ClientID:     data.Miners[0],
				ToClientID:   ADDRESS,
				CreationDate: creationTime,
			},
			input: func() []byte {
				var mpks []string
				for i := 0; i < viper.GetInt(bk.InternalT); i++ {
					mpks = append(mpks, data.Miners[i])
				}
				return (&block.MPK{
					Mpk: mpks,
				}).Encode()
			}(),
		},
		{
			name:     "miner.shareSignsOrShares",
			endpoint: msc.shareSignsOrShares,
			txn: &transaction.Transaction{
				ClientID:     data.Miners[0],
				CreationDate: creationTime,
			},
			input: func() []byte {
				var sos = make(map[string]*bls.DKGKeyShare)
				for i := 0; i < viper.GetInt(bk.InternalT); i++ {
					sos[data.Miners[i]] = nil
				}
				return (&block.ShareOrSigns{
					ShareOrSigns: sos,
				}).Encode()
			}(),
		},
		{
			name:     "miner.update_globals",
			endpoint: msc.updateGlobals,
			txn: &transaction.Transaction{
				ClientID:     owner,
				CreationDate: creationTime,
			},
			input: (&sc.StringMap{
				Fields: map[string]string{
					"server_chain.block.min_block_size":                  "1",
					"server_chain.block.max_block_size":                  "10",
					"server_chain.block.max_byte_size":                   "1638400",
					"server_chain.block.replicators":                     "0",
					"server_chain.block.proposal.max_wait_time":          "100ms",
					"server_chain.block.proposal.wait_mode":              "static",
					"server_chain.block.consensus.threshold_by_count":    "66",
					"server_chain.block.consensus.threshold_by_stake":    "0",
					"server_chain.block.sharding.min_active_sharders":    "25",
					"server_chain.block.sharding.min_active_replicators": "25",
					"server_chain.block.validation.batch_size":           "1000",
					"server_chain.block.reuse_txns":                      "false",
					"server_chain.round_range":                           "10000000",
					"server_chain.round_timeouts.softto_min":             "3000",
					"server_chain.round_timeouts.softto_mult":            "3",
					"server_chain.round_timeouts.round_restart_mult":     "2",
					"server_chain.transaction.payload.max_size":          "98304",
					"server_chain.client.signature_scheme":               "bls0chain",
					"server_chain.messages.verification_tickets_to":      "all_miners",
					"server_chain.state.prune_below_count":               "100",
				},
			}).Encode(),
		},
		{
			name:     "miner.update_settings",
			endpoint: msc.updateSettings,
			txn: &transaction.Transaction{
				ClientID:     owner,
				CreationDate: creationTime,
			},
			input: (&sc.StringMap{
				Fields: map[string]string{
					"max_n":                               "7",
					"min_n":                               "3",
					"t_percent":                           "0.66",
					"k_percent":                           "0.75",
					"x_percent":                           "0.70",
					"max_s":                               "2",
					"min_s":                               "1",
					"max_delegates":                       "200",
					"reward_round_frequency":              "64250",
					"reward_rate":                         "1.0",
					"share_ratio":                         "50",
					"block_reward":                        "021",
					"max_charge":                          "0.5",
					"epoch":                               "6415000000",
					"reward_decline_rate":                 "0.1",
					"owner_id":                            "f769ccdf8587b8cab6a0f6a8a5a0a91d3405392768f283c80a45d6023a1bfa1f",
					"cost.add_miner":                      "111",
					"cost.add_sharder":                    "111",
					"cost.delete_miner":                   "111",
					"cost.miner_health_check":             "111",
					"cost.sharder_health_check":           "111",
					strings.ToLower("cost.contributeMpk"): "111",
					strings.ToLower("cost.shareSignsOrShares"): "111",
					"cost.wait":                                    "111",
					"cost.update_globals":                          "111",
					"cost.update_settings":                         "111",
					"cost.update_miner_settings":                   "111",
					"cost.update_sharder_settings":                 "111",
					strings.ToLower("cost.payFees"):                "111",
					strings.ToLower("cost.feesPaid"):               "111",
					strings.ToLower("cost.mintedTokens"):           "111",
					strings.ToLower("cost.addToDelegatePool"):      "111",
					strings.ToLower("cost.deleteFromDelegatePool"): "111",
					"cost.sharder_keep":                            "111",
					"cost.kill_miner":                              "111",
					"cost.kill_sharder":                            "111",
				},
			}).Encode(),
		},
		{
			name:     "miner.update_miner_settings",
			endpoint: msc.UpdateMinerSettings,
			txn: &transaction.Transaction{
				ClientID:     data.Clients[0],
				CreationDate: creationTime,
			},
			input: (&MinerNode{
				SimpleNode: &SimpleNode{
					Provider: provider.Provider{
						ID:           data.Miners[0],
						ProviderType: spenum.Miner,
					},
				},
				StakePool: &stakepool.StakePool{
					Pools: make(map[string]*stakepool.DelegatePool),
					Settings: stakepool.Settings{
						ServiceChargeRatio: viper.GetFloat64(bk.MinerMaxCharge),
						MaxNumDelegates:    viper.GetInt(bk.MinerMaxDelegates),
					},
				},
			}).Encode(),
		},
		{
			name:     "miner.update_sharder_settings",
			endpoint: msc.UpdateSharderSettings,
			txn: &transaction.Transaction{
				ClientID:     data.Clients[0],
				CreationDate: creationTime,
			},
			input: (&MinerNode{
				SimpleNode: &SimpleNode{
					Provider: provider.Provider{
						ID:           data.Sharders[0],
						ProviderType: spenum.Sharder,
					},
				},
				StakePool: &stakepool.StakePool{
					Pools: make(map[string]*stakepool.DelegatePool),
					Settings: stakepool.Settings{
						ServiceChargeRatio: viper.GetFloat64(bk.MinerMaxCharge),
						MaxNumDelegates:    viper.GetInt(bk.MinerMaxDelegates),
					},
				},
			}).Encode(),
		},
		{
			name:     "miner.addToDelegatePool",
			endpoint: msc.addToDelegatePool,
			txn: &transaction.Transaction{
				HashIDField: datastore.HashIDField{
					Hash: encryption.Hash("transaction hash"),
				},
				ClientID:     data.Clients[0],
				ToClientID:   ADDRESS,
				Value:        1e10,
				CreationDate: creationTime,
			},
			input: (&deletePool{
				ProviderType: spenum.Miner,
				ProviderID:   data.Miners[0],
			}).Encode(),
		},
		{
			name:     "miner.deleteFromDelegatePool",
			endpoint: msc.deleteFromDelegatePool,
			txn: &transaction.Transaction{
				ClientID:     getMinerDelegatePoolId(0, 0, data.Clients),
				ToClientID:   ADDRESS,
				CreationDate: creationTime,
			},
			input: (&deletePool{
				ProviderType: spenum.Miner,
				ProviderID:   data.Miners[0],
			}).Encode(),
		},
		{
			name:     "miner.sharder_keep",
			endpoint: msc.sharderKeep,
			txn:      &transaction.Transaction{CreationDate: creationTime},
			input: (&MinerNode{
				SimpleNode: &SimpleNode{
					Provider: provider.Provider{
						ID:           data.Sharders[0],
						ProviderType: spenum.Sharder,
					},
					PublicKey: "my public key",
				},
			}).Encode(),
		},
		{
			name:     "miner.delete_miner",
			endpoint: msc.DeleteMiner,
			txn: &transaction.Transaction{
				ToClientID: ADDRESS,
			},
			input: (&MinerNode{
				SimpleNode: &SimpleNode{
					Provider: provider.Provider{
						ID:           data.Miners[1],
						ProviderType: spenum.Miner,
					},
					PublicKey: "my public key",
				},
			}).Encode(),
		},
		{
			name:     "miner.delete_sharder",
			endpoint: msc.DeleteSharder,
			txn: &transaction.Transaction{
				ToClientID: ADDRESS,
			},
			input: (&MinerNode{
				SimpleNode: &SimpleNode{
					Provider: provider.Provider{
						ID:           data.Sharders[0],
						ProviderType: spenum.Sharder,
					},
					PublicKey: "my public key",
				},
			}).Encode(),
		},
		{
			name:     "miner.collect_reward",
			endpoint: msc.collectReward,
			txn: &transaction.Transaction{
				ClientID:   data.Clients[0],
				ToClientID: ADDRESS,
			},
			input: func() []byte {
				bytes, _ := json.Marshal(&stakepool.CollectRewardRequest{
					ProviderType: spenum.Miner,
					ProviderId:   data.Miners[0],
				})
				return bytes
			}(),
		},
	}
	var testsI []bk.BenchTestI
	for _, test := range tests {
		testsI = append(testsI, test)
	}
	return bk.TestSuite{
		Source:     bk.Miner,
		Benchmarks: testsI,
	}
}
