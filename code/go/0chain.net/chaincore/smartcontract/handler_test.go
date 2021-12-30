package smartcontract_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"

	"0chain.net/smartcontract/interestpoolsc"
	"0chain.net/smartcontract/multisigsc"
	"0chain.net/smartcontract/vestingsc"
	"0chain.net/smartcontract/zcnsc"
	"github.com/stretchr/testify/require"

	"0chain.net/core/viper"
	"github.com/rcrowley/go-metrics"
	"github.com/stretchr/testify/mock"

	chstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/mocks"
	. "0chain.net/chaincore/smartcontract"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/datastore"
	"0chain.net/core/logging"
	"0chain.net/core/util"
	"0chain.net/smartcontract/faucetsc"
	"0chain.net/smartcontract/minersc"
	"0chain.net/smartcontract/setupsc"
	"0chain.net/smartcontract/storagesc"
)

func init() {
	metrics.DefaultRegistry = metrics.NewRegistry()
	viper.Set("development.smart_contract.faucet", true)
	viper.Set("development.smart_contract.storage", true)
	viper.Set("development.smart_contract.zcn", true)
	viper.Set("development.smart_contract.interest", true)
	viper.Set("development.smart_contract.multisig", true)
	viper.Set("development.smart_contract.miner", true)
	viper.Set("development.smart_contract.vesting", true)

	config.SmartContractConfig = viper.New()
	config.SmartContractConfig.Set("smart_contracts.faucetsc.ownerId", "1746b06bb09f55ee01b33b5e2e055d6cc7a900cb57c0a3a5eaabb8a0e7745802")
	config.SmartContractConfig.Set("smart_contracts.minersc.ownerId", "1746b06bb09f55ee01b33b5e2e055d6cc7a900cb57c0a3a5eaabb8a0e7745802")
	config.SmartContractConfig.Set("smart_contracts.interestpoolsc.ownerId", "1746b06bb09f55ee01b33b5e2e055d6cc7a900cb57c0a3a5eaabb8a0e7745802")
	config.SmartContractConfig.Set("smart_contracts.vestingsc.ownerId", "1746b06bb09f55ee01b33b5e2e055d6cc7a900cb57c0a3a5eaabb8a0e7745802")
	config.SmartContractConfig.Set("smart_contracts.storagesc.ownerId", "1746b06bb09f55ee01b33b5e2e055d6cc7a900cb57c0a3a5eaabb8a0e7745802")

	setupsc.SetupSmartContracts()
	logging.InitLogging("testing")
}

func TestExecuteRestAPI(t *testing.T) {
	t.Parallel()

	gn := &faucetsc.GlobalNode{}
	blob := gn.Encode()

	sc := mocks.StateContextI{}
	sc.On("GetTrieNode", mock.AnythingOfType("string")).Return(
		func(_ datastore.Key) util.Serializable {
			return &util.SecureSerializableValue{Buffer: blob}
		},
		func(_ datastore.Key) error {
			return nil
		},
	)

	type args struct {
		ctx      context.Context
		scAdress string
		restpath string
		params   url.Values
		balances chstate.StateContextI
	}
	tests := []struct {
		name    string
		args    args
		want    interface{}
		wantErr bool
	}{
		{
			name: "Unregistered_SC_ERR",
			args: args{
				scAdress: storagesc.ADDRESS,
			},
			wantErr: true,
		},
		{
			name: "Unknown_REST_Path_ERR",
			args: args{
				restpath: "unknown path",
				scAdress: faucetsc.ADDRESS,
			},
			wantErr: true,
		},
		{
			name: "OK",
			args: args{
				restpath: "/pourAmount",
				scAdress: faucetsc.ADDRESS,
				balances: &sc,
			},
			want:    "Pour amount per request: 0",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ExecuteRestAPI(tt.args.ctx, tt.args.scAdress, tt.args.restpath, tt.args.params, tt.args.balances)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExecuteRestAPI() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ExecuteRestAPI() got = %v, want %v", got, tt.want)
			}
		})
	}
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
			restpoints: 17,
		},
		{
			name:       "interest",
			address:    interestpoolsc.ADDRESS,
			restpoints: 3,
		},
		{
			name:       "multisig",
			address:    multisigsc.Address,
			restpoints: 0,
		},
		{
			name:       "miner",
			address:    minersc.ADDRESS,
			restpoints: 15,
		},
		{
			name:       "vesting",
			address:    vestingsc.ADDRESS,
			restpoints: 3,
		},
		{
			name:       "zcn",
			address:    zcnsc.ADDRESS,
			restpoints: 1,
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
			require.EqualValues(t, tt.restpoints, len(got.GetRestPoints()))
		})
	}
}

