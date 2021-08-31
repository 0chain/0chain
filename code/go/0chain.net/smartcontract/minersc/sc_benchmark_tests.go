package minersc

import (
	"0chain.net/chaincore/block"
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/smartcontract"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
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
	txn   transaction.Transaction
	input []byte
}

func (bt BenchTest) Name() string {
	return bt.name
}

func (bt BenchTest) Transaction() transaction.Transaction {
	return bt.txn
}

func (bt BenchTest) Run(balances cstate.StateContextI) {
	gn, err := getGlobalNode(balances)
	if err != nil {
		panic(err)
	}
	_, err = bt.endpoint(&bt.txn, bt.input, gn, balances)
	if err != nil {
		panic(err)
	}
}

func BenchmarkTests(
	data bk.BenchData, sigScheme bk.SignatureScheme,
) bk.TestSuit {
	var msc = MinerSmartContract{
		SmartContract: sci.NewSC(ADDRESS),
	}
	msc.setSC(msc.SmartContract, &smartcontract.BCContext{})
	var tests = []BenchTest{
		{
			name:     "miner.add_miner",
			endpoint: msc.AddMiner,
			txn:      transaction.Transaction{},
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
			txn:      transaction.Transaction{},
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
			name:     "miner.add_sharder",
			endpoint: msc.minerHealthCheck,
			txn: transaction.Transaction{
				ClientID: data.Miners[0],
			},
			input: nil,
		},
		{
			name:     "miner.sharder_health_check",
			endpoint: msc.sharderHealthCheck,
			txn: transaction.Transaction{
				ClientID: data.Sharders[0],
			},
			input: nil,
		},
		{
			name:     "miner.payFees",
			endpoint: msc.payFees,
			txn: transaction.Transaction{
				ClientID:   data.Miners[0],
				ToClientID: ADDRESS,
			},
			input: nil,
		},
		{
			name:     "miner.contributeMpk",
			endpoint: msc.contributeMpk,
			txn: transaction.Transaction{
				ClientID:   data.Miners[0],
				ToClientID: ADDRESS,
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
		}, /* todo need to set PhaseNode.Phase differently for different tests
		{
			name:     "miner.shareSignsOrShares",
			endpoint: msc.shareSignsOrShares,
			txn: transaction.Transaction{
				ClientID:   data.Miners[0],
			},
			input: func() []byte {
				var sos = make(map[string]*bls.DKGKeyShare)
				for i := 0; i < viper.GetInt(bk.InternalT); i++ {
					sos[data.Miners[i]] = &bls.DKGKeyShare{}
				}
				return (&block.ShareOrSigns{
					ShareOrSigns: sos,
				}).Encode()
			}(),
		},*/
		{
			name:     "miner.update_miner_settings",
			endpoint: msc.UpdateMinerSettings,
			txn: transaction.Transaction{
				ClientID: data.Miners[0],
			},
			input: (&MinerNode{
				SimpleNode: &SimpleNode{
					ID:                data.Miners[0],
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
			txn: transaction.Transaction{
				ClientID: data.Sharders[0],
			},
			input: (&MinerNode{
				SimpleNode: &SimpleNode{
					ID:                data.Sharders[0],
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
			txn:      transaction.Transaction{},
			input: (&deletePool{
				MinerID: data.Miners[0],
				PoolID:  getMockDelegateId(0, viper.GetInt(bk.NumMinerDelegates)),
			}).Encode(),
		},
		{
			name:     "miner.deleteFromDelegatePool",
			endpoint: msc.deleteFromDelegatePool,
			txn: transaction.Transaction{
				ClientID: data.Miners[0],
			},
			input: (&deletePool{
				MinerID: data.Miners[0],
				PoolID:  getMockDelegateId(0, 0),
			}).Encode(),
		},
		{
			name:     "miner.sharder_keep",
			endpoint: msc.sharderKeep,
			txn:      transaction.Transaction{},
			input: (&MinerNode{
				SimpleNode: &SimpleNode{
					ID:        data.Sharders[0],
					PublicKey: "my public key",
				},
			}).Encode(),
		},
	}
	var testsI []bk.BenchTestI
	for _, test := range tests {
		testsI = append(testsI, test)
	}
	return bk.TestSuit{bk.Miner, testsI}
}
