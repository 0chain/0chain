package sharder_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/round"
	"0chain.net/core/common"
	"0chain.net/core/encryption"
	"0chain.net/sharder"
	"0chain.net/sharder/blockstore"
)

func TestLatestRoundRequestHandler(t *testing.T) {
	const baseUrl = "/v1/_s2s/latest_round/get"

	sc := sharder.GetSharderChain()
	var num int64 = 1
	r := round.NewRound(num)
	sc.AddRound(r)

	type test struct {
		name          string
		request       *http.Request
		wantStatus    int
		wantCurrRound int64
	}

	tests := []test{
		{
			name: "Test_LatestRoundRequestHandler_OK",
			request: func() *http.Request {
				req, err := http.NewRequest(http.MethodGet, baseUrl, nil)
				if err != nil {
					t.Fatal(err)
				}

				return req
			}(),
			wantCurrRound: num,
			wantStatus:    http.StatusOK,
		},
		{
			name: "Test_LatestRoundRequestHandler_ERR",
			request: func() *http.Request {
				req, err := http.NewRequest(http.MethodGet, baseUrl, nil)
				if err != nil {
					t.Fatal(err)
				}

				return req
			}(),
			wantCurrRound: 123,
			wantStatus:    http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(common.UserRateLimit(common.ToJSONResponse(sharder.LatestRoundRequestHandler)))
			sc.CurrentRound = tt.wantCurrRound

			handler.ServeHTTP(rr, tt.request)

			if status := rr.Code; status != tt.wantStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
				fmt.Println(rr.Body.String())
			}
		})
	}
}

func TestBlockSummaryRequestHandler(t *testing.T) {
	t.Parallel()

	const baseUrl = "/v1/_s2s/blocksummary/get"

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
			name: "Test_BlockSummaryRequestHandler_Empty_Hash_ERR",
			request: func() *http.Request {
				req, err := http.NewRequest(http.MethodGet, baseUrl, nil)
				if err != nil {
					t.Fatal(err)
				}

				return req
			}(),
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "Test_BlockSummaryRequestHandler_OK",
			request: func() *http.Request {
				u, err := url.Parse(baseUrl)
				if err != nil {
					t.Fatal(err)
				}
				v := map[string]string{
					"hash": encryption.Hash("data"),
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
			name: "Test_BlockSummaryRequestHandler_Invalid_Hash_ERR",
			request: func() *http.Request {
				u, err := url.Parse(baseUrl)
				if err != nil {
					t.Fatal(err)
				}
				v := map[string]string{
					"hash": encryption.Hash("data")[:62],
				}

				req, err := http.NewRequest(http.MethodGet, makeTestURL(*u, v), nil)

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
			handler := http.HandlerFunc(common.UserRateLimit(common.ToJSONResponse(sharder.BlockSummaryRequestHandler)))

			handler.ServeHTTP(rr, tt.request)

			if status := rr.Code; status != tt.wantStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.wantStatus)
			}
		})
	}
}

