package minersc

import (
	"fmt"
	"math/rand"

	"0chain.net/chaincore/currency"
	"0chain.net/smartcontract/partitions"

	"0chain.net/smartcontract/stakepool"
	"0chain.net/smartcontract/stakepool/spenum"

	"0chain.net/chaincore/block"
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"

	. "github.com/0chain/common/core/logging"
	"github.com/rcrowley/go-metrics"
	"go.uber.org/zap"
)

func activatePending(mn *MinerNode) (bool, error) {
	var change bool
	for _, pool := range mn.Pools {
		if pool.Status == spenum.Pending {
			pool.Status = spenum.Active
			change = true

			newTotalStaked, err := currency.AddCoin(mn.TotalStaked, pool.Balance)
			if err != nil {
				Logger.Error("Staked_Amount_Overflow", zap.Error(err))
				return false, err
			}
			mn.TotalStaked = newTotalStaked
		}
	}
	//TODO: emit delegate pool status update events
	return change, nil
}

// LRU cache in action.
func deletePoolFromUserNode(
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
func unlockDeleted(mn *MinerNode) {
	for _, pool := range mn.Pools {
		if pool.Status == spenum.Deleting {
			pool.Status = spenum.Deleted
		}
	}
}

// unlock all delegate pools of offline node
func unlockOffline(
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
			err = deletePoolFromUserNode(pool.DelegateID, mn.ID, spenum.Miner, balances)
		case NodeTypeSharder:
			err = deletePoolFromUserNode(pool.DelegateID, mn.ID, spenum.Sharder, balances)
		default:
			err = fmt.Errorf("unrecognised node type: %s", mn.NodeType.String())
		}
		if err != nil {
			return common.NewError("pay_fees/unlock_offline", err.Error())
		}

		pool.Status = spenum.Deleted
	}

	return nil
}

func (msc *MinerSmartContract) viewChangePoolsWork(balances cstate.StateContextI,
	mb *block.MagicBlock, minersPart, shardersPart *partitions.Partitions) error {
	var (
		mbMiners   = make(map[string]struct{}, mb.Miners.Size())
		mbSharders = make(map[string]struct{}, mb.Sharders.Size())
	)

	for _, k := range mb.Miners.Keys() {
		mbMiners[k] = struct{}{}
	}

	for _, k := range mb.Sharders.Keys() {
		mbSharders[k] = struct{}{}
	}

	if err := viewChangeWork(balances, minersPart, mbMiners); err != nil {
		return err
	}

	return viewChangeWork(balances, shardersPart, mbSharders)
}

