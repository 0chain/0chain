package miner

import (
	"context"
	"fmt"
	"math"
	"reflect"
	"sync"
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

const (
	// transactions
	scNameContributeMpk = "contributeMpk"
	scNamePublishShares = "shareSignsOrShares"
	scNameWait          = "wait"
	// REST API requests
	scRestAPIGetPhase      = "/getPhase"
	scRestAPIGetDKGMiners  = "/getDkgList"
	scRestAPIGetMinersMPKS = "/getMpksList"
	scRestAPIGetMagicBlock = "/getMagicBlock"
	scRestAPIGetMinerList  = "/getMinerList"
)

// SmartContractFunctions represents local VC function returns optional
// Miner SC transactions.
type SmartContractFunctions func(ctx context.Context, lfb *block.Block,
	lfmb *block.MagicBlock, isActive bool) (tx *httpclientutil.Transaction,
	err error)

// view change process controlling
type viewChangeProcess struct {
	sync.Mutex

	scFunctions   map[minersc.Phase]SmartContractFunctions
	currentPhase  minersc.Phase
	shareOrSigns  *block.ShareOrSigns
	mpks          *block.Mpks
	viewChangeDKG *bls.DKG

	// round we expect next view change (can be adjusted)
	nvcmx          sync.RWMutex
	nextViewChange int64 // expected
}

func (vcp *viewChangeProcess) CurrentPhase() minersc.Phase {
	return vcp.currentPhase
}

func (vcp *viewChangeProcess) SetCurrentPhase(ph minersc.Phase) {
	vcp.currentPhase = ph
}

func (vcp *viewChangeProcess) init(mc *Chain) {
	vcp.scFunctions = map[minersc.Phase]SmartContractFunctions{
		minersc.Start:      mc.DKGProcessStart,
		minersc.Contribute: mc.ContributeMpk,
		minersc.Share:      mc.SendSijs,
		minersc.Publish:    mc.PublishShareOrSigns,
		minersc.Wait:       mc.Wait,
	}
	vcp.shareOrSigns = block.NewShareOrSigns()
	vcp.shareOrSigns.ID = node.Self.Underlying().GetKey()
	vcp.currentPhase = minersc.Unknown
	vcp.mpks = block.NewMpks()
}

func (mc *Chain) isActiveInChain(lfb *block.Block, mb *block.MagicBlock) bool {
	return mb.Miners.HasNode(node.Self.GetKey()) &&
		mb.StartingRound < mc.GetCurrentRound() && lfb.ClientState != nil
}

// After stop/start we have to repair nextViewCahnge round number from
// store if there is "latest" MB saved in Miner SC;
func (vcp *viewChangeProcess) setupNextViewChange(ctx context.Context) {
	var mb, err = GetLatestMagicBlockFromStore(ctx)
	if err != nil {
		Logger.Info("getting latest MB from store", zap.Error(err))
		return
	}
	Logger.Info("next view change", zap.Int64("round", mb.StartingRound))
	vcp.SetNextViewChange(mb.StartingRound)
}

// DKGProcess starts DKG process and works on it. It blocks.
func (mc *Chain) DKGProcess(ctx context.Context) {

	mc.viewChangeProcess.setupNextViewChange(ctx)

	const (
		timeoutPhase        = 5
		thresholdCheckPhase = 2
		thisNodeMovements   = -1
	)

	var (
		timer                    = time.NewTimer(time.Duration(time.Second))
		counterCheckPhaseSharder = thresholdCheckPhase + 1 // check on sharders

		oldPhaseRound   minersc.Phase
		oldCurrentRound int64
	)

	for {
		timer.Reset(timeoutPhase * time.Second) // setup timer to wait

		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}

		var (
			lfb    = mc.GetLatestFinalizedBlock()
			mb     = mc.GetMagicBlock(lfb.Round)
			active = mc.isActiveInChain(lfb, mb)

			checkOnSharder = counterCheckPhaseSharder >= thresholdCheckPhase
			pn, err        = mc.GetPhase(lfb, mb, active, checkOnSharder)
		)

		if err != nil {
			Logger.Error("dkg process getting phase", zap.Any("error", err),
				zap.Int("sc funcs", len(mc.viewChangeProcess.scFunctions)),
				zap.Any("phase", pn))
			continue
		}

		if !checkOnSharder && pn.Phase == oldPhaseRound &&
			pn.CurrentRound == oldCurrentRound {

			counterCheckPhaseSharder++
		} else {
			counterCheckPhaseSharder = 0
			oldPhaseRound = pn.Phase
			oldCurrentRound = pn.CurrentRound
		}

		Logger.Debug("dkg process trying", zap.Any("next_phase", pn),
			zap.Int("phase", int(mc.CurrentPhase())),
			zap.Int("sc funcs", len(mc.viewChangeProcess.scFunctions)))

		if pn == nil || pn.Phase == mc.CurrentPhase() {
			continue
		}

		if pn.Phase != 0 && pn.Phase != mc.CurrentPhase()+1 {
			Logger.Debug("jumping over a phase; skip, wait for 'start'",
				zap.Int("current_phase", int(mc.CurrentPhase())),
				zap.Int("phase", int(pn.Phase)))
			mc.SetCurrentPhase(minersc.Unknown)
			continue
		}

		Logger.Info("dkg process start", zap.Any("next_phase", pn),
			zap.Any("phase", mc.CurrentPhase()),
			zap.Int("sc funcs", len(mc.viewChangeProcess.scFunctions)))

		var scFunc, ok = mc.viewChangeProcess.scFunctions[pn.Phase]
		if !ok {
			Logger.Debug("dkg process: no such phase function",
				zap.Any("phase", pn.Phase))
			continue
		}

		var txn *httpclientutil.Transaction
		if txn, err = scFunc(ctx, lfb, mb, active); err != nil {
			Logger.Error("smart contract function failed",
				zap.Any("error", err), zap.Any("next_phase", pn))
			continue
		}

		Logger.Debug("dkg process move phase",
			zap.Any("next_phase", pn),
			zap.Any("phase", mc.CurrentPhase()),
			zap.Any("txn", txn))

		if txn == nil || (txn != nil && mc.ConfirmTransaction(txn)) {
			var oldPhase = mc.CurrentPhase()
			mc.SetCurrentPhase(pn.Phase)
			Logger.Debug("dkg process moved phase",
				zap.Any("old_phase", oldPhase),
				zap.Any("phase", mc.CurrentPhase()))
		}
	}

}

