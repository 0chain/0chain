package sharder

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"go.uber.org/zap"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/ememorystore"
	. "0chain.net/core/logging"
)

var (
	// RoundRequestor -
	RoundRequestor node.EntityRequestor
	// RoundSummariesRequestor -
	RoundSummariesRequestor node.EntityRequestor
	// BlockRequestor -
	BlockRequestor node.EntityRequestor
	// BlockSummaryRequestor -
	BlockSummaryRequestor node.EntityRequestor
	// BlockSummariesRequestor -
	BlockSummariesRequestor node.EntityRequestor
)

// SetupS2SRequestors -
func SetupS2SRequestors() {
	options := &node.SendOptions{Timeout: node.TimeoutLargeMessage, CODEC: node.CODEC_MSGPACK, Compress: true}
	roundEntityMetadata := datastore.GetEntityMetadata("round")

	RoundRequestor = node.RequestEntityHandler("/v1/_s2s/round/get", options, roundEntityMetadata)

	blockEntityMetadata := datastore.GetEntityMetadata("block")
	BlockRequestor = node.RequestEntityHandler("/v1/_s2s/block/get", options, blockEntityMetadata)

	blockSummaryEntityMetadata := datastore.GetEntityMetadata("block_summary")
	BlockSummaryRequestor = node.RequestEntityHandler("/v1/_s2s/blocksummary/get", options, blockSummaryEntityMetadata)

	options = &node.SendOptions{Timeout: node.TimeoutLargeMessage, CODEC: node.CODEC_MSGPACK, Compress: true}
	roundSummariesEntityMetadata := datastore.GetEntityMetadata("round_summaries")
	RoundSummariesRequestor = node.RequestEntityHandler("/v1/_s2s/roundsummaries/get", options, roundSummariesEntityMetadata)

	blockSummariesEntityMetadata := datastore.GetEntityMetadata("block_summaries")
	BlockSummariesRequestor = node.RequestEntityHandler("/v1/_s2s/blocksummaries/get", options, blockSummariesEntityMetadata)
}

// SetupS2SResponders -
func SetupS2SResponders() {
	http.HandleFunc("/v1/_s2s/latest_round/get", node.ToN2NSendEntityHandler(LatestRoundRequestHandler))
	http.HandleFunc("/v1/_s2s/round/get", node.ToN2NSendEntityHandler(RoundRequestHandler))
	http.HandleFunc("/v1/_s2s/roundsummaries/get", node.ToN2NSendEntityHandler(RoundSummariesHandler))
	http.HandleFunc("/v1/_s2s/block/get", node.ToN2NSendEntityHandler(RoundBlockRequestHandler))
	http.HandleFunc("/v1/_s2s/blocksummary/get", node.ToN2NSendEntityHandler(BlockSummaryRequestHandler))
	http.HandleFunc("/v1/_s2s/blocksummaries/get", node.ToN2NSendEntityHandler(BlockSummariesHandler))
}

const (
	getBlockX2SV1Pattern = "/v1/_x2s/block/get"
)

func x2sRespondersMap() map[string]func(http.ResponseWriter, *http.Request) {
	return map[string]func(http.ResponseWriter, *http.Request){
		getBlockX2SV1Pattern: node.ToN2NSendEntityHandler(
			// BlockRequestHandler - used by nodes to get missing FB by received LFB
			// ticket from sharder sent the ticket.
			RoundBlockRequestHandler,
		),
	}
}

type wrappedResponseWriter struct {
	http.ResponseWriter
	statusCode int
	response   bytes.Buffer
}

func newWrappedResponseWriter(w http.ResponseWriter) *wrappedResponseWriter {
	return &wrappedResponseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
		response:       bytes.Buffer{},
	}
}

func (lrw *wrappedResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

func (lrw *wrappedResponseWriter) Write(buff []byte) (int, error) {
	retVal, err := lrw.ResponseWriter.Write(buff)
	if lrw.statusCode >= 400 {
		lrw.response.Write(buff)
	}
	return retVal, err
}

func elapsedHandler(handler func(http.ResponseWriter, *http.Request)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		lrw := newWrappedResponseWriter(w)
		start := time.Now()
		handler(lrw, r)

		if lrw.statusCode != http.StatusTooManyRequests {
			Logger.Debug("API",
				zap.String("src", r.RemoteAddr),
				zap.Int("status", lrw.statusCode),
				zap.String("method", r.Method),
				zap.String("url", r.URL.Path),
				zap.Any("time", time.Since(start)))
		}
	}
}

func setupHandlers(handlers map[string]func(http.ResponseWriter, *http.Request)) {
	for pattern, handler := range handlers {
		http.HandleFunc(pattern, elapsedHandler(handler))
	}
}

