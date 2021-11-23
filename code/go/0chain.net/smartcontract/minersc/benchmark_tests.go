package minersc

import (
	"testing"

	sc "0chain.net/smartcontract"

	"0chain.net/chaincore/block"
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/smartcontract"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/threshold/bls"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/viper"
	bk "0chain.net/smartcontract/benchmark"
)

var (
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

func (bt BenchTest) Run(balances cstate.StateContextI, b *testing.B) error {
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
	var msc = MinerSmartContract{
		SmartContract: sci.NewSC(ADDRESS),
	}
	msc.setSC(msc.SmartContract, &smartcontract.BCContext{})
	var tests = []BenchTest{
		{
			name:     "miner.add_miner",
			endpoint: msc.AddMiner,
			txn:      &transaction.Transaction{},
			input: (&MinerNode{
				SimpleNode: &SimpleNode{
					ID:                encryption.Hash("my new miner"),
					PublicKey:         "miner's public key",
					ServiceCharge:     viper.GetFloat64(bk.MinerMaxCharge),
					NumberOfDelegates: viper.GetInt(bk.MinerMaxDelegates),
					MinStake:          state.Balance(viper.GetFloat64(bk.MinerMinStake) * 1e10),
					MaxStake:          state.Balance(viper.GetFloat64(bk.MinerMaxStake) * 1e10),
					N2NHost:           "new n2n_host",
					Host:              "new host",
					Port:              1234,
				},
			}).Encode(),
		},
		{
			name:     "miner.add_sharder",
			endpoint: msc.AddSharder,
			txn:      &transaction.Transaction{},
			input: (&MinerNode{
				SimpleNode: &SimpleNode{
					ID:                encryption.Hash("my new sharder"),
					PublicKey:         "sharder's public key",
					ServiceCharge:     viper.GetFloat64(bk.MinerMaxCharge),
					NumberOfDelegates: viper.GetInt(bk.MinerMaxDelegates),
					MinStake:          state.Balance(viper.GetFloat64(bk.MinerMinStake) * 1e10),
					MaxStake:          state.Balance(viper.GetFloat64(bk.MinerMaxStake) * 1e10),
					N2NHost:           "new n2n_host",
					Host:              "new host",
					Port:              1234,
				},
			}).Encode(),
		},
		{
			name:     "miner.update_globals",
			endpoint: msc.minerHealthCheck,
			txn: &transaction.Transaction{
				ClientID: GetMockNodeId(0, NodeTypeMiner),
			},
			input: nil,
		},
		{
			name:     "miner.miner_heath_check",
			endpoint: msc.minerHealthCheck,
			txn: &transaction.Transaction{
				ClientID: GetMockNodeId(0, NodeTypeMiner),
			},
			input: nil,
		},
		{
			name:     "miner.sharder_health_check",
			endpoint: msc.sharderHealthCheck,
			txn: &transaction.Transaction{
				ClientID: GetMockNodeId(0, NodeTypeSharder),
			},
			input: nil,
		},
		{
			name:     "miner.payFees",
			endpoint: msc.payFees,
			txn: &transaction.Transaction{
				ClientID:   GetMockNodeId(0, NodeTypeMiner),
				ToClientID: ADDRESS,
			},
			input: nil,
		},
		{
			name:     "miner.contributeMpk",
			endpoint: msc.contributeMpk,
			txn: &transaction.Transaction{
				ClientID:   GetMockNodeId(0, NodeTypeMiner),
				ToClientID: ADDRESS,
			},
			input: func() []byte {
				var mpks []string
				for i := 0; i < viper.GetInt(bk.InternalT); i++ {
					mpks = append(mpks, GetMockNodeId(i, NodeTypeMiner))
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
				ClientID: GetMockNodeId(0, NodeTypeMiner),
			},
			input: func() []byte {
				var sos = make(map[string]*bls.DKGKeyShare)
				for i := 0; i < viper.GetInt(bk.InternalT); i++ {
					sos[GetMockNodeId(i, NodeTypeMiner)] = nil
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
				ClientID: owner,
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
				ClientID: owner,
			},
			input: (&sc.StringMap{
				Fields: map[string]string{
					"min_stake":              "0.0",
					"max_stake":              "100",
					"max_n":                  "7",
					"min_n":                  "3",
					"t_percent":              "0.66",
					"k_percent":              "0.75",
					"x_percent":              "0.70",
					"max_s":                  "2",
					"min_s":                  "1",
					"max_delegates":          "200",
					"reward_round_frequency": "64250",
					"interest_rate":          "0.0",
					"reward_rate":            "1.0",
					"share_ratio":            "50",
					"block_reward":           "021",
					"max_charge":             "0.5",
					"epoch":                  "6415000000",
					"reward_decline_rate":    "0.1",
					"interest_decline_rate":  "0.1",
					"max_mint":               "1500000.0",
				},
			}).Encode(),
		},
		{
			name:     "miner.update_miner_settings",
			endpoint: msc.UpdateMinerSettings,
			txn: &transaction.Transaction{
				ClientID: GetMockNodeId(0, NodeTypeMiner),
			},
			input: (&MinerNode{
				SimpleNode: &SimpleNode{
					ID:                GetMockNodeId(0, NodeTypeMiner),
					ServiceCharge:     viper.GetFloat64(bk.MinerMaxCharge),
					NumberOfDelegates: viper.GetInt(bk.MinerMaxDelegates),
					MinStake:          state.Balance(viper.GetFloat64(bk.MinerMinStake) * 1e10),
					MaxStake:          state.Balance(viper.GetFloat64(bk.MinerMaxStake) * 1e10),
				},
			}).Encode(),
		},
		{
			name:     "miner.update_sharder_settings",
			endpoint: msc.UpdateSharderSettings,
			txn: &transaction.Transaction{
				ClientID: GetMockNodeId(0, NodeTypeSharder),
			},
			input: (&MinerNode{
				SimpleNode: &SimpleNode{
					ID:                GetMockNodeId(0, NodeTypeSharder),
					ServiceCharge:     viper.GetFloat64(bk.MinerMaxCharge),
					NumberOfDelegates: viper.GetInt(bk.MinerMaxDelegates),
					MinStake:          state.Balance(viper.GetFloat64(bk.MinerMinStake) * 1e10),
					MaxStake:          state.Balance(viper.GetFloat64(bk.MinerMaxStake) * 1e10),
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
				ClientID: data.Clients[0],
				Value:    1e10,
			},
			input: (&deletePool{
				MinerID: GetMockNodeId(0, NodeTypeMiner),
				PoolID:  getMinerDelegatePoolId(0, 0, NodeTypeMiner),
			}).Encode(),
		},
		{
			name:     "miner.deleteFromDelegatePool",
			endpoint: msc.deleteFromDelegatePool,
			txn: &transaction.Transaction{
				ClientID: data.Clients[0],
			},
			input: (&deletePool{
				MinerID: GetMockNodeId(0, NodeTypeMiner),
				PoolID:  getMinerDelegatePoolId(0, 0, NodeTypeMiner),
			}).Encode(),
		},
		{
			name:     "miner.sharder_keep",
			endpoint: msc.sharderKeep,
			txn:      &transaction.Transaction{},
			input: (&MinerNode{
				SimpleNode: &SimpleNode{
					ID:        GetMockNodeId(0, NodeTypeSharder),
					PublicKey: "my public key",
				},
			}).Encode(),
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