func viewChangeWork(balances cstate.StateContextI, part *partitions.Partitions, mbNodes map[string]struct{}) error {
	deleteNodeKeys := make(map[int][]string)
	if err := forEachNodesWithPart(balances, part, func(partIndex int, mn *MinerNode, cc *changesCount) (bool, error) {
		unlockDeleted(mn)
		if mn.Delete {
			deleteNodeKeys[partIndex] = append(deleteNodeKeys[partIndex], mn.GetKey())
			return false, nil
		}

		change, err := activatePending(mn)
		if err != nil {
			return false, err
		}

		if change {
			cc.increase()
		}

		if _, ok := mbNodes[mn.ID]; !ok {
			if err = unlockOffline(mn, balances); err != nil {
				return false, err
			}
			cc.increase()
		}

		return false, nil
	}); err != nil {
		return err
	}

	// remove deleted miners
	for idx, keys := range deleteNodeKeys {
		if err := part.RemoveItems(balances, idx, keys); err != nil {
			return err
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

	b := balances.GetBlock()
	if b.Round == gn.ViewChange {
		if err := msc.SetMagicBlock(gn, balances); err != nil {
			return "", common.NewErrorf("pay_fee",
				"can't set magic b round=%d viewChange=%d, %v",
				b.Round, gn.ViewChange, err)
		}
	}

	if t.ClientID != b.MinerID {
		return "", common.NewError("pay_fee", "not block generator")
	}

	if b.Round <= gn.LastRound {
		return "", common.NewError("pay_fee", "jumped back in time?")
	}

	fees, err := msc.sumFee(b, true)
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

	// the b generator
	mPart, err := minersPartitions.getPart(balances)
	if err != nil {
		return "", err
	}

	// pay random N miners
	var (
		r  = rand.New(rand.NewSource(b.GetRoundRandomSeed()))
		mn = NewMinerNode()
	)

	mn.ID = b.MinerID
	if err := mPart.Update(balances, mn.GetKey(), func(data []byte) ([]byte, error) {
		_, err := mn.UnmarshalMsg(data)
		if err != nil {
			return nil, err
		}

		if err := mn.StakePool.DistributeRewardsRandN(moveValue, mn.ID, spenum.Miner, r, 10, balances); err != nil {
			return nil, err
		}

		return mn.MarshalMsg(nil)
	}); err != nil {
		return "", common.NewErrorf("pay_fee", "distribute miner reward failed: %v", err)
	}

	Logger.Debug("Pay fees, distribute miner reward successfully",
		zap.String("miner id", b.MinerID),
		zap.Int64("round", b.Round),
		zap.String("block", b.Hash))

	sPart, err := shardersPartitions.getPart(balances)
	if err != nil {
		return "", common.NewErrorf("pay_fee", "could not get sharders: %v", err)
	}

	if gn.RewardRoundFrequency != 0 && b.Round%gn.RewardRoundFrequency == 0 {
		var lfmb = balances.GetLastestFinalizedMagicBlock().MagicBlock
		if lfmb != nil {
			err = msc.viewChangePoolsWork(balances, lfmb, mPart, sPart)
			if err != nil {
				return "", common.NewErrorf("pay_fee", "view change pools work failed: %v", err)
			}
		} else {
			return "", common.NewError("pay fees", "cannot find latest magic bock")
		}
	}

	if err := msc.payShardersAndDelegates(balances, sPart, r, 5, sharderFees, sharderRewards); err != nil {
		return "", common.NewErrorf("pay_fee", "could not reward sharders and delegate pools: %v", err)
	}

	gn.setLastRound(b.Round)
	if err = gn.save(balances); err != nil {
		return "", common.NewErrorf("pay_fees",
			"saving global node: %v", err)
	}

	if err := mPart.Save(balances); err != nil {
		return "", common.NewErrorf("pay_fees", "saving miners changes failed", zap.Error(err))
	}

	if err := sPart.Save(balances); err != nil {
		return "", common.NewErrorf("pay_fees", "saving sharders changes failed", zap.Error(err))
	}

	return resp, nil
}

// pay fees and mint sharders
func (msc *MinerSmartContract) payShardersAndDelegates(balances cstate.StateContextI,
	part *partitions.Partitions, r *rand.Rand, randN int, fee, mint currency.Coin) error {
	sSize, err := part.Size(balances)
	if err != nil {
		return fmt.Errorf("failed to get sharders size: %v", err)
	}

	if sSize <= 0 {
		//return errors.New("no sharders to pay")
		return nil
	}

	if sSize < randN {
		randN = sSize
	}

	rewardSharder, err := distributeShardersFeeAndRewards(balances, fee, randN, mint)
	if err != nil {
		return err
	}

	var (
		mbShardersKeys = getMagicBlockSharders(balances)
		rewardNum      int
	)
	if err := part.UpdateRandomItems(balances, r, randN, func(key string, data []byte) ([]byte, error) {
		_, ok := mbShardersKeys[key]
		if !ok {
			// not in magic block
			return data, nil
		}

		var sh MinerNode
		_, err := sh.UnmarshalMsg(data)
		if err != nil {
			return nil, err
		}

		if err := rewardSharder(&sh, r); err != nil {
			return nil, err
		}
		rewardNum++

		return sh.MarshalMsg(nil)
	}); err != nil {
		return err
	}

	Logger.Debug("pay_fees - pay sharders and delegate pools", zap.Int("reward num", rewardNum))

	return nil
}

func distributeShardersFeeAndRewards(balances cstate.StateContextI, fee currency.Coin, shardersNum int, mint currency.Coin) (
	func(sh *MinerNode, r *rand.Rand) error, error) {
	// fess and mint
	feeShare, feeLeft, err := currency.DistributeCoin(fee, int64(shardersNum))
	if err != nil {
		return nil, err
	}

	mintShare, mintLeft, err := currency.DistributeCoin(mint, int64(shardersNum))
	if err != nil {
		return nil, err
	}

	sharderShare, err := currency.AddCoin(feeShare, mintShare)
	if err != nil {
		return nil, err
	}

	totalCoinLeft, err := currency.AddCoin(feeLeft, mintLeft)
	if err != nil {
		return nil, err
	}

	if totalCoinLeft > currency.Coin(shardersNum) {
		clShare, cl, err := currency.DistributeCoin(totalCoinLeft, int64(shardersNum))
		if err != nil {
			return nil, err
		}
		sharderShare, err = currency.AddCoin(sharderShare, clShare)
		if err != nil {
			return nil, err
		}

		totalCoinLeft = cl
	}

	return func(sh *MinerNode, r *rand.Rand) error {
		var extraShare currency.Coin
		if totalCoinLeft > 0 {
			extraShare = 1
			totalCoinLeft--
		}

		moveValue, err := currency.AddCoin(sharderShare, extraShare)
		if err != nil {
			return err
		}
		if err = sh.StakePool.DistributeRewardsRandN(
			moveValue, sh.ID, spenum.Sharder, r, 1, balances,
		); err != nil {
			return common.NewErrorf("pay_fees/pay_sharders",
				"distributing rewards: %v", err)
		}

		return nil
	}, nil
}

// getMagicBlockSharders - list the sharders in magic block
func getMagicBlockSharders(balances cstate.StateContextI) map[string]struct{} {
	pool := balances.GetMagicBlock(balances.GetBlock().Round).Sharders
	if pool == nil {
		return nil
	}

	nodes := pool.CopyNodes()

	sharderKeys := make(map[string]struct{}, len(nodes))
	for _, sharder := range nodes {
		sharderKeys[GetSharderKey(sharder.GetKey())] = struct{}{}
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