func makeSCRESTAPICall(address, relative, sharder string,
	seri util.Serializable, collect chan util.Serializable) {

	var err = httpclientutil.MakeSCRestAPICall(address, relative, nil,
		[]string{sharder}, seri, 1)
	if err != nil {
		Logger.Error("requesting phase node from sharder",
			zap.String("sharder", sharder),
			zap.Error(err))
	}
	collect <- seri // regardless error
}

// is given reflect value zero (reflect.Value.IsZero added in go1.13)
func isValueZero(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32,
		reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return math.Float64bits(v.Float()) == 0
	case reflect.Complex64, reflect.Complex128:
		c := v.Complex()
		return math.Float64bits(real(c)) == 0 && math.Float64bits(imag(c)) == 0
	case reflect.Array:
		for i := 0; i < v.Len(); i++ {
			if !isValueZero(v.Index(i)) {
				return false
			}
		}
		return true
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map,
		reflect.Ptr, reflect.Slice, reflect.UnsafePointer:
		return v.IsNil()
	case reflect.String:
		return v.Len() == 0
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if !isValueZero(v.Field(i)) {
				return false
			}
		}
		return true
	default:
		panic("reflect.Value.IsZero")
	}
	return false
}

func isZero(val interface{}) bool {
	if val == nil {
		return true
	}
	var rval = reflect.ValueOf(val)
	if isValueZero(rval) {
		return true
	}
	if rval.Kind() == reflect.Ptr {
		return isValueZero(rval.Elem())
	}
	return false
}

