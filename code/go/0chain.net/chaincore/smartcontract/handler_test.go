package smartcontract_test

import (
	"0chain.net/chaincore/block"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"0chain.net/smartcontract/stakepool/spenum"

	"0chain.net/chaincore/chain"

	"github.com/0chain/common/core/currency"

	"0chain.net/smartcontract/dbs/event"
	"0chain.net/smartcontract/multisigsc"
	"0chain.net/smartcontract/setupsc"
	"0chain.net/smartcontract/vestingsc"
	"0chain.net/smartcontract/zcnsc"
	"github.com/stretchr/testify/require"

	"0chain.net/core/common"
	"0chain.net/core/viper"
	"github.com/rcrowley/go-metrics"
	"github.com/stretchr/testify/mock"

	chstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/chain/state/mocks"
	. "0chain.net/chaincore/smartcontract"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/datastore"
	"0chain.net/smartcontract/faucetsc"
	"0chain.net/smartcontract/minersc"
	"0chain.net/smartcontract/storagesc"
	"github.com/0chain/common/core/util"
)

func init() {
	metrics.DefaultRegistry = metrics.NewRegistry()
	viper.Set("server_chain.smart_contract.faucet", true)
	viper.Set("server_chain.smart_contract.storage", true)
	viper.Set("server_chain.smart_contract.zcn", true)
	viper.Set("server_chain.smart_contract.multisig", true)
	viper.Set("server_chain.smart_contract.miner", true)
	viper.Set("server_chain.smart_contract.vesting", true)
	setupsc.SetupSmartContracts()
}

