package chain

import (
	"context"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"

	"go.uber.org/zap"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/node"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"github.com/0chain/common/core/logging"
	"github.com/0chain/common/core/util"
)

/*SetupNodeHandlers - setup the handlers for the chain */
func (c *Chain) SetupMinerNodeHandlers() {
	http.HandleFunc("/_nh/list/m", common.Recover(c.GetMinersHandler))
	http.HandleFunc("/v1/scstats/", common.WithCORS(common.UserRateLimit(c.GetSCStats)))
}

func (c *Chain) SetupSharderNodeHandlers() {
	http.HandleFunc("/_nh/list/s", common.Recover(c.GetShardersHandler))
}

var (
	// MinerNotarizedBlockRequestor - reuqest a notarized block from a node.
	MinerNotarizedBlockRequestor node.EntityRequestor
	//BlockStateChangeRequestor - request state changes for the block.
	BlockStateChangeRequestor node.EntityRequestor

	// disables (doesn't work, sharders doesn't give changes)
	//
	// ShardersBlockStateChangeRequestor is the same, but from sharders.
	// ShardersBlockStateChangeRequestor node.EntityRequestor

	// StateNodesRequestor - request a set of state nodes given their keys.
	StateNodesRequestor node.EntityRequestor
	// LatestFinalizedMagicBlockRequestor - RequestHandler for latest finalized
	// magic block to a node.
	LatestFinalizedMagicBlockRequestor node.EntityRequestor

	// FBRequestor represents FB from sharders reqeustor.
	FBRequestor node.EntityRequestor
	// MinerLatestFinalizedBlockRequestor - RequestHandler for latest finalized
	// block to a node.
	MinerLatestFinalizedBlockRequestor node.EntityRequestor
)

// setupX2MRequestors - setup requestors */
func setupX2MRequestors() {
	options := &node.SendOptions{Timeout: node.TimeoutLargeMessage, CODEC: node.CODEC_MSGPACK, Compress: true}

	blockEntityMetadata := datastore.GetEntityMetadata("block")
	MinerNotarizedBlockRequestor = node.RequestEntityHandler("/v1/_x2m/block/notarized_block/get", options, blockEntityMetadata)

	options = &node.SendOptions{Timeout: node.TimeoutLargeMessage, CODEC: node.CODEC_MSGPACK, Compress: true}
	blockStateChangeEntityMetadata := datastore.GetEntityMetadata("block_state_change")
	BlockStateChangeRequestor = node.RequestEntityHandler("/v1/_x2x/block/state_change/get", options, blockStateChangeEntityMetadata)
	// ShardersBlockStateChangeRequestor = node.RequestEntityHandler("/v1/_x2s/block/state_change/get", options, blockStateChangeEntityMetadata)

	stateNodesEntityMetadata := datastore.GetEntityMetadata("state_nodes")
	StateNodesRequestor = node.RequestEntityHandler("/v1/_x2x/state/get_nodes", options, stateNodesEntityMetadata)
}

func setupX2SRequestors() {
	blockEntityMetadata := datastore.GetEntityMetadata("block")
	options := &node.SendOptions{Timeout: node.TimeoutLargeMessage, MaxRelayLength: 0, CurrentRelayLength: 0, Compress: false}
	LatestFinalizedMagicBlockRequestor = node.RequestEntityHandler("/v1/block/get/latest_finalized_magic_block", options, blockEntityMetadata)

	var opts = node.SendOptions{
		Timeout:  node.TimeoutLargeMessage,
		CODEC:    node.CODEC_MSGPACK,
		Compress: true,
	}
	FBRequestor = node.RequestEntityHandler("/v1/_x2s/block/get", &opts,
		datastore.GetEntityMetadata("block"))

	options = &node.SendOptions{Timeout: node.TimeoutLargeMessage, CODEC: node.CODEC_MSGPACK, Compress: true}
	// Though it is `_m2s`, but it can also be called by sharder for sharders to get latest finalized block
	// this is to make it backward compatible
	MinerLatestFinalizedBlockRequestor = node.RequestEntityHandler("/v1/_m2s/block/latest_finalized/get", options, blockEntityMetadata)
}