func mostConsensus(list []util.Serializable) (elem util.Serializable) {

	type valueCount struct {
		value util.Serializable
		count int
	}

	var cons = make(map[string]*valueCount, len(list))
	for _, val := range list {
		var str = string(val.Encode())
		if vc, ok := cons[str]; ok {
			vc.count++
		} else {
			cons[str] = &valueCount{value: val, count: 1}
		}
	}

	var most int
	for _, vc := range cons {
		if vc.count > most {
			elem = vc.value // use this
			most = vc.count // update
		}
	}
	return
}

func getFromSharders(address, relative string, sharders []string,
	newFunc func() util.Serializable,
	rejectFunc func(seri util.Serializable) bool) (got util.Serializable) {

	var collect = make(chan util.Serializable, len(sharders))
	for _, sharder := range sharders {
		go makeSCRESTAPICall(address, relative, sharder,
			newFunc(), collect)
	}
	var list = make([]util.Serializable, 0, len(sharders))
	for range sharders {
		if val := <-collect; !isZero(val) && !rejectFunc(val) {
			list = append(list, val) // don't add zero values
		}
	}
	return mostConsensus(list)
}

func getNodePath(path string) util.Path {
	return util.Path(encryption.Hash(path))
}

func (mc *Chain) GetLFBState() util.MerklePatriciaTrieI {
	return chain.CreateTxnMPT(mc.GetLatestFinalizedBlock().ClientState)
}

func (mc *Chain) GetLFBStateNode(path string) (util.Serializable, error) {
	var state = mc.GetLFBState()
	return state.GetNodeValue(getNodePath(path))
}

func (mc *Chain) GetBlockStateNode(block *block.Block, path string) (
	util.Serializable, error) {

	return block.ClientState.GetNodeValue(getNodePath(path))
}

func (mc *Chain) GetPhase(lfb *block.Block, mb *block.MagicBlock, active bool,
	fromSharder bool) (pn *minersc.PhaseNode, err error) {

	pn = &minersc.PhaseNode{}

	if active {
		var node util.Serializable
		if node, err = mc.GetBlockStateNode(lfb, pn.GetKey()); err != nil {
			return nil, err
		}
		if node == nil {
			return nil, common.NewError("key_not_found", "key was not found")
		}
		if err = pn.Decode(node.Encode()); err != nil {
			return nil, err
		}
		if !fromSharder {
			return pn, nil
		}
	}

	var got util.Serializable

	got = getFromSharders(minersc.ADDRESS, scRestAPIGetPhase,
		mb.Sharders.N2NURLs(), func() util.Serializable {
			return new(minersc.PhaseNode)
		}, func(val util.Serializable) bool {
			if pn, ok := val.(*minersc.PhaseNode); ok {
				if pn.StartRound < mb.StartingRound {
					return true // reject
				}
				return false // keep
			}
			return true // reject
		})

	var phase, ok = got.(*minersc.PhaseNode)
	if !ok {
		return nil, common.NewError("get_dkg_phase_from_sharders",
			"can't get, no phases given")
	}

	if active && (pn.Phase != phase.Phase || pn.StartRound != phase.StartRound) {
		Logger.Warn("dkg_process -- phase in state does not match shader",
			zap.Any("phase_state", pn.Phase),
			zap.Any("phase_sharder", phase.Phase),
			zap.Any("start_round_state", pn.StartRound),
			zap.Any("start_round_sharder", phase.StartRound))
		return phase, nil
	}

	if active {
		return pn, nil
	}

	return phase, nil
}

func (vcp *viewChangeProcess) clearViewChange() {
	vcp.shareOrSigns = block.NewShareOrSigns()
	vcp.shareOrSigns.ID = node.Self.Underlying().GetKey()
	vcp.mpks = block.NewMpks()
	vcp.viewChangeDKG = nil
}

//
//                               S T A R T
//

// DKGProcessStart represents 'start' phase function.
func (mc *Chain) DKGProcessStart(context.Context, *block.Block,
	*block.MagicBlock, bool) (*httpclientutil.Transaction, error) {

	mc.viewChangeProcess.Lock()
	defer mc.viewChangeProcess.Unlock()

	mc.viewChangeProcess.clearViewChange()
	return nil, nil
}

