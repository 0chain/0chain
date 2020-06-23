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
)

func revertString(s string) string {
	r := []rune(s)
	for i, j := 0, len(r)-1; i < len(r)/2; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return string(r)
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

	var state = crpc.Client().State()
	switch nodeID := n.GetKey(); {
	case state.Shares.IsGood(state, nodeID):
		params.Add("secret_share", secShare.GetHexString())
	case state.Shares.IsBad(state, nodeID):
		params.Add("secret_share", revertString(secShare.GetHexString()))
	default:
		return common.NewError("failed to send DKG share", "skipped by tests")
	}

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

	var isRevealed = crpc.Client().State().IsRevealed

	for k := range mpks.Mpks {
		if k == selfNodeKey {
			continue
		}
		var _, ok = shareOrSigns.ShareOrSigns[k]
		if isRevealed || !ok {
			if isRevealed {
				println("IS REVEALED")
			}
			share := mc.viewChangeDKG.Sij[bls.ComputeIDdkg(k)]
			shareOrSigns.ShareOrSigns[k] = &bls.DKGKeyShare{
				Share: share.GetHexString(),
			}
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
	_, ok := shareOrSigns.Validate(mpks, publicKeys,
		chain.GetServerChain().GetSignatureScheme())
	if !ok {
		Logger.Error("failed to verify share or signs", zap.Any("mpks", mpks))
	}

	var (
		state = crpc.Client().State()
		clone = shareOrSigns.Clone()
	)

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

	scData := &httpclientutil.SmartContractTxnData{}
	scData.Name = scNamePublishShares
	scData.InputArgs = shareOrSigns.Clone()
	txn.ToClientID = minersc.ADDRESS

	var minerUrls []string
	for id := range dkgMiners.SimpleNodes {
		nodeSend := node.GetNode(id)
		if nodeSend != nil {
			minerUrls = append(minerUrls, nodeSend.GetN2NURLBase())
		} else {
			Logger.Warn("failed to get node", zap.Any("id", id))
		}
	}
	err = httpclientutil.SendSmartContractTxn(txn, minersc.ADDRESS, 0, 0, scData, minerUrls)
	return txn, err
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
	urls []string) (txn *httpclientutil.Transaction, err error) {

	var scData = new(httpclientutil.SmartContractTxnData)
	scData.Name = scNameContributeMpk
	scData.InputArgs = mpk
	txn = httpclientutil.NewTransactionEntity(selfNode.GetKey(), mc.ID,
		selfNode.PublicKey)
	txn.ToClientID = minersc.ADDRESS
	err = httpclientutil.SendSmartContractTxn(txn, minersc.ADDRESS, 0, 0,
		scData, urls)
	return
}

func (mc *Chain) ContributeMpk() (txn *httpclientutil.Transaction, err error) {
	magicBlock := mc.GetCurrentMagicBlock()
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
			return nil, common.NewError("contribute_mpk",
				"failed to contribute mpk:dkg is not set yet")
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

	var (
		state             = crpc.Client().State()
		good, bad         = state.Split(state.MPK, magicBlock.Miners.Nodes)
		goodurls, badurls = getBaseN2NURLs(good), getBaseN2NURLs(bad)
		badMPK            = getBadMPK(mpk)
	)

	// send bad MPK first
	if len(bad) > 0 {
		txn, err = mc.sendMpkTransaction(selfNode, badMPK, badurls)
		if err != nil {
			return
		}
	}

	// send good MPK second
	if len(good) > 0 {
		txn, err = mc.sendMpkTransaction(selfNode, mpk, goodurls)
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
	if !mc.isDKGSet() {
		return nil, common.NewError("failed to sign share", "dkg not set")
	}

	viewChangeMutex.Lock()
	defer viewChangeMutex.Unlock()

	var mpks = mc.GetMpks()
	if len(mpks) < mc.viewChangeDKG.T {
		return nil, common.NewError("failed to sign", "don't have mpks yet")
	}

	var (
		message = datastore.GetEntityMetadata("dkg_share").
			Instance().(*bls.DKGKeyShare)
		share bls.Key
	)
	if err = share.SetHexString(secShare); err != nil {
		Logger.Error("failed to set hex string", zap.Any("error", err))
		return nil, err
	}

	var (
		mpk       = bls.ConvertStringToMpk(mpks[nodeID].Mpk)
		mpkString []string
	)
	for _, pk := range mpk {
		mpkString = append(mpkString, pk.GetHexString())
	}

	if !mc.viewChangeDKG.ValidateShare(mpk, share) {
		Logger.Error("failed to verify dkg share", zap.Any("share", secShare), zap.Any("node_id", nodeID))
		return nil, common.NewError("failed to sign", "failed to verify dkg share")
	}
	err = mc.viewChangeDKG.AddSecretShare(bls.ComputeIDdkg(nodeID), secShare,
		false)
	if err != nil {
		return nil, err
	}

	message.Message = node.Self.Underlying().GetKey()
	message.Sign, err = node.Self.Sign(message.Message)
	if err != nil {
		Logger.Error("failed to sign dkg share message", zap.Any("error", err))
		return nil, err
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
