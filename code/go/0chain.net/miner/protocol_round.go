package miner

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"sync"
	"time"

	"0chain.net/core/config"
	"github.com/rcrowley/go-metrics"
	"go.uber.org/zap"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/memorystore"
	"github.com/0chain/common/core/logging"
	"github.com/0chain/common/core/util"
)

var rbgTimer metrics.Timer // round block generation timer

func init() {
	rbgTimer = metrics.GetOrRegisterTimer("rbg_time", nil)
}

// SetNetworkRelayTime - set the network relay time.
func SetNetworkRelayTime(delta time.Duration) {
	chain.SetNetworkRelayTime(delta)
}

func (mc *Chain) addMyVRFShare(ctx context.Context, pr *Round, r *Round) {

	var (
		rn  = r.GetRoundNumber()
		mb  = mc.GetMagicBlock(rn)
		snk = node.Self.Underlying().GetKey()
	)

	if !mb.Miners.HasNode(snk) {
		return
	}

	var dkg = mc.GetDKG(rn)
	if dkg == nil {
		logging.Logger.Error("add_my_vrf_share -- DKG is nil, my VRF share is not added",
			zap.Int64("round", rn))
		return
	}

	var (
		vrfs = &round.VRFShare{}
		err  error
	)
	vrfs.Round = rn
	vrfs.RoundTimeoutCount = r.GetTimeoutCount()

	if vrfs.Share, err = mc.GetBlsShare(ctx, r.Round); err != nil {
		logging.Logger.Error("add_my_vrf_share", zap.Int64("round", vrfs.Round),
			zap.Int("round_timeout", vrfs.RoundTimeoutCount),
			zap.Int64("dkg_starting_round", dkg.StartingRound),
			zap.Int64("dkg_mb_number", dkg.MagicBlockNumber),
			zap.Int64("pr_seed", pr.GetRandomSeed()),
			zap.String("pr_vrf_seed", strconv.FormatInt(pr.GetRandomSeed(), 16)),
			zap.Error(err))
		return
	}

	logging.Logger.Info("add_my_vrf_share", zap.Int64("round", vrfs.Round),
		zap.Int("round_timeout", vrfs.RoundTimeoutCount),
		zap.Int64("dkg_starting_round", dkg.StartingRound),
		zap.Int64("dkg_mb_number", dkg.MagicBlockNumber),
		zap.Int64("pr_seed", pr.GetRandomSeed()),
		zap.String("pr_vrf_seed", strconv.FormatInt(pr.GetRandomSeed(), 16)),
		zap.String("share", vrfs.Share))

	vrfs.SetParty(node.Self.Underlying())
	r.SetVrfShare(vrfs)
	// TODO: do we need to check if AddVRFShare is success or not?
	mc.AddVRFShare(ctx, r, vrfs)
	go mc.SendVRFShare(context.Background(), vrfs.Clone())
}

func (mc *Chain) isAheadOfSharders(ctx context.Context, round int64) bool {

	var (
		ahead = config.GetLFBTicketAhead()
		tk    = mc.GetLatestLFBTicket(ctx)
	)

	if tk == nil {
		return true // context done, treat it as ahead
	}

	if round+1 > tk.Round+int64(ahead) {
		logging.Logger.Warn("[is ahead]", zap.Int64("round", round),
			zap.Int64("ticket", tk.Round), zap.Int("ahead", ahead))
		return true // is ahead
	}

	logging.Logger.Debug("[is not ahead]", zap.Int64("round", round),
		zap.Int64("ticket", tk.Round), zap.Int("ahead", ahead))
	return false // is not ahead, can move on
}

// The waitNotAhead is similar to the isAheadOfSharders but it checks state and,
// if it's ahead, waits to be non-ahead or context is done or restart round
// event. Only in first case (successful case) it returns true. In two other
// cases it returns false and new round should not be created.
//
// The waitNotAhead may block for restart round time.
func (mc *Chain) waitNotAhead(ctx context.Context, round int64) (ok bool) {

	logging.Logger.Debug("[wait not ahead]")

	// subscribe to new LFB-tickets events; subscribe to restart round events
	var (
		tksubq = mc.SubLFBTicket()
		rrsubq = mc.subRestartRoundEvent()
	)
	defer mc.UnsubLFBTicket(tksubq)
	defer mc.unsubRestartRoundEvent(rrsubq)

	// check the current ticket

	var (
		ahead   = config.GetLFBTicketAhead()
		tk      = mc.GetLatestLFBTicket(ctx)
		lfb     = mc.GetLatestFinalizedBlock()
		tkRound int64
	)

	if tk == nil || lfb == nil {
		logging.Logger.Debug("[wait not ahead] [1] context is done or restart round")
		return false // context is done, can't wait anymore
	}

	if tk.Round > lfb.Round {
		tkRound = lfb.Round
	} else {
		tkRound = tk.Round
	}

	if round+1 <= tkRound+int64(ahead) {
		logging.Logger.Debug("[wait not ahead] [2] not ahead, can move on",
			zap.Int64("round", round),
			zap.Int64("lfb tk round", tkRound))
		return true // not ahead, can move on
	}

	// wait in loop
	tkr := time.NewTicker(100 * time.Millisecond)
	for {
		select {
		case <-tkr.C:
			lfb = mc.GetLatestFinalizedBlock()
			tk = mc.GetLatestLFBTicket(ctx)
			if tk.Round > lfb.Round {
				tkRound = lfb.Round
			} else {
				tkRound = tk.Round
			}

			if round+1 <= tkRound+int64(ahead) {
				logging.Logger.Debug("[wait not ahead] [3*] not ahead, can move on")
				return true // not ahead, can move on
			}
			logging.Logger.Debug("[wait not ahead] [4*] still ahead, can't move on")
			if tk.Round < lfb.Round {
				mc.BumpLFBTicket(ctx)
			}

		case ntk := <-tksubq: // the ntk can't be nil
			lfb = mc.GetLatestFinalizedBlock()
			if ntk.Round > lfb.Round { // ntk is ahead, use lfb
				tkRound = lfb.Round
			} else {
				tkRound = ntk.Round // lfb is ahead, use ntk?
			}

			if round+1 <= tkRound+int64(ahead) {
				logging.Logger.Debug("[wait not ahead] [3] not ahead, can move on")
				return true // not ahead, can move on
			}
			logging.Logger.Debug("[wait not ahead] [4] still ahead, can't move on",
				zap.Int64("current round", round),
				zap.Int64("ntk round", ntk.Round),
				zap.Int64("lfb round", lfb.Round))
		case <-rrsubq:
			logging.Logger.Debug("[wait not ahead] [5] restart round triggered")
			return // false, shouldn't move on
		case <-ctx.Done():
			logging.Logger.Debug("[wait not ahead] [6] context is done")
			return // context is done, shouldn't move on
		}
	}

}

func (mc *Chain) finalizeRound(ctx context.Context, r *Round) {
	logging.Logger.Debug("finalizedRound - cancel round verification", zap.Int64("round", r.GetRoundNumber()))
	mc.CancelRoundVerification(ctx, r)
	go mc.FinalizeRound(r.Round)
}

// Creates the next round, if next round exists and has RRS returns existent.
// If RRS is not present, starts VRF phase for this round
func (mc *Chain) startNextRound(ctx context.Context, r *Round) *Round {
	var (
		rn = r.GetRoundNumber()
		pr = mc.GetMinerRound(rn - 1)
	)

	if pr != nil && !pr.IsFinalizing() && !pr.IsFinalized() {
		mc.finalizeRound(ctx, pr) // finalize the previous round
	}

	var (
		nr = round.NewRound(rn + 1)
		mr = mc.CreateRound(nr)
		er = mc.AddRound(mr).(*Round)
	)
	mc.SetCurrentRound(er.GetRoundNumber())
	mc.finalizeRound(ctx, r) // finalize the notarized block round

	if er != mr && mc.isStarted() && er.HasRandomSeed() {
		logging.Logger.Info("StartNextRound found next round with RRS. No VRFShares Sent",
			zap.Int64("er_round", er.GetRoundNumber()),
			zap.Int64("rrs", er.GetRandomSeed()),
			zap.Bool("is_started", mc.isStarted()))
		return er
	}

	if r.HasRandomSeed() && er.VrfShare() == nil {
		logging.Logger.Info("StartNextRound - add VRF", zap.Int64("round", er.GetRoundNumber()))
		mc.addMyVRFShare(ctx, r, er)
	} else {
		logging.Logger.Info("StartNextRound no VRFShares sent -- "+
			"current round has no random seed",
			zap.Int64("rrs", r.GetRandomSeed()), zap.Int64("r_round", rn))
	}

	return er
}

