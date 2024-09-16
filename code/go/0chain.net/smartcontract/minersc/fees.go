package minersc

import (
	"encoding/json"
	"fmt"
	"math/rand"

	"github.com/0chain/common/core/currency"
	"github.com/0chain/common/core/logging"

	"0chain.net/smartcontract/stakepool/spenum"

	"0chain.net/chaincore/block"
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"github.com/0chain/common/core/util"

	"github.com/rcrowley/go-metrics"
	"go.uber.org/zap"
)

//nolint:unused
func (msc *MinerSmartContract) activatePending(mn *MinerNode) error {
	orderedPoolIds := mn.OrderedPoolIds()
	for _, id := range orderedPoolIds {
		pool := mn.Pools[id]
		if pool.Status == spenum.Pending {
			pool.Status = spenum.Active

			newTotalStaked, err := currency.AddCoin(mn.TotalStaked, pool.Balance)
			if err != nil {
				logging.Logger.Error("Staked_Amount_Overflow", zap.Error(err))
				return err
			}
			mn.TotalStaked = newTotalStaked
		}
	}
	//TODO: emit delegate pool status update events
	return nil
}

// unlock all delegate pools of offline node
//
//nolint:unused
func (msc *MinerSmartContract) unlockOffline(
	mn *MinerNode,
	balances cstate.StateContextI,
) error {
	orderedPoolIds := mn.OrderedPoolIds()
	for _, id := range orderedPoolIds {
		pool := mn.Pools[id]
		transfer := state.NewTransfer(ADDRESS, pool.DelegateID, pool.Balance)
		if err := balances.AddTransfer(transfer); err != nil {
			return fmt.Errorf("pay_fees/unlock_offline: adding transfer: %v", err)
		}
		pool.Status = spenum.Deleted
	}

	if err := mn.save(balances); err != nil {
		return err
	}

	return nil
}

func (msc *MinerSmartContract) viewChangeDeleteNodes(balances cstate.StateContextI) error {
	if err := deleteNodesOnViewChange(balances, spenum.Miner); err != nil {
		return err
	}

	return deleteNodesOnViewChange(balances, spenum.Sharder)
}

//nolint:unused
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
		//if err = msc.unlockDeleted(mn); err != nil {
		//	return err
		//}
		if mn.Delete {
			miners.Nodes = append(miners.Nodes[:i], miners.Nodes[i+1:]...)
			if _, err := balances.DeleteTrieNode(mn.GetKey()); err != nil {
				return fmt.Errorf("deleting miner node: %v", err)
			}
			minerDelete = true
			continue
		}

		// TODO: remove as there is no pending status anymore
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
		//if err = msc.unlockDeleted(sn); err != nil {
		//	return err
		//}
		if sn.Delete {
			sharders.Nodes = append(sharders.Nodes[:i], sharders.Nodes[i+1:]...)
			if err = sn.save(balances); err != nil {
				return err
			}
			sharderDelete = true
			continue
		}
		// TODO: remove as there is no pending status anymore
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
		if err := updateMinersList(balances, miners); err != nil {
			return err
		}
	}

	if sharderDelete {
		if err = updateAllShardersList(balances, sharders); err != nil {
			return common.NewErrorf("view_change_pools_work",
				"failed saving all sharders list: %v", err)
		}
	}
	return nil
}

