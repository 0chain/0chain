package miner

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"time"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/client"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/httpclientutil"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/threshold/bls"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	. "0chain.net/core/logging"
	"0chain.net/core/memorystore"
	"0chain.net/core/util"
	"0chain.net/smartcontract/minersc"

	"go.uber.org/zap"
)

var (
	scFunctions            = make(map[int]SmartContractFunctions)
	currentPhase           int
	gmpks                  *block.Mpks
	shareOrSigns           *block.ShareOrSigns
	txnConfirmationChannel = make(chan txnConfirmation)
	dkgSet                 bool
)

const (
	scNameAddMiner         = "add_miner"
	scNameContributeMpk    = "contributeMpk"
	scNamePublishShares    = "shareSignsOrShares"
	scRestAPIGetPhase      = "/getPhase"
	scRestAPIGetDKGMiners  = "/getDkgList"
	scRestAPIGetMinersMPKS = "/getMpksList"
	scRestAPIGetMagicBlock = "/getMagicBlock"
)

type SmartContractFunctions func() (*httpclientutil.Transaction, error)

type txnConfirmation struct {
	TxnHash   string
	Confirmed bool
}

func (mc *Chain) initSetup() {
	scFunctions[minersc.Contribute] = mc.ContributeMpk
	scFunctions[minersc.Share] = mc.SendSijs
	scFunctions[minersc.Publish] = mc.PublishShareOrSigns
	scFunctions[minersc.Wait] = mc.Wait
	gmpks = block.NewMpks()
	shareOrSigns = block.NewShareOrSigns()
	shareOrSigns.ID = node.Self.ID
	currentPhase = -1
	if !mc.IsActiveMiner(node.Self.ID, mc.CurrentRound) {
		mc.RegisterClient()
	}
	var registered bool
	for !registered {
		txn, err := mc.RegisterMiner()
		if err == nil && mc.confirmTransaction(txn) {
			registered = true
		} else {
			time.Sleep(time.Second)
		}
	}
}

func (mc *Chain) DKGProcess(ctx context.Context) {
	mc.initSetup()
	var timer = time.NewTimer(time.Duration(time.Second))
	for true {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			pn, err := mc.GetPhase()
			if err == nil && pn != nil && pn.Phase != currentPhase {
				Logger.Info("dkg process", zap.Any("sc funcs", len(scFunctions)), zap.Any("phase", pn))
				if scFunc, ok := scFunctions[pn.Phase]; ok {
					txn, err := scFunc()
					if err != nil {
						Logger.Error("smart contract function failed", zap.Any("error", err), zap.Any("current_phase", pn))
					} else {
						if txn == nil || (txn != nil && mc.confirmTransaction(txn)) {
							currentPhase = pn.Phase
						}
					}
				}
			}
			timer = time.NewTimer(time.Duration(5 * time.Second))
		}
	}
}

func (mc *Chain) GetPhase() (*minersc.PhaseNode, error) {
	pn := &minersc.PhaseNode{}
	lfb := mc.GetLatestFinalizedBlock()
	if mc.IsActiveMiner(node.Self.ID, lfb.Round) && lfb.ClientState != nil {
		clientState := chain.CreateTxnMPT(lfb.ClientState)
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
	} else {
		var sharders []string
		for _, sharder := range mc.Sharders.NodesMap {
			sharders = append(sharders, "http://"+sharder.N2NHost+":"+strconv.Itoa(sharder.Port))
		}
		err := httpclientutil.MakeSCRestAPICall(minersc.ADDRESS, scRestAPIGetPhase, nil, sharders, pn, 1)
		if err != nil {
			return nil, err
		}
	}
	return pn, nil
}

func (mc *Chain) RegisterMiner() (*httpclientutil.Transaction, error) {
	txn := httpclientutil.NewTransactionEntity(node.Self.ID, mc.ID, node.Self.PublicKey)

	mn := minersc.NewMinerNode()
	mn.ID = node.Self.GetKey()
	mn.N2NHost = node.Self.N2NHost
	mn.Host = node.Self.Host
	mn.Port = node.Self.Port
	mn.PublicKey = node.Self.PublicKey
	mn.ShortName = node.Self.Description
	mn.MinerPercentage = .5
	mn.BuildTag = node.Self.Info.BuildTag

	scData := &httpclientutil.SmartContractTxnData{}
	scData.Name = scNameAddMiner
	scData.InputArgs = mn

	txn.ToClientID = minersc.ADDRESS
	txn.PublicKey = node.Self.PublicKey
	var minerUrls []string
	for _, node := range mc.Miners.NodesMap {
		minerUrls = append(minerUrls, node.GetN2NURLBase())
	}
	err := httpclientutil.SendSmartContractTxn(txn, minersc.ADDRESS, 0, 0, scData, minerUrls)
	return txn, err
}

