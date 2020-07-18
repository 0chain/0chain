package miner

import (
	"context"
	"errors"
	"fmt"
	"math"
	"reflect"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	// TORM
	"encoding/json"

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

	var (
		mb  = mc.GetCurrentMagicBlock()
		got util.Serializable
	)

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
			zap.Any("phase_state", pn.Phase), zap.Any("phase_sharder", phase.Phase),
			zap.Any("start_round_state", pn.StartRound),
			zap.Any("start_round_sharder", phase.StartRound))
		return phase, nil
	}
	if active {
		return pn, nil
	}
	return phase, nil
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
		var (
			mb  = mc.GetCurrentMagicBlock()
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
		var (
			mb  = mc.GetCurrentMagicBlock()
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
			return nil, common.NewError("get_mpks_from_sharders",
				"no MPKs given")
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
			return nil, common.NewError("failed to get magic block",
				"no magic block available")
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
		var (
			mb  = mc.GetCurrentMagicBlock()
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
	}
	return magicBlock, nil
}

func mustJSON(val interface{}) string {
	b, err := json.Marshal(val)
	if err != nil {
		panic("JSON MARSHALING ERR: " + err.Error())
	}
	return string(b)
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
