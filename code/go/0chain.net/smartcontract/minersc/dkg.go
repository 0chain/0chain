package minersc

import (
	"errors"
	"reflect"
	"runtime"
	"sort"

	"0chain.net/chaincore/block"
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/util"

	. "0chain.net/core/logging"
	"go.uber.org/zap"
)

var moveFunctions = make(map[int]movePhaseFunctions)

/*
- Start      : moveToContribute
- Contribute : moveToShareOrPublish
- Share      : moveToShareOrPublish
- Publish    : moveToWait
- Wait       : moveToStart
*/

func (msc *MinerSmartContract) moveToContribute(balances cstate.StateContextI,
	pn *PhaseNode, gn *globalNode) (result bool) {

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

	return allMinersList != nil &&
		len(allMinersList.Nodes) >= dkgMinersList.K &&
		len(allShardersList.Nodes) >= gn.MinS
}

func (msc *MinerSmartContract) moveToShareOrPublish(
	balances cstate.StateContextI, pn *PhaseNode, gn *globalNode) (
	result bool) {

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

	dkgMinersList, err = msc.getMinersDKGList(balances)
	if err != nil {
		Logger.Error("failed to get miners DKG", zap.Any("error", err), zap.Any("phase", pn.Phase))
		return false
	}

	msc.mutexMinerMPK.Lock()
	defer msc.mutexMinerMPK.Unlock()

	var mpksBytes util.Serializable
	mpksBytes, err = balances.GetTrieNode(MinersMPKKey)
	if err != nil {
		Logger.Error("failed to get node MinersMPKKey", zap.Any("error", err), zap.Any("phase", pn.Phase))
		return false
	}

	if err = mpks.Decode(mpksBytes.Encode()); err != nil {
		Logger.Error("failed to decode node MinersMPKKey", zap.Any("error", err), zap.Any("phase", pn.Phase))
		return false
	}

	return mpks != nil && len(mpks.Mpks) >= dkgMinersList.K
}

func (msc *MinerSmartContract) moveToWait(balances cstate.StateContextI, pn *PhaseNode, gn *globalNode) (result bool) {

	var err error
	var dkgMinersList *DKGMinerNodes
	gsos := block.NewGroupSharesOrSigns()

	dkgMinersList, err = msc.getMinersDKGList(balances)
	if err != nil {
		Logger.Error("failed to get miners DKG", zap.Any("error", err), zap.Any("phase", pn.Phase))
		return false
	}

	var groupBytes util.Serializable
	groupBytes, err = balances.GetTrieNode(GroupShareOrSignsKey)
	if err != nil {
		Logger.Error("failed to get node GroupShareOrSignsKey", zap.Any("error", err), zap.Any("phase", pn.Phase))
		return false
	}

	if err = gsos.Decode(groupBytes.Encode()); err != nil {
		Logger.Error("failed to decode node GroupShareOrSignsKey", zap.Any("error", err), zap.Any("phase", pn.Phase))
		return false
	}
	return len(gsos.Shares) >= dkgMinersList.K
}

func (msc *MinerSmartContract) moveToStart(balances cstate.StateContextI, pn *PhaseNode, gn *globalNode) bool {
	return true
}

