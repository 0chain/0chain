package miner

import (
	"context"
	"math"
	"sort"
	"sync"
	"time"

	metrics "github.com/rcrowley/go-metrics"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	. "0chain.net/core/logging"
	"0chain.net/core/memorystore"
	"0chain.net/core/util"
	"go.uber.org/zap"
)

var rbgTimer metrics.Timer // round block generation timer

func init() {
	rbgTimer = metrics.GetOrRegisterTimer("rbg_time", nil)
}

//SetNetworkRelayTime - set the network relay time
func SetNetworkRelayTime(delta time.Duration) {
	chain.SetNetworkRelayTime(delta)
}

/*StartNextRound - start the next round as a notarized block is discovered for the current round */
func (mc *Chain) StartNextRound(ctx context.Context, r *Round) *Round {
	pr := mc.GetMinerRound(r.GetRoundNumber() - 1)
	if pr != nil {
		mc.CancelRoundVerification(ctx, pr)
		go mc.FinalizeRound(ctx, pr.Round, mc)
	}
	var nr = round.NewRound(r.GetRoundNumber() + 1)
	mr := mc.CreateRound(nr)
	er := mc.AddRound(mr)
	if er != mr {
		Logger.Info("StartNextRound found nextround ready. No VRFs Sent",
			zap.Int64("er_round", er.GetRoundNumber()), zap.Int64("rrs", r.GetRandomSeed()))
		return er.(*Round)
	}
	if r.HasRandomSeed() {
		mc.addMyVRFShare(ctx, r, mr)
	} else {
		Logger.Info("StartNextRound no VRFs sent -current round has no randomseed",
			zap.Int64("rrs", r.GetRandomSeed()), zap.Int64("r_round", r.GetRoundNumber()))

	}
	return mr
}

func (mc *Chain) getRound(ctx context.Context, roundNumber int64) *Round {
	var mr *Round
	pr := mc.GetMinerRound(roundNumber - 1)
	if pr != nil {
		Logger.Info("Starting next round in getRound", zap.Int64("nextRoundNum", roundNumber))
		mr = mc.StartNextRound(ctx, pr)
	} else {
		var r = round.NewRound(roundNumber)
		mr = mc.CreateRound(r)
		mr = mc.AddRound(mr).(*Round)
	}
	return mr
}

// RedoVrfShare re-calculateVrfShare and send
func (mc *Chain) RedoVrfShare(ctx context.Context, r *Round) bool {
	var pr *Round
	if r.GetRoundNumber() > 0 {
		pr = mc.GetMinerRound(r.GetRoundNumber() - 1)
	} else {
		pr = r
	}
	if pr == nil {
		Logger.Info("no pr info inside RedoVrfShare", zap.Int64("Round", r.GetRoundNumber()))
		return false
	}
	if pr.HasRandomSeed() {
		r.vrfShare = nil
		Logger.Info("RedoVrfShare after vrfShare is nil",
			zap.Int64("round", r.GetRoundNumber()), zap.Int("round_timeout", r.GetTimeoutCount()))
		mc.addMyVRFShare(ctx, pr, r)
		return true
	}
	return false
}

func (mc *Chain) addMyVRFShare(ctx context.Context, pr *Round, r *Round) {
	var err error
	vrfs := &round.VRFShare{}
	vrfs.Round = r.GetRoundNumber()
	vrfs.RoundTimeoutCount = r.GetTimeoutCount()
	vrfs.Share, err = mc.GetBlsShare(ctx, r.Round)
	if err != nil {
		Logger.DPanic(err.Error())
	}
	vrfs.SetParty(node.Self.Underlying())
	r.vrfShare = vrfs
	// TODO: do we need to check if AddVRFShare is success or not?
	mc.AddVRFShare(ctx, r, r.vrfShare)
	go mc.SendVRFShare(ctx, r.vrfShare)

}

