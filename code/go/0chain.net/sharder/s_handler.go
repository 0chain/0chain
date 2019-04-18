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
	"0chain.net/core/ememorystore"
	. "0chain.net/core/logging"
	"go.uber.org/zap"
)

var LatestRoundRequestor node.EntityRequestor
var RoundRequestor node.EntityRequestor
var RoundSummariesRequestor node.EntityRequestor
var BlockRequestor node.EntityRequestor
var BlockSummaryRequestor node.EntityRequestor
var BlockSummariesRequestor node.EntityRequestor

func SetupS2SRequestors() {
	options := &node.SendOptions{Timeout: node.TimeoutLargeMessage, CODEC: node.CODEC_MSGPACK, Compress: true}
	roundEntityMetadata := datastore.GetEntityMetadata("round")
	LatestRoundRequestor = node.RequestEntityHandler("/v1/_s2s/latest_round/get", options, roundEntityMetadata)

	RoundRequestor = node.RequestEntityHandler("/v1/_s2s/round/get", options, roundEntityMetadata)

	blockEntityMetadata := datastore.GetEntityMetadata("block")
	BlockRequestor = node.RequestEntityHandler("/v1/_s2s/block/get", options, blockEntityMetadata)

	blockSummaryEntityMetadata := datastore.GetEntityMetadata("block_summary")
	BlockSummaryRequestor = node.RequestEntityHandler("/v1/_s2s/blocksummary/get", options, blockSummaryEntityMetadata)

	options = &node.SendOptions{Timeout: node.TimeoutLargeMessage, CODEC: node.CODEC_JSON, Compress: true}
	roundSummariesEntityMetadata := datastore.GetEntityMetadata("round_summaries")
	RoundSummariesRequestor = node.RequestEntityHandler("/v1/_s2s/roundsummaries/get", options, roundSummariesEntityMetadata)

	blockSummariesEntityMetadata := datastore.GetEntityMetadata("block_summaries")
	BlockSummariesRequestor = node.RequestEntityHandler("/v1/_s2s/blocksummaries/get", options, blockSummariesEntityMetadata)
}

func SetupS2SResponders() {
	http.HandleFunc("/v1/_s2s/latest_round/get", node.ToN2NSendEntityHandler(LatestRoundRequestHandler))
	http.HandleFunc("/v1/_s2s/round/get", node.ToN2NSendEntityHandler(RoundRequestHandler))
	http.HandleFunc("/v1/_s2s/roundsummaries/get", node.ToN2NSendEntityHandler(RoundSummariesHandler))
	http.HandleFunc("/v1/_s2s/block/get", node.ToN2NSendEntityHandler(RoundBlockRequestHandler))
	http.HandleFunc("/v1/_s2s/blocksummary/get", node.ToN2NSendEntityHandler(BlockSummaryRequestHandler))
	http.HandleFunc("/v1/_s2s/blocksummaries/get", node.ToN2NSendEntityHandler(BlockSummariesHandler))
}

func RoundSummariesHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	sc := GetSharderChain()
	roundValue := r.FormValue("round")
	roundRange := r.FormValue("range")
	roundNum, err := strconv.ParseInt(roundValue, 10, 64)
	var rRange int
	rRange, err = strconv.Atoi(roundRange)
	if err == nil {
		beginR := roundNum
		if rRange < 0 {
			rRange = -rRange
			beginR = roundNum - int64(rRange)
			if beginR < 1 {
				beginR = 1
			}
		}
		roundS := sc.getRoundSummaries(beginR, rRange)
		Logger.Info("fetched round summaries", zap.Int64("beginR", beginR), zap.Int("range", rRange))
		rs := &RoundSummaries{}
		rs.RSummaryList = roundS
		return rs, nil
	}
	return nil, err
}

func BlockSummariesHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	sc := GetSharderChain()
	roundValue := r.FormValue("round")
	roundRange := r.FormValue("range")
	roundNum, err := strconv.ParseInt(roundValue, 10, 64)
	var rRange int
	rRange, err = strconv.Atoi(roundRange)
	if err == nil {
		beginR := roundNum
		if rRange < 0 {
			rRange = -rRange
			beginR = roundNum - int64(rRange)
			if beginR < 1 {
				beginR = 1
			}
		}
		rs := sc.getRoundSummaries(beginR, rRange)
		bs := &BlockSummaries{}
		blockS := make([]*block.BlockSummary, rRange)
		for i, roundS := range rs {
			if roundS != nil {
				bSummaryEntityMetadata := datastore.GetEntityMetadata("block_summary")
				bctx := ememorystore.WithEntityConnection(ctx, bSummaryEntityMetadata)
				defer ememorystore.Close(bctx)
				blockS[i], _ = sc.GetBlockSummary(bctx, roundS.BlockHash)
			}
		}
		bs.BSummaryList = blockS
		Logger.Info("fetched block summaries", zap.Int64("beginR", beginR), zap.Int("range", rRange))
		return bs, nil
	}
	Logger.Error("failed reading/parsing the params", zap.Error(err))
	return nil, err
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

func BlockSummaryRequestHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	sc := GetSharderChain()
	bHash := r.FormValue("hash")
	if bHash != "" {
		bSummaryEntityMetadata := datastore.GetEntityMetadata("block_summary")
		bctx := ememorystore.WithEntityConnection(ctx, bSummaryEntityMetadata)
		defer ememorystore.Close(bctx)
		blockS, err := sc.GetBlockSummary(bctx, bHash)
		if err == nil {
			return blockS, nil
		}
		return nil, err
	}
	return nil, common.InvalidRequest("block hash is required")
}

func RoundBlockRequestHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	sc := GetSharderChain()
	hash := r.FormValue("hash")
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

func (sc *Chain) getRoundSummaries(beginR int64, rRange int) []*round.Round {
	roundS := make([]*round.Round, rRange)
	for loopR := 0; loopR < rRange; loopR++ {
		roundS[loopR] = sc.GetSharderRound(beginR + int64(loopR))
	}
	return roundS
}
