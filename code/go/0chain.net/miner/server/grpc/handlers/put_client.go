package handlers

import (
	"context"
	"encoding/json"

	"0chain.net/chaincore/client"
	"0chain.net/core/cache"
	"0chain.net/core/datastore"
	minerproto "0chain.net/miner/proto/api/src/proto"
	"google.golang.org/genproto/googleapis/api/httpbody"
)

//  cache
var cacher cache.Cache = cache.NewLFUCache(10 * 1024)

// PutClient
func (m *minerGRPCService) PutClient(ctx context.Context, req *minerproto.PutClientRequest) (*httpbody.HttpBody, error) {
	c := client.NewClient()
	c.SetKey(req.ClientId)
	c.SetPublicKey(req.PublicKey)

	response, err := datastore.PutEntityHandler(ctx, c)
	if err != nil {
		return nil, err
	}

	output, err := json.Marshal(response)
	if err != nil {
		return nil, err
	}

	cacher.Add(c.GetKey(), c)
	return &httpbody.HttpBody{
		ContentType: "text/html;charset=UTF-8",
		Data:        output,
	}, nil
}
