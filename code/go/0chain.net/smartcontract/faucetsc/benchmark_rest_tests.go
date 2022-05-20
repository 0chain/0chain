package faucetsc

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
	bk "0chain.net/smartcontract/benchmark"
)

type RestBenchTest struct {
	name        string
	params      map[string]string
	shownResult bool
}

func (rbt *RestBenchTest) Name() string {
	return "faucet_rest." + rbt.name
}

func (rbt *RestBenchTest) Transaction() *transaction.Transaction {
	return &transaction.Transaction{}
}

func (rbt *RestBenchTest) Run(balances cstate.StateContextI, b *testing.B) error {
	b.StopTimer()
	req := httptest.NewRequest("GET", "http://localhost/v1/screst/"+ADDRESS+"/"+rbt.name, nil)
	rec := httptest.NewRecorder()
	if len(rbt.params) > 0 {
		q := req.URL.Query()
		for k, v := range rbt.params {
			q.Add(k, v)
		}
		req.URL.RawQuery = q.Encode()
	}
	b.StartTimer()

	http.DefaultServeMux.ServeHTTP(rec, req)

	b.StopTimer()
	resp := rec.Result()
	if viper.GetBool(bk.ShowOutput) && !rbt.shownResult {
		body, _ := io.ReadAll(resp.Body)
		var prettyJSON bytes.Buffer
		err := json.Indent(&prettyJSON, body, "", "\t")
		require.NoError(b, err)
		fmt.Println(req.URL.String()+" : ", prettyJSON.String())
		rbt.shownResult = true
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status code %v not ok: %v", resp.StatusCode, resp.Status)
	}
	b.StartTimer()

	return nil
}

func BenchmarkRestTests(
	data bk.BenchData, _ bk.SignatureScheme,
) bk.TestSuite {
	var tests = []*RestBenchTest{
		{
			name: "personalPeriodicLimit",
			params: map[string]string{
				"client_id": data.Clients[0],
			},
		},
		{
			name: "globalPeriodicLimit",
		},
		{
			name: "pourAmount",
		},
		{
			name: "getConfig",
		},
	}
	var testsI []bk.BenchTestI
	for _, test := range tests {
		testsI = append(testsI, test)
	}
	return bk.TestSuite{
		Source:     bk.FaucetRest,
		Benchmarks: testsI,
	}
}
