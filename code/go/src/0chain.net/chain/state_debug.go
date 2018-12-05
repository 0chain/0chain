package chain

import (
	"bytes"
	"context"
	"fmt"
	"os"

	"0chain.net/block"
	. "0chain.net/logging"
	"0chain.net/state"
	"0chain.net/util"
	"go.uber.org/zap"
)

var stateOut *os.File

/*SetupStateLogger - a separate logger for state to be able to debug state */
func SetupStateLogger(file string) {
	out, err := os.Create(file)
	if err != nil {
		panic(err)
	}
	stateOut = out
	fmt.Fprintf(stateOut, "starting state log ...\n")
}

//StateSanityCheck - after generating a block or verification of a block, this can be called to run some state sanity checks
func (c *Chain) StateSanityCheck(ctx context.Context, b *block.Block) {
	if !state.DebugBlock() {
		return
	}
	if bytes.Compare(b.ClientStateHash, b.PrevBlock.ClientStateHash) == 0 {
		return
	}
	if err := c.validateState(ctx, b, b.PrevBlock.ClientState.GetRoot()); err != nil {
		Logger.DPanic("state sanity check - state change validation", zap.Error(err))
	}
	if err := c.validateStateChangesRoot(b); err != nil {
		Logger.DPanic("state sanity check - state changes root validation", zap.Error(err))
	}
}

func (c *Chain) validateState(ctx context.Context, b *block.Block, priorRoot util.Key) error {
	if len(b.ClientState.GetChangeCollector().GetChanges()) > 0 {
		changes := block.NewBlockStateChange(b)
		stateRoot := changes.GetRoot()
		if stateRoot == nil {
			if stateOut != nil {
				b.ClientState.PrettyPrint(stateOut)
			}
			if state.DebugBlock() {
				Logger.DPanic("validate state - state root is null", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Int("changes", len(changes.Nodes)))
			} else {
				Logger.Error("validate state - state root is null", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Int("changes", len(changes.Nodes)))
			}
		}
		if bytes.Compare(stateRoot.GetHashBytes(), b.ClientState.GetRoot()) != 0 {
			if stateOut != nil {
				b.ClientState.GetChangeCollector().PrintChanges(stateOut)
				b.ClientState.PrettyPrint(stateOut)
			}
			if state.DebugBlock() {
				Logger.DPanic("validate state", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Any("state", util.ToHex(b.ClientState.GetRoot())), zap.String("computed_state", stateRoot.GetHash()), zap.Int("changes", len(changes.Nodes)))
			} else {
				Logger.Error("validate state", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Any("state", util.ToHex(b.ClientState.GetRoot())), zap.String("computed_state", stateRoot.GetHash()), zap.Int("changes", len(changes.Nodes)))
			}
		}
		if priorRoot == nil {
			priorRoot = b.PrevBlock.ClientState.GetRoot()
		}
		err := changes.Validate(ctx)
		if err != nil {
			Logger.Error("validate state - changes validate failure", zap.Error(err))
			pstate := util.CloneMPT(b.ClientState)
			pstate.SetRoot(priorRoot)
			printStates(b.ClientState, pstate)
			return err
		}
		err = b.ClientState.Validate()
		if err != nil {
			Logger.Error("validate state - client state validate failure", zap.Error(err))
			pstate := util.CloneMPT(b.ClientState)
			pstate.SetRoot(priorRoot)
			printStates(b.ClientState, pstate)
			/*
				if state.Debug() && stateOut != nil {
					fmt.Fprintf(stateOut, "previous block\n")
					if bytes.Compare(b.PrevBlock.ClientState.GetRoot(), priorRoot) != 0 {
						b.PrevBlock.ClientState.PrettyPrint(stateOut)
					}
				}*/
			return err
		}
	}
	/*
		if b.Round > 15 {
			state.SetDebugLevel(state.DebugLevelTxn)
		}*/
	return nil
}

func (c *Chain) validateStateChangesRoot(b *block.Block) error {
	bsc := block.NewBlockStateChange(b)
	if b.ClientStateHash != nil && (bsc.GetRoot() == nil || bytes.Compare(bsc.GetRoot().GetHashBytes(), b.ClientStateHash) != 0) {
		computedRoot := ""
		if bsc.GetRoot() != nil {
			computedRoot = bsc.GetRoot().GetHash()
		}
		Logger.Error("block state change - root mismatch", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("state_root", util.ToHex(b.ClientStateHash)), zap.Any("computed_root", computedRoot))
		return ErrStateMismatch
	}
	return nil
}

func printStates(cstate util.MerklePatriciaTrieI, pstate util.MerklePatriciaTrieI) {
	if !state.Debug() || stateOut == nil {
		return
	}
	cstate.PrettyPrint(stateOut)
	fmt.Fprintf(stateOut, "previous state\n")
	pstate.PrettyPrint(stateOut)
}
