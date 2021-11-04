package handlers

import (
	"context"

	"0chain.net/core/datastore"
	minerproto "0chain.net/miner/proto/api/src/proto"
	"google.golang.org/genproto/googleapis/api/httpbody"
)

func GetClient(ctx context.Context, req *minerproto.GetClientRequest) (*httpbody.HttpBody, error) {

	response, err := datastore.GetEntityHandler(ctx, req.Id, datastore.GetEntityMetadata("client"))

	if err != nil {
		return nil, err
	}

	return &httpbody.HttpBody{
		// Done by datastore.ToJSONEntityReqResponse middelware. Need to find workaround.
		Data: []byte{response},
	}, nil
}
