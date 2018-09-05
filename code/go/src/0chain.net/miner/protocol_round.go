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
	if mr.Number < mc.CurrentRound {
		Logger.Debug("start new round (current round higher)", zap.Int64("round", mr.Number), zap.Int64("current_round", mc.CurrentRound))
		return
	}
	if !mc.AddRound(mr) {
		Logger.Debug("start new round (round already exists)", zap.Int64("round", mr.Number))
		return
	}
	pr := mc.GetRound(mr.Number - 1)
	//TODO: If for some reason the server is lagging behind (like network outage) we need to fetch the previous round info
	// before proceeding
	if pr == nil {
		Logger.Debug("start new round (previous round not found)", zap.Int64("round", mr.Number))
		return
	}
	self := node.GetSelfNode(ctx)
	rank := mr.GetMinerRank(self.SetIndex)
	Logger.Info("*** starting round ***", zap.Int64("round", mr.Number), zap.Int("index", self.SetIndex), zap.Int("rank", rank), zap.Int64("lf_round", mc.LatestFinalizedBlock.Round))
	if !mc.CanGenerateRound(&mr.Round, self.Node) {
		return
	}
	//NOTE: If there are not enough txns, this will not advance further even though rest of the network is. That's why this is a goroutine
	go mc.GenerateRoundBlock(ctx, mr)
}

