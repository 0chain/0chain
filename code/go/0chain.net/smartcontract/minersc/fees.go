package minersc

import (
	"fmt"
	"sort"

	"0chain.net/smartcontract/stakepool"

	"0chain.net/chaincore/block"
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/util"

	. "0chain.net/core/logging"
	"github.com/rcrowley/go-metrics"
	"go.uber.org/zap"
)

func (msc *MinerSmartContract) activatePending(mn *MinerNode) {
	for _, pool := range mn.Pools {
		if pool.Status == stakepool.Pending {
			pool.Status = stakepool.Active
			mn.NumPending--
			mn.NumActive++
			mn.TotalStaked += int64(pool.Balance)
		}
	}
}

// LRU cache in action.
func (msc *MinerSmartContract) deletePoolFromUserNode(
	delegateID, nodeID,
	poolID string,
	providerType stakepool.Provider,
	balances cstate.StateContextI,
) error {

	usp, err := stakepool.GetUserStakePool(providerType, delegateID, balances)
	if err != nil {
		return fmt.Errorf("getting user node: %v", err)
	}
	usp.Del(nodeID, poolID)
	if err := usp.Save(stakepool.Blobber, delegateID, balances); err != nil {
		return fmt.Errorf("saving user node: %v", err)
	}

	return nil
}

/*
func (msc *MinerSmartContract) emptyPool(mn *MinerNode,
	pool *sci.DelegatePool, _ int64, balances cstate.StateContextI) (
	resp string, err error) {

	mn.TotalStaked -= int64(pool.Balance)

	// transfer, empty
	var transfer *state.Transfer
	transfer, resp, err = pool.EmptyPool(ADDRESS, pool.DelegateID, nil)
	if err != nil {
		return "", fmt.Errorf("error emptying delegate pool: %v", err)
	}
	if err = balances.AddTransfer(transfer); err != nil {
		return "", fmt.Errorf("adding transfer: %v", err)
	}

	err = msc.deletePoolFromUserNode(pool.DelegateID, mn.ID, pool.ID, balances)
	return
}
*/
// unlock deleted pools
func (msc *MinerSmartContract) unlockDeleted(mn *MinerNode, round int64,
	balances cstate.StateContextI) (err error) {
	for _, pool := range mn.Pools {
		if pool.Status == stakepool.Deleting {
			pool.Status = stakepool.Deleted
			mn.NumActive--
		}
	}

	return
}

// unlock all delegate pools of offline node
func (msc *MinerSmartContract) unlockOffline(
	mn *MinerNode,
	balances cstate.StateContextI,
) error {
	for id, pool := range mn.Pools {
		transfer := state.NewTransfer(ADDRESS, pool.DelegateID, pool.Balance)
		if err := balances.AddTransfer(transfer); err != nil {
			return fmt.Errorf("pay_fees/unlock_offline: adding transfer: %v", err)
		}
		var err error
		switch mn.NodeType {
		case NodeTypeMiner:
			err = msc.deletePoolFromUserNode(pool.DelegateID, mn.ID, id, stakepool.Miner, balances)
		case NodeTypeSharder:
			err = msc.deletePoolFromUserNode(pool.DelegateID, mn.ID, id, stakepool.Sharder, balances)
		default:
			err = fmt.Errorf("unrecognised node type: %s", mn.NodeType.String())
		}
		if err != nil {
			return common.NewError("pay_fees/unlock_offline", err.Error())
		}

		pool.Status = stakepool.Deleted
	}
	mn.NumPending = 0
	mn.NumActive = 0

	if err := mn.save(balances); err != nil {
		return err
	}

	return nil
}

