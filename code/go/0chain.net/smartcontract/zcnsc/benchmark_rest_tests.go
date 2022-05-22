package zcnsc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/smartcontract/benchmark"
)

type restBenchTest struct {
	name        string
	params      map[string]string
	shownResult bool
}

func (bt *restBenchTest) Name() string {
	return "zcnsc_rest." + bt.name
}

func (bt *restBenchTest) Transaction() *transaction.Transaction {
	return &transaction.Transaction{}
}

func (bt *restBenchTest) Run(balances cstate.StateContextI, b *testing.B) error {
	b.StopTimer()
	req := httptest.NewRequest("GET", "http://localhost/v1/screst/"+ADDRESS+"/"+bt.name, nil)
	rec := httptest.NewRecorder()
	if len(bt.params) > 0 {
		q := req.URL.Query()
		for k, v := range bt.params {
			q.Add(k, v)
		}
		req.URL.RawQuery = q.Encode()
	}
	b.StartTimer()

	http.DefaultServeMux.ServeHTTP(rec, req)

	b.StopTimer()
	resp := rec.Result()
	if viper.GetBool(benchmark.ShowOutput) && !bt.shownResult {
		body, _ := io.ReadAll(resp.Body)
		var prettyJSON bytes.Buffer
		err := json.Indent(&prettyJSON, body, "", "\t")
		require.NoError(b, err)
		fmt.Println(req.URL.String()+" : ", prettyJSON.String())
		bt.shownResult = true
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status code %v not ok: %v", resp.StatusCode, resp.Status)
	}
	b.StartTimer()

	return nil
}

func BenchmarkRestTests(data benchmark.BenchData, _ benchmark.SignatureScheme) benchmark.TestSuite {
	return benchmark.GetRestTests(
		[]benchmark.TestParameters{
			{
				FuncName: "getAuthorizerNodes",
			},
			{
				FuncName: "getGlobalConfig",
			},
			{
				FuncName: "getAuthorizer",
				Params: map[string]string{
					"id": data.Clients[0],
				},
			},
		},
		ADDRESS,
	)
}
