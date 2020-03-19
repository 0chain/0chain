package miner

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"sync"
	"time"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/httpclientutil"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/threshold/bls"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	. "0chain.net/core/logging"
	"0chain.net/core/util"
	"0chain.net/smartcontract/minersc"

	"go.uber.org/zap"
)

var (
	scFunctions            = make(map[int]SmartContractFunctions)
	currentPhase           int
	shareOrSigns           *block.ShareOrSigns
	txnConfirmationChannel = make(chan txnConfirmation)
	viewChangeMutex        = sync.Mutex{}
)

const (
	scNameAddMiner         = "add_miner"
	scNameContributeMpk    = "contributeMpk"
	scNamePublishShares    = "shareSignsOrShares"
	scRestAPIGetPhase      = "/getPhase"
	scRestAPIGetDKGMiners  = "/getDkgList"
	scRestAPIGetMinersMPKS = "/getMpksList"
	scRestAPIGetMagicBlock = "/getMagicBlock"
	scRestAPIGetMinerList  = "/getMinerList"
)

type SmartContractFunctions func() (*httpclientutil.Transaction, error)

type txnConfirmation struct {
	TxnHash   string
	Confirmed bool
}

func (mc *Chain) initSetup() {
	scFunctions[minersc.Start] = mc.DKGProcessStart
	scFunctions[minersc.Contribute] = mc.ContributeMpk
	scFunctions[minersc.Share] = mc.SendSijs
	scFunctions[minersc.Publish] = mc.PublishShareOrSigns
	scFunctions[minersc.Wait] = mc.Wait
	mc.mpks = block.NewMpks()
	shareOrSigns = block.NewShareOrSigns()
	shareOrSigns.ID = node.Self.Underlying().GetKey()
	currentPhase = minersc.Unknown
}

func (mc *Chain) DKGProcess(ctx context.Context) {
	mc.initSetup()
	const timeoutPhase = 5
	const thresholdCheckPhase = 2
	var timer = time.NewTimer(time.Duration(time.Second))
	counterCheckPhaseSharder := 0
	var oldPhaseRound int
	var oldCurrentRound int64
	for true {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			checkOnSharder := counterCheckPhaseSharder >= thresholdCheckPhase
			pn, err := mc.GetPhase(checkOnSharder)
			if err == nil {
				if !checkOnSharder && pn.Phase == oldPhaseRound && pn.CurrentRound == oldCurrentRound {
					counterCheckPhaseSharder++
				} else {
					counterCheckPhaseSharder = 0
					oldPhaseRound = pn.Phase
					oldCurrentRound = pn.CurrentRound
				}
			}

			Logger.Debug("dkg process trying", zap.Any("next_phase", pn), zap.Any("phase", currentPhase), zap.Any("sc funcs", len(scFunctions)))
			if err == nil && pn != nil && pn.Phase != currentPhase {
				Logger.Info("dkg process start", zap.Any("next_phase", pn), zap.Any("phase", currentPhase), zap.Any("sc funcs", len(scFunctions)))
				if scFunc, ok := scFunctions[pn.Phase]; ok {
					txn, err := scFunc()
					if err != nil {
						Logger.Error("smart contract function failed", zap.Any("error", err), zap.Any("next_phase", pn))
					} else {
						Logger.Debug("dkg process move phase", zap.Any("next_phase", pn), zap.Any("phase", currentPhase), zap.Any("txn", txn))
						if txn == nil || (txn != nil && mc.ConfirmTransaction(txn)) {
							oldPhase := currentPhase
							currentPhase = pn.Phase
							Logger.Debug("dkg process moved phase", zap.Any("old_phase", oldPhase), zap.Any("phase", currentPhase))

						}
					}
				}
			} else if err != nil {
				Logger.Error("dkg process", zap.Any("error", err), zap.Any("sc funcs", len(scFunctions)), zap.Any("phase", pn))
			}
			timer = time.NewTimer(timeoutPhase * time.Second)
		}
	}
}

