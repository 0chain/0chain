package minersc

import (
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
	return bt.txn
}

func (bt BenchTest) Run(balances cstate.StateContextI) {
	gn, err := getGlobalNode(balances)
	if err != nil {
		panic(err)
	}
	_, err = bt.endpoint(bt.txn, bt.input, gn, balances)
	if err != nil {
		panic(err)
	}
}

func BenchmarkTests(
	data bk.BenchData, _ bk.SignatureScheme,
) bk.TestSuit {
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
			name: "miner.shareSignsOrShares",
			endpoint: func(
				txn *transaction.Transaction,
				input []byte,
				gn *GlobalNode,
				balances cstate.StateContextI,
			) (string, error) {
				// This is not best practise as adding the node will count as part
				// of the test duration.
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
				return msc.shareSignsOrShares(txn, input, gn, balances)
			},
			txn: &transaction.Transaction{
				ClientID: GetMockNodeId(0, NodeTypeMiner),
			},
			input: func() []byte {
				var sos = make(map[string]*bls.DKGKeyShare)
				for i := 0; i < viper.GetInt(bk.InternalT); i++ {
					//sos[GetMockNodeId(i, NodeTypeMiner)] = &bls.DKGKeyShare{}
					sos[GetMockNodeId(i, NodeTypeMiner)] = nil
				}
				return (&block.ShareOrSigns{
					ShareOrSigns: sos,
				}).Encode()
			}(),
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
	return bk.TestSuit{
		Source:     bk.Miner,
		Benchmarks: testsI,
	}
}
