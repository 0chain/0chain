package minersc

import (
	"fmt"
	"reflect"
	"runtime"

	"0chain.net/chaincore/block"
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/logging"
	"0chain.net/core/util"

	. "0chain.net/core/logging"
	"go.uber.org/zap"
)

var moveFunctions = make(map[Phase]movePhaseFunctions)

/*
- Start      : moveToContribute
- Contribute : moveToShareOrPublish
- Share      : moveToShareOrPublish
- Publish    : moveToWait
- Wait       : moveToStart
*/

func (msc *MinerSmartContract) moveToContribute(balances cstate.StateContextI,
	pn *PhaseNode, gn *GlobalNode) error {

	var (
		allMinersList *MinerNodes
		dkgMinersList *DKGMinerNodes

		allShardersList *MinerNodes

		err error
	)

	if allMinersList, err = msc.getMinersList(balances); err != nil {
		return common.NewError("move_to_contribute_failed", err.Error())
	}

	if dkgMinersList, err = getDKGMinersList(balances); err != nil {
		return common.NewError("move_to_contribute_failed", err.Error())
	}
	allShardersList, err = getAllShardersList(balances)
	if err != nil {
		return common.NewError("move_to_contribute_failed", err.Error())
	}

	if len(allShardersList.Nodes) < gn.MinS {
		return common.NewErrorf("move_to_contribute_failed",
			"not enough sharders in all sharders list to move phase, all: %d, min_s: %d",
			len(allShardersList.Nodes), gn.MinS)
	}

	if !gn.hasPrevShader(allShardersList, balances) {
		return common.NewErrorf("move_to_contribute_failed",
			"invalid state: all sharders list hasn't a sharder from previous VC set, "+
				"all: %d, min_s: %d", len(allShardersList.Nodes), gn.MinS)
	}

	if !gn.hasPrevMiner(allMinersList, balances) {
		return common.NewErrorf("move_to_contribute_failed",
			"invalid state: all miners list hasn't a miner from previous VC set, "+
				"all: %d, min_n: %d", len(allMinersList.Nodes), gn.MinN)
	}

	if allMinersList == nil {
		return common.NewError("move_to_contribute_failed", "allMinersList is nil")
	}

	if len(allMinersList.Nodes) < dkgMinersList.K {
		return common.NewErrorf("move_to_contribute_failed",
			"len(allMinersList.Nodes) < dkgMinersList.K, l_miners: %d, K: %d",
			len(allMinersList.Nodes), dkgMinersList.K)
	}

	if len(allShardersList.Nodes) < gn.MinS {
		return common.NewErrorf("move_to_contribute_failed",
			"len(allShardersList.Nodes) < gn.MinS, l_shards: %d, min_s: %d",
			len(allShardersList.Nodes), gn.MinS)
	}

	Logger.Debug("miner sc: move phase to contribute",
		zap.Int("miners", len(allMinersList.Nodes)),
		zap.Int("K", dkgMinersList.K),
		zap.Int("sharders", len(allShardersList.Nodes)),
		zap.Int("min_s", gn.MinS))
	return nil
}