// The getOrCreateRound returns existing round or creates new one. It ignores 'aheadness' and previous round presence.
func (mc *Chain) getOrCreateRound(ctx context.Context, rn int64) (mr *Round) {

	if mr = mc.GetMinerRound(rn); mr != nil {
		return // got existing round, ok
	}

	// create the round regardless everything

	var rx = round.NewRound(rn)
	mr = mc.CreateRound(rx)

	return mc.AddRound(mr).(*Round)
}

// RedoVrfShare re-calculateVrfShare and send.
func (mc *Chain) RedoVrfShare(ctx context.Context, r *Round) bool {
	var pr *Round
	if r.GetRoundNumber() > 0 {
		pr = mc.GetMinerRound(r.GetRoundNumber() - 1)
	} else {
		pr = r
	}

	if pr == nil {
		logging.Logger.Info("no pr info inside RedoVrfShare",
			zap.Int64("Round", r.GetRoundNumber()))
		return false
	}

	if pr.HasRandomSeed() {
		r.SetVrfShare(nil)
		logging.Logger.Info("RedoVrfShare after vrfShare is nil",
			zap.Int64("round", r.GetRoundNumber()),
			zap.Int("round_timeout", r.GetTimeoutCount()))
		mc.addMyVRFShare(ctx, pr, r)
		return true
	}

	return false
}

// TryProposeBlock generates block and sends it to the network if generator
func (mc *Chain) TryProposeBlock(ctx context.Context, mr *Round) {
	var rn = mr.GetRoundNumber()

	if rn < mc.GetCurrentRound() {
		logging.Logger.Debug("start new round (current round higher)",
			zap.Int64("round", rn),
			zap.Int64("current_round", mc.GetCurrentRound()))
		return
	}

	pr := mc.GetRound(rn - 1)
	if pr == nil {
		logging.Logger.Debug("start new round (previous round not found)",
			zap.Int64("round", rn))
		return
	}

	var (
		self = node.Self.Underlying()
		mn   = mc.GetMagicBlock(rn).Miners.GetNode(self.GetKey())
		rank = mr.GetMinerRank(mn)
	)

	if !mc.IsRoundGenerator(mr, mn) {
		logging.Logger.Info("TOC_FIX Not a generator", zap.Int64("round", rn),
			zap.Int("index", self.SetIndex),
			zap.Int("rank", rank),
			zap.Int("timeout_count", mr.GetTimeoutCount()),
			zap.Int64("random_seed", mr.GetRandomSeed()))
		// TODO: remove this debug
		go func(r int64) {
			// check if this round has got a block in 3 seconds, report if not
			time.AfterFunc(5*time.Second, func() {
				if bs := mc.GetRoundBlocks(r); len(bs) == 0 {
					logging.Logger.Warn("no block for new round in 5 seconds",
						zap.Int64("round", r))
				}
			})
		}(rn)
		return
	}

	logging.Logger.Info("*** TOC_FIX starting round block generation ***",
		zap.Int64("round", rn), zap.Int("index", self.SetIndex),
		zap.Int("rank", rank), zap.Int("timeout_count", mr.GetTimeoutCount()),
		zap.Int64("random_seed", mr.GetRandomSeed()),
		zap.Int64("lf_round", mc.GetLatestFinalizedBlock().Round))

	// NOTE: If there are not enough txns, this will not advance further even
	// though rest of the network is. That's why this is a goroutine.
	go func() {
		if _, err := mc.GenerateRoundBlock(ctx, mr); err != nil {
			logging.Logger.Error("generate round block failed", zap.Error(err))
		}
	}()
}

// GetBlockToExtend - Get the block to extend from the given round.
func (mc *Chain) getBlockToExtend(ctx context.Context, r round.RoundI) (
	bnb *block.Block) {

	if bnb = r.GetHeaviestNotarizedBlock(); bnb == nil {
		type pBlock struct {
			Block     string
			Proposals int
		}
		proposals := r.GetProposedBlocks()
		var pcounts []*pBlock
		for _, pb := range proposals {
			pcount := pb.VerificationTicketsSize()
			if pcount == 0 {
				continue
			}
			pcounts = append(pcounts, &pBlock{Block: pb.Hash, Proposals: pcount})
		}
		sort.SliceStable(pcounts, func(i, j int) bool {
			return pcounts[i].Proposals > pcounts[j].Proposals
		})
		logging.Logger.Info("get block to extend - no notarized block",
			zap.Int64("round", r.GetRoundNumber()),
			zap.Int("num_proposals", len(proposals)),
			zap.Any("verification_tickets", pcounts))
		bnb = mc.GetHeaviestNotarizedBlock(ctx, r)
	}

	if bnb == nil {
		logging.Logger.Debug("get block to extend - no block",
			zap.Int64("round", r.GetRoundNumber()),
			zap.Int64("current_round", mc.GetCurrentRound()))
		return // nil
	}

	if !bnb.IsStateComputed() {
		logging.Logger.Debug("best notarized block not computed yet", zap.Int64("round", r.GetRoundNumber()))
		err := mc.ComputeOrSyncState(ctx, bnb)
		if err != nil {
			logging.Logger.Debug("failed to compute or sync state of best notarized block",
				zap.Int64("round", r.GetRoundNumber()),
				zap.String("block", bnb.Hash),
				zap.Error(err))
			if state.DebugBlock() {
				logging.Logger.Error("get block to extend - best nb compute state",
					zap.Int64("round", r.GetRoundNumber()),
					zap.String("block", bnb.Hash), zap.Error(err))
				return nil
			}
		}
	}

	return // bnb
}

