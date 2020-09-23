package minersc

import (
	"errors"
	"fmt"
	"sort"

	"0chain.net/chaincore/block"
	cstate "0chain.net/chaincore/chain/state"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/util"

	. "0chain.net/core/logging"
	"github.com/rcrowley/go-metrics"
	"go.uber.org/zap"
)

var (
	ErrExecutionStatsNotFound = errors.New("SmartContractExecutionStats stat not found")
)

func (msc *MinerSmartContract) activatePending(mn *MinerNode) {
	for id, pool := range mn.Pending {
		pool.Status = ACTIVE
		mn.Active[id] = pool
		mn.TotalStaked += int64(pool.Balance)
		delete(mn.Pending, id)
	}
}

// pay interests for active pools
func (msc *MinerSmartContract) payInterests(mn *MinerNode, gn *GlobalNode,
	balances cstate.StateContextI) (err error) {

	if !gn.canMint() {
		return // no mints anymore
	}

	// all active
	for _, pool := range mn.Active {
		var amount = state.Balance(float64(pool.Balance) * gn.InterestRate)
		if amount == 0 {
			continue
		}
		var mint = state.NewMint(ADDRESS, pool.DelegateID, amount)
		if err = balances.AddMint(mint); err != nil {
			return common.NewErrorf("pay_fees/pay_interests",
				"error adding mint for stake %v-%v: %v", mn.ID, pool.ID, err)
		}
		msc.addMint(gn, mint.Amount) //
		pool.AddInterests(amount)    // stat
	}

	return
}

// LRU cache in action.
func (msc *MinerSmartContract) deletePoolFromUserNode(delegateID, nodeID,
	poolID string, balances cstate.StateContextI) (err error) {

	var un *UserNode
	if un, err = msc.getUserNode(delegateID, balances); err != nil {
		return fmt.Errorf("getting user node: %v", err)
	}

	var pools, ok = un.Pools[nodeID]
	if !ok {
		return // not found (invalid state?)
	}

	var i int
	for _, id := range pools {
		if id == poolID {
			continue
		}
		pools[i], i = id, i+1
	}
	pools = pools[:i]

	if len(pools) == 0 {
		delete(un.Pools, nodeID) // delete empty
	} else {
		un.Pools[nodeID] = pools // update
	}

	if err = un.save(balances); err != nil {
		return fmt.Errorf("saving user node: %v", err)
	}

	return
}

