package zcnsc

//
//func TestMintNonceHandler(t *testing.T) {
//	const baseUrl = "/v1/mint_nonce"
//
//	b := block.NewBlock("", 10)
//	b.Hash = encryption.Hash("data")
//
//	sc := makeTestChain(t)
//	sc.AddBlock(b)
//	sc.SetLatestFinalizedBlock(b)
//
//	cl := initDBs(t)
//	defer cl()
//
//	eventDb, err := event.NewInMemoryEventDb(config.DbAccess{}, config.DbSettings{
//		Debug:                 true,
//		PartitionChangePeriod: 1,
//	})
//	require.NoError(t, err)
//
//	t.Cleanup(func() {
//		err = eventDb.Drop()
//		require.NoError(t, err)
//
//		eventDb.Close()
//	})
//
//	sc.EventDb = eventDb
//
//	tests := []struct {
//		name string
//		body func(t *testing.T)
//	}{
//		{
//			name: "Get processed mint nonces of the client, which hasn't performed any mint operations, should work",
//			body: func(t *testing.T) {
//				err := eventDb.Get().Model(&event.User{}).Create(&event.User{
//					UserID: clientID,
//				}).Error
//				require.NoError(t, err)
//
//				target := url.URL{Path: baseUrl}
//
//				query := target.Query()
//
//				query.Add("client_id", clientID)
//
//				target.RawQuery = query.Encode()
//
//				req := httptest.NewRequest(http.MethodGet, target.String(), nil)
//
//				respRaw, err := MintNonceHandler(context.Background(), req)
//				require.NoError(t, err)
//
//				resp, ok := respRaw.(int64)
//				require.True(t, ok)
//				require.Equal(t, int64(0), resp)
//
//				err = eventDb.Get().Model(&event.User{}).Where("user_id = ?", clientID).Delete(&event.User{}).Error
//				require.NoError(t, err)
//			},
//		},
//		{
//			name: "Get mint nonces of the client, which has performed mint operation, should work",
//			body: func(t *testing.T) {
//				err := eventDb.Get().Model(&event.User{}).Create(&event.User{
//					UserID:    clientID,
//					MintNonce: 1,
//				}).Error
//				require.NoError(t, err)
//
//				target := url.URL{Path: baseUrl}
//
//				query := target.Query()
//
//				query.Add("client_id", clientID)
//
//				target.RawQuery = query.Encode()
//
//				req := httptest.NewRequest(http.MethodGet, target.String(), nil)
//
//				respRaw, err := MintNonceHandler(context.Background(), req)
//				require.NoError(t, err)
//
//				resp, ok := respRaw.(int64)
//				require.True(t, ok)
//				require.Equal(t, int64(1), resp)
//
//				err = eventDb.Get().Model(&event.User{}).Where("user_id = ?", clientID).Delete(&event.User{}).Error
//				require.NoError(t, err)
//			},
//		},
//		{
//			name: "Get processed mint nonces not providing client id, should not work",
//			body: func(t *testing.T) {
//				target := url.URL{Path: baseUrl}
//
//				req := httptest.NewRequest(http.MethodGet, target.String(), nil)
//
//				resp, err := MintNonceHandler(context.Background(), req)
//				require.Error(t, err)
//				require.Nil(t, resp)
//			},
//		},
//		{
//			name: "Get processed mint nonces for the client, which does not exist, should not work",
//			body: func(t *testing.T) {
//				target := url.URL{Path: baseUrl}
//
//				query := target.Query()
//
//				query.Add("client_id", clientID)
//
//				target.RawQuery = query.Encode()
//
//				req := httptest.NewRequest(http.MethodGet, target.String(), nil)
//
//				respRaw, err := MintNonceHandler(context.Background(), req)
//				require.Error(t, err)
//				require.Nil(t, respRaw)
//			},
//		},
//	}
//
//	for _, test := range tests {
//		t.Run(test.name, test.body)
//	}
//}
//
//func TestNotProcessedBurnTicketsHandler(t *testing.T) {
//	const baseUrl = "/v1/not_processed_burn_tickets"
//
//	b := block.NewBlock("", 10)
//	b.Hash = encryption.Hash("data")
//
//	sc := makeTestChain(t)
//	sc.AddBlock(b)
//	sc.SetLatestFinalizedBlock(b)
//
//	cl := initDBs(t)
//	defer cl()
//
//	eventDb, err := event.NewInMemoryEventDb(config.DbAccess{}, config.DbSettings{
//		Debug:                 true,
//		PartitionChangePeriod: 1,
//	})
//	require.NoError(t, err)
//
//	t.Cleanup(func() {
//		err = eventDb.Drop()
//		require.NoError(t, err)
//
//		eventDb.Close()
//	})
//
//	sc.EventDb = eventDb
//
//	tests := []struct {
//		name string
//		body func(t *testing.T)
//	}{
//		{
//			name: "Get not processed burn tickets of the client, which hasn't performed any burn operations, should work",
//			body: func(t *testing.T) {
//				target := url.URL{Path: baseUrl}
//
//				query := target.Query()
//
//				query.Add("ethereum_address", ethereumAddress)
//				query.Add("nonce", "0")
//
//				target.RawQuery = query.Encode()
//
//				req := httptest.NewRequest(http.MethodGet, target.String(), nil)
//
//				respRaw, err := NotProcessedBurnTicketsHandler(context.Background(), req)
//				require.NoError(t, err)
//
//				resp, ok := respRaw.([]*state.BurnTicket)
//				require.True(t, ok)
//				require.Len(t, resp, 0)
//			},
//		},
//		{
//			name: "Get not processed burn tickets of the client, which has performed burn operation, should work",
//			body: func(t *testing.T) {
//				err := eventDb.Get().Model(&event.BurnTicket{}).Create(&event.BurnTicket{
//					EthereumAddress: ethereumAddress,
//					Hash:            hash,
//					Nonce:           1,
//				}).Error
//				require.NoError(t, err)
//
//				target := url.URL{Path: baseUrl}
//
//				query := target.Query()
//
//				query.Add("ethereum_address", ethereumAddress)
//				query.Add("nonce", "0")
//
//				target.RawQuery = query.Encode()
//
//				req := httptest.NewRequest(http.MethodGet, target.String(), nil)
//
//				respRaw, err := NotProcessedBurnTicketsHandler(context.Background(), req)
//				require.NoError(t, err)
//
//				resp, ok := respRaw.([]*state.BurnTicket)
//				require.True(t, ok)
//				require.Len(t, resp, 1)
//
//				err = eventDb.Get().Model(&event.BurnTicket{}).Where("ethereum_address = ?", ethereumAddress).Delete(&event.BurnTicket{}).Error
//				require.NoError(t, err)
//			},
//		},
//		{
//			name: "Get not processed burn tickets not providing ethereum address, should not work",
//			body: func(t *testing.T) {
//				target := url.URL{Path: baseUrl}
//
//				query := target.Query()
//
//				query.Add("nonce", "0")
//
//				target.RawQuery = query.Encode()
//
//				req := httptest.NewRequest(http.MethodGet, target.String(), nil)
//
//				resp, err := NotProcessedBurnTicketsHandler(context.Background(), req)
//				require.Error(t, err)
//				require.Nil(t, resp)
//			},
//		},
//		{
//			name: "Get not processed burn tickets not providing nonce, should work",
//			body: func(t *testing.T) {
//				err := eventDb.Get().Model(&event.BurnTicket{}).Create(&event.BurnTicket{
//					EthereumAddress: ethereumAddress,
//					Hash:            hash,
//					Nonce:           1,
//				}).Error
//				require.NoError(t, err)
//
//				target := url.URL{Path: baseUrl}
//
//				query := target.Query()
//
//				query.Add("ethereum_address", ethereumAddress)
//
//				target.RawQuery = query.Encode()
//
//				req := httptest.NewRequest(http.MethodGet, target.String(), nil)
//
//				respRaw, err := notProcessedBurnTicketsHandler(context.Background(), req)
//				require.NoError(t, err)
//
//				resp, ok := respRaw.([]*state.BurnTicket)
//				require.True(t, ok)
//				require.Len(t, resp, 1)
//
//				err = eventDb.Get().Model(&event.BurnTicket{}).Where("ethereum_address = ?", ethereumAddress).Delete(&event.BurnTicket{}).Error
//				require.NoError(t, err)
//			},
//		},
//	}
//
//	for _, test := range tests {
//		t.Run(test.name, test.body)
//	}
//}