// generateRoundBlock - given a round number generates a block.
func (mc *Chain) generateRoundBlock(ctx context.Context, r *Round) (*block.Block, error) {
	var ts = time.Now()
	defer func() { rbgTimer.UpdateSince(ts) }()

	roundNumber := r.GetRoundNumber()
	pround := mc.GetRound(roundNumber - 1)
	if pround == nil {
		logging.Logger.Error("generate round block - no prior round", zap.Int64("round", roundNumber-1))
		return nil, common.NewError("invalid_round,", "Round not available")
	}

	pb := mc.GetBlockToExtend(ctx, pround)
	if pb == nil {
		logging.Logger.Error("generate round block - no block to extend", zap.Int64("round", roundNumber))
		return nil, common.NewError("block_gen_no_block_to_extend", "Do not have the block to extend this round")
	}

	if !pb.IsStateComputed() {
		logging.Logger.Debug("GenerateRoundBlock, state of prior round block not computed",
			zap.Int8("state status", pb.GetStateStatus()))
	}
	withCancel, cancelFunc := context.WithCancel(common.GetRootContext())
	r.SetGenerationCancelf(cancelFunc)
	txnEntityMetadata := datastore.GetEntityMetadata("txn")
	cctx := memorystore.WithEntityConnection(withCancel, txnEntityMetadata)
	defer memorystore.Close(cctx)

	var (
		rn = r.GetRoundNumber()              //
		b  = block.NewBlock(mc.GetKey(), rn) //

		rnoff = mbRoundOffset(rn)   //
		nvc   = mc.NextViewChange() // we can use LFB because there is VC offset
	)
	lfmbr := mc.GetLatestFinalizedMagicBlockRound(rn) // related magic block
	if lfmbr == nil {
		return nil, errors.New("can't get lfmbr")
	}

	if nvc > 0 && rnoff >= nvc && lfmbr.StartingRound < nvc {
		logging.Logger.Error("gen_block",
			zap.String("err", "required MB missing or still not finalized"),
			zap.Int64("round offset", rnoff),
			zap.Int64("next_vc", nvc),
			zap.Int64("round", rn),
			zap.Int64("lfmbr_sr", lfmbr.StartingRound),
		)
		// TODO: correct next view change round on start, sometimes the rnoff could be
		// >= nvc when on start and the view change was not worked properly.
		return nil, common.NewError("gen_block",
			"required MB missing or still not finalized")
	}

	b.LatestFinalizedMagicBlockHash = lfmbr.Hash
	b.LatestFinalizedMagicBlockRound = lfmbr.Round

	logging.Logger.Debug("Setting LFMB round/hash for a block",
		zap.Int64("rn", r.GetRoundNumber()), zap.Int64("mc.crn", mc.GetCurrentRound()),
		zap.Int64("rnoff", mbRoundOffset(rn)), zap.Int64("nvc", mc.NextViewChange()),
		zap.Int64("r", lfmbr.Round), zap.String("h", lfmbr.Hash),
		zap.Int64("b.lfmbr", b.LatestFinalizedMagicBlockRound), zap.String("b.lfmbh", b.LatestFinalizedMagicBlockHash),
	)

	b.MinerID = node.Self.Underlying().GetKey()

	// set block round random seed
	roundSeed := r.GetRandomSeed()
	if roundSeed == 0 {
		logging.Logger.Error("Round seed reset to zero", zap.Int64("round", rn))
		return nil, common.NewError("gen_block", "round seed reset to zero")
	}

	b.SetRoundRandomSeed(roundSeed)

	mc.SetPreviousBlock(r, b, pb)

	var (
		start             = time.Now()
		generationTimeout = time.Millisecond * time.Duration(mc.GetGenerationTimeout())

		makeBlock       bool
		generationTries int
		startLogging    time.Time
	)
	for {
		if mc.GetCurrentRound() > b.Round {
			logging.Logger.Error("generate block - round mismatch",
				zap.Int64("round", roundNumber),
				zap.Int64("current_round", mc.GetCurrentRound()))
			return nil, ErrRoundMismatch
		}

		txnCount := transaction.GetTransactionCount()
		generationTries++
		if !pb.IsStateComputed() {
			logging.Logger.Debug("GenerateRoundBlock, previous block state is not computed",
				zap.Int64("round", b.Round),
				zap.Int64("pre round", pb.Round),
				zap.Int8("prior_block_state", pb.GetStateStatus()))
		}

		//b.SetStateDB(pb, mc.GetStateDB())

		if err := mc.syncAndRetry(cctx, b, "generate block", func(ctx context.Context, waitC chan struct{}) error {
			return mc.GenerateBlock(ctx, b, makeBlock, waitC)
		}); err != nil {
			cerr, ok := err.(*common.Error)
			if ok {
				switch cerr.Code {
				case InsufficientTxns:
					for {
						delay := mc.GetRetryWaitTime()
						time.Sleep(time.Duration(delay) * time.Millisecond)
						if startLogging.IsZero() || time.Since(startLogging) > time.Second {
							startLogging = time.Now()
							logging.Logger.Info("generate block",
								zap.Int64("round", roundNumber),
								zap.Int("delay", delay),
								zap.Uint64("txn_count", txnCount),
								zap.Uint64("t.txn_count", transaction.GetTransactionCount()),
								zap.Any("error", cerr))
						}
						if mc.GetCurrentRound() > b.Round {
							logging.Logger.Error("generate block - round mismatch",
								zap.Int64("round", roundNumber),
								zap.Int64("current_round", mc.GetCurrentRound()))
							return nil, ErrRoundMismatch
						}
						if txnCount != transaction.GetTransactionCount() || time.Since(start) > generationTimeout {
							makeBlock = true
							break
						}
					}
					continue
				case RoundMismatch:
					logging.Logger.Info("generate block", zap.Error(err))
					continue
				case RoundTimeout:
					logging.Logger.Error("generate block",
						zap.Int64("round", roundNumber),
						zap.Error(err))
					return nil, err
				}
			}
			if startLogging.IsZero() || time.Since(startLogging) > time.Second {
				logging.Logger.Info("generate block", zap.Int64("round", roundNumber),
					zap.Uint64("txn_count", txnCount),
					zap.Uint64("t.txn_count", transaction.GetTransactionCount()),
					zap.Error(err))
			}
			return nil, err
		}

		//todo actually it is not a problem, since RRS can be changed only during timeout and this block can be reused
		if !areRoundAndBlockSeedsEqual(r, b) {
			logging.Logger.Error("round random seed mismatch",
				zap.Int64("round", b.Round),
				zap.Int64("round_rrs", r.GetRandomSeed()),
				zap.Int64("blk_rrs", b.GetRoundRandomSeed()))
			return nil, ErrRRSMismatch
		}

		b = mc.AddRoundBlock(r, b)
		if generationTries > 1 {
			logging.Logger.Info("generate block - multiple tries",
				zap.Int64("round", b.Round), zap.Int("tries", generationTries))
		}
		break
	}

	if r.IsVerificationComplete() {
		logging.Logger.Warn("generate block - verification complete, we are late, cancel block generation",
			zap.Int64("round", roundNumber),
			zap.Int("notarized", len(r.GetNotarizedBlocks())))
		return nil, nil
	}

	b = mc.addToRoundVerification(r, b)
	r.AddProposedBlock(b)

	go mc.SendBlock(ctx, b)
	return b, nil
}

// AddToRoundVerification - add a block to verify.
func (mc *Chain) AddToRoundVerification(ctx context.Context, mr *Round, b *block.Block) {
	if mr.IsVerificationComplete() {
		logging.Logger.Debug("handle verify block - round verification completed",
			zap.Int64("round", mr.GetRoundNumber()))
		return
	}

	//if mr.GetRandomSeed() != b.GetRoundRandomSeed() {
	//	logging.Logger.Error("handle verify block - got a block for verification with wrong random seed",
	//		zap.Int64("round", mr.GetRoundNumber()),
	//		zap.Int("roundToc", mr.GetTimeoutCount()),
	//		zap.Int("blockToc", b.RoundTimeoutCount),
	//		zap.Int64("round_rrs", mr.GetRandomSeed()),
	//		zap.Int64("block_rrs", b.GetRoundRandomSeed()))
	//	return
	//}

	logging.Logger.Info("handle verify block - added block for ticket verification",
		zap.Int64("round", b.Round),
		zap.String("block", b.Hash),
		zap.String("magic block", b.LatestFinalizedMagicBlockHash),
		zap.Int64("magic block round", b.LatestFinalizedMagicBlockRound))

	if mr.IsFinalizing() || mr.IsFinalized() {
		b.SetBlockState(block.StateVerificationRejected)
		logging.Logger.Debug("add to verification", zap.Int64("round", b.Round),
			zap.String("block", b.Hash),
			zap.Bool("finalizing", mr.IsFinalizing()),
			zap.Bool("finalized", mr.IsFinalized()))
		return
	}

	if !mc.ValidateMagicBlock(ctx, mr.Round, b) {
		b.SetBlockState(block.StateVerificationRejected)
		logging.Logger.Error("add to verification (invalid magic block)",
			zap.Int64("round", b.Round), zap.String("block", b.Hash),
			zap.String("magic_block", b.LatestFinalizedMagicBlockHash),
			zap.Int64("magic_block_round", b.LatestFinalizedMagicBlockRound))
		return
	}

	var bNode = mc.GetMiners(mr.GetRoundNumber()).GetNode(b.MinerID)
	if bNode == nil {
		b.SetBlockState(block.StateVerificationRejected)
		logging.Logger.Error("add to round verification (invalid miner)",
			zap.Int64("round", b.Round), zap.String("block", b.Hash),
			zap.String("miner_id", b.MinerID))
		return
	}

	var pr = mc.GetMinerRound(mr.Number - 1)
	if pr == nil {
		logging.Logger.Error("add to verification (prior block's verify round is nil)",
			zap.Int64("round", mr.Number-1),
			zap.String("prev_block", b.PrevHash),
			zap.Int("pb_v_tickets", b.PrevBlockVerificationTicketsSize()))
		return
	}

	if b.PrevBlock != nil {
		if b.CreationDate < b.PrevBlock.CreationDate {
			logging.Logger.Error("add to verification (creation_date out of sequence",
				zap.Int64("round", mr.Number), zap.String("block", b.Hash),
				zap.Any("creation_date", b.CreationDate),
				zap.String("prev_block", b.PrevHash),
				zap.Any("prev_creation_date", b.PrevBlock.CreationDate))
			return
		}
		mc.updatePriorBlock(mr.Round, b)
	}

	mr.AddProposedBlock(b)
	//if mc.AddRoundBlock(mr, b) != b {
	//	logging.Logger.Warn("Add round block, block already exist, merge tickets", zap.Int64("round", b.Round))
	//	// block already exist, means the verification collection worker already started.
	//	return
	//}

	mc.addToRoundVerification(mr, b)
}

