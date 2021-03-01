package sharder_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"0chain.net/chaincore/block"
	"0chain.net/core/common"
	"0chain.net/core/encryption"
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
	const baseUrl = "/v1/block/get"

	b := block.NewBlock("", 10)
	b.Hash = encryption.Hash("data")

	sc := sharder.GetSharderChain()
	sc.AddBlock(b)
	sc.LatestFinalizedBlock = b

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
			name: "Test_BlockHandler_Invalid_Round_ERR",
			request: func() *http.Request {
				u, err := url.Parse(baseUrl)
				if err != nil {
					t.Fatal(err)
				}
				v := map[string]string{
					"block": b.Hash,
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
		{
			name: "Test_BlockHandler_Round_Non_Existing_Block_ERR",
			request: func() *http.Request {
				u, err := url.Parse(baseUrl)
				if err != nil {
					t.Fatal(err)
				}
				v := map[string]string{
					"block": b.Hash[:62],
					"round": "9",
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
			name: "Test_BlockHandler_ERR", // test covers error when latest finalized block round lower than round in request
			request: func() *http.Request {
				u, err := url.Parse(baseUrl)
				if err != nil {
					t.Fatal(err)
				}
				v := map[string]string{
					"block": b.Hash,
					"round": "11",
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
		{
			name: "Test_BlockHandler_Invalid_Round_ERR",
			request: func() *http.Request {
				u, err := url.Parse(baseUrl)
				if err != nil {
					t.Fatal(err)
				}
				v := map[string]string{
					"block": b.Hash,
					"round": "qwe",
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
		t.Run(tt.name, func(t *testing.T) {
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

func TestSetupHandlers(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "Test_SetupHandlers_OK",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sharder.SetupHandlers()
		})
	}
}

func TestChainStatsHandler(t *testing.T) {
	const baseUrl = "/v1/chain/get/stats"

	type test struct {
		name       string
		request    *http.Request
		wantStatus int
	}

	tests := []test{
		{
			name: "Test_ChainStatsHandler_OK",
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
			handler := http.HandlerFunc(common.UserRateLimit(common.ToJSONResponse(sharder.ChainStatsHandler)))

			handler.ServeHTTP(rr, tt.request)

			if status := rr.Code; status != tt.wantStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.wantStatus)
			}
		})
	}
}

func TestMagicBlockHandler(t *testing.T) {
	sharder.GetSharderChain().LatestFinalizedBlock = block.NewBlock("", 1)

	const baseUrl = "/v1/block/magic/get"

	type test struct {
		name       string
		request    *http.Request
		wantStatus int
	}

	tests := []test{
		{
			name: "Test_MagicBlockHandler_Empty_Magic_Block_Num_ERR",
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
			name: "Test_MagicBlockHandler_OK",
			request: func() *http.Request {
				mbm := &block.MagicBlockMap{}
				mbm.Hash = encryption.Hash("mbm data")
				mbm.ID = "1"
				if err := mbm.GetEntityMetadata().GetStore().Write(common.GetRootContext(), mbm); err != nil {
					t.Fatal(err)
				}

				b := block.NewBlock("", 1)
				b.Hash = mbm.Hash
				sharder.GetSharderChain().AddBlock(b)

				u, err := url.Parse(baseUrl)
				if err != nil {
					t.Fatal(err)
				}
				v := map[string]string{
					"magic_block_number": mbm.ID,
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
			name: "Test_MagicBlockHandler_Unknown_Block_For_MBM_ERR",
			request: func() *http.Request {
				mbm := &block.MagicBlockMap{}
				mbm.Hash = encryption.Hash("mbm data")[:62]
				mbm.ID = "2"
				if err := mbm.GetEntityMetadata().GetStore().Write(common.GetRootContext(), mbm); err != nil {
					t.Fatal(err)
				}

				u, err := url.Parse(baseUrl)
				if err != nil {
					t.Fatal(err)
				}
				v := map[string]string{
					"magic_block_number": mbm.ID,
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
			handler := http.HandlerFunc(common.UserRateLimit(common.ToJSONResponse(sharder.MagicBlockHandler)))

			handler.ServeHTTP(rr, tt.request)

			if status := rr.Code; status != tt.wantStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.wantStatus)
			}
		})
	}
}
