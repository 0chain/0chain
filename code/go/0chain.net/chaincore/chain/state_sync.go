package chain

import (
	"bytes"
	"context"
	"errors"
	"net/url"
	"time"

	"go.uber.org/zap"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/state"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"github.com/0chain/common/core/logging"
	"github.com/0chain/common/core/util"
)

var ErrNodeNull = common.NewError("node_null", "Node is not available")

var ErrStopIterator = common.NewError("stop_iterator", "Stop MPT Iteration")

var MaxStateNodesForSync = 10000

func (c *Chain) GetBlockStateChangeForce(ctx context.Context, b *block.Block) error {
	return common.RunWithRetries(ctx, 20, func() error {
		err := c.GetBlockStateChange(b)
		if err != nil {
			logging.Logger.Info("can't get block state changes, retrying", zap.Error(err))
		}
		return err
	})
}

// GetBlockStateChange - get the state change of the block from the network
func (c *Chain) GetBlockStateChange(b *block.Block) error {
	ts := time.Now()
	if b.PrevBlock != nil && bytes.Equal(b.PrevBlock.ClientStateHash, b.ClientStateHash) {
		logging.Logger.Debug("block has the same state", zap.Any("block", b.Hash),
			zap.Any("block_state_hash", b.ClientStateHash))
		if !b.PrevBlock.IsStateComputed() {
			return common.NewError("get_block_state_changes", "block is not changed but prev block state is not computed")
		}
		s := block.CreateStateWithPreviousBlock(b.PrevBlock, c.GetStateDB(), b.Round)
		b.SetClientState(s)
		b.SetStateStatus(b.PrevBlock.GetStateStatus())

		logging.Logger.Debug("get_block_state_changes - apply took",
			zap.Int64("round", b.Round),
			zap.Any("duration", time.Since(ts)))
		return nil
	}
	bsc, err := c.getBlockStateChange(b)
	if err != nil {
		return common.NewError("get block state changes", err.Error())
	}
	logging.Logger.Debug("get_block_state_changes - get took",
		zap.Int64("round", b.Round),
		zap.Any("duration", time.Since(ts)))

	ts = time.Now()
	err = c.ApplyBlockStateChange(b, bsc)
	if err != nil {
		return common.NewError("apply block state changes", err.Error())
	}

	logging.Logger.Debug("get_block_state_changes - apply took",
		zap.Int64("round", b.Round),
		zap.Any("duration", time.Since(ts)))
	return nil
}

// GetStateNodes - get a bunch of state nodes from the network
func (c *Chain) GetStateNodes(ctx context.Context, keys []util.Key) {
	ns, err := c.getStateNodes(ctx, keys)
	if err != nil {
		skeys := make([]string, len(keys))
		for idx, key := range keys {
			skeys[idx] = util.ToHex(key)
		}
		logging.Logger.Error("get state nodes", zap.Int("num_keys", len(keys)),
			zap.Any("keys", skeys), zap.Error(err))
		return
	}
	keysStr := make([]string, len(keys))
	for i := range keys {
		keysStr[i] = util.ToHex(keys[i])
	}
	err = c.SaveStateNodes(ctx, ns)
	if err != nil {
		logging.Logger.Error("get state nodes - error saving",
			zap.Int("num_keys", len(keys)),
			zap.Strings("keys:", keysStr),
			zap.Error(err))
	} else {
		logging.Logger.Info("get state nodes - saving",
			zap.Int("num_keys", len(keys)),
			zap.Strings("keys:", keysStr),
			zap.Int("nodes", len(ns.Nodes)))
	}
}

// UpdateStateFromNetwork get a bunch of state nodes from the network
func (c *Chain) UpdateStateFromNetwork(ctx context.Context, mpt util.MerklePatriciaTrieI, keys []util.Key) error {
	ns, err := c.getStateNodes(ctx, keys)
	if err != nil {
		return err
	}

	logging.Logger.Debug("UpdateStateFromNetwork get state nodes", zap.Int("num", len(ns.Nodes)))

	return ns.SaveState(ctx, mpt.GetNodeDB())
}

