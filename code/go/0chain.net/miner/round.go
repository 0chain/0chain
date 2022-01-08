package miner

import (
	"context"
	"sync"
	"time"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/round"
	"0chain.net/core/logging"
	"go.uber.org/zap"
)

/*Round - a round from miner's perspective */
type Round struct {
	*round.Round
	roundGuard            sync.RWMutex
	vrfThresholdGuard     sync.RWMutex
	blocksToVerifyChannel chan *block.Block
	verificationCancelf   context.CancelFunc
	generationCancelf     context.CancelFunc
	delta                 time.Duration
	verificationTickets   map[string]*block.BlockVerificationTicket
	vrfShare              *round.VRFShare
	vrfSharesCache        *vrfSharesCache
	ownVerificationTicket *block.BlockVerificationTicket
}

func (r *Round) SetGenerationCancelf(generationCancelf context.CancelFunc) {
	r.roundGuard.Lock()
	r.generationCancelf = generationCancelf
	r.roundGuard.Unlock()
}

func (r *Round) TryCancelBlockGeneration() {
	r.roundGuard.Lock()
	if r.generationCancelf == nil {
		logging.Logger.Info("Try to cancel block generation that have not been started yet",
			zap.Int64("round", r.Number))
		r.roundGuard.Unlock()
		return
	}
	logging.Logger.Info("Cancelling block generation", zap.Int64("round", r.Number))
	f := r.generationCancelf
	r.generationCancelf = nil
	r.roundGuard.Unlock()

	f()
}

func (r *Round) SetVerificationCancelf(verificationCancelf context.CancelFunc) {
	r.roundGuard.Lock()
	r.verificationCancelf = verificationCancelf
	r.roundGuard.Unlock()
}

func (r *Round) VrfShare() *round.VRFShare {
	r.roundGuard.RLock()
	defer r.roundGuard.RUnlock()
	return r.vrfShare
}

func (r *Round) SetVrfShare(vrfShare *round.VRFShare) {
	r.roundGuard.Lock()
	defer r.roundGuard.Unlock()
	r.vrfShare = vrfShare
}

func (r *Round) OwnVerificationTicket() *block.BlockVerificationTicket {
	r.roundGuard.RLock()
	defer r.roundGuard.RUnlock()
	return r.ownVerificationTicket
}

func (r *Round) SetOwnVerificationTicket(ownVerificationTicket *block.BlockVerificationTicket) {
	r.roundGuard.Lock()
	r.ownVerificationTicket = ownVerificationTicket
	r.roundGuard.Unlock()
}

type vrfSharesCache struct {
	vrfShares map[string]*round.VRFShare
	mutex     *sync.Mutex
}

func newVRFSharesCache() *vrfSharesCache {
	return &vrfSharesCache{
		vrfShares: make(map[string]*round.VRFShare),
		mutex:     &sync.Mutex{},
	}
}

func (v *vrfSharesCache) add(vrfShare *round.VRFShare) {
	v.mutex.Lock()
	defer v.mutex.Unlock()
	k := vrfShare.GetParty().GetKey()
	if _, ok := v.vrfShares[k]; ok {
		return
	}
	v.vrfShares[k] = vrfShare
}

func (v *vrfSharesCache) getAll() []*round.VRFShare {
	v.mutex.Lock()
	vrfShares := make([]*round.VRFShare, 0, len(v.vrfShares))
	for _, vrf := range v.vrfShares {
		vrfShares = append(vrfShares, vrf)
	}
	v.mutex.Unlock()
	return vrfShares
}

// clean deletes shares that has round time out count <= the 'count' value
func (v *vrfSharesCache) clean(count int) {
	v.mutex.Lock()
	for s, vrf := range v.vrfShares {
		if vrf.GetRoundTimeoutCount() <= count {
			delete(v.vrfShares, s)
		}
	}
	v.mutex.Unlock()
}

