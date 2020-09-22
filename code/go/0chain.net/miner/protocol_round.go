package miner

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"sync"
	"time"

	metrics "github.com/rcrowley/go-metrics"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/memorystore"
	"0chain.net/core/util"

	. "0chain.net/core/logging"
	"go.uber.org/zap"
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
		println("don't add VRFS: not the active member")
		return
	}

	var dkg = mc.GetDKG(rn)
	if dkg == nil {
		Logger.Error("add_my_vrf_share -- DKG is nil, my VRF share is not added",
			zap.Any("round", rn))
		return
	}

	var (
		vrfs = &round.VRFShare{}
		err  error
	)
	vrfs.Round = rn
	vrfs.RoundTimeoutCount = r.GetTimeoutCount()

	if vrfs.Share, err = mc.GetBlsShare(ctx, r.Round); err != nil {
		Logger.Error("add_my_vrf_share", zap.Any("round", vrfs.Round),
			zap.Any("round_timeout", vrfs.RoundTimeoutCount),
			zap.Int64("dkg_starting_round", dkg.StartingRound),
			zap.Int64("dkg_mb_number", dkg.MagicBlockNumber),
			zap.String("pr_seed", strconv.FormatInt(pr.GetRandomSeed(), 16)),
			zap.Error(err))
		return
	}

	Logger.Info("add_my_vrf_share", zap.Any("round", vrfs.Round),
		zap.Any("round_timeout", vrfs.RoundTimeoutCount),
		zap.Int64("dkg_starting_round", dkg.StartingRound),
		zap.Int64("dkg_mb_number", dkg.MagicBlockNumber),
		zap.String("pr_seed", strconv.FormatInt(pr.GetRandomSeed(), 16)))

	vrfs.SetParty(node.Self.Underlying())
	r.vrfShare = vrfs
	// TODO: do we need to check if AddVRFShare is success or not?
	mc.AddVRFShare(ctx, r, r.vrfShare)
	go mc.SendVRFShare(ctx, r.vrfShare)
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
		return true // is ahead
	}

	return false // is not ahead, can move on
}

func (mc *Chain) finalizeRound(ctx context.Context, r *Round) {
	mc.CancelRoundVerification(ctx, r)
	go mc.FinalizeRound(ctx, r.Round, mc)
}

func (mc *Chain) startNextRoundAfterPulling(ctx context.Context, r *Round) {
	var rn = r.GetRoundNumber()

	// don't start if the round already exists
	if mc.GetCurrentRound() > r.GetRoundNumber() {
		Logger.Info("start next round after pulling -- behind current round",
			zap.Int64("round", rn))
		return // miner is ahead
	}

	// start
	mc.StartNextRound(ctx, r)
}

func (mc *Chain) pullNotarizedBlocks(ctx context.Context, r *Round) {
	Logger.Info("pull not. block for", zap.Int64("round", r.GetRoundNumber()))
	if mc.GetHeaviestNotarizedBlock(ctx, r) != nil {
		if r.GetRoundNumber() > mc.GetCurrentRound() {
			mc.SetCurrentRound(r.GetRoundNumber())
			mc.StartNextRound(ctx, r)
		}
	}
}

// StartNextRound - start the next round as a notarized
// block is discovered for the current round.
func (mc *Chain) StartNextRound(ctx context.Context, r *Round) *Round {

	var rn = r.GetRoundNumber()

	if mc.isAheadOfSharders(ctx, rn) {
		// try to slow down generation where the miner is far ahead of sharders
		select {
		case <-time.After(400 * time.Millisecond):
		case <-ctx.Done():
		}
		if mc.isAheadOfSharders(ctx, rn) {
			return nil // can't move on, still is far ahead of sharders
		}
	}

	var pr = mc.GetMinerRound(rn - 1)
	if pr != nil && !pr.IsFinalizing() && !pr.IsFinalized() {
		mc.finalizeRound(ctx, pr) // finalize the previous round
	}

	var (
		nr = round.NewRound(rn + 1)
		mr = mc.CreateRound(nr)
		er = mc.AddRound(mr).(*Round)
	)

	if er != mr && mc.isStarted() {
		Logger.Info("StartNextRound found next round ready. No VRFs Sent",
			zap.Int64("er_round", er.GetRoundNumber()),
			zap.Int64("rrs", r.GetRandomSeed()),
			zap.Bool("is_started", mc.isStarted()))

		// joining at VC case
		if mc.isJoining(rn + 1) {
			Logger.Info("StartNextRound node is joining on VC (middle)",
				zap.Int64("round", rn+1))
			go mc.pullNotarizedBlocks(ctx, er)
			return er // no VRF share exchanging for joining node (no DKG no VRF)
		}
		return er
	}

	if mc.isJoining(rn + 1) {
		Logger.Info("StartNextRound node is joining on VC (below)",
			zap.Int64("round", rn+1))
		go mc.pullNotarizedBlocks(ctx, er)
		return er // no VRF share exchanging for joining node (no DKG no VRF)
	}

	if r.HasRandomSeed() {
		mc.addMyVRFShare(ctx, r, er)
	} else {
		Logger.Info("StartNextRound no VRFs sent -- "+
			"current round has no random seed",
			zap.Int64("rrs", r.GetRandomSeed()), zap.Int64("r_round", rn))
	}

	return er
}

func (mc *Chain) getRound(ctx context.Context, rn int64) (mr *Round) {

	var pr = mc.GetMinerRound(rn - 1)

	if pr == nil {
		Logger.Error("get_round -- no previous round", zap.Int64("round", rn),
			zap.Int64("prev_round", rn-1))
		go mc.enterOnViewChange(ctx, rn-1)
		return // (nil)
	}

	Logger.Info("Starting next round in getRound",
		zap.Int64("nextRoundNum", rn))
	return mc.StartNextRound(ctx, pr) // can return nil
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
		Logger.Info("no pr info inside RedoVrfShare",
			zap.Int64("Round", r.GetRoundNumber()))
		return false
	}

	if pr.HasRandomSeed() {
		r.vrfShare = nil
		Logger.Info("RedoVrfShare after vrfShare is nil",
			zap.Int64("round", r.GetRoundNumber()),
			zap.Int("round_timeout", r.GetTimeoutCount()))
		mc.addMyVRFShare(ctx, pr, r)
		return true
	}

	return false
}

func (mc *Chain) startRound(ctx context.Context, r *Round, seed int64) {
	if !mc.SetRandomSeed(r.Round, seed) {
		return
	}
	Logger.Info("Starting a new round", zap.Int64("round", r.GetRoundNumber()))
	mc.startNewRound(ctx, r)
}

