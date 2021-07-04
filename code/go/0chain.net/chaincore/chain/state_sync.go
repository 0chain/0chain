package chain

import (
	"bytes"
	"context"
	"errors"
	"net/url"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/state"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/logging"
	"0chain.net/core/util"
	"go.uber.org/zap"
)

var ErrNodeNull = common.NewError("node_null", "Node is not available")

var ErrStopIterator = common.NewError("stop_iterator", "Stop MPT Iteration")

var MaxStateNodesForSync = 10000

//GetBlockStateChange - get the state change of the block from the network
func (c *Chain) GetBlockStateChange(b *block.Block) error {
	bsc, err := c.getBlockStateChange(b)
	if err != nil {
		return common.NewError("get block state changes", err.Error())
	}

	err = c.ApplyBlockStateChange(b, bsc)
	if err != nil {
		return common.NewError("apply block state changes", err.Error())
	}

	return nil
}

//GetPartialState - get the partial state from the network
func (c *Chain) GetPartialState(ctx context.Context, key util.Key) {
	ps, err := c.getPartialState(ctx, key)
	if err != nil {
		logging.Logger.Error("get partial state - no ps", zap.String("key", util.ToHex(key)), zap.Error(err))
		return
	}
	err = c.SavePartialState(ctx, ps)
	if err != nil {
		logging.Logger.Error("get partial state - error saving", zap.String("key", util.ToHex(key)), zap.Error(err))
	} else {
		logging.Logger.Info("get partial state - saving", zap.String("key", util.ToHex(key)), zap.Int("nodes", len(ps.Nodes)))
	}
}

//GetStateNodes - get a bunch of state nodes from the network
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
	return
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

//GetStateNodesSharders - get a bunch of state nodes from the network
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
	return
}

//GetStateFrom - get the state from a given node
func (c *Chain) GetStateFrom(ctx context.Context, key util.Key) (*state.PartialState, error) {
	var partialState = state.NewPartialState(key)
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
		partialState.ComputeProperties()
		return partialState, nil
	}
	return nil, util.ErrNodeNotFound
}

//GetStateNodesFrom - get the state nodes from db
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

//SyncPartialState - sync partial state
func (c *Chain) SyncPartialState(ctx context.Context, ps *state.PartialState) error {
	if ps.GetRoot() == nil {
		return ErrNodeNull
	}
	c.SavePartialState(ctx, ps)
	return nil
}

//SavePartialState - save the partial state
func (c *Chain) SavePartialState(ctx context.Context, ps *state.PartialState) error {
	c.stateMutex.Lock()
	defer c.stateMutex.Unlock()
	return ps.SaveState(ctx, c.stateDB)
}

//SaveStateNodes - save the state nodes
func (c *Chain) SaveStateNodes(ctx context.Context, ns *state.Nodes) error {
	c.stateMutex.Lock()
	defer c.stateMutex.Unlock()
	return ns.SaveState(ctx, c.stateDB)
}

func (c *Chain) getPartialState(ctx context.Context, key util.Key) (*state.PartialState, error) {
	psRequestor := PartialStateRequestor
	params := &url.Values{}
	params.Add("node", util.ToHex(key))
	ctx, cancelf := context.WithCancel(common.GetRootContext())
	var ps *state.PartialState
	handler := func(ctx context.Context, entity datastore.Entity) (interface{}, error) {
		logging.Logger.Debug("get partial state", zap.String("ps_id", entity.GetKey()))
		rps, ok := entity.(*state.PartialState)
		if !ok {
			return nil, datastore.ErrInvalidEntity
		}
		logging.Logger.Info("get partial state", zap.String("key", util.ToHex(key)), zap.Int("nodes", len(rps.Nodes)))
		if bytes.Compare(key, rps.Hash) != 0 {
			logging.Logger.Error("get partial state - state hash mismatch error", zap.String("key", util.ToHex(key)), zap.Any("hash", util.ToHex(ps.Hash)))
			return nil, state.ErrHashMismatch
		}
		root := rps.GetRoot()
		if root == nil {
			logging.Logger.Error("get partial state - state root error", zap.Int("state_nodes", len(ps.Nodes)))
			return nil, common.NewError("state_root_error", "Partial state root calculcation error")
		}
		cancelf()
		ps = rps
		return rps, nil
	}
	c.RequestEntityFromMinersOnMB(ctx, c.GetCurrentMagicBlock(), psRequestor, params, handler)
	if ps == nil {
		return nil, common.NewError("partial_state_change_error", "Error getting the partial state")
	}
	return ps, nil
}

