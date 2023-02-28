package sharder

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/state"
	"0chain.net/core/common"
	"0chain.net/core/encryption"
	"0chain.net/smartcontract/dbs/event"
	"github.com/stretchr/testify/require"
)

const (
	clientID        = "client_id"
	ethereumAddress = "ethereum_address"
	hash            = "hash"
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

func TestMintNonceHandler(t *testing.T) {
	const baseUrl = "/v1/mint_nonce"

	b := block.NewBlock("", 10)
	b.Hash = encryption.Hash("data")

	sc := makeTestChain(t)
	sc.AddBlock(b)
	sc.SetLatestFinalizedBlock(b)

	cl := initDBs(t)
	defer cl()

	eventDb, err := event.NewInMemoryEventDb(config.DbAccess{}, config.DbSettings{
		Debug:                 true,
		PartitionChangePeriod: 1,
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		err = eventDb.Drop()
		require.NoError(t, err)

		eventDb.Close()
	})

	sc.EventDb = eventDb

	tests := []struct {
		name string
		body func(t *testing.T)
	}{
		{
			name: "Get processed mint nonces of the client, which hasn't performed any mint operations, should work",
			body: func(t *testing.T) {
				err := eventDb.Get().Model(&event.User{}).Create(&event.User{
					UserID: clientID,
				}).Error
				require.NoError(t, err)

				target := url.URL{Path: baseUrl}

				query := target.Query()

				query.Add("client_id", clientID)

				target.RawQuery = query.Encode()

				req := httptest.NewRequest(http.MethodGet, target.String(), nil)

				respRaw, err := MintNonceHandler(context.Background(), req)
				require.NoError(t, err)

				resp, ok := respRaw.(int64)
				require.True(t, ok)
				require.Equal(t, int64(0), resp)

				err = eventDb.Get().Model(&event.User{}).Where("user_id = ?", clientID).Delete(&event.User{}).Error
				require.NoError(t, err)
			},
		},
		{
			name: "Get mint nonces of the client, which has performed mint operation, should work",
			body: func(t *testing.T) {
				err := eventDb.Get().Model(&event.User{}).Create(&event.User{
					UserID:    clientID,
					MintNonce: 1,
				}).Error
				require.NoError(t, err)

				target := url.URL{Path: baseUrl}

				query := target.Query()

				query.Add("client_id", clientID)

				target.RawQuery = query.Encode()

				req := httptest.NewRequest(http.MethodGet, target.String(), nil)

				respRaw, err := MintNonceHandler(context.Background(), req)
				require.NoError(t, err)

				resp, ok := respRaw.(int64)
				require.True(t, ok)
				require.Equal(t, int64(1), resp)

				err = eventDb.Get().Model(&event.User{}).Where("user_id = ?", clientID).Delete(&event.User{}).Error
				require.NoError(t, err)
			},
		},
		{
			name: "Get processed mint nonces not providing client id, should not work",
			body: func(t *testing.T) {
				target := url.URL{Path: baseUrl}

				req := httptest.NewRequest(http.MethodGet, target.String(), nil)

				resp, err := MintNonceHandler(context.Background(), req)
				require.Error(t, err)
				require.Nil(t, resp)
			},
		},
		{
			name: "Get processed mint nonces for the client, which does not exist, should not work",
			body: func(t *testing.T) {
				target := url.URL{Path: baseUrl}

				query := target.Query()

				query.Add("client_id", clientID)

				target.RawQuery = query.Encode()

				req := httptest.NewRequest(http.MethodGet, target.String(), nil)

				respRaw, err := MintNonceHandler(context.Background(), req)
				require.Error(t, err)
				require.Nil(t, respRaw)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, test.body)
	}
}

func TestNotProcessedBurnTicketsHandler(t *testing.T) {
	const baseUrl = "/v1/not_processed_burn_tickets"

	b := block.NewBlock("", 10)
	b.Hash = encryption.Hash("data")

	sc := makeTestChain(t)
	sc.AddBlock(b)
	sc.SetLatestFinalizedBlock(b)

	cl := initDBs(t)
	defer cl()

	eventDb, err := event.NewInMemoryEventDb(config.DbAccess{}, config.DbSettings{
		Debug:                 true,
		PartitionChangePeriod: 1,
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		err = eventDb.Drop()
		require.NoError(t, err)

		eventDb.Close()
	})

	sc.EventDb = eventDb

	tests := []struct {
		name string
		body func(t *testing.T)
	}{
		{
			name: "Get not processed burn tickets of the client, which hasn't performed any burn operations, should work",
			body: func(t *testing.T) {
				target := url.URL{Path: baseUrl}

				query := target.Query()

				query.Add("ethereum_address", ethereumAddress)
				query.Add("client_id", clientID)
				query.Add("nonce", "0")

				target.RawQuery = query.Encode()

				req := httptest.NewRequest(http.MethodGet, target.String(), nil)

				respRaw, err := NotProcessedBurnTicketsHandler(context.Background(), req)
				require.NoError(t, err)

				resp, ok := respRaw.([]*state.BurnTicket)
				require.True(t, ok)
				require.Len(t, resp, 0)
			},
		},
		{
			name: "Get not processed burn tickets of the client, which has performed burn operation, should work",
			body: func(t *testing.T) {
				err := eventDb.Get().Model(&event.BurnTicket{}).Create(&event.BurnTicket{
					UserID:          clientID,
					EthereumAddress: ethereumAddress,
					Hash:            hash,
					Nonce:           1,
				}).Error
				require.NoError(t, err)

				target := url.URL{Path: baseUrl}

				query := target.Query()

				query.Add("ethereum_address", ethereumAddress)
				query.Add("client_id", clientID)
				query.Add("nonce", "0")

				target.RawQuery = query.Encode()

				req := httptest.NewRequest(http.MethodGet, target.String(), nil)

				respRaw, err := NotProcessedBurnTicketsHandler(context.Background(), req)
				require.NoError(t, err)

				resp, ok := respRaw.([]*state.BurnTicket)
				require.True(t, ok)
				require.Len(t, resp, 1)

				err = eventDb.Get().Model(&event.BurnTicket{}).Where("user_id = ?", clientID).Delete(&event.BurnTicket{}).Error
				require.NoError(t, err)
			},
		},
		{
			name: "Get not processed burn tickets not providing client id, should not work",
			body: func(t *testing.T) {
				target := url.URL{Path: baseUrl}

				query := target.Query()

				query.Add("ethereum_address", ethereumAddress)
				query.Add("nonce", "0")

				target.RawQuery = query.Encode()

				req := httptest.NewRequest(http.MethodGet, target.String(), nil)

				resp, err := NotProcessedBurnTicketsHandler(context.Background(), req)
				require.Error(t, err)
				require.Nil(t, resp)
			},
		},
		{
			name: "Get not processed burn tickets not providing ethereum address, should not work",
			body: func(t *testing.T) {
				target := url.URL{Path: baseUrl}

				query := target.Query()

				query.Add("client_id", clientID)
				query.Add("nonce", "0")

				target.RawQuery = query.Encode()

				req := httptest.NewRequest(http.MethodGet, target.String(), nil)

				resp, err := NotProcessedBurnTicketsHandler(context.Background(), req)
				require.Error(t, err)
				require.Nil(t, resp)
			},
		},
		{
			name: "Get not processed burn tickets not providing nonce, should work",
			body: func(t *testing.T) {
				err := eventDb.Get().Model(&event.BurnTicket{}).Create(&event.BurnTicket{
					UserID:          clientID,
					EthereumAddress: ethereumAddress,
					Hash:            hash,
					Nonce:           1,
				}).Error
				require.NoError(t, err)

				target := url.URL{Path: baseUrl}

				query := target.Query()

				query.Add("ethereum_address", ethereumAddress)
				query.Add("client_id", clientID)

				target.RawQuery = query.Encode()

				req := httptest.NewRequest(http.MethodGet, target.String(), nil)

				respRaw, err := NotProcessedBurnTicketsHandler(context.Background(), req)
				require.NoError(t, err)

				resp, ok := respRaw.([]*state.BurnTicket)
				require.True(t, ok)
				require.Len(t, resp, 1)

				err = eventDb.Get().Model(&event.BurnTicket{}).Where("user_id = ?", clientID).Delete(&event.BurnTicket{}).Error
				require.NoError(t, err)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, test.body)
	}
}

func TestBlockHandler(t *testing.T) {
	const baseUrl = "/v1/block/get"

	b := block.NewBlock("", 10)
	b.Hash = encryption.Hash("data")

	sc := makeTestChain(t)
	sc.AddBlock(b)
	sc.SetLatestFinalizedBlock(b)

	cl := initDBs(t)
	defer cl()

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
