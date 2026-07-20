package sharder

import (
	"0chain.net/chaincore/node"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"0chain.net/chaincore/block"
	"0chain.net/core/common"
	"0chain.net/core/encryption"
)

func init() {
	common.ConfigRateLimits()
}

func makeTestURL(url url.URL, values map[string]string) string {
	q := url.Query()

	for k, v := range values {
		q.Set(k, v)
	}
	url.RawQuery = q.Encode()

	return url.String()
}

func TestBlockHandler(t *testing.T) {
	const baseUrl = "/v1/block/get"

	b := block.NewBlock("", 10)
	b.Hash = encryption.Hash("data")

	sc := makeTestChain(t)
	sc.AddBlock(b)

	cl := initDBs(t)
	defer cl()
	sc.Initialize()
	mb := block.NewMagicBlock()
	mb.Miners = node.NewPool(node.NodeTypeMiner)
	mb.Sharders = node.NewPool(node.NodeTypeSharder)
	sc.SetMagicBlock(mb)
	sc.SetLatestFinalizedBlock(b)

	type test struct {
		name       string
		request    *http.Request
		wantStatus int
	}

	tests := []test{
		{
			name: "Test_BlockHandler_OK",
			request: func() *http.Request {
				u, err := url.Parse(baseUrl)
				if err != nil {
					t.Fatal(err)
				}
				v := map[string]string{
					"block": b.Hash,
				}

				req, err := http.NewRequest(http.MethodGet, makeTestURL(*u, v), nil)
				if err != nil {
					t.Fatal(err)
				}

				return req
			}(),
			wantStatus: http.StatusOK,
		},
		{
			name: "Test_BlockHandler_Empty_Block_ERR",
			request: func() *http.Request {
				req, err := http.NewRequest(http.MethodGet, baseUrl, nil)
				if err != nil {
					t.Fatal(err)
				}

				return req
			}(),
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(common.UserRateLimit(common.ToJSONResponse(BlockHandler)))

			handler.ServeHTTP(rr, tt.request)

			if status := rr.Code; status != tt.wantStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.wantStatus)
			}
		})
	}
}

func TestChainStatsWriter(t *testing.T) {
	const baseUrl = "/_chain_stats"

	type test struct {
		name       string
		request    *http.Request
		wantStatus int
	}

	tests := []test{
		{
			name: "Test_ChainStatsWriter_OK",
			request: func() *http.Request {
				req, err := http.NewRequest(http.MethodGet, baseUrl, nil)
				if err != nil {
					t.Fatal(err)
				}

				return req
			}(),
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(common.UserRateLimit(ChainStatsWriter))

			handler.ServeHTTP(rr, tt.request)

			if status := rr.Code; status != tt.wantStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.wantStatus)
			}
		})
	}
}
