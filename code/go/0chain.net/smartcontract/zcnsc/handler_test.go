package zcnsc_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/config"
	"0chain.net/core/encryption"
	"0chain.net/core/memorystore"
	"0chain.net/smartcontract/dbs/event"
	"0chain.net/smartcontract/rest"
	. "0chain.net/smartcontract/zcnsc"
	"github.com/stretchr/testify/require"
)

const (
	clientID        = "client_id"
	ethereumAddress = "ethereum_address"
	hash            = "hash"
)

func init() {
	block.SetupEntity(memorystore.GetStorageProvider())
}

func TestMintNonceHandler(t *testing.T) {
	const baseUrl = "/v1/mint_nonce"

	b := block.NewBlock("", 10)
	b.Hash = encryption.Hash("data")

	rh := rest.NewRestHandler(&rest.TestQueryChainer{})
	srh := NewZcnRestHandler(rh)

	sctx := MakeMockTimedQueryStateContext()

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

	sctx.SetEventDb(eventDb)

	srh.SetQueryStateContext(sctx)

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

				rr := httptest.NewRecorder()
				handler := http.HandlerFunc(srh.MintNonceHandler)

				req := httptest.NewRequest(http.MethodGet, target.String(), nil)

				handler.ServeHTTP(rr, req)

				require.Equal(t, http.StatusOK, rr.Result().StatusCode)

				var resp int64
				err = json.NewDecoder(rr.Body).Decode(&resp)
				require.NoError(t, err)
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

				rr := httptest.NewRecorder()
				handler := http.HandlerFunc(srh.MintNonceHandler)

				req := httptest.NewRequest(http.MethodGet, target.String(), nil)

				handler.ServeHTTP(rr, req)

				require.Equal(t, http.StatusOK, rr.Result().StatusCode)

				var resp int64
				err = json.NewDecoder(rr.Body).Decode(&resp)
				require.NoError(t, err)
				require.Equal(t, int64(1), resp)

				err = eventDb.Get().Model(&event.User{}).Where("user_id = ?", clientID).Delete(&event.User{}).Error
				require.NoError(t, err)
			},
		},
		{
			name: "Get processed mint nonces not providing client id, should not work",
			body: func(t *testing.T) {
				target := url.URL{Path: baseUrl}

				rr := httptest.NewRecorder()
				handler := http.HandlerFunc(srh.MintNonceHandler)

				req := httptest.NewRequest(http.MethodGet, target.String(), nil)

				handler.ServeHTTP(rr, req)

				require.Equal(t, http.StatusBadRequest, rr.Result().StatusCode)

				var resp int64
				err = json.NewDecoder(rr.Body).Decode(&resp)
				require.Error(t, err)
			},
		},
		{
			name: "Get processed mint nonces for the client, which does not exist, should not work",
			body: func(t *testing.T) {
				target := url.URL{Path: baseUrl}

				query := target.Query()

				query.Add("client_id", clientID)

				target.RawQuery = query.Encode()

				rr := httptest.NewRecorder()
				handler := http.HandlerFunc(srh.MintNonceHandler)

				req := httptest.NewRequest(http.MethodGet, target.String(), nil)

				handler.ServeHTTP(rr, req)

				require.Equal(t, http.StatusBadRequest, rr.Result().StatusCode)

				var resp int64
				err = json.NewDecoder(rr.Body).Decode(&resp)
				require.Error(t, err)
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

	rh := rest.NewRestHandler(&rest.TestQueryChainer{})
	srh := NewZcnRestHandler(rh)

	sctx := MakeMockTimedQueryStateContext()

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

	sctx.SetEventDb(eventDb)

	srh.SetQueryStateContext(sctx)

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
				query.Add("nonce", "0")

				target.RawQuery = query.Encode()

				rr := httptest.NewRecorder()
				handler := http.HandlerFunc(srh.NotProcessedBurnTicketsHandler)

				req := httptest.NewRequest(http.MethodGet, target.String(), nil)

				handler.ServeHTTP(rr, req)

				require.Equal(t, http.StatusOK, rr.Result().StatusCode)

				var resp []*BurnTicket
				err = json.NewDecoder(rr.Body).Decode(&resp)
				require.NoError(t, err)
				require.Len(t, resp, 0)
			},
		},
		{
			name: "Get not processed burn tickets of the client, which has performed burn operation, should work",
			body: func(t *testing.T) {
				err := eventDb.Get().Model(&event.BurnTicket{}).Create(&event.BurnTicket{
					EthereumAddress: ethereumAddress,
					Hash:            hash,
					Nonce:           1,
				}).Error
				require.NoError(t, err)

				target := url.URL{Path: baseUrl}

				query := target.Query()

				query.Add("ethereum_address", ethereumAddress)
				query.Add("nonce", "0")

				target.RawQuery = query.Encode()

				rr := httptest.NewRecorder()
				handler := http.HandlerFunc(srh.NotProcessedBurnTicketsHandler)

				req := httptest.NewRequest(http.MethodGet, target.String(), nil)

				handler.ServeHTTP(rr, req)

				require.Equal(t, http.StatusOK, rr.Result().StatusCode)

				var resp []*BurnTicket
				err = json.NewDecoder(rr.Body).Decode(&resp)
				require.NoError(t, err)
				require.Len(t, resp, 1)

				err = eventDb.Get().Model(&event.BurnTicket{}).Where("ethereum_address = ?", ethereumAddress).Delete(&event.BurnTicket{}).Error
				require.NoError(t, err)
			},
		},
		{
			name: "Get not processed burn tickets not providing ethereum address, should not work",
			body: func(t *testing.T) {
				target := url.URL{Path: baseUrl}

				query := target.Query()

				query.Add("nonce", "0")

				target.RawQuery = query.Encode()

				rr := httptest.NewRecorder()
				handler := http.HandlerFunc(srh.MintNonceHandler)

				req := httptest.NewRequest(http.MethodGet, target.String(), nil)

				handler.ServeHTTP(rr, req)

				require.Equal(t, http.StatusBadRequest, rr.Result().StatusCode)

				var resp int64
				err = json.NewDecoder(rr.Body).Decode(&resp)
				require.Error(t, err)
			},
		},
		{
			name: "Get not processed burn tickets not providing nonce, should work",
			body: func(t *testing.T) {
				err := eventDb.Get().Model(&event.BurnTicket{}).Create(&event.BurnTicket{
					EthereumAddress: ethereumAddress,
					Hash:            hash,
					Nonce:           1,
				}).Error
				require.NoError(t, err)

				target := url.URL{Path: baseUrl}

				query := target.Query()

				query.Add("ethereum_address", ethereumAddress)

				target.RawQuery = query.Encode()

				rr := httptest.NewRecorder()
				handler := http.HandlerFunc(srh.NotProcessedBurnTicketsHandler)

				req := httptest.NewRequest(http.MethodGet, target.String(), nil)

				handler.ServeHTTP(rr, req)

				require.Equal(t, http.StatusOK, rr.Result().StatusCode)

				var resp []*BurnTicket
				err = json.NewDecoder(rr.Body).Decode(&resp)
				require.NoError(t, err)
				require.Len(t, resp, 1)

				err = eventDb.Get().Model(&event.BurnTicket{}).Where("ethereum_address = ?", ethereumAddress).Delete(&event.BurnTicket{}).Error
				require.NoError(t, err)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, test.body)
	}
}