func convertToBlockVerificationTickets(vts []*block.VerificationTicket, round int64, hash string) []*block.BlockVerificationTicket {
	bvts := make([]*block.BlockVerificationTicket, len(vts))
	for i, vt := range vts {
		bvts[i] = &block.BlockVerificationTicket{
			VerificationTicket: *vt,
			Round:              round,
			BlockID:            hash,
		}
	}
	return bvts
}

func (mc *Chain) getOrSetBlockNotarizing(hash string) (isNotarizing bool, finish func(result bool)) {
	finish = func(result bool) {}

	mc.nbmMutex.Lock()
	_, isNotarizing = mc.notarizingBlocksTasks[hash]
	if isNotarizing {
		mc.nbmMutex.Unlock()
		return
	}
	logging.Logger.Debug("notarizing block", zap.Int("current task size", len(mc.notarizingBlocksTasks)))

	mc.notarizingBlocksTasks[hash] = make(chan struct{})
	mc.nbmMutex.Unlock()

	finish = func(result bool) {
		mc.nbmMutex.Lock()
		task := mc.notarizingBlocksTasks[hash]
		delete(mc.notarizingBlocksTasks, hash)
		//TODO refactor cache addition
		if err := mc.notarizingBlocksResults.Add(hash, result); err != nil {
			logging.Logger.Warn("cache notarizing block result failed", zap.Error(err))
		}

		close(task)
		mc.nbmMutex.Unlock()
	}
	return
}

func (mc *Chain) getBlockNotarizationResultSync(ctx context.Context, hash string) bool {
	logging.Logger.Debug("Getting block notarization results", zap.String("hash", hash))
	mc.nbmMutex.Lock()
	c, ok := mc.notarizingBlocksTasks[hash]
	mc.nbmMutex.Unlock()
	if ok {
		select {
		case <-c:
			get, err := mc.notarizingBlocksResults.Get(hash)
			if err != nil {
				return false
			}
			return get.(bool)
		case <-ctx.Done():
			{
				return false
			}
		}
	}
	get, err := mc.notarizingBlocksResults.Get(hash)

	if err != nil {
		return false
	}
	return get.(bool)
}

// nolint: staticcheck
func (mc *Chain) updatePreviousBlockNotarization(ctx context.Context, b *block.Block, pr *Round) error {
	//we don't want to cancel previous notarization too early, previous block should be notarized often
	var cancel func()
	//nolint: staticcheck
	ctx, cancel = context.WithTimeout(common.GetRootContext(), 5*time.Second)
	defer cancel()
	pb := mc.GetPreviousBlock(ctx, b)
	if pb == nil {
		return fmt.Errorf("update_previous_notarization: previous block can't be synched")
	}

	// merge the tickets
	if pb.IsBlockNotarized() {
		logging.Logger.Debug("update prev block notarization, already notarized",
			zap.Int64("round", pb.Round),
			zap.String("block", pb.Hash))
		return nil
	}

	isNotarizing, finish := mc.getOrSetBlockNotarizing(b.PrevHash)
	if isNotarizing {
		return nil
	}

	err := mc.verifyBlockNotarizationWorker.Run(ctx, func() error {
		logging.Logger.Debug("update prev block notarization, verify tickets",
			zap.Int64("round", b.Round-1), zap.String("block", b.PrevHash))

		// reset ctx so the timeout of parent ctx would not stop the ticket verification here
		ctx = context.Background()
		if err := mc.VerifyNotarization(ctx, b.PrevHash, b.GetPrevBlockVerificationTickets(), b.Round-1, pb.LatestFinalizedMagicBlockRound); err != nil {
			logging.Logger.Error("update prev block notarization failed",
				zap.Int64("round", pr.Number), zap.String("miner_id", b.MinerID),
				zap.String("block", b.PrevHash),
				zap.Int("v_tickets", b.PrevBlockVerificationTicketsSize()),
				zap.Error(err))
			return err
		}
		return nil
	})

	if err != nil {
		finish(false)
		return err
	}

	pbvts := convertToBlockVerificationTickets(b.GetPrevBlockVerificationTickets(), b.Round-1, b.PrevHash)
	pr.AddVerificationTickets(pbvts)

	pr.CancelVerification()
	pb.MergeVerificationTickets(b.GetPrevBlockVerificationTickets())
	mc.AddNotarizedBlockToRound(pr, pb)
	finish(true)
	return nil
}

func (mc *Chain) addToRoundVerification(mr *Round, b *block.Block) *block.Block {
	logging.Logger.Info("adding block to verify",
		zap.Int64("round", b.Round),
		zap.String("block", b.Hash),
		zap.String("prev_block", b.PrevHash),
		zap.String("state_hash", util.ToHex(b.ClientStateHash)),
		zap.Float64("weight", b.Weight()))
	//mc.StartVerification(ctx, mr)
	b = mc.AddBlock(b)
	mr.AddBlockToVerify(b)
	return b
}

func (mc *Chain) StartVerification(ctx context.Context, mr *Round) {
	//we use one minute timeout here for emergency case. normally context should be cancelled due to network progress
	vctx := mr.StartVerificationBlockCollection(ctx)
	gen := mc.GetGenerators(mr)

	if vctx != nil && len(gen) > 0 && mr.IsVRFComplete() {

		//mr.delta = mc.GetBlockProposalWaitTime(mr.Round)
		waitTime := mc.GetBlockProposalWaitTime(mr.Round)
		minerNT := time.Duration(int64(gen[0].GetLargeMessageSendTime()/1000000)) * time.Millisecond
		if minerNT >= waitTime {
			mr.delta = time.Millisecond
		} else {
			mr.delta = waitTime - minerNT
		}
		logging.Logger.Info("Starting round verification", zap.Int64("round", mr.Number), zap.Duration("delta", mr.delta))
		mr.SetPhase(round.Verify)
		go mc.CollectBlocksForVerification(vctx, mr)
	}
}

/*GetBlockProposalWaitTime - get the time to wait for the block proposals of the given round */
func (mc *Chain) GetBlockProposalWaitTime(r round.RoundI) time.Duration {
	if mc.BlockProposalWaitMode() == chain.BlockProposalWaitDynamic {
		return mc.computeBlockProposalDynamicWaitTime(r)
	}
	return mc.BlockProposalMaxWaitTime()
}

func (mc *Chain) computeBlockProposalDynamicWaitTime(r round.RoundI) time.Duration {
	mb := mc.GetMagicBlock(r.GetRoundNumber())
	medianTime := mb.Miners.GetMedianNetworkTime()
	generators := mc.GetGenerators(r)
	for _, g := range generators {
		sendTime := g.GetLargeMessageSendTime()
		if sendTime < medianTime {
			return time.Duration(int64(math.Round(sendTime)/1000000)) * time.Millisecond
		}
	}
	/*
		medianTimeMS := time.Duration(int64(math.Round(medianTime)/1000000)) * time.Millisecond
		if medianTimeMS > mc.BlockProposalMaxWaitTime {
			return medianTimeMS
		}*/
	return mc.BlockProposalMaxWaitTime()
}

