package sharder

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/node"

	"github.com/rcrowley/go-metrics"

	"0chain.net/chaincore/chain"
)

func makeTestChain(t *testing.T) *Chain {
	ch, ok := chain.Provider().(*chain.Chain)
	if !ok {
		t.Fatal("types missmatching")
	}
	ch.ChainConfig = chain.NewConfigImpl(&chain.ConfigData{BlockSize: 1024})
	config.Configuration().ChainConfig = ch.ChainConfig
	ch.Initialize()
	SetupSharderChain(ch)
	chain.SetServerChain(ch)
	mb := block.NewMagicBlock()
	mb.Miners = node.NewPool(node.NodeTypeMiner)
	mb.Sharders = node.NewPool(node.NodeTypeSharder)
	ch.SetMagicBlock(mb)
	return GetSharderChain()
}

func TestHealthCheckWriter(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "/_healthcheck", nil)
	if err != nil {
		t.Fatal(err)
	}

	ch := makeTestChain(t)
	ch.BlockSyncStats.cycle[DeepScan] = CycleControl{
		BlockSyncTimer: metrics.NewTimer(),
	}
	ch.BlockSyncStats.cycle[ProximityScan] = CycleControl{
		BlockSyncTimer: metrics.NewTimer(),
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(HealthCheckWriter)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}