func (msc *MinerSmartContract) viewChangePoolsWork(gn *GlobalNode,
	mb *block.MagicBlock, round int64, balances cstate.StateContextI) (
	err error) {

	var miners, sharders *MinerNodes
	if miners, err = getMinersList(balances); err != nil {
		return fmt.Errorf("getting all miners list: %v", err)
	}
	sharders, err = getAllShardersList(balances)
	if err != nil {
		return fmt.Errorf("getting all sharders list: %v", err)
	}

	var (
		mbMiners   = make(map[string]struct{}, mb.Miners.Size())
		mbSharders = make(map[string]struct{}, mb.Miners.Size())

		minersOffline, shardersOffline []*MinerNode
	)

	for _, k := range mb.Miners.Keys() {
		mbMiners[k] = struct{}{}
	}

	for _, k := range mb.Sharders.Keys() {
		mbSharders[k] = struct{}{}
	}

	// miners
	minerDelete := false
	for i := len(miners.Nodes) - 1; i >= 0; i-- {
		mn := miners.Nodes[i]

		m, er := getMinerNode(mn.ID, balances)
		switch er {
		case nil:
			mn = m
			// ref back to the miners list, otherwise the changes on the miner would
			// not be saved to the miners list.
			miners.Nodes[i] = mn
		case util.ErrValueNotPresent:
		default:
			return fmt.Errorf("could not get miner node: %v", er)
		}

		if err = msc.unlockDeleted(mn, round, balances); err != nil {
			return
		}
		if mn.Delete {
			miners.Nodes = append(miners.Nodes[:i], miners.Nodes[i+1:]...)
			if _, err := balances.DeleteTrieNode(mn.GetKey()); err != nil {
				return fmt.Errorf("deleting miner node: %v", err)
			}
			minerDelete = true
			continue
		}
		msc.activatePending(mn)
		if _, ok := mbMiners[mn.ID]; !ok {
			minersOffline = append(minersOffline, mn)
			continue
		}
		// save excluding offline nodes
		if err = mn.save(balances); err != nil {
			return
		}
	}

	// sharders
	sharderDelete := false
	for i := len(sharders.Nodes) - 1; i >= 0; i-- {
		sn := sharders.Nodes[i]
		n, er := msc.getSharderNode(sn.ID, balances)
		switch er {
		case nil:
			sn = n
			sharders.Nodes[i] = sn
		case util.ErrValueNotPresent:
		default:
			return fmt.Errorf("could not found sharder node: %v", er)
		}

		if err = msc.unlockDeleted(sn, round, balances); err != nil {
			return
		}
		if sn.Delete {
			sharders.Nodes = append(sharders.Nodes[:i], sharders.Nodes[i+1:]...)
			if err = sn.save(balances); err != nil {
				return
			}
			sharderDelete = true
			continue
		}
		msc.activatePending(sn)
		if _, ok := mbSharders[sn.ID]; !ok {
			shardersOffline = append(shardersOffline, sn)
			continue
		}
		// save excluding offline nodes
		if err = sn.save(balances); err != nil {
			return
		}
	}

	// unlockOffline
	for _, mn := range minersOffline {
		if err = msc.unlockOffline(mn, balances); err != nil {
			return
		}
	}

	for _, mn := range shardersOffline {
		if err = msc.unlockOffline(mn, balances); err != nil {
			return
		}
	}

	if minerDelete {
		if _, err = balances.InsertTrieNode(AllMinersKey, miners); err != nil {
			return common.NewErrorf("view_change_pools_work",
				"failed saving all miners list: %v", err)
		}
	}

	if sharderDelete {
		if _, err = balances.InsertTrieNode(AllShardersKey, sharders); err != nil {
			return common.NewErrorf("view_change_pools_work",
				"failed saving all sharder list: %v", err)
		}
	}
	return
}

func (msc *MinerSmartContract) adjustViewChange(gn *GlobalNode,
	balances cstate.StateContextI) (err error) {

	var b = balances.GetBlock()

	if b.Round != gn.ViewChange {
		return // don't do anything, not a view change
	}

	var dmn *DKGMinerNodes
	if dmn, err = getDKGMinersList(balances); err != nil {
		return common.NewErrorf("adjust_view_change",
			"can't get DKG miners: %v", err)
	}

	var waited int
	for k := range dmn.SimpleNodes {
		if !dmn.Waited[k] {
			delete(dmn.SimpleNodes, k)
			continue
		}
		waited++
	}

	err = dmn.reduceNodes(true, gn, balances)
	if err == nil && waited < dmn.K {
		err = fmt.Errorf("< K miners succeed 'wait' phase: %d < %d",
			waited, dmn.K)
	}
	if err != nil {
		Logger.Info("adjust_view_change", zap.Error(err))
		// don't do this view change, save the gn later
		// reset the ViewChange to previous one (for miners)
		var prev = gn.prevMagicBlock(balances)
		gn.ViewChange = prev.StartingRound
		// reset this error, since it's not fatal, we just don't do
		// this view change, because >= T miners didn't send 'wait' transaction
		err = nil
		// don't return here -> reset DKG miners list first
	}

	// don't clear the nodes don't waited from MB, since MB
	// already saved by miners; if T miners doesn't save their
	// DKG summary and MB data, then we just don't do the
	// view change

	// clear DKG miners list
	dmn = NewDKGMinerNodes()
	if err := updateDKGMinersList(balances, dmn); err != nil {
		return common.NewErrorf("adjust_view_change",
			"can't cleanup DKG miners: %v", err)
	}

	return
}

