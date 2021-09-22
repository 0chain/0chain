package miner

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"sync"
	"time"

	"0chain.net/core/encryption"
	"0chain.net/smartcontract/minersc"

	"0chain.net/core/logging"
	"0chain.net/core/memorystore"
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
	"0chain.net/core/util"

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
		return
	}

	var dkg = mc.GetDKG(rn)
	if dkg == nil {
		logging.Logger.Error("add_my_vrf_share -- DKG is nil, my VRF share is not added",
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
		logging.Logger.Error("add_my_vrf_share", zap.Any("round", vrfs.Round),
			zap.Any("round_timeout", vrfs.RoundTimeoutCount),
			zap.Int64("dkg_starting_round", dkg.StartingRound),
			zap.Int64("dkg_mb_number", dkg.MagicBlockNumber),
			zap.String("pr_seed", strconv.FormatInt(pr.GetRandomSeed(), 16)),
			zap.Error(err))
		return
	}

	logging.Logger.Info("add_my_vrf_share", zap.Any("round", vrfs.Round),
		zap.Any("round_timeout", vrfs.RoundTimeoutCount),
		zap.Int64("dkg_starting_round", dkg.StartingRound),
		zap.Int64("dkg_mb_number", dkg.MagicBlockNumber),
		zap.String("pr_seed", strconv.FormatInt(pr.GetRandomSeed(), 16)))

	vrfs.SetParty(node.Self.Underlying())
	r.vrfShare = vrfs
	// TODO: do we need to check if AddVRFShare is success or not?
	if mc.AddVRFShare(ctx, r, vrfs) {
		go mc.SendVRFShare(ctx, vrfs.Clone())
	}
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
		logging.Logger.Debug("[is ahead]", zap.Int64("round", round),
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
		ahead = config.GetLFBTicketAhead()
		tk    = mc.GetLatestLFBTicket(ctx)
	)

	if tk == nil {
		logging.Logger.Debug("[wait not ahead] [1] context is done or restart round")
		return // context is done, can't wait anymore
	}

	if round+1 <= tk.Round+int64(ahead) {
		logging.Logger.Debug("[wait not ahead] [2] not ahead, can move on")
		return true // not ahead, can move on
	}

	// wait in loop
	for {
		select {
		case ntk := <-tksubq: // the ntk can't be nil
			if round+1 <= ntk.Round+int64(ahead) {
				logging.Logger.Debug("[wait not ahead] [3] not ahead, can move on")
				return true // not ahead, can move on
			}
			logging.Logger.Debug("[wait not ahead] [4] still ahead, can't move on")
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
	logging.Logger.Debug("finalizedRound - cancel round verification")
	mc.CancelRoundVerification(ctx, r)
	go mc.FinalizeRound(ctx, r.Round, mc)
}

func (mc *Chain) pullNotarizedBlocks(ctx context.Context, r *Round) {
	logging.Logger.Info("pull not. block for", zap.Int64("round", r.GetRoundNumber()))
	if mc.GetHeaviestNotarizedBlock(ctx, r) != nil {
		if r.GetRoundNumber() >= mc.GetCurrentRound() {
			mc.SetCurrentRound(r.GetRoundNumber())
			mc.StartNextRound(ctx, r)
		}
	}
}

// StartNextRound - start the next round as a notarized
// block is discovered for the current round.
func (mc *Chain) StartNextRound(ctx context.Context, r *Round) *Round {

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

	if er != mr && mc.isStarted() {
		logging.Logger.Info("StartNextRound found next round ready. No VRFs Sent",
			zap.Int64("er_round", er.GetRoundNumber()),
			zap.Int64("rrs", r.GetRandomSeed()),
			zap.Bool("is_started", mc.isStarted()))
		return er
	}

	if r.HasRandomSeed() {
		mc.addMyVRFShare(ctx, r, er)
	} else {
		logging.Logger.Info("StartNextRound no VRFs sent -- "+
			"current round has no random seed",
			zap.Int64("rrs", r.GetRandomSeed()), zap.Int64("r_round", rn))
	}

	return er
}

// The getOrStartRound returns existing miner round, or starts new if there is
// previous one. It never checks and waits 'ahead of sharders' cases.
func (mc *Chain) getOrStartRound(ctx context.Context, rn int64) (
	mr *Round) {

	if mr = mc.GetMinerRound(rn); mr != nil {
		return // got existing round
	}

	var pr = mc.GetMinerRound(rn - 1)
	if pr == nil {
		logging.Logger.Error("get_or_start_round -- no previous round",
			zap.Int64("current_round", mc.GetCurrentRound()),
			zap.Int64("round", rn),
			zap.Int64("pr", rn-1))
		return
	}

	logging.Logger.Info("start round in getOrStartRound", zap.Int64("round", rn))
	return mc.StartNextRound(ctx, pr) // can return nil
}

func (mc *Chain) getOrStartRoundNotAhead(ctx context.Context, rn int64) (
	mr *Round) {

	if mr = mc.GetMinerRound(rn); mr != nil {
		return // already have the round
	}

	if mc.isAheadOfSharders(ctx, rn) {
		return // ahead of sharders, don't start new round anyway
	}

	// get existing or start new round checking previous round first
	return mc.getOrStartRound(ctx, rn)
}

// The getOrCreateRound returns existing round or creates new one. It used for
// LFB, and ignores 'aheadness' and previous round presence.
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
		r.vrfShare = nil
		logging.Logger.Info("RedoVrfShare after vrfShare is nil",
			zap.Int64("round", r.GetRoundNumber()),
			zap.Int("round_timeout", r.GetTimeoutCount()))
		mc.addMyVRFShare(ctx, pr, r)
		return true
	}

	return false
}

