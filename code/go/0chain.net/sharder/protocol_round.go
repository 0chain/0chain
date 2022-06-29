package sharder

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	. "0chain.net/core/logging"
	"go.uber.org/zap"
)

var ErrNoPreviousBlock = errors.New("previous block does not exist")

// AddNotarizedBlock - add a notarized block for a given round.
func (sc *Chain) AddNotarizedBlock(ctx context.Context, r round.RoundI,
	b *block.Block) error {

	_, _ = r.AddNotarizedBlock(b)

	if sc.BlocksToSharder == chain.FINALIZED {
		nb := r.GetNotarizedBlocks()
		if len(nb) > 0 {
			Logger.Error("*** different blocks for the same round ***",
				zap.Any("round", b.Round), zap.Any("block", b.Hash),
				zap.Any("existing_block", nb[0].Hash))
		}
	}

	pb, _ := sc.GetBlock(ctx, b.PrevHash)
	if pb == nil {
		return ErrNoPreviousBlock
	}

	if pb.ClientState == nil || pb.GetStateStatus() != block.StateSuccessful {
		return fmt.Errorf("previous block state is not computed, state status: %d, hash: %s",
			pb.GetStateStatus(), pb.Hash)
	}

	errC := make(chan error)
	doneC := make(chan struct{})
	t := time.Now()
	tc := math.Max(float64(time.Duration(len(b.Txns))*50*time.Millisecond), float64(3*time.Second))
	cctx, cancel := context.WithTimeout(ctx, time.Duration(tc))
	defer cancel()
	go func(ctx context.Context) {
		defer close(doneC)
		if b.ClientState != nil {
			// check if the block's client state is correct
			if !bytes.Equal(b.ClientStateHash, b.ClientState.GetRoot()) {
				select {
				case errC <- errors.New("AddNotarizedBlock block client state does not match"):
				default:
				}
				return
			}
		} else {
			Logger.Debug("AddNotarizedBlock client state is nil", zap.Int64("round", b.Round))
		}

		if err := sc.ComputeState(ctx, b); err != nil {
			select {
			case errC <- err:
			default:
			}
			return
		}
	}(cctx)

	select {
	case <-doneC:
		Logger.Debug("AddNotarizedBlock compute state successfully", zap.Any("duration", time.Since(t)))
	case err := <-errC:
		Logger.Error("AddNotarizedBlock failed to compute state",
			zap.Int64("round", b.Round),
			zap.Error(err))
		if node.Self.IsSharder() {
			return err
		}
	}

	sc.SetCurrentRound(r.GetRoundNumber())
	sc.UpdateNodeState(b)

	go sc.FinalizeRound(r)
	return nil
}
