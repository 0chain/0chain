// +build integration_tests

package miner

import (
	"context"
	"net/url"

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

	"0chain.net/conductor/conductrpc" // integration tests
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

	switch {
	case conductrpc.IsSendShareFor(n.GetKey()):
		params.Add("secret_share", secShare.GetHexString())
	case conductrpc.IsSendBadShareFor(n.GetKey()):
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

	var isRevealed = conductrpc.IsRevealed()

	for k := range mpks.Mpks {
		if k == selfNodeKey {
			continue
		}
		var _, ok = shareOrSigns.ShareOrSigns[k]
		if isRevealed || !ok {
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
	_, ok := shareOrSigns.Validate(mpks, publicKeys, chain.GetServerChain().GetSignatureScheme())
	if !ok {
		Logger.Error("failed to verify share or signs", zap.Any("mpks", mpks))
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