func (mc *Chain) getClientState(
	ctx context.Context,
	roundNumber int64,
) *util.MerklePatriciaTrie {
	pround := mc.GetRound(roundNumber - 1)
	pb := mc.GetBlockToExtend(ctx, pround)
	pndb := pb.ClientState.GetNodeDB()
	root := pb.ClientStateHash
	mndb := util.NewMemoryNodeDB()
	ndb := util.NewLevelNodeDB(mndb, pndb, false)
	clientState := util.NewMerklePatriciaTrie(ndb, util.Sequence(roundNumber-1), root)
	return clientState
}

func (mc *Chain) getConfigMap(clientState *util.MerklePatriciaTrie) (*minersc.GlobalSettings, error) {
	key_hash := encryption.Hash(minersc.GLOBALS_KEY)
	val, err := clientState.GetNodeValue(util.Path(key_hash))
	if err != nil {
		return nil, err
	}

	gl := &minersc.GlobalSettings{
		Fields: make(map[string]string),
	}
	err = gl.Decode(val.Encode())
	if err != nil {
		return nil, err
	}

	return gl, nil
}

func (mc *Chain) startRound(ctx context.Context, r *Round, seed int64) {
	if !mc.SetRandomSeed(r.Round, seed) {
		return
	}

	configMap, err := mc.getConfigMap(mc.getClientState(ctx, r.GetRoundNumber()))
	if err == nil {
		mc.Config.Update(configMap)
	}

	mc.startNewRound(ctx, r)
}