func (mc *Chain) startNewRound(ctx context.Context, mr *Round) {

	var rn = mr.GetRoundNumber()

	if rn < mc.GetCurrentRound() {
		Logger.Debug("start new round (current round higher)",
			zap.Int64("round", rn),
			zap.Int64("current_round", mc.GetCurrentRound()))
		return
	}

	if pr := mc.GetRound(rn - 1); pr == nil {
		Logger.Debug("start new round (previous round not found)",
			zap.Int64("round", rn))
		return
	}

	if mc.isJoining(mr.GetRoundNumber()) {
		mc.pullNotarizedBlocks(ctx, mr)
		return // can't be a generator
	}

	var (
		self = node.Self.Underlying()
		rank = mr.GetMinerRank(self)
	)

	if !mc.IsRoundGenerator(mr, self) {
		Logger.Info("TOC_FIX Not a generator", zap.Int64("round", rn),
			zap.Int("index", self.SetIndex),
			zap.Int("rank", rank),
			zap.Int("timeout_count", mr.GetTimeoutCount()),
			zap.Any("random_seed", mr.GetRandomSeed()))
		return
	}

	Logger.Info("*** TOC_FIX starting round block generation ***",
		zap.Int64("round", rn), zap.Int("index", self.SetIndex),
		zap.Int("rank", rank), zap.Int("timeout_count", mr.GetTimeoutCount()),
		zap.Any("random_seed", mr.GetRandomSeed()),
		zap.Int64("lf_round", mc.GetLatestFinalizedBlock().Round))

	const freezeTime = 400 * time.Millisecond

	if mc.isAheadOfSharders(ctx, rn) {
		// try to slow down generation where the miner is far ahead of sharders
		select {
		case <-time.After(freezeTime):
		case <-ctx.Done():
		}
		if mc.isAheadOfSharders(ctx, rn) {
			Logger.Info("start new round: can't move on, still is far ahead",
				zap.Int64("round", rn))
			return // can't move on, still is far ahead of sharders
		}
	}

	// NOTE: If there are not enough txns, this will not advance further even
	// though rest of the network is. That's why this is a goroutine.
	go mc.GenerateRoundBlock(ctx, mr)
}

// GetBlockToExtend - Get the block to extend from the given round.
func (mc *Chain) GetBlockToExtend(ctx context.Context, r round.RoundI) (
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
		Logger.Error("get block to extend - no notarized block",
			zap.Int64("round", r.GetRoundNumber()),
			zap.Int("num_proposals", len(proposals)),
			zap.Any("verification_tickets", pcounts))
		bnb = mc.GetHeaviestNotarizedBlock(ctx, r)
	}

	if bnb == nil {
		Logger.Debug("get block to extend - no block",
			zap.Int64("round", r.GetRoundNumber()),
			zap.Int64("current_round", mc.GetCurrentRound()))
		return // nil
	}

	if !bnb.IsStateComputed() {
		err := mc.ComputeOrSyncState(ctx, bnb)
		if err != nil {
			if state.DebugBlock() {
				Logger.Error("get block to extend - best nb compute state",
					zap.Any("round", r.GetRoundNumber()),
					zap.Any("block", bnb.Hash), zap.Error(err))
				return nil
			}
		}
	}

	return // bnb
}

// GenerateRoundBlock - given a round number generates a block.
func (mc *Chain) GenerateRoundBlock(ctx context.Context, r *Round) (*block.Block, error) {
	var ts = time.Now()
	defer func() { rbgTimer.UpdateSince(ts) }()

	roundNumber := r.GetRoundNumber()
	pround := mc.GetMinerRound(roundNumber - 1)
	if pround == nil {
		Logger.Error("generate round block - no prior round", zap.Any("round", roundNumber-1))
		return nil, common.NewError("invalid_round,", "Round not available")
	}

	pb := mc.GetBlockToExtend(ctx, pround)
	if pb == nil {
		Logger.Error("generate round block - no block to extend", zap.Any("round", roundNumber))
		return nil, common.NewError("block_gen_no_block_to_extend", "Do not have the block to extend this round")
	}

	txnEntityMetadata := datastore.GetEntityMetadata("txn")
	ctx = memorystore.WithEntityConnection(ctx, txnEntityMetadata)
	defer memorystore.Close(ctx)

	var (
		rn    = r.GetRoundNumber()                       //
		b     = block.NewBlock(mc.GetKey(), rn)          //
		lfmbr = mc.GetLatestFinalizedMagicBlockRound(rn) // related magic block

		rnoff = mbRoundOffset(rn)   //
		nvc   = mc.NextViewChange() //
	)

	if nvc > 0 && rnoff >= nvc && lfmbr.StartingRound < nvc {
		Logger.Error("gen_block",
			zap.String("err", "required MB missing or still not finalized"),
			zap.Int64("next_vc", nvc),
			zap.Int64("round", rn),
			zap.Int64("lfmbr_sr", lfmbr.StartingRound),
		)
		return nil, common.NewError("gen_block",
			"required MB missing or still not finalized")
	}

	b.LatestFinalizedMagicBlockHash = lfmbr.Hash
	b.LatestFinalizedMagicBlockRound = lfmbr.Round

	b.MinerID = node.Self.Underlying().GetKey()
	mc.SetPreviousBlock(ctx, r, b, pb)

	var (
		start             = time.Now()
		generationTimeout = time.Millisecond * time.Duration(mc.GetGenerationTimeout())

		makeBlock       bool
		generationTries int
		startLogging    time.Time
	)

	for true {
		if mc.GetCurrentRound() > b.Round {
			Logger.Error("generate block - round mismatch",
				zap.Any("round", roundNumber),
				zap.Any("current_round", mc.GetCurrentRound()))
			return nil, ErrRoundMismatch
		}

		txnCount := transaction.GetTransactionCount()
		b.SetStateDB(pb)
		generationTries++
		// if pb.GetStateStatus() != block.StateSuccessful {
		// 	if err := mc.ComputeOrSyncState(ctx, pb); err != nil {
		// 		Logger.Error("(re) computing previous block", zap.Error(err))
		// 	}
		// }

		err := mc.GenerateBlock(ctx, b, mc, makeBlock)
		if err != nil {
			cerr, ok := err.(*common.Error)
			if ok {
				switch cerr.Code {
				case InsufficientTxns:
					for true {
						delay := mc.GetRetryWaitTime()
						time.Sleep(time.Duration(delay) * time.Millisecond)
						if startLogging.IsZero() || time.Now().Sub(startLogging) > time.Second {
							startLogging = time.Now()
							Logger.Info("generate block",
								zap.Any("round", roundNumber),
								zap.Any("delay", delay),
								zap.Any("txn_count", txnCount),
								zap.Any("t.txn_count", transaction.GetTransactionCount()),
								zap.Any("error", cerr))
						}
						if mc.GetCurrentRound() > b.Round {
							Logger.Error("generate block - round mismatch",
								zap.Any("round", roundNumber),
								zap.Any("current_round", mc.GetCurrentRound()))
							return nil, ErrRoundMismatch
						}
						if txnCount != transaction.GetTransactionCount() || time.Now().Sub(start) > generationTimeout {
							makeBlock = true
							break
						}
					}
					continue
				case RoundMismatch:
					Logger.Info("generate block", zap.Error(err))
					continue
				case RoundTimeout:
					Logger.Error("generate block",
						zap.Int64("round", roundNumber),
						zap.Error(err))
					return nil, err
				}
			}
			if startLogging.IsZero() || time.Now().Sub(startLogging) > time.Second {
				startLogging = time.Now()
				Logger.Info("generate block", zap.Any("round", roundNumber),
					zap.Any("txn_count", txnCount),
					zap.Any("t.txn_count", transaction.GetTransactionCount()),
					zap.Any("error", err))
			}
			return nil, err
		}

		if r.GetRandomSeed() != b.GetRoundRandomSeed() {
			Logger.Error("round random seed mismatch",
				zap.Int64("round", b.Round),
				zap.Int64("round_rrs", r.GetRandomSeed()),
				zap.Int64("blk_rrs", b.GetRoundRandomSeed()))
			return nil, ErrRRSMismatch
		}

		mc.AddRoundBlock(r, b)
		if generationTries > 1 {
			Logger.Error("generate block - multiple tries",
				zap.Int64("round", b.Round), zap.Int("tries", generationTries))
		}
		break
	}

	if r.IsVerificationComplete() {
		Logger.Error("generate block - verification complete",
			zap.Any("round", roundNumber),
			zap.Any("notarized", len(r.GetNotarizedBlocks())))
		return nil, nil
	}

	mc.addToRoundVerification(ctx, r, b)
	r.AddProposedBlock(b)
	go mc.SendBlock(ctx, b)
	return b, nil
}

