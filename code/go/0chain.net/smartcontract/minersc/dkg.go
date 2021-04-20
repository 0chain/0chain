package minersc

import (
	"0chain.net/smartcontract"
	"errors"
	"reflect"
	"runtime"
	"sort"

	"0chain.net/chaincore/block"
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
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
	pn *PhaseNode, gn *GlobalNode) (ok bool) {

	var (
		allMinersList *MinerNodes
		dkgMinersList *DKGMinerNodes

		allShardersList *MinerNodes

		err error
	)

	if allMinersList, err = msc.GetMinersList(balances); err != nil {
		return false
	}

	if dkgMinersList, err = msc.getMinersDKGList(balances); err != nil {
		return false
	}
	allShardersList, err = msc.getShardersList(balances, AllShardersKey)
	if err != nil {
		return false
	}

	if len(allShardersList.Nodes) < gn.MinS {
		Logger.Error("not enough sharders in all sharders list to move phase",
			zap.Int("all", len(allShardersList.Nodes)),
			zap.Int("min_s", gn.MinS))
		return false
	}

	if !gn.hasPrevShader(allShardersList, balances) {
		Logger.Error("invalid state: all sharders list hasn't a sharder"+
			" from previous VC set",
			zap.Int("all", len(allShardersList.Nodes)),
			zap.Int("min_s", gn.MinS))
		return false
	}

	if !gn.hasPrevMiner(allMinersList, balances) {
		Logger.Error("invalid state: all miners list hasn't a miner"+
			" from previous VC set",
			zap.Int("all", len(allMinersList.Nodes)),
			zap.Int("min_n", gn.MinN))
		return false
	}

	ok = allMinersList != nil &&
		len(allMinersList.Nodes) >= dkgMinersList.K &&
		len(allShardersList.Nodes) >= gn.MinS

	Logger.Debug("miner sc: move phase to contribute",
		zap.Int("miners", len(allMinersList.Nodes)),
		zap.Int("K", dkgMinersList.K),
		zap.Int("sharders", len(allShardersList.Nodes)),
		zap.Int("min_s", gn.MinS),
		zap.Bool("ok", ok))
	return
}

func (msc *MinerSmartContract) moveToShareOrPublish(
	balances cstate.StateContextI, pn *PhaseNode, gn *GlobalNode) (
	ok bool) {

	var (
		dkgMinersList *DKGMinerNodes
		mpks          = block.NewMpks()

		shardersKeep, err = msc.getShardersList(balances, ShardersKeepKey)
	)

	if err != nil {
		Logger.Error("failed to get sharders keep list", zap.Error(err))
		return false
	}

	if len(shardersKeep.Nodes) < gn.MinS {
		Logger.Error("not enough sharders in keep list to move phase",
			zap.Int("keep", len(shardersKeep.Nodes)),
			zap.Int("min_s", gn.MinS))
		return false
	}

	if !gn.hasPrevShader(shardersKeep, balances) {
		Logger.Error("missing at least one sharder from previous set in "+
			"sharders keep list to move phase",
			zap.Int("keep", len(shardersKeep.Nodes)),
			zap.Int("min_s", gn.MinS))
		return false
	}

	dkgMinersList, err = msc.getMinersDKGList(balances)
	if err != nil {
		Logger.Error("failed to get miners DKG", zap.Any("error", err),
			zap.Any("phase", pn.Phase))
		return false
	}

	msc.mutexMinerMPK.Lock()
	defer msc.mutexMinerMPK.Unlock()

	var mpksBytes util.Serializable
	mpksBytes, err = balances.GetTrieNode(MinersMPKKey)
	if err != nil {
		Logger.Error("failed to get node MinersMPKKey", zap.Any("error", err),
			zap.Any("phase", pn.Phase))
		return false
	}

	if err = mpks.Decode(mpksBytes.Encode()); err != nil {
		Logger.Error("failed to decode node MinersMPKKey",
			zap.Any("error", err), zap.Any("phase", pn.Phase))
		return false
	}

	// should have at least one miner from previous VC set
	if !gn.hasPrevMinerInMPKs(mpks, balances) {
		Logger.Error("no miner from previous VC set in MPKS",
			zap.Int("l_mpks", len(mpks.Mpks)),
			zap.Int("DB", int(balances.GetState().GetVersion())),
			zap.Int("DB version", int(balances.GetState().GetVersion())),
		)
		return false
	}

	ok = mpks != nil && len(mpks.Mpks) >= dkgMinersList.K

	Logger.Debug("miner sc: move phase to share or publish",
		zap.Int("mpks", len(mpks.Mpks)),
		zap.Int("K", dkgMinersList.K),
		zap.Int64("DB version", int64(balances.GetState().GetVersion())),
		zap.Bool("ok", ok))

	return
}