// moveToShareOrPublish move function for contribute phase
func (msc *MinerSmartContract) moveToShareOrPublish(
	balances cstate.StateContextI, pn *PhaseNode, gn *GlobalNode) error {

	shardersKeep, err := getShardersKeepList(balances)
	if err != nil {
		return common.NewErrorf("move_to_share_or_publish_failed",
			"failed to get sharders keep list: %v", err)
	}

	if len(shardersKeep.Nodes) < gn.MinS {
		return common.NewErrorf("move_to_share_or_publish_failed",
			"not enough sharders in keep list to move phase keep: %d, min_s: %d", len(shardersKeep.Nodes), gn.MinS)
	}

	if !gn.hasPrevShader(shardersKeep, balances) {
		return common.NewErrorf("move_to_share_or_publish_failed",
			"missing at least one sharder from previous set in "+
				"sharders keep list to move phase, keep: %d, min_s: %d",
			len(shardersKeep.Nodes), gn.MinS)
	}

	dkgMinersList, err := getDKGMinersList(balances)
	if err != nil {
		return common.NewErrorf("move_to_share_or_publish_failed",
			"failed to get miners DKG, phase: %v, err: %v", pn.Phase, err)
	}

	msc.mutexMinerMPK.Lock()
	defer msc.mutexMinerMPK.Unlock()

	mpks, err := getMinersMPKs(balances)
	if err != nil {
		return common.NewErrorf("move_to_share_or_publish_failed",
			"phase: %v, err: %v", pn.Phase, err)
	}

	if len(mpks.Mpks) == 0 {
		return common.NewError("move_to_share_or_publish_failed", "empty miners mpks keys")
	}

	// should have at least one miner from previous VC set
	if !gn.hasPrevMinerInMPKs(mpks, balances) {
		return common.NewErrorf("move_to_share_or_publish_failed",
			"no miner from previous VC set in MPKS, l_mpks: %d, DB: %d, DB version: %d",
			len(mpks.Mpks),
			int(balances.GetState().GetVersion()),
			int(balances.GetState().GetVersion()))
	}

	if len(mpks.Mpks) < dkgMinersList.K {
		return common.NewErrorf("move_to_share_or_publish_failed",
			"len(mpks.Mpks) < dkgMinersList.K, l_mpks: %d, K: %d",
			len(mpks.Mpks), dkgMinersList.K)
	}

	Logger.Debug("miner sc: move phase to share or publish",
		zap.Int("mpks", len(mpks.Mpks)),
		zap.Int("K", dkgMinersList.K),
		zap.Int64("DB version", int64(balances.GetState().GetVersion())))

	return nil
}

func (msc *MinerSmartContract) moveToWait(balances cstate.StateContextI,
	pn *PhaseNode, gn *GlobalNode) error {

	dkgMinersList, err := getDKGMinersList(balances)
	if err != nil {
		return common.NewErrorf("move_to_wait_failed",
			"failed to get miners DKG, phase: %v, err: %v", pn.Phase, err)
	}

	gsos, err := getGroupShareOrSigns(balances)
	switch err {
	case nil:
	case util.ErrValueNotPresent:
		return common.NewError("move_to_wait_failed", "empty sharder or sign keys")
	default:
		return common.NewErrorf("move_to_wait_failed", "phase: %v, err: %v", pn.Phase, err)
	}

	if !gn.hasPrevMinerInGSoS(gsos, balances) {
		return common.NewErrorf("move_to_wait_failed",
			"no miner from previous VC set in GSoS, "+
				"l_gsos: %d, K: %d, DB version: %d, gsos_shares: %v",
			len(gsos.Shares), dkgMinersList.K, int(balances.GetState().GetVersion()), gsos.Shares)
	}

	if len(gsos.Shares) < dkgMinersList.K {
		return common.NewErrorf("move_to_wait_failed",
			"len(gsos.Shares) < dkgMinersList.K, l_gsos: %d, K: %d",
			len(gsos.Shares), dkgMinersList.K)
	}

	Logger.Debug("miner sc: move phase to wait",
		zap.Int("shares", len(gsos.Shares)),
		zap.Int("K", dkgMinersList.K))

	return nil
}

func (msc *MinerSmartContract) moveToStart(balances cstate.StateContextI,
	pn *PhaseNode, gn *GlobalNode) error {
	return nil
}

// GetPhaseNode gets phase nodes from client state
func GetPhaseNode(statectx cstate.CommonStateContextI) (
	*PhaseNode, error) {

	pn := &PhaseNode{}
	err := statectx.GetTrieNode(pn.GetKey(), pn)
	if err != nil {
		if err != util.ErrValueNotPresent {
			return nil, err
		}

		pn.Phase = Start
		pn.CurrentRound = statectx.GetBlock().Round
		pn.StartRound = statectx.GetBlock().Round
		return pn, nil
	}

	pn.CurrentRound = statectx.GetBlock().Round
	return pn, nil
}