func TestRoundBlockRequestHandler(t *testing.T) {
	const baseUrl = "/v1/_s2s/block/get"

	b := block.NewBlock("", 2)
	b.Hash = encryption.Hash("data")
	sharder.GetSharderChain().AddBlock(b)

	storeB := block.NewBlock("", 2)
	storeB.Hash = encryption.Hash("another data")
	if err := blockstore.GetStore().Write(storeB); err != nil {
		t.Fatal(err)
	}

	type test struct {
		name       string
		request    *http.Request
		wantStatus int
	}

	tests := []test{
		{
			name: "Test_RoundBlockRequestHandler_Empty_Block_ERR",
			request: func() *http.Request {
				req, err := http.NewRequest(http.MethodGet, baseUrl, nil)
				if err != nil {
					t.Fatal(err)
				}

				return req
			}(),
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "Test_RoundBlockRequestHandler_OK",
			request: func() *http.Request {
				u, err := url.Parse(baseUrl)
				if err != nil {
					t.Fatal(err)
				}

				b := block.NewBlock("", 1)
				b.Hash = encryption.Hash("Test_RoundBlockRequestHandler_OK") // uniq hash
				if err := blockstore.GetStore().Write(b); err != nil {
					t.Fatal(err)
				}

				v := map[string]string{
					"hash":  b.Hash,
					"round": "1",
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
			name: "Test_RoundBlockRequestHandler_Stored_Block_OK",
			request: func() *http.Request {
				u, err := url.Parse(baseUrl)
				if err != nil {
					t.Fatal(err)
				}
				v := map[string]string{
					"hash": b.Hash,
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
			name: "Test_RoundBlockRequestHandler_Unknown_Block_And_Invalid_Round_ERR",
			request: func() *http.Request {
				u, err := url.Parse(baseUrl)
				if err != nil {
					t.Fatal(err)
				}
				v := map[string]string{
					"hash": encryption.Hash("Test_RoundBlockRequestHandler_Unknown_Block_And_Invalid_Round_ERR"),
				}

				req, err := http.NewRequest(http.MethodGet, makeTestURL(*u, v), nil)
				if err != nil {
					t.Fatal(err)
				}

				return req
			}(),
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "Test_RoundBlockRequestHandler_Block_From_Store_OK",
			request: func() *http.Request {
				u, err := url.Parse(baseUrl)
				if err != nil {
					t.Fatal(err)
				}
				v := map[string]string{
					"hash":  storeB.Hash,
					"round": "2",
				}

				req, err := http.NewRequest(http.MethodGet, makeTestURL(*u, v), nil)
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
			handler := http.HandlerFunc(common.UserRateLimit(common.ToJSONResponse(sharder.RoundBlockRequestHandler)))

			handler.ServeHTTP(rr, tt.request)

			if status := rr.Code; status != tt.wantStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.wantStatus)
				fmt.Println(rr.Body.String())
			}
		})
	}
}

func TestRoundSummariesHandler(t *testing.T) {
	t.Parallel()

	const baseUrl = "/v1/_s2s/roundsummaries/get"

	type test struct {
		name       string
		request    *http.Request
		wantStatus int
	}

	tests := []test{
		{
			name: "Test_RoundSummariesHandler_OK",
			request: func() *http.Request {
				u, err := url.Parse(baseUrl)
				if err != nil {
					t.Fatal(err)
				}
				v := map[string]string{
					"round": "1",
					"range": "2",
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
			name: "Test_RoundSummariesHandler_ERR",
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
			handler := http.HandlerFunc(common.UserRateLimit(common.ToJSONResponse(sharder.RoundSummariesHandler)))

			handler.ServeHTTP(rr, tt.request)

			if status := rr.Code; status != tt.wantStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.wantStatus)
				fmt.Println(rr.Body.String())
			}
		})
	}
}

func TestBlockSummariesHandler(t *testing.T) {
	t.Parallel()

	const baseUrl = "/v1/_s2s/roundsummaries/get"

	type test struct {
		name       string
		request    *http.Request
		wantStatus int
	}

	tests := []test{
		{
			name: "Test_RoundSummariesHandler_Empty_Params_ERR",
			request: func() *http.Request {
				req, err := http.NewRequest(http.MethodGet, baseUrl, nil)
				if err != nil {
					t.Fatal(err)
				}

				return req
			}(),
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "Test_RoundSummariesHandler_OK",
			request: func() *http.Request {
				u, err := url.Parse(baseUrl)
				if err != nil {
					t.Fatal(err)
				}
				v := map[string]string{
					"round": "1",
					"range": "1",
				}

				req, err := http.NewRequest(http.MethodGet, makeTestURL(*u, v), nil)

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
			handler := http.HandlerFunc(common.UserRateLimit(common.ToJSONResponse(sharder.BlockSummariesHandler)))

			handler.ServeHTTP(rr, tt.request)

			if status := rr.Code; status != tt.wantStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.wantStatus)
				fmt.Println(rr.Body.String())
			}
		})
	}
}

func TestRoundRequestHandler(t *testing.T) {
	t.Parallel()

	const baseUrl = "/v1/_s2s/round/get"

	type test struct {
		name       string
		request    *http.Request
		wantStatus int
	}

	tests := []test{
		{
			name: "Test_RoundRequestHandler_Empty_Args_ERR",
			request: func() *http.Request {
				req, err := http.NewRequest(http.MethodGet, baseUrl, nil)
				if err != nil {
					t.Fatal(err)
				}

				return req
			}(),
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "Test_RoundRequestHandler_OK",
			request: func() *http.Request {
				u, err := url.Parse(baseUrl)
				if err != nil {
					t.Fatal(err)
				}
				v := map[string]string{
					"round": "1",
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
			name: "Test_RoundRequestHandler_From_Sharder_OK",
			request: func() *http.Request {
				u, err := url.Parse(baseUrl)
				if err != nil {
					t.Fatal(err)
				}
				v := map[string]string{
					"round": "57",
				}

				sharder.GetSharderChain().AddRound(round.NewRound(57))

				req, err := http.NewRequest(http.MethodGet, makeTestURL(*u, v), nil)

				if err != nil {
					t.Fatal(err)
				}

				return req
			}(),
			wantStatus: http.StatusOK,
		},
		{
			name: "Test_RoundRequestHandler_Neg_Num_ERR",
			request: func() *http.Request {
				u, err := url.Parse(baseUrl)
				if err != nil {
					t.Fatal(err)
				}
				v := map[string]string{
					"round": "-1",
				}

				req, err := http.NewRequest(http.MethodGet, makeTestURL(*u, v), nil)

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
			handler := http.HandlerFunc(common.UserRateLimit(common.ToJSONResponse(sharder.RoundRequestHandler)))

			handler.ServeHTTP(rr, tt.request)

			if status := rr.Code; status != tt.wantStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.wantStatus)
				fmt.Println(rr.Body.String())
			}
		})
	}
}