func (msc *MinerSmartContract) payFees(t *transaction.Transaction,
	_ []byte, gn *GlobalNode, balances cstate.StateContextI) (
	resp string, err error) {

	if config.DevConfiguration.ViewChange {
		// TODO: cache the phase node so if when there's no view change happens, we
		// can avoid unnecessary MPT access
		var pn *PhaseNode
		if pn, err = GetPhaseNode(balances); err != nil {
			return
		}
		if err = msc.setPhaseNode(balances, pn, gn, t); err != nil {
			return "", common.NewErrorf("pay_fees",
				"error inserting phase node: %v", err)
		}

		if err = msc.adjustViewChange(gn, balances); err != nil {
			return // adjusting view change error
		}
	}

	var mb = balances.GetBlock()
	if mb.Round == gn.ViewChange && !msc.SetMagicBlock(gn, balances) {
		return "", common.NewErrorf("pay_fee",
			"can't set magic mb round=%d viewChange=%d",
			mb.Round, gn.ViewChange)
	}

	if t.ClientID != mb.MinerID {
		return "", common.NewError("pay_fee", "not mb generator")
	}

	if mb.Round <= gn.LastRound {
		return "", common.NewError("pay_fee", "jumped back in time?")
	}

	// the mb generator
	var mn *MinerNode
	if mn, err = getMinerNode(mb.MinerID, balances); err != nil {
		return "", common.NewErrorf("pay_fee", "can't get generator '%s': %v",
			mb.MinerID, err)
	}

	Logger.Debug("Pay fees, get miner id successfully",
		zap.String("miner id", mb.MinerID),
		zap.Int64("round", mb.Round),
		zap.String("block", mb.Hash))

	var (
		// mb reward -- mint for the mb
		blockReward = state.Balance(
			float64(gn.BlockReward) * gn.RewardRate,
		)
		minerr, sharderr = gn.splitByShareRatio(blockReward)
		// fees         -- total fees for the mb
		fees             = msc.sumFee(mb, true)
		minerf, sharderf = gn.splitByShareRatio(fees)
		// intermediate response
		iresp string
	)

	//if mn.numActiveDelegates() == 0 {
	//	return "", common.NewErrorf("pay_fees",
	//		"miner %v has no delegates", mn.ID)
	//	} else {
	mn.Stat.GeneratorRewards += minerr + minerf
	if err := mn.StakePool.DistributeRewards(
		float64(minerr+minerf), mn.ID, stakepool.Miner, balances,
	); err != nil {
		return "", err
	}
	//}
	// pay and mint rest for mb sharders
	iresp, err = msc.payShardersAndDelegates(sharderf, sharderr, mb, gn, balances)
	if err != nil {
		return "", err
	}
	resp += iresp

	// save node first, for the VC pools work
	if err = mn.save(balances); err != nil {
		return "", common.NewErrorf("pay_fees",
			"saving generator node: %v", err)
	}

	if err = emitUpdateMiner(mn, balances, false); err != nil {
		return "", common.NewErrorf("pay_fees", "saving generator node to db: %v", err)
	}

	if gn.RewardRoundFrequency != 0 && mb.Round%gn.RewardRoundFrequency == 0 {
		var lfmb = balances.GetLastestFinalizedMagicBlock().MagicBlock
		if lfmb != nil {
			err = msc.viewChangePoolsWork(gn, lfmb, mb.Round, balances)
			if err != nil {
				return "", err
			}
		} else {
			return "", common.NewError("pay fees", "cannot find latest magic bock")
		}
	}

	gn.setLastRound(mb.Round)
	if err = gn.save(balances); err != nil {
		return "", common.NewErrorf("pay_fees",
			"saving global node: %v", err)
	}

	return resp, nil
}

