package sharder

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/round"
	"0chain.net/core/common"
)

func TestLatestRoundRequestHandler(t *testing.T) {
	const baseUrl = "/v1/_s2s/latest-round"

	sc := makeTestChain(t)
	var num int64 = 1
	r := round.NewRound(num§)
	sc.AddRound(r)
	sc.SetCurrentRound(num)

	type test struct {
		name       string
		request    *http.Request
		wantStatus int
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
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(common.UserRateLimit(common.ToJSONResponse(LatestRoundRequestHandler)))

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

	const baseUrl = "/v1/_s2s/block-summary"

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
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cl := initDBs(t)
			defer cl()

			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(common.UserRateLimit(common.ToJSONResponse(BlockSummaryRequestHandler)))

			handler.ServeHTTP(rr, tt.request)

			if status := rr.Code; status != tt.wantStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.wantStatus)
			}
		})
	}
}

func TestRoundBlockRequestHandler(t *testing.T) {
	const baseUrl = "/v1/_s2s/block/get"

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
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(common.UserRateLimit(common.ToJSONResponse(RoundBlockRequestHandler)))

			handler.ServeHTTP(rr, tt.request)

			if status := rr.Code; status != tt.wantStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.wantStatus)
				fmt.Println(rr.Body.String())
			}
		})
	}
}
