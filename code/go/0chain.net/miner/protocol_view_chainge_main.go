//go:build !integration_tests
// +build !integration_tests

package miner

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/httpclientutil"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/threshold/bls"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/smartcontract/minersc"
	"github.com/0chain/common/core/logging"

	"go.uber.org/zap"
)

// The sendDKGShare sends the generated secShare to the given node.
func (mc *Chain) sendDKGShare(ctx context.Context, to string) (err error) {

	if !mc.ChainConfig.IsDkgEnabled() {
		return common.NewError("send_dkg_share", "dkg is not enabled")
	}

	var n = node.GetNode(to)

	if n == nil {
		return common.NewErrorf("send_dkg_share", "node %q not found", to)
	}

	if node.Self.Underlying().GetKey() == n.ID {
		return // don't send to itself
	}

	var (
		params = &url.Values{}
		nodeID = bls.ComputeIDdkg(n.ID)
	)

	shareOrSignSuccess := make(map[string]*bls.DKGKeyShare)
	secShare, ok := mc.getNodeSij(nodeID)
	if !ok {
		return common.NewErrorf("send_dkg_share", "could not found sec share of node id: %s", to)
	}

	params.Add("secret_share", secShare.GetHexString())

	var handler = func(ctx context.Context, entity datastore.Entity) (
		_ interface{}, _ error) {

		var share, ok = entity.(*bls.DKGKeyShare)
		if !ok {
			logging.Logger.Error("invalid share or sign", zap.Any("entity", entity))
			return
		}

		if share.Message == "" || share.Sign == "" {
			return // do nothing
		}

		var signatureScheme = chain.GetServerChain().GetSignatureScheme()
		if err := signatureScheme.SetPublicKey(n.PublicKey); err != nil {
			return nil, err
		}

		var err error
		ok, err = signatureScheme.Verify(share.Sign, share.Message)
		if !ok || err != nil {
			logging.Logger.Error("invalid share or sign",
				zap.Error(err), zap.Bool("minersc/dkg.gosign_status", ok),
				zap.String("message", share.Message), zap.String("sign", share.Sign))
			return
		}

		// share.ID = nodeID.GetHexString()
		// share.Share = secShare.GetHexString()
		shareOrSignSuccess[n.ID] = share

		return
	}

	if !n.RequestEntityFromNode(ctx, DKGShareSender, params, handler) {
		return common.NewError("send_dkg_share", "send message failed")
	}

	if len(shareOrSignSuccess) == 0 {
		return common.NewError("send_dkg_share", "miner returned error")
	}

	mc.setSecretShares(shareOrSignSuccess)
	logging.Logger.Debug("[mvc] sed dkg share", zap.Any("shareOrSignSuccess", shareOrSignSuccess))
	return
}

//
// P U B L I S H
//