func (msc *MinerSmartContract) moveToWait(balances cstate.StateContextI,
	pn *PhaseNode, gn *GlobalNode) (ok bool) {

	var err error
	var dkgMinersList *DKGMinerNodes
	gsos := block.NewGroupSharesOrSigns()

	dkgMinersList, err = msc.getMinersDKGList(balances)
	if err != nil {
		Logger.Error("failed to get miners DKG", zap.Any("error", err),
			zap.Any("phase", pn.Phase))
		return false
	}

	var groupBytes util.Serializable
	groupBytes, err = balances.GetTrieNode(GroupShareOrSignsKey)
	if err != nil {
		Logger.Error("failed to get node GroupShareOrSignsKey",
			zap.Any("error", err), zap.Any("phase", pn.Phase))
		return false
	}

	if err = gsos.Decode(groupBytes.Encode()); err != nil {
		Logger.Error("failed to decode node GroupShareOrSignsKey",
			zap.Any("error", err), zap.Any("phase", pn.Phase))
		return false
	}

	if !gn.hasPrevMinerInGSoS(gsos, balances) {
		Logger.Error("no miner from previous VC set in GSoS",
			zap.Int("l_gsos", len(gsos.Shares)),
			zap.Int("K", dkgMinersList.K),
			zap.Int("DB version", int(balances.GetState().GetVersion())),
			zap.Any("gsos_shares", gsos.Shares))
		return false
	}

	ok = len(gsos.Shares) >= dkgMinersList.K

	Logger.Debug("miner sc: move phase to wait",
		zap.Int("shares", len(gsos.Shares)),
		zap.Int("K", dkgMinersList.K),
		zap.Bool("ok", ok))

	return
}

func (msc *MinerSmartContract) moveToStart(balances cstate.StateContextI,
	pn *PhaseNode, gn *GlobalNode) bool {
	return true
}

func (msc *MinerSmartContract) getPhaseNode(statectx cstate.StateContextI) (
	*PhaseNode, error) {

	pn := &PhaseNode{}
	phaseNodeBytes, err := statectx.GetTrieNode(pn.GetKey())
	if err != nil && err != util.ErrValueNotPresent {
		return nil, err
	}
	if phaseNodeBytes == nil {
		pn.Phase = Start
		pn.CurrentRound = statectx.GetBlock().Round
		pn.StartRound = statectx.GetBlock().Round
		return pn, nil
	}
	pn.Decode(phaseNodeBytes.Encode())
	pn.CurrentRound = statectx.GetBlock().Round
	return pn, nil
}