func (mc *Chain) startRound(ctx context.Context, r *Round, seed int64) {
	if !mc.SetRandomSeed(r.Round, seed) {
		return
	}
	Logger.Info("Starting a new round", zap.Int64("round", r.GetRoundNumber()))
	mc.ViewChange(ctx, r.Number)
	mc.startNewRound(ctx, r)
}

func (mc *Chain) startNewRound(ctx context.Context, mr *Round) {
	if mr.GetRoundNumber() < mc.GetCurrentRound() {
		Logger.Debug("start new round (current round higher)", zap.Int64("round", mr.GetRoundNumber()), zap.Int64("current_round", mc.GetCurrentRound()))
		return
	}
	pr := mc.GetRound(mr.GetRoundNumber() - 1)
	if pr == nil {
		Logger.Debug("start new round (previous round not found)", zap.Int64("round", mr.GetRoundNumber()))
		return
	}

	self := node.GetSelfNode(ctx)
	rank := mr.GetMinerRank(self.Underlying())
	if !mc.IsRoundGenerator(mr, self.Underlying()) {
		Logger.Info("TOC_FIX Not a generator", zap.Int64("round", mr.GetRoundNumber()), zap.Int("index", self.Underlying().SetIndex), zap.Int("rank", rank), zap.Int("timeoutcount", mr.GetTimeoutCount()), zap.Any("random_seed", mr.GetRandomSeed()))
		return
	}
	Logger.Info("*** TOC_FIX starting round block generation ***", zap.Int64("round", mr.GetRoundNumber()), zap.Int("index", self.Underlying().SetIndex), zap.Int("rank", rank), zap.Int("timeoutcount", mr.GetTimeoutCount()), zap.Any("random_seed", mr.GetRandomSeed()), zap.Int64("lf_round", mc.GetLatestFinalizedBlock().Round))

	//NOTE: If there are not enough txns, this will not advance further even though rest of the network is. That's why this is a goroutine
	go mc.GenerateRoundBlock(ctx, mr)
}

/*GetBlockToExtend - Get the block to extend from the given round */
func (mc *Chain) GetBlockToExtend(ctx context.Context, r round.RoundI) *block.Block {
	bnb := r.GetHeaviestNotarizedBlock()
	if bnb == nil {
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
		sort.SliceStable(pcounts, func(i, j int) bool { return pcounts[i].Proposals > pcounts[j].Proposals })
		Logger.Error("get block to extend - no notarized block", zap.Int64("round", r.GetRoundNumber()), zap.Int("num_proposals", len(proposals)), zap.Any("verification_tickets", pcounts))
		bnb = mc.GetHeaviestNotarizedBlock(r)
	}
	if bnb != nil {
		if !bnb.IsStateComputed() {
			err := mc.ComputeOrSyncState(ctx, bnb)
			if err != nil {
				if state.DebugBlock() {
					Logger.Error("get block to extend - best nb compute state", zap.Any("round", r.GetRoundNumber()), zap.Any("block", bnb.Hash), zap.Error(err))
					return nil
				}
			}
		}
		return bnb
	}
	Logger.Debug("get block to extend - no block", zap.Int64("round", r.GetRoundNumber()), zap.Int64("current_round", mc.GetCurrentRound()))
	return nil
}

