package benchmark

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/rest/restinterface"
	"github.com/spf13/viper"

	"bytes"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

type TestSuiteFunc func(data BenchData, sigScheme SignatureScheme) TestSuite

type TestParameters struct {
	FuncName string                                       `json:"func_name,omitempty"`
	Params   map[string]string                            `json:"params,omitempty"`
	Endpoint func(w http.ResponseWriter, r *http.Request) `json:"endpoint,omitempty"`
	Receiver restinterface.RestHandlerI                   `json:"receiver"`
}

type QueryBenchTest struct {
	TestParameters
	shownResult bool
	address     string
	source      Source
}

func NewQueryBenchTest(
	test TestParameters,
	address string,
	source Source,
) BenchTestI {
	return &QueryBenchTest{
		TestParameters: test,
		address:        address,
		source:         source,
	}
}

func (qbt *QueryBenchTest) Name() string {
	return SourceNames[qbt.source] + "." + qbt.FuncName
}

func (qbt *QueryBenchTest) Transaction() *transaction.Transaction {
	return &transaction.Transaction{}
}

func (qbt *QueryBenchTest) Run(balances cstate.StateContextI, b *testing.B) error {
	b.StopTimer()
	req := httptest.NewRequest("GET", "http://localhost/v1/screst/"+qbt.address+"/"+qbt.FuncName, nil)
	rec := httptest.NewRecorder()
	if len(qbt.Params) > 0 {
		q := req.URL.Query()
		for k, v := range qbt.Params {
			q.Add(k, v)
		}
		req.URL.RawQuery = q.Encode()
	}
	b.StartTimer()

	qbt.Receiver.SetQueryStateContext(balances)
	qbt.Endpoint(rec, req)

	b.StopTimer()
	resp := rec.Result()
	if viper.GetBool(ShowOutput) && !qbt.shownResult {
		body, _ := io.ReadAll(resp.Body)
		defer resp.Body.Close()
		var prettyJSON bytes.Buffer
		err := json.Indent(&prettyJSON, body, "", "\t")
		require.NoError(b, err)
		fmt.Println(req.URL.String()+" : ", prettyJSON.String())
		qbt.shownResult = true
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status code %v not ok: %v", resp.StatusCode, resp.Status)
	}
	b.StartTimer()

	return nil
}

func GetRestTests(
	tests []TestParameters,
	address string,
	reciever restinterface.RestHandlerI,
	source Source,
) TestSuite {
	var testsI []BenchTestI
	for _, test := range tests {
		test.Receiver = reciever
		newTest := NewQueryBenchTest(test, address, source)
		testsI = append(testsI, newTest)
	}
	return TestSuite{
		Source:     source,
		Benchmarks: testsI,
	}
}