func (c *Chain) getStateNodes(ctx context.Context, keys []util.Key) (*state.Nodes, error) {
	nsRequestor := StateNodesRequestor
	params := &url.Values{}
	for _, key := range keys {
		params.Add("nodes", util.ToHex(key))
	}
	ctx, cancelf := context.WithCancel(common.GetRootContext())
	var ns *state.Nodes
	handler := func(ctx context.Context, entity datastore.Entity) (interface{}, error) {
		rns, ok := entity.(*state.Nodes)
		if !ok {
			return nil, datastore.ErrInvalidEntity
		}
		if len(rns.Nodes) == 0 {
			return nil, util.ErrNodeNotFound
		}
		logging.Logger.Info("get state nodes", zap.Int("keys", len(keys)), zap.Int("nodes", len(rns.Nodes)))
		cancelf()
		ns = rns
		return rns, nil
	}
	mb := c.GetCurrentMagicBlock()
	c.RequestEntityFromMinersOnMB(ctx, mb, nsRequestor, params, handler)
	if ns == nil {
		c.RequestEntityFromShardersOnMB(ctx, mb, nsRequestor, params, handler)
	}
	if ns == nil {
		return nil, common.NewError("state_nodes_error", "Error getting the state nodes")
	}
	return ns, nil
}

func (c *Chain) getStateNodesFromSharders(ctx context.Context, keys []util.Key) (*state.Nodes, error) {
	nsRequestor := StateNodesRequestor
	params := &url.Values{}
	for _, key := range keys {
		params.Add("nodes", util.ToHex(key))
	}
	ctx, cancelf := context.WithCancel(common.GetRootContext())
	var ns *state.Nodes
	handler := func(ctx context.Context, entity datastore.Entity) (interface{}, error) {
		rns, ok := entity.(*state.Nodes)
		if !ok {
			return nil, datastore.ErrInvalidEntity
		}
		if len(rns.Nodes) == 0 {
			return nil, util.ErrNodeNotFound
		}
		logging.Logger.Info("get state nodes", zap.Int("keys", len(keys)), zap.Int("nodes", len(rns.Nodes)))
		cancelf()
		ns = rns
		return rns, nil
	}
	c.RequestEntityFromShardersOnMB(ctx, c.GetCurrentMagicBlock(), nsRequestor, params, handler)
	if ns == nil {
		return nil, common.NewError("state_nodes_error", "Error getting the state nodes")
	}
	return ns, nil
}

func (c *Chain) getBlockStateChange(b *block.Block) (*block.StateChange, error) {
	var (
		params       = &url.Values{}
		ctx, cancelf = context.WithCancel(common.GetRootContext())
		bsc          *block.StateChange
	)
	params.Add("block", b.Hash)

	var handler = func(ctx context.Context, entity datastore.Entity) (
		resp interface{}, err error) {

		var rsc, ok = entity.(*block.StateChange)
		if !ok {
			return nil, datastore.ErrInvalidEntity
		}

		if rsc.Block != b.Hash {
			logging.Logger.Error("get_block_state_change",
				zap.Error(errors.New("block hash mismatch")),
				zap.Int64("round", b.Round),
				zap.String("block", b.Hash))
			return nil, block.ErrBlockHashMismatch
		}

		if bytes.Compare(b.ClientStateHash, rsc.Hash) != 0 {
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

		logging.Logger.Debug("get_block_state_change - success with root",
			zap.Int64("round", b.Round),
			zap.String("bsc root", rsc.GetRoot().GetHash()),
			zap.String("block state hash", util.ToHex(b.ClientStateHash)))
		cancelf()
		bsc = rsc
		return rsc, nil
	}

	c.RequestEntityFromMinersOnMB(ctx, c.GetMagicBlock(b.Round), BlockStateChangeRequestor, params, handler)

	if bsc == nil {
		return nil, common.NewError("block_state_change_error",
			"error getting the block state change")
	}

	return bsc, nil
}

// ApplyBlockStateChange - applies the state chagnes to the block state.
func (c *Chain) ApplyBlockStateChange(b *block.Block, bsc *block.StateChange) error {
	return b.ApplyBlockStateChange(bsc, c)
}