// AddToRoundVerification - add a block to verify.
func (mc *Chain) AddToRoundVerification(ctx context.Context, mr *Round, b *block.Block) {

	mr.AddProposedBlock(b)

	if mr.IsFinalizing() || mr.IsFinalized() {
		b.SetBlockState(block.StateVerificationRejected)
		Logger.Debug("add to verification", zap.Int64("round", b.Round),
			zap.String("block", b.Hash),
			zap.Bool("finalizing", mr.IsFinalizing()),
			zap.Bool("finalized", mr.IsFinalized()))
		return
	}

	if !mc.ValidateMagicBlock(ctx, mr.Round, b) {
		b.SetBlockState(block.StateVerificationRejected)
		Logger.Error("add to verification (invalid magic block)",
			zap.Int64("round", b.Round), zap.String("block", b.Hash),
			zap.String("magic_block", b.LatestFinalizedMagicBlockHash))
		return
	}

	var bNode = mc.GetMiners(mr.GetRoundNumber()).GetNode(b.MinerID)
	if bNode == nil {
		b.SetBlockState(block.StateVerificationRejected)
		Logger.Error("add to round verification (invalid miner)",
			zap.Int64("round", b.Round), zap.String("block", b.Hash),
			zap.String("miner_id", b.MinerID))
		return
	}

	var pr = mc.GetMinerRound(mr.Number - 1)
	if pr == nil {
		Logger.Error("add to verification (prior block's verify round is nil)",
			zap.Int64("round", mr.Number-1),
			zap.String("prev_block", b.PrevHash),
			zap.Int("pb_v_tickets", b.PrevBlockVerificationTicketsSize()))
		return
	}

	if b.Round > 1 {
		pb, err := mc.GetBlock(ctx, b.PrevHash)
		if err != nil || pb == nil {
			return
		}

		err = mc.VerifyNotarization(ctx, pb,
			b.GetPrevBlockVerificationTickets(), pr.GetRoundNumber())
		if err != nil {
			Logger.Error("add to verification (prior block verify notarization)",
				zap.Int64("round", pr.Number), zap.Any("miner_id", b.MinerID),
				zap.String("block", b.PrevHash),
				zap.Int("v_tickets", b.PrevBlockVerificationTicketsSize()),
				zap.Error(err))
			return
		}
	}

	if mc.AddRoundBlock(mr, b) != b {
		return
	}

	if b.PrevBlock != nil {
		if b.CreationDate < b.PrevBlock.CreationDate {
			Logger.Error("add to verification (creation_date out of sequence",
				zap.Int64("round", mr.Number), zap.String("block", b.Hash),
				zap.Any("creation_date", b.CreationDate),
				zap.String("prev_block", b.PrevHash),
				zap.Any("prev_creation_date", b.PrevBlock.CreationDate))
			return
		}
		b.ComputeChainWeight()
		mc.updatePriorBlock(ctx, mr.Round, b)
	} else {
		// We can establish an upper bound for chain weight at the current
		// round, subtract 1 and add block's own weight and check if that's
		// less than the chain weight sent.
		var (
			lfb                   = mc.GetLatestFinalizedBlock()
			chainWeightUpperBound = lfb.ChainWeight + float64(b.Round-lfb.Round)
		)
		if b.ChainWeight > chainWeightUpperBound-1+b.Weight() {
			Logger.Error("add to verification (wrong chain weight)",
				zap.Int64("round", b.Round), zap.String("block", b.Hash),
				zap.Float64("chain_weight", b.ChainWeight))
			return
		}
	}

	mc.addToRoundVerification(ctx, mr, b)
}

func (mc *Chain) addToRoundVerification(ctx context.Context, mr *Round, b *block.Block) {
	Logger.Info("adding block to verify", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("prev_block", b.PrevHash), zap.String("state_hash", util.ToHex(b.ClientStateHash)), zap.Float64("weight", b.Weight()), zap.Float64("chain_weight", b.ChainWeight))
	vctx := mr.StartVerificationBlockCollection(ctx)
	miner := mc.GetMiners(mr.GetRoundNumber()).GetNode(b.MinerID)
	if vctx != nil && miner != nil {

		waitTime := mc.GetBlockProposalWaitTime(mr.Round)
		minerNT := time.Duration(int64(miner.GetLargeMessageSendTime()/1000000)) * time.Millisecond
		if minerNT >= waitTime {
			mr.delta = time.Millisecond
		} else {
			mr.delta = waitTime - minerNT
		}
		go mc.CollectBlocksForVerification(vctx, mr)
	}
	mr.AddBlockToVerify(b)
}

/*GetBlockProposalWaitTime - get the time to wait for the block proposals of the given round */
func (mc *Chain) GetBlockProposalWaitTime(r round.RoundI) time.Duration {
	if mc.BlockProposalWaitMode == chain.BlockProposalWaitDynamic {
		return mc.computeBlockProposalDynamicWaitTime(r)
	}
	return mc.BlockProposalMaxWaitTime
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
	return mc.BlockProposalMaxWaitTime
}