func (mc *Chain) GetPhase(fromSharder bool) (*minersc.PhaseNode, error) {
	pn := &minersc.PhaseNode{}
	active := mc.ActiveInChain()
	if active {
		clientState := chain.CreateTxnMPT(mc.GetLatestFinalizedBlock().ClientState)
		node, err := clientState.GetNodeValue(util.Path(encryption.Hash(pn.GetKey())))
		if err != nil {
			return nil, err
		}
		if node == nil {
			return nil, common.NewError("key_not_found", "key was not found")
		}
		err = pn.Decode(node.Encode())
		if err != nil {
			return nil, err
		}
		if !fromSharder {
			return pn, nil
		}
	}

	mb := mc.GetMagicBlock()
	pnSharder := &minersc.PhaseNode{}
	var sharders = mb.Sharders.N2NURLs()
	err := httpclientutil.MakeSCRestAPICall(minersc.ADDRESS, scRestAPIGetPhase, nil, sharders, pnSharder, 1)
	if err != nil {
		return nil, err
	}
	if active && (pn.Phase != pnSharder.Phase || pn.StartRound != pnSharder.StartRound) {
		Logger.Warn("dkg_process -- phase in state does not match shader",
			zap.Any("phase_state", pn.Phase), zap.Any("phase_sharder", pnSharder.Phase),
			zap.Any("start_round_state", pn.StartRound),
			zap.Any("start_round_sharder", pnSharder.StartRound))
		return pnSharder, nil
	}
	if active {
		return pn, nil
	}
	return pnSharder, nil
}

func (mc *Chain) DKGProcessStart() (*httpclientutil.Transaction, error) {
	viewChangeMutex.Lock()
	defer viewChangeMutex.Unlock()
	mc.clearViewChange()
	return nil, nil
}

func (mc *Chain) ContributeMpk() (*httpclientutil.Transaction, error) {
	magicBlock := mc.GetMagicBlock()
	if magicBlock == nil {
		return nil, common.NewError("contribute_mpk", "magic block empty")
	}
	dmn, err := mc.GetDKGMiners()
	if err != nil {
		Logger.Error("can't contribute", zap.Any("error", err))
		return nil, err
	}
	selfNode := node.Self.Underlying()
	selfNodeKey := selfNode.GetKey()
	mpk := &block.MPK{ID: selfNodeKey}
	if !mc.isDKGSet() {
		if dmn.N == 0 {
			return nil, common.NewError("contribute_mpk", "failed to contribute mpk:dkg is not set yet")
		}
		vc := bls.MakeDKG(dmn.T, dmn.N, selfNodeKey)
		vc.ID = bls.ComputeIDdkg(selfNodeKey)
		vc.MagicBlockNumber = magicBlock.MagicBlockNumber + 1
		viewChangeMutex.Lock()
		mc.viewChangeDKG = vc
		viewChangeMutex.Unlock()
	}

	viewChangeMutex.Lock()
	defer viewChangeMutex.Unlock()
	for _, v := range mc.viewChangeDKG.Mpk {
		mpk.Mpk = append(mpk.Mpk, v.GetHexString())
	}

	scData := &httpclientutil.SmartContractTxnData{}
	scData.Name = scNameContributeMpk
	scData.InputArgs = mpk

	txn := httpclientutil.NewTransactionEntity(selfNodeKey, mc.ID, selfNode.PublicKey)
	txn.ToClientID = minersc.ADDRESS
	var minerUrls []string
	for _, node := range magicBlock.Miners.CopyNodes() {
		minerUrls = append(minerUrls, node.GetN2NURLBase())
	}
	err = httpclientutil.SendSmartContractTxn(txn, minersc.ADDRESS, 0, 0, scData, minerUrls)
	return txn, err
}