// GetStateNodesSharders - get a bunch of state nodes from the network
func (c *Chain) GetStateNodesFromSharders(ctx context.Context, keys []util.Key) {
	ns, err := c.getStateNodesFromSharders(ctx, keys)
	if err != nil {
		skeys := make([]string, len(keys))
		for idx, key := range keys {
			skeys[idx] = util.ToHex(key)
		}
		logging.Logger.Error("get state nodes", zap.Int("num_keys", len(keys)),
			zap.Any("keys", skeys), zap.Error(err))
		return
	}
	keysStr := make([]string, len(keys))
	for i := range keys {
		keysStr[i] = util.ToHex(keys[i])
	}
	err = c.SaveStateNodes(ctx, ns)
	if err != nil {
		logging.Logger.Error("get state nodes - error saving",
			zap.Int("num_keys", len(keys)),
			zap.Strings("keys:", keysStr),
			zap.Error(err))
	} else {
		logging.Logger.Info("get state nodes - saving",
			zap.Int("num_keys", len(keys)),
			zap.Strings("keys:", keysStr),
			zap.Int("nodes", len(ns.Nodes)))
	}
}

// GetStateFrom - get the state from a given node
func (c *Chain) GetStateFrom(ctx context.Context, key util.Key) (*state.PartialState, error) {
	partialState := &state.PartialState{}
	partialState.Hash = key
	handler := func(ctx context.Context, path util.Path, key util.Key, node util.Node) error {
		if node == nil {
			return ErrNodeNull
		}
		partialState.AddNode(node)
		if len(partialState.Nodes) >= MaxStateNodesForSync {
			return ErrStopIterator
		}
		return nil
	}
	err := c.GetLatestFinalizedBlock().ClientState.IterateFrom(ctx, key, handler, util.NodeTypeLeafNode|util.NodeTypeFullNode|util.NodeTypeExtensionNode)
	if err != nil {
		if err != ErrStopIterator {
			return nil, err
		}
	}
	if len(partialState.Nodes) > 0 {
		if err := partialState.ComputeProperties(); err != nil {
			return nil, err
		}
		return partialState, nil
	}
	return nil, util.ErrNodeNotFound
}

// GetStateNodesFrom - get the state nodes from db
func (c *Chain) GetStateNodesFrom(ctx context.Context, keys []util.Key) (*state.Nodes, error) {
	var stateNodes = state.NewStateNodes()
	nodes, err := c.stateDB.MultiGetNode(keys)
	if err != nil {
		if nodes == nil {
			return nil, err
		}
	}
	stateNodes.Nodes = nodes
	return stateNodes, nil
}

// SyncPartialState - sync partial state
func (c *Chain) SyncPartialState(ctx context.Context, ps *state.PartialState) error {
	if ps.GetRoot() == nil {
		return ErrNodeNull
	}
	return c.SavePartialState(ctx, ps)
}

// SavePartialState - save the partial state
func (c *Chain) SavePartialState(ctx context.Context, ps *state.PartialState) error {
	c.stateMutex.Lock()
	defer c.stateMutex.Unlock()
	return ps.SaveState(ctx, c.stateDB)
}

// SaveStateNodes - save the state nodes
func (c *Chain) SaveStateNodes(ctx context.Context, ns *state.Nodes) error {
	c.stateMutex.Lock()
	defer c.stateMutex.Unlock()
	return ns.SaveState(ctx, c.stateDB)
}

func (c *Chain) getStateNodes(ctx context.Context, keys []util.Key) (*state.Nodes, error) {
	nsRequestor := StateNodesRequestor
	params := &url.Values{}
	for _, key := range keys {
		params.Add("nodes", util.ToHex(key))
	}
	cctx, cancel := context.WithCancel(ctx)
	defer cancel()
	nodeStateC := make(chan *state.Nodes, 1)
	handler := func(_ context.Context, entity datastore.Entity) (interface{}, error) {
		rns, ok := entity.(*state.Nodes)
		if !ok {
			return nil, datastore.ErrInvalidEntity
		}
		if len(rns.Nodes) == 0 {
			return nil, util.ErrNodeNotFound
		}
		cancel()
		select {
		case nodeStateC <- rns:
		default:
		}
		return rns, nil
	}
	mb := c.GetCurrentMagicBlock()
	c.RequestEntityFromMinersOnMB(cctx, mb, nsRequestor, params, handler)
	var ns *state.Nodes
	select {
	case ns = <-nodeStateC:
	default:
	}

	if ns == nil {
		c.RequestEntityFromShardersOnMB(cctx, mb, nsRequestor, params, handler)
	}

	select {
	case ns = <-nodeStateC:
	default:
	}

	if ns == nil {
		return nil, common.NewError("state_nodes_error", "error getting the state nodes")
	}

	logging.Logger.Info("get state nodes",
		zap.Int("keys", len(keys)),
		zap.Int("nodes", len(ns.Nodes)))
	return ns, nil
}

