package miner

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/httpclientutil"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/threshold/bls"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/ememorystore"
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
	scNameAddMiner      = "add_miner"
	scNameContributeMpk = "contributeMpk"
	scNamePublishShares = "shareSignsOrShares"
)

const (
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

	mb := mc.GetCurrentMagicBlock()
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
	for key := range dkgMiners.SimpleNodes {
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
		mb := mc.GetCurrentMagicBlock()
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
		mb := mc.GetCurrentMagicBlock()
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
		mb := mc.GetCurrentMagicBlock()
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
		mc.UpdateNodesFromMagicBlock(magicBlock)
		viewChangeMutex.Lock()
		mc.clearViewChange()
		viewChangeMutex.Unlock()
		return nil, nil
	}

	if !mc.isDKGSet() {
		// mc.setDKGFromCurrentMagicBlock()
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
	// set T and N from the magic block
	mc.viewChangeDKG.T = magicBlock.T
	mc.viewChangeDKG.N = magicBlock.N
	summary := mc.viewChangeDKG.GetDKGSummary()
	ctx := common.GetRootContext()
	if err := StoreDKGSummary(ctx, summary); err != nil {
		Logger.DPanic(err.Error())
	}
	if err := mc.SetDKG(mc.viewChangeDKG, magicBlock.StartingRound); err != nil {
		Logger.Error("failed to set dkg", zap.Error(err))
	}
	mbData := block.NewMagicBlockData(magicBlock)
	if err := StoreMagicBlockDataLatest(ctx, mbData); err != nil {
		Logger.DPanic(err.Error())
	}
	mc.clearViewChange()
	mc.SetViewChangeMagicBlock(magicBlock)
	mc.SetNextViewChange(magicBlock.StartingRound)
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

func (mc *Chain) GetNextViewChange() int64 {
	return atomic.LoadInt64(&mc.nextViewChange)
}

func (mc *Chain) SetNextViewChange(value int64) {
	atomic.StoreInt64(&mc.nextViewChange, value)
}

func StoreMagicBlockDataLatest(ctx context.Context, data *block.MagicBlockData) error {
	if err := StoreMagicBlockData(ctx, data); err != nil {
		return err
	}
	// latest
	dataLatest := block.NewMagicBlockData(data.MagicBlock)
	dataLatest.ID = "latest"
	if err := StoreMagicBlockData(ctx, dataLatest); err != nil {
		return err
	}
	return nil
}

func StoreMagicBlockData(ctx context.Context, data *block.MagicBlockData) error {
	magicBlockMetadata := data.GetEntityMetadata()
	dctx := ememorystore.WithEntityConnection(ctx, magicBlockMetadata)
	defer ememorystore.Close(dctx)
	if err := data.Write(dctx); err != nil {
		return err
	}
	connection := ememorystore.GetEntityCon(dctx, magicBlockMetadata)
	if err := connection.Commit(); err != nil {
		return err
	}
	return nil
}

func GetMagicBlockDataFromStore(ctx context.Context, id string) (*block.MagicBlockData, error) {
	magicBlockData := datastore.GetEntity("magicblockdata").(*block.MagicBlockData)
	magicBlockData.ID = id
	magicBlockDataMetadata := magicBlockData.GetEntityMetadata()
	dctx := ememorystore.WithEntityConnection(ctx, magicBlockDataMetadata)
	defer ememorystore.Close(dctx)
	err := magicBlockData.Read(dctx, magicBlockData.GetKey())
	if err != nil {
		return nil, err
	}
	return magicBlockData, nil
}