/*CollectBlocksForVerification - keep collecting the blocks till timeout and then start verifying */
func (mc *Chain) CollectBlocksForVerification(ctx context.Context, r *Round) {
	verifyAndSend := func(ctx context.Context, r *Round, b *block.Block) bool {
		b.SetBlockState(block.StateVerificationAccepted)
		miner := mc.GetMiners(r.GetRoundNumber()).GetNode(b.MinerID)
		if miner == nil || miner.ProtocolStats == nil {
			Logger.Error("verify round block -- failed miner", zap.Any("round", r.Number), zap.Any("block", b.Hash), zap.Any("miner", b.MinerID))
			b.SetBlockState(block.StateVerificationFailed)
			return false
		}
		minerStats := miner.ProtocolStats.(*chain.MinerStats)
		bvt, err := mc.VerifyRoundBlock(ctx, r, b)
		if err != nil {
			b.SetBlockState(block.StateVerificationFailed)
			minerStats.VerificationFailures++
			if cerr, ok := err.(*common.Error); ok {
				if cerr.Code == RoundMismatch {
					Logger.Debug("verify round block", zap.Any("round", r.Number), zap.Any("block", b.Hash), zap.Any("current_round", mc.GetCurrentRound()))
				} else {
					Logger.Error("verify round block", zap.Any("round", r.Number), zap.Any("block", b.Hash), zap.Error(err))
				}
			} else {
				Logger.Error("verify round block", zap.Any("round", r.Number), zap.Any("block", b.Hash), zap.Error(err))
			}
			return false
		}
		b.SetBlockState(block.StateVerificationSuccessful)
		bnb := r.GetBestRankedNotarizedBlock()
		if bnb == nil || (bnb != nil && bnb.Hash == b.Hash) {
			Logger.Info("Sending verification ticket", zap.Int64("round", r.Number), zap.String("block", b.Hash))
			go mc.SendVerificationTicket(ctx, b, bvt)
		}
		if bnb == nil {
			r.Block = b
			mc.ProcessVerifiedTicket(ctx, r, b, &bvt.VerificationTicket)
		}
		if b.RoundRank >= mc.NumGenerators || b.RoundRank < 0 {
			Logger.Warn("round rank is invalid or greater then num_generators",
				zap.String("hash", b.Hash), zap.Int64("round", b.Round),
				zap.Int("round_rank", b.RoundRank),
				zap.Int("num_generators", mc.NumGenerators))
		} else {
			minerStats.VerificationTicketsByRank[b.RoundRank]++
		}
		return true
	}
	var sendVerification = false
	var blocks = make([]*block.Block, 0, 8)
	initiateVerification := func() {
		// Sort the accumulated blocks by the rank and process them
		blocks = r.GetBlocksByRank(blocks)
		// Keep verifying all the blocks collected so far in the best rank order till the first successul verification
		for _, b := range blocks {
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
	r.SetState(round.RoundCollectingBlockProposals)
	for true {
		select {
		case <-ctx.Done():
			r.SetState(round.RoundStateVerificationTimedOut)
			bRank := -1
			if r.Block != nil {
				bRank = r.Block.RoundRank
			}
			for _, b := range blocks {
				bs := b.GetBlockState()
				if bRank != 0 && bRank != b.RoundRank {
					Logger.Info("verification cancel (failing block)", zap.Int64("round", r.Number), zap.String("block", b.Hash), zap.Int("block_rank", b.RoundRank), zap.Int("best_rank", bRank), zap.Int8("block_state", bs))
				}
				if bs == block.StateVerificationPending || bs == block.StateVerificationAccepted {
					b.SetBlockState(block.StateVerificationFailed)
				}
			}
			return
		case <-blockTimeTimer.C:
			initiateVerification()
		case b := <-r.GetBlocksToVerifyChannel():
			if sendVerification {
				// Is this better than the current best block
				if r.Block == nil || r.Block.RoundRank >= b.RoundRank {
					b.SetBlockState(block.StateVerificationPending)
					verifyAndSend(ctx, r, b)
				} else {
					b.SetBlockState(block.StateVerificationRejected)
				}
			} else { // Accumulate all the blocks into this array till the BlockTime timeout
				b.SetBlockState(block.StateVerificationPending)
				blocks = append(blocks, b)
			}
		}
	}
}

/*VerifyRoundBlock - given a block is verified for a round*/
func (mc *Chain) VerifyRoundBlock(ctx context.Context, r *Round, b *block.Block) (*block.BlockVerificationTicket, error) {
	if !mc.CanShardBlocks(r.Number) {
		return nil, common.NewError("fewer_active_sharders", "Number of active sharders not sufficient")
	}
	if !mc.CanReplicateBlock(b) {
		return nil, common.NewError("fewer_active_replicators", "Number of active replicators not sufficient")
	}
	if mc.GetCurrentRound() != r.Number {
		return nil, ErrRoundMismatch
	}
	if b.MinerID == node.Self.Underlying().GetKey() {
		return mc.SignBlock(ctx, b)
	}
	var hasPriorBlock = b.PrevBlock != nil
	bvt, err := mc.VerifyBlock(ctx, b)
	if err != nil {
		b.SetVerificationStatus(block.VerificationFailed)
		return nil, err
	}
	if !hasPriorBlock && b.PrevBlock != nil {
		mc.updatePriorBlock(ctx, r.Round, b)
	}
	return bvt, nil
}

func (mc *Chain) updatePriorBlock(ctx context.Context, r *round.Round, b *block.Block) {
	pb := b.PrevBlock
	mc.MergeVerificationTickets(ctx, pb, b.GetPrevBlockVerificationTickets())
	pr := mc.GetMinerRound(pb.Round)
	if pr != nil {
		mc.AddNotarizedBlock(ctx, pr, pb)
	} else {
		Logger.Error("verify round - previous round not present", zap.Int64("round", r.Number), zap.String("block", b.Hash), zap.String("prev_block", b.PrevHash))
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
	if !mc.AddVerificationTicket(ctx, b, vt) {
		return
	}

	if notarized {
		Logger.Info("Block is notarized", zap.Int64("round", r.Number),
			zap.String("block", b.Hash))
		return
	}

	mc.checkBlockNotarization(ctx, r, b)
}

func (mc *Chain) checkBlockNotarization(ctx context.Context, r *Round, b *block.Block) bool {
	if !b.IsBlockNotarized() {
		Logger.Info("checkBlockNotarization -- block is not Notarized. Returning",
			zap.Int64("round", b.Round))
		return false
	}
	if !mc.AddNotarizedBlock(ctx, r, b) {
		return true
	}
	mc.SetRandomSeed(r, b.GetRoundRandomSeed())
	go mc.SendNotarization(ctx, b)
	Logger.Debug("check block notarization - block notarized", zap.Int64("round", b.Round), zap.String("block", b.Hash))
	mc.StartNextRound(common.GetRootContext(), r) // start or skip
	return true
}

// MergeNotarization - merge a notarization.
func (mc *Chain) MergeNotarization(ctx context.Context, r *Round, b *block.Block, vts []*block.VerificationTicket) {
	for _, t := range vts {
		if err := mc.VerifyTicket(ctx, b.Hash, t, r.GetRoundNumber()); err != nil {
			Logger.Error("merge notarization", zap.Int64("round", b.Round),
				zap.String("block", b.Hash), zap.Error(err))
		}
	}
	notarized := b.IsBlockNotarized()
	mc.MergeVerificationTickets(ctx, b, vts)
	if notarized {
		return
	}
	mc.checkBlockNotarization(ctx, r, b)
}

/*AddNotarizedBlock - add a notarized block for a given round */
func (mc *Chain) AddNotarizedBlock(ctx context.Context, r *Round, b *block.Block) bool {
	if _, ok := r.AddNotarizedBlock(b); !ok {
		return false
	}
	if !b.IsStateComputed() {
		Logger.Info("add notarized block - computing state",
			zap.Int64("round", b.Round), zap.String("block", b.Hash))
		go mc.ComputeState(ctx, b)
	}
	if !r.IsVerificationComplete() {
		mc.CancelRoundVerification(ctx, r)
	}
	b.SetBlockState(block.StateNotarized)
	mc.UpdateNodeState(b)
	return true
}

/*CancelRoundVerification - cancel verifications happening within a round */
func (mc *Chain) CancelRoundVerification(ctx context.Context, r *Round) {
	r.CancelVerification() // No need for further verification of any blocks
}

type BlockConsensus struct {
	*block.Block
	Consensus int
}

/*GetLatestFinalizedBlockFromSharder - request for latest finalized block from all the sharders */
func (mc *Chain) GetLatestFinalizedBlockFromSharder(ctx context.Context) []*BlockConsensus {
	mb := mc.GetCurrentMagicBlock()
	m2s := mb.Sharders
	finalizedBlocks := make([]*BlockConsensus, 0, 1)
	fbMutex := &sync.Mutex{}
	//Params are nil? Do we need to send any params like sending the miner ID ?
	handler := func(ctx context.Context, entity datastore.Entity) (interface{}, error) {
		fb, ok := entity.(*block.Block)
		if fb.Round == 0 {
			return nil, nil
		}
		if !ok {
			return nil, datastore.ErrInvalidEntity
		}
		err := fb.Validate(ctx)
		if err != nil {
			Logger.Error("lfb from sharder - invalid",
				zap.Int64("round", fb.Round),
				zap.String("block", fb.Hash),
				zap.Error(err))
			return nil, err
		}
		var r = mc.GetRound(fb.Round)
		if r == nil {
			if r = mc.getRound(ctx, fb.Round); isNilRound(r) {
				// for a far ahead sharders case, create round, since the
				// getRound can return nil
				var (
					rx = round.NewRound(fb.Round)
					mr = mc.CreateRound(rx)
				)
				r = mc.AddRound(mr).(*Round)
			}
		}
		err = mc.VerifyNotarization(ctx, fb, fb.GetVerificationTickets(),
			r.GetRoundNumber())
		if err != nil {
			Logger.Error("lfb from sharder - notarization failed", zap.Int64("round", fb.Round),
				zap.String("block", fb.Hash), zap.Error(err))
			return nil, err
		}
		fbMutex.Lock()
		defer fbMutex.Unlock()
		for i, b := range finalizedBlocks {
			if b.Hash == fb.Hash {
				finalizedBlocks[i].Consensus++
				return fb, nil
			}
		}
		finalizedBlocks = append(finalizedBlocks, &BlockConsensus{
			Block:     fb,
			Consensus: 1,
		})
		return fb, nil
	}

	m2s.RequestEntityFromAll(ctx, MinerLatestFinalizedBlockRequestor, nil, handler)

	// highest (the first sorting order), most popular (the second order)
	sort.Slice(finalizedBlocks, func(i int, j int) bool {
		return finalizedBlocks[i].Round >= finalizedBlocks[j].Round ||
			finalizedBlocks[i].Consensus > finalizedBlocks[j].Consensus

	})
	return finalizedBlocks
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
		mb              = mc.GetCurrentMagicBlock()
		sharders        = mb.Sharders
		finalizedBlocks = make([]*blockConsensus, 0, 1)

		fbMutex sync.Mutex
	)

	var handler = func(ctx context.Context, entity datastore.Entity) (
		resp interface{}, err error) {

		var fb, ok = entity.(*block.Block)
		if !ok {
			return nil, datastore.ErrInvalidEntity
		}

		if err = fb.Validate(ctx); err != nil {
			Logger.Error("FB from sharder - invalid",
				zap.Int64("round", fb.Round),
				zap.String("block", fb.Hash),
				zap.Error(err))
			return nil, err
		}

		var r = mc.GetRound(fb.Round)
		if r == nil {
			r = mc.getRound(ctx, fb.Round)
		}
		err = mc.VerifyNotarization(ctx, fb, fb.GetVerificationTickets(),
			r.GetRoundNumber())
		if err != nil {
			Logger.Error("FB from sharder - notarization failed",
				zap.Int64("round", fb.Round),
				zap.String("block", fb.Hash), zap.Error(err))
			return nil, err
		}

		fbMutex.Lock()
		defer fbMutex.Unlock()
		for i, b := range finalizedBlocks {
			if b.Hash == fb.Hash {
				finalizedBlocks[i].consensus++
				return fb, nil
			}
		}
		finalizedBlocks = append(finalizedBlocks, &blockConsensus{
			Block:     fb,
			consensus: 1,
		})

		return fb, nil
	}

	sharders.RequestEntityFromAll(ctx, chain.FBRequestor, nil, handler)

	// highest (the first sorting order), most popular (the second order)
	sort.Slice(finalizedBlocks, func(i int, j int) bool {
		return finalizedBlocks[i].Round >= finalizedBlocks[j].Round ||
			finalizedBlocks[i].consensus > finalizedBlocks[j].consensus

	})

	if len(finalizedBlocks) == 0 {
		Logger.Error("FB from sharders -- no block given",
			zap.String("hash", hash))
		return nil
	}

	return finalizedBlocks[0].Block
}

// GetNextRoundTimeoutTime returns time in milliseconds.
func (mc *Chain) GetNextRoundTimeoutTime(ctx context.Context) int {

	ssft := int(math.Ceil(chain.SteadyStateFinalizationTimer.Mean() / 1000000))
	tick := mc.RoundTimeoutSofttoMin
	if tick < mc.RoundTimeoutSofttoMult*ssft {
		tick = mc.RoundTimeoutSofttoMult * ssft
	}
	Logger.Info("nextTimeout", zap.Int("tick", tick))
	return tick
}

// HandleRoundTimeout handle timeouts appropriately.
func (mc *Chain) HandleRoundTimeout(ctx context.Context) {

	var (
		rn  = mc.GetCurrentRound()
		mmb = mc.GetMagicBlock(rn + chain.ViewChangeOffset + 1)
		cmb = mc.GetMagicBlock(rn)

		selfNodeKey = node.Self.Underlying().GetKey()
	)

	// miner should be member of current magic block; also, we have to call the
	// restartRound on last round of MB the miner is not member (on joining)
	if cmb == nil || !cmb.Miners.HasNode(selfNodeKey) &&
		!mmb.Miners.HasNode(selfNodeKey) {

		return
	}

	var r = mc.GetMinerRound(rn)

	if r.GetSoftTimeoutCount() == mc.RoundRestartMult {
		Logger.Info("triggering restartRound",
			zap.Int64("round", r.GetRoundNumber()))
		mc.restartRound(ctx)
		return
	}

	Logger.Info("triggering handleNoProgress",
		zap.Int64("round", r.GetRoundNumber()))
	mc.handleNoProgress(ctx)
	r.IncSoftTimeoutCount()
}

func (mc *Chain) handleNoProgress(ctx context.Context) {
	r := mc.GetMinerRound(mc.GetCurrentRound())
	proposals := r.GetProposedBlocks()
	if len(proposals) > 0 { // send the best block to the network
		b := r.Block
		if b != nil {
			if mc.GetRoundTimeoutCount() <= 10 {
				Logger.Error("sending the best block to the network",
					zap.Int64("round", b.Round), zap.String("block", b.Hash),
					zap.Int("rank", b.RoundRank))
			}
			mc.SendBlock(ctx, b)
		}
	}

	if r.vrfShare != nil {
		go mc.SendVRFShare(ctx, r.vrfShare)
		Logger.Info("Sent vrf shares in handle NoProgress")
	} else {
		Logger.Info("Did not send vrf shares as it is nil", zap.Int64("round_num", r.GetRoundNumber()))
	}
	switch crt := mc.GetRoundTimeoutCount(); {
	case crt < 10:
		Logger.Error("handleNoProgress", zap.Any("round", mc.GetCurrentRound()), zap.Int64("count_round_timeout", crt), zap.Any("num_vrf_share", len(r.GetVRFShares())))
	case crt == 10:
		Logger.Error("handleNoProgress (no further timeout messages will be displayed)", zap.Any("round", mc.GetCurrentRound()), zap.Int64("count_round_timeout", crt), zap.Any("num_vrf_share", len(r.GetVRFShares())))
		//TODO: should have a means to send an email/SMS to someone or something like that
	}

}

func (mc *Chain) kickFinalization(ctx context.Context) {

	Logger.Info("restartRound->kickFinalization")

	// kick all blocks finalization

	var lfb = mc.GetLatestFinalizedBlock()

	// don't kick more then 5 blocks at once
	var i, e, j = lfb.Round, mc.GetCurrentRound(), 0 // loop variables
	for ; i < e && j < 5; i, j = i+1, j+1 {
		var mr = mc.GetMinerRound(i)
		if mr == nil || mr.IsFinalized() {
			continue // skip finalized blocks, skip nil miner rounds
		}
		Logger.Info("restartRound->kickFinalization:",
			zap.Int64("round", mr.GetRoundNumber()))
		go mc.FinalizeRound(ctx, mr.Round, mc) // kick finalization again
	}
}

// push notarized blocks to sharders again (the far ahead state)
func (mc *Chain) kickSharders(ctx context.Context) {

	var (
		lfb = mc.GetLatestFinalizedBlock()
		tk  = mc.GetLatestLFBTicket(ctx)
	)

	if lfb.Round <= tk.Round {
		mc.kickFinalization(ctx)
		return
	}

	Logger.Info("restartRound->kickSharders: kick sharders")

	// don't kick more then 5 blocks at once
	var (
		s, c, i = tk.Round, mc.GetCurrentRound(), 0 // loop variables
		ahead   = config.GetLFBTicketAhead()
	)
	for ; s < c && i < ahead; s, i = s+1, i+1 {
		var mr = mc.GetMinerRound(s)
		// send block to sharders again, if missing sharders side
		if mr != nil && mr.Block != nil && mr.Block.IsBlockNotarized() &&
			mr.Block.GetStateStatus() == block.StateSuccessful {

			Logger.Info("restartRound->kickSharders: kick sharder FB",
				zap.Int64("round", mr.GetRoundNumber()))
			go mc.ForcePushNotarizedBlock(ctx, mr.Block)
		}
	}
}

func isNilRound(r round.RoundI) bool {
	return r == nil || r == round.RoundI((*Round)(nil))
}

func (mc *Chain) kickRoundByLFB(ctx context.Context, lfb *block.Block) {

	var (
		sr = round.NewRound(lfb.Round)
		mr = mc.CreateRound(sr)
		nr *Round
	)

	if !mc.ensureState(ctx, lfb) {
		return // don't kick -- no block state
	}

	mr, _ = mc.AddRound(mr).(*Round)
	mc.SetRandomSeed(sr, lfb.RoundRandomSeed)
	mc.AddBlock(lfb)
	if nr = mc.StartNextRound(ctx, mr); nr == nil {
		return
	}
	mc.SetCurrentRound(nr.Number)
}

func (mc *Chain) restartRound(ctx context.Context) {

	var crn = mc.GetCurrentRound()

	mc.IncrementRoundTimeoutCount()
	var r = mc.GetMinerRound(crn)

	switch crt := mc.GetRoundTimeoutCount(); {
	case crt < 10:
		Logger.Error("restartRound - round timeout occurred",
			zap.Any("round", mc.GetCurrentRound()), zap.Int64("count", crt),
			zap.Any("num_vrf_share", len(r.GetVRFShares())))

	case crt == 10:
		Logger.Error("restartRound - round timeout occurred (no further"+
			" timeout messages will be displayed)",
			zap.Any("round", mc.GetCurrentRound()), zap.Int64("count", crt),
			zap.Any("num_vrf_share", len(r.GetVRFShares())))

		// TODO: should have a means to send an email/SMS to someone or
		// something like that
	}
	mc.RoundTimeoutsCount++

	println("RR ", "CRN", crn, "CTC", mc.RoundTimeoutsCount)

	// get LFMB and LFB from sharders
	var updated, err = mc.ensureLatestFinalizedBlocks(ctx)
	if err != nil {
		Logger.Error("restartRound - ensure lfb", zap.Error(err))
	}

	// node joining on VC, it hasn't DKG and can VRF, and should pull not.
	// blocks to create and move rounds until VC, setting RRS by the blocks
	if mc.isJoining(crn) {
		Logger.Info("restartRound node is joining on VC",
			zap.Int64("round", crn))
		// pull previous round
		var pr = mc.GetMinerRound(crn - 1)
		if pr == nil {
			var nr = round.NewRound(crn - 1)
			pr = mc.CreateRound(nr)
			pr = mc.AddRound(pr).(*Round)
		}
		if !pr.HasRandomSeed() {
			mc.pullNotarizedBlocks(ctx, pr)
			println("RR IS J", "CRN", crn, "PULL PREV R.")
		}
		// pull current round
		mc.pullNotarizedBlocks(ctx, r)
		println("RR IS J", "PULL CRN")
		return
	}

	var isAhead = mc.isAheadOfSharders(ctx, crn)

	if updated {
		// kick new round from the new LFB from sharders, if it's newer
		// then the current one
		var lfb = mc.GetLatestFinalizedBlock()
		if lfb.Round > crn {
			mc.kickRoundByLFB(ctx, lfb)
			println("RR ", "CRN", crn, "KICK ROUND BY LFB", lfb.Round)
			return
		}
	} else {
		if isAhead {
			println("RR ", "CRN", crn, "KICK SHARDERS")
			mc.kickSharders(ctx) // not updated, kick sharders
		}
	}

	if !updated && crn > 1 && !isAhead &&
		r.GetHeaviestNotarizedBlock() != nil && r.HasRandomSeed() {

		Logger.Info("StartNextRound after sending notarized "+
			"block in restartRound.",
			zap.Int64("current_round", r.GetRoundNumber()))
		nextR := mc.GetRound(r.GetRoundNumber())
		nr := mc.StartNextRound(ctx, r)
		if nr == nil {
			Logger.Info("restartRound: skip due to far ahead")
			println("RR ", "CRN", crn, "FAR AHEAD (1)")
			return // shouldn't happen
		}
		/*
			if the next round object already exists, StartNextRound does not send VRFs.
			So to be sure send it.
		*/
		if r.HasRandomSeed() {
			println("RR ", "CRN", crn, "HAS SEED -> NEXT ROUND", nextR != nil)
			if nextR != nil {
				Logger.Info("RedoVRFshare after sending notarized"+
					" block in restartRound.",
					zap.Int64("round", nr.GetRoundNumber()),
					zap.Int("round_toc", nr.GetTimeoutCount()))

				nr.Restart()
				//Recalculate VRF shares and send
				nr.IncrementTimeoutCount(r.GetRandomSeed(),
					mc.GetMiners(nr.GetRoundNumber()))
				redo := mc.RedoVrfShare(ctx, nr)
				if !redo {
					Logger.Info("Could not  RedoVrfShare",
						zap.Int64("round", r.GetRoundNumber()),
						zap.Int("round_timeout", r.GetTimeoutCount()))
				}

			} else {
				//StartNextRound would have sent the VRFs. No need to do that again
				Logger.Info("after sending notarized block in"+
					" restartRound NextR was nil. startNextRound"+
					" would have sent VRF.",
					zap.Int64("round", nr.GetRoundNumber()),
					zap.Int("round_toc", nr.GetTimeoutCount()))

			}
			return
		}
		Logger.Error("Has notarized block in restartRound, but no randomseed.",
			zap.Int64("current_round", r.GetRoundNumber()))
	}

	r.Restart()

	// recalculate VRF shares and send
	var pr = mc.GetMinerRound(crn - 1)
	if pr == nil {
		var nr = round.NewRound(crn - 1)
		pr = mc.CreateRound(nr)
		pr = mc.AddRound(pr).(*Round)
	}
	if !pr.HasRandomSeed() {
		mc.pullNotarizedBlocks(ctx, pr)
		println("RR ", "CRN", crn, "PULL PREV R.")
	}
	r.IncrementTimeoutCount(pr.GetRandomSeed(), mc.GetMiners(crn))
	if redo := mc.RedoVrfShare(ctx, r); !redo {
		Logger.Info("Could not RedoVrfShare",
			zap.Int64("round", r.GetRoundNumber()),
			zap.Int("round_timeout", r.GetTimeoutCount()))
	}
}

// ensureState makes sure block state is computed and initialized, it can't
// be sure the block stat will not be changed later (or will not become invalid)
// so, it's optimistic, let's track logs to find out how it's critical
// in reality
func (mc *Chain) ensureState(ctx context.Context, b *block.Block) (ok bool) {

	var err error
	if !b.IsStateComputed() {
		if err = mc.ComputeOrSyncState(ctx, b); err != nil {
			Logger.Error("ensure_state -- compute or sync",
				zap.Error(err), zap.Int64("round", b.Round))
		}
	}
	if b.ClientState == nil {
		if err = mc.InitBlockState(b); err != nil {
			Logger.Error("ensure_state -- initialize block state",
				zap.Error(err), zap.Int64("round", b.Round))
		}
	}

	// ensure next view change (from sharders)
	if ok = (b.IsStateComputed() && b.ClientState != nil); ok {
		var nvc int64
		if nvc, err = mc.NextViewChangeOfBlock(b); err != nil {
			Logger.Error("ensure_state -- next view change",
				zap.Error(err), zap.Int64("round", b.Round))
			return // but return result
		}
		mc.SetNextViewChange(nvc)
	}

	return
}

func (mc *Chain) ensureLatestFinalizedBlock(ctx context.Context) (
	updated bool, err error) {

	// LFB regardless a ticket
	var (
		rcvd *block.Block
		have = mc.GetLatestFinalizedBlock()
		list = mc.GetLatestFinalizedBlockFromSharder(ctx)
	)

	if len(list) == 0 {
		return // no LFB given
	}

	rcvd = list[0].Block // the highest received LFB

	if have != nil && rcvd.Round <= have.Round {
		mc.ensureState(ctx, have)
		return // nothing to update
	}

	mc.bumpLFBTicket(ctx, rcvd)
	if !mc.ensureState(ctx, rcvd) {
		// but continue with the block (?)
	}
	// it create corresponding round or makes sure it exists
	mc.SetLatestFinalizedBlock(ctx, rcvd)
	return true, nil // updated
}

func (mc *Chain) ensureDKG(ctx context.Context, mb *block.Block) {
	if mb == nil {
		return
	}
	if !config.DevConfiguration.ViewChange {
		return
	}
	var err error
	if err = mc.SetDKGSFromStore(ctx, mb.MagicBlock); err != nil {
		Logger.Error("setting DKG from store",
			zap.Int64("mb_round", mb.Round), zap.Error(err))
	}
}

func (mc *Chain) ensureLatestFinalizedBlocks(ctx context.Context) (
	updated bool, err error) {

	defer mc.SetupLatestAndPreviousMagicBlocks(ctx)

	// LFB
	if updated, err = mc.ensureLatestFinalizedBlock(ctx); err != nil {
		return
	}

	// LFMB
	var (
		lfmb = mc.GetLatestFinalizedMagicBlock()
		list = mc.GetLatestFinalizedMagicBlockFromSharder(ctx)
		rcvd *block.Block
	)

	mc.ensureDKG(ctx, lfmb)

	if len(list) == 0 {
		return
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].StartingRound > list[j].StartingRound
	})
	rcvd = list[0]

	if lfmb != nil && rcvd.MagicBlockNumber <= lfmb.MagicBlockNumber {
		return
	}

	if err = mc.VerifyChainHistory(ctx, rcvd, nil); err != nil {
		return false, err
	}
	if err = mc.UpdateMagicBlock(rcvd.MagicBlock); err != nil {
		return false, err
	}
	mc.UpdateNodesFromMagicBlock(rcvd.MagicBlock)
	mc.SetLatestFinalizedMagicBlock(rcvd)
	mc.ensureDKG(ctx, rcvd)

	// bump the ticket if necessary
	var tk = mc.GetLatestLFBTicket(ctx)
	if tk == nil || tk.Round < rcvd.Round {
		mc.AddReceivedLFBTicket(ctx, &chain.LFBTicket{
			Round: rcvd.Round,
		})
	}

	// don't set the MB as latest finalized MB; it should be done inside
	// ensure view change;

	updated = true
	return
}