func (mc *Chain) startNewRound(ctx context.Context, mr *Round) {
	var rn = mr.GetRoundNumber()

	if rn < mc.GetCurrentRound() {
		logging.Logger.Debug("start new round (current round higher)",
			zap.Int64("round", rn),
			zap.Int64("current_round", mc.GetCurrentRound()))
		return
	}

	if pr := mc.GetRound(rn - 1); pr == nil {
		logging.Logger.Debug("start new round (previous round not found)",
			zap.Int64("round", rn))
		return
	}

	var (
		self = node.Self.Underlying()
		rank = mr.GetMinerRank(self)
	)

	if !mc.IsRoundGenerator(mr, self) {
		logging.Logger.Info("TOC_FIX Not a generator", zap.Int64("round", rn),
			zap.Int("index", self.SetIndex),
			zap.Int("rank", rank),
			zap.Int("timeout_count", mr.GetTimeoutCount()),
			zap.Any("random_seed", mr.GetRandomSeed()))
		return
	}

	logging.Logger.Info("*** TOC_FIX starting round block generation ***",
		zap.Int64("round", rn), zap.Int("index", self.SetIndex),
		zap.Int("rank", rank), zap.Int("timeout_count", mr.GetTimeoutCount()),
		zap.Any("random_seed", mr.GetRandomSeed()),
		zap.Int64("lf_round", mc.GetLatestFinalizedBlock().Round))

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
	pround := mc.GetRound(roundNumber - 1)
	if pround == nil {
		logging.Logger.Error("generate round block - no prior round", zap.Any("round", roundNumber-1))
		return nil, common.NewError("invalid_round,", "Round not available")
	}

	pb := mc.GetBlockToExtend(ctx, pround)
	if pb == nil {
		logging.Logger.Error("generate round block - no block to extend", zap.Any("round", roundNumber))
		return nil, common.NewError("block_gen_no_block_to_extend", "Do not have the block to extend this round")
	}

	if !pb.IsStateComputed() {
		logging.Logger.Debug("GenerateRoundBlock, state of prior round block not computed",
			zap.Any("state status", pb.GetStateStatus()))
	}

	txnEntityMetadata := datastore.GetEntityMetadata("txn")
	ctx = memorystore.WithEntityConnection(ctx, txnEntityMetadata)
	defer memorystore.Close(ctx)

	var (
		rn    = r.GetRoundNumber()                       //
		b     = block.NewBlock(mc.GetKey(), rn)          //
		lfmbr = mc.GetLatestFinalizedMagicBlockRound(rn) // related magic block

		rnoff = mbRoundOffset(rn)   //
		nvc   = mc.NextViewChange() // we can use LFB because there is VC offset
	)

	if nvc > 0 && rnoff >= nvc && lfmbr.StartingRound < nvc {
		logging.Logger.Error("gen_block",
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
	for true {
		if mc.GetCurrentRound() > b.Round {
			logging.Logger.Error("generate block - round mismatch",
				zap.Any("round", roundNumber),
				zap.Any("current_round", mc.GetCurrentRound()))
			return nil, ErrRoundMismatch
		}

		txnCount := transaction.GetTransactionCount()
		generationTries++
		if !pb.IsStateComputed() {
			logging.Logger.Debug("GenerateRoundBlock, previous block state is not computed",
				zap.Int64("round", b.Round),
				zap.Int64("pre round", pb.Round),
				zap.Any("prior_block_state", pb.GetStateStatus()))
		}

		b.SetStateDB(pb, mc.GetStateDB())

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
							logging.Logger.Info("generate block",
								zap.Any("round", roundNumber),
								zap.Any("delay", delay),
								zap.Any("txn_count", txnCount),
								zap.Any("t.txn_count", transaction.GetTransactionCount()),
								zap.Any("error", cerr))
						}
						if mc.GetCurrentRound() > b.Round {
							logging.Logger.Error("generate block - round mismatch",
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
					logging.Logger.Info("generate block", zap.Error(err))
					continue
				case RoundTimeout:
					logging.Logger.Error("generate block",
						zap.Int64("round", roundNumber),
						zap.Error(err))
					return nil, err
				}
			}
			if startLogging.IsZero() || time.Now().Sub(startLogging) > time.Second {
				startLogging = time.Now()
				logging.Logger.Info("generate block", zap.Any("round", roundNumber),
					zap.Any("txn_count", txnCount),
					zap.Any("t.txn_count", transaction.GetTransactionCount()),
					zap.Any("error", err))
			}
			return nil, err
		}

		if r.GetRandomSeed() != b.GetRoundRandomSeed() {
			logging.Logger.Error("round random seed mismatch",
				zap.Int64("round", b.Round),
				zap.Int64("round_rrs", r.GetRandomSeed()),
				zap.Int64("blk_rrs", b.GetRoundRandomSeed()))
			return nil, ErrRRSMismatch
		}

		mc.AddRoundBlock(r, b)
		if generationTries > 1 {
			logging.Logger.Info("generate block - multiple tries",
				zap.Int64("round", b.Round), zap.Int("tries", generationTries))
		}
		break
	}

	if r.IsVerificationComplete() {
		logging.Logger.Error("generate block - verification complete",
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

	if b.Round > 1 {
		pb, err := mc.GetBlock(ctx, b.PrevHash)
		if err != nil || pb == nil {
			return
		}

		err = mc.VerifyNotarization(ctx, pb, b.GetPrevBlockVerificationTickets(), pr.GetRoundNumber())
		if err != nil {
			logging.Logger.Error("add to verification (prior block verify notarization)",
				zap.Int64("round", pr.Number), zap.Any("miner_id", b.MinerID),
				zap.String("block", b.PrevHash),
				zap.Int("v_tickets", b.PrevBlockVerificationTicketsSize()),
				zap.Error(err))
			return
		}
	}

	if err := mc.ComputeState(ctx, b); err != nil {
		logging.Logger.Error("AddToRoundVerification compute state failed", zap.Error(err))
		return
	}

	mr.AddProposedBlock(b)
	if mc.AddRoundBlock(mr, b) != b {
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
			logging.Logger.Error("add to verification (wrong chain weight)",
				zap.Int64("round", b.Round), zap.String("block", b.Hash),
				zap.Float64("chain_weight", b.ChainWeight))
			return
		}
	}

	mc.addToRoundVerification(ctx, mr, b)
}

func (mc *Chain) addToRoundVerification(ctx context.Context, mr *Round, b *block.Block) {
	logging.Logger.Info("adding block to verify",
		zap.Int64("round", b.Round),
		zap.String("block", b.Hash),
		zap.String("prev_block", b.PrevHash),
		zap.String("state_hash", util.ToHex(b.ClientStateHash)),
		zap.Float64("weight", b.Weight()),
		zap.Float64("chain_weight", b.ChainWeight))
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
			logging.Logger.Error("verify round block -- failed miner",
				zap.Any("round", r.Number), zap.Any("block", b.Hash),
				zap.Any("miner", b.MinerID))
			b.SetBlockState(block.StateVerificationFailed)
			return false
		}
		minerStats := miner.ProtocolStats.(*chain.MinerStats)
		bvt, err := mc.VerifyRoundBlock(ctx, r, b)
		if err != nil {
			b.SetBlockState(block.StateVerificationFailed)
			minerStats.VerificationFailures++
			switch err {
			case context.Canceled:
			default:
				if cerr, ok := err.(*common.Error); ok {
					if cerr.Code == RoundMismatch {
						logging.Logger.Debug("verify round block",
							zap.Any("round", r.Number), zap.Any("block", b.Hash),
							zap.Any("current_round", mc.GetCurrentRound()))
					} else {
						logging.Logger.Error("verify round block",
							zap.Any("round", r.Number), zap.Any("block", b.Hash),
							zap.Error(err))
					}
				} else {
					logging.Logger.Error("verify round block", zap.Any("round", r.Number),
						zap.Any("block", b.Hash), zap.Error(err))
				}
			}
			return false
		}
		b.SetBlockState(block.StateVerificationSuccessful)
		bnb := r.GetBestRankedNotarizedBlock()
		if bnb == nil || bnb.Hash == b.Hash {
			logging.Logger.Info("Sending verification ticket", zap.Int64("round", r.Number), zap.String("block", b.Hash))
			go mc.SendVerificationTicket(ctx, b, bvt)
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
		return true
	}
	var sendVerification = false
	var blocks = make([]*block.Block, 0, 8)
	initiateVerification := func() {
		// Sort the accumulated blocks by the rank and process them
		blocks = r.GetBlocksByRank(blocks)
		// Keep verifying all the blocks collected so far in the best rank order
		// till the first successful verification.
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
	for {
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
					logging.Logger.Info("verification cancel (failing block)", zap.Int64("round", r.Number), zap.String("block", b.Hash), zap.Int("block_rank", b.RoundRank), zap.Int("best_rank", bRank), zap.Int8("block_state", bs))
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
func (mc *Chain) VerifyRoundBlock(ctx context.Context, r round.RoundI, b *block.Block) (*block.BlockVerificationTicket, error) {
	if !mc.CanShardBlocks(r.GetRoundNumber()) {
		return nil, common.NewError("fewer_active_sharders", "Number of active sharders not sufficient")
	}
	if !mc.CanReplicateBlock(b) {
		return nil, common.NewError("fewer_active_replicators", "Number of active replicators not sufficient")
	}
	if mc.GetCurrentRound() != r.GetRoundNumber() {
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
		mc.updatePriorBlock(ctx, r, b)
	}
	return bvt, nil
}

func (mc *Chain) updatePriorBlock(ctx context.Context, r round.RoundI, b *block.Block) {
	pb := b.PrevBlock
	mc.MergeVerificationTickets(ctx, pb, b.GetPrevBlockVerificationTickets())
	pr := mc.GetMinerRound(pb.Round)
	if pr != nil {
		mc.AddNotarizedBlock(ctx, pr, pb)
	} else {
		logging.Logger.Error("verify round - previous round not present",
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
	if !mc.AddVerificationTicket(ctx, b, vt) {
		return
	}

	if notarized {
		logging.Logger.Info("Block is notarized", zap.Int64("round", r.Number),
			zap.String("block", b.Hash))
		return
	}

	mc.checkBlockNotarization(ctx, r, b)
}

func (mc *Chain) checkBlockNotarization(ctx context.Context, r *Round, b *block.Block) bool {
	if !b.IsBlockNotarized() {
		logging.Logger.Error("checkBlockNotarization -- block is not Notarized. Returning",
			zap.Int64("round", b.Round),
			zap.Any("block hash", b.Hash))
		return false
	}
	if !mc.AddNotarizedBlock(ctx, r, b) {
		return true
	}

	seed := b.GetRoundRandomSeed()
	if seed == 0 {
		logging.Logger.Error("checkBlockNotarization -- block random seed is 0", zap.Int64("round", b.Round))
		return false
	}

	mc.SetRandomSeed(r, seed)
	go mc.SendNotarization(ctx, b)

	logging.Logger.Debug("check block notarization - block notarized",
		zap.Int64("round", b.Round), zap.String("block", b.Hash))

	// start next round if not ahead of sharders
	go mc.startNextRoundNotAhead(common.GetRootContext(), r)
	return true
}

func (mc *Chain) startNextRoundNotAhead(ctx context.Context, r *Round) {
	var rn = r.GetRoundNumber()
	if !mc.waitNotAhead(ctx, rn) {
		logging.Logger.Debug("start next round not ahead -- terminated",
			zap.Int64("round", rn))
		return // terminated
	}
	mc.StartNextRound(ctx, r)
}

// MergeNotarization - merge a notarization.
func (mc *Chain) MergeNotarization(ctx context.Context, r *Round, b *block.Block, vts []*block.VerificationTicket) {
	for _, t := range vts {
		if err := mc.VerifyTicket(ctx, b.Hash, t, r.GetRoundNumber()); err != nil {
			logging.Logger.Error("merge notarization", zap.Int64("round", b.Round),
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
	_, ok, err := r.AddNotarizedBlock(b)
	if err != nil {
		logging.Logger.Error("add notarized block failed",
			zap.Int64("round", r.GetRoundNumber()),
			zap.String("block", b.Hash),
			zap.Error(err))
		return false
	}

	if !ok {
		return false
	}

	mc.UpdateNodeState(b)

	if !b.IsStateComputed() {
		logging.Logger.Info("add notarized block - computing state",
			zap.Int64("round", b.Round), zap.String("block", b.Hash))
		mc.ComputeState(ctx, b)
	}

	if !r.IsVerificationComplete() {
		logging.Logger.Debug("AddNotarizedBlock - cancel round verification")
		mc.CancelRoundVerification(ctx, r)
	}
	b.SetBlockState(block.StateNotarized)
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

// GetLatestFinalizedBlockFromSharder - request for latest finalized block from
// all the sharders.
func (mc *Chain) GetLatestFinalizedBlockFromSharder(ctx context.Context) (
	fbs []*BlockConsensus) {

	var mx sync.Mutex
	var handler = func(ctx context.Context, entity datastore.Entity) (
		resp interface{}, err error) {

		var fb, ok = entity.(*block.Block)
		if !ok {
			return nil, datastore.ErrInvalidEntity
		}

		if fb.Round == 0 {
			return
		}

		if err = fb.Validate(ctx); err != nil {
			logging.Logger.Error("lfb from sharder - invalid",
				zap.Int64("round", fb.Round), zap.String("block", fb.Hash),
				zap.Error(err))
			return
		}

		err = mc.VerifyNotarization(ctx, fb, fb.GetVerificationTickets(),
			fb.Round)
		if err != nil {
			logging.Logger.Error("lfb from sharder - notarization failed",
				zap.Int64("round", fb.Round), zap.String("block", fb.Hash),
				zap.Error(err))
			return nil, err
		}

		// don't use the round, just create it or make sure it's created
		mc.getOrCreateRound(ctx, fb.Round) // can' return nil

		// async safe
		mx.Lock()
		defer mx.Unlock()

		// increase consensus
		for i, b := range fbs {
			if b.Hash == fb.Hash {
				fbs[i].Consensus++
				return fb, nil
			}
		}

		// add new block
		fbs = append(fbs, &BlockConsensus{
			Block:     fb,
			Consensus: 1,
		})

		return fb, nil
	}

	fbs = make([]*BlockConsensus, 0, len(mc.GetLatestFinalizedMagicBlockBrief().ShardersN2NURLs))
	mc.RequestEntityFromSharders(ctx, MinerLatestFinalizedBlockRequestor, nil, handler)

	// highest (the first sorting order), most popular (the second order)
	sort.Slice(fbs, func(i int, j int) bool {
		return fbs[i].Round >= fbs[j].Round ||
			fbs[i].Consensus > fbs[j].Consensus

	})

	return
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

	var handler = func(ctx context.Context, entity datastore.Entity) (
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

		err = mc.VerifyNotarization(ctx, fb, fb.GetVerificationTickets(),
			fb.Round)
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
	tick := mc.RoundTimeoutSofttoMin
	if tick < mc.RoundTimeoutSofttoMult*ssft {
		tick = mc.RoundTimeoutSofttoMult * ssft
	}
	logging.Logger.Info("nextTimeout", zap.Int("tick", tick))
	return tick
}

// HandleRoundTimeout handle timeouts appropriately.
func (mc *Chain) HandleRoundTimeout(ctx context.Context, round int64) {
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

	if r.GetSoftTimeoutCount() == mc.RoundRestartMult {
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

func (mc *Chain) handleNoProgress(ctx context.Context, round int64) {
	r := mc.GetMinerRound(round)
	proposals := r.GetProposedBlocks()
	if len(proposals) > 0 { // send the best block to the network
		b := r.Block
		if b != nil {
			if mc.GetRoundTimeoutCount() <= 10 {
				logging.Logger.Info("sending the best block to the network",
					zap.Int64("round", b.Round), zap.String("block", b.Hash),
					zap.Int("rank", b.RoundRank))
			}
			lfmbr := mc.GetLatestFinalizedMagicBlockRound(round) // related magic block
			if lfmbr.Hash != b.LatestFinalizedMagicBlockHash {
				logging.Logger.Error("handleNoProgress mismatch latest finalized magic block",
					zap.Any("lfmbr hash", lfmbr.Hash),
					zap.Any("block lfmbr hash", b.LatestFinalizedMagicBlockHash),
					zap.Int64("lfmbr starting round", lfmbr.Round),
					zap.Int64("block lfmbr starting round", b.LatestFinalizedMagicBlockRound))
			} else {
				logging.Logger.Debug("handleNoProgress match latest finalized magic block",
					zap.Any("lfmbr hash", lfmbr.Hash),
					zap.Int64("lfmbr round", lfmbr.Round))
			}
			mc.SendBlock(ctx, b)
		}
	}

	if r.vrfShare != nil {
		go mc.SendVRFShare(ctx, r.vrfShare)
		logging.Logger.Info("Sent vrf shares in handle NoProgress")
	} else {
		logging.Logger.Info("Did not send vrf shares as it is nil", zap.Int64("round_num", r.GetRoundNumber()))
	}
	switch crt := mc.GetRoundTimeoutCount(); {
	case crt < 10:
		logging.Logger.Info("handleNoProgress",
			zap.Any("round", mc.GetCurrentRound()),
			zap.Int64("count_round_timeout", crt),
			zap.Any("num_vrf_share", len(r.GetVRFShares())))
	case crt == 10:
		logging.Logger.Error("handleNoProgress (no further timeout messages will be displayed)",
			zap.Any("round", mc.GetCurrentRound()),
			zap.Int64("count_round_timeout", crt),
			zap.Any("num_vrf_share", len(r.GetVRFShares())))
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
			logging.Logger.Info("restartRound->kickFinalization continued",
				zap.Any("miner round", mr),
				zap.Bool("miner is finalized", mr.IsFinalized()))
			i++
			count++
			continue // skip finalized blocks, skip nil miner rounds
		}
		logging.Logger.Info("restartRound->kickFinalization:",
			zap.Int64("round", mr.GetRoundNumber()))
		go mc.FinalizeRound(ctx, mr.Round, mc) // kick finalization again
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
			mr.Block.GetStateStatus() == block.StateSuccessful {

			logging.Logger.Info("restartRound->kickSharders: kick sharder FB",
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
		// ignore state error
	}

	mr, _ = mc.AddRound(mr).(*Round)
	mc.SetRandomSeed(sr, lfb.RoundRandomSeed)
	mc.AddBlock(lfb)
	if nr = mc.StartNextRound(ctx, mr); nr == nil {
		return
	}
	mc.SetCurrentRound(nr.Number)
}

func (mc *Chain) getRoundRandomSeed(rn int64) (seed int64) {
	var mr = mc.GetMinerRound(rn)
	if mr == nil {
		return // zero, no seed
	}
	return mr.GetRandomSeed() // can be zero too
}

func (mc *Chain) startNextRoundInRestartRound(ctx context.Context, i int64) {

	// don't start the round if the miner is ahead of sharders
	if mc.isAheadOfSharders(ctx, i) {
		logging.Logger.Error("[start next round in RR] is ahead, don't start")
		return
	}

	// previous round required
	var pr = mc.GetMinerRound(i - 1)
	if pr == nil {
		logging.Logger.Error("[start next round in RR] no previous round",
			zap.Int64("pr", i-1)) // critical, critical, critical, critical
		return
	}

	// start the next
	mc.StartNextRound(ctx, pr)
}

func (mc *Chain) restartRound(ctx context.Context, rn int64) {

	mc.sendRestartRoundEvent(ctx) // trigger restart round event

	mc.IncrementRoundTimeoutCount()
	var r = mc.GetMinerRound(rn)

	switch crt := mc.GetRoundTimeoutCount(); {
	case crt < 10:
		logging.Logger.Error("restartRound - round timeout occurred",
			zap.Any("round", rn), zap.Int64("count", crt),
			zap.Any("num_vrf_share", len(r.GetVRFShares())))

	case crt == 10:
		logging.Logger.Error("restartRound - round timeout occurred (no further"+
			" timeout messages will be displayed)",
			zap.Any("round", rn), zap.Int64("count", crt),
			zap.Any("num_vrf_share", len(r.GetVRFShares())))

		// TODO: should have a means to send an email/SMS to someone or
		// something like that
	}
	mc.RoundTimeoutsCount++

	// get LFMB and LFB from sharders
	var updated, err = mc.ensureLatestFinalizedBlocks(ctx)
	if err != nil {
		logging.Logger.Error("restartRound - ensure lfb", zap.Error(err))
	}

	var (
		isAhead = mc.isAheadOfSharders(ctx, rn)
		lfb     = mc.GetLatestFinalizedBlock()
	)

	// initialize rrs for lfb round
	if lfb.Round > 0 {
		lfbr := mc.GetMinerRound(lfb.Round)
		if lfbr == nil {
			lfbr = mc.AddRound(mc.CreateRound(round.NewRound(lfb.Round))).(*Round)
		}
		if lfb.RoundRandomSeed != 0 && lfbr.RandomSeed != lfb.RoundRandomSeed {
			lfbr.SetRandomSeedForNotarizedBlock(lfb.RoundRandomSeed, 0)
		}
	}

	// kick new round from the new LFB from sharders, if it's newer
	// than the current one
	if updated {
		if lfb.Round > rn {
			mc.kickRoundByLFB(ctx, lfb) // and continue
			//round = mc.GetCurrentRound()
		}
	} else if isAhead {
		mc.kickSharders(ctx) // not updated, kick sharders finalization
	}

	// walk up for first round with no not. block of all nodes
	for i := lfb.Round + 1; ; i++ {
		var xr = mc.GetMinerRound(i)
		if xr == nil {
			mc.startNextRoundInRestartRound(ctx, i)
			return // <============================================= [exit loop]
		}

		if xr.IsFinalized() || xr.IsFinalizing() {
			continue // skip rounds finalizing or finalized <=== [continue loop]
		}
		// check out corresponding not. block
		var xrhnb = xr.GetHeaviestNotarizedBlock()
		if xrhnb == nil || (xrhnb != nil && xrhnb.GetRoundRandomSeed() == 0) {
			mc.pullNotarizedBlocks(ctx, xr)        // try to pull a not. block
			xrhnb = xr.GetHeaviestNotarizedBlock() //
		}
		// if no not. block for the round, then we just redo VRFS sending
		// (previous round random seed required for it)
		if xrhnb == nil {
			xr.Restart()
			xr.IncrementTimeoutCount(mc.getRoundRandomSeed(i-1), mc.GetMiners(i))
			mc.RedoVrfShare(ctx, xr)
			return // the round has restarted <===================== [exit loop]
		}

		xrhnb, _, err = mc.AddNotarizedBlockToRound(xr, xrhnb)
		if err != nil {
			logging.Logger.Error("restartRound failed",
				zap.Int64("round", i),
				zap.String("block", xrhnb.Hash),
				zap.Error(err))
			return
		}

		if !xrhnb.IsStateComputed() {
			lfmb := mc.GetLatestFinalizedMagicBlockRound(xr.GetRoundNumber())
			if lfmb != nil && lfmb.Miners.HasNode(node.Self.GetKey()) {
				if err := mc.ComputeOrSyncState(ctx, xrhnb); err != nil {
					logging.Logger.Debug("restartRound: Notarized block ComputeOrSyncState", zap.Error(err))
				}
			}
		}
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
			logging.Logger.Error("ensure_state -- compute or sync",
				zap.Error(err), zap.Int64("round", b.Round))
		}
	}
	if b.ClientState == nil {
		if err = mc.InitBlockState(b); err != nil {
			logging.Logger.Error("ensure_state -- initialize block state",
				zap.Error(err), zap.Int64("round", b.Round))
		}
	}

	// ensure next view change (from sharders)
	if ok = (b.IsStateComputed() && b.ClientState != nil); ok {
		var nvc int64
		if nvc, err = mc.NextViewChangeOfBlock(b); err != nil {
			logging.Logger.Error("ensure_state -- next view change",
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
		logging.Logger.Error("setting DKG from store",
			zap.Int64("mb_round", mb.Round), zap.Error(err))
	}
}

func (mc *Chain) ensureLatestFinalizedBlocks(ctx context.Context) (
	updated bool, err error) {

	// So, there is worker that updates LFMB from configured 0DNS sever.
	// This MB used to request other nodes.

	defer mc.SetupLatestAndPreviousMagicBlocks(ctx)

	if updated, err = mc.ensureLatestFinalizedBlock(ctx); err != nil {
		return
	}

	// LFMB. The LFMB can be already update by LFMD worker (which uses 0DNS)
	// and here we just set correct DKG.

	rcvd := mc.GetLatestFinalizedMagicBlockFromSharders(ctx)
	if rcvd == nil {
		return
	}

	lfmb := mc.GetLatestFinalizedMagicBlock()
	mc.ensureDKG(ctx, lfmb)

	if lfmb != nil && rcvd.MagicBlockNumber <= lfmb.MagicBlockNumber {
		logging.Logger.Debug("lfmb from sharders has MagicBlockNumber <= lfmb",
			zap.Int64("sharder lfmb number", rcvd.MagicBlockNumber),
			zap.Int64("local lfmb number", lfmb.MagicBlockNumber))
		return
	}

	if err = mc.VerifyChainHistoryAndRepair(ctx, rcvd, nil); err != nil {
		return false, err
	}
	if err = mc.UpdateMagicBlock(rcvd.MagicBlock); err != nil {
		return false, err
	}
	mc.ensureDKG(ctx, rcvd)
	mc.SetLatestFinalizedMagicBlock(rcvd)

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
				logging.Logger.Error("getting latest blocks from sharders",
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

	// latest MB
	var (
		latest *block.MagicBlock
		err    error
	)
	if latest, err = LoadLatestMB(ctx); err != nil {
		logging.Logger.Error("load_mbs_and_dkg -- loading the latest MB",
			zap.Error(err))
		return // can't continue
	}

	// don't setup the latest MB since it can be promoted

	if latest.MagicBlockNumber <= 1 {
		return // done
	}
	// otherwise, load and setup previous
	var (
		prev *block.MagicBlock
		id   = strconv.FormatInt(latest.MagicBlockNumber-1, 10)
	)
	if prev, err = LoadMagicBlock(ctx, id); err != nil {
		logging.Logger.Info("load_mbs_and_dkg -- loading previous MB",
			zap.String("sr", id), zap.Error(err))
		return // can't continue
	}
	if err = mc.setupLoadedMagicBlock(prev); err != nil {
		logging.Logger.Info("load_mbs_and_dkg -- updating previous MB",
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
		logging.Logger.Info("load_mbs_and_dkg -- loading previous DKG",
			zap.Error(err))
	}
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

	var waitingSharders = make([]string, 0, lmb.Sharders.MapSize())
	for _, nodeSharder := range lmb.Sharders.CopyNodesMap() {
		waitingSharders = append(waitingSharders,
			fmt.Sprintf("id: %s; n2nhost: %s ", nodeSharder.ID, nodeSharder.N2NHost))
	}

	logging.Logger.Debug("waiting for sharders",
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
			logging.Logger.Info("Waiting for Sharders.", zap.Time("ts", ts),
				zap.Any("sharders", waitingSharders))
			lmb.Sharders.OneTimeStatusMonitor(ctx, lmb.StartingRound) // just mark 'em active
		}
	}
}