func (msc *MinerSmartContract) setPhaseNode(balances cstate.StateContextI,
	pn *PhaseNode, gn *GlobalNode, t *transaction.Transaction, isViewChange bool) error {

	// move phase condition
	var movePhase = isViewChange &&
		pn.CurrentRound-pn.StartRound >= PhaseRounds[pn.Phase]

	// move
	if movePhase {
		Logger.Debug("setPhaseNode",
			zap.String("phase", pn.Phase.String()),
			zap.Int64("phase current round", pn.CurrentRound),
			zap.Int64("phase start round", pn.StartRound))

		currentMoveFunc := moveFunctions[pn.Phase]
		if err := currentMoveFunc(balances, pn, gn); err != nil {
			Logger.Error("failed to move phase",
				zap.Any("phase", pn.Phase),
				zap.Int64("phase start round", pn.StartRound),
				zap.Int64("phase current round", pn.CurrentRound),
				zap.Any("move_func", getFunctionName(currentMoveFunc)),
				zap.Error(err))
			msc.RestartDKG(pn, balances)
		} else {
			Logger.Debug("setPhaseNode move phase success", zap.String("phase", pn.Phase.String()))
			var err error
			if phaseFunc, ok := phaseFuncs[pn.Phase]; ok {
				if lock, found := lockPhaseFunctions[pn.Phase]; found {
					lock.Lock()
					err = phaseFunc(balances, gn)
					lock.Unlock()
				} else {
					err = phaseFunc(balances, gn)
				}

				if err != nil {
					msc.RestartDKG(pn, balances)
					Logger.Error("failed to set phase node",
						zap.Any("error", err),
						zap.Any("phase", pn.Phase))
				}
			}
			if err == nil {
				if pn.Phase >= Phase(len(PhaseRounds)-1) {
					pn.Phase = 0
					pn.Restarts = 0
				} else {
					pn.Phase++
				}
				pn.StartRound = pn.CurrentRound
				Logger.Debug("setPhaseNode", zap.String("next_phase", pn.Phase.String()))
			}
		}
	}

	_, err := balances.InsertTrieNode(pn.GetKey(), pn)
	if err != nil && err != util.ErrValueNotPresent {
		Logger.DPanic("failed to set phase node -- insert failed",
			zap.Any("error", err))
		return err
	}

	return nil
}

func (msc *MinerSmartContract) createDKGMinersForContribute(
	balances cstate.StateContextI, gn *GlobalNode) error {

	allMinersList, err := msc.getMinersList(balances)
	if err != nil {
		Logger.Error("createDKGMinersForContribute -- failed to get miner list",
			zap.Any("error", err))
		return err
	}

	if len(allMinersList.Nodes) < gn.MinN {
		return common.NewErrorf("failed to create dkg miners", "too few miners for dkg, l_all_miners: %d, N: %d", len(allMinersList.Nodes), gn.MinN)
	}

	dkgMiners := NewDKGMinerNodes()
	if lmb := balances.GetChainCurrentMagicBlock(); lmb != nil {
		num := lmb.Miners.Size()
		Logger.Debug("Calculate TKN from lmb",
			zap.Int64("starting round", lmb.StartingRound),
			zap.Int("miners num", num))
		if num >= gn.MinN {
			dkgMiners.calculateTKN(gn, num)
		} else {
			Logger.Debug("Calculate TKN from all miner list",
				zap.Int("all count", len(allMinersList.Nodes)),
				zap.Int64("gn.LastRound", gn.LastRound))
			dkgMiners.calculateTKN(gn, len(allMinersList.Nodes))
		}
	} else {
		Logger.Debug("Calculate TKN from all miner list",
			zap.Int("all count", len(allMinersList.Nodes)),
			zap.Int64("gn.LastRound", gn.LastRound))
		dkgMiners.calculateTKN(gn, len(allMinersList.Nodes))
	}

	for _, nd := range allMinersList.Nodes {
		dkgMiners.SimpleNodes[nd.ID] = nd.SimpleNode
	}

	dkgMiners.StartRound = gn.LastRound
	if err := updateDKGMinersList(balances, dkgMiners); err != nil {
		return err
	}

	// sharders
	allSharderKeepList := new(MinerNodes)
	return updateShardersKeepList(balances, allSharderKeepList)
}