// bump the ticket if necessary
func (mc *Chain) bumpLFBTicket(ctx context.Context, lfbs *block.Block) {
	if lfbs == nil {
		return
	}
	var tk = mc.GetLatestLFBTicket(ctx) // is the worker starts
	if tk == nil || tk.Round < lfbs.Round {
		mc.AddReceivedLFBTicket(ctx, &chain.LFBTicket{Round: lfbs.Round})
	}
}

// func (mc *Chain) initLFBState(lfb *block.Block) {
// 	var err error
// 	if err = mc.InitBlockState(lfb); err != nil {
// 		go mc.asyncInitLFBState()
// 	}
// }

// func (mc *Chain) asyncInitLFBState() {
// 	var (
// 		lfb *block.Block
// 		err error
// 	)
// 	for {
// 		time.Sleep(time.Second * 1)
// 		lfb = mc.GetLatestFinalizedBlock()
// 		if err = mc.InitBlockState(lfb); err == nil {
// 			return
// 		}
// 		Logger.Error("start_protocol", zap.Error(err))
// 	}
// }

func (mc *Chain) startProtocolOnLFB(ctx context.Context, lfb *block.Block) (
	mr *Round) {

	if lfb == nil {
		return // nil
	}

	mc.bumpLFBTicket(ctx, lfb)

	// we can't compute state in the start protocol
	if err := mc.InitBlockState(lfb); err != nil {
		lfb.SetStateStatus(0)
	}

	mc.SetLatestFinalizedBlock(ctx, lfb)
	return mc.GetMinerRound(lfb.Round)
}