/*AddBlockToVerify - adds a block to the round. Assumes non-concurrent update */
func (r *Round) AddBlockToVerify(b *block.Block) {
	roundNumber := r.GetRoundNumber()
	//if r.IsVerificationComplete() {
	//	logging.Logger.Debug("block proposal - verification complete", zap.Int64("round", roundNumber), zap.String("block", b.Hash))
	//	return
	//}
	if roundNumber != b.Round {
		logging.Logger.Error("block proposal - round mismatch", zap.Int64("round", roundNumber), zap.Int64("block_round", b.Round), zap.String("block", b.Hash))
		return
	}
	//if b.GetRoundRandomSeed() != r.GetRandomSeed() {
	//	return
	//}

	if b.GetRoundRandomSeed() == 0 {
		logging.Logger.Error("block proposal - block with no RRS",
			zap.Int64("round", roundNumber))
		return
	}

	logging.Logger.Debug("Adding block to verifyChannel",
		zap.Int64("round", b.Round),
		zap.String("block hash", b.Hash),
		zap.String("magic block", b.LatestFinalizedMagicBlockHash),
		zap.Int64("magic block round", b.LatestFinalizedMagicBlockRound))

	//we use one minute timeout here for emergency case, when buffered channel is full
	timeout, _ := context.WithTimeout(context.Background(), time.Minute)
	select {
	case <-timeout.Done():
		logging.Logger.Debug("Can't add block to verify channel, context is shut")
	case r.blocksToVerifyChannel <- b:
	default:
	}
}

// AddVerificationTickets - add verification tickets
func (r *Round) AddVerificationTickets(bvts []*block.BlockVerificationTicket) {
	r.roundGuard.Lock()
	defer r.roundGuard.Unlock()
	for i, bvt := range bvts {
		r.verificationTickets[bvt.Signature] = bvts[i]
	}
}

/*GetVerificationTickets - get verification tickets for a given block in this round */
func (r *Round) GetVerificationTickets(blockID string) []*block.VerificationTicket {
	var vts []*block.VerificationTicket
	r.roundGuard.Lock()
	defer r.roundGuard.Unlock()
	for _, bvt := range r.verificationTickets {
		if blockID == bvt.BlockID {
			vts = append(vts, &bvt.VerificationTicket)
		}
	}
	return vts
}

// IsTicketCollected checks if the ticket has already verified and collected
func (r *Round) IsTicketCollected(ticket *block.VerificationTicket) (exist bool) {
	r.roundGuard.Lock()
	vt, ok := r.verificationTickets[ticket.Signature]
	exist = ok && vt.VerificationTicket == *ticket
	r.roundGuard.Unlock()
	return
}

/*GetBlocksToVerifyChannel - a channel where all the blocks requiring verification are put into */
func (r *Round) GetBlocksToVerifyChannel() chan *block.Block {
	return r.blocksToVerifyChannel
}

/*IsVerificationComplete - indicates if the verification process for the round is complete */
func (r *Round) IsVerificationComplete() bool {
	return r.isVerificationComplete()
}

func (r *Round) isVerificationComplete() bool {
	return r.GetPhase() >= round.Notarize
}

/*StartVerificationBlockCollection - start collecting blocks for verification */
func (r *Round) StartVerificationBlockCollection(ctx context.Context) context.Context {
	r.roundGuard.Lock()
	defer r.roundGuard.Unlock()

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
	r.roundGuard.Lock()
	defer r.roundGuard.Unlock()
	f := r.verificationCancelf
	if f == nil {
		return
	}
	logging.Logger.Info("Cancelling verification", zap.Int64("round", r.Number))
	r.verificationCancelf = nil
	f()
}

/*Clear - clear any pending state before deleting this round */
func (r *Round) Clear() {
	logging.Logger.Debug("Round clear - cancel verification")
	r.CancelVerification()
}

//IsVRFComplete - is the VRF process complete?
func (r *Round) IsVRFComplete() bool {
	return r.HasRandomSeed()
}

// Restart resets round and vrf shares cache
func (r *Round) Restart() error {

	if err := r.Round.Restart(); err != nil {
		return err
	}
	r.CancelVerification()
	r.TryCancelBlockGeneration()

	r.roundGuard.Lock()
	r.vrfSharesCache = newVRFSharesCache()
	r.blocksToVerifyChannel = make(chan *block.Block, cap(r.blocksToVerifyChannel))
	r.verificationTickets = make(map[string]*block.BlockVerificationTicket)
	r.roundGuard.Unlock()
	return nil
}

func (r *Round) IsComplete() bool {
	return r.GetPhase() == round.Complete
}