func (mc *Chain) ContributeMpk() (*httpclientutil.Transaction, error) {
	dmn, err := mc.GetDKGMiners()
	if err != nil {
		Logger.Error("can't contribute", zap.Any("error", err))
		return nil, err
	}
	mpk := &block.MPK{ID: node.Self.ID}
	if !dkgSet {
		if dmn.N == 0 {
			return nil, common.NewError("failed to contribute mpk", "dkg is not set yet")
		}
		vc := bls.MakeDKG(dmn.T, dmn.N, node.Self.ID)
		vc.ID = bls.ComputeIDdkg(node.Self.ID)
		mc.ViewChangeDKG = vc
		dkgSet = true
	}
	for _, v := range mc.ViewChangeDKG.Mpk {
		mpk.Mpk = append(mpk.Mpk, v.GetHexString())
	}
	scData := &httpclientutil.SmartContractTxnData{}
	scData.Name = scNameContributeMpk
	scData.InputArgs = mpk

	txn := httpclientutil.NewTransactionEntity(node.Self.ID, mc.ID, node.Self.PublicKey)
	txn.ToClientID = minersc.ADDRESS
	var minerUrls []string
	for _, node := range mc.Miners.Nodes {
		minerUrls = append(minerUrls, node.GetN2NURLBase())
	}
	err = httpclientutil.SendSmartContractTxn(txn, minersc.ADDRESS, 0, 0, scData, minerUrls)
	return txn, err
}