/*CollectBlocksForVerification - keep collecting the blocks till timeout and then start verifying */
func (mc *Chain) CollectBlocksForVerification(ctx context.Context, r *Round) {
	verifyAndSend := func(ctx context.Context, r *Round, b *block.Block) bool {
		logging.Logger.Debug("verifyAndSend - started", zap.String("block", b.Hash))
		b.SetBlockState(block.StateVerificationAccepted)
		miner := mc.GetMiners(r.GetRoundNumber()).GetNode(b.MinerID)
		if miner == nil || miner.ProtocolStats == nil {
			logging.Logger.Error("verifyAndSend -- failed miner",
				zap.Int64("round", r.Number), zap.String("block", b.Hash),
				zap.String("miner", b.MinerID))
			b.SetBlockState(block.StateVerificationFailed)
			return false
		}
		minerStats := miner.ProtocolStats.(*chain.MinerStats)

		bvt, err := mc.VerifyRoundBlock(ctx, r, b)
		if err != nil {
			b.SetVerificationStatus(block.VerificationFailed)
			logging.Logger.Debug("verifyAndSend - got error on verify round block",
				zap.String("phase", round.GetPhaseName(r.GetPhase())), zap.Error(err))
			switch err {
			case context.DeadlineExceeded:
				if !r.isVerificationComplete() {
					b.SetBlockState(block.StateVerificationFailed)
					logging.Logger.Error("verifyAndSend - deadline exceed without round verification completed",
						zap.Int64("round", b.Round), zap.Error(err))
				}
				return false
			case context.Canceled:
				if !r.isVerificationComplete() {
					b.SetBlockState(block.StateVerificationFailed)
					logging.Logger.Debug("verifyAndSend - canceled without round verification completed",
						zap.Int64("round", b.Round), zap.Error(err))
				}
				return false
			case ErrRoundMismatch:
				if !r.isVerificationComplete() {
					b.SetBlockState(block.StateVerificationFailed)
					logging.Logger.Warn("verifyAndSend failed, verification cancelled (round mismatch)",
						zap.Int64("round", r.Number),
						zap.String("block", b.Hash),
						zap.Int64("current_round", mc.GetCurrentRound()))
				}
				return false
			default:
				b.SetBlockState(block.StateVerificationFailed)
				minerStats.VerificationFailures++
				logging.Logger.Error("verifyAndSend failed",
					zap.Int64("round", r.Number),
					zap.String("block", b.Hash),
					zap.Error(err))
			}
			return false
		}
		b.SetBlockState(block.StateVerificationSuccessful)

		bnb := r.GetBestRankedNotarizedBlock()
		if bnb == nil || bnb.Hash == b.Hash {
			logging.Logger.Info("verifyAndSend - sending verification ticket", zap.Int64("round", r.Number), zap.String("block", b.Hash),
				zap.Int("block_rank", b.RoundRank), zap.Int64("RRS", b.RoundRandomSeed))
			go mc.SendVerificationTicket(ctx, b, bvt)
			r.SetOwnVerificationTicket(bvt)
		}
		if bnb == nil {
			r.Block = b
			mc.ProcessVerifiedTicket(ctx, r, b, &bvt.VerificationTicket)
		}
		numGenerators := mc.GetGeneratorsNumOfRound(r.GetRoundNumber())
		if b.RoundRank >= numGenerators || b.RoundRank < 0 {
			logging.Logger.Warn("round rank is invalid or greater than num_generators",
				zap.String("hash", b.Hash), zap.Int64("round", b.Round),
				zap.Int("round_rank", b.RoundRank),
				zap.Int("num_generators", numGenerators))
		} else {
			if numGenerators > len(minerStats.VerificationTicketsByRank) {
				newRankStat := make([]int64, numGenerators)
				copy(newRankStat, minerStats.VerificationTicketsByRank)
				minerStats.VerificationTicketsByRank = newRankStat
			}
			minerStats.VerificationTicketsByRank[b.RoundRank]++
		}
		logging.Logger.Debug("verifyAndSend - finished successfully",
			zap.Int64("round", b.Round),
			zap.String("block", b.Hash))
		return true
	}
	var sendVerification = false
	var blocks = make([]*block.Block, 0, 512)
	initiateVerification := func() {
		logging.Logger.Info("Started main verification loop", zap.Int64("round", r.Number))
		// Sort the accumulated blocks by the rank and process them
		blocks = r.GetBlocksByRank(blocks)
		for _, b := range blocks {
			//don't know why we change rrs and rank inside AddRoundBlock
			r.AddProposedBlock(b)
			if mc.AddRoundBlock(r, b) != b {
				logging.Logger.Warn("Add round block, block already exist", zap.Int64("round", b.Round))
				// block already exist, means the verification collection worker already started.
				// TODO do we really need to return false here?
			}
		}

		// Keep verifying all the blocks collected so far in the best rank order
		// till the first successful verification.
		for _, b := range blocks {
			if ctx.Err() != nil {
				break
			}
			if verifyAndSend(ctx, r, b) {
				break
			}
		}
		for _, b := range blocks {
			if b.GetBlockState() == block.StateVerificationPending {
				b.SetBlockState(block.StateVerificationRejected)
			}
		}
		sendVerification = true
	}
	var blockTimeTimer = time.NewTimer(r.delta)
	var verifiedBlocks []*block.Block
	for {
		select {
		case <-ctx.Done():
			logging.Logger.Info("verification_complete", zap.Int64("round", r.Number),
				zap.Int("verified_blocks", len(verifiedBlocks)))
			bRank := -1
			if r.Block != nil {
				bRank = r.Block.RoundRank
				logging.Logger.Debug("verification_complete", zap.Int("top_verified_rank", r.Block.RoundRank))
			}
			for _, b := range blocks {
				bs := b.GetBlockState()
				if bRank != 0 && bRank != b.RoundRank {
					logging.Logger.Info("failing block", zap.Int64("round", r.Number), zap.String("block", b.Hash), zap.Int("block_rank", b.RoundRank), zap.Int("best_rank", bRank), zap.Int8("block_state", bs))
				}
				if bs == block.StateVerificationPending || bs == block.StateVerificationAccepted {
					b.SetBlockState(block.StateVerificationFailed)
				}
			}
			return
		case <-blockTimeTimer.C:
			initiateVerification()
		case b := <-r.GetBlocksToVerifyChannel():
			r.AddProposedBlock(b)
			b = mc.AddRoundBlock(r, b)

			if sendVerification {
				// Is this better than the current best block
				if r.Block == nil || r.Block.RoundRank >= b.RoundRank {
					b.SetBlockState(block.StateVerificationPending)
					logging.Logger.Debug("verify block - try to verify and send",
						zap.Int64("round", b.Round),
						zap.String("block", b.Hash),
						zap.Bool("has verified block", r.Block != nil),
						zap.Int("block round rank", b.RoundRank))
					if verifyAndSend(ctx, r, b) {
						verifiedBlocks = append(verifiedBlocks, b)
					}
				} else {
					logging.Logger.Debug("verify block - rejected",
						zap.Int64("round", b.Round),
						zap.String("block", b.Hash))
					b.SetBlockState(block.StateVerificationRejected)
				}
			} else { // Accumulate all the blocks into this array till the BlockTime timeout
				logging.Logger.Debug("verify block - pending",
					zap.Int64("round", b.Round),
					zap.String("block", b.Hash))
				b.SetBlockState(block.StateVerificationPending)
				blocks = append(blocks, b)
			}
		}
	}
}

/*VerifyRoundBlock - given a block is verified for a round*/
func (mc *Chain) VerifyRoundBlock(ctx context.Context, r round.RoundI, b *block.Block) (*block.BlockVerificationTicket, error) {
	logging.Logger.Debug("verify round block", zap.Int64("round", r.GetRoundNumber()), zap.String("block", b.Hash))
	if !mc.CanShardBlocks(r.GetRoundNumber()) {
		return nil, common.NewError("fewer_active_sharders", "number of active sharders not sufficient")
	}
	if !mc.CanReplicateBlock(b) {
		return nil, common.NewError("fewer_active_replicators", "number of active replicators not sufficient")
	}
	if mc.GetCurrentRound() != r.GetRoundNumber() {
		return nil, ErrRoundMismatch
	}

	if b.GetRoundRandomSeed() == 0 {
		return nil, common.NewErrorf("verify_round_block", "block with no RRS, %d, %s", b.Round, b.Hash)
	}

	if !areRoundAndBlockSeedsEqual(r, b) {
		return nil, common.NewError("seed_mismatch", "block RRS mismatch")
	}

	if !mc.ValidGenerator(r, b) {
		return nil, common.NewError("verify_round_block", "not a valid generator")
	}

	if b.MinerID == node.Self.Underlying().GetKey() {
		return mc.SignBlock(ctx, b)
	}

	logging.Logger.Debug("verify round block - started verification", zap.String("block", b.Hash))
	bvt, err := mc.VerifyBlock(ctx, b)
	logging.Logger.Debug("verify round block - finished verification", zap.String("block", b.Hash))
	if err != nil {
		b.SetVerificationStatus(block.VerificationFailed)
		return nil, err
	}

	if b.PrevBlock != nil && b.PrevBlock.IsBlockNotarized() {
		mc.updatePriorBlock(r, b)
		return bvt, nil
	}

	if !mc.getBlockNotarizationResultSync(ctx, b.PrevHash) {
		b.SetVerificationStatus(block.VerificationFailed)
		return nil, common.NewError("verify_round_block", "Verification tickets of previous block are not valid")
	}

	return bvt, nil
}