func StartProtocol(ctx context.Context, gb *block.Block) {
	var (
		mc  = GetMinerChain()
		lfb = getLatestBlockFromSharders(ctx)
		mr  *Round
	)
	if lfb != nil {
		mr = mc.startProtocolOnLFB(ctx, lfb)
	} else {
		// start on genesis block
		mc.bumpLFBTicket(ctx, gb)
		var r = round.NewRound(gb.Round)
		mr = mc.CreateRound(r)
		mr = mc.AddRound(mr).(*Round)
	}
	var nr = mc.StartNextRound(ctx, mr)
	for nr == nil {
		select {
		case <-time.After(4 * time.Second): // repeat after some time
			if _, err := mc.ensureLatestFinalizedBlocks(ctx); err != nil {
				Logger.Error("getting latest blocks from sharders",
					zap.Error(err))
				continue
			}
			lfb = mc.GetLatestFinalizedBlock()

			mr = mc.startProtocolOnLFB(ctx, lfb)
		case <-ctx.Done():
			return
		}
		nr = mc.StartNextRound(ctx, mr)
	}
	mc.SetCurrentRound(nr.Number)
	Logger.Info("starting the blockchain ...", zap.Int64("round", nr.Number))
}

func (mc *Chain) setupLoadedMagicBlock(mb *block.MagicBlock) (err error) {
	if err = mc.UpdateMagicBlock(mb); err != nil {
		return
	}
	mc.UpdateNodesFromMagicBlock(mb)
	return
}