func (msc *MinerSmartContract) adjustViewChange(gn *GlobalNode,
	pn *PhaseNode, balances cstate.StateContextI) (err error) {
	var b = balances.GetBlock()
	if b.Round != gn.ViewChange {
		// logging.Logger.Debug("[mvc] adjust view change: not a view change round")
		return // don't do anything, not a view change
	}

	var dmn *DKGMinerNodes
	if dmn, err = getDKGMinersList(balances); err != nil {
		logging.Logger.Error("[mvc] adjust view change: can't get DKG miners", zap.Error(err))
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

	mb, err := getMagicBlock(balances)
	if err != nil {
		logging.Logger.Error("adjust_view_change, failed to get magic block",
			zap.Error(err), zap.Int64("round", balances.GetBlock().Round))
		return common.NewErrorf("adjust_view_change failed to get magic block", "%v", err)
	}

	for _, n := range mb.Miners.Nodes {
		if !dmn.Waited[n.GetKey()] {
			logging.Logger.Error("adjust_view_change, miner not waited",
				zap.String("miner", n.GetKey()))
			// return
			err = common.NewErrorf("adjust_view_change miner not waited", "%v", err)
			break
		}
	}

	// restart DKG if any of the miner in new MB is not waited
	if err != nil {
		var prev = gn.prevMagicBlock(balances)
		gn.ViewChange = prev.StartingRound

		// reset DKG if any of the
		logging.Logger.Warn("adjust_view_change no new magic block, restart DKG", zap.Error(err))
		if err := msc.RestartDKG(pn, balances); err != nil {
			logging.Logger.Error("adjust_view_change restart DKG failed", zap.Error(err))
			return err
		}
		return nil
	}

	// set magic block when all good
	if err := msc.SetMagicBlock(gn, balances); err != nil {
		return common.NewErrorf("pay_fees", "can't set magic b round=%d viewChange=%d, %v",
			b.Round, gn.ViewChange, err)
	}

	// clear DKG miners list
	dmn = NewDKGMinerNodes()
	logging.Logger.Debug("[mvc] adjust_view_change: clear dkg miners list", zap.Int64("round", b.Round))
	if err := updateDKGMinersList(balances, dmn); err != nil {
		return common.NewErrorf("adjust_view_change",
			"can't cleanup DKG miners: %v", err)
	}

	return
}

type PayFeesInput struct {
	Round int64 `json:"round,omitempty"`
}

func (msc *MinerSmartContract) payFees(t *transaction.Transaction,
	input []byte, gn *GlobalNode, balances cstate.StateContextI) (
	resp string, err error) {

	var (
		// configuration = config.Configuration()
		// isViewChange  = configuration.ChainConfig.IsViewChangeEnabled()
		b = balances.GetBlock()
	)

	// if isViewChange || b.Round == gn.ViewChange {
	// if isViewChange || b.Round == gn.ViewChange {
	// TODO: cache the phase node so if when there's no view change happens, we
	// can avoid unnecessary MPT access
	// logging.Logger.Debug("[mvc] payFees: view change, get phase node", zap.Int64("round", b.Round))
	var pn *PhaseNode
	if pn, err = GetPhaseNode(balances); err != nil {
		return
	}

	if err = msc.setPhaseNode(balances, pn, gn, t); err != nil {
		return "", common.NewErrorf("pay_fees", "error setting phase node: %v", err)
	}

	if err = msc.adjustViewChange(gn, pn, balances); err != nil {
		return // adjusting view change error
	}

	if t.ClientID != b.MinerID {
		return "", common.NewError("pay_fees", "not block generator")
	}

	inputRound := PayFeesInput{}
	if err := json.Unmarshal(input, &inputRound); err != nil {
		return "", err
	}

	if inputRound.Round != b.Round {
		return "", common.NewError("pay_fees", fmt.Sprintf("bad round, block %v but input %v", b.Round, inputRound.Round))
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

	var mn *MinerNode
	if mn, err = getRewardedMiner(b, balances); err != nil {
		return "", common.NewErrorf("pay_fees", "cannot get miner to reward, %v", err)
	}
	if mn == nil {
		logging.Logger.Info("pay_fees, could not find miner to reward", zap.Int64("round", b.Round))
	} else {
		logging.Logger.Debug("pay_fees, got miner id successfully",
			zap.String("miner id", mn.ID),
			zap.Int64("round", b.Round),
			zap.String("block", b.Hash))
		if err := mn.StakePool.DistributeRewardsRandN(
			minerRewards,
			mn.ID,
			spenum.Miner,
			b.GetRoundRandomSeed(),
			gn.NumMinerDelegatesRewarded,
			spenum.BlockRewardMiner,
			balances,
		); err != nil {
			return "", err
		}

		if err := mn.StakePool.DistributeRewardsRandN(
			minerFees,
			mn.ID,
			spenum.Miner,
			b.GetRoundRandomSeed(),
			gn.NumMinerDelegatesRewarded,
			spenum.FeeRewardMiner,
			balances,
		); err != nil {
			return "", err
		}
	}

	shardersIDs, err := getLiveSharderIds(balances)
	if err != nil {
		if err != util.ErrValueNotPresent {
			return "", err
		}
	}

	if len(shardersIDs) > 0 {
		seed := b.GetRoundRandomSeed()
		randS := rand.New(rand.NewSource(seed))
		mbShardersIDs := getRegisterShardersInMagicBlock(balances, shardersIDs)

		randS.Shuffle(len(mbShardersIDs), func(i, j int) {
			mbShardersIDs[i], mbShardersIDs[j] = mbShardersIDs[j], mbShardersIDs[i]
		})

		shardersPaid := gn.NumShardersRewarded
		if shardersPaid > len(mbShardersIDs) {
			shardersPaid = len(mbShardersIDs)
		}

		rewardShardersIDs := mbShardersIDs[:shardersPaid]
		rewardSharders, err := cstate.GetItemsByIDs(rewardShardersIDs, getSharderNode, balances)
		if err != nil {
			return "", err
		}

		if err := msc.payShardersAndDelegates(
			gn, rewardSharders, sharderFees, seed,
			spenum.FeeRewardSharder, balances); err != nil {
			return "", err
		}

		if err := msc.payShardersAndDelegates(
			gn, rewardSharders, sharderRewards, seed,
			spenum.BlockRewardSharder, balances); err != nil {
			return "", err
		}

		for _, sh := range rewardSharders {
			if err = sh.save(balances); err != nil {
				return "", common.NewErrorf("pay_fees/pay_sharders",
					"saving sharder node: %v", err)
			}
		}
	} else {
		logging.Logger.Info("pay_fee could not find sharder to reward", zap.Int64("round", b.Round))
	}

	if mn != nil {
		// save node first, for the VC pools work
		if err = mn.save(balances); err != nil {
			return "", common.NewErrorf("pay_fees",
				"saving generator node: %v", err)
		}
	}

	if gn.RewardRoundFrequency != 0 && b.Round%gn.RewardRoundFrequency == 0 {
		var lfmb = balances.GetLastestFinalizedMagicBlock().MagicBlock
		if lfmb != nil {
			// TODO: use viewChangePoolsWork when view change is enabled
			//err = msc.viewChangePoolsWork(lfmb, b.Round, sharders, balances)
			if err = msc.viewChangeDeleteNodes(balances); err != nil {
				return "", err
			}
		} else {
			return "", common.NewError("pay_fees", "cannot find latest magic bock")
		}
	}

	gn.setLastRound(b.Round)
	if err = gn.save(balances); err != nil {
		return "", common.NewErrorf("pay_fees",
			"saving global node: %v", err)
	}

	return resp, nil
}

// getRewardedMiner
// if there is a valid un-killed block miner use that
// otherwise select a random un-killed miner.
func getRewardedMiner(bk *block.Block, balances cstate.CommonStateContextI) (*MinerNode, error) {
	mn, err := getMinerNode(bk.MinerID, balances)
	if err != nil {
		logging.Logger.Error("error getting block miner",
			zap.Int64("round", bk.Round),
			zap.String("block miner id", bk.MinerID),
			zap.Error(err))
	} else {
		if !mn.HasBeenKilled {
			return mn, nil
		}
	}
	nodeList, err := getMinersList(balances)
	if err != nil {
		return nil, err
	}
	miners := filterDeadNodes(nodeList.Nodes)
	if len(miners) == 0 {
		return nil, nil
	}

	randS := rand.New(rand.NewSource(bk.GetRoundRandomSeed()))
	return miners[randS.Intn(len(miners))], nil
}

func filterDeadNodes(nodes []*MinerNode) []*MinerNode {
	var filteredNodes []*MinerNode
	for _, node := range nodes {
		if !node.SimpleNode.HasBeenKilled {
			filteredNodes = append(filteredNodes, node)
		}
	}
	return filteredNodes
}

func getLiveSharderIds(balances cstate.StateContextI) ([]string, error) {
	nodes, err := getAllShardersList(balances)
	if err != nil {
		return nil, err
	}
	var ids []string
	for i := range nodes.Nodes {
		if !nodes.Nodes[i].SimpleNode.HasBeenKilled {
			ids = append(ids, nodes.Nodes[i].ID)
		}
	}
	return ids, nil
}

func getRegisterShardersInMagicBlock(balances cstate.StateContextI, shardersIDs []string) []string {
	var (
		shardersKeysInMB = getMagicBlockSharders(balances)
		smap             = make(map[string]struct{}, len(shardersKeysInMB))
	)

	for _, key := range shardersKeysInMB {
		smap[key] = struct{}{}
	}

	retSharders := make([]string, 0, len(shardersKeysInMB))
	for _, id := range shardersIDs {
		if _, ok := smap[GetSharderKey(id)]; ok {
			retSharders = append(retSharders, id)
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
	gn *GlobalNode,
	rewardSharders []*MinerNode,
	reward currency.Coin,
	seed int64,
	rewardType spenum.Reward,
	balances cstate.StateContextI,
) error {
	n := int64(len(rewardSharders))
	sharderShare, totalCoinLeft, err := currency.DistributeCoin(reward, n)
	if err != nil {
		return err
	}
	if totalCoinLeft > currency.Coin(n) {
		clShare, cl, err := currency.DistributeCoin(totalCoinLeft, n)
		if err != nil {
			return err
		}
		sharderShare, err = currency.AddCoin(sharderShare, clShare)
		if err != nil {
			return err
		}

		totalCoinLeft = cl
	}

	rewardSharder := func(sh *MinerNode) error {
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
			moveValue, sh.ID, spenum.Sharder, seed, gn.NumSharderDelegatesRewarded, rewardType, balances,
		); err != nil {
			return common.NewErrorf("pay_fees/pay_sharders",
				"distributing rewards: %v", err)
		}

		return nil
	}

	for i := range rewardSharders {
		if err := rewardSharder(rewardSharders[i]); err != nil {
			return err
		}
	}

	return nil
}
