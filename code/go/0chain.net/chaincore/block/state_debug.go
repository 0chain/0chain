package block

import (
	"bytes"
	"context"
	"fmt"
	"os"

	"0chain.net/chaincore/state"
	"github.com/0chain/common/core/logging"
	"github.com/0chain/common/core/util"
	"go.uber.org/zap"
)

var StateOut *os.File

/*SetupStateLogger - a separate logger for state to be able to debug state */
func SetupStateLogger(file string) {
	out, err := os.Create(file)
	if err != nil {
		panic(err)
	}
	StateOut = out
	fmt.Fprintf(StateOut, "starting state log ...\n")
}

//StateSanityCheck - after generating a block or verification of a block, this can be called to run some state sanity checks
func StateSanityCheck(ctx context.Context, b *Block) {
	if !state.DebugBlock() {
		return
	}
	if bytes.Equal(b.ClientStateHash, b.PrevBlock.ClientStateHash) {
		return
	}
	if err := ValidateState(ctx, b, b.PrevBlock.ClientState.GetRoot()); err != nil {
		logging.Logger.DPanic("state sanity check - state change validation", zap.Error(err))
	}
	if err := validateStateChangesRoot(b); err != nil {
		logging.Logger.DPanic("state sanity check - state changes root validation", zap.Error(err))
	}
}

func validateStateChangesRoot(b *Block) error {
	bsc, err := NewBlockStateChange(b)
	if err != nil {
		return err
	}

	if b.ClientStateHash != nil && (bsc.GetRoot() == nil ||
		!bytes.Equal(bsc.GetRoot().GetHashBytes(), b.ClientStateHash)) {
		computedRoot := ""
		if bsc.GetRoot() != nil {
			computedRoot = bsc.GetRoot().GetHash()
		}
		logging.Logger.Error("block state change - root mismatch", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("state_root", util.ToHex(b.ClientStateHash)), zap.Any("computed_root", computedRoot))
		return ErrStateMismatch
	}
	return nil
}

func PrintStates(cstate util.MerklePatriciaTrieI, pstate util.MerklePatriciaTrieI) {
	if !state.Debug() || StateOut == nil {
		return
	}
	cstate.PrettyPrint(StateOut)
	fmt.Fprintf(StateOut, "previous state\n")
	pstate.PrettyPrint(StateOut)
}

func ValidateState(ctx context.Context, b *Block, priorRoot util.Key) error {
	if b.ClientState.GetChangeCount() > 0 {
		changes, err := NewBlockStateChange(b)
		if err != nil {
			return err
		}

		stateRoot := changes.GetRoot()
		if stateRoot == nil {
			if StateOut != nil {
				b.ClientState.PrettyPrint(StateOut)
			}
			if state.DebugBlock() {
				logging.Logger.DPanic("validate state - state root is null", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Int("changes", len(changes.Nodes)))
			} else {
				logging.Logger.Error("validate state - state root is null", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Int("changes", len(changes.Nodes)))
			}
		}
		if !bytes.Equal(stateRoot.GetHashBytes(), b.ClientState.GetRoot()) {
			if StateOut != nil {
				_, changes, _, _ := b.ClientState.GetChanges()
				util.PrintChanges(StateOut, changes)
				b.ClientState.PrettyPrint(StateOut)
			}
			if state.DebugBlock() {
				logging.Logger.DPanic("validate state", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Any("state", util.ToHex(b.ClientState.GetRoot())), zap.String("computed_state", stateRoot.GetHash()), zap.Int("changes", len(changes.Nodes)))
			} else {
				logging.Logger.Error("validate state", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Any("state", util.ToHex(b.ClientState.GetRoot())), zap.String("computed_state", stateRoot.GetHash()), zap.Int("changes", len(changes.Nodes)))
			}
		}
		if priorRoot == nil {
			priorRoot = b.PrevBlock.ClientState.GetRoot()
		}
		err = changes.Validate(ctx)
		if err != nil {
			logging.Logger.Error("validate state - changes validate failure", zap.Error(err))
			pstate := util.NewMerklePatriciaTrie(b.ClientState.GetNodeDB(), b.ClientState.GetVersion(), priorRoot)
			PrintStates(b.ClientState, pstate)
			return err
		}
		err = b.ClientState.Validate()
		if err != nil {
			logging.Logger.Error("validate state - client state validate failure", zap.Error(err))
			pstate := util.NewMerklePatriciaTrie(b.ClientState.GetNodeDB(), b.ClientState.GetVersion(), priorRoot)
			PrintStates(b.ClientState, pstate)
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
