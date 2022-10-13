package minersc

import (
	"errors"
	"fmt"

	"0chain.net/chaincore/currency"

	"0chain.net/smartcontract/stakepool"
	"0chain.net/smartcontract/stakepool/spenum"

	"0chain.net/chaincore/block"
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"github.com/0chain/common/core/util"

	. "github.com/0chain/common/core/logging"
	"github.com/rcrowley/go-metrics"
	"go.uber.org/zap"
)

func (msc *MinerSmartContract) activatePending(mn *MinerNode) error {
	for _, pool := range mn.Pools {
		if pool.Status == spenum.Pending {
			pool.Status = spenum.Active

			newTotalStaked, err := currency.AddCoin(mn.TotalStaked, pool.Balance)
			if err != nil {
				Logger.Error("Staked_Amount_Overflow", zap.Error(err))
				return err
			}
			mn.TotalStaked = newTotalStaked
		}
	}
	return nil
}

// LRU cache in action.
func (msc *MinerSmartContract) deletePoolFromUserNode(
	delegateID, nodeID string,
	providerType spenum.Provider,
	balances cstate.StateContextI,
) error {

	usp, err := stakepool.GetUserStakePools(providerType, delegateID, balances)
	if err != nil {
		return fmt.Errorf("getting user node: %v", err)
	}
	usp.Del(nodeID)
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
	for _, pool := range mn.Pools {
		transfer := state.NewTransfer(ADDRESS, pool.DelegateID, pool.Balance)
		if err := balances.AddTransfer(transfer); err != nil {
			return fmt.Errorf("pay_fees/unlock_offline: adding transfer: %v", err)
		}
		var err error
		switch mn.NodeType {
		case NodeTypeMiner:
			err = msc.deletePoolFromUserNode(pool.DelegateID, mn.ID, spenum.Miner, balances)
		case NodeTypeSharder:
			err = msc.deletePoolFromUserNode(pool.DelegateID, mn.ID, spenum.Sharder, balances)
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

func (msc *MinerSmartContract) viewChangePoolsWork(
	mb *block.MagicBlock,
	round int64,
	sharders *MinerNodes,
	balances cstate.StateContextI) error {
	miners, err := getMinersList(balances)
	if err != nil {
		return fmt.Errorf("getting all miners list: %v", err)
	}

	var (
		mbMiners                       = make(map[string]struct{}, mb.Miners.Size())
		mbSharders                     = make(map[string]struct{}, mb.Miners.Size())
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
			return err
		}
		if mn.Delete {
			miners.Nodes = append(miners.Nodes[:i], miners.Nodes[i+1:]...)
			if _, err := balances.DeleteTrieNode(mn.GetKey()); err != nil {
				return fmt.Errorf("deleting miner node: %v", err)
			}
			minerDelete = true
			continue
		}
		if err = msc.activatePending(mn); err != nil {
			return err
		}
		if _, ok := mbMiners[mn.ID]; !ok {
			minersOffline = append(minersOffline, mn)
			continue
		}
		// save excluding offline nodes
		if err = mn.save(balances); err != nil {
			return err
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
			return err
		}
		if sn.Delete {
			sharders.Nodes = append(sharders.Nodes[:i], sharders.Nodes[i+1:]...)
			if err = sn.save(balances); err != nil {
				return err
			}
			sharderDelete = true
			continue
		}
		if err = msc.activatePending(sn); err != nil {
			return err
		}
		if _, ok := mbSharders[sn.ID]; !ok {
			shardersOffline = append(shardersOffline, sn)
			continue
		}
		// save excluding offline nodes
		if err = sn.save(balances); err != nil {
			return err
		}
	}

	// unlockOffline
	for _, mn := range minersOffline {
		if err = msc.unlockOffline(mn, balances); err != nil {
			return err
		}
	}

	for _, mn := range shardersOffline {
		if err = msc.unlockOffline(mn, balances); err != nil {
			return err
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
	return nil
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
	if mb.Round == gn.ViewChange {
		if err := msc.SetMagicBlock(gn, balances); err != nil {
			return "", common.NewErrorf("pay_fee",
				"can't set magic mb round=%d viewChange=%d, %v",
				mb.Round, gn.ViewChange, err)
		}
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
	sharders, err := getAllShardersList(balances)
	if err != nil {
		if err != util.ErrValueNotPresent {
			return "", err
		}
	}

	if len(sharders.Nodes) > 0 {
		mbSharders := getRegisterShardersInMagicBlock(balances, sharders)
		if err := msc.payShardersAndDelegates(mbSharders, sharderFees, sharderRewards, balances); err != nil {
			return "", err
		}
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
			err = msc.viewChangePoolsWork(lfmb, mb.Round, sharders, balances)
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

func getRegisterShardersInMagicBlock(balances cstate.StateContextI, sharders *MinerNodes) []*MinerNode {
	var (
		shardersKeys = getMagicBlockSharders(balances)
		smap         = make(map[string]struct{}, len(shardersKeys))
	)

	for _, key := range shardersKeys {
		smap[key] = struct{}{}
	}

	retSharders := make([]*MinerNode, 0, len(shardersKeys))
	for i, s := range sharders.Nodes {
		if _, ok := smap[s.GetKey()]; ok {
			retSharders = append(retSharders, sharders.Nodes[i])
			continue
		}
	}
	return retSharders
}

// getMagicBlockSharders - list the sharders in magic block
func getMagicBlockSharders(balances cstate.StateContextI) []string {
	var (
		pool  = balances.GetMagicBlock(balances.GetBlock().Round).Sharders
		nodes = pool.CopyNodes()
	)

	sharderKeys := make([]string, 0, len(nodes))
	for _, sharder := range nodes {
		sharderKeys = append(sharderKeys, GetSharderKey(sharder.GetKey()))
	}

	return sharderKeys
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

// pay fees and mint sharders
func (msc *MinerSmartContract) payShardersAndDelegates(
	sharders []*MinerNode, fee, mint currency.Coin, balances cstate.StateContextI,
) error {
	sn := len(sharders)
	if sn <= 0 {
		return errors.New("no sharders to pay")
	}

	// fess and mint
	feeShare, feeLeft, err := currency.DistributeCoin(fee, int64(sn))
	if err != nil {
		return err
	}
	mintShare, mintLeft, err := currency.DistributeCoin(mint, int64(sn))
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
		clShare, cl, err := currency.DistributeCoin(totalCoinLeft, int64(sn))
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