func (c *Chain) getStateNodesFromSharders(ctx context.Context, keys []util.Key) (*state.Nodes, error) {
	nsRequestor := StateNodesRequestor
	params := &url.Values{}
	for _, key := range keys {
		params.Add("nodes", util.ToHex(key))
	}
	cctx, cancel := context.WithCancel(ctx)
	defer cancel()
	stateNodeC := make(chan *state.Nodes, 1)
	handler := func(_ context.Context, entity datastore.Entity) (interface{}, error) {
		rns, ok := entity.(*state.Nodes)
		if !ok {
			return nil, datastore.ErrInvalidEntity
		}
		if len(rns.Nodes) == 0 {
			return nil, util.ErrNodeNotFound
		}
		select {
		case stateNodeC <- rns:
		default:
		}
		cancel()
		return rns, nil
	}

	c.RequestEntityFromShardersOnMB(cctx, c.GetCurrentMagicBlock(), nsRequestor, params, handler)
	var ns *state.Nodes
	select {
	case ns = <-stateNodeC:
	default:
	}

	if ns == nil {
		return nil, common.NewError("state_nodes_error", "error getting the state nodes")
	}

	logging.Logger.Info("get state nodes", zap.Int("keys", len(keys)), zap.Int("nodes", len(ns.Nodes)))
	return ns, nil
}

func (c *Chain) getBlockStateChange(b *block.Block) (*block.StateChange, error) {
	cctx, cancel := context.WithCancel(common.GetRootContext())
	defer cancel()
	params := &url.Values{}
	params.Add("block", b.Hash)
	mb := c.GetLatestFinalizedMagicBlock(cctx)
	if mb == nil {
		return nil, errors.New("can't get mb")
	}
	stateChangesC := make(chan *block.StateChange, mb.Miners.Size())

	var handler = func(_ context.Context, entity datastore.Entity) (
		resp interface{}, err error) {

		var rsc, ok = entity.(*block.StateChange)
		if !ok {
			return nil, datastore.ErrInvalidEntity
		}

		if len(rsc.Nodes) != b.StateChangesCount {
			logging.Logger.Error("get_block_state_change",
				zap.Error(state.ErrPartialStateRootMismatch),
				zap.Int64("round", b.Round),
				zap.String("block", b.Hash))
			return nil, state.ErrMalformedPartialState
		}

		if rsc.Block != b.Hash {
			logging.Logger.Error("get_block_state_change",
				zap.Error(errors.New("block hash mismatch")),
				zap.Int64("round", b.Round),
				zap.String("block", b.Hash))
			return nil, block.ErrBlockHashMismatch
		}

		if !bytes.Equal(b.ClientStateHash, rsc.Hash) {
			logging.Logger.Error("get_block_state_change",
				zap.Error(errors.New("state hash mismatch")),
				zap.Int64("round", b.Round),
				zap.String("block", b.Hash))
			return nil, block.ErrBlockStateHashMismatch
		}

		var root = rsc.GetRoot()
		if root == nil {
			logging.Logger.Error("get_block_state_change",
				zap.Error(errors.New("state root error")),
				zap.Int64("round", b.Round),
				zap.String("block", b.Hash),
				zap.Int("state_nodes", len(rsc.Nodes)))
			return nil, common.NewError("state_root_error",
				"block state root calculation error")
		}

		cancel()
		select {
		case stateChangesC <- rsc:
		default:
		}
		return rsc, nil
	}

	c.RequestEntityFromMiners(cctx, BlockStateChangeRequestor, params, handler)
	var bsc *block.StateChange
	select {
	case bsc = <-stateChangesC:
	default:
	}
	if bsc == nil {
		c.RequestEntityFromSharders(cctx, BlockStateChangeRequestor, params, handler)
		select {
		case bsc = <-stateChangesC:
		default:
		}
		if bsc == nil {
			logging.Logger.Error("get_block_state_change - could not get state changes from remote",
				zap.Int64("round", b.Round),
				zap.String("block", b.Hash))
			return nil, common.NewError("block_state_change_error",
				"error getting the block state change")
		}
	}

	logging.Logger.Debug("get_block_state_change - success with root",
		zap.Int64("round", b.Round),
		zap.String("bsc root", bsc.GetRoot().GetHash()),
		zap.String("block state hash", util.ToHex(b.ClientStateHash)))
	return bsc, nil
}

// ApplyBlockStateChange - applies the state chagnes to the block state.
func (c *Chain) ApplyBlockStateChange(b *block.Block, bsc *block.StateChange) error {
	return b.ApplyBlockStateChange(bsc, c)
}
