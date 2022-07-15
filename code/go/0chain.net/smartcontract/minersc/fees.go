package minersc

import (
	"fmt"
	"sort"

	"0chain.net/chaincore/currency"

	"0chain.net/smartcontract/stakepool"
	"0chain.net/smartcontract/stakepool/spenum"

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
		if pool.Status == spenum.Pending {
			pool.Status = spenum.Active
			mn.TotalStaked += pool.Balance //810
		}
	}
}

// LRU cache in action.
func (msc *MinerSmartContract) deletePoolFromUserNode(
	delegateID, nodeID,
	poolID string,
	providerType spenum.Provider,
	balances cstate.StateContextI,
) error {

	usp, err := stakepool.GetUserStakePools(providerType, delegateID, balances)
	if err != nil {
		return fmt.Errorf("getting user node: %v", err)
	}
	usp.Del(nodeID, poolID)
	if err := usp.Save(providerType, delegateID, balances); err != nil {
		return fmt.Errorf("saving user node: %v", err)
	}

	return nil
}

// unlock deleted pools
func (msc *MinerSmartContract) unlockDeleted(mn *MinerNode, round int64,
	balances cstate.StateContextI) (err error) {
	for _, pool := range mn.Pools {
		if pool.Status == spenum.Deleting {
			pool.Status = spenum.Deleted
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
			err = msc.deletePoolFromUserNode(pool.DelegateID, mn.ID, id, spenum.Miner, balances)
		case NodeTypeSharder:
			err = msc.deletePoolFromUserNode(pool.DelegateID, mn.ID, id, spenum.Sharder, balances)
		default:
			err = fmt.Errorf("unrecognised node type: %s", mn.NodeType.String())
		}
		if err != nil {
			return common.NewError("pay_fees/unlock_offline", err.Error())
		}

		pool.Status = spenum.Deleted
	}

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

	configuration := config.Configuration()
	isViewChange := configuration.ChainConfig.IsViewChangeEnabled()
	if isViewChange {
		// TODO: cache the phase node so if when there's no view change happens, we
		// can avoid unnecessary MPT access
		var pn *PhaseNode
		if pn, err = GetPhaseNode(balances); err != nil {
			return
		}
		if err = msc.setPhaseNode(balances, pn, gn, t, isViewChange); err != nil {
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

	fees, err := msc.sumFee(mb, true)
	if err != nil {
		return "", err
	}
	blockReward, err := currency.MultFloat64(gn.BlockReward, gn.RewardRate)

	if err != nil {
		return "", err
	}

	minerRewards, sharderRewards, err := gn.splitByShareRatio(blockReward)
	if err != nil {
		return "", fmt.Errorf("error splitting rewards by ratio: %v", err)
	}
	minerFees, sharderFees, err := gn.splitByShareRatio(fees)
	if err != nil {
		return "", fmt.Errorf("error splitting fees by ratio: %v", err)
	}

	moveValue, err := currency.AddCoin(minerRewards, minerFees)
	if err != nil {
		return "", err
	}

	if err := mn.StakePool.DistributeRewards(
		moveValue, mn.ID, spenum.Miner, balances,
	); err != nil {
		return "", err
	}

	// pay and mint rest for mb sharders
	if err := msc.payShardersAndDelegates(sharderFees, sharderRewards, mb, gn, balances); err != nil {
		return "", err
	}

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
	updateStats bool) (currency.Coin, error) {

	var (
		totalMaxFee currency.Coin
		feeStats    metrics.Counter
		err         error
	)
	if stat := msc.SmartContractExecutionStats["feesPaid"]; stat != nil {
		feeStats = stat.(metrics.Counter)
	}
	for _, txn := range b.Txns {
		totalMaxFee, err = currency.AddCoin(totalMaxFee, txn.Fee)
		if err != nil {
			return 0, err
		}
	}

	intTotalMaxFee, err := totalMaxFee.Int64()
	if err != nil {
		return 0, err
	}
	if updateStats && feeStats != nil {
		feeStats.Inc(intTotalMaxFee)
	}
	return totalMaxFee, nil
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
	fee, mint currency.Coin, block *block.Block, gn *GlobalNode, balances cstate.StateContextI,
) error {
	var err error
	var sharders []*MinerNode
	if sharders, err = msc.getBlockSharders(block, balances); err != nil {
		return err
	}

	sn := len(sharders)
	// fess and mint
	feeShare, feeLeft, err := currency.DivideCoin(fee, int64(sn))
	if err != nil {
		return err
	}
	mintShare, mintLeft, err := currency.DivideCoin(mint, int64(sn))
	if err != nil {
		return err
	}

	sharderShare, err := currency.AddCoin(feeShare, mintShare)
	if err != nil {
		return err
	}

	totalCoinLeft, err := currency.AddCoin(feeLeft, mintLeft)
	if err != nil {
		return err
	}

	if totalCoinLeft > currency.Coin(sn) {
		clShare, cl, err := currency.DivideCoin(totalCoinLeft, int64(sn))
		if err != nil {
			return err
		}
		sharderShare, err = currency.AddCoin(sharderShare, clShare)
		if err != nil {
			return err
		}

		totalCoinLeft = cl
	}

	// part for every sharder
	for _, sh := range sharders {
		var extraShare currency.Coin
		if totalCoinLeft > 0 {
			extraShare = 1
			totalCoinLeft--
		}

		moveValue, err := currency.AddCoin(sharderShare, extraShare)
		if err != nil {
			return err
		}
		if err = sh.StakePool.DistributeRewards(
			moveValue, sh.ID, spenum.Sharder, balances,
		); err != nil {
			return common.NewErrorf("pay_fees/pay_sharders",
				"distributing rewards: %v", err)
		}

		if err = sh.save(balances); err != nil {
			return common.NewErrorf("pay_fees/pay_sharders",
				"saving sharder node: %v", err)
		}
	}

	return nil
}
