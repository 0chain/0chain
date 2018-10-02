package miner

import (
	"context"
	"math/rand"
	"time"

	"0chain.net/block"
	"0chain.net/chain"
	"0chain.net/common"
	"0chain.net/config"
	"0chain.net/datastore"
	. "0chain.net/logging"
	"0chain.net/memorystore"
	"0chain.net/node"
	"0chain.net/round"
	"0chain.net/transaction"
	"0chain.net/util"
	"go.uber.org/zap"
)

//SetNetworkRelayTime - set the network relay time
func SetNetworkRelayTime(delta time.Duration) {
	chain.SetNetworkRelayTime(delta)
}

func (mc *Chain) startNewRound(ctx context.Context, mr *Round) {
	if mr.GetRoundNumber() < mc.CurrentRound {
		Logger.Debug("start new round (current round higher)", zap.Int64("round", mr.GetRoundNumber()), zap.Int64("current_round", mc.CurrentRound))
		return
	}
	if mc.AddRound(mr) != mr {
		Logger.Debug("start new round (round already exists)", zap.Int64("round", mr.GetRoundNumber()))
		return
	}
	pr := mc.GetRound(mr.GetRoundNumber() - 1)
	//TODO: If for some reason the server is lagging behind (like network outage) we need to fetch the previous round info
	// before proceeding
	if pr == nil {
		Logger.Debug("start new round (previous round not found)", zap.Int64("round", mr.GetRoundNumber()))
		return
	}
	self := node.GetSelfNode(ctx)
	rank := mr.GetMinerRank(self.Node)
	Logger.Info("*** starting round ***", zap.Int64("round", mr.GetRoundNumber()), zap.Int("index", self.SetIndex), zap.Int("rank", rank), zap.Int64("lf_round", mc.LatestFinalizedBlock.Round))
	if !mc.CanGenerateRound(mr, self.Node) {
		return
	}
	//NOTE: If there are not enough txns, this will not advance further even though rest of the network is. That's why this is a goroutine
	go mc.GenerateRoundBlock(ctx, mr)
}

/*GetBlockToExtend - Get the block to extend from the given round */
func (mc *Chain) GetBlockToExtend(ctx context.Context, r round.RoundI) *block.Block {
	count := 0
	sleepTime := 10 * time.Millisecond
	for true { // Need to do this for timing issues where a start round might come before a notarization and there is no notarized block to extend from
		if r.GetRoundNumber()+1 != mc.CurrentRound {
			break
		}
		bnb := r.GetBestNotarizedBlock()
		if bnb == nil {
			bnb = mc.GetNotarizedBlockForRound(r)
		}
		if bnb != nil {
			if !bnb.IsStateComputed() {
				err := mc.ComputeState(ctx, bnb)
				if err != nil {
					if config.DevConfiguration.State {
						Logger.Error("get block to extend (best nb compute state)", zap.Any("round", r.GetRoundNumber()), zap.Any("block", bnb.Hash), zap.Error(err))
					}
				}
			}
			return bnb
		}
		Logger.Error("block to extend - no notarized block yet", zap.Int64("round", r.GetRoundNumber()))
		count++
		if count == 10 {
			count = 0
			sleepTime *= 2
			if sleepTime > time.Second {
				sleepTime = time.Second
			}
		}
		time.Sleep(sleepTime)
	}
	Logger.Debug("no block to extend", zap.Int64("round", r.GetRoundNumber()), zap.Int64("current_round", mc.CurrentRound))
	return nil
}