/*GenerateRoundBlock - given a round number generates a block*/
func (mc *Chain) GenerateRoundBlock(ctx context.Context, r *Round) (*block.Block, error) {
	ts := time.Now()
	defer func() {
		rbgTimer.UpdateSince(ts)
	}()
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
	b := block.NewBlock(mc.GetKey(), r.GetRoundNumber())
	b.LatestFinalizedMagicBlockHash = mc.GetLatestFinalizedMagicBlock().Hash
	b.MinerID = node.Self.Underlying().GetKey()
	mc.SetPreviousBlock(ctx, r, b, pb)
	start := time.Now()
	makeBlock := false
	generationTimeout := time.Millisecond * time.Duration(mc.GetGenerationTimeout())
	generationTries := 0
	var startLogging time.Time
	for true {
		if mc.GetCurrentRound() > b.Round {
			Logger.Error("generate block - round mismatch", zap.Any("round", roundNumber), zap.Any("current_round", mc.GetCurrentRound()))
			return nil, ErrRoundMismatch
		}
		txnCount := transaction.GetTransactionCount()
		b.SetStateDB(pb)
		generationTries++
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
							Logger.Info("generate block", zap.Any("round", roundNumber), zap.Any("delay", delay), zap.Any("txn_count", txnCount), zap.Any("t.txn_count", transaction.GetTransactionCount()), zap.Any("error", cerr))
						}
						if mc.GetCurrentRound() > b.Round {
							Logger.Error("generate block - round mismatch", zap.Any("round", roundNumber), zap.Any("current_round", mc.GetCurrentRound()))
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
					Logger.Error("generate block", zap.Int64("round", roundNumber), zap.Error(err))
					return nil, err
				}
			}
			if startLogging.IsZero() || time.Now().Sub(startLogging) > time.Second {
				startLogging = time.Now()
				Logger.Info("generate block", zap.Any("round", roundNumber), zap.Any("txn_count", txnCount), zap.Any("t.txn_count", transaction.GetTransactionCount()), zap.Any("error", err))
			}
			return nil, err
		}

		if r.GetRandomSeed() != b.GetRoundRandomSeed() {
			Logger.Error("round random seed mismatch", zap.Int64("round", b.Round), zap.Int64("round_rrs", r.GetRandomSeed()), zap.Int64("blk_rrs", b.GetRoundRandomSeed()))
			return nil, ErrRRSMismatch
		}
		mc.AddRoundBlock(r, b)
		if generationTries > 1 {
			Logger.Error("generate block - multiple tries", zap.Int64("round", b.Round), zap.Int("tries", generationTries))
		}
		break
	}
	if r.IsVerificationComplete() {
		Logger.Error("generate block - verification complete", zap.Any("round", roundNumber), zap.Any("notarized", len(r.GetNotarizedBlocks())))
		return nil, nil
	}
	mc.addToRoundVerification(ctx, r, b)
	r.AddProposedBlock(b)
	go mc.SendBlock(ctx, b)
	return b, nil
}

/*AddToRoundVerification - Add a block to verify  */
func (mc *Chain) AddToRoundVerification(ctx context.Context, mr *Round, b *block.Block) {
	mr.AddProposedBlock(b)
	if mr.IsFinalizing() || mr.IsFinalized() {
		b.SetBlockState(block.StateVerificationRejected)
		Logger.Debug("add to verification", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Bool("finalizing", mr.IsFinalizing()), zap.Bool("finalized", mr.IsFinalized()))
		return
	}
	if !mc.ValidateMagicBlock(ctx, b) {
		b.SetBlockState(block.StateVerificationRejected)
		Logger.Error("add to verification (invalid magic block)", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("magic_block", b.LatestFinalizedMagicBlockHash))
		return
	}
	bNode := mc.GetMiners(mr).GetNode(b.MinerID)
	if bNode == nil {
		b.SetBlockState(block.StateVerificationRejected)
		Logger.Error("add to round verification (invalid miner)", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("miner_id", b.MinerID))
		return
	}
	pr := mc.GetMinerRound(mr.Number - 1)
	if pr == nil {
		Logger.Error("add to verification (prior block's verify round is nil)", zap.Int64("round", mr.Number-1),
			zap.String("prev_block", b.PrevHash), zap.Int("pb_v_tickets", b.PrevBlockVerificationTicketsSize()))
		return
	}
	if b.Round > 1 {
		if err := mc.VerifyNotarization(ctx, b.PrevHash, b.GetPrevBlockVerificationTickets(), pr); err != nil {
			Logger.Error("add to verification (prior block verify notarization)", zap.Int64("round", pr.Number),
				zap.Any("miner_id", b.MinerID), zap.String("block", b.PrevHash), zap.Int("v_tickets", b.PrevBlockVerificationTicketsSize()), zap.Error(err))
			return
		}
	}
	if mc.AddRoundBlock(mr, b) != b {
		return
	}
	if b.PrevBlock != nil {
		if b.CreationDate < b.PrevBlock.CreationDate {
			Logger.Error("add to verification (creation_date out of sequence", zap.Int64("round", mr.Number), zap.String("block", b.Hash), zap.Any("creation_date", b.CreationDate), zap.String("prev_block", b.PrevHash), zap.Any("prev_creation_date", b.PrevBlock.CreationDate))
			return
		}
		b.ComputeChainWeight()
		mc.updatePriorBlock(ctx, mr.Round, b)
	} else {
		// We can establish an upper bound for chain weight at the current round, subtract 1 and add block's own weight and check if that's less than the chain weight sent
		lfb := mc.GetLatestFinalizedBlock()
		chainWeightUpperBound := lfb.ChainWeight + float64(b.Round-lfb.Round)
		if b.ChainWeight > chainWeightUpperBound-1+b.Weight() {
			Logger.Error("add to verification (wrong chain weight)", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Float64("chain_weight", b.ChainWeight))
			return
		}
	}
	mc.addToRoundVerification(ctx, mr, b)
}

