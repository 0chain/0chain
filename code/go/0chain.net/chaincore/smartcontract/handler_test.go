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

	"github.com/rcrowley/go-metrics"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	chstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/smartcontract"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/datastore"
	"0chain.net/core/logging"
	"0chain.net/core/util"
	"0chain.net/mocks"
	"0chain.net/smartcontract/faucetsc"
	"0chain.net/smartcontract/setupsc"
	"0chain.net/smartcontract/storagesc"
)

func init() {
	viper.Set("development.smart_contract.faucet", true)
	setupsc.SetupSmartContracts()
	logging.InitLogging("testing")
}

func TestExecuteRestAPI(t *testing.T) {
	t.Skip("smart contract aren't protected against parallel access")

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

			got, err := smartcontract.ExecuteRestAPI(tt.args.ctx, tt.args.scAdress, tt.args.restpath, tt.args.params, tt.args.balances)
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
.bold {font-weight:bold;}</style><table width='100%'><tr><td><h2>pour</h2><table width='100%'><tr><td class='sheader' colspan=2'>Metrics</td></tr><tr><td>Count</td><td>0</td></tr><tr><td class='sheader' colspan='2'>Time taken</td></tr><tr><td>Min</td><td>0.00 ms</td></tr><tr><td>Mean</td><td>0.00 &plusmn;0.00 ms</td></tr><tr><td>Max</td><td>0.00 ms</td></tr><tr><td>50.00%</td><td>0.00 ms</td></tr><tr><td>90.00%</td><td>0.00 ms</td></tr><tr><td>95.00%</td><td>0.00 ms</td></tr><tr><td>99.00%</td><td>0.00 ms</td></tr><tr><td>99.90%</td><td>0.00 ms</td></tr><tr><td class='sheader' colspan='2'>Rate per second</td></tr><tr><td>Last 1-min rate</td><td>0.00</td></tr><tr><td>Last 5-min rate</td><td>0.00</td></tr><tr><td>Last 15-min rate</td><td>0.00</td></tr><tr><td>Overall mean rate</td><td>0.00</td></tr></table></td><td><h2>refill</h2><table width='100%'><tr><td class='sheader' colspan=2'>Metrics</td></tr><tr><td>Count</td><td>0</td></tr><tr><td class='sheader' colspan='2'>Time taken</td></tr><tr><td>Min</td><td>0.00 ms</td></tr><tr><td>Mean</td><td>0.00 &plusmn;0.00 ms</td></tr><tr><td>Max</td><td>0.00 ms</td></tr><tr><td>50.00%</td><td>0.00 ms</td></tr><tr><td>90.00%</td><td>0.00 ms</td></tr><tr><td>95.00%</td><td>0.00 ms</td></tr><tr><td>99.00%</td><td>0.00 ms</td></tr><tr><td>99.90%</td><td>0.00 ms</td></tr><tr><td class='sheader' colspan='2'>Rate per second</td></tr><tr><td>Last 1-min rate</td><td>0.00</td></tr><tr><td>Last 5-min rate</td><td>0.00</td></tr><tr><td>Last 15-min rate</td><td>0.00</td></tr><tr><td>Overall mean rate</td><td>0.00</td></tr></table></td></tr><tr><td><h2>token refills</h2><table width='100%'><tr><td class='sheader' colspan=2'>Metrics</td></tr><tr><td>Count</td><td>0</td></tr><tr><td class='sheader' colspan='2'>Metric Value</td></tr><tr><td>Min</td><td>0.00</td></tr><tr><td>Mean</td><td>0.00 &plusmn;0.00</td></tr><tr><td>Max</td><td>0.00</td></tr><tr><td>50.00%</td><td>0.00</td></tr><tr><td>90.00%</td><td>0.00</td></tr><tr><td>95.00%</td><td>0.00</td></tr><tr><td>99.00%</td><td>0.00</td></tr><tr><td>99.90%</td><td>0.00</td></tr></table></td><td><h2>tokens Poured</h2><table width='100%'><tr><td class='sheader' colspan=2'>Metrics</td></tr><tr><td>Count</td><td>0</td></tr><tr><td class='sheader' colspan='2'>Metric Value</td></tr><tr><td>Min</td><td>0.00</td></tr><tr><td>Mean</td><td>0.00 &plusmn;0.00</td></tr><tr><td>Max</td><td>0.00</td></tr><tr><td>50.00%</td><td>0.00</td></tr><tr><td>90.00%</td><td>0.00</td></tr><tr><td>95.00%</td><td>0.00</td></tr><tr><td>99.00%</td><td>0.00</td></tr><tr><td>99.90%</td><td>0.00</td></tr></table></td></tr><tr><td><h2>updateLimits</h2><table width='100%'><tr><td class='sheader' colspan=2'>Metrics</td></tr><tr><td>Count</td><td>0</td></tr><tr><td class='sheader' colspan='2'>Time taken</td></tr><tr><td>Min</td><td>0.00 ms</td></tr><tr><td>Mean</td><td>0.00 &plusmn;0.00 ms</td></tr><tr><td>Max</td><td>0.00 ms</td></tr><tr><td>50.00%</td><td>0.00 ms</td></tr><tr><td>90.00%</td><td>0.00 ms</td></tr><tr><td>95.00%</td><td>0.00 ms</td></tr><tr><td>99.00%</td><td>0.00 ms</td></tr><tr><td>99.90%</td><td>0.00 ms</td></tr><tr><td class='sheader' colspan='2'>Rate per second</td></tr><tr><td>Last 1-min rate</td><td>0.00</td></tr><tr><td>Last 5-min rate</td><td>0.00</td></tr><tr><td>Last 15-min rate</td><td>0.00</td></tr><tr><td>Overall mean rate</td><td>0.00</td></tr></table></body></html>`

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
				scAdress: storagesc.ADDRESS,
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

			smartcontract.ExecuteStats(tt.args.ctx, tt.args.scAdress, tt.args.params, tt.args.w)
			assert.Equal(t, tt.wantW, tt.args.w)
		})
	}
}