func (msc *MinerSmartContract) getPhaseNode(statectx cstate.StateContextI) (*PhaseNode, error) {
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

func (msc *MinerSmartContract) setPhaseNode(balances cstate.StateContextI, pn *PhaseNode, gn *globalNode) error {
	if pn.CurrentRound-pn.StartRound >= PhaseRounds[pn.Phase] {
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
				if len(PhaseRounds)-1 > pn.Phase {
					pn.Phase++
				} else {
					pn.Phase = 0
					pn.Restarts = 0
				}
				pn.StartRound = pn.CurrentRound
				if msc.callbackPhase != nil {
					msc.callbackPhase(pn.Phase)
				}
			}
		} else {
			Logger.Warn("failed to move phase",
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

func (msc *MinerSmartContract) createDKGMinersForContribute(balances cstate.StateContextI, gn *globalNode) error {

	allminerslist, err := msc.GetMinersList(balances)
	if err != nil {
		Logger.Error("createDKGMinersForContribute -- failed to get miner list", zap.Any("error", err))
		return err
	}

	if len(allminerslist.Nodes) < gn.MinN {
		return common.NewError("failed to create dkg miners", "too few miners for dkg")
	}

	dkgMiners := NewDKGMinerNodes()
	dkgMiners.calculateTKN(gn, len(allminerslist.Nodes))
	for _, node := range allminerslist.Nodes {
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

func (msc *MinerSmartContract) widdleDKGMinersForShare(balances cstate.StateContextI, gn *globalNode) error {

	dkgMiners, err := msc.getMinersDKGList(balances)
	if err != nil {
		Logger.Error("widdle dkg miners -- failed to get dkgMiners", zap.Any("error", err))
		return err
	}

	msc.mutexMinerMPK.Lock()
	defer msc.mutexMinerMPK.Unlock()

	mpks := block.NewMpks()
	mpksBytes, err := balances.GetTrieNode(MinersMPKKey)
	if err != nil {
		Logger.Error("widdle dkg miners -- failed to get miners mpks", zap.Any("error", err))
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

	if err = dkgMiners.recalculateTKN(false); err != nil {
		Logger.Error("widdle dkg miners", zap.Error(err))
		return err
	}

	_, err = balances.InsertTrieNode(DKGMinersKey, dkgMiners)
	if err != nil {
		Logger.Error("widdle dkg miners -- failed to insert dkg miners", zap.Any("error", err))
		return err
	}
	return nil
}

func (msc *MinerSmartContract) reduceShardersList(keep, all *MinerNodes,
	gn *globalNode) (list []*MinerNode, err error) {

	list = make([]*MinerNode, 0, len(keep.Nodes))
	for _, ksh := range keep.Nodes {
		var ash = all.FindNodeById(ksh.ID)
		if ash == nil {
			return nil, common.NewErrorf("invalid state", "a sharder exists in"+
				" keep list doesn't exists in all sharders list: %s", ksh.ID)
		}
		list = append(list, ash)
	}

	if len(list) <= gn.MaxS {
		return // doesn't need to sort
	}

	// get max staked
	sort.Slice(list, func(i, j int) bool {
		return list[i].TotalStaked > list[j].TotalStaked
	})

	list = list[:gn.MaxS]
	return
}

func (msc *MinerSmartContract) createMagicBlockForWait(balances cstate.StateContextI, gn *globalNode) error {

	println("(MINER SC) CREATE MB FOR WAIT")

	pn, err := msc.getPhaseNode(balances)
	if err != nil {
		println("(MINER SC) (E) no phase node")
		return err
	}
	dkgMinersList, err := msc.getMinersDKGList(balances)
	if err != nil {
		println("(MINER SC) (E) no DKG miners list")
		return err
	}
	gsos := block.NewGroupSharesOrSigns()
	groupBytes, err := balances.GetTrieNode(GroupShareOrSignsKey)
	if err != nil {
		println("(MINER SC) (E) no group share or signs")
		return err
	}
	if err = gsos.Decode(groupBytes.Encode()); err != nil {
		println("(MINER SC) (E) decoding group share or signs")
		return err
	}

	msc.mutexMinerMPK.Lock()
	defer msc.mutexMinerMPK.Unlock()

	mpks := block.NewMpks()
	mpksBytes, err := balances.GetTrieNode(MinersMPKKey)
	if err != nil {
		println("(MINER SC) (E) can't get miners MPKS")
		return common.NewErrorf("create_magic_block_failed", "error with miner's mpk: %v", err)
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
			gn)
		if err != nil {
			return err
		}
	}

	if err = dkgMinersList.recalculateTKN(true); err != nil {
		Logger.Error("create magic block for wait", zap.Error(err))
		return err
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
	dkgMinersList = NewDKGMinerNodes()
	_, err = balances.InsertTrieNode(DKGMinersKey, dkgMinersList)
	if err != nil {
		return err
	}
	allMinersList := new(MinerNodes)
	_, err = balances.InsertTrieNode(ShardersKeepKey, allMinersList)
	if err != nil {
		return err
	}
	return nil
}

func (msc *MinerSmartContract) contributeMpk(t *transaction.Transaction,
	inputData []byte, gn *globalNode, balances cstate.StateContextI) (
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

	if mpksBytes, _ = balances.GetTrieNode(MinersMPKKey); mpksBytes != nil {
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
		println("(MIENR SC) CMPK: already have MPK for", mpk.ID, ", sender:", t.ClientID, "mpks l", len(mpks.Mpks))
		return "", common.NewError("contribute_mpk_failed",
			"already have mpk for miner")
	}

	mpks.Mpks[mpk.ID] = mpk
	_, err = balances.InsertTrieNode(MinersMPKKey, mpks)
	if err != nil {
		return "", common.NewErrorf("contribute_mpk_failed",
			"saving MPK key: %v", err)
	}

	return string(mpk.Encode()), nil
}

func (msc *MinerSmartContract) shareSignsOrShares(t *transaction.Transaction,
	inputData []byte, gn *globalNode, balances cstate.StateContextI) (
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
		println("(MINER SC) SOS validation failed", sos.ID, "sos l", len(sos.ShareOrSigns))
		for k, v := range sos.ShareOrSigns {
			println(" - (inspect sos)", k, "M-S-Si", v.Message, v.Share, v.Sign)
		}
		return "", common.NewError("share_signs_or_shares",
			"share or signs failed validation")
	}

	for _, share := range shares {
		dmn.RevealedShares[share]++
	}

	sos.ID = t.ClientID
	gsos.Shares[t.ClientID] = sos

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

func (msc *MinerSmartContract) getMinersDKGList(statectx cstate.StateContextI) (*DKGMinerNodes, error) {
	allMinersList := NewDKGMinerNodes()
	allMinersBytes, err := statectx.GetTrieNode(DKGMinersKey)
	if err != nil && err != util.ErrValueNotPresent {
		return nil, errors.New("get_miners_dkg_list_failed - " +
			"failed to retrieve existing miners list: " + err.Error())
	}
	if allMinersBytes == nil {
		return allMinersList, nil
	}
	allMinersList.Decode(allMinersBytes.Encode())
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

func (msc *MinerSmartContract) RestartDKG(pn *PhaseNode, balances cstate.StateContextI) {

	msc.mutexMinerMPK.Lock()
	defer msc.mutexMinerMPK.Unlock()
	mpks := block.NewMpks()
	_, err := balances.InsertTrieNode(MinersMPKKey, mpks)
	if err != nil {
		Logger.Error("failed to restart dkg", zap.Any("error", err))
	}
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

func (msc *MinerSmartContract) SetMagicBlock(balances cstate.StateContextI) bool {

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

	balances.SetMagicBlock(magicBlock)
	return true
}

func getFunctionName(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}