// LoadMagicBlocksAndDKG from store to start working. It loads MB and previous
// MB (regardless Miner SC rejection) and related DKG and sets them. It can
// return the latest MB and an error related to DKG or previous MB/DKG loading.
// The method can write INFO logs that doesn't really mean an error. Since, the
// method is optimistic and tries to load latest and previous MBs and related
// DKGs. But in a normal case miner can have or haven't the MBs and the DKGs.
func (mc *Chain) LoadMagicBlocksAndDKG(ctx context.Context) {

	// latest MB
	var (
		latest *block.MagicBlock
		err    error
	)
	if latest, err = LoadLatestMB(ctx); err != nil {
		Logger.Info("load_mbs_and_dkg -- loading the latest MB",
			zap.Error(err))
		return // can't continue
	}

	// don't setup the latest MB since it can be promoted

	//	if err = mc.setupLoadedMagicBlock(latest); err != nil {
	//		Logger.Info("load_mbs_and_dkg -- updating the latest MB",
	//			zap.Error(err))
	//		return // can't continue
	//	}
	//	mc.SetMagicBlock(latest)
	//	// related DKG (if any)
	//	if err = mc.SetDKGSFromStore(ctx, latest); err != nil {
	//		Logger.Info("load_mbs_and_dkg -- loading the latest DKG",
	//			zap.Error(err))
	//	}
	if latest.MagicBlockNumber <= 1 {
		return // done
	}
	// otherwise, load and setup previous
	var (
		prev *block.MagicBlock
		id   = strconv.FormatInt(latest.MagicBlockNumber-1, 10)
	)
	if prev, err = LoadMagicBlock(ctx, id); err != nil {
		Logger.Info("load_mbs_and_dkg -- loading previous MB",
			zap.String("sr", id), zap.Error(err))
		return // can't continue
	}
	if err = mc.setupLoadedMagicBlock(prev); err != nil {
		Logger.Info("load_mbs_and_dkg -- updating previous MB",
			zap.Error(err))
		return // can't continue
	}
	mc.SetMagicBlock(prev)

	// don't setup latest MB since it can be promoted
	//
	//	// and then setup the latest again for proper nodes registration,
	//	// ignoring its error
	//	mc.setupLoadedMagicBlock(latest)
	// DKG relates previous MB

	if err = mc.SetDKGSFromStore(ctx, prev); err != nil {
		Logger.Info("load_mbs_and_dkg -- loading previous DKG",
			zap.Error(err))
	}
	// everything is OK
}