func (msc *MinerSmartContract) widdleDKGMinersForShare(
	balances cstate.StateContextI, gn *GlobalNode) error {

	dkgMiners, err := getDKGMinersList(balances)
	if err != nil {
		Logger.Error("widdle dkg miners -- failed to get dkgMiners",
			zap.Any("error", err))
		return err
	}

	msc.mutexMinerMPK.Lock()
	defer msc.mutexMinerMPK.Unlock()

	mpks, err := getMinersMPKs(balances)
	if err != nil {
		Logger.Error("widdle dkg miners -- failed to get miners mpks",
			zap.Any("error", err))
		return err
	}

	for k := range dkgMiners.SimpleNodes {
		if _, ok := mpks.Mpks[k]; !ok {
			delete(dkgMiners.SimpleNodes, k)
		}
	}

	if err = dkgMiners.reduceNodes(false, gn, balances); err != nil {
		Logger.Error("widdle dkg miners", zap.Error(err))
		return err
	}

	if err := updateDKGMinersList(balances, dkgMiners); err != nil {
		Logger.Error("widdle dkg miners -- failed to insert dkg miners",
			zap.Any("error", err))
		return err
	}
	return nil
}

func (msc *MinerSmartContract) reduceShardersList(
	keep,
	all *MinerNodes,
	gn *GlobalNode,
	balances cstate.StateContextI) (nodes []*MinerNode, err error) {

	simpleNodes := NewSimpleNodes()

	tmpMinerNodes := make([]*MinerNode, 0, len(keep.Nodes))

	for _, keepNode := range keep.Nodes {
		var found = all.FindNodeById(keepNode.ID)
		if found == nil {
			return nil, common.NewErrorf("invalid state", "a sharder exists in"+
				" keep list doesn't exists in all sharders list: %s", keepNode.ID)
		}
		tmpMinerNodes = append(tmpMinerNodes, found)
		simpleNodes[found.ID] = found.SimpleNode
	}

	if len(simpleNodes) < gn.MinS {
		return nil, fmt.Errorf("to few sharders: %d, want at least: %d", len(simpleNodes), gn.MinS)
	}

	var pmbrss int64
	var pmbnp *node.Pool
	pmb := balances.GetLastestFinalizedMagicBlock()
	if pmb != nil {
		pmbrss = pmb.RoundRandomSeed
		if pmb.MagicBlock != nil {
			pmbnp = pmb.MagicBlock.Sharders
		}
	}
	logging.Logger.Debug("sharder keep before", zap.Int("num", len(simpleNodes)))
	simpleNodes.reduce(gn.MaxS, gn.XPercent, pmbrss, pmbnp)
	logging.Logger.Debug("sharder keep after", zap.Int("num", len(simpleNodes)))

	nodes = make([]*MinerNode, 0, len(simpleNodes))

	for _, mn := range tmpMinerNodes {
		if sn, ok := simpleNodes[mn.ID]; ok {
			mn.SimpleNode = sn
			nodes = append(nodes, mn)
		}
	}

	if !hasPrevSharderInList(pmb.MagicBlock, nodes) {
		var prev = rankedPrevSharders(pmb.MagicBlock, nodes)
		if len(prev) == 0 {
			panic("must not happen")
		}
		nodes = append(nodes, prev[0])
	}

	return
}