func TestExecuteStats(t *testing.T) {
	t.Parallel()
	i := `<!DOCTYPE html><html><body><style>
.number { text-align: right; }
.menu li { list-style-type: none; }
table, td, th { border: 1px solid black;  border-collapse: collapse;}
tr.header { background-color: #E0E0E0;  }
.inactive { background-color: #F44336; }
.warning { background-color: #FFEB3B; }
.optimal { color: #1B5E20; }
.slow { font-style: italic; }
.bold {font-weight:bold;}</style><table width='100%'><tr><td><h2>pour</h2><table width='100%'><tr><td class='sheader' colspan=2'>Metrics</td></tr><tr><td>Count</td><td>0</td></tr><tr><td class='sheader' colspan='2'>Time taken</td></tr><tr><td>Min</td><td>0.00 ms</td></tr><tr><td>Mean</td><td>0.00 &plusmn;0.00 ms</td></tr><tr><td>Max</td><td>0.00 ms</td></tr><tr><td>50.00%</td><td>0.00 ms</td></tr><tr><td>90.00%</td><td>0.00 ms</td></tr><tr><td>95.00%</td><td>0.00 ms</td></tr><tr><td>99.00%</td><td>0.00 ms</td></tr><tr><td>99.90%</td><td>0.00 ms</td></tr><tr><td class='sheader' colspan='2'>Rate per second</td></tr><tr><td>Last 1-min rate</td><td>0.00</td></tr><tr><td>Last 5-min rate</td><td>0.00</td></tr><tr><td>Last 15-min rate</td><td>0.00</td></tr><tr><td>Overall mean rate</td><td>0.00</td></tr></table></td><td><h2>refill</h2><table width='100%'><tr><td class='sheader' colspan=2'>Metrics</td></tr><tr><td>Count</td><td>0</td></tr><tr><td class='sheader' colspan='2'>Time taken</td></tr><tr><td>Min</td><td>0.00 ms</td></tr><tr><td>Mean</td><td>0.00 &plusmn;0.00 ms</td></tr><tr><td>Max</td><td>0.00 ms</td></tr><tr><td>50.00%</td><td>0.00 ms</td></tr><tr><td>90.00%</td><td>0.00 ms</td></tr><tr><td>95.00%</td><td>0.00 ms</td></tr><tr><td>99.00%</td><td>0.00 ms</td></tr><tr><td>99.90%</td><td>0.00 ms</td></tr><tr><td class='sheader' colspan='2'>Rate per second</td></tr><tr><td>Last 1-min rate</td><td>0.00</td></tr><tr><td>Last 5-min rate</td><td>0.00</td></tr><tr><td>Last 15-min rate</td><td>0.00</td></tr><tr><td>Overall mean rate</td><td>0.00</td></tr></table></td></tr><tr><td><h2>token refills</h2><table width='100%'><tr><td class='sheader' colspan=2'>Metrics</td></tr><tr><td>Count</td><td>0</td></tr><tr><td class='sheader' colspan='2'>Metric Value</td></tr><tr><td>Min</td><td>0.00</td></tr><tr><td>Mean</td><td>0.00 &plusmn;0.00</td></tr><tr><td>Max</td><td>0.00</td></tr><tr><td>50.00%</td><td>0.00</td></tr><tr><td>90.00%</td><td>0.00</td></tr><tr><td>95.00%</td><td>0.00</td></tr><tr><td>99.00%</td><td>0.00</td></tr><tr><td>99.90%</td><td>0.00</td></tr></table></td><td><h2>tokens Poured</h2><table width='100%'><tr><td class='sheader' colspan=2'>Metrics</td></tr><tr><td>Count</td><td>0</td></tr><tr><td class='sheader' colspan='2'>Metric Value</td></tr><tr><td>Min</td><td>0.00</td></tr><tr><td>Mean</td><td>0.00 &plusmn;0.00</td></tr><tr><td>Max</td><td>0.00</td></tr><tr><td>50.00%</td><td>0.00</td></tr><tr><td>90.00%</td><td>0.00</td></tr><tr><td>95.00%</td><td>0.00</td></tr><tr><td>99.00%</td><td>0.00</td></tr><tr><td>99.90%</td><td>0.00</td></tr></table></td></tr><tr><td><h2>update-settings</h2><table width='100%'><tr><td class='sheader' colspan=2'>Metrics</td></tr><tr><td>Count</td><td>0</td></tr><tr><td class='sheader' colspan='2'>Time taken</td></tr><tr><td>Min</td><td>0.00 ms</td></tr><tr><td>Mean</td><td>0.00 &plusmn;0.00 ms</td></tr><tr><td>Max</td><td>0.00 ms</td></tr><tr><td>50.00%</td><td>0.00 ms</td></tr><tr><td>90.00%</td><td>0.00 ms</td></tr><tr><td>95.00%</td><td>0.00 ms</td></tr><tr><td>99.00%</td><td>0.00 ms</td></tr><tr><td>99.90%</td><td>0.00 ms</td></tr><tr><td class='sheader' colspan='2'>Rate per second</td></tr><tr><td>Last 1-min rate</td><td>0.00</td></tr><tr><td>Last 5-min rate</td><td>0.00</td></tr><tr><td>Last 15-min rate</td><td>0.00</td></tr><tr><td>Overall mean rate</td><td>0.00</td></tr></table></body></html>`
	type args struct {
		ctx      context.Context
		scAdress string
		params   url.Values
		w        http.ResponseWriter
	}
	tests := []struct {
		name  string
		args  args
		wantW http.ResponseWriter
	}{
		{
			name: "OK",
			args: args{
				ctx:      context.TODO(),
				w:        httptest.NewRecorder(),
				scAdress: faucetsc.ADDRESS,
			},
			wantW: func() http.ResponseWriter {
				w := httptest.NewRecorder()
				if _, err := fmt.Fprintf(w, "%v", i); err != nil {
					t.Fatal(err)
				}

				return w
			}(),
		},
		{
			name: "Nil_OK",
			args: args{
				w:        httptest.NewRecorder(),
				scAdress: "",
			},
			wantW: func() http.ResponseWriter {
				w := httptest.NewRecorder()
				if _, err := fmt.Fprintf(w, "invalid_sc: Invalid Smart contract address"); err != nil {
					t.Fatal(err)
				}

				return w
			}(),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ExecuteStats(tt.args.ctx, tt.args.scAdress, tt.args.params, tt.args.w)
			require.Equal(t, tt.wantW, tt.args.w)
		})
	}
}

func TestGetSmartContract(t *testing.T) {
	t.Parallel()

	common.ConfigRateLimits()
	tests := []struct {
		name       string
		address    string
		restpoints int
		null       bool
	}{
		{
			name:       "faucet",
			address:    faucetsc.ADDRESS,
			restpoints: 4,
		},
		{
			name:       "storage",
			address:    storagesc.ADDRESS,
			restpoints: 46,
		},
		{
			name:       "multisig",
			address:    multisigsc.Address,
			restpoints: 0,
		},
		{
			name:       "miner",
			address:    minersc.ADDRESS,
			restpoints: 22,
		},
		{
			name:       "vesting",
			address:    vestingsc.ADDRESS,
			restpoints: 3,
		},
		{
			name:       "zcnsc",
			address:    zcnsc.ADDRESS,
			restpoints: 5,
		},
		{
			name:    "Nil_OK",
			address: "not an address",
			null:    true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := GetSmartContract(tt.address)
			require.True(t, tt.null == (got == nil))
			if got == nil {
				return
			}
			require.EqualValues(t, tt.name, got.GetName())
			require.EqualValues(t, tt.address, got.GetAddress())
			require.EqualValues(t, tt.restpoints, len(chain.GetFunctionNames(tt.address)))
		})
	}
}