func (mc *Chain) PublishShareOrSigns(ctx context.Context, lfb *block.Block,
	mb *block.MagicBlock) (tx *httpclientutil.Transaction,
	err error) {

	if mc.isSyncingBlocks() {
		logging.Logger.Debug("[mvc] sendsijs, block is syncing")
		return nil, nil
	}

	mc.viewChangeProcess.Lock()
	defer mc.viewChangeProcess.Unlock()

	if !mc.viewChangeProcess.isDKGSet() {
		return nil, common.NewError("publish_sos", "DKG is not set")
	}

	var (
		selfNode    = node.Self.Underlying()
		selfNodeKey = selfNode.GetKey()
	)

	var mpks *block.Mpks
	if mpks, err = mc.getMinersMpks(ctx, lfb, mb); err != nil {
		logging.Logger.Error("[mvc] publishShareOrSigns, failed to get miners mpks", zap.Error(err))
		return nil, err
	}
	if _, ok := mpks.Mpks[selfNodeKey]; !ok {
		logging.Logger.Error("[mvc] publishShareOrSigns, miner not part of mpks", zap.String("miner", selfNodeKey))
		return nil, nil
	}

	var sos = mc.viewChangeProcess.shareOrSigns // local reference

	for k := range mpks.Mpks {
		if k == selfNodeKey {
			continue
		}

		if _, ok := sos.ShareOrSigns[k]; !ok {
			share := mc.viewChangeDKG.GetDKGKeyShare(bls.ComputeIDdkg(k))
			if share != nil {
				sos.ShareOrSigns[k] = share
			}
		}
	}

	logging.Logger.Debug("[mvc] create sos",
		zap.Any("sos", sos),
		zap.Any("mpks", mpks.Mpks))

	var dmn *minersc.DKGMinerNodes
	if dmn, err = mc.getDKGMiners(ctx, lfb, mb); err != nil {
		logging.Logger.Error("[mvc] publishShareOrSigns, failed to get miners DKG", zap.Error(err))
		return nil, err
	}

	if len(dmn.SimpleNodes) == 0 {
		logging.Logger.Error("[mvc] publishShareOrSigns, no miners in DKG")
		return nil, common.NewError("publish_sos", "no miners in DKG")
	}

	var publicKeys = make(map[string]string)
	for _, n := range dmn.SimpleNodes {
		publicKeys[n.ID] = n.PublicKey
	}

	_, ok := sos.Validate(mpks, publicKeys, chain.GetServerChain().GetSignatureScheme())
	if !ok {
		logging.Logger.Error("[mvc] failed to verify share or signs", zap.Any("mpks", mpks))
		return nil, common.NewError("publish_sos", "failed to verify share or signs")
	}

	var data = &httpclientutil.SmartContractTxnData{}
	data.Name = scNamePublishShares
	data.InputArgs = sos.Clone()

	tx = httpclientutil.NewSmartContractTxn(selfNodeKey, mc.ID, selfNode.PublicKey, minersc.ADDRESS)
	var minerUrls []string
	for id := range dmn.SimpleNodes {
		var nodeSend = node.GetNode(id)
		if nodeSend == nil {
			logging.Logger.Warn("failed to get node", zap.String("id", id))
			continue
		}
		minerUrls = append(minerUrls, nodeSend.GetN2NURLBase())
	}

	err = mc.SendSmartContractTxn(tx, data, minerUrls, mb.Sharders.N2NURLs())
	return
}

//
// C O N T R I B U T E
//

func (mc *Chain) ContributeMpk(ctx context.Context, lfb *block.Block,
	mb *block.MagicBlock) (tx *httpclientutil.Transaction,
	err error) {

	if mc.isSyncingBlocks() {
		logging.Logger.Debug("[mvc] contribute_mpk, block is syncing")
		return nil, nil
	}

	var dmn *minersc.DKGMinerNodes
	if dmn, err = mc.getDKGMiners(ctx, lfb, mb); err != nil {
		logging.Logger.Error("can't contribute", zap.Error(err))
		return
	}

	var (
		selfNode    = node.Self.Underlying()
		selfNodeKey = selfNode.GetKey()
		mpk         = &block.MPK{ID: selfNodeKey}
	)

	mc.viewChangeProcess.Lock()
	defer mc.viewChangeProcess.Unlock()

	if !mc.viewChangeProcess.isDKGSet() {
		if dmn.N == 0 {
			return nil, common.NewError("contribute_mpk",
				"failed to contribute mpk: dkg is not set yet")
		}

		logging.Logger.Debug("[mvc] contribute_mpk set VC")
		var vc = bls.MakeDKG(dmn.T, dmn.N, selfNodeKey)
		vc.MagicBlockNumber = mb.MagicBlockNumber + 1
		mc.viewChangeProcess.viewChangeDKG = vc
	}

	logging.Logger.Debug("[vc] contribute_mpk", zap.Int("T", dmn.T),
		zap.Int("K", dmn.K), zap.Int("N", dmn.N),
		zap.Int64("mb_number",
			mc.viewChangeProcess.viewChangeDKG.MagicBlockNumber))

	for _, v := range mc.viewChangeProcess.viewChangeDKG.GetMPKs() {
		mpk.Mpk = append(mpk.Mpk, v.GetHexString())
	}

	logging.Logger.Debug("[vc] mpks len", zap.Int("mpks_len", len(mpk.Mpk)))

	var data = new(httpclientutil.SmartContractTxnData)
	data.Name = scNameContributeMpk
	data.InputArgs = mpk

	tx = httpclientutil.NewSmartContractTxn(selfNodeKey, mc.ID, selfNode.PublicKey, minersc.ADDRESS)
	err = mc.SendSmartContractTxn(tx, data, mb.Miners.N2NURLs(), mb.Sharders.N2NURLs())
	logging.Logger.Info("[vc] contribute mpk", zap.Any("tx", tx), zap.Any("err", err))
	return
}