func (mc *Chain) WaitForActiveSharders(ctx context.Context) error {
	var oldRound = mc.GetCurrentRound()
	defer mc.SetCurrentRound(oldRound)
	defer chain.ResetStatusMonitor(oldRound)

	var lmb = mc.GetCurrentMagicBlock()

	// we can't use the lmb.StartingRound as current round, since the lmb
	// is saved in store but can be rejected by Miner SC if the LMB is not
	// view changed yet (451-502 rounds)

	if mc.CanShardBlocksSharders(lmb.Sharders) {
		return nil
	}

	var waitingSharders = make([]string, 0, lmb.Sharders.MapSize())
	for _, nodeSharder := range lmb.Sharders.CopyNodesMap() {
		waitingSharders = append(waitingSharders,
			fmt.Sprintf("id: %s; n2nhost: %s ", nodeSharder.ID, nodeSharder.N2NHost))
	}

	Logger.Debug("waiting for sharders",
		zap.Int64("latest_magic_block_round", lmb.StartingRound),
		zap.Int64("latest_magic_block_number", lmb.MagicBlockNumber),
		zap.Any("sharders", waitingSharders))

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
			Logger.Info("Waiting for Sharders.", zap.Time("ts", ts),
				zap.Any("sharders", waitingSharders))
			lmb.Sharders.OneTimeStatusMonitor(ctx) // just mark 'em active
		}
	}
}