func makeTestStateContextIMock() *mocks.StateContextI {
	stateContextI := mocks.StateContextI{}

	stateContextI.On("GetClientBalance", mock.AnythingOfType("string")).Return(
		func(_ datastore.Key) currency.Coin {
			return 5
		},
		func(_ datastore.Key) error {
			return nil
		},
	)
	stateContextI.On("AddTransfer", mock.AnythingOfType("*state.Transfer")).Return(
		func(_ *state.Transfer) error {
			return nil
		},
	)
	stateContextI.On("InsertTrieNode", mock.AnythingOfType("string"), mock.AnythingOfType("*faucetsc.GlobalNode")).Return(
		func(_ datastore.Key, _ util.MPTSerializable) datastore.Key {
			return ""
		},
		func(_ datastore.Key, _ util.MPTSerializable) error {
			return nil
		},
	)
	stateContextI.On("InsertTrieNode", mock.AnythingOfType("string"), mock.AnythingOfType("*minersc.MinerNodes")).Return(
		func(_ datastore.Key, _ util.MPTSerializable) datastore.Key {
			return ""
		},
		func(_ datastore.Key, _ util.MPTSerializable) error {
			return nil
		},
	)
	stateContextI.On("InsertTrieNode", mock.AnythingOfType("string"), mock.AnythingOfType("*minersc.MinerNode")).Return(
		func(_ datastore.Key, _ util.MPTSerializable) datastore.Key {
			return ""
		},
		func(_ datastore.Key, _ util.MPTSerializable) error {
			return nil
		},
	)

	hardForks := []string{"apollo", "ares", "artemis", "athena", "demeter", "electra", "hercules", "hermes"}
	for _, name := range hardForks {
		h := chstate.NewHardFork(name, 0)
		key := datastore.Key(name)
		stateContextI.On("InsertTrieNode", key, h).Return(
			func(_ datastore.Key, _ util.MPTSerializable) datastore.Key {
				return key
			},
			func(_ datastore.Key, _ util.MPTSerializable) error {
				return nil
			},
		)
	}

	return &stateContextI
}