func (msc *MinerSmartContract) emptyPool(mn *MinerNode,
	pool *sci.DelegatePool, round int64, balances cstate.StateContextI) (
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

// unlock deleted pools
func (msc *MinerSmartContract) unlockDeleted(mn *MinerNode, round int64,
	balances cstate.StateContextI) (err error) {

	for id := range mn.Deleting {
		var pool = mn.Active[id]
		if _, err = msc.emptyPool(mn, pool, round, balances); err != nil {
			return common.NewError("pay_fees/unlock_deleted", err.Error())
		}
		delete(mn.Active, id)
		delete(mn.Deleting, id)
	}

	return
}

// unlock all delegate pools of offline node
func (msc *MinerSmartContract) unlockOffline(mn *MinerNode,
	balances cstate.StateContextI) (err error) {

	mn.Deleting = make(map[string]*sci.DelegatePool) // reset

	// unlock all pending
	for id, pool := range mn.Pending {
		if _, err = msc.emptyPool(mn, pool, 0, balances); err != nil {
			return common.NewError("pay_fees/unlock_offline", err.Error())
		}
		delete(mn.Pending, id)
	}

	// unlock all active
	for id, pool := range mn.Active {
		if _, err = msc.emptyPool(mn, pool, 0, balances); err != nil {
			return common.NewError("pay_fees/unlock_offline", err.Error())
		}
		delete(mn.Active, id)
	}

	if err = mn.save(balances); err != nil {
		return
	}

	return
}

func (msc *MinerSmartContract) viewChangePoolsWork(gn *GlobalNode,
	mb *block.MagicBlock, round int64, balances cstate.StateContextI) (
	err error) {

	var miners, sharders *MinerNodes
	if miners, err = msc.getMinersList(balances); err != nil {
		return fmt.Errorf("getting all miners list: %v", err)
	}
	sharders, err = msc.getShardersList(balances, AllShardersKey)
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
	for _, mn := range miners.Nodes {
		if mn, err = msc.getMinerNode(mn.ID, balances); err != nil {
			return fmt.Errorf("missing miner node: %v", err)
		}
		if err = msc.payInterests(mn, gn, balances); err != nil {
			return
		}
		if err = msc.unlockDeleted(mn, round, balances); err != nil {
			return
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
	for _, mn := range sharders.Nodes {
		if mn, err = msc.getSharderNode(mn.ID, balances); err != nil {
			return fmt.Errorf("missing sharder node: %v", err)
		}
		if err = msc.payInterests(mn, gn, balances); err != nil {
			return
		}
		if err = msc.unlockDeleted(mn, round, balances); err != nil {
			return
		}
		msc.activatePending(mn)
		if _, ok := mbSharders[mn.ID]; !ok {
			shardersOffline = append(shardersOffline, mn)
			continue
		}
		// save excluding offline nodes
		if err = mn.save(balances); err != nil {
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

	return
}

func (msc *MinerSmartContract) adjustViewChange(gn *GlobalNode,
	balances cstate.StateContextI) (err error) {

	var b = balances.GetBlock()

	if b.Round != gn.ViewChange {
		return // don't do anything, not a view change
	}

	var dmn *DKGMinerNodes
	if dmn, err = msc.getMinersDKGList(balances); err != nil {
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

	err = dmn.recalculateTKN(true, gn, balances)
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
	_, err = balances.InsertTrieNode(DKGMinersKey, dmn)
	if err != nil {
		return common.NewErrorf("adjust_view_change",
			"can't cleanup DKG miners: %v", err)
	}

	return
}

func (msc *MinerSmartContract) payFees(t *transaction.Transaction,
	inputData []byte, gn *GlobalNode, balances cstate.StateContextI) (
	resp string, err error) {

	var pn *PhaseNode
	if pn, err = msc.getPhaseNode(balances); err != nil {
		return
	}
	if err = msc.setPhaseNode(balances, pn, gn, t); err != nil {
		return "", common.NewErrorf("pay_fees",
			"error inserting phase node: %v", err)
	}

	if err = msc.adjustViewChange(gn, balances); err != nil {
		return // adjusting view change error
	}

	var block = balances.GetBlock()
	if block.Round == gn.ViewChange && !msc.SetMagicBlock(gn, balances) {
		return "", common.NewErrorf("pay_fee",
			"can't set magic block round=%d viewChange=%d",
			block.Round, gn.ViewChange)
	}

	if t.ClientID != block.MinerID {
		return "", common.NewError("pay_fee", "not block generator")
	}

	if block.Round <= gn.LastRound {
		return "", common.NewError("pay_fee", "jumped back in time?")
	}

	// the block generator
	var mn *MinerNode
	if mn, err = msc.getMinerNode(block.MinerID, balances); err != nil {
		return "", common.NewErrorf("pay_fee", "can't get generator '%s': %v",
			block.MinerID, err)
	}

	var (
		// block reward -- mint for the block
		blockReward = state.Balance(
			float64(gn.BlockReward) * gn.RewardRate,
		)
		charger, restr   = mn.splitByServiceCharge(blockReward)
		minerr, sharderr = gn.splitByShareRatio(restr)
		// fees         -- total fees for the block
		fees             = msc.sumFee(block, true)
		chargef, restf   = mn.splitByServiceCharge(fees)
		minerf, sharderf = gn.splitByShareRatio(restf)
		// intermediate response
		iresp string
	)

	// pay for the generator (charge + share ratio)
	iresp, err = msc.mintStakeHolders(minerr+charger, mn, gn, false, balances)
	if err != nil {
		return "", err
	}
	resp += iresp
	// mint for the generator (charge + share ratio)
	iresp, err = msc.payStakeHolders(minerf+chargef, mn, false, balances)
	if err != nil {
		return "", err
	}
	resp += iresp
	// pay and mint rest for block sharders
	iresp, err = msc.paySharders(sharderf, sharderr, block, gn, balances)
	if err != nil {
		return "", err
	}
	resp += iresp

	// save node first, for the VC pools work
	if err = mn.save(balances); err != nil {
		return "", common.NewErrorf("pay_fees",
			"saving generator node: %v", err)
	}

	// view change stuff
	if block.Round == gn.ViewChange {
		var mb = balances.GetBlock().MagicBlock
		err = msc.viewChangePoolsWork(gn, mb, block.Round, balances)
		if err != nil {
			return "", err
		}
	}

	gn.setLastRound(block.Round)
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

func (msc *MinerSmartContract) mintStakeHolders(value state.Balance,
	node *MinerNode, gn *GlobalNode, isSharder bool,
	balances cstate.StateContextI) (resp string, err error) {

	if !gn.canMint() {
		return // can't mint anymore
	}

	if value == 0 {
		return // nothing to mint
	}

	if isSharder {
		node.Stat.SharderRewards += value
	} else {
		node.Stat.GeneratorRewards += value
	}

	var totalStaked = node.TotalStaked

	for _, pool := range node.orderedActivePools() {
		var (
			ratio    = float64(pool.Balance) / float64(totalStaked)
			userMint = state.Balance(float64(value) * ratio)
		)

		Logger.Info("mint delegate",
			zap.Any("pool", pool),
			zap.Any("mint", userMint))

		if userMint == 0 {
			continue // avoid insufficient minting
		}

		var mint = state.NewMint(ADDRESS, pool.DelegateID, userMint)
		if err = balances.AddMint(mint); err != nil {
			resp += fmt.Sprintf("pay_fee/minting - adding mint: %v", err)
			continue
		}
		msc.addMint(gn, mint.Amount)
		pool.AddRewards(userMint)

		resp += string(mint.Encode())
	}

	return resp, nil
}

func (msc *MinerSmartContract) payStakeHolders(value state.Balance,
	node *MinerNode, isSharder bool,
	balances cstate.StateContextI) (resp string, err error) {

	if value == 0 {
		return // nothing to pay
	}

	if isSharder {
		node.Stat.SharderFees += value
	} else {
		node.Stat.GeneratorFees += value
	}

	var totalStaked = node.TotalStaked

	for _, pool := range node.orderedActivePools() {
		var (
			ratio   = float64(pool.Balance) / float64(totalStaked)
			userFee = state.Balance(float64(value) * ratio)
		)

		Logger.Info("pay delegate",
			zap.Any("pool", pool),
			zap.Any("fee", userFee))

		if userFee == 0 {
			continue // avoid insufficient transfer
		}

		var transfer = state.NewTransfer(ADDRESS, pool.DelegateID, userFee)
		if err = balances.AddTransfer(transfer); err != nil {
			return "", fmt.Errorf("adding transfer: %v", err)
		}

		pool.AddRewards(userFee)
		resp += string(transfer.Encode())
	}

	return resp, nil
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
		if err != nil && err != util.ErrValueNotPresent {
			return nil, fmt.Errorf("unexpected error: %v", err)
		}
		sharders, err = append(sharders, sn), nil // even if it's nil, reset err
	}

	return
}

// pay fees and mint sharders
func (msc *MinerSmartContract) paySharders(fee, mint state.Balance,
	block *block.Block, gn *GlobalNode, balances cstate.StateContextI) (
	resp string, err error) {

	var sharders []*MinerNode
	if sharders, err = msc.getBlockSharders(block, balances); err != nil {
		return // unexpected error
	}

	// fess and mint
	var (
		partf = state.Balance(float64(fee) / float64(len(sharders)))
		partm = state.Balance(float64(mint) / float64(len(sharders)))
	)

	// part for every sharder
	for _, sh := range sharders {

		var sresp string
		sresp, err = msc.payStakeHolders(partf, sh, true, balances)
		if err != nil {
			return "", common.NewErrorf("pay_fees/pay_sharders",
				"paying block sharder fees: %v", err)
		}

		resp += sresp

		sresp, err = msc.mintStakeHolders(partm, sh, gn, true, balances)
		if err != nil {
			return "", common.NewErrorf("pay_fees/mint_sharders",
				"minting block sharder reward: %v", err)
		}

		resp += sresp

		if err = sh.save(balances); err != nil {
			return "", common.NewErrorf("pay_fees/pay_sharders",
				"saving sharder node: %v", err)
		}
	}

	return
}
