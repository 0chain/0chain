//go:build !integration_tests
// +build !integration_tests

package miner

import (
	"context"
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
		logging.Logger.Error("[mvc] publishShareOrSigns, failed to get miners mpks", zap.Error(err))
		return nil, err
	}
	if _, ok := mpks.Mpks[selfNodeKey]; !ok {
		logging.Logger.Error("[mvc] publishShareOrSigns, miner not part of mpks", zap.String("miner", selfNodeKey))
		// return nil, common.NewError("publish_sos", "miner not part of mpks")
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

	var dmn *minersc.DKGMinerNodes
	if dmn, err = mc.getDKGMiners(ctx, lfb, mb, active); err != nil {
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
	mb *block.MagicBlock, active bool) (tx *httpclientutil.Transaction,
	err error) {

	var dmn *minersc.DKGMinerNodes
	if dmn, err = mc.getDKGMiners(ctx, lfb, mb, active); err != nil {
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
	return
}

func afterSignShareRequestHandler(message *bls.DKGKeyShare, nodeID string) (messageResult *bls.DKGKeyShare, err error) {
	return message, nil
}
