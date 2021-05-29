// +build !integration_tests

package miner

import (
	"context"
	"net/http"
	"net/url"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/httpclientutil"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/threshold/bls"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/logging"
	"0chain.net/smartcontract/minersc"

	"go.uber.org/zap"
)

// The sendDKGShare sends the generated secShare to the given node.
func (mc *Chain) sendDKGShare(ctx context.Context, to string) (err error) {

	if !config.DevConfiguration.IsDkgEnabled {
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
		signatureScheme.SetPublicKey(n.PublicKey)

		var err error
		ok, err = signatureScheme.Verify(share.Sign, share.Message)
		if !ok || err != nil {
			logging.Logger.Error("invalid share or sign",
				zap.Error(err), zap.Any("minersc/dkg.gosign_status", ok),
				zap.Any("message", share.Message), zap.Any("sign", share.Sign))
			return
		}
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
	return
}

//
// P U B L I S H
//

func (mc *Chain) PublishShareOrSigns(ctx context.Context, lfb *block.Block,
	mb *block.MagicBlock, active bool) (tx *httpclientutil.Transaction,
	err error) {

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
	if mpks, err = mc.getMinersMpks(ctx, lfb, mb, active); err != nil {
		return nil, err
	}
	if _, ok := mpks.Mpks[selfNodeKey]; !ok {
		return // (nil, nil)
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

	var dmn *minersc.DKGMinerNodes
	if dmn, err = mc.getDKGMiners(ctx, lfb, mb, active); err != nil {
		return nil, err
	}
	if len(dmn.SimpleNodes) == 0 {
		return nil, common.NewError("publish_sos", "no miners in DKG")
	}

	var publicKeys = make(map[string]string)
	for _, n := range dmn.SimpleNodes {
		publicKeys[n.ID] = n.PublicKey
	}

	var _, ok = sos.Validate(mpks, publicKeys,
		chain.GetServerChain().GetSignatureScheme())
	if !ok {
		logging.Logger.Error("failed to verify share or signs", zap.Any("mpks", mpks))
	}

	tx = httpclientutil.NewTransactionEntity(selfNodeKey, mc.ID,
		selfNode.PublicKey)

	var data = &httpclientutil.SmartContractTxnData{}
	data.Name = scNamePublishShares
	data.InputArgs = sos.Clone()

	tx.ToClientID = minersc.ADDRESS

	var minerUrls []string
	for id := range dmn.SimpleNodes {
		var nodeSend = node.GetNode(id)
		if nodeSend == nil {
			logging.Logger.Warn("failed to get node", zap.Any("id", id))
			continue
		}
		minerUrls = append(minerUrls, nodeSend.GetN2NURLBase())
	}
	err = httpclientutil.SendSmartContractTxn(tx, minersc.ADDRESS, 0, 0, data,
		minerUrls)
	return
}

//
// C O N T R I B U T E
//

func (mc *Chain) ContributeMpk(ctx context.Context, lfb *block.Block,
	mb *block.MagicBlock, active bool) (tx *httpclientutil.Transaction,
	err error) {

	var dmn *minersc.DKGMinerNodes
	if dmn, err = mc.getDKGMiners(ctx, lfb, mb, active); err != nil {
		logging.Logger.Error("can't contribute", zap.Any("error", err))
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

	tx = httpclientutil.NewTransactionEntity(selfNodeKey, mc.ID,
		selfNode.PublicKey)
	tx.ToClientID = minersc.ADDRESS

	err = httpclientutil.SendSmartContractTxn(tx, minersc.ADDRESS, 0, 0, data,
		mb.Miners.N2NURLs())
	return
}

func SignShareRequestHandler(ctx context.Context, r *http.Request) (
	resp interface{}, err error) {

	var (
		nodeID   = r.Header.Get(node.HeaderNodeID)
		secShare = r.FormValue("secret_share")
		mc       = GetMinerChain()
	)

	mc.viewChangeProcess.Lock()
	defer mc.viewChangeProcess.Unlock()

	if !mc.viewChangeProcess.isDKGSet() {
		return nil, common.NewError("sign_share", "DKG is not set")
	}

	var (
		mpks        = mc.viewChangeProcess.mpks.GetMpks()
		lmpks, dkgt = len(mpks), mc.viewChangeProcess.viewChangeDKG.T
	)
	if lmpks < dkgt {
		return nil, common.NewErrorf("sign_share", "don't have enough mpks"+
			" yet, l mpks (%d) < dkg t (%d)", lmpks, dkgt)
	}

	var (
		message = datastore.GetEntityMetadata("dkg_share").
			Instance().(*bls.DKGKeyShare)

		share bls.Key
	)

	if err = share.SetHexString(secShare); err != nil {
		logging.Logger.Error("failed to set hex string", zap.Any("error", err))
		return nil, common.NewErrorf("sign_share",
			"setting hex string: %v", err)
	}

	var (
		mpk       = bls.ConvertStringToMpk(mpks[nodeID].Mpk)
		mpkString []string
	)
	for _, pk := range mpk {
		mpkString = append(mpkString, pk.GetHexString())
	}

	if !mc.viewChangeProcess.viewChangeDKG.ValidateShare(mpk, share) {
		logging.Logger.Error("failed to verify dkg share", zap.Any("share", secShare),
			zap.Any("node_id", nodeID))
		return nil, common.NewError("sign_share", "failed to verify DKG share")
	}

	err = mc.viewChangeProcess.viewChangeDKG.AddSecretShare(
		bls.ComputeIDdkg(nodeID), secShare, false)
	if err != nil {
		return nil, common.NewErrorf("sign_share",
			"adding secret share: %v", err)
	}

	message.Message = encryption.Hash(secShare)
	message.Sign, err = node.Self.Sign(message.Message)
	if err != nil {
		logging.Logger.Error("failed to sign DKG share message", zap.Any("error", err))
		return nil, common.NewErrorf("sign_share",
			"signing DKG share message: %v", err)
	}

	return message, nil
}