func (mc *Chain) updatePriorBlock(r round.RoundI, b *block.Block) {
	pb := b.PrevBlock
	mc.MergeVerificationTickets(pb, b.GetPrevBlockVerificationTickets())
	pr := mc.GetMinerRound(pb.Round)
	if pr == nil {
		logging.Logger.Error("update prior block - previous round not present",
			zap.Int64("round", r.GetRoundNumber()),
			zap.String("block", b.Hash),
			zap.String("prev_block", b.PrevHash))
	}
	if pb.VerificationTicketsSize() > b.PrevBlockVerificationTicketsSize() {
		b.SetPrevBlockVerificationTickets(pb.GetVerificationTickets())
	}
}

/*ProcessVerifiedTicket - once a verified ticket is received, do further processing with it */
func (mc *Chain) ProcessVerifiedTicket(ctx context.Context, r *Round, b *block.Block, vt *block.VerificationTicket) {

	var notarized = b.IsBlockNotarized()

	// NOTE: We keep collecting verification tickets even if a block is
	// notarized. Knowing who all know about a block can be used to optimize
	// other parts of the protocol.
	if !mc.AddVerificationTicket(b, vt) {
		return
	}

	if notarized {
		// block was notarized before adding new tickets
		logging.Logger.Info("Block is notarized", zap.Int64("round", r.Number),
			zap.String("block", b.Hash))
		return
	}

	mc.checkBlockNotarization(ctx, r, b, true)
}

func (mc *Chain) checkBlockNotarization(ctx context.Context, r *Round, b *block.Block, broadcast bool) bool {
	if !b.IsBlockNotarized() {
		logging.Logger.Info("checkBlockNotarization -- block is not Notarized. Returning",
			zap.Int64("round", b.Round),
			zap.String("block hash", b.Hash))
		return false
	}
	if !mc.AddNotarizedBlock(r, b) {
		logging.Logger.Warn("checkBlockNotarization -- block already notarized before",
			zap.Int64("round", b.Round),
			zap.String("block", b.Hash))
		return true
	}

	seed := b.GetRoundRandomSeed()
	if seed == 0 {
		logging.Logger.Error("checkBlockNotarization -- block random seed is 0", zap.Int64("round", b.Round))
		return false
	}

	if broadcast {
		go mc.SendNotarization(context.Background(), b)
	}

	logging.Logger.Debug("check block notarization - block notarized",
		zap.Int64("round", b.Round), zap.String("block", b.Hash))

	// start next round if not ahead of sharders
	//TODO implement round centric context, that is cancelled when transition to the next happens
	mc.ProgressOnNotarization(r)
	return true
}

func (mc *Chain) moveToNextRoundNotAheadImpl(ctx context.Context, r *Round, beforeStartNextRound func()) {
	r.SetPhase(round.Complete)
	var (
		rn = r.GetRoundNumber()
		pr = mc.GetMinerRound(rn - 1)
	)

	if pr != nil && !pr.IsFinalizing() && !pr.IsFinalized() {
		mc.finalizeRound(ctx, pr) // finalize the previous round
	}

	if !mc.waitNotAhead(ctx, rn) {
		logging.Logger.Debug("start next round not ahead -- terminated",
			zap.Int64("round", rn))
		return // terminated
	}

	beforeStartNextRound()

	//TODO start if not started, atm we  resend vrf share here
	mc.StartNextRound(ctx, r)
}

// MergeNotarization - merge a notarization.
func (mc *Chain) MergeNotarization(ctx context.Context, r *Round, b *block.Block, vts []*block.VerificationTicket) error {
	return mc.BlockTicketsVerifyWithLock(ctx, b.Hash, func() error {
		if b.IsBlockNotarized() {
			logging.Logger.Debug("merge notarization, already notarized",
				zap.Int64("round", b.Round),
				zap.String("block", b.Hash))
			return nil
		}

		if err := mc.VerifyTickets(ctx, b.Hash, vts, r.GetRoundNumber()); err != nil {
			return err
		}

		mc.MergeVerificationTickets(b, vts)

		mc.checkBlockNotarization(ctx, r, b, false)
		return nil
	})
}

/*AddNotarizedBlock - add a notarized block for a given round */
func (mc *Chain) AddNotarizedBlock(r *Round, b *block.Block) bool {
	ctx, cancel := context.WithTimeout(common.GetRootContext(), 30*time.Second)
	defer cancel()
	mc.AddNotarizedBlockToRound(r, b)
	mc.UpdateNodeState(b)

	if !b.IsStateComputed() {
		logging.Logger.Info("add notarized block - computing state",
			zap.Int64("round", b.Round), zap.String("block", b.Hash))
		if err := mc.syncAndRetry(ctx, b, "add notarized block", func(ctx context.Context, waitC chan struct{}) error {
			return mc.ComputeState(ctx, b, waitC)
		}); err != nil {
			logging.Logger.Error("add notarized block failed", zap.Error(err),
				zap.Int64("round", b.Round), zap.String("block", b.Hash))
			return false
		}
	}

	b.SetBlockState(block.StateNotarized)
	return true
}

/*CancelRoundVerification - cancel verifications happening within a round */
func (mc *Chain) CancelRoundVerification(ctx context.Context, r *Round) {
	r.CancelVerification() // No need for further verification of any blocks
	r.TryCancelBlockGeneration()
}

// SyncFetchFinalizedBlockFromSharders fetches FB from sharders by hash.
// It used by miner to get FB by LFB ticket in restart round.
func (mc *Chain) SyncFetchFinalizedBlockFromSharders(ctx context.Context,
	hash string) (fb *block.Block) {

	var b, err = mc.GetBlock(ctx, hash)
	if b != nil && err == nil {
		return b // already have the block
	}

	type blockConsensus struct {
		*block.Block
		consensus int
	}

	var (
		mb       = mc.GetCurrentMagicBlock()
		sharders = mb.Sharders
		fbs      = make([]*blockConsensus, 0, 1)

		mx sync.Mutex
	)

	var handler = func(_ context.Context, entity datastore.Entity) (
		resp interface{}, err error) {

		var fb, ok = entity.(*block.Block)
		if !ok {
			return nil, datastore.ErrInvalidEntity
		}

		if err = fb.Validate(ctx); err != nil {
			logging.Logger.Error("FB from sharder - invalid",
				zap.Int64("round", fb.Round),
				zap.String("block", fb.Hash),
				zap.Error(err))
			return nil, err
		}

		err = mc.VerifyBlockNotarization(ctx, fb)
		if err != nil {
			logging.Logger.Error("FB from sharder - notarization failed",
				zap.Int64("round", fb.Round),
				zap.String("block", fb.Hash), zap.Error(err))
			return nil, err
		}

		mc.getOrCreateRound(ctx, fb.Round)

		mx.Lock()
		defer mx.Unlock()
		for i, b := range fbs {
			if b.Hash == fb.Hash {
				fbs[i].consensus++
				return fb, nil
			}
		}
		fbs = append(fbs, &blockConsensus{
			Block:     fb,
			consensus: 1,
		})

		return fb, nil
	}

	sharders.RequestEntityFromAll(ctx, chain.FBRequestor, nil, handler)

	// highest (the first sorting order), most popular (the second order)
	sort.Slice(fbs, func(i int, j int) bool {
		return fbs[i].Round >= fbs[j].Round ||
			fbs[i].consensus > fbs[j].consensus

	})

	if len(fbs) == 0 {
		logging.Logger.Error("FB from sharders -- no block given",
			zap.String("hash", hash))
		return nil
	}

	return fbs[0].Block
}

// GetNextRoundTimeoutTime returns time in milliseconds.
func (mc *Chain) GetNextRoundTimeoutTime(ctx context.Context) int {

	ssft := int(math.Ceil(chain.SteadyStateFinalizationTimer.Mean() / 1000000))
	tick := mc.RoundTimeoutSofttoMin()
	if tick < mc.RoundTimeoutSofttoMult()*ssft {
		tick = mc.RoundTimeoutSofttoMult() * ssft
	}
	logging.Logger.Info("nextTimeout", zap.Int("tick", tick))
	return tick
}