func (mc *Chain) CreateSijs() error {
	viewChangeMutex.Lock()
	defer viewChangeMutex.Unlock()

	mpks, err := mc.GetMinersMpks()
	if err != nil {
		Logger.Error("can't share", zap.Any("error", err))
		return err
	}
	dmn, err := mc.GetDKGMiners()
	if err != nil {
		Logger.Error("can't share", zap.Any("error", err))
		return err
	}
	for k := range mpks.Mpks {
		if node.GetNode(k) != nil {
			continue
		}
		v := dmn.SimpleNodes[k]
		n := node.Provider()
		n.ID = v.ID
		n.N2NHost = v.N2NHost
		n.Host = v.Host
		n.Port = v.Port
		n.PublicKey = v.PublicKey
		n.Description = v.ShortName
		n.Type = node.NodeTypeMiner
		n.Info.BuildTag = v.BuildTag
		n.SetStatus(node.NodeStatusActive)
		node.Setup(n)
		node.RegisterNode(n)
	}
	mc.mutexMpks.Lock()
	mc.mpks = mpks
	mc.mutexMpks.Unlock()

	foundSelf := false
	lfb := mc.GetLatestFinalizedBlock()

	for k := range mc.GetMpks() {
		id := bls.ComputeIDdkg(k)
		share, err := mc.viewChangeDKG.ComputeDKGKeyShare(id)
		if err != nil {
			Logger.Error("can't compute secret share", zap.Any("error", err))
			return err
		}
		if k == node.Self.Underlying().GetKey() {
			mc.viewChangeDKG.AddSecretShare(id, share.GetHexString(), false)
			foundSelf = true
		}
	}
	if !foundSelf {
		Logger.Error("failed to add secret key for self", zap.Any("lfb_round", lfb.Round))
	}
	return nil
}

func (mc *Chain) SendSijs() (*httpclientutil.Transaction, error) {
	dkgMiners, err := mc.GetDKGMiners()
	if err != nil {
		return nil, err
	}
	selfNodeKey := node.Self.Underlying().GetKey()
	if _, ok := dkgMiners.SimpleNodes[selfNodeKey]; !mc.isDKGSet() || !ok {
		Logger.Error("failed to send sijs", zap.Any("dkg_set", mc.isDKGSet()), zap.Any("ok", ok))
		return nil, nil
	}

	viewChangeMutex.Lock()
	needCreateSijs := len(mc.viewChangeDKG.Sij) < mc.viewChangeDKG.T
	viewChangeMutex.Unlock()

	if needCreateSijs {
		err := mc.CreateSijs()
		if err != nil {
			return nil, err
		}
	}

	var failedSend []string
	nodes := node.GetMinerNodesKeys()
	for _, key := range nodes {
		if key != selfNodeKey {
			_, ok := shareOrSigns.ShareOrSigns[key]
			if !ok {
				err := mc.SendDKGShare(node.GetNode(key))
				if err != nil {
					failedSend = append(failedSend, fmt.Sprintf("%s(%v);", key, err))
				}
			}
		}
	}
	if len(failedSend) > 0 {
		return nil, common.NewError("failed to send sijs", fmt.Sprintf("failed to send share to miners: %v", failedSend))
	}
	return nil, nil
}

func (mc *Chain) GetDKGMiners() (*minersc.DKGMinerNodes, error) {
	dmn := minersc.NewDKGMinerNodes()
	if mc.ActiveInChain() {
		lfb := mc.GetLatestFinalizedBlock()
		clientState := chain.CreateTxnMPT(lfb.ClientState)
		node, err := clientState.GetNodeValue(util.Path(encryption.Hash(minersc.DKGMinersKey)))
		if err != nil {
			return nil, err
		}
		if node == nil {
			return nil, common.NewError("key_not_found", "key was not found")
		}
		err = dmn.Decode(node.Encode())
		if err != nil {
			return nil, err
		}

	} else {
		mb := mc.GetMagicBlock()
		var sharders = mb.Sharders.N2NURLs()
		err := httpclientutil.MakeSCRestAPICall(minersc.ADDRESS, scRestAPIGetDKGMiners, nil, sharders, dmn, 1)
		if err != nil {
			return nil, err
		}
	}
	return dmn, nil
}