// RoundSummariesHandler -
func RoundSummariesHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	sc := GetSharderChain()

	var roundRange int64
	var err error
	roundEdgeValue := r.FormValue("round")
	roundRangeValue := r.FormValue("range")
	roundEdge, err := strconv.ParseInt(roundEdgeValue, 10, 64)
	if err == nil {
		roundRange, err = strconv.ParseInt(roundRangeValue, 10, 64)
	}
	if err == nil {
		rangeBounds := GetRangeBounds(roundEdge, roundRange)
		roundS := sc.getRoundSummaries(ctx, rangeBounds)
		Logger.Info("RoundSummariesHandler",
			zap.String("object", "roundSummaries"),
			zap.Int64("low", rangeBounds.roundLow),
			zap.Int64("high", rangeBounds.roundHigh),
			zap.Int64("range", rangeBounds.roundRange))
		rs := &RoundSummaries{}
		rs.RSummaryList = roundS
		return rs, nil
	}
	Logger.Error("RoundSummariesHandler - Parsing Param Error",
		zap.String("round", roundEdgeValue),
		zap.String("range", roundRangeValue),
		zap.Error(err))
	return nil, err
}

// BlockSummariesHandler -
func BlockSummariesHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	sc := GetSharderChain()
	var roundRange int64
	var err error
	roundEdgeValue := r.FormValue("round")
	roundRangeValue := r.FormValue("range")
	roundEdge, err := strconv.ParseInt(roundEdgeValue, 10, 64)
	if err == nil {
		roundRange, err = strconv.ParseInt(roundRangeValue, 10, 64)
	}
	if err == nil {
		rangeBounds := GetRangeBounds(roundEdge, roundRange)
		rs := sc.getRoundSummaries(ctx, rangeBounds)
		Logger.Info("BlockSummariesHandler",
			zap.String("object", "roundSummaries"),
			zap.Int64("low", rangeBounds.roundLow),
			zap.Int64("high", rangeBounds.roundHigh),
			zap.Int64("range", rangeBounds.roundRange))

		bs := &BlockSummaries{}
		blockS := make([]*block.BlockSummary, len(rs))

		// Get block summary connection.
		bSummaryEntityMetadata := datastore.GetEntityMetadata("block_summary")
		bctx := ememorystore.WithEntityConnection(ctx, bSummaryEntityMetadata)
		defer ememorystore.Close(bctx)

		for i, roundS := range rs {
			if roundS != nil {
				blockS[i], _ = sc.GetBlockSummary(bctx, roundS.BlockHash)
			} else {
				blockS[i] = nil
			}
		}
		bs.BSummaryList = blockS
		Logger.Info("BlockSummariesHandler",
			zap.String("object", "blockSummaries"),
			zap.Int64("low", rangeBounds.roundLow),
			zap.Int64("high", rangeBounds.roundHigh),
			zap.Int64("range", rangeBounds.roundRange))
		return bs, nil
	}
	Logger.Error("BlockSummariesHandler - Parsing Param Error",
		zap.String("round", roundEdgeValue),
		zap.String("range", roundRangeValue),
		zap.Error(err))
	return nil, err

}

// LatestRoundRequestHandler - returns latest finalized round info.
func LatestRoundRequestHandler(_ context.Context, _ *http.Request) (
	resp interface{}, err error) {
	var (
		sc = GetSharderChain()
		cr = sc.GetRound(sc.GetCurrentRound())
	)
	if cr == nil {
		return nil, common.NewError("no_round_info",
			"cannot retrieve the round info")
	}
	return cr, nil
}

// RoundRequestHandler -
func RoundRequestHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	sc := GetSharderChain()
	roundValue := r.FormValue("round")
	roundNum, err := strconv.ParseInt(roundValue, 10, 64)
	if err == nil {
		Logger.Debug("RoundRequestHandler",
			zap.String("object", "round"),
			zap.Int64("round", roundNum))
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

// BlockSummaryRequestHandler -
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

// roundBlockRequestHandler -
func roundBlockRequestHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	sc := GetSharderChain()
	hash := r.FormValue("hash")
	if hash != "" {
		b, err := sc.GetBlock(ctx, hash)
		if err == nil {
			return b, nil
		}
	}

	rs := r.FormValue("round")
	if rs == "" {
		return nil, errors.New("round number is missing")
	}

	roundNum, err := strconv.ParseInt(rs, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid round number: %v, err: %v", rs, err)
	}

	if hash == "" {
		hash, err = sc.GetBlockHash(ctx, roundNum)
		if err != nil {
			return nil, err
		}
	}

	return sc.GetBlockFromStore(hash, roundNum)
}

func (sc *Chain) getRoundSummaries(ctx context.Context, bounds RangeBounds) []*round.Round {
	roundS := make([]*round.Round, bounds.roundRange+1)
	loop := 0
	for index := bounds.roundLow; index <= bounds.roundHigh; index++ {
		roundEntity := sc.GetSharderRound(index)
		if roundEntity == nil {
			// Try from the store
			roundEntity, _ = sc.GetRoundFromStore(ctx, index)
		}
		roundS[loop] = roundEntity
		loop++
	}
	return roundS
}