// handleRoundTimeout handle timeouts appropriately.
func (mc *Chain) handleRoundTimeout(ctx context.Context, round int64) {
	// 	mmb = mc.GetMagicBlock(rn + chain.ViewChangeOffset + 1)
	// 	cmb = mc.GetMagicBlock(rn)

	// 	selfNodeKey = node.Self.Underlying().GetKey()
	// )

	// // miner should be member of current magic block; also, we have to call the
	// // restartRound on last round of MB the miner is not member (on joining)
	// if cmb == nil || !cmb.Miners.HasNode(selfNodeKey) &&
	// 	!mmb.Miners.HasNode(selfNodeKey) {

	// 	return
	// }

	var r = mc.GetMinerRound(round)

	if r.GetSoftTimeoutCount() == mc.RoundRestartMult() {
		logging.Logger.Info("triggering restartRound",
			zap.Int64("round", r.GetRoundNumber()))
		mc.restartRound(ctx, round)
		return
	}

	logging.Logger.Info("triggering handleNoProgress",
		zap.Int64("round", r.GetRoundNumber()))
	mc.handleNoProgress(ctx, round)
	r.IncSoftTimeoutCount()
}

func (mc *Chain) handleNoProgress(ctx context.Context, rn int64) {
	r := mc.GetMinerRound(rn)
	proposals := r.GetProposedBlocks()
	if len(proposals) > 0 { // send the best block to the network
		b := r.Block
		if b != nil {
			if mc.GetRoundTimeoutCount() <= 10 {
				logging.Logger.Info("sending the best block to the network",
					zap.Int64("round", b.Round), zap.String("block", b.Hash),
					zap.Int("rank", b.RoundRank))
			}
			lfmbr := mc.GetLatestFinalizedMagicBlockRound(rn) // related magic block
			if lfmbr == nil {
				logging.Logger.Error("can't get lfmb")
				return
			}
			if lfmbr.Hash != b.LatestFinalizedMagicBlockHash {
				logging.Logger.Error("handleNoProgress mismatch latest finalized magic block",
					zap.Int64("round", b.Round),
					zap.String("block miner", b.MinerID),
					zap.String("lfmbr hash", lfmbr.Hash),
					zap.String("block lfmbr hash", b.LatestFinalizedMagicBlockHash),
					zap.Int64("lfmbr starting round", lfmbr.Round),
					zap.Int64("block lfmbr starting round", b.LatestFinalizedMagicBlockRound))
			} else {
				logging.Logger.Debug("handleNoProgress match latest finalized magic block",
					zap.String("lfmbr hash", lfmbr.Hash),
					zap.Int64("lfmbr round", lfmbr.Round))
			}
			logging.Logger.Info("Sent proposal in handle NoProgress")
			go mc.sendBlock(context.Background(), b)

			if r.OwnVerificationTicket() != nil {
				if mc.GetRoundTimeoutCount() <= 10 {
					logging.Logger.Info("Sending verification ticket in handle NoProgress",
						zap.Int64("round", r.Number), zap.String("block", b.Hash))
				}
				go mc.SendVerificationTicket(ctx, b, r.OwnVerificationTicket())
			}
		}
	}

	if r.VrfShare() != nil {
		// send VRF share in goroutine, use new context, the old one will be canceled soon
		// after the function is returned
		go mc.SendVRFShare(context.Background(), r.VrfShare().Clone())
		logging.Logger.Info("Sent vrf shares in handle NoProgress")
	} else {
		logging.Logger.Info("Did not send vrf shares as it is nil", zap.Int64("round_num", r.GetRoundNumber()))
	}

	switch crt := mc.GetRoundTimeoutCount(); {
	case crt < 10:
		logging.Logger.Info("handleNoProgress",
			zap.Int64("round", mc.GetCurrentRound()),
			zap.Int64("count_round_timeout", crt),
			zap.Int("num_vrf_share", len(r.GetVRFShares())))
	case crt == 10:
		logging.Logger.Error("handleNoProgress (no further timeout messages will be displayed)",
			zap.Int64("round", mc.GetCurrentRound()),
			zap.Int64("count_round_timeout", crt),
			zap.Int("num_vrf_share", len(r.GetVRFShares())))
		//TODO: should have a means to send an email/SMS to someone or something like that
	}

}

func (mc *Chain) kickFinalization(ctx context.Context) {

	logging.Logger.Info("restartRound->kickFinalization")

	// kick all blocks finalization

	var lfb = mc.GetLatestFinalizedBlock()

	// don't kick more than 5 blocks at once
	e := mc.GetCurrentRound() // loop variables
	var count int
	i := lfb.Round

	for i < e && count < 5 {
		var mr = mc.GetMinerRound(i)
		if mr == nil || mr.IsFinalized() {
			if mr != nil {
				logging.Logger.Info("restartRound->kickFinalization continued",
					zap.Int64("miner round", mr.Number),
					zap.Bool("miner is finalized", mr.IsFinalized()))
			} else {
				logging.Logger.Info("restartRound->kickFinalization continued. Miner Round is nil")
			}
			i++
			count++
			continue // skip finalized blocks, skip nil miner rounds
		}
		logging.Logger.Info("restartRound->kickFinalization:",
			zap.Int64("round", mr.GetRoundNumber()))
		go mc.FinalizeRound(mr.Round) // kick finalization again
		i++
		count++
	}
}

// push notarized blocks to sharders again (the far ahead state)
func (mc *Chain) kickSharders(ctx context.Context) {

	var (
		lfb = mc.GetLatestFinalizedBlock()
		tk  = mc.GetLatestLFBTicket(ctx)
	)

	if lfb == nil || tk == nil {
		return
	}

	if lfb.Round <= tk.Round {
		mc.kickFinalization(ctx)
		return
	}

	logging.Logger.Info("restartRound->kickSharders: kick sharders")

	// don't kick more than 5 blocks at once
	var (
		s, c, i = tk.Round, mc.GetCurrentRound(), 0 // loop variables
		ahead   = config.GetLFBTicketAhead()
	)
	for ; s < c && i < ahead; s, i = s+1, i+1 {
		var mr = mc.GetMinerRound(s)
		// send block to sharders again, if missing sharders side
		if mr != nil && mr.Block != nil && mr.Block.IsBlockNotarized() &&
			(mr.Block.GetStateStatus() == block.StateSuccessful ||
				mr.Block.GetStateStatus() == block.StateSynched ||
				mr.Block.GetBlockState() == block.StateNotarized) {

			logging.Logger.Info("restartRound->kickSharders: kick sharder FB",
				zap.Int64("round", mr.GetRoundNumber()))
			go mc.ForcePushNotarizedBlock(common.GetRootContext(), mr.Block)
		}
	}
}

func (mc *Chain) getRoundRandomSeed(rn int64) (seed int64) {
	var mr = mc.GetMinerRound(rn)
	if mr == nil {
		return // zero, no seed
	}
	return mr.GetRandomSeed() // can be zero too
}