/*GenerateRoundBlock - given a round number generates a block*/
func (mc *Chain) GenerateRoundBlock(ctx context.Context, r *Round) (*block.Block, error) {
	txnEntityMetadata := datastore.GetEntityMetadata("txn")
	ctx = memorystore.WithEntityConnection(ctx, txnEntityMetadata)
	defer memorystore.Close(ctx)
	roundNumber := r.GetRoundNumber()
	pround := mc.GetMinerRound(roundNumber - 1)
	if pround == nil {
		Logger.Error("generate round block (prior round not found)", zap.Any("round", roundNumber-1))
		return nil, common.NewError("invalid_round,", "Round not available")
	}
	pb := mc.GetBlockToExtend(ctx, pround)
	if pb == nil {
		Logger.Error("generate round block (prior block not found)", zap.Any("round", roundNumber))
		return nil, common.NewError("block_gen_no_block_to_extend", "Do not have the block to extend this round")
	}
	b := datastore.GetEntityMetadata("block").Instance().(*block.Block)
	b.ChainID = mc.ID
	b.MagicBlockHash = mc.CurrentMagicBlock.Hash
	b.MinerID = node.Self.GetKey()
	mc.SetPreviousBlock(ctx, r, b, pb)
	b.SetStateDB(pb)
	waitTime := time.Now()
	waitOver := false
	for true {
		if time.Now().Sub(waitTime) > time.Millisecond*time.Duration(mc.GenerateTimeout) {
			waitOver = true
		}
		if mc.CurrentRound > b.Round {
			Logger.Debug("generate block (round mismatch)", zap.Any("round", r.Number), zap.Any("current_round", mc.CurrentRound))
			return nil, ErrRoundMismatch
		}
		txnCount := transaction.TransactionCount
		b.ClientState.ResetChangeCollector(b.PrevBlock.ClientStateHash)
		err := mc.GenerateBlock(ctx, b, mc, waitOver)
		if err != nil {
			cerr, ok := err.(*common.Error)
			if ok {
				switch cerr.Code {
				case InsufficientTxns:
					if !config.MainNet() {
						Logger.Error("generate block", zap.Error(err))
					}
					delay := 128 * time.Millisecond
					for true {
						time.Sleep(delay)
						Logger.Debug("generate block", zap.Any("round", roundNumber), zap.Any("delay", delay), zap.Any("txn_count", txnCount), zap.Any("t.txn_count", transaction.TransactionCount))
						if mc.CurrentRound > b.Round {
							Logger.Debug("generate block (round mismatch)", zap.Any("round", roundNumber), zap.Any("current_round", mc.CurrentRound))
							return nil, ErrRoundMismatch
						}
						if txnCount != transaction.TransactionCount {
							break
						}
						delay = 2 * delay
						if delay > time.Second {
							delay = time.Second
						}
					}
					continue
				case RoundMismatch:
					Logger.Info("generate block", zap.Error(err))
					continue
				}
			}
			Logger.Error("generate block", zap.Error(err))
			return nil, err
		}
		mc.AddBlock(b)
		break
	}
	if mc.CurrentRound > b.Round {
		Logger.Info("generate block (round mismatch)", zap.Any("round", roundNumber), zap.Any("current_round", mc.CurrentRound))
		return nil, ErrRoundMismatch
	}
	mc.AddToRoundVerification(ctx, r, b)
	mc.SendBlock(ctx, b)
	return b, nil
}

