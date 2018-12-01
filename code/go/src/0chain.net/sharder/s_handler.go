package sharder

import (
	"net/http"

	"0chain.net/datastore"
	"0chain.net/node"
)

var BlockRequestor node.EntityRequestor

func SetupS2SRequestors() {
	options := &node.SendOptions{Timeout: node.TimeoutLargeMessage, CODEC: node.CODEC_MSGPACK, Compress: true}

	blockEntityMetadata := datastore.GetEntityMetadata("block")
	BlockRequestor = node.RequestEntityHandler("/v1/_s2s/block/get", options, blockEntityMetadata)
}

func SetS2SResponders() {
	http.HandleFunc("/v1/_s2s/block/get", node.ToN2NSendEntityHandler(BlockHandler))
}
