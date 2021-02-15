package sharder

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rcrowley/go-metrics"

	"0chain.net/chaincore/chain"
)

func init() {
	serverChain := chain.NewChainFromConfig()
	SetupSharderChain(serverChain)

	GetSharderChain().BlockSyncStats.cycle[DeepScan].BlockSyncTimer = metrics.NewTimer()
	GetSharderChain().BlockSyncStats.cycle[ProximityScan].BlockSyncTimer = metrics.NewTimer()
}

func TestHealthCheckWriter(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "/_health_check", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(HealthCheckWriter)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}