func TestGetSmartContract(t *testing.T) {
	t.Parallel()

	type args struct {
		scAddress string
	}
	tests := []struct {
		name string
		args args
		want sci.SmartContractInterface
	}{
		{
			name: "OK",
			args: args{
				scAddress: faucetsc.ADDRESS,
			},
			want: smartcontract.ContractMap[faucetsc.ADDRESS],
		},
		{
			name: "Nil_OK",
			args: args{
				scAddress: storagesc.ADDRESS,
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := smartcontract.GetSmartContract(tt.args.scAddress); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetSmartContract() = %v, want %v", got, tt.want)
			}
		})
	}
}

func makeTestStateContextIMock() *mocks.StateContextI{
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

			got, err := smartcontract.ExecuteWithStats(tt.args.smcoi, tt.args.sc, tt.args.t, tt.args.funcName, tt.args.input, tt.args.balances)
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
	t.Skip("smart contract aren't protected against parallel access")

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
					ToClientID:      faucetsc.ADDRESS,
					TransactionData: "}{",
				},
			},
			wantErr: true,
		},
		{
			name: "Execute_ERR",
			args: args{
				balances: stateContextIMock,
				t: &transaction.Transaction{
					ToClientID:      faucetsc.ADDRESS,
					TransactionData: "{}",
				},
			},
			wantErr: true,
		},
		{
			name: "OK",
			args: args{
				balances: stateContextIMock,
				t: &transaction.Transaction{
					ToClientID: faucetsc.ADDRESS,
					TransactionData: func() string {
						smartContractData := sci.SmartContractTransactionData{
							FunctionName: "refill",
						}

						blob, err := json.Marshal(smartContractData)
						if err != nil {
							t.Fatal(err)
						}

						return string(blob)
					}(),
				},
			},
			want:    "{\"from\":\"\",\"to\":\"6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d3\",\"amount\":0}",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := smartcontract.ExecuteSmartContract(tt.args.ctx, tt.args.t, tt.args.balances)
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