func (msc *MinerSmartContract) createMagicBlockForWait(
	balances cstate.StateContextI, gn *GlobalNode) error {

	pn, err := GetPhaseNode(balances)
	if err != nil {
		return err
	}
	dkgMinersList, err := getDKGMinersList(balances)
	if err != nil {
		return err
	}

	gsos, err := getGroupShareOrSigns(balances)
	switch err {
	case nil:
	case util.ErrValueNotPresent:
		gsos = block.NewGroupSharesOrSigns()
	default:
		return err
	}

	msc.mutexMinerMPK.Lock()
	defer msc.mutexMinerMPK.Unlock()

	mpks, err := getMinersMPKs(balances)
	if err != nil {
		return common.NewError("create_magic_block_failed", err.Error())
	}

	for key := range mpks.Mpks {
		if _, ok := gsos.Shares[key]; !ok {
			Logger.Debug("create magic block - delete miner because no share found", zap.String("key", key))
			delete(dkgMinersList.SimpleNodes, key)
			delete(gsos.Shares, key)
			delete(mpks.Mpks, key)
		}
	}
	// TODO: check the necessary of the commented code below
	//for key, sharesRevealed := range dkgMinersList.RevealedShares {
	//	if sharesRevealed >= dkgMinersList.T {
	//		Logger.Debug("create magic block - delete miner because share revealed >= T found",
	//			zap.String("key", key),
	//			zap.Int("shares revealed", sharesRevealed),
	//			zap.Int("T", dkgMinersList.T))
	//		delete(dkgMinersList.SimpleNodes, key)
	//		delete(gsos.Shares, key)
	//		delete(mpks.Mpks, key)
	//	}
	//}

	// sharders
	sharders, err := getShardersKeepList(balances)
	if err != nil {
		return err
	}
	allSharderList, err := getAllShardersList(balances)
	if err != nil {
		return err
	}

	if sharders == nil || len(sharders.Nodes) == 0 {
		sharders = allSharderList
	} else {
		sharders.Nodes, err = msc.reduceShardersList(sharders, allSharderList, gn, balances)
		if err != nil {
			return err
		}
	}

	if err = dkgMinersList.reduceNodes(true, gn, balances); err != nil {
		Logger.Error("create magic block for wait - reduce nodes failed", zap.Error(err))
		return err
	}

	for id := range gsos.Shares {
		if _, ok := dkgMinersList.SimpleNodes[id]; !ok {
			delete(gsos.Shares, id)
		}
	}

	for id := range mpks.Mpks {
		if _, ok := dkgMinersList.SimpleNodes[id]; !ok {
			delete(mpks.Mpks, id)
		}
	}

	if len(dkgMinersList.SimpleNodes) < dkgMinersList.K {
		return common.NewErrorf("create_magic_block_failed",
			"len(dkgMinersList.SimpleNodes) [%d] < dkgMinersList.K [%d]", len(dkgMinersList.SimpleNodes), dkgMinersList.K)
	}

	magicBlock, err := msc.createMagicBlock(balances, sharders, dkgMinersList, gsos, mpks, pn)
	if err != nil {
		return err
	}

	gn.ViewChange = magicBlock.StartingRound
	mpks = block.NewMpks()
	if err := updateMinersMPKs(balances, mpks); err != nil {
		return err
	}

	Logger.Debug("create_mpks in createMagicBlockForWait", zap.Int64("DB version", int64(balances.GetState().GetVersion())))

	gsos = block.NewGroupSharesOrSigns()
	if err := updateGroupShareOrSigns(balances, gsos); err != nil {
		return err
	}

	err = updateMagicBlock(balances, magicBlock)
	if err != nil {
		Logger.Error("failed to insert magic block", zap.Any("error", err))
		return err
	}
	// dkgMinersList = NewDKGMinerNodes()
	// _, err = balances.InsertTrieNode(DKGMinersKey, dkgMinersList)
	// if err != nil {
	// 	return err
	// }
	allSharderKeepList := new(MinerNodes)
	return updateShardersKeepList(balances, allSharderKeepList)
}