func (mc *Chain) addToRoundVerification(ctx context.Context, mr *Round, b *block.Block) {
	Logger.Info("adding block to verify", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("prev_block", b.PrevHash), zap.String("state_hash", util.ToHex(b.ClientStateHash)), zap.Float64("weight", b.Weight()), zap.Float64("chain_weight", b.ChainWeight))
	vctx := mr.StartVerificationBlockCollection(ctx)
	if vctx != nil {
		miner := mc.GetMiners(mr).GetNode(b.MinerID)
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
	medianTime := mc.Miners.GetMedianNetworkTime()
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
		miner := mc.GetMiners(r).GetNode(b.MinerID)
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
		minerStats.VerificationTicketsByRank[b.RoundRank]++
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
	if !mc.CanShardBlocks() {
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
	notarized := b.IsBlockNotarized()
	//NOTE: We keep collecting verification tickets even if a block is notarized.
	// Knowing who all know about a block can be used to optimize other parts of the protocol
	if !mc.AddVerificationTicket(ctx, b, vt) {
		return
	}
	if notarized {
		Logger.Info("Block is notarized", zap.Int64("round", r.Number), zap.String("block", b.Hash))

		return
	}
	mc.checkBlockNotarization(ctx, r, b)
}

func (mc *Chain) checkBlockNotarization(ctx context.Context, r *Round, b *block.Block) bool {
	if !b.IsBlockNotarized() {
		Logger.Info("checkBlockNotarization --block is not Notarized. Returning", zap.Int64("round", b.Round))
		return false
	}
	if !mc.AddNotarizedBlock(ctx, r, b) {
		return true
	}
	mc.SetRandomSeed(r, b.GetRoundRandomSeed())
	go mc.SendNotarization(ctx, b)
	Logger.Debug("check block notarization - block notarized", zap.Int64("round", b.Round), zap.String("block", b.Hash))
	mc.StartNextRound(common.GetRootContext(), r)
	return true
}

//MergeNotarization - merge a notarization
func (mc *Chain) MergeNotarization(ctx context.Context, r *Round, b *block.Block, vts []*block.VerificationTicket) {
	for _, t := range vts {
		if err := mc.VerifyTicket(ctx, b.Hash, t, r); err != nil {
			Logger.Error("merge notarization", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Error(err))
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
		Logger.Info("add notarized block - computing state", zap.Int64("round", b.Round), zap.String("block", b.Hash))
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

/*BroadcastNotarizedBlocks - send the heaviest notarized block to all the miners */
func (mc *Chain) BroadcastNotarizedBlocks(ctx context.Context, pr *Round) {
	nb := pr.GetHeaviestNotarizedBlock()
	if nb != nil {
		Logger.Info("sending notarized block", zap.Int64("round", pr.Number), zap.String("block", nb.Hash))
		go mc.SendNotarizedBlockToMiners(ctx, nb)
	}
}

/*GetLatestFinalizedBlockFromSharder - request for latest finalized block from all the sharders */
func (mc *Chain) GetLatestFinalizedBlockFromSharder(ctx context.Context) []*block.Block {
	m2s := mc.Sharders
	finalizedBlocks := make([]*block.Block, 0, 1)
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
			Logger.DPanic("lfb from sharder - invalid", zap.Int64("round", fb.Round), zap.String("block", fb.Hash))
		}
		r := mc.GetRound(fb.Round)
		if r == nil {
			r = mc.getRound(ctx, fb.Round)
		}
		err = mc.VerifyNotarization(ctx, fb.Hash, fb.GetVerificationTickets(), r)
		if err != nil {
			Logger.DPanic("lfb from sharder - notarization failed", zap.Int64("round", fb.Round),
				zap.String("block", fb.Hash), zap.Error(err))
			return nil, err
		}
		fbMutex.Lock()
		defer fbMutex.Unlock()
		for _, b := range finalizedBlocks {
			if b.Hash == fb.Hash {
				return fb, nil
			}
		}
		finalizedBlocks = append(finalizedBlocks, fb)
		return fb, nil
	}
	m2s.RequestEntityFromAll(ctx, MinerLatestFinalizedBlockRequestor, nil, handler)
	return finalizedBlocks
}

// GetNextRoundTimeoutTime returns time in milliseconds
func (mc *Chain) GetNextRoundTimeoutTime(ctx context.Context) int {

	ssft := int(math.Ceil(chain.SteadyStateFinalizationTimer.Mean() / 1000000))
	tick := mc.RoundTimeoutSofttoMin
	if tick < mc.RoundTimeoutSofttoMult*ssft {
		tick = mc.RoundTimeoutSofttoMult * ssft
	}
	Logger.Info("nextTimeout", zap.Int("tick", tick))
	return tick
}

// HandleRoundTimeout handle timeouts appropriately
func (mc *Chain) HandleRoundTimeout(ctx context.Context) {
	r := mc.GetMinerRound(mc.GetCurrentRound())
	if r.Number == 0 {
		return
	}
	if r.GetSoftTimeoutCount() == mc.RoundRestartMult {
		Logger.Info("triggering restartRound", zap.Int64("round", r.GetRoundNumber()))
		mc.restartRound(ctx)
	} else {
		Logger.Info("triggering handleNoProgress", zap.Int64("round", r.GetRoundNumber()))
		mc.handleNoProgress(ctx)
		r.IncSoftTimeoutCount()
	}
}

func (mc *Chain) handleNoProgress(ctx context.Context) {
	r := mc.GetMinerRound(mc.GetCurrentRound())
	proposals := r.GetProposedBlocks()
	if len(proposals) > 0 { // send the best block to the network
		b := r.Block
		if b != nil {
			if mc.GetRoundTimeoutCount() <= 10 {
				Logger.Error("sending the best block to the network", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Int("rank", b.RoundRank))
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
		Logger.Error("handleNoProgress", zap.Any("round", mc.GetCurrentRound()), zap.Int64("count", crt), zap.Any("num_vrf_share", len(r.GetVRFShares())))
	case crt == 10:
		Logger.Error("handleNoProgress (no further timeout messages will be displayed)", zap.Any("round", mc.GetCurrentRound()), zap.Int64("count", crt), zap.Any("num_vrf_share", len(r.GetVRFShares())))
		//TODO: should have a means to send an email/SMS to someone or something like that
	}

}

func (mc *Chain) restartRound(ctx context.Context) {
	mc.IncrementRoundTimeoutCount()
	r := mc.GetMinerRound(mc.GetCurrentRound())
	switch crt := mc.GetRoundTimeoutCount(); {
	case crt < 10:
		Logger.Error("restartRound - round timeout occured", zap.Any("round", mc.GetCurrentRound()), zap.Int64("count", crt), zap.Any("num_vrf_share", len(r.GetVRFShares())))
	case crt == 10:
		Logger.Error("restartRound - round timeout occured (no further timeout messages will be displayed)", zap.Any("round", mc.GetCurrentRound()), zap.Int64("count", crt), zap.Any("num_vrf_share", len(r.GetVRFShares())))
		//TODO: should have a means to send an email/SMS to someone or something like that
	}
	mc.RoundTimeoutsCount++

	if r.GetRoundNumber() > 1 {
		if r.GetHeaviestNotarizedBlock() != nil {
			mc.BroadcastNotarizedBlocks(ctx, r)
			Logger.Info("StartNextRound after sending notarized block in restartRound.", zap.Int64("current_round", r.GetRoundNumber()))
			nextR := mc.GetRound(r.GetRoundNumber())
			nr := mc.StartNextRound(ctx, r)
			/*
				if the next round object already exists, StartNextRound does not send VRFs.
				So to be sure send it.
			*/
			if r.HasRandomSeed() {
				if nextR != nil {
					Logger.Info("RedoVRFshare after sending notarized block in restartRound.", zap.Int64("round", nr.GetRoundNumber()), zap.Int("round_toc", nr.GetTimeoutCount()))

					nr.Restart()
					//Recalculate VRF shares and send
					nr.IncrementTimeoutCount()
					redo := mc.RedoVrfShare(ctx, nr)
					if !redo {
						Logger.Info("Could not  RedoVrfShare", zap.Int64("round", r.GetRoundNumber()), zap.Int("round_timeout", r.GetTimeoutCount()))
					}

				} else {
					//StartNextRound would have sent the VRFs. No need to do that again
					Logger.Info("after sending notarized block in restartRound NextR was nil. startNextRound would have sent VRF.", zap.Int64("round", nr.GetRoundNumber()), zap.Int("round_toc", nr.GetTimeoutCount()))

				}
				return
			}
			Logger.Error("Has notarized block in restartRound, but no randomseed.", zap.Int64("current_round", r.GetRoundNumber()))
		}
		pr := mc.GetMinerRound(r.GetRoundNumber() - 1)
		if pr != nil {
			mc.BroadcastNotarizedBlocks(ctx, pr)
		}
	}

	r.Restart()
	//Recalculate VRF shares and send
	r.IncrementTimeoutCount()
	redo := mc.RedoVrfShare(ctx, r)

	if !redo {
		Logger.Info("Could not  RedoVrfShare", zap.Int64("round", r.GetRoundNumber()), zap.Int("round_timeout", r.GetTimeoutCount()))
		return
	}
}

func StartProtocol() {
	mc := GetMinerChain()
	if mc.GetCurrentRound() > 0 {
		return
	}
	ctx := common.GetRootContext()

	lfb := getLatestBlockFromSharders(ctx)
	var mr *Round
	if lfb != nil {
		sr := round.NewRound(lfb.Round)
		mr = mc.CreateRound(sr)
		mr = mc.AddRound(mr).(*Round)
		mc.SetRandomSeed(sr, lfb.GetRoundRandomSeed())
		mc.AddBlock(lfb)
		mc.InitBlockState(lfb)
		mc.SetLatestFinalizedBlock(ctx, lfb)
	} else {
		mr = mc.GetMinerRound(0)
	}
	Logger.Info("starting the blockchain ...", zap.Int64("round", mr.GetRoundNumber()))
	mc.StartNextRound(ctx, mr)
	mc.setStarted()
}

func (mc *Chain) WaitForActiveSharders(ctx context.Context) error {
	if mc.CanShardBlocks() {
		return nil
	}

	ticker := time.NewTicker(5 * chain.DELTA)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ts := <-ticker.C:
			if mc.CanShardBlocks() {
				return nil
			}
			Logger.Info("Waiting for Sharders.", zap.Time("ts", ts))
		}
	}
}