func afterSignShareRequestHandler(message *bls.DKGKeyShare, nodeID string) (messageResult *bls.DKGKeyShare, err error) {
	return message, nil
}

//
//                               S H A R E
//

func (mc *Chain) SendSijs(ctx context.Context, lfb *block.Block,
	mb *block.MagicBlock) (tx *httpclientutil.Transaction,
	err error) {

	if mc.isSyncingBlocks() {
		logging.Logger.Debug("[mvc] sendsijs, block is syncing")
		return nil, nil
	}

	var (
		sendFail []string
		sendTo   []string
	)

	// it locks the mutex, but after, its free
	if sendTo, err = mc.sendSijsPrepare(ctx, lfb, mb); err != nil {
		return
	}

	for _, key := range sendTo {
		if err := mc.sendDKGShare(ctx, key); err != nil {
			sendFail = append(sendFail, fmt.Sprintf("%s(%v);", key, err))
		}
	}

	totalSentNum := len(sendTo)
	failNum := len(sendFail)
	successNum := totalSentNum - failNum
	if failNum > 0 && totalSentNum > mb.K && successNum < mb.K {
		logging.Logger.Error("[mvc] failed to send sijs",
			zap.Int("total sent num", totalSentNum),
			zap.Int("fail num", failNum),
			zap.Int("K", mb.K),
			zap.Strings("fail to miners", sendFail))
		return nil, errors.New("failed to send sijs")
	}
	logging.Logger.Debug("[mvc] send sijs success", zap.Int("total sent num", totalSentNum),
		zap.Int("success num", successNum))

	return // (nil, nil)
}

//
//                               W A I T
//