func makeTestStateContextIMock() *mocks.StateContextI {
	stateContextI := mocks.StateContextI{}
	stateContextI.On("GetClientBalance", mock.AnythingOfType("string")).Return(
		func(_ datastore.Key) state.Balance {
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
		func(_ datastore.Key, _ util.Serializable) datastore.Key {
			return ""
		},
		func(_ datastore.Key, _ util.Serializable) error {
			return nil
		},
	)
	stateContextI.On("InsertTrieNode", mock.AnythingOfType("string"), mock.AnythingOfType("*minersc.MinerNodes")).Return(
		func(_ datastore.Key, _ util.Serializable) datastore.Key {
			return ""
		},
		func(_ datastore.Key, _ util.Serializable) error {
			return nil
		},
	)
	stateContextI.On("InsertTrieNode", mock.AnythingOfType("string"), mock.AnythingOfType("*minersc.MinerNode")).Return(
		func(_ datastore.Key, _ util.Serializable) datastore.Key {
			return ""
		},
		func(_ datastore.Key, _ util.Serializable) error {
			return nil
		},
	)

	return &stateContextI
}

func TestExecuteWithStats(t *testing.T) {
	t.Parallel()

	smcoi := faucetsc.FaucetSmartContract{
		SmartContract: sci.NewSC(faucetsc.ADDRESS),
	}
	smcoi.SmartContract.SmartContractExecutionStats["token refills"] = metrics.NewHistogram(metrics.NilSample{})
	smcoi.SmartContract.SmartContractExecutionStats["refill"] = metrics.NewTimer()

	gn := &faucetsc.GlobalNode{}
	blob := gn.Encode()

	stateContextIMock := makeTestStateContextIMock()
	stateContextIMock.On("GetTrieNode", mock.AnythingOfType("string")).Return(
		func(_ datastore.Key) util.Serializable {
			return &util.SecureSerializableValue{Buffer: blob}
		},
		func(_ datastore.Key) error {
			return nil
		},
	)

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
				t:        &transaction.Transaction{},
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
				t:        &transaction.Transaction{},
			},
			want:    "{\"from\":\"\",\"to\":\"\",\"amount\":0}",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ExecuteWithStats(tt.args.smcoi, tt.args.t, tt.args.funcName, tt.args.input, tt.args.balances)
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

	gn := &minersc.GlobalNode{}
	blob := gn.Encode()

	stateContextIMock := makeTestStateContextIMock()
	stateContextIMock.On("GetTrieNode", mock.AnythingOfType("string")).Return(
		func(_ datastore.Key) util.Serializable {
			return &util.SecureSerializableValue{Buffer: blob}
		},
		func(_ datastore.Key) error {
			return nil
		},
	)

	type args struct {
		ctx      context.Context
		t        *transaction.Transaction
		balances chstate.StateContextI
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
				},
			},
			wantErr: true,
		},
		{
			name: "Invalid_JSON_Data_ERR",
			args: args{
				t: &transaction.Transaction{
					ToClientID:      minersc.ADDRESS,
					TransactionData: "}{",
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
					TransactionData: func() string {
						smartContractData := sci.SmartContractTransactionData{
							FunctionName: "miner_health_check",
						}

						blob, err := json.Marshal(smartContractData)
						if err != nil {
							t.Fatal(err)
						}

						return string(blob)
					}(),
				},
			},
			want:    "{\"simple_miner\":{\"id\":\"\",\"n2n_host\":\"\",\"host\":\"\",\"port\":0,\"path\":\"\",\"public_key\":\"\",\"short_name\":\"\",\"build_tag\":\"\",\"total_stake\":0,\"delete\":false,\"delegate_wallet\":\"\",\"service_charge\":0,\"number_of_delegates\":0,\"min_stake\":0,\"max_stake\":0,\"stat\":{},\"last_health_check\":0}}",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ExecuteSmartContract(tt.args.ctx, tt.args.t, tt.args.balances)
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