/*GetBlockToExtend - Get the block to extend from the given round */
func (mc *Chain) GetBlockToExtend(ctx context.Context, r *Round) *block.Block {
	count := 0
	sleepTime := 10 * time.Millisecond
	for true { // Need to do this for timing issues where a start round might come before a notarization and there is no notarized block to extend from
		if r.Number+1 != mc.CurrentRound {
			break
		}
		bnb := r.GetBestNotarizedBlock()
		if bnb == nil {
			bnb = mc.GetNotarizedBlockForRound(&r.Round)
		}
		if bnb != nil {
			if !bnb.IsStateComputed() {
				err := mc.ComputeState(ctx, bnb)
				if err != nil {
					Logger.Error("get block to extend (best nb compute state)", zap.Any("round", r.Number), zap.Any("block", bnb.Hash), zap.Error(err))
				}
			}
			return bnb
		}
		Logger.Error("block to extend - no notarized block yet", zap.Int64("round", r.Number))
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
	Logger.Debug("no block to extend", zap.Int64("round", r.Number), zap.Int64("current_round", mc.CurrentRound))
	return nil
}

/*GenerateRoundBlock - given a round number generates a block*/
func (mc *Chain) GenerateRoundBlock(ctx context.Context, r *Round) (*block.Block, error) {
	txnEntityMetadata := datastore.GetEntityMetadata("txn")
	ctx = memorystore.WithEntityConnection(ctx, txnEntityMetadata)
	defer memorystore.Close(ctx)
	pround := mc.GetRound(r.Number - 1)
	if pround == nil {
		Logger.Error("generate round block (prior round not found)", zap.Any("round", r.Number-1))
		return nil, common.NewError("invalid_round,", "Round not available")
	}
	pb := mc.GetBlockToExtend(ctx, pround)
	if pb == nil {
		Logger.Error("generate round block (prior block not found)", zap.Any("round", r.Number))
		return nil, common.NewError("block_gen_no_block_to_extend", "Do not have the block to extend this round")
	}
	b := datastore.GetEntityMetadata("block").Instance().(*block.Block)
	b.ChainID = mc.ID
	b.MagicBlockHash = mc.CurrentMagicBlock.Hash
	mc.SetPreviousBlock(ctx, &r.Round, b, pb)
	b.SetStateDB(pb)
	for true {
		if mc.CurrentRound > b.Round {
			Logger.Debug("generate block (round mismatch)", zap.Any("round", r.Number), zap.Any("current_round", mc.CurrentRound))
			return nil, ErrRoundMismatch
		}
		txnCount := transaction.TransactionCount
		b.ClientState.ResetChangeCollector(b.PrevBlock.ClientStateHash)
		err := mc.GenerateBlock(ctx, b, mc)
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
						Logger.Debug("generate block", zap.Any("round", r.Number), zap.Any("delay", delay), zap.Any("txn_count", txnCount), zap.Any("t.txn_count", transaction.TransactionCount))
						if mc.CurrentRound > b.Round {
							Logger.Debug("generate block (round mismatch)", zap.Any("round", r.Number), zap.Any("current_round", mc.CurrentRound))
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
		Logger.Info("generate block (round mismatch)", zap.Any("round", r.Number), zap.Any("current_round", mc.CurrentRound))
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
		b.RoundRank = mr.GetMinerRank(bNode.SetIndex)
		b = mc.AddBlock(b)
		if b.PrevBlock != nil {
			b.ComputeChainWeight()
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

		//TODO: Dfinity suggests broadcasting the prior block so it saturates the network
		//While saturation is good, it's going to be expensive, hence TODO for now. Also, if we are proceeding verification based on partial block info,
		// we can't broadcast that block
		if !mc.IsBlockNotarized(ctx, b) {
			mc.SendVerificationTicket(ctx, b, bvt)

			// since block.AddVerificationTicket is not thread-safe, directly doing ProcessVerifiedTicket will not work in rare cases as incoming verification tickets get added concurrently
			//mc.ProcessVerifiedTicket(ctx, r, b, &bvt.VerificationTicket)
			bm := NewBlockMessage(MessageVerificationTicket, node.Self.Node, r, b)
			bm.BlockVerificationTicket = bvt
			mc.BlockMessageChannel <- bm
		} else {
			b.SetBlockState(block.StateNotarized)
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
	if b.PrevBlock == nil {
		pb, err := mc.GetBlock(ctx, b.PrevHash)
		if err != nil {
			Logger.Error("verify round", zap.Any("round", r.Number), zap.Any("block", b.Hash), zap.Any("prev_block", b.PrevHash), zap.Error(err))
			return nil, chain.ErrPreviousBlockUnavailable
		}
		b.SetPreviousBlock(pb)
		if pb.PrevBlockVerficationTickets == nil {
			Logger.Info("verify round (incomplete prev block) TODO: sync block", zap.Int64("round", r.Number), zap.String("block", b.Hash), zap.String("prev_block", b.PrevHash))
		}
	}
	/* Note: We are verifying the notarization of the previous block we have with
	   the prev verification tickets of the current block. This is right as all the
	   necessary verification tickets & notarization message may not have arrived to us */
	if err := mc.VerifyNotarization(ctx, b.PrevBlock, b.PrevBlockVerficationTickets); err != nil {
		Logger.Info("verify round block (prior block verify notarization)", zap.Int64("round", r.Number), zap.String("block", b.Hash), zap.String("prev_block", b.PrevHash), zap.Error(err))
		return nil, err
	}
	notarized := mc.IsBlockNotarized(ctx, b.PrevBlock)
	b.PrevBlock.MergeVerificationTickets(b.PrevBlockVerficationTickets) // grab any unknown verification tickets of previous block from the current block
	if !notarized {
		pr := mc.GetRound(b.PrevBlock.Round)
		if pr != nil {
			mc.AddNotarizedBlock(ctx, &pr.Round, b.PrevBlock)
		} else {
			Logger.Error("verify round - previous round not present", zap.Int64("round", r.Number), zap.String("block", b.Hash), zap.String("prev_block", b.PrevHash))
		}
	}
	/*
		if !b.PrevBlock.IsStateComputed() {
			pbState := b.PrevBlock.GetBlockState()
			Logger.Info("verify round block - previous block state not ready", zap.Int64("round", r.Number), zap.String("block", b.Hash), zap.String("prev_block", b.PrevHash), zap.Int8("prev_block_state", pbState))
			err := mc.ComputeState(ctx, b.PrevBlock)
			if err != nil {
				Logger.Error("verify round block - previous block state error", zap.Int64("round", r.Number), zap.String("block", b.Hash), zap.String("prev_block", b.PrevHash), zap.Int8("prev_block_state", pbState), zap.Error(err))
			}
		}*/
	bvt, err := mc.VerifyBlock(ctx, b)
	if err != nil {
		return nil, err
	}
	return bvt, nil
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
		mc.SendNotarization(ctx, b)
		mc.AddNotarizedBlock(ctx, &r.Round, b)
	}
}

/*AddNotarizedBlock - add a notarized block for a given round */
func (mc *Chain) AddNotarizedBlock(ctx context.Context, r *round.Round, b *block.Block) bool {
	if !r.AddNotarizedBlock(b) {
		return false
	}
	mc.UpdateNodeState(b)
	mc.startRound(r)

	pr := mc.GetRound(r.Number - 1)
	if pr != nil {
		pr.CancelVerification()
		go mc.FinalizeRound(ctx, &pr.Round, mc)
	}
	return true
}

func (mc *Chain) startRound(r *round.Round) {
	if mc.GetRound(r.Number+1) == nil {
		nr := datastore.GetEntityMetadata("round").Instance().(*round.Round)
		nr.Number = r.Number + 1
		//TODO: We need to do VRF at which time the Sleep should be removed
		time.Sleep(chain.DELTA)
		nr.RandomSeed = rand.New(rand.NewSource(r.RandomSeed)).Int63()
		nmr := mc.CreateRound(nr)
		// Even if the context is cancelled, we want to proceed with the next round, hence start with a root context
		Logger.Debug("starting a new round", zap.Int64("round", nr.Number))
		go mc.startNewRound(common.GetRootContext(), nmr)
		//TODO: Not required once VRF is in place
		mc.Miners.SendAll(RoundStartSender(nr))
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
		mc.SendNotarizedBlockToMiners(ctx, nb)
	}
}
