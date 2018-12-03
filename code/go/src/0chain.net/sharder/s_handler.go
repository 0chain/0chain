package sharder

import (
	"context"
	"net/http"

	"0chain.net/datastore"
	"0chain.net/node"
	"0chain.net/round"
)

var BlockRequestor node.EntityRequestor
var LatestRoundRequestor node.EntityRequestor

func SetupS2SRequestors() {
	options := &node.SendOptions{Timeout: node.TimeoutLargeMessage, CODEC: node.CODEC_MSGPACK, Compress: true}

	blockEntityMetadata := datastore.GetEntityMetadata("block")
	BlockRequestor = node.RequestEntityHandler("/v1/_s2s/block/get", options, blockEntityMetadata)

	roundEntityMetadata := datastore.GetEntityMetadata("round")
	LatestRoundRequestor = node.RequestEntityHandler("/v1/_s2s/latest_round/get", options, roundEntityMetadata)
}

func SetS2SResponders() {
	http.HandleFunc("/v1/_s2s/block/get", node.ToN2NSendEntityHandler(BlockHandler))
	http.HandleFunc("/v1/_s2s/latest_round/get", node.ToN2NSendEntityHandler(LatestRoundRequestHandler))
}

func LatestRoundRequestHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	sc := GetSharderChain()
	currRound := sc.GetRound(sc.CurrentRound)
	if currRound != nil {
		lr := currRound.(*round.Round)
		return lr, nil
	}
	return nil, nil
}
