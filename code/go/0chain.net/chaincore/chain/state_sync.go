package chain

import (
	"context"
	"bytes"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/block"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	. "0chain.net/core/logging"
	"0chain.net/core/util"
	"go.uber.org/zap"
)

var ErrNodeNull = common.NewError("node_null", "Node is not available")

var ErrStopIterator = common.NewError("stop_iterator", "Stop MPT Iteration")

var MaxStateNodesForSync = 10000

//GetBlockStateChange - get the state change of the block
func (c *Chain) GetBlockStateChange(b *block.Block) {
	bsc, err := c.getBlockStateChange(b)
	if err != nil {
		Logger.Error("get block state change - no bsc", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("state_hash", util.ToHex(b.ClientStateHash)), zap.Error(err))
		return
	}
	if bsc == nil {
		return
	}
	Logger.Info("get block state change", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("state_hash", util.ToHex(b.ClientStateHash)), zap.Int8("state_status", b.GetStateStatus()))
	err = c.ApplyBlockStateChange(b, bsc)
	if err != nil {
		Logger.Error("get block state change", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("state_hash", util.ToHex(b.ClientStateHash)), zap.Error(err))
	}
}

//GetPartialState - get the partial state from the network
func (c *Chain) GetPartialState(ctx context.Context, key util.Key) {
	ps, err := c.getPartialState(ctx, key)
	if err != nil {
		Logger.Error("get partial state - no ps", zap.String("key", util.ToHex(key)), zap.Error(err))
		return
	}
	err = c.SavePartialState(ctx, ps)
	if err != nil {
		Logger.Error("get partial state - error saving", zap.String("key", util.ToHex(key)), zap.Error(err))
	} else {
		Logger.Info("get partial state - saving", zap.String("key", util.ToHex(key)), zap.Int("nodes",len(ps.Nodes)))
	}
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
	err := c.LatestFinalizedBlock.ClientState.IterateFrom(ctx, key, handler, util.NodeTypeLeafNode|util.NodeTypeFullNode|util.NodeTypeExtensionNode)
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

func (c *Chain) getPartialState(ctx context.Context, key util.Key) (*state.PartialState, error) {
	psRequestor := PartialStateRequestor
	params := map[string]string{"node": util.ToHex(key)}
	ctx, cancelf := context.WithCancel(common.GetRootContext())
	var ps *state.PartialState
	handler := func(ctx context.Context, entity datastore.Entity) (interface{}, error) {
		Logger.Debug("get partial state", zap.String("ps_id", entity.GetKey()))
		rps, ok := entity.(*state.PartialState)
		if !ok {
			return nil, datastore.ErrInvalidEntity
		}
		Logger.Info("get partial state",zap.String("key", util.ToHex(key)),zap.Int("nodes",len(rps.Nodes)))
		if bytes.Compare(key, rps.Hash) != 0 {
			Logger.Error("get partial state - state hash mismatch error", zap.String("key", util.ToHex(key)), zap.Any("hash", util.ToHex(ps.Hash)))
			return nil, state.ErrHashMismatch
		}
		root := rps.GetRoot()
		if root == nil {
			Logger.Error("get partial state - state root error", zap.Int("state_nodes", len(ps.Nodes)))
			return nil, common.NewError("state_root_error", "Partial state root calculcation error")
		}
		cancelf()
		ps = rps
		return rps, nil
	}
	c.Miners.RequestEntity(ctx, psRequestor, params, handler)
	if ps == nil {
		return nil, common.NewError("partial_state_change_error", "Error getting the partial state")
	}
	return ps, nil
}

func (c *Chain) getBlockStateChange(b *block.Block) (*block.StateChange, error) {
	if b.PrevBlock == nil {
		return nil, ErrPreviousBlockUnavailable
	}
	if bytes.Compare(b.ClientStateHash, b.PrevBlock.ClientStateHash) == 0 {
		b.SetStateStatus(block.StateSynched)
		return nil, nil
	}
	bscRequestor := BlockStateChangeRequestor
	params := map[string]string{"block": b.Hash}
	ctx, cancelf := context.WithCancel(common.GetRootContext())
	var bsc *block.StateChange
	handler := func(ctx context.Context, entity datastore.Entity) (interface{}, error) {
		Logger.Debug("get block state change", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("bsc_id", entity.GetKey()))
		rsc, ok := entity.(*block.StateChange)
		if !ok {
			return nil, datastore.ErrInvalidEntity
		}
		if rsc.Block != b.Hash {
			Logger.Error("get block state change - hash mismatch error", zap.Int64("round", b.Round), zap.String("block", b.Hash))
			return nil, block.ErrBlockHashMismatch
		}
		if bytes.Compare(b.ClientStateHash, rsc.Hash) != 0 {
			Logger.Error("get block state change - state hash mismatch error", zap.Int64("round", b.Round), zap.String("block", b.Hash))
			return nil, block.ErrBlockStateHashMismatch
		}
		root := rsc.GetRoot()
		if root == nil {
			Logger.Error("get block state change - state root error", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Int("state_nodes", len(rsc.Nodes)))
			return nil, common.NewError("state_root_error", "Block state root calculcation error")
		}
		cancelf()
		bsc = rsc
		return rsc, nil
	}
	c.Miners.RequestEntity(ctx, bscRequestor, params, handler)
	if bsc == nil {
		return nil, common.NewError("block_state_change_error", "Error getting the block state change")
	}
	return bsc, nil
}

//ApplyBlockStateChange - apply the state chagnes to the block state
func (c *Chain) ApplyBlockStateChange(b *block.Block, bsc *block.StateChange) error {
	lock := b.StateMutex
	lock.Lock()
	defer lock.Unlock()
	return c.applyBlockStateChange(b, bsc)
}

func (c *Chain) applyBlockStateChange(b *block.Block, bsc *block.StateChange) error {
	if b.Hash != bsc.Block {
		return block.ErrBlockHashMismatch
	}
	if bytes.Compare(b.ClientStateHash, bsc.Hash) != 0 {
		return block.ErrBlockStateHashMismatch
	}
	root := bsc.GetRoot()
	if root == nil {
		if b.PrevBlock != nil && bytes.Compare(b.PrevBlock.ClientStateHash, b.ClientStateHash) == 0 {
			return nil
		}
		return common.NewError("state_root_error", "state root not correct")
	}
	if b.ClientState == nil {
		b.CreateState(bsc.GetNodeDB())
	}

	c.stateMutex.Lock()
	defer c.stateMutex.Unlock()

	err := b.ClientState.MergeDB(bsc.GetNodeDB(), bsc.GetRoot().GetHashBytes())
	if err != nil {
		Logger.Error("apply block state change - error merging", zap.Int64("round", b.Round), zap.String("block", b.Hash))
	}
	b.SetStateStatus(block.StateSynched)
	return nil
}
