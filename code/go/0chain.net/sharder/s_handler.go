package sharder

import (
	"context"
	"net/http"
	"strconv"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
)

var LatestRoundRequestor node.EntityRequestor
var RoundRequestor node.EntityRequestor
var BlockRequestor node.EntityRequestor

func SetupS2SRequestors() {
	options := &node.SendOptions{Timeout: node.TimeoutLargeMessage, CODEC: node.CODEC_MSGPACK, Compress: true}

	roundEntityMetadata := datastore.GetEntityMetadata("round")
	LatestRoundRequestor = node.RequestEntityHandler("/v1/_s2s/latest_round/get", options, roundEntityMetadata)

	RoundRequestor = node.RequestEntityHandler("/v1/_s2s/round/get", options, roundEntityMetadata)

	blockEntityMetadata := datastore.GetEntityMetadata("block")
	BlockRequestor = node.RequestEntityHandler("/v1/_s2s/block/get", options, blockEntityMetadata)
}

func SetupS2SResponders() {
	http.HandleFunc("/v1/_s2s/latest_round/get", node.ToN2NSendEntityHandler(LatestRoundRequestHandler))
	http.HandleFunc("/v1/_s2s/round/get", node.ToN2NSendEntityHandler(RoundRequestHandler))
	http.HandleFunc("/v1/_s2s/block/get", node.ToN2NSendEntityHandler(RoundBlockRequestHandler))
}

func LatestRoundRequestHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	sc := GetSharderChain()
	currRound := sc.GetRound(sc.CurrentRound)
	if currRound != nil {
		lr := currRound.(*round.Round)
		return lr, nil
	}
	return nil, common.NewError("no_round_info", "cannot retrieve the round info")
}

func RoundRequestHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	sc := GetSharderChain()
	roundValue := r.FormValue("round")
	roundNum, err := strconv.ParseInt(roundValue, 10, 64)
	if err == nil {
		roundEntity := sc.GetSharderRound(roundNum)
		if roundEntity == nil {
			var err error
			roundEntity, err = sc.GetRoundFromStore(ctx, roundNum)
			if err == nil {
				return roundEntity, nil
			}
			return nil, err
		}
		return roundEntity, nil
	}
	return nil, err
}

func RoundBlockRequestHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	sc := GetSharderChain()
	hash := r.FormValue("block")
	var b *block.Block
	var roundNumber int64
	if hash == "" {
		return nil, common.InvalidRequest("block hash is required")
	}
	b, err := sc.GetBlock(ctx, hash)
	if err == nil {
		return b, nil
	}
	roundNumber, err = strconv.ParseInt(r.FormValue("round"), 10, 64)
	if err == nil {
		b, err = sc.GetBlockFromStore(hash, roundNumber)
		if err == nil {
			return b, nil
		}
	}
	return nil, err
}