// Wait create 'wait' transaction to commit the miner
func (mc *Chain) Wait(ctx context.Context,
	lfb *block.Block, mb *block.MagicBlock) (tx *httpclientutil.Transaction, err error) {
	mc.viewChangeProcess.Lock()
	defer mc.viewChangeProcess.Unlock()

	if !mc.viewChangeProcess.isDKGSet() {
		return nil, common.NewError("vc_wait", "DKG is not set")
	}

	var magicBlock *block.MagicBlock
	if magicBlock, err = mc.GetMagicBlockFromSC(ctx, lfb, mb); err != nil {
		logging.Logger.Error("chain wait failed", zap.Error(err))
		return // error
	}

	if magicBlock.MagicBlockNumber != mb.MagicBlockNumber+1 {
		logging.Logger.Error("[mvc] dkg wait failed, not new magic block",
			zap.Int64("mb_num", magicBlock.MagicBlockNumber),
			zap.Int64("mb_sr", magicBlock.StartingRound),
			zap.String("mb_hash", magicBlock.Hash),
			zap.Int64("current_mb_num", mb.MagicBlockNumber))
		return nil, common.NewError("vc_wait", "not new magic block")
	}

	if !magicBlock.Miners.HasNode(node.Self.Underlying().GetKey()) {
		logging.Logger.Error("[mvc] vc wait failed, magic miners does not have self node")
		mc.viewChangeProcess.clearViewChange()
		return // node leaves BC, don't do anything here
	}

	if mc.isSyncingBlocks() {
		// Just store the magic block and return
		if err = StoreMagicBlock(ctx, magicBlock); err != nil {
			logging.Logger.Panic("failed to store magic block", zap.Error(err))
		}
		logging.Logger.Debug("[mvc] dkg wait, miner is synching, store magic block with no DKG summary",
			zap.Int64("mb_rs", magicBlock.StartingRound),
			zap.Int64("mb_rc", magicBlock.MagicBlockNumber),
			zap.String("mb_hash", magicBlock.Hash))
		return nil, nil
	}

	var (
		mpks        = mc.viewChangeProcess.mpks.GetMpks()
		vcdkg       = mc.viewChangeProcess.viewChangeDKG
		selfNodeKey = node.Self.Underlying().GetKey()
	)

	for key, share := range magicBlock.GetShareOrSigns().GetShares() {
		if key == selfNodeKey {
			continue // skip self
		}
		var myShare, ok = share.ShareOrSigns[selfNodeKey]
		if ok && myShare.Share != "" {
			var share bls.Key
			if err := share.SetHexString(myShare.Share); err != nil {
				return nil, err
			}
			lmpks, err := bls.ConvertStringToMpk(mpks[key].Mpk)
			if err != nil {
				return nil, err
			}

			var validShare = vcdkg.ValidateShare(lmpks, share)
			if !validShare {
				continue
			}
			err = vcdkg.AddSecretShare(bls.ComputeIDdkg(key), myShare.Share,
				true)
			if err != nil {
				return nil, common.NewErrorf("vc_wait",
					"adding secret share: %v", err)
			}
		}
	}

	var miners []string
	for key := range mc.viewChangeProcess.mpks.GetMpks() {
		if _, ok := magicBlock.Mpks.Mpks[key]; !ok {
			miners = append(miners, key)
		}
	}
	vcdkg.DeleteFromSet(miners)
	mpkMap, err := magicBlock.Mpks.GetMpkMap()
	if err != nil {
		return nil, err
	}
	if err := vcdkg.AggregatePublicKeyShares(mpkMap); err != nil {
		return nil, err
	}

	vcdkg.AggregateSecretKeyShares()
	vcdkg.StartingRound = magicBlock.StartingRound
	vcdkg.MagicBlockNumber = magicBlock.MagicBlockNumber
	// set T and N from the magic block
	vcdkg.T = magicBlock.T
	vcdkg.N = magicBlock.N

	// save DKG and MB
	dkgSum := vcdkg.GetDKGSummary()
	if err = StoreDKGSummary(ctx, dkgSum); err != nil {
		return nil, common.NewErrorf("vc_wait", "saving DKG summary: %v", err)
	}
	logging.Logger.Debug("[mvc] dkg wait: store dkg summary",
		zap.String("id", dkgSum.ID),
		zap.Int64("mb_num", magicBlock.MagicBlockNumber),
		zap.Int64("mb_sr", magicBlock.StartingRound),
		zap.String("mb_hash", magicBlock.Hash),
	)

	if err = StoreMagicBlock(ctx, magicBlock); err != nil {
		return nil, common.NewErrorf("vc_wait", "saving MB data: %v", err)
	}

	logging.Logger.Debug("[mvc] dkg wait: store mb",
		zap.Int64("mb_num", magicBlock.MagicBlockNumber),
		zap.Int64("mb_sr", magicBlock.StartingRound),
		zap.String("mb_hash", magicBlock.Hash))

	// create 'wait' transaction
	if tx, err = mc.waitTransaction(mb); err != nil {
		return nil, common.NewErrorf("vc_wait",
			"sending 'wait' transaction: %v", err)
	}

	return // the transaction
}

// isSyncingBlocks checks if the miner is syncing blocks
// NOTE: to differ it from the mc.IsBlockSyncing(), this function allows a
// certain numbers of blocks behind to be considered as synced, otherwise
// the VC process will be stuck as it always seeing it in syncing state.
// Especially when the miner is not in the MB.
func (mc *Chain) isSyncingBlocks() bool {
	var (
		allowBehind  = int64(20) // base on the observation of the max number of blocks was behind the LFB ticket
		lfb          = mc.GetLatestFinalizedBlock()
		lfbTkt       = mc.GetLatestLFBTicket(context.Background())
		aheadN       = int64(3)
		currentRound = mc.GetCurrentRound()
	)

	if currentRound+allowBehind < lfbTkt.Round ||
		lfb.Round+aheadN+allowBehind < lfbTkt.Round {
		return true
	}
	return false
}