func (msc *MinerSmartContract) contributeMpk(t *transaction.Transaction,
	inputData []byte, gn *GlobalNode, balances cstate.StateContextI) (
	string, error) {
	Logger.Debug("miner smart contract, contribute Mpk")
	pn, err := GetPhaseNode(balances)
	if err != nil {
		return "", common.NewErrorf("contribute_mpk_failed",
			"can't get phase node: %v", err)
	}

	if pn.Phase != Contribute {
		return "", common.NewErrorf("contribute_mpk_failed",
			"this is not the correct phase to contribute mpk: %v", pn.Phase.String())
	}

	dmn, err := getDKGMinersList(balances)
	if err != nil {
		return "", err
	}

	if _, ok := dmn.SimpleNodes[t.ClientID]; !ok {
		return "", common.NewError("contribute_mpk_failed",
			"miner not part of dkg set")
	}

	msc.mutexMinerMPK.Lock()
	defer msc.mutexMinerMPK.Unlock()

	mpk := &block.MPK{ID: t.ClientID}
	if err := mpk.Decode(inputData); err != nil {
		return "", common.NewErrorf("contribute_mpk_failed",
			"decoding request: %v", err)
	}

	if len(mpk.Mpk) != dmn.T {
		return "", common.NewErrorf("contribute_mpk_failed",
			"mpk sent (size: %v) is not correct size: %v", len(mpk.Mpk), dmn.T)
	}

	mpks, err := getMinersMPKs(balances)
	switch err {
	case util.ErrValueNotPresent:
		// the mpks could be empty when the first time to contribute mpks
		mpks = block.NewMpks()
	case nil:
	default:
		return "", common.NewError("contribute_mpk_failed", err.Error())
	}

	if _, ok := mpks.Mpks[mpk.ID]; ok {
		return "", common.NewError("contribute_mpk_failed",
			"already have mpk for miner")
	}

	mpks.Mpks[mpk.ID] = mpk
	if err := updateMinersMPKs(balances, mpks); err != nil {
		return "", common.NewError("contribute_mpk_failed", err.Error())
	}

	Logger.Debug("contribute_mpk success",
		zap.Int64("DB version", int64(balances.GetState().GetVersion())),
		zap.String("mpk id", mpk.ID),
		zap.Int("len", len(mpks.Mpks)),
		zap.Int64("pn_start_round", pn.StartRound),
		zap.String("phase", pn.Phase.String()))

	return string(mpk.Encode()), nil
}

func (msc *MinerSmartContract) shareSignsOrShares(t *transaction.Transaction,
	inputData []byte, gn *GlobalNode, balances cstate.StateContextI) (
	resp string, err error) {

	var pn *PhaseNode
	if pn, err = GetPhaseNode(balances); err != nil {
		return "", common.NewErrorf("share_signs_or_shares",
			"can't get phase node: %v", err)
	}

	if pn.Phase != Publish {
		return "", common.NewErrorf("share_signs_or_shares",
			"this is not the correct phase to publish signs or shares, phase node: %v",
			string(pn.Encode()))
	}

	var gsos *block.GroupSharesOrSigns
	gsos, err = getGroupShareOrSigns(balances)
	switch err {
	case nil:
	case util.ErrValueNotPresent:
		gsos = block.NewGroupSharesOrSigns()
	default:
		return "", common.NewError("share_signs_or_shares_failed", err.Error())
	}

	var ok bool
	if _, ok = gsos.Shares[t.ClientID]; ok {
		return "", common.NewErrorf("share_signs_or_shares",
			"already have share or signs for miner %v", t.ClientID)
	}

	var dmn *DKGMinerNodes
	if dmn, err = getDKGMinersList(balances); err != nil {
		return "", common.NewErrorf("share_signs_or_shares",
			"getting miners DKG list %v", err)
	}

	var sos = block.NewShareOrSigns()
	if err = sos.Decode(inputData); err != nil {
		return "", common.NewErrorf("share_signs_or_shares",
			"decoding input %v", err)
	}

	if len(sos.ShareOrSigns) < dmn.K-1 {
		return "", common.NewErrorf("share_signs_or_shares",
			"not enough share or signs for this dkg, l_sos: %d, K - 1: %d",
			len(sos.ShareOrSigns), dmn.K-1)
	}

	msc.mutexMinerMPK.Lock()
	defer msc.mutexMinerMPK.Unlock()

	var mpks *block.Mpks
	mpks, err = getMinersMPKs(balances)
	if err != nil {
		return "", common.NewError("share_signs_or_shares_failed", err.Error())
	}

	var publicKeys = make(map[string]string)
	for key, miner := range dmn.SimpleNodes {
		publicKeys[key] = miner.PublicKey
	}

	var shares []string
	shares, ok = sos.Validate(mpks, publicKeys, balances.GetSignatureScheme())
	if !ok {
		return "", common.NewError("share_signs_or_shares",
			"share or signs failed validation")
	}

	for _, share := range shares {
		dmn.RevealedShares[share]++
	}

	sos.ID = t.ClientID
	gsos.Shares[t.ClientID] = sos

	Logger.Debug("update gsos",
		zap.Int64("gn.LastRound", gn.LastRound),
		zap.Int64("state.version", int64(balances.GetState().GetVersion())))
	err = updateGroupShareOrSigns(balances, gsos)
	if err != nil {
		return "", common.NewErrorf("share_signs_or_shares",
			"saving group share of signs: %v", err)
	}

	if err := updateDKGMinersList(balances, dmn); err != nil {
		return "", common.NewErrorf("share_signs_or_shares",
			"saving DKG miners: %v", err)
	}

	return string(sos.Encode()), nil
}

