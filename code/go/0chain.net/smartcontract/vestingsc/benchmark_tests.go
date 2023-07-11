package vestingsc

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/0chain/common/core/currency"

	"github.com/spf13/viper"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/smartcontract"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	sc "0chain.net/smartcontract"
	bk "0chain.net/smartcontract/benchmark"
)

type BenchTest struct {
	name     string
	endpoint func(
		*transaction.Transaction,
		[]byte,
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
		ClientID:     bt.txn.ClientID,
		ToClientID:   bt.txn.ToClientID,
		Value:        bt.txn.Value,
		CreationDate: bt.txn.CreationDate,
	}
}

func (bt BenchTest) Run(balances cstate.TimedQueryStateContext, _ *testing.B) error {
	_, err := bt.endpoint(bt.Transaction(), bt.input, balances)
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

	var vsc = VestingSmartContract{
		SmartContract: sci.NewSC(ADDRESS),
	}
	vestingMinLock, err := currency.ParseZCN(viper.GetFloat64(bk.VestingMinLock))
	if err != nil {
		panic(err)
	}
	vsc.setSC(vsc.SmartContract, &smartcontract.BCContext{})
	var tests = []BenchTest{
		{
			name:     "vesting.trigger",
			endpoint: vsc.trigger,
			txn: &transaction.Transaction{
				ClientID:     data.Clients[0],
				CreationDate: creationTime,
			},
			input: func() []byte {
				bytes, _ := json.Marshal(&poolRequest{
					PoolID: geMockVestingPoolId(0),
				})
				return bytes
			}(),
		},
		{
			name:     "vesting.updateConfig",
			endpoint: vsc.updateConfig,
			txn: &transaction.Transaction{
				ClientID:     viper.GetString(bk.VestingPoolOwner),
				CreationDate: creationTime,
			},
			input: (&sc.StringMap{
				Fields: map[string]string{
					Settings[MinLock]:              "1",
					Settings[MinDuration]:          "2s",
					Settings[MaxDuration]:          "3m",
					Settings[MaxDestinations]:      "5",
					Settings[MaxDescriptionLength]: "7",
				},
			}).Encode(),
		},
		{
			name:     "vesting.unlock",
			endpoint: vsc.unlock,
			txn: &transaction.Transaction{
				ClientID:     data.Clients[0],
				CreationDate: creationTime,
			},
			input: func() []byte {
				bytes, _ := json.Marshal(&poolRequest{
					PoolID: geMockVestingPoolId(0),
				})
				return bytes
			}(),
		},
		{
			name:     "vesting.add",
			endpoint: vsc.add,
			txn: &transaction.Transaction{
				ClientID:     data.Clients[0],
				Value:        vestingMinLock,
				CreationDate: creationTime,
			},
			input: func() []byte {
				var dests destinations
				for i := 0; i < viper.GetInt(bk.VestingMaxDestinations); i++ {
					dests = append(dests, &destination{})
				}
				bytes, _ := json.Marshal(&addRequest{
					Description:  "my description",
					StartTime:    creationTime,
					Duration:     time.Hour,
					Destinations: dests,
				})
				return bytes
			}(),
		},
		{
			name:     "vesting.stop",
			endpoint: vsc.stop,
			txn: &transaction.Transaction{
				ClientID:     data.Clients[0],
				CreationDate: creationTime,
			},
			input: func() []byte {
				bytes, _ := json.Marshal(&stopRequest{
					PoolID:      geMockVestingPoolId(0),
					Destination: getMockDestinationId(0, 0),
				})
				return bytes
			}(),
		},
		{
			name:     "vesting.delete",
			endpoint: vsc.delete,
			txn: &transaction.Transaction{
				ClientID:     data.Clients[0],
				CreationDate: creationTime,
			},
			input: func() []byte {
				bytes, _ := json.Marshal(&poolRequest{
					PoolID: geMockVestingPoolId(0),
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
		Source:     bk.Vesting,
		Benchmarks: testsI,
	}
}