/*AddToRoundVerification - Add a block to verify : WARNING: does not support concurrent access for a given round */
func (mc *Chain) AddToRoundVerification(ctx context.Context, mr *Round, b *block.Block) {
	if b.MinerID != node.GetSelfNode(ctx).GetKey() {
		if mr.IsFinalizing() || mr.IsFinalized() {
			b.SetBlockState(block.StateVerificationRejected)
			Logger.Debug("add to verification", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Bool("finalizing", mr.IsFinalizing()), zap.Bool("finalized", mr.IsFinalized()))
			return
		}
		if !mc.ValidateMagicBlock(ctx, b) {
			b.SetBlockState(block.StateVerificationRejected)
			Logger.Error("invalid magic block", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("magic_block", b.MagicBlockHash))
			return
		}
		bNode := node.GetNode(b.MinerID)
		if bNode == nil {
			b.SetBlockState(block.StateVerificationRejected)
			Logger.Error("add to round verification (invalid miner)", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("miner_id", b.MinerID))
			return
		}
		if b.Round > 1 {
			if err := mc.VerifyNotarization(ctx, b.PrevHash, b.PrevBlockVerficationTickets); err != nil {
				Logger.Error("verify round block (prior block verify notarization)", zap.Int64("round", mr.Number), zap.String("block", b.Hash), zap.String("prev_block", b.PrevHash), zap.Error(err))
				return
			}
		}
		if mc.AddBlock(b) != b {
			return
		}
		b.RoundRank = mr.GetMinerRank(bNode)
		if b.PrevBlock != nil {
			b.ComputeChainWeight()
			mc.updatePriorBlock(ctx, mr.Round, b)
		} else {
			// We can establish an upper bound for chain weight at the current round, subtract 1 and add block's own weight and check if that's less than the chain weight sent
			chainWeightUpperBound := mc.LatestFinalizedBlock.ChainWeight + float64(b.Round-mc.LatestFinalizedBlock.Round)
			if b.ChainWeight > chainWeightUpperBound-1+b.Weight() {
				Logger.Error("add to verification (wrong chain weight)", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Float64("chain_weight", b.ChainWeight))
				return
			}
		}
	}
	Logger.Info("adding block to verify", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("prev_block", b.PrevHash), zap.String("state_hash", util.ToHex(b.ClientStateHash)), zap.Float64("weight", b.Weight()), zap.Float64("chain_weight", b.ChainWeight))
	vctx := mr.StartVerificationBlockCollection(ctx)
	if vctx != nil {
		miner := mc.Miners.GetNode(b.MinerID)
		minerNT := time.Duration(int64(1000 * miner.LargeMessageSendTime))
		if minerNT >= chain.DELTA {
			mr.delta = time.Millisecond
		} else {
			mr.delta = chain.DELTA - minerNT
		}
		go mc.CollectBlocksForVerification(vctx, mr)
	}
	mr.AddBlockToVerify(b)
}