// Wait used to notify SC that DKG summary and magic block data saved by
// a miner locally and VC can moves on. If >= T miners don't save their
// DKG/MB then we can't do a view change because some miners can save,
// but some can not. And this case we got BC stuck.
func (msc *MinerSmartContract) wait(t *transaction.Transaction,
	inputData []byte, gn *GlobalNode, balances cstate.StateContextI) (
	resp string, err error) {

	var pn *PhaseNode
	if pn, err = GetPhaseNode(balances); err != nil {
		return "", common.NewErrorf("msc - wait",
			"can't get phase node: %v", err)
	}

	if pn.Phase != Wait {
		return "", common.NewErrorf("msc - wait", "this is not the"+
			" correct phase to wait: %s", pn.Phase)
	}

	var dmn *DKGMinerNodes
	if dmn, err = getDKGMinersList(balances); err != nil {
		return "", common.NewErrorf("msc - wait", "can't get DKG miners: %v", err)
	}

	if already, ok := dmn.Waited[t.ClientID]; ok && already {
		return "", common.NewError("msc - wait", "already checked in")
	}

	dmn.Waited[t.ClientID] = true

	if err := updateDKGMinersList(balances, dmn); err != nil {
		return "", common.NewErrorf("msc - wait", "saving DKG miners: %v", err)
	}

	return
}