func SetupX2XResponders(c *Chain) {
	middleHandlers := func(h common.JSONResponderF) common.ReqRespHandlerf {
		return common.N2NRateLimit(node.ToN2NSendEntityHandler(h))
	}
	http.HandleFunc("/v1/_x2x/state/get_nodes", middleHandlers(StateNodesHandler))
	http.HandleFunc("/v1/_x2x/block/state_change/get", middleHandlers(c.BlockStateChangeHandler))
}

// StateNodesHandler - return a list of state nodes
func StateNodesHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	// this is needed as we get multiple values for the same key
	if err := r.ParseForm(); err != nil {
		return nil, err
	}

	nodes := r.Form["nodes"]
	c := GetServerChain()
	keys := make([]util.Key, len(nodes))
	for idx, nd := range nodes {
		key, err := hex.DecodeString(nd)
		if err != nil {
			return nil, err
		}
		keys[idx] = key
	}
	ns, err := c.GetStateNodesFrom(ctx, keys)
	if err != nil {
		if ns != nil {
			logging.Logger.Error("state nodes handler", zap.Int("keys", len(nodes)), zap.Int("found_keys", len(ns.Nodes)), zap.Error(err))
			return ns, nil
		}

		logging.Logger.Error("state nodes handler",
			zap.Int("keys", len(nodes)),
			zap.Int64("current round", c.GetCurrentRound()),
			zap.Error(err))

		return nil, err
	}
	logging.Logger.Info("state nodes handler", zap.Int("keys", len(keys)), zap.Int("nodes", len(ns.Nodes)))
	return ns, nil
}

// blockStateChangeHandler - provide the state changes associated with a block.
func (c *Chain) blockStateChangeHandler(ctx context.Context, r *http.Request) (*block.StateChange, error) {
	var b, err = c.getNotarizedBlock(ctx, r.FormValue("round"), r.FormValue("block"))
	if err != nil {
		return nil, err
	}

	if b.GetStateStatus() != block.StateSuccessful && b.GetStateStatus() != block.StateSynched {
		return nil, common.NewError("state_not_verified",
			"state is not computed and validated locally")
	}

	bsc, err := block.NewBlockStateChange(b)
	if err != nil {
		logging.Logger.Error("block state change handler",
			zap.Int64("round", b.Round),
			zap.String("block", b.Hash),
			zap.Error(err))
		return nil, err
	}

	return bsc, nil
}

func (c *Chain) getNotarizedBlock(ctx context.Context, roundStr, blockHash string) (*block.Block, error) {
	var (
		cr = c.GetCurrentRound()
	)

	errBlockNotAvailable := common.NewError("block_not_available",
		fmt.Sprintf("Requested block is not available, current round: %d, request round: %s, request hash: %s",
			cr, roundStr, blockHash))

	if blockHash != "" {
		b, err := c.GetBlock(ctx, blockHash)
		if err != nil {
			return nil, err
		}

		if b.IsBlockNotarized() {
			return b, nil
		}
		logging.Logger.Debug("requested block is not notarized yet")
		return nil, errBlockNotAvailable
	}

	if roundStr == "" {
		return nil, common.NewError("none_round_or_hash_provided",
			"no block hash or round number is provided")
	}

	roundN, err := strconv.ParseInt(roundStr, 10, 64)
	if err != nil {
		return nil, err
	}

	rd := c.GetRound(roundN)
	if rd == nil {
		return nil, errBlockNotAvailable
	}

	b := rd.GetHeaviestNotarizedBlock()
	if b == nil {
		return nil, errBlockNotAvailable
	}

	return b, nil
}
