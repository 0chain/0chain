// +build integration_tests

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
	"0chain.net/smartcontract/minersc"

	. "0chain.net/core/logging"
	"go.uber.org/zap"

	crpc "0chain.net/conductor/conductrpc" // integration tests
	crpcutils "0chain.net/conductor/utils"
)

func revertString(s string) string {
	r := []rune(s)
	for i, j := 0, len(r)-1; i < len(r)/2; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return string(r)
}

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

	var (
		secShare           = mc.getNodeSij(nodeID)
		shareOrSignSuccess = make(map[string]*bls.DKGKeyShare)
	)

	var state = crpc.Client().State()
	switch nodeID := n.GetKey(); {
	case state.Shares.IsGood(state, nodeID):
		params.Add("secret_share", secShare.GetHexString())
	case state.Shares.IsBad(state, nodeID):
		params.Add("secret_share", revertString(secShare.GetHexString()))
	default:
		return common.NewError("failed to send DKG share", "skipped by tests")
	}

	var handler = func(ctx context.Context, entity datastore.Entity) (
		_ interface{}, _ error) {

		var share, ok = entity.(*bls.DKGKeyShare)
		if !ok {
			Logger.Error("invalid share or sign", zap.Any("entity", entity))
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
			Logger.Error("invalid share or sign",
				zap.Error(err), zap.Any("sign_status", ok),
				zap.Any("message", share.Message), zap.Any("sign", share.Sign))
			return
		}
		shareOrSignSuccess[n.ID] = share

		return
	}

	n.RequestEntityFromNode(ctx, DKGShareSender, params, handler)

	if len(shareOrSignSuccess) == 0 {
		return common.NewError("send_dkg_share", "miner returned error")
	}

	mc.setSecretShares(shareOrSignSuccess)
	return
}

func (mc *Chain) PublishShareOrSigns(_ context.Context, lfb *block.Block,
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
	if mpks, err = mc.getMinersMpks(lfb, mb, active); err != nil {
		return nil, err
	}
	if _, ok := mpks.Mpks[selfNodeKey]; !ok {
		return // (nil, nil)
	}

	var (
		state      = crpc.Client().State()
		isRevealed = state.IsRevealed

		sos = mc.viewChangeProcess.shareOrSigns
	)

	for k := range mpks.Mpks {
		if k == selfNodeKey {
			continue
		}
		var _, ok = sos.ShareOrSigns[k]
		if isRevealed || !ok {
			share := mc.viewChangeDKG.Sij[bls.ComputeIDdkg(k)]
			sos.ShareOrSigns[k] = &bls.DKGKeyShare{Share: share.GetHexString()}
		}
	}

	var dmn *minersc.DKGMinerNodes
	if dmn, err = mc.getDKGMiners(lfb, mb, active); err != nil {
		return nil, err
	}
	if len(dmn.SimpleNodes) == 0 {
		return nil, common.NewError("publish_sos", "no miners in DKG")
	}

	var publicKeys = make(map[string]string)
	for _, node := range dmn.SimpleNodes {
		publicKeys[node.ID] = node.PublicKey
	}

	var _, ok = sos.Validate(mpks, publicKeys,
		chain.GetServerChain().GetSignatureScheme())
	if !ok {
		Logger.Error("failed to verify share or signs", zap.Any("mpks", mpks))
	}

	var clone = sos.Clone()

	for id, share := range clone.ShareOrSigns {
		switch {
		case state.Publish.IsBad(state, id):
			share.Sign = revertString(share.Sign)
			share.Share = revertString(share.Share)
			share.Message = revertString(share.Message)
		case state.Publish.IsGood(state, id):
			// keep as is
		default:
			delete(clone.ShareOrSigns, id) // don't send at all
		}
	}

	tx = httpclientutil.NewTransactionEntity(selfNodeKey, mc.ID,
		selfNode.PublicKey)

	var data = &httpclientutil.SmartContractTxnData{}
	data.Name = scNamePublishShares
	data.InputArgs = clone

	tx.ToClientID = minersc.ADDRESS

	var minerUrls []string
	for id := range dmn.SimpleNodes {
		var nodeSend = node.GetNode(id)
		if nodeSend == nil {
			Logger.Warn("failed to get node", zap.Any("id", id))
			continue
		}
		minerUrls = append(minerUrls, nodeSend.GetN2NURLBase())
	}
	err = httpclientutil.SendSmartContractTxn(tx, minersc.ADDRESS, 0, 0, data,
		minerUrls)
	return
}

