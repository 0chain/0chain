package miner

import (
	"context"
	"time"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/round"
	. "0chain.net/core/logging"
	"go.uber.org/zap"
)

/*Round - a round from miner's perspective */
type Round struct {
	*round.Round
	blocksToVerifyChannel chan *block.Block
	verificationCancelf   context.CancelFunc
	delta                 time.Duration
	verificationTickets   map[string]*block.BlockVerificationTicket
	vrfShare              *round.VRFShare
}

/*AddBlockToVerify - adds a block to the round. Assumes non-concurrent update */
func (r *Round) AddBlockToVerify(b *block.Block) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	if r.isVerificationComplete() {
		Logger.Debug("block proposal - verification complete", zap.Int64("round", r.GetRoundNumber()), zap.String("block", b.Hash))
		return
	}
	if r.GetRoundNumber() != b.Round {
		Logger.Error("block proposal - round mismatch", zap.Int64("round", r.GetRoundNumber()), zap.Int64("block_round", b.Round), zap.String("block", b.Hash))
		return
	}
	if b.RoundRandomSeed != r.RandomSeed {
		return
	}
	Logger.Debug("Adding block to verifyChannel")
	r.blocksToVerifyChannel <- b
}

/*AddVerificationTicket - add a verification ticket */
func (r *Round) AddVerificationTicket(bvt *block.BlockVerificationTicket) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	r.verificationTickets[bvt.Signature] = bvt
}

/*GetVerificationTickets - get verification tickets for a given block in this round */
func (r *Round) GetVerificationTickets(blockID string) []*block.VerificationTicket {
	var vts []*block.VerificationTicket
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	for _, bvt := range r.verificationTickets {
		if blockID == bvt.BlockID {
			vts = append(vts, &bvt.VerificationTicket)
		}
	}
	return vts
}

/*GetBlocksToVerifyChannel - a channel where all the blocks requiring verification are put into */
func (r *Round) GetBlocksToVerifyChannel() chan *block.Block {
	return r.blocksToVerifyChannel
}

/*IsVerificationComplete - indicates if the verification process for the round is complete */
func (r *Round) IsVerificationComplete() bool {
	r.Mutex.RLock()
	defer r.Mutex.RUnlock()
	return r.isVerificationComplete()
}

func (r *Round) isVerificationComplete() bool {
	return r.GetState() >= round.RoundStateVerificationTimedOut
}

/*StartVerificationBlockCollection - start collecting blocks for verification */
func (r *Round) StartVerificationBlockCollection(ctx context.Context) context.Context {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	if r.verificationCancelf != nil {
		return nil
	}
	if r.isVerificationComplete() {
		return nil
	}
	lctx, cancelf := context.WithCancel(ctx)
	r.verificationCancelf = cancelf
	return lctx
}

/*CancelVerification - Cancel verification of blocks */
func (r *Round) CancelVerification() {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	f := r.verificationCancelf
	if f == nil {
		return
	}
	r.verificationCancelf = nil
	f()
}

/*Clear - clear any pending state before deleting this round */
func (r *Round) Clear() {
	r.CancelVerification()
}

//IsVRFComplete - is the VRF process complete?
func (r *Round) IsVRFComplete() bool {
	return r.HasRandomSeed()
}