func (msc *MinerSmartContract) createMagicBlock(
	balances cstate.StateContextI,
	sharders *MinerNodes,
	dkgMinersList *DKGMinerNodes,
	gsos *block.GroupSharesOrSigns,
	mpks *block.Mpks,
	pn *PhaseNode,
) (*block.MagicBlock, error) {

	pmb := balances.GetLastestFinalizedMagicBlock()

	magicBlock := block.NewMagicBlock()

	magicBlock.Miners = node.NewPool(node.NodeTypeMiner)
	magicBlock.Sharders = node.NewPool(node.NodeTypeSharder)
	magicBlock.SetShareOrSigns(gsos)
	magicBlock.Mpks = mpks
	magicBlock.T = dkgMinersList.T
	magicBlock.K = dkgMinersList.K
	magicBlock.N = dkgMinersList.N
	magicBlock.MagicBlockNumber = pmb.MagicBlock.MagicBlockNumber + 1
	magicBlock.PreviousMagicBlockHash = pmb.MagicBlock.Hash
	magicBlock.StartingRound = pn.CurrentRound + PhaseRounds[Wait]

	logging.Logger.Debug("create magic block",
		zap.Int64("view change", magicBlock.StartingRound),
		zap.Int("dkg miners num", len(dkgMinersList.SimpleNodes)))

	for _, v := range dkgMinersList.SimpleNodes {
		n := node.Provider()
		n.ID = v.ID
		n.N2NHost = v.N2NHost
		n.Host = v.Host
		n.Port = v.Port
		n.Path = v.Path
		n.PublicKey = v.PublicKey
		n.Description = v.ShortName
		n.Type = node.NodeTypeMiner
		n.Info.BuildTag = v.BuildTag
		n.Status = node.NodeStatusActive
		n.InPrevMB = pmb.Miners.HasNode(v.ID)
		if err := magicBlock.Miners.AddNode(n); err != nil {
			return nil, err
		}

		mn := NewMinerNode()
		mn.SimpleNode = v
		mn.Status = n.Status
		emitAddOrOverwriteMiner(mn, balances)

	}

	for _, v := range sharders.Nodes {
		n := node.Provider()
		n.ID = v.ID
		n.N2NHost = v.N2NHost
		n.Host = v.Host
		n.Port = v.Port
		n.Path = v.Path
		n.PublicKey = v.PublicKey
		n.Description = v.ShortName
		n.Type = node.NodeTypeSharder
		n.Info.BuildTag = v.BuildTag
		n.Status = node.NodeStatusActive
		n.InPrevMB = pmb.Sharders.HasNode(v.ID)
		if err := magicBlock.Sharders.AddNode(n); err != nil {
			return nil, err
		}

		sn := v
		sn.Status = n.Status
		if err := emitAddOrOverwriteSharder(sn, balances); err != nil {
			return nil, err
		}
	}

	magicBlock.Hash = magicBlock.GetHash()
	return magicBlock, nil
}

func (msc *MinerSmartContract) RestartDKG(pn *PhaseNode,
	balances cstate.StateContextI) {
	Logger.Debug("RestartDKG", zap.Int64("DB version", int64(balances.GetState().GetVersion())))
	msc.mutexMinerMPK.Lock()
	defer msc.mutexMinerMPK.Unlock()
	mpks := block.NewMpks()
	if err := updateMinersMPKs(balances, mpks); err != nil {
		Logger.Error("failed to restart dkg", zap.Any("error", err))
	}

	Logger.Debug("create_mpks in RestartDKG", zap.Int64("DB version", int64(balances.GetState().GetVersion())))

	gsos := block.NewGroupSharesOrSigns()
	if err := updateGroupShareOrSigns(balances, gsos); err != nil {
		Logger.Error("failed to restart dkg", zap.Any("error", err))
	}
	dkgMinersList := NewDKGMinerNodes()
	dkgMinersList.StartRound = pn.CurrentRound
	if err := updateDKGMinersList(balances, dkgMinersList); err != nil {
		Logger.Error("failed to restart dkg", zap.Any("error", err))
	}

	sharderKeepList := new(MinerNodes)
	if err := updateShardersKeepList(balances, sharderKeepList); err != nil {
		Logger.Error("failed to restart dkg", zap.Any("error", err))
	}
	pn.Phase = Start
	pn.Restarts++
	pn.StartRound = pn.CurrentRound
}

func (msc *MinerSmartContract) SetMagicBlock(gn *GlobalNode,
	balances cstate.StateContextI) bool {
	magicBlock, err := getMagicBlock(balances)
	if err != nil {
		Logger.Error("could not get magic block from MPT", zap.Error(err))
		return false
	}

	if magicBlock.StartingRound == 0 && magicBlock.MagicBlockNumber == 0 {
		Logger.Error("SetMagicBlock smart contract, starting round is 0, magic block number is 0")
	}

	// keep the magic block to track previous nodes list next view change
	// (deny VC leaving for at least 1 miner and 1 sharder of previous set)
	gn.PrevMagicBlock = magicBlock

	balances.SetMagicBlock(magicBlock)
	return true
}

func getFunctionName(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}