func (mc *Chain) CreateSijs() error {
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
		v := dmn.SimpleMinerNodes[k]
		n := node.Provider()
		n.ID = v.ID
		n.N2NHost = v.N2NHost
		n.Host = v.Host
		n.Port = v.Port
		n.PublicKey = v.PublicKey
		n.Description = v.ShortName
		n.Type = node.NodeTypeMiner
		n.Info.BuildTag = v.BuildTag
		n.Status = node.NodeStatusActive
		node.Setup(n)
		node.RegisterNode(n)
	}
	gmpks = mpks
	foundSelf := false
	lfb := mc.GetLatestFinalizedBlock()
	for k := range gmpks.Mpks {
		id := bls.ComputeIDdkg(k)
		share, err := mc.ViewChangeDKG.ComputeDKGKeyShare(id)
		if err != nil {
			Logger.Error("can't compute secret share", zap.Any("error", err))
			return err
		}
		if k == node.Self.ID {
			mc.ViewChangeDKG.AddSecretShare(id, share.GetHexString())
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
	if _, ok := dkgMiners.SimpleMinerNodes[node.Self.ID]; !dkgSet || !ok {
		Logger.Error("failed to send sijs", zap.Any("dkg_set", dkgSet), zap.Any("ok", ok))
		return nil, nil
	}
	if len(mc.ViewChangeDKG.Sij) < mc.ViewChangeDKG.T {
		err := mc.CreateSijs()
		if err != nil {
			return nil, err
		}
	}
	var failedSend []string
	nodes := node.GetMinerNodesKeys()
	for _, key := range nodes {
		if key != node.Self.ID {
			_, ok := shareOrSigns.ShareOrSigns[key]
			if !ok {
				err := mc.SendDKGShare(node.GetNode(key))
				if err != nil {
					failedSend = append(failedSend, key)
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
	if mc.IsActiveMiner(node.Self.ID, mc.CurrentRound) {
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
		var sharders []string
		for _, sharder := range mc.Sharders.NodesMap {
			sharders = append(sharders, "http://"+sharder.N2NHost+":"+strconv.Itoa(sharder.Port))
		}
		err := httpclientutil.MakeSCRestAPICall(minersc.ADDRESS, scRestAPIGetDKGMiners, nil, sharders, dmn, 1)
		if err != nil {
			return nil, err
		}
	}
	return dmn, nil
}

func (mc *Chain) GetMinersMpks() (*block.Mpks, error) {
	mpks := block.NewMpks()
	if mc.IsActiveMiner(node.Self.ID, mc.CurrentRound) {
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
		var sharders []string
		for _, sharder := range mc.Sharders.NodesMap {
			sharders = append(sharders, "http://"+sharder.N2NHost+":"+strconv.Itoa(sharder.Port))
		}
		err := httpclientutil.MakeSCRestAPICall(minersc.ADDRESS, scRestAPIGetMinersMPKS, nil, sharders, mpks, 1)
		if err != nil {
			return nil, err
		}
	}
	return mpks, nil
}

func (mc *Chain) GetFinalizedMagicBlock() (*block.MagicBlock, error) {
	magicBlock := block.NewMagicBlock()
	if mc.IsActiveMiner(node.Self.ID, mc.CurrentRound) {
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
	if mc.IsActiveMiner(node.Self.ID, mc.CurrentRound) {
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
		var sharders []string
		var err error
		for _, sharder := range mc.Sharders.NodesMap {
			sharders = append(sharders, "http://"+sharder.N2NHost+":"+strconv.Itoa(sharder.Port))
		}
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
	if !dkgSet {
		return nil, common.NewError("failed to sign share", "dkg not set")
	}
	if len(gmpks.Mpks) < mc.ViewChangeDKG.T {
		return nil, common.NewError("failed to sign", "don't have mpks yet")
	}
	message := datastore.GetEntityMetadata("dkg_share").Instance().(*bls.DKGKeyShare)
	var share bls.Key
	share.SetHexString(secShare)

	mpk := bls.ConvertStringToMpk(gmpks.Mpks[nodeID].Mpk)
	var mpkString []string
	for _, pk := range mpk {
		mpkString = append(mpkString, pk.GetHexString())
	}
	if mc.ViewChangeDKG.ValidateShare(mpk, share) {
		err := mc.ViewChangeDKG.AddSecretShare(bls.ComputeIDdkg(nodeID), secShare)
		if err != nil {
			return nil, err
		}
		message.Message = node.Self.ID
		message.Sign = share.Sign(message.Message).GetHexString()
	} else {
		Logger.Error("failed to verify dkg share", zap.Any("share", secShare), zap.Any("node_id", nodeID))
	}
	return message, nil
}

/*SendDKGShare sends the generated secShare to the given node */
func (mc *Chain) SendDKGShare(n *node.Node) error {
	if !config.DevConfiguration.IsDkgEnabled {
		return common.NewError("failed to send dkg share", "dkg is not enabled")
	}
	if node.Self.ID == n.ID {
		return nil
	}
	var success bool

	params := &url.Values{}
	nodeID := bls.ComputeIDdkg(n.ID)
	secShare := mc.ViewChangeDKG.Sij[nodeID]
	params.Add("secret_share", secShare.GetHexString())
	handler := func(ctx context.Context, entity datastore.Entity) (interface{}, error) {
		share, ok := entity.(*bls.DKGKeyShare)
		if !ok {
			Logger.Error("invalid share or sign", zap.Any("entity", entity))
			return nil, nil
		}
		if share.Message != "" && share.Sign != "" {
			var sign bls.Sign
			var publicKey bls.PublicKey
			publicKey.Set(mc.ViewChangeDKG.Mpk, &nodeID)
			sign.SetHexString(share.Sign)
			if !sign.Verify(&publicKey, share.Message) {
				Logger.Error("invalid share or sign", zap.Any("publicKey", publicKey.GetHexString()), zap.Any("message", share.Message), zap.Any("sign", sign.GetHexString()))
				return nil, nil
			}
			success = true
			shareOrSigns.ShareOrSigns[n.ID] = share
		}
		return nil, nil
	}
	ctx := common.GetRootContext()
	n.RequestEntityFromNode(ctx, DKGShareSender, params, handler)
	if !success {
		return common.NewError("failed to send dkg", "miner returned error")
	}
	return nil
}

func (mc *Chain) PublishShareOrSigns() (*httpclientutil.Transaction, error) {
	if !dkgSet {
		Logger.Error("failed to publish share or signs", zap.Any("dkg_set", dkgSet))
		return nil, nil
	}
	txn := httpclientutil.NewTransactionEntity(node.Self.ID, mc.ID, node.Self.PublicKey)

	mpks, err := mc.GetMinersMpks()
	if err != nil {
		return nil, err
	}
	if _, ok := mpks.Mpks[node.Self.ID]; !ok {
		return nil, nil
	}
	for k := range mpks.Mpks {
		if _, ok := shareOrSigns.ShareOrSigns[k]; !ok && k != node.Self.ID {
			share := mc.ViewChangeDKG.Sij[bls.ComputeIDdkg(k)]
			shareOrSigns.ShareOrSigns[k] = &bls.DKGKeyShare{Share: share.GetHexString()}
		}
	}
	_, ok := shareOrSigns.Validate(mpks)
	if !ok {
		Logger.Error("failed to verify share or signs", zap.Any("mpks", mpks))
	}
	scData := &httpclientutil.SmartContractTxnData{}
	scData.Name = scNamePublishShares
	scData.InputArgs = shareOrSigns

	txn.ToClientID = minersc.ADDRESS
	var minerUrls []string
	for _, node := range mc.Miners.Nodes {
		minerUrls = append(minerUrls, node.GetN2NURLBase())
	}
	err = httpclientutil.SendSmartContractTxn(txn, minersc.ADDRESS, 0, 0, scData, minerUrls)
	return txn, err
}

func (mc *Chain) Wait() (*httpclientutil.Transaction, error) {
	magicBlock, err := mc.GetMagicBlockFromSC()
	if err != nil {
		return nil, err
	}
	if _, ok := magicBlock.Miners.NodesMap[node.Self.ID]; !ok {
		err := mc.UpdateMagicBlock(magicBlock)
		if err != nil {
			Logger.DPanic(fmt.Sprintf("failed to update magic block: %v", err.Error()))
		}
		return nil, nil
	}
	for key, share := range magicBlock.ShareOrSigns.Shares {
		myShare, ok := share.ShareOrSigns[node.Self.ID]
		if ok && myShare.Share != "" {
			var share bls.Key
			share.SetHexString(myShare.Share)
			if mc.ViewChangeDKG.ValidateShare(bls.ConvertStringToMpk(gmpks.Mpks[key].Mpk), share) {
				err := mc.ViewChangeDKG.AddSecretShare(bls.ComputeIDdkg(key), myShare.Share)
				if err != nil {
					return nil, err
				}
			}
		}
	}
	var miners []string
	for key := range gmpks.Mpks {
		if _, ok := magicBlock.Mpks.Mpks[key]; !ok {
			miners = append(miners, key)
		}
	}

	mc.ViewChangeDKG.DeleteFromSet(miners)
	mc.ViewChangeDKG.AggregatePublicKeyShares(magicBlock.Mpks.GetMpkMap())
	mc.ViewChangeDKG.AggregateSecretKeyShares()
	mc.ViewChangeMagicBlock = magicBlock
	mc.ViewChangeDKG.StartingRound = magicBlock.StartingRound
	storeDKGSummary(common.GetRootContext(), mc.ViewChangeDKG.GetDKGSummary())
	mc.NextViewChange = magicBlock.StartingRound
	shareOrSigns = block.NewShareOrSigns()
	shareOrSigns.ID = node.Self.ID
	gmpks = block.NewMpks()
	dkgSet = false
	return nil, nil
}

func (mc *Chain) confirmTransaction(t *httpclientutil.Transaction) bool {
	active := mc.IsActiveMiner(node.Self.ID, mc.CurrentRound)
	var found, pastTime bool
	var urls []string
	for _, sharder := range mc.Sharders.NodesMap {
		if !active || sharder.Status == node.NodeStatusActive {
			urls = append(urls, "http://"+sharder.N2NHost+":"+strconv.Itoa(sharder.Port))
		}
	}
	for !found && !pastTime {
		txn, err := httpclientutil.GetTransactionStatus(t.Hash, urls, 1)
		if active {
			lfb := mc.GetLatestFinalizedBlock()
			pastTime = lfb != nil && !common.WithinTime(int64(lfb.CreationDate), int64(t.CreationDate), transaction.TXN_TIME_TOLERANCE)
		} else {
			blockSummary, err := httpclientutil.GetBlockSummaryCall(urls, 1, false)
			if err != nil {
				return false
			}
			pastTime = blockSummary != nil && !common.WithinTime(int64(blockSummary.CreationDate), int64(t.CreationDate), transaction.TXN_TIME_TOLERANCE)
		}
		found = err == nil && txn != nil
		if !found {
			time.Sleep(time.Second)
		}
	}
	return found

}

//RegisterClient registers client on BC
func (mc *Chain) RegisterClient() {
	clientMetadataProvider := datastore.GetEntityMetadata("client")
	ctx := memorystore.WithEntityConnection(common.GetRootContext(), clientMetadataProvider)
	defer memorystore.Close(ctx)
	ctx = datastore.WithAsyncChannel(ctx, client.ClientEntityChannel)

	_, err := client.PutClient(ctx, &node.Self.Client)
	if err != nil {
		panic(err)
	}

	nodeBytes, _ := json.Marshal(node.Self.Client)
	miners := make(map[string]*node.Node)
	for key, miner := range mc.Miners.NodesMap {
		miners[key] = miner
	}
	for len(mc.Miners.NodesMap) < 2*len(miners) {
		for key, miner := range miners {
			body, err := httpclientutil.SendPostRequest("http://"+miner.N2NHost+":"+strconv.Itoa(miner.Port)+httpclientutil.RegisterClient, nodeBytes, "", "", nil)
			if err != nil {
				Logger.Error("error in register client", zap.Error(err), zap.Any("body", body))
			} else {
				delete(miners, key)
			}
			time.Sleep(httpclientutil.SleepBetweenRetries * time.Millisecond)
		}
		time.Sleep(httpclientutil.SleepBetweenRetries * time.Millisecond)
	}
}