func (mc *Chain) restartRound(ctx context.Context, rn int64) {
	mc.BumpLFBTicket(ctx)
	mc.sendRestartRoundEvent(ctx) // trigger restart round event

	mc.IncrementRoundTimeoutCount()
	var r = mc.GetMinerRound(rn)

	switch crt := mc.GetRoundTimeoutCount(); {
	case crt < 10:
		logging.Logger.Error("restartRound - round timeout occurred",
			zap.Int64("round", rn), zap.Int64("count", crt),
			zap.Int("num_vrf_share", len(r.GetVRFShares())),
			zap.String("current_phase", round.GetPhaseName(r.GetPhase())),
			zap.Bool("is_finalized", r.IsFinalized()))

	case crt == 10:
		logging.Logger.Error("restartRound - round timeout occurred (no further"+
			" timeout messages will be displayed)",
			zap.Int64("round", rn), zap.Int64("count", crt),
			zap.Int("num_vrf_share", len(r.GetVRFShares())),
			zap.String("current_phase", round.GetPhaseName(r.GetPhase())),
			zap.Bool("is_finalized", r.IsFinalized()))

		// TODO: should have a means to send an email/SMS to someone or
		// something like that
	}
	mc.RoundTimeoutsCount++

	// get LFMB and LFB from sharders
	var (
		isAhead = mc.isAheadOfSharders(ctx, rn)
		lfb     = mc.GetLatestFinalizedBlock()
	)

	if isAhead {
		mc.kickSharders(ctx) // not updated, resend latest HNB we know for this round
	}

	//we can still miss the notarization for current round, so we try to request it and make progress
	// check out corresponding not. block
	var xrhnb = r.GetHeaviestNotarizedBlock()
	if xrhnb != nil {
		logging.Logger.Debug("restartRound - round is notarized",
			zap.Int64("round", r.Number),
			zap.Int64("lfb_round", lfb.Round))
		mc.ProgressOnNotarization(r)
		return // skip notarized rounds <=== [continue loop]
	}

	//if network made progress and current miner is still stuck try to get next to lfb block, which is notarized,
	//else try to get notarized block for current round
	if lfb.Round+1 > r.Number {
		r = mc.getOrCreateRound(ctx, lfb.Round+1)
	}
	// fetch from remote
	xrhnb = mc.GetHeaviestNotarizedBlock(ctx, r)
	if xrhnb == nil {
		logging.Logger.Debug("restartRound - could not get HNB",
			zap.Int64("round", r.GetRoundNumber()),
			zap.Int64("lfb_round", lfb.Round),
			zap.Int("soft_timeout", r.GetSoftTimeoutCount()))

		if r.GetSoftTimeoutCount() >= mc.RoundRestartMult() {
			logging.Logger.Debug("restartRound - could not get HNB, redo vrf share",
				zap.Int64("round", r.GetRoundNumber()),
				zap.Int64("lfb_round", lfb.Round))
			if err := r.Restart(); err == round.CompleteRoundRestartError {
				logging.Logger.Warn("Attempt to restart already notarized round, skip this attempt")
				return
			}
			r.IncrementTimeoutCount(mc.getRoundRandomSeed(r.Number-1), mc.GetMiners(r.Number))
			mc.RedoVrfShare(ctx, r)
		}
		return // the round has restarted <===================== [exit loop]
	}
	if r.IsVRFComplete() && xrhnb.GetRoundRandomSeed() != r.GetRandomSeed() {
		logging.Logger.Info("Received notarization with different RRS, use it and make transition")
	}
	mc.ProgressOnNotarization(r)
}

func (mc *Chain) startProtocolOnLFB(ctx context.Context, lfb *block.Block) (
	mr *Round) {

	if lfb == nil {
		return // nil
	}

	mc.BumpTicket(ctx, lfb)

	// we can't compute state in the start protocol
	if err := mc.InitBlockState(lfb); err != nil {
		logging.Logger.Error("start protocol on LFB - init block state failed",
			zap.Int64("round", lfb.Round),
			zap.String("block", lfb.Hash),
			zap.Error(err))
		lfb.SetStateStatus(0)
	}

	logging.Logger.Info("start protocoal on LFB - set lfb", zap.Int64("round", lfb.Round),
		zap.Any("status", lfb.GetStateStatus()))

	mc.SetLatestFinalizedBlock(ctx, lfb)
	logging.Logger.Info("start protocoal on LFB - set lfb",
		zap.Int64("round", lfb.Round))
	return mc.GetMinerRound(lfb.Round)
}

func StartProtocol(ctx context.Context, gb *block.Block) {

	var (
		mc = GetMinerChain()
		mr *Round
	)

	if err := mc.LoadLatestBlocksFromStore(ctx); err != nil {
		logging.Logger.Error(fmt.Sprintf("can't load latest blocks from store, err: %v", err))
		// return
	}

	mc.loadLatestFinalizedMagicBlockFromStore(ctx)

	mc.BumpLFBTicket(ctx)

	lfb := mc.GetLatestFinalizedBlock()
	if lfb != nil {
		mr = mc.startProtocolOnLFB(ctx, lfb)
	} else {
		// start on genesis block
		mc.BumpTicket(ctx, gb)
		var r = round.NewRound(gb.Round)
		mr = mc.CreateRound(r)
		mr = mc.AddRound(mr).(*Round)
	}
	var nr = mc.StartNextRound(ctx, mr)
	logging.Logger.Info("starting the blockchain ...", zap.Int64("round", nr.Number))
}

func (mc *Chain) setupLoadedMagicBlock(mb *block.MagicBlock) (err error) {
	return mc.UpdateMagicBlock(mb)
}

// LoadMagicBlocksAndDKG from store to start working. It loads MB and previous
// MB (regardless Miner SC rejection) and related DKG and sets them. It can
// return the latest MB and an error related to DKG or previous MB/DKG loading.
// The method can write INFO logs that doesn't really mean an error. Since, the
// method is optimistic and tries to load latest and previous MBs and related
// DKGs. But in a normal case miner can have or haven't the MBs and the DKGs.
func (mc *Chain) LoadMagicBlocksAndDKG(ctx context.Context) {

	// current MB
	var (
		current *block.MagicBlock
		err     error
	)

	lfbr, err := mc.LoadLFBRound()
	if err != nil {
		logging.Logger.Error("load_lfb - could not load lfb from state DB", zap.Error(err))
		return
	}

	logging.Logger.Debug("load_lfb - load magic blocks and dkg",
		zap.Int64("round", lfbr.Round),
		zap.String("block", lfbr.Hash),
		zap.Int64("mb_round", lfbr.MagicBlockNumber))

	current, err = LoadLatestMB(ctx, lfbr.Round, lfbr.MagicBlockNumber)
	if err != nil {
		logging.Logger.Error("load_mbs_and_dkg -- loading the latest MB",
			zap.Error(err))
		return // can't continue
	}

	logging.Logger.Debug("[mvc] load current MB",
		zap.Int64("mb number", current.MagicBlockNumber),
		zap.Int64("mb sr", current.StartingRound),
		zap.String("mb hash", current.Hash))

	if err = mc.setupLoadedMagicBlock(current); err != nil {
		logging.Logger.Info("load_mbs_and_dkg -- updating previous MB",
			zap.Error(err))
		return // can't continue
	}
	mc.SetMagicBlock(current)
	if err = mc.SetDKGSFromStore(ctx, current); err != nil {
		logging.Logger.Info("load_mbs_and_dkg -- loading previous DKG",
			zap.Error(err))
	}

	// check if there are new MB which is possible, load them into memory store if any
	newMBNum := lfbr.MagicBlockNumber + 1
	newMB, err := LoadMagicBlock(ctx, strconv.FormatInt(newMBNum, 10))
	if err != nil {
		logging.Logger.Debug("load_mbs_and_dkg -- see no newer MB")
		return
	}

	if err := mc.SetDKGSFromStore(ctx, newMB); err != nil {
		logging.Logger.Info("load_mbs_and_dkg -- see no newer DKG")
		return
	}

	logging.Logger.Debug("load_mbs_and_dkg -- load newer MB and DKG",
		zap.Int64("mb number", newMB.MagicBlockNumber),
		zap.Int64("mb sr", newMB.StartingRound),
		zap.String("mb hash", newMB.Hash))

	mc.SetMagicBlock(newMB)

	// everything is OK
}

func (mc *Chain) WaitForActiveSharders(ctx context.Context) error {
	var (
		oldRound = mc.GetCurrentRound()
		lmb      = mc.GetCurrentMagicBlock()
	)

	defer func() {
		mc.SetCurrentRound(oldRound)
		chain.ResetStatusMonitor(lmb.StartingRound)
	}()

	// we can't use the lmb.StartingRound as current round, since the lmb
	// is saved in store but can be rejected by Miner SC if the LMB is not
	// view changed yet (451-502 rounds)

	if mc.CanShardBlocksSharders(lmb.Sharders) {
		return nil
	}

	var waitingSharders = make([]string, 0, lmb.Sharders.Size())
	for _, nodeSharder := range lmb.Sharders.CopyNodesMap() {
		waitingSharders = append(waitingSharders,
			fmt.Sprintf("id: %s; n2nhost: %s ", nodeSharder.ID, nodeSharder.N2NHost))
	}

	logging.Logger.Debug("waiting for sharders",
		zap.Int64("latest_magic_block_round", lmb.StartingRound),
		zap.Int64("latest_magic_block_number", lmb.MagicBlockNumber),
		zap.Strings("sharders", waitingSharders))

	var ticker = time.NewTicker(5 * chain.DELTA)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ts := <-ticker.C:
			if mc.CanShardBlocksSharders(lmb.Sharders) {
				return nil
			}
			logging.Logger.Info("Waiting for Sharders.", zap.Time("ts", ts),
				zap.Strings("sharders", waitingSharders))
			lmb.Sharders.OneTimeStatusMonitor(ctx, lmb.StartingRound) // just mark 'em active
		}
	}
}