func TestExecuteWithStats(t *testing.T) {
	t.Parallel()

	smcoi := faucetsc.FaucetSmartContract{
		SmartContract: sci.NewSC(faucetsc.ADDRESS),
	}
	smcoi.SmartContract.SmartContractExecutionStats["token refills"] = metrics.NewHistogram(metrics.NilSample{})
	smcoi.SmartContract.SmartContractExecutionStats["refill"] = metrics.NewTimer()

	gn := &faucetsc.GlobalNode{
		FaucetConfig: &faucetsc.FaucetConfig{},
	}
	blob, err := gn.MarshalMsg(nil)
	require.NoError(t, err)

	stateContextIMock := makeTestStateContextIMock()
	stateContextIMock.On("GetTrieNode", mock.AnythingOfType("string"), mock.Anything).Return(nil).Run(
		func(args mock.Arguments) {
			v := args.Get(1).(*faucetsc.GlobalNode)
			_, err := v.UnmarshalMsg(blob)
			require.NoError(t, err)
		})

	type args struct {
		smcoi    sci.SmartContractInterface
		sc       *sci.SmartContract
		t        *transaction.Transaction
		funcName string
		input    []byte
		balances chstate.StateContextI
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "ERR",
			args: args{
				smcoi:    &smcoi,
				sc:       smcoi.SmartContract,
				funcName: "unknown func",
				balances: stateContextIMock,
				t: &transaction.Transaction{
					TransactionType: transaction.TxnTypeSmartContract,
					SmartContractData: &transaction.SmartContractData{
						FunctionName: "unknown func",
						InputData:    json.RawMessage{},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "OK",
			args: args{
				smcoi:    &smcoi,
				sc:       smcoi.SmartContract,
				funcName: "refill",
				balances: stateContextIMock,
				t: &transaction.Transaction{
					TransactionType: transaction.TxnTypeSmartContract,
					SmartContractData: &transaction.SmartContractData{
						FunctionName: "refill",
						InputData:    json.RawMessage{},
					},
				},
			},
			want:    "{\"from\":\"\",\"to\":\"\",\"amount\":0}",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ExecuteWithStats(tt.args.smcoi, tt.args.t, tt.args.balances)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExecuteWithStats() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ExecuteWithStats() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExecuteSmartContract(t *testing.T) {
	t.Parallel()

	stateContextIMock := makeTestStateContextIMock()
	stateContextIMock.On("GetTrieNode",
		mock.AnythingOfType("string"),
		mock.MatchedBy(func(v *minersc.MinerNodes) bool {
			return true
		})).Return(nil)

	mockBlock := &block.Block{}
	mockBlock.Round = 0
	stateContextIMock.On("GetBlock").Return(mockBlock).Maybe()

	stateContextIMock.On("GetTrieNode", mock.AnythingOfType("string"), mock.MatchedBy(func(v *chstate.HardFork) bool {
		return true
	})).Return(nil)

	stateContextIMock.On("GetTrieNode",
		mock.AnythingOfType("string"),
		mock.MatchedBy(func(v *minersc.GlobalNode) bool {
			gn := minersc.NewGlobalNode("", map[string]int{})
			blob, err := gn.MarshalMsg(nil)
			require.NoError(t, err)

			_, err = v.UnmarshalMsg(blob)
			require.NoError(t, err)
			return true
		})).Return(nil)
	stateContextIMock.On("GetTrieNode",
		mock.AnythingOfType("string"),
		mock.MatchedBy(func(v *minersc.SimpleNode) bool {
			sn := &minersc.SimpleNode{}
			blob, err := sn.MarshalMsg(nil)
			require.NoError(t, err)

			_, err = v.UnmarshalMsg(blob)
			require.NoError(t, err)
			return true
		})).Return(nil)
	stateContextIMock.On("GetTrieNode",
		mock.AnythingOfType("string"),
		mock.MatchedBy(func(v *minersc.MinerNode) bool {
			v.ProviderType = spenum.Miner
			blob, err := v.MarshalMsg(nil)
			require.NoError(t, err)

			_, err = v.UnmarshalMsg(blob)
			require.NoError(t, err)
			return true
		})).Return(nil)
	stateContextIMock.On("GetTrieNode",
		mock.AnythingOfType("string"),
		mock.MatchedBy(func(v *faucetsc.GlobalNode) bool {
			gn := &faucetsc.GlobalNode{
				FaucetConfig: &faucetsc.FaucetConfig{},
			}
			blob, err := gn.MarshalMsg(nil)
			require.NoError(t, err)

			_, err = v.UnmarshalMsg(blob)
			require.NoError(t, err)
			return true
		})).Return(nil)
	stateContextIMock.On("EmitEvent",
		mock.Anything,
		mock.MatchedBy(func(v event.EventTag) bool {
			return v == event.TagMinerHealthCheck ||
				v == event.TagSharderHealthCheck ||
				v == event.TagBlobberHealthCheck ||
				v == event.TagValidatorHealthCheck ||
				v == event.TagAuthorizerHealthCheck
		}),
		mock.Anything,
		mock.Anything,
	).Return(nil)

	type args struct {
		ctx      context.Context
		t        *transaction.Transaction
		td       *sci.SmartContractTransactionData
		balances chstate.StateContextI
	}

	smartContractData := sci.SmartContractTransactionData{
		FunctionName: "miner_health_check",
	}

	scData, err := json.Marshal(smartContractData)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "Invalid_Address_ERR",
			args: args{
				t: &transaction.Transaction{
					ToClientID: "unknown",
					SmartContractData: &transaction.SmartContractData{
						FunctionName: "miner_health_check",
						InputData:    json.RawMessage{},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Invalid_JSON_Data_ERR",
			args: args{
				balances: stateContextIMock,
				t: &transaction.Transaction{
					ToClientID: faucetsc.ADDRESS,
					SmartContractData: &transaction.SmartContractData{
						FunctionName: "update-settings",
						InputData:    json.RawMessage("}{"),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "OK",
			args: args{
				balances: stateContextIMock,
				t: &transaction.Transaction{
					ToClientID: minersc.ADDRESS,
					SmartContractData: &transaction.SmartContractData{
						FunctionName: "miner_health_check",
						InputData:    scData,
					},
				},
			},
			want:    "{\"simple_miner\":{\"id\":\"\",\"is_shut_down\":false,\"is_killed\":false,\"provider_type\":1,\"n2n_host\":\"\",\"host\":\"\",\"port\":0,\"path\":\"\",\"public_key\":\"\",\"short_name\":\"\",\"build_tag\":\"\",\"total_stake\":0,\"delete\":false,\"last_health_check\":0,\"last_setting_update_round\":0},\"stake_pool\":{\"pools\":{},\"rewards\":0,\"settings\":{\"delegate_wallet\":\"\",\"num_delegates\":0,\"min_stake\":0,\"service_charge\":0},\"minter\":0,\"is_dead\":false}}",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			//txnData, err := json.Marshal(tt.args.td)
			//require.NoError(t, err)
			//tt.args.t.TransactionData = string(txnData)
			got, err := ExecuteSmartContract(tt.args.t, tt.args.balances)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExecuteSmartContract() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ExecuteSmartContract() got = %v, want %v", got, tt.want)
			}
		})
	}
}