func (msc *MinerSmartContract) setPhaseNode(balances cstate.StateContextI,
	pn *PhaseNode, gn *GlobalNode, t *transaction.Transaction) error {

	// move phase condition
	var movePhase = config.DevConfiguration.ViewChange &&
		pn.CurrentRound-pn.StartRound >= PhaseRounds[pn.Phase]

	// move
	if movePhase {
		currentMoveFunc := moveFunctions[pn.Phase]
		if currentMoveFunc(balances, pn, gn) {
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
				if Phase(len(PhaseRounds))-1 > pn.Phase {
					pn.Phase++
				} else {
					pn.Phase = 0
					pn.Restarts = 0
				}
				pn.StartRound = pn.CurrentRound
			}
		} else {
			Logger.Error("failed to move phase",
				zap.Any("phase", pn.Phase),
				zap.Any("move_func", getFunctionName(currentMoveFunc)))
			msc.RestartDKG(pn, balances)
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

	allMinersList, err := msc.GetMinersList(balances)
	if err != nil {
		Logger.Error("createDKGMinersForContribute -- failed to get miner list",
			zap.Any("error", err))
		return err
	}

	if len(allMinersList.Nodes) < gn.MinN {
		return common.NewError("failed to create dkg miners", "too few miners for dkg")
	}

	dkgMiners := NewDKGMinerNodes()
	if lmb := balances.GetChainCurrentMagicBlock(); lmb != nil {
		activeCount := lmb.Miners.GetActiveCount()
		Logger.Debug("Calculate TKN from lmb",
			zap.Int("active count", activeCount),
			zap.Int64("starting round", lmb.StartingRound))
		dkgMiners.calculateTKN(gn, activeCount)
	} else {
		Logger.Debug("Calculate TKN from all miner list",
			zap.Int("all count", len(allMinersList.Nodes)),
			zap.Int64("gn.LastRound", gn.LastRound))
		dkgMiners.calculateTKN(gn, len(allMinersList.Nodes))
	}

	for _, node := range allMinersList.Nodes {
		dkgMiners.SimpleNodes[node.ID] = node.SimpleNode
	}

	dkgMiners.StartRound = gn.LastRound
	_, err = balances.InsertTrieNode(DKGMinersKey, dkgMiners)
	if err != nil {
		return err
	}

	//sharders
	allSharderKeepList := new(MinerNodes)
	_, err = balances.InsertTrieNode(ShardersKeepKey, allSharderKeepList)
	if err != nil {
		return err
	}
	return nil
}

func (msc *MinerSmartContract) widdleDKGMinersForShare(
	balances cstate.StateContextI, gn *GlobalNode) error {

	dkgMiners, err := msc.getMinersDKGList(balances)
	if err != nil {
		Logger.Error("widdle dkg miners -- failed to get dkgMiners",
			zap.Any("error", err))
		return err
	}

	msc.mutexMinerMPK.Lock()
	defer msc.mutexMinerMPK.Unlock()

	mpks := block.NewMpks()
	mpksBytes, err := balances.GetTrieNode(MinersMPKKey)
	if err != nil {
		Logger.Error("widdle dkg miners -- failed to get miners mpks",
			zap.Any("error", err))
		return err
	}
	if err = mpks.Decode(mpksBytes.Encode()); err != nil {
		return err
	}
	for k := range dkgMiners.SimpleNodes {
		if _, ok := mpks.Mpks[k]; !ok {
			delete(dkgMiners.SimpleNodes, k)
		}
	}

	if err = dkgMiners.recalculateTKN(false, gn, balances); err != nil {
		Logger.Error("widdle dkg miners", zap.Error(err))
		return err
	}

	_, err = balances.InsertTrieNode(DKGMinersKey, dkgMiners)
	if err != nil {
		Logger.Error("widdle dkg miners -- failed to insert dkg miners",
			zap.Any("error", err))
		return err
	}
	return nil
}

func (msc *MinerSmartContract) reduceShardersList(keep, all *MinerNodes,
	gn *GlobalNode, balances cstate.StateContextI) (
	list []*MinerNode, err error) {

	var (
		pmb  = gn.prevMagicBlock(balances)
		hasp bool // TODO (sfxdx): remove the temporary debug code
	)

	list = make([]*MinerNode, 0, len(keep.Nodes))
	for _, ksh := range keep.Nodes {
		var ash = all.FindNodeById(ksh.ID)
		if ash == nil {
			return nil, common.NewErrorf("invalid state", "a sharder exists in"+
				" keep list doesn't exists in all sharders list: %s", ksh.ID)
		}
		list = append(list, ash)
		hasp = hasp || pmb.Sharders.HasNode(ksh.ID)
	}

	if !hasp {
		panic("must not happen")
	}

	if len(list) <= gn.MaxS {
		return // doesn't need to sort, has sharder from previous set
	}

	// get max staked
	sort.Slice(list, func(i, j int) bool {
		return list[i].TotalStaked > list[j].TotalStaked ||
			list[i].ID < list[j].ID
	})

	if !gn.hasPrevSharderInList(list[:gn.MaxS], balances) {
		var prev = gn.rankedPrevSharders(list, balances)
		if len(prev) == 0 {
			panic("must not happen")
		}
		list[gn.MaxS-1] = prev[0] // best rank
	}

	list = list[:gn.MaxS]
	return
}

func (msc *MinerSmartContract) createMagicBlockForWait(
	balances cstate.StateContextI, gn *GlobalNode) error {

	pn, err := msc.getPhaseNode(balances)
	if err != nil {
		return err
	}
	dkgMinersList, err := msc.getMinersDKGList(balances)
	if err != nil {
		return err
	}
	gsos := block.NewGroupSharesOrSigns()
	groupBytes, err := balances.GetTrieNode(GroupShareOrSignsKey)
	if err != nil {
		return err
	}
	if err = gsos.Decode(groupBytes.Encode()); err != nil {
		return err
	}

	msc.mutexMinerMPK.Lock()
	defer msc.mutexMinerMPK.Unlock()

	mpks := block.NewMpks()
	mpksBytes, err := balances.GetTrieNode(MinersMPKKey)
	if err != nil {
		return common.NewErrorf("create_magic_block_failed",
			"error with miner's mpk: %v", err)
	}
	if err = mpks.Decode(mpksBytes.Encode()); err != nil {
		return err
	}

	for key := range mpks.Mpks {
		if _, ok := gsos.Shares[key]; !ok {
			delete(dkgMinersList.SimpleNodes, key)
			delete(gsos.Shares, key)
			delete(mpks.Mpks, key)
		}
	}
	for key, sharesRevealed := range dkgMinersList.RevealedShares {
		if sharesRevealed == dkgMinersList.N {
			delete(dkgMinersList.SimpleNodes, key)
			delete(gsos.Shares, key)
			delete(mpks.Mpks, key)
		}
	}

	// sharders
	sharders, err := msc.getShardersList(balances, ShardersKeepKey)
	if err != nil {
		return err
	}
	allSharderList, err := msc.getShardersList(balances, AllShardersKey)
	if err != nil {
		return err
	}

	if sharders == nil || len(sharders.Nodes) == 0 {
		sharders = allSharderList
	} else {
		sharders.Nodes, err = msc.reduceShardersList(sharders, allSharderList,
			gn, balances)
		if err != nil {
			return err
		}
	}

	if err = dkgMinersList.recalculateTKN(true, gn, balances); err != nil {
		Logger.Error("create magic block for wait", zap.Error(err))
		return err
	}

	// 1. remove GSOS not listed in DKG
	// 2. remove MPKS not listed in DKG
	// 3. continue as usual

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

	magicBlock, err := msc.CreateMagicBlock(balances, sharders, dkgMinersList,
		gsos, mpks, pn)
	if err != nil {
		return err
	}

	gn.ViewChange = magicBlock.StartingRound
	mpks = block.NewMpks()
	_, err = balances.InsertTrieNode(MinersMPKKey, mpks)
	if err != nil {
		return err
	}
	Logger.Debug("create_mpks in createMagicBlockForWait", zap.Int64("DB version", int64(balances.GetState().GetVersion())))

	gsos = block.NewGroupSharesOrSigns()
	_, err = balances.InsertTrieNode(GroupShareOrSignsKey, gsos)
	if err != nil {
		return err
	}
	_, err = balances.InsertTrieNode(MagicBlockKey, magicBlock)
	if err != nil {
		Logger.Error("failed to insert magic block", zap.Any("error", err))
		return err
	}
	// dkgMinersList = NewDKGMinerNodes()
	// _, err = balances.InsertTrieNode(DKGMinersKey, dkgMinersList)
	// if err != nil {
	// 	return err
	// }
	allMinersList := new(MinerNodes)
	_, err = balances.InsertTrieNode(ShardersKeepKey, allMinersList)
	if err != nil {
		return err
	}

	return nil
}

func (msc *MinerSmartContract) contributeMpk(t *transaction.Transaction,
	inputData []byte, gn *GlobalNode, balances cstate.StateContextI) (
	resp string, err error) {

	var pn *PhaseNode
	if pn, err = msc.getPhaseNode(balances); err != nil {
		return "", common.NewErrorf("contribute_mpk_failed",
			"can't get phase node: %v", err)
	}

	if pn.Phase != Contribute {
		return "", common.NewError("contribute_mpk_failed",
			"this is not the correct phase to contribute mpk")
	}

	var dmn *DKGMinerNodes
	if dmn, err = msc.getMinersDKGList(balances); err != nil {
		return "", err
	}

	var ok bool
	if _, ok = dmn.SimpleNodes[t.ClientID]; !ok {
		return "", common.NewError("contribute_mpk_failed",
			"miner not part of dkg set")
	}

	msc.mutexMinerMPK.Lock()
	defer msc.mutexMinerMPK.Unlock()

	var (
		mpks      = block.NewMpks()
		mpk       = &block.MPK{ID: t.ClientID}
		mpksBytes util.Serializable
	)

	if mpksBytes, err = balances.GetTrieNode(MinersMPKKey); mpksBytes != nil {
		if err != nil {
			return "", common.NewError("contribute_mpk_failed", "failed to get MPKs from state db")
		}

		if err = mpks.Decode(mpksBytes.Encode()); err != nil {
			return "", common.NewErrorf("contribute_mpk_failed",
				"invalid state: decoding MPKS: %v", err)
		}
	}

	if err = mpk.Decode(inputData); err != nil {
		return "", common.NewErrorf("contribute_mpk_failed",
			"decoding request: %v", err)
	}

	if len(mpk.Mpk) != dmn.T {
		return "", common.NewErrorf("contribute_mpk_failed",
			"mpk sent (size: %v) is not correct size: %v", len(mpk.Mpk), dmn.T)
	}
	if _, ok = mpks.Mpks[mpk.ID]; ok {
		return "", common.NewError("contribute_mpk_failed",
			"already have mpk for miner")
	}

	mpks.Mpks[mpk.ID] = mpk
	_, err = balances.InsertTrieNode(MinersMPKKey, mpks)
	if err != nil {
		return "", common.NewErrorf("contribute_mpk_failed",
			"saving MPK key: %v", err)
	}

	Logger.Debug("contribute_mpk success", zap.Int64("DB version", int64(balances.GetState().GetVersion())))

	return string(mpk.Encode()), nil
}

func (msc *MinerSmartContract) shareSignsOrShares(t *transaction.Transaction,
	inputData []byte, gn *GlobalNode, balances cstate.StateContextI) (
	resp string, err error) {

	var pn *PhaseNode
	if pn, err = msc.getPhaseNode(balances); err != nil {
		return "", common.NewErrorf("share_signs_or_shares",
			"can't get phase node: %v", err)
	}

	if pn.Phase != Publish {
		return "", common.NewErrorf("share_signs_or_shares", "this is not the"+
			" correct phase to publish signs or shares, phase node: %v",
			string(pn.Encode()))
	}

	var (
		gsos      = block.NewGroupSharesOrSigns()
		gsosBytes util.Serializable
	)
	gsosBytes, err = balances.GetTrieNode(GroupShareOrSignsKey)
	if err != nil && err != util.ErrValueNotPresent {
		return "", common.NewErrorf("share_signs_or_shares",
			"can't get group share of signs: %v", err)
	}

	if err == nil {
		if err = gsos.Decode(gsosBytes.Encode()); err != nil {
			return "", common.NewErrorf("share_signs_or_shares",
				"decoding group share of signs: %v", err)
		}
	}

	err = nil // reset possible util.ErrValueNotPresent

	var ok bool
	if _, ok = gsos.Shares[t.ClientID]; ok {
		return "", common.NewErrorf("share_signs_or_shares",
			"already have share or signs for miner %v", t.ClientID)
	}

	var dmn *DKGMinerNodes
	if dmn, err = msc.getMinersDKGList(balances); err != nil {
		return "", common.NewErrorf("share_signs_or_shares",
			"getting miners DKG list %v", err)
	}

	var sos = block.NewShareOrSigns()
	if err = sos.Decode(inputData); err != nil {
		return "", common.NewErrorf("share_signs_or_shares",
			"decoding input %v", err)
	}

	// TODO (sfxdx): What the dmn.N-2 means here?
	//               Should it be T or K, shouldn't it?
	if len(sos.ShareOrSigns) < dmn.N-2 {
		return "", common.NewError("share_signs_or_shares",
			"number of share or signs doesn't equal N for this dkg")
	}

	msc.mutexMinerMPK.Lock()
	defer msc.mutexMinerMPK.Unlock()

	var (
		mpks      = block.NewMpks()
		mpksBytes util.Serializable
	)
	mpksBytes, err = balances.GetTrieNode(MinersMPKKey)
	if err != nil {
		return "", common.NewErrorf("share_signs_or_shares",
			"getting miners MPK: %v", err)
	}

	if err = mpks.Decode(mpksBytes.Encode()); err != nil {
		return "", common.NewErrorf("share_signs_or_shares",
			"invalid state: decoding miners MPK: %v", err)
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
	_, err = balances.InsertTrieNode(GroupShareOrSignsKey, gsos)
	if err != nil {
		return "", common.NewErrorf("share_signs_or_shares",
			"saving group share of signs: %v", err)
	}

	_, err = balances.InsertTrieNode(DKGMinersKey, dmn)
	if err != nil {
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
	if pn, err = msc.getPhaseNode(balances); err != nil {
		return "", common.NewErrorf("wait",
			"can't get phase node: %v", err)
	}

	if pn.Phase != Wait {
		return "", common.NewErrorf("wait", "this is not the"+
			" correct phase to wait: %s", pn.Phase)
	}

	var dmn *DKGMinerNodes
	if dmn, err = msc.getMinersDKGList(balances); err != nil {
		return "", common.NewErrorf("wait", "can't get DKG miners: %v", err)
	}

	if already, ok := dmn.Waited[t.ClientID]; ok && already {
		return "", common.NewError("wait", "already checked in")
	}

	dmn.Waited[t.ClientID] = true

	if _, err = balances.InsertTrieNode(DKGMinersKey, dmn); err != nil {
		return "", common.NewErrorf("wait", "saving DKG miners: %v", err)
	}

	return
}

func (msc *MinerSmartContract) getMinersDKGList(statectx cstate.StateContextI) (
	*DKGMinerNodes, error) {

	allMinersList := NewDKGMinerNodes()
	allMinersBytes, err := statectx.GetTrieNode(DKGMinersKey)
	if err != nil && err != util.ErrValueNotPresent {
		return nil, errors.New("get_miners_dkg_list_failed - " +
			"failed to retrieve existing miners list: " + err.Error())
	}
	if allMinersBytes == nil {
		return allMinersList, nil
	}
	err = allMinersList.Decode(allMinersBytes.Encode())
	if err != nil {
		return nil, smartcontract.NewError(smartcontract.DecodingErr, err)
	}
	return allMinersList, nil
}

func (msc *MinerSmartContract) CreateMagicBlock(balances cstate.StateContextI,
	sharderList *MinerNodes, dkgMinersList *DKGMinerNodes,
	gsos *block.GroupSharesOrSigns, mpks *block.Mpks, pn *PhaseNode) (
	*block.MagicBlock, error) {

	magicBlock := block.NewMagicBlock()
	magicBlock.Miners = node.NewPool(node.NodeTypeMiner)
	magicBlock.Sharders = node.NewPool(node.NodeTypeSharder)
	magicBlock.SetShareOrSigns(gsos)
	magicBlock.Mpks = mpks
	magicBlock.T = dkgMinersList.T
	magicBlock.K = dkgMinersList.K
	magicBlock.N = dkgMinersList.N
	for _, v := range dkgMinersList.SimpleNodes {
		n := &node.Node{}
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
		magicBlock.Miners.AddNode(n)
	}
	prevMagicBlock := balances.GetLastestFinalizedMagicBlock()

	for _, v := range sharderList.Nodes {
		n := &node.Node{}
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
		magicBlock.Sharders.AddNode(n)
	}
	magicBlock.MagicBlockNumber = prevMagicBlock.MagicBlock.MagicBlockNumber + 1
	magicBlock.PreviousMagicBlockHash = prevMagicBlock.MagicBlock.Hash
	magicBlock.StartingRound = pn.CurrentRound + PhaseRounds[Wait]
	magicBlock.Hash = magicBlock.GetHash()
	return magicBlock, nil
}

func (msc *MinerSmartContract) RestartDKG(pn *PhaseNode,
	balances cstate.StateContextI) {

	msc.mutexMinerMPK.Lock()
	defer msc.mutexMinerMPK.Unlock()
	mpks := block.NewMpks()
	_, err := balances.InsertTrieNode(MinersMPKKey, mpks)
	if err != nil {
		Logger.Error("failed to restart dkg", zap.Any("error", err))
	}
	Logger.Debug("create_mpks in RestartDKG", zap.Int64("DB version", int64(balances.GetState().GetVersion())))

	gsos := block.NewGroupSharesOrSigns()
	_, err = balances.InsertTrieNode(GroupShareOrSignsKey, gsos)
	if err != nil {
		Logger.Error("failed to restart dkg", zap.Any("error", err))
	}
	dkgMinersList := NewDKGMinerNodes()
	dkgMinersList.StartRound = pn.CurrentRound
	_, err = balances.InsertTrieNode(DKGMinersKey, dkgMinersList)
	if err != nil {
		Logger.Error("failed to restart dkg", zap.Any("error", err))
	}

	allMinersList := new(MinerNodes)
	_, err = balances.InsertTrieNode(ShardersKeepKey, allMinersList)
	if err != nil {
		Logger.Error("failed to restart dkg", zap.Any("error", err))
	}
	pn.Phase = Start
	pn.Restarts++
	pn.StartRound = pn.CurrentRound
}

func (msc *MinerSmartContract) SetMagicBlock(gn *GlobalNode,
	balances cstate.StateContextI) bool {

	magicBlockBytes, err := balances.GetTrieNode(MagicBlockKey)
	if err != nil {
		Logger.Error("could not get magic block from MPT", zap.Error(err))
		return false
	}
	magicBlock := block.NewMagicBlock()
	err = magicBlock.Decode(magicBlockBytes.Encode())
	if err != nil {
		Logger.Error("could not decode magic block from MPT", zap.Error(err))
		return false
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