func (mc *Chain) GetMinersMpks() (*block.Mpks, error) {
	mpks := block.NewMpks()
	if mc.ActiveInChain() {
		lfb := mc.GetLatestFinalizedBlock()
		clientState := chain.CreateTxnMPT(lfb.ClientState)
		node, err := clientState.GetNodeValue(util.Path(encryption.Hash(minersc.MinersMPKKey)))
		if err != nil {
			return nil, err
		}
		if node == nil {
			return nil, common.NewError("key_not_found", "key was not found")
		}
		err = mpks.Decode(node.Encode())
		if err != nil {
			return nil, err
		}
	} else {
		mb := mc.GetMagicBlock()
		var sharders = mb.Sharders.N2NURLs()
		err := httpclientutil.MakeSCRestAPICall(minersc.ADDRESS, scRestAPIGetMinersMPKS, nil, sharders, mpks, 1)
		if err != nil {
			return nil, err
		}
	}
	return mpks, nil
}

func (mc *Chain) GetFinalizedMagicBlock() (*block.MagicBlock, error) {
	magicBlock := block.NewMagicBlock()
	if mc.ActiveInChain() {
		lfb := mc.GetLatestFinalizedBlock()
		clientState := chain.CreateTxnMPT(lfb.ClientState)
		node, err := clientState.GetNodeValue(util.Path(encryption.Hash(minersc.MagicBlockKey)))
		if err != nil {
			return nil, err
		}
		if node == nil {
			return nil, common.NewError("key_not_found", "key was not found")
		}
		err = magicBlock.Decode(node.Encode())
		if err != nil {
			return nil, err
		}
	} else {
		mbs := mc.GetLatestFinalizedMagicBlockFromSharder(common.GetRootContext())
		if len(mbs) == 0 {
			return nil, common.NewError("failed to get magic block", "no magic block available")
		}
		if len(mbs) > 1 {
			sort.Slice(mbs, func(i, j int) bool {
				return mbs[i].StartingRound < mbs[j].StartingRound
			})
		}
		block := mbs[0]
		mc.SetLatestFinalizedMagicBlock(block)
		magicBlock = block.MagicBlock
	}
	return magicBlock, nil
}

func (mc *Chain) GetMagicBlockFromSC() (*block.MagicBlock, error) {
	magicBlock := block.NewMagicBlock()
	if mc.ActiveInChain() {
		lfb := mc.GetLatestFinalizedBlock()
		clientState := chain.CreateTxnMPT(lfb.ClientState)
		node, err := clientState.GetNodeValue(util.Path(encryption.Hash(minersc.MagicBlockKey)))
		if err != nil {
			return nil, err
		}
		if node == nil {
			return nil, common.NewError("key_not_found", "key was not found")
		}
		err = magicBlock.Decode(node.Encode())
		if err != nil {
			return nil, err
		}
	} else {
		mb := mc.GetMagicBlock()
		var (
			sharders = mb.Sharders.N2NURLs()
			err      error
		)
		err = httpclientutil.MakeSCRestAPICall(minersc.ADDRESS, scRestAPIGetMagicBlock, nil, sharders, magicBlock, 1)
		if err != nil {
			return nil, err
		}
	}
	return magicBlock, nil
}

func SignShareRequestHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	nodeID := r.Header.Get(node.HeaderNodeID)
	secShare := r.FormValue("secret_share")
	mc := GetMinerChain()
	if !mc.isDKGSet() {
		return nil, common.NewError("failed to sign share", "dkg not set")
	}

	viewChangeMutex.Lock()
	defer viewChangeMutex.Unlock()

	mpks := mc.GetMpks()
	if len(mpks) < mc.viewChangeDKG.T {
		return nil, common.NewError("failed to sign", "don't have mpks yet")
	}
	message := datastore.GetEntityMetadata("dkg_share").Instance().(*bls.DKGKeyShare)
	var share bls.Key
	if err := share.SetHexString(secShare); err != nil {
		Logger.Error("failed to set hex string", zap.Any("error", err))
		return nil, err
	}

	mpk := bls.ConvertStringToMpk(mpks[nodeID].Mpk)
	var mpkString []string
	for _, pk := range mpk {
		mpkString = append(mpkString, pk.GetHexString())
	}

	if !mc.viewChangeDKG.ValidateShare(mpk, share) {
		Logger.Error("failed to verify dkg share", zap.Any("share", secShare), zap.Any("node_id", nodeID))
		return nil, common.NewError("failed to sign", "failed to verify dkg share")
	}
	err := mc.viewChangeDKG.AddSecretShare(bls.ComputeIDdkg(nodeID), secShare, false)
	if err != nil {
		return nil, err
	}

	message.Message = node.Self.Underlying().GetKey()
	message.Sign, err = node.Self.Sign(message.Message)
	if err != nil {
		Logger.Error("failed to sign dkg share message", zap.Any("error", err))
		return nil, err
	}
	return message, nil
}

/*SendDKGShare sends the generated secShare to the given node */
func (mc *Chain) SendDKGShare(n *node.Node) error {
	if !config.DevConfiguration.IsDkgEnabled {
		return common.NewError("failed to send dkg share", "dkg is not enabled")
	}
	if node.Self.Underlying().GetKey() == n.ID {
		return nil
	}
	params := &url.Values{}
	nodeID := bls.ComputeIDdkg(n.ID)

	viewChangeMutex.Lock()
	defer viewChangeMutex.Unlock()
	secShare := mc.viewChangeDKG.Sij[nodeID]

	params.Add("secret_share", secShare.GetHexString())
	shareOrSignSuccess := make(map[string]*bls.DKGKeyShare)
	handler := func(ctx context.Context, entity datastore.Entity) (interface{}, error) {
		share, ok := entity.(*bls.DKGKeyShare)
		if !ok {
			Logger.Error("invalid share or sign", zap.Any("entity", entity))
			return nil, nil
		}
		if share.Message != "" && share.Sign != "" {
			signatureScheme := chain.GetServerChain().GetSignatureScheme()
			signatureScheme.SetPublicKey(n.PublicKey)
			sigOK, err := signatureScheme.Verify(share.Sign, share.Message)
			if !sigOK || err != nil {
				Logger.Error("invalid share or sign",
					zap.Error(err), zap.Any("sign_status", sigOK),
					zap.Any("message", share.Message), zap.Any("sign", share.Sign))
				return nil, nil
			}
			shareOrSignSuccess[n.ID] = share
		}
		return nil, nil
	}
	ctx := common.GetRootContext()
	n.RequestEntityFromNode(ctx, DKGShareSender, params, handler)
	if len(shareOrSignSuccess) == 0 {
		return common.NewError("failed to send dkg", "miner returned error")
	}
	for id, share := range shareOrSignSuccess {
		shareOrSigns.ShareOrSigns[id] = share
	}
	return nil
}

func (mc *Chain) PublishShareOrSigns() (*httpclientutil.Transaction, error) {
	if !mc.isDKGSet() {
		Logger.Error("failed to publish share or signs", zap.Any("dkg_set", mc.isDKGSet()))
		return nil, nil
	}
	viewChangeMutex.Lock()
	defer viewChangeMutex.Unlock()

	selfNode := node.Self.Underlying()
	selfNodeKey := selfNode.GetKey()
	txn := httpclientutil.NewTransactionEntity(selfNodeKey, mc.ID, selfNode.PublicKey)

	mpks, err := mc.GetMinersMpks()
	if err != nil {
		return nil, err
	}
	if _, ok := mpks.Mpks[selfNodeKey]; !ok {
		return nil, nil
	}

	for k := range mpks.Mpks {
		if _, ok := shareOrSigns.ShareOrSigns[k]; !ok && k != selfNodeKey {
			share := mc.viewChangeDKG.Sij[bls.ComputeIDdkg(k)]
			shareOrSigns.ShareOrSigns[k] = &bls.DKGKeyShare{Share: share.GetHexString()}
		}
	}

	dkgMiners, err := mc.GetDKGMiners()
	if err != nil {
		return nil, err
	}
	if len(dkgMiners.SimpleNodes) == 0 {
		return nil, common.NewError("failed to publish share or signs", "no miners in DKG")
	}
	publicKeys := make(map[string]string)
	for _, node := range dkgMiners.SimpleNodes {
		publicKeys[node.ID] = node.PublicKey
	}
	_, ok := shareOrSigns.Validate(mpks, publicKeys, chain.GetServerChain().GetSignatureScheme())
	if !ok {
		Logger.Error("failed to verify share or signs", zap.Any("mpks", mpks))
	}
	scData := &httpclientutil.SmartContractTxnData{}
	scData.Name = scNamePublishShares
	scData.InputArgs = shareOrSigns.Clone()
	txn.ToClientID = minersc.ADDRESS

	mb := mc.GetMagicBlock()

	var minerUrls []string
	for _, node := range mb.Miners.CopyNodes() {
		minerUrls = append(minerUrls, node.GetN2NURLBase())
	}
	err = httpclientutil.SendSmartContractTxn(txn, minersc.ADDRESS, 0, 0, scData, minerUrls)
	return txn, err
}