func (msc *MinerSmartContract) sumFee(b *block.Block,
	updateStats bool) state.Balance {

	var totalMaxFee int64
	var feeStats metrics.Counter
	if stat := msc.SmartContractExecutionStats["feesPaid"]; stat != nil {
		feeStats = stat.(metrics.Counter)
	}
	for _, txn := range b.Txns {
		totalMaxFee += txn.Fee
	}

	if updateStats && feeStats != nil {
		feeStats.Inc(totalMaxFee)
	}
	return state.Balance(totalMaxFee)
}

func (msc *MinerSmartContract) payStakeHolders(
	value state.Balance,
	node *MinerNode,
	isSharder bool,
	balances cstate.StateContextI,
) (string, error) {
	if value == 0 {
		return "", nil
	}
	var err error
	if isSharder {
		node.Stat.SharderRewards += value
		err = node.StakePool.DistributeRewards(
			float64(value), node.ID, stakepool.Sharder, balances,
		)
	} else {
		node.Stat.GeneratorRewards += value
		err = node.StakePool.DistributeRewards(
			float64(value), node.ID, stakepool.Miner, balances,
		)
	}
	return "", err
}

func (msc *MinerSmartContract) getBlockSharders(block *block.Block,
	balances cstate.StateContextI) (sharders []*MinerNode, err error) {

	if block.PrevBlock == nil {
		return nil, fmt.Errorf("missing previous block in state context %d, %s",
			block.Round, block.Hash)
	}

	var sids = balances.GetBlockSharders(block.PrevBlock)
	sort.Strings(sids)

	sharders = make([]*MinerNode, 0, len(sids))

	for _, sid := range sids {
		var sn *MinerNode
		sn, err = msc.getSharderNode(sid, balances)
		switch err {
		case nil:
		case util.ErrValueNotPresent:
			sn = NewMinerNode()
			sn.ID = sid
		default:
			return nil, fmt.Errorf("unexpected error: %v", err)
		}
		sharders, err = append(sharders, sn), nil // even if it's nil, reset err
	}

	return
}

// pay fees and mint sharders
func (msc *MinerSmartContract) payShardersAndDelegates(
	fee, mint state.Balance, block *block.Block, gn *GlobalNode, balances cstate.StateContextI,
) (string, error) {
	var err error
	var sharders []*MinerNode
	if sharders, err = msc.getBlockSharders(block, balances); err != nil {
		return "", err
	}

	// fess and mint
	var (
		partf = state.Balance(float64(fee) / float64(len(sharders)))
		partm = state.Balance(float64(mint) / float64(len(sharders)))
	)

	// part for every sharder
	for _, sh := range sharders {
		sh.Stat.SharderRewards += partf + partm
		if err = sh.StakePool.DistributeRewards(
			float64(partf+partm), sh.ID, stakepool.Sharder, balances,
		); err != nil {
			return "", common.NewErrorf("pay_fees/pay_sharders",
				"distributing rewards: %v", err)
		}

		if err = sh.save(balances); err != nil {
			return "", common.NewErrorf("pay_fees/pay_sharders",
				"saving sharder node: %v", err)
		}
	}

	return "", nil
}

func (msc *MinerSmartContract) payNode(reward, fee state.Balance, mn *MinerNode,
	gn *GlobalNode, balances cstate.StateContextI) (
	resp string, err error) {

	if reward != 0 {
		Logger.Info("pay "+mn.NodeType.String()+" service charge",
			zap.Any("delegate_wallet", mn.DelegateWallet),
			zap.Any("service_charge_reward", reward))

		mn.Stat.GeneratorRewards += reward
		var mint = state.NewMint(ADDRESS, mn.DelegateWallet, reward)
		if err = balances.AddMint(mint); err != nil {
			resp += fmt.Sprintf("pay_fee/minting - adding mint: %v", err)
		}
		msc.addMint(gn, mint.Amount)
		resp += string(mint.Encode())
	}
	if fee != 0 {
		Logger.Info("pay "+mn.NodeType.String()+" service charge",
			zap.Any("delegate_wallet", mn.DelegateWallet),
			zap.Any("service_charge_fee", fee))

		mn.Stat.GeneratorFees += fee
		var transfer = state.NewTransfer(ADDRESS, mn.DelegateWallet, fee)
		if err = balances.AddTransfer(transfer); err != nil {
			return "", fmt.Errorf("adding transfer: %v", err)
		}
		resp += string(transfer.Encode())
	}

	return resp, nil
}