func (vcp *viewChangeProcess) isNeedCreateSijs() (ok bool) {
	return vcp.viewChangeDKG != nil &&
		len(vcp.viewChangeDKG.Sij) < vcp.viewChangeDKG.T
}

func (mc *Chain) getMinersMpks(lfb *block.Block, mb *block.MagicBlock,
	active bool) (mpks *block.Mpks, err error) {

	if active {

		var node util.Serializable
		node, err = mc.GetBlockStateNode(lfb, minersc.MinersMPKKey)
		if err != nil {
			return
		}
		if node == nil {
			return nil, common.NewError("key_not_found", "key was not found")
		}

		mpks = block.NewMpks()
		if err = mpks.Decode(node.Encode()); err != nil {
			return nil, err
		}

		return
	}

	var (
		got util.Serializable
		ok  bool
	)

	got = getFromSharders(minersc.ADDRESS, scRestAPIGetMinersMPKS,
		mb.Sharders.N2NURLs(), func() util.Serializable {
			return block.NewMpks()
		}, func(val util.Serializable) bool {
			return false // keep all (how to reject old?)
		})

	if mpks, ok = got.(*block.Mpks); !ok {
		return nil, common.NewError("get_mpks_from_sharders", "no MPKs given")
	}

	return
}

func (mc *Chain) getDKGMiners(lfb *block.Block, mb *block.MagicBlock,
	active bool) (dmn *minersc.DKGMinerNodes, err error) {

	if active {

		var node util.Serializable
		node, err = mc.GetBlockStateNode(lfb, minersc.DKGMinersKey)
		if err != nil {
			return
		}
		if node == nil {
			return nil, common.NewError("key_not_found", "key was not found")
		}

		dmn = minersc.NewDKGMinerNodes()
		err = dmn.Decode(node.Encode())
		if err != nil {
			return nil, err
		}

		return
	}

	var (
		got util.Serializable
		ok  bool
	)

	got = getFromSharders(minersc.ADDRESS, scRestAPIGetDKGMiners,
		mb.Sharders.N2NURLs(), func() util.Serializable {
			return new(minersc.DKGMinerNodes)
		}, func(val util.Serializable) bool {
			if dmn, ok := val.(*minersc.DKGMinerNodes); ok {
				if dmn.StartRound < mb.StartingRound {
					return true // reject
				}
				return false // keep
			}
			return true // reject
		})

	if dmn, ok = got.(*minersc.DKGMinerNodes); !ok {
		return nil, common.NewError("get_dkg_miner_nodes_from_sharders",
			"no DKG miner nodes given")
	}

	return
}