func (mc *Chain) Wait() (result *httpclientutil.Transaction, err2 error) {
	magicBlock, err := mc.GetMagicBlockFromSC()
	if err != nil {
		return nil, err
	}
	if !magicBlock.Miners.HasNode(node.Self.Underlying().GetKey()) {
		err := mc.UpdateMagicBlock(magicBlock)
		if err != nil {
			Logger.DPanic(fmt.Sprintf("failed to update magic block: %v", err.Error()))
		}
		viewChangeMutex.Lock()
		mc.clearViewChange()
		viewChangeMutex.Unlock()
		return nil, nil
	}

	if !mc.isDKGSet() {
		return nil, errors.New("unexpected not isDKGSet")
	}

	viewChangeMutex.Lock()
	defer viewChangeMutex.Unlock()

	mpks := mc.GetMpks()
	for key, share := range magicBlock.GetShareOrSigns().GetShares() {
		if key == node.Self.Underlying().GetKey() {
			continue
		}
		myShare, ok := share.ShareOrSigns[node.Self.Underlying().GetKey()]
		if ok && myShare.Share != "" {
			var share bls.Key
			share.SetHexString(myShare.Share)
			if mc.viewChangeDKG.ValidateShare(bls.ConvertStringToMpk(mpks[key].Mpk), share) {
				err := mc.viewChangeDKG.AddSecretShare(bls.ComputeIDdkg(key), myShare.Share, true)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	var miners []string
	for key := range mc.GetMpks() {
		if _, ok := magicBlock.Mpks.Mpks[key]; !ok {
			miners = append(miners, key)
		}
	}
	mc.viewChangeDKG.DeleteFromSet(miners)

	mc.viewChangeDKG.AggregatePublicKeyShares(magicBlock.Mpks.GetMpkMap())
	mc.viewChangeDKG.AggregateSecretKeyShares()
	mc.viewChangeDKG.StartingRound = magicBlock.StartingRound
	mc.viewChangeDKG.MagicBlockNumber = magicBlock.MagicBlockNumber
	summary := mc.viewChangeDKG.GetDKGSummary()
	if err := StoreDKGSummary(common.GetRootContext(), summary); err != nil {
		Logger.DPanic(err.Error())
	}
	mc.clearViewChange()
	mc.SetViewChangeMagicBlock(magicBlock)
	mc.nextViewChange = magicBlock.StartingRound
	return nil, nil
}

func (mc *Chain) isDKGSet() bool {
	viewChangeMutex.Lock()
	defer viewChangeMutex.Unlock()
	return mc.viewChangeDKG != nil
}

func (mc *Chain) clearViewChange() {
	shareOrSigns = block.NewShareOrSigns()
	shareOrSigns.ID = node.Self.Underlying().GetKey()
	mc.mpks = block.NewMpks()
	mc.viewChangeDKG = nil
}