func getBadMPK(mpk *block.MPK) (bad *block.MPK) {
	bad = new(block.MPK)
	bad.ID = mpk.ID
	bad.Mpk = make([]string, 0, len(mpk.Mpk))
	for _, x := range mpk.Mpk {
		bad.Mpk = append(bad.Mpk, revertString(x))
	}
	return
}

func getBaseN2NURLs(nodes []*node.Node) (urls []string) {
	urls = make([]string, 0, len(nodes))
	for _, n := range nodes {
		urls = append(urls, n.GetN2NURLBase())
	}
	return
}

func (mc *Chain) sendMpkTransaction(selfNode *node.Node, mpk *block.MPK,
	urls []string) (tx *httpclientutil.Transaction, err error) {

	var scData = new(httpclientutil.SmartContractTxnData)
	scData.Name = scNameContributeMpk
	scData.InputArgs = mpk
	tx = httpclientutil.NewTransactionEntity(selfNode.GetKey(), mc.ID,
		selfNode.PublicKey)
	tx.ToClientID = minersc.ADDRESS
	err = httpclientutil.SendSmartContractTxn(tx, minersc.ADDRESS, 0, 0,
		scData, urls)
	return
}

func (mc *Chain) ContributeMpk(_ context.Context, lfb *block.Block,
	mb *block.MagicBlock, active bool) (tx *httpclientutil.Transaction,
	err error) {

	var dmn *minersc.DKGMinerNodes
	if dmn, err = mc.getDKGMiners(lfb, mb, active); err != nil {
		Logger.Error("can't contribute", zap.Any("error", err))
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

	for _, v := range mc.viewChangeProcess.viewChangeDKG.Mpk {
		mpk.Mpk = append(mpk.Mpk, v.GetHexString())
	}

	var (
		state     = crpc.Client().State()
		good, bad = crpcutils.Split(state, state.MPK,
			mb.Miners.Nodes)
		goodurls, badurls = getBaseN2NURLs(good), getBaseN2NURLs(bad)
		badMPK            = getBadMPK(mpk)
	)

	// send bad MPK first
	if len(bad) > 0 {
		tx, err = mc.sendMpkTransaction(selfNode, badMPK, badurls)
		if err != nil {
			return
		}
	}

	// send good MPK second
	if len(good) > 0 {
		tx, err = mc.sendMpkTransaction(selfNode, mpk, goodurls)
	}

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

	var mpks = mc.viewChangeProcess.mpks.GetMpks()
	if len(mpks) < mc.viewChangeProcess.viewChangeDKG.T {
		return nil, common.NewError("sign_share", "don't have enough mpks yet")
	}

	var (
		message = datastore.GetEntityMetadata("dkg_share").
			Instance().(*bls.DKGKeyShare)

		share bls.Key
	)

	if err = share.SetHexString(secShare); err != nil {
		Logger.Error("failed to set hex string", zap.Any("error", err))
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
		Logger.Error("failed to verify dkg share", zap.Any("share", secShare),
			zap.Any("node_id", nodeID))
		return nil, common.NewError("sign_share", "failed to verify DKG share")
	}

	err = mc.viewChangeProcess.viewChangeDKG.AddSecretShare(
		bls.ComputeIDdkg(nodeID), secShare, false)
	if err != nil {
		return nil, common.NewErrorf("sign_share",
			"adding secret share: %v", err)
	}

	message.Message = node.Self.Underlying().GetKey()
	message.Sign, err = node.Self.Sign(message.Message)
	if err != nil {
		Logger.Error("failed to sign DKG share message", zap.Any("error", err))
		return nil, common.NewErrorf("sign_share",
			"signing DKG share message: %v", err)
	}

	var state = crpc.Client().State()

	switch {
	case state.Signatures.IsBad(state, nodeID):
		message.Sign = revertString(message.Sign)
	default:
		return nil, common.NewError("integration_tests", "send_no_signatures")
	case state.Signatures.IsGood(state, nodeID):
		// as usual
	}

	return message, nil
}
