package sharder_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/core/common"
	"0chain.net/sharder"
)

func makeTestURL(url url.URL, values map[string]string) string {
	q := url.Query()

	for k, v := range values {
		q.Set(k, v)
	}
	url.RawQuery = q.Encode()

	return url.String()
}

func TestBlockHandler(t *testing.T) {
	t.Parallel()

	const baseUrl = "/v1/block/get"

	b := block.NewBlock("", 1)
	b.HashBlock()

	chain.ServerChain = chain.Provider().(*chain.Chain)
	chain.ServerChain.AddBlock(b)

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
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(common.UserRateLimit(common.ToJSONResponse(sharder.BlockHandler)))

			handler.ServeHTTP(rr, tt.request)

			if status := rr.Code; status != tt.wantStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.wantStatus)
			}
		})
	}
}

func TestChainStatsWriter(t *testing.T) {
	t.Parallel()

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
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(common.UserRateLimit(sharder.ChainStatsWriter))

			handler.ServeHTTP(rr, tt.request)

			if status := rr.Code; status != tt.wantStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.wantStatus)
			}
		})
	}
}

func TestSharderStatsHandler(t *testing.T) {
	t.Parallel()

	const baseUrl = "/v1/sharder/get/stats"

	type test struct {
		name       string
		request    *http.Request
		wantStatus int
	}

	tests := []test{
		{
			name: "Test_SharderStatsHandler_OK",
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
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(common.UserRateLimit(common.ToJSONResponse(sharder.SharderStatsHandler)))

			// setting lfb because return handler function is panic with nil lfb
			b := block.NewBlock("", 132)
			sharder.GetSharderChain().SetLatestFinalizedBlock(b)

			handler.ServeHTTP(rr, tt.request)

			if status := rr.Code; status != tt.wantStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.wantStatus)
			}
		})
	}
}