func (mc *Chain) createSijs(lfb *block.Block, mb *block.MagicBlock,
	active bool) (err error) {

	if !mc.viewChangeProcess.isNeedCreateSijs() {
		return // doesn't need to create them
	}

	var mpks *block.Mpks
	if mpks, err = mc.getMinersMpks(lfb, mb, active); err != nil {
		Logger.Error("can't share", zap.Any("error", err))
		return
	}

	var dmn *minersc.DKGMinerNodes
	if dmn, err = mc.getDKGMiners(lfb, mb, active); err != nil {
		Logger.Error("can't share", zap.Any("error", err))
		return
	}

	for k := range mpks.Mpks {
		if node.GetNode(k) != nil {
			continue // already registered
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

	mc.viewChangeProcess.mpks = mpks // set

	var foundSelf = false

	for k := range mc.mpks.GetMpks() {
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
		Logger.Error("failed to add secret key for self",
			zap.Any("lfb_round", lfb.Round))
	}

	return nil
}

func (vcp *viewChangeProcess) isDKGSet() bool {
	return vcp.viewChangeDKG != nil
}

//
//                                 S H A R E
//

func (mc *Chain) SendSijs(ctx context.Context, lfb *block.Block,
	mb *block.MagicBlock, active bool) (tx *httpclientutil.Transaction,
	err error) {

	mc.viewChangeProcess.Lock()
	defer mc.viewChangeProcess.Unlock()

	if !mc.viewChangeProcess.isDKGSet() {
		return nil, common.NewError("dkg_not_set", "send_sijs: DKG is not set")
	}

	var dkgMiners *minersc.DKGMinerNodes
	if dkgMiners, err = mc.getDKGMiners(lfb, mb, active); err != nil {
		return // error
	}

	var selfNodeKey = node.Self.Underlying().GetKey()
	if _, ok := dkgMiners.SimpleNodes[selfNodeKey]; !mc.isDKGSet() || !ok {
		Logger.Error("failed to send sijs",
			zap.Any("dkg_set", mc.isDKGSet()), zap.Any("ok", ok))
		return // (nil, nil)
	}

	if err = mc.createSijs(lfb, mb, active); err != nil {
		return // error
	}

	// we haven't to make sure all nodes are registered, because the
	// createSijs registers them; and after a restart ('deregister')
	// we have to restart DKG for this miner, since secret key is lost

	var sendFail []string
	for key := range dkgMiners.SimpleNodes {
		if key == selfNodeKey {
			continue // don't send to self
		}
		if _, ok := mc.viewChangeProcess.shareOrSigns.ShareOrSigns[key]; !ok {
			if err := mc.SendDKGShare(ctx, node.GetNode(key)); err != nil {
				sendFail = append(sendFail, fmt.Sprintf("%s(%v);", key, err))
			}
		}
	}

	if len(sendFail) > 0 {
		return nil, common.NewError("failed to send sijs",
			fmt.Sprintf("failed to send share to miners: %v", sendFail))
	}

	return // (nil, nil)
}

func (mc *Chain) GetMagicBlockFromSC(lfb *block.Block, mb *block.MagicBlock,
	active bool) (magicBlock *block.MagicBlock, err error) {

	if active {
		var node util.Serializable
		node, err = mc.GetBlockStateNode(lfb, minersc.MagicBlockKey)
		if err != nil {
			return // error
		}
		if node == nil {
			return nil, common.NewError("key_not_found", "key was not found")
		}

		magicBlock = block.NewMagicBlock()
		if err = magicBlock.Decode(node.Encode()); err != nil {
			return nil, err
		}

		return // ok
	}

	var (
		got util.Serializable
		ok  bool
	)

	got = getFromSharders(minersc.ADDRESS, scRestAPIGetMagicBlock,
		mb.Sharders.N2NURLs(), func() util.Serializable {
			return block.NewMagicBlock()
		}, func(val util.Serializable) bool {
			if mx, ok := val.(*block.MagicBlock); ok {
				if mx.StartingRound < mb.StartingRound {
					return true // reject
				}
				return false // keep
			}
			return true // reject
		})

	if magicBlock, ok = got.(*block.MagicBlock); !ok {
		return nil, common.NewError("get_magic_block_from_sharders",
			"no magic block given")
	}

	return
}

func (mc *Chain) waitTransaction(mb *block.MagicBlock) (
	tx *httpclientutil.Transaction, err error) {

	var data = new(httpclientutil.SmartContractTxnData)
	data.Name = scNameWait

	var selfNode = node.Self.Underlying()

	tx = httpclientutil.NewTransactionEntity(selfNode.GetKey(), mc.ID,
		selfNode.PublicKey)
	tx.ToClientID = minersc.ADDRESS

	err = httpclientutil.SendSmartContractTxn(tx, minersc.ADDRESS, 0, 0, data,
		mb.Miners.N2NURLs())
	return
}

func (vcp *viewChangeProcess) NextViewChange() (round int64) {
	vcp.nvcmx.RLock()
	defer vcp.nvcmx.RUnlock()

	return vcp.nextViewChange
}

func (vcp *viewChangeProcess) SetNextViewChange(round int64) {
	vcp.nvcmx.Lock()
	defer vcp.nvcmx.Unlock()

	vcp.nextViewChange = round
}

//
//                               W A I T
//

func (mc *Chain) Wait(ctx context.Context, lfb *block.Block,
	mb *block.MagicBlock, active bool) (tx *httpclientutil.Transaction,
	err error) {

	mc.viewChangeProcess.Lock()
	defer mc.viewChangeProcess.Unlock()

	if !mc.viewChangeProcess.isDKGSet() {
		return nil, common.NewError("vc_wait", "DKG is not set")
	}

	var magicBlock *block.MagicBlock
	if magicBlock, err = mc.GetMagicBlockFromSC(lfb, mb, active); err != nil {
		return // error
	}

	if !magicBlock.Miners.HasNode(node.Self.Underlying().GetKey()) {
		mc.viewChangeProcess.clearViewChange()
		return // node leaves BC, don't do anything here
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
			share.SetHexString(myShare.Share)
			var validShare = vcdkg.ValidateShare(
				bls.ConvertStringToMpk(mpks[key].Mpk), share)
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

	vcdkg.AggregatePublicKeyShares(magicBlock.Mpks.GetMpkMap())
	vcdkg.AggregateSecretKeyShares()
	vcdkg.StartingRound = magicBlock.StartingRound
	vcdkg.MagicBlockNumber = magicBlock.MagicBlockNumber
	// set T and N from the magic block
	vcdkg.T = magicBlock.T
	vcdkg.N = magicBlock.N

	// save DKG and MB

	var summary = vcdkg.GetDKGSummary()
	if err = StoreDKGSummary(ctx, summary); err != nil {
		return nil, common.NewErrorf("vc_wait", "saving DKG summary: %v", err)
	}

	var mbData = block.NewMagicBlockData(magicBlock)
	if err = StoreMagicBlockDataLatest(ctx, mbData); err != nil {
		return nil, common.NewErrorf("vc_wait", "saving MB data: %v", err)
	}

	// don't set DKG until MB finalized

	// TODO (sfxdx): SET DKG WERE MB FINALIZED
	//
	// if err := mc.SetDKG(mc.viewChangeDKG, magicBlock.StartingRound); err != nil {
	// 	Logger.Error("failed to set dkg", zap.Error(err))
	// }

	mc.viewChangeProcess.clearViewChange()
	mc.SetNextViewChange(magicBlock.StartingRound)

	// create 'wait' transaction
	if tx, err = mc.waitTransaction(mb); err != nil {
		return nil, common.NewErrorf("vc_wait",
			"sending 'wait' transaction: %v", err)
	}

	return // the transaction
}

//
// other VC helpers
//

func StoreMagicBlockDataLatest(ctx context.Context,
	data *block.MagicBlockData) (err error) {

	if err = StoreMagicBlockData(ctx, data); err != nil {
		return
	}
	// latest
	var dataLatest = block.NewMagicBlockData(data.MagicBlock)
	dataLatest.ID = "latest"
	if err = StoreMagicBlockData(ctx, dataLatest); err != nil {
		return
	}
	return // ok
}

func StoreMagicBlockData(ctx context.Context, data *block.MagicBlockData) (
	err error) {

	var (
		emd  = data.GetEntityMetadata()
		dctx = ememorystore.WithEntityConnection(ctx, emd)
	)
	defer ememorystore.Close(dctx)

	if err = data.Write(dctx); err != nil {
		return
	}

	var connection = ememorystore.GetEntityCon(dctx, emd)
	return connection.Commit()
}

func GetMagicBlockDataFromStore(ctx context.Context, id string) (
	mbd *block.MagicBlockData, err error) {

	mbd = datastore.GetEntity("magicblockdata").(*block.MagicBlockData)
	mbd.ID = id

	var (
		emd  = mbd.GetEntityMetadata()
		dctx = ememorystore.WithEntityConnection(ctx, emd)
	)
	defer ememorystore.Close(dctx)

	err = mbd.Read(dctx, mbd.GetKey())
	return
}

func GetLatestMagicBlockFromStore(ctx context.Context) (
	magicBlock *block.MagicBlock, err error) {

	var data *block.MagicBlockData
	if data, err = GetMagicBlockDataFromStore(ctx, "latest"); err != nil {
		return
	}

	magicBlock = data.MagicBlock
	return
}