/*CollectBlocksForVerification - keep collecting the blocks till timeout and then start verifying */
func (mc *Chain) CollectBlocksForVerification(ctx context.Context, r *Round) {
	verifyAndSend := func(ctx context.Context, r *Round, b *block.Block) bool {
		b.SetBlockState(block.StateVerificationAccepted)
		bvt, err := mc.VerifyRoundBlock(ctx, r, b)
		if err != nil {
			b.SetBlockState(block.StateVerificationFailed)
			ierr, ok := err.(*common.Error)
			if ok {
				if ierr.Code == RoundMismatch {
					Logger.Debug("verify round block", zap.Any("round", r.Number), zap.Any("block", b.Hash), zap.Any("current_round", mc.CurrentRound))
				} else {
					Logger.Error("verify round block", zap.Any("round", r.Number), zap.Any("block", b.Hash), zap.Error(err))
				}
			} else {
				Logger.Error("verify round block", zap.Any("round", r.Number), zap.Any("block", b.Hash), zap.Error(err))
			}
			return false
		}
		r.Block = b
		b.SetBlockState(block.StateVerificationSuccessful)
		if mc.IsBlockNotarized(ctx, b) {
			b.SetBlockState(block.StateNotarized)
		}
		bnb := r.GetBestNotarizedBlock()
		if bnb == nil || bnb.RoundRank >= b.RoundRank {
			go mc.SendVerificationTicket(ctx, b, bvt)
			// since block.AddVerificationTicket is not thread-safe, directly doing ProcessVerifiedTicket will not work in rare cases as incoming verification tickets get added concurrently
			bm := NewBlockMessage(MessageVerificationTicket, node.Self.Node, r, b)
			bm.BlockVerificationTicket = bvt
			mc.BlockMessageChannel <- bm
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
	for true {
		select {
		case <-ctx.Done():
			for _, b := range blocks {
				if b.GetBlockState() == block.StateVerificationPending || b.GetBlockState() == block.StateVerificationAccepted {
					Logger.Info("cancel verification (failing block)", zap.Int64("round", r.Number), zap.String("block", b.Hash))
					b.SetBlockState(block.StateVerificationFailed)
				}
			}
			return
		case <-blockTimeTimer.C:
			initiateVerification()
		case b := <-r.GetBlocksToVerifyChannel():
			if sendVerification {
				// Is this better than the current best block
				if r.Block == nil || b.RoundRank < r.Block.RoundRank {
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
	if mc.CurrentRound != r.Number {
		return nil, ErrRoundMismatch
	}
	if b.MinerID == node.Self.GetKey() {
		return mc.SignBlock(ctx, b)
	}
	var hasPriorBlock = b.PrevBlock != nil
	bvt, err := mc.VerifyBlock(ctx, b)
	if err != nil {
		return nil, err
	}
	if !hasPriorBlock && b.PrevBlock != nil {
		mc.updatePriorBlock(ctx, r.Round, b)
	}
	return bvt, nil
}

func (mc *Chain) updatePriorBlock(ctx context.Context, r *round.Round, b *block.Block) {
	pb := b.PrevBlock
	notarized := mc.IsBlockNotarized(ctx, pb)
	pb.MergeVerificationTickets(b.PrevBlockVerficationTickets) // grab any unknown verification tickets of previous block from the current block
	if !notarized {
		pr := mc.GetMinerRound(pb.Round)
		if pr != nil {
			mc.AddNotarizedBlock(ctx, pr.Round, pb)
		} else {
			Logger.Error("verify round - previous round not present", zap.Int64("round", r.Number), zap.String("block", b.Hash), zap.String("prev_block", b.PrevHash))
		}
	}
}

/*ProcessVerifiedTicket - once a verified ticket is received, do further processing with it */
func (mc *Chain) ProcessVerifiedTicket(ctx context.Context, r *Round, b *block.Block, vt *block.VerificationTicket) {
	notarized := mc.IsBlockNotarized(ctx, b)
	//NOTE: We keep collecting verification tickets even if a block is notarized.
	// This is useful since Dfinity suggest broadcasting the previous notarized block when verifying the current block
	// If we know how many verifications already exists for a block, we only need to broadcast to the rest. Hence collecting any prior block verifications is OK.
	if !mc.AddVerificationTicket(ctx, b, vt) {
		return
	}
	if notarized {
		return
	}
	if mc.IsBlockNotarized(ctx, b) {
		b.SetBlockState(block.StateNotarized)
		r.Block = b
		mc.CancelRoundVerification(ctx, r)
		mc.AddNotarizedBlock(ctx, r.Round, b)
		go mc.SendNotarization(ctx, b)
	}
}

/*AddNotarizedBlock - add a notarized block for a given round */
func (mc *Chain) AddNotarizedBlock(ctx context.Context, r round.RoundI, b *block.Block) bool {
	if r.AddNotarizedBlock(b) != b {
		return false
	}
	mc.UpdateNodeState(b)
	pr := mc.GetMinerRound(r.GetRoundNumber() - 1)
	if pr != nil {
		pr.CancelVerification()
		go mc.FinalizeRound(ctx, pr.Round, mc)
	}
	mc.startNextRound(r)
	return true
}

func (mc *Chain) startNextRound(r round.RoundI) {
	nrNumber := r.GetRoundNumber() + 1
	if mc.GetRound(nrNumber) == nil {
		nr := datastore.GetEntityMetadata("round").Instance().(*round.Round)
		nr.Number = nrNumber
		//TODO: We need to do VRF at which time the Sleep should be removed
		time.Sleep(chain.DELTA)
		nr.RandomSeed = rand.New(rand.NewSource(r.GetRandomSeed())).Int63()
		nmr := mc.CreateRound(nr)
		// Even if the context is cancelled, we want to proceed with the next round, hence start with a root context
		Logger.Debug("starting a new round", zap.Int64("round", nr.Number))
		ctx := common.GetRootContext()
		go mc.startNewRound(ctx, nmr)
		//TODO: Not required once VRF is in place
		go mc.SendRoundStart(ctx, nr)
	}
}

/*CancelRoundVerification - cancel verifications happening within a round */
func (mc *Chain) CancelRoundVerification(ctx context.Context, r *Round) {
	r.CancelVerification() // No need for further verification of any blocks
}

/*BroadcastNotarizedBlocks - send all the notarized blocks to all generating miners for a round*/
func (mc *Chain) BroadcastNotarizedBlocks(ctx context.Context, pr *Round, r *Round) {
	nb := pr.GetBestNotarizedBlock()
	if nb != nil {
		Logger.Info("sending notarized block", zap.Int64("round", pr.Number), zap.String("block", nb.Hash))
		go mc.SendNotarizedBlockToMiners(ctx, nb)
	}
}
