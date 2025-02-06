package miner

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"net/http"
	"path/filepath"
	"reflect"
	"runtime"
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
	"github.com/0chain/common/core/logging"

	"0chain.net/smartcontract/minersc"

	hbls "github.com/herumi/bls-go-binary/bls"

	"go.uber.org/zap"
)

const (
	// transactions
	scNameContributeMpk = "contributeMpk"
	scNamePublishShares = "shareSignsOrShares"
	scNameWait          = "wait"
	// REST API requests
	scRestAPIGetDKGMiners  = "/getDkgList"
	scRestAPIGetMinersMPKS = "/getMpksList"
	scRestAPIGetMagicBlock = "/getMagicBlock"
)

// PhaseFunc represents local VC function returns optional
// Miner SC transactions.
type PhaseFunc func(ctx context.Context, lfb *block.Block,
	lfmb *block.MagicBlock) (tx *httpclientutil.Transaction,
	err error)

// view change process controlling
type viewChangeProcess struct {
	sync.Mutex

	phaseFuncs    map[minersc.Phase]PhaseFunc
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
	vcp.phaseFuncs = map[minersc.Phase]PhaseFunc{
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

func (mc *Chain) DKGProcess(ctx context.Context) {
	logging.Logger.Info("manual view change process started!!")

	var (
		newPhaseEvent chain.PhaseEvent

		// start round of the accepted phase
		phaseStartRound    int64
		hadTxnAndConfirmed bool

		// flag indicating whether previous share phase failed
		// and should be retried with the already generated shares
		retrySharePhase bool
	)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			pe, ok := mc.PhaseEvents().Pop()
			if !ok {
				time.Sleep(200 * time.Millisecond)
				continue
			}

			newPhaseEvent = pe.Data.(chain.PhaseEvent)
		}

		pn := newPhaseEvent.Phase
		// only retry if new phase is share phase
		retrySharePhase = pn.Phase == minersc.Share && retrySharePhase

		notMoveRound := pn.StartRound == phaseStartRound

		if !retrySharePhase && notMoveRound && hadTxnAndConfirmed {
			// skip if not retry share phase, and not the move round, also previously, txn was confirmed,
			continue
		}

		if notMoveRound && hadTxnAndConfirmed && !retrySharePhase {
			continue // phase already accepted
		}

		logging.Logger.Debug("[mvc] process: in phase",
			zap.String("phase", pn.Phase.String()),
			zap.Int64("start_round", pn.StartRound),
			zap.Int64("phase start round", phaseStartRound))

		lfb := mc.GetLatestFinalizedBlock()
		lfbPhaseNode, err := mc.GetPhaseOfBlock(lfb)
		if err != nil {
			logging.Logger.Error("[mvc] update finalized block - get phase of block failed", zap.Error(err))
			continue
		}

		if lfbPhaseNode.Phase > pn.Phase {
			logging.Logger.Error("[mvc] lfb phase > pn phase - skip",
				zap.String("lfb_phase", lfbPhaseNode.Phase.String()),
				zap.String("pn_phase", pn.Phase.String()))
			continue
		}

		phaseFunc, ok := mc.viewChangeProcess.phaseFuncs[pn.Phase]
		if !ok {
			logging.Logger.Debug("[mvc] dkg process: no such phase func",
				zap.String("phase", pn.Phase.String()))
			continue
		}

		phaseFuncName := getFunctionName(phaseFunc)

		// only go through if pn.Phase is expected
		if !(pn.Phase == minersc.Start ||
			pn.Phase == mc.CurrentPhase()+1 || retrySharePhase) {
			logging.Logger.Debug(
				"[mvc] dkg process: jumping over a phase; skip and wait for restart",
				zap.String("current_phase", mc.CurrentPhase().String()),
				zap.String("next_phase", pn.Phase.String()))
			mc.SetCurrentPhase(minersc.Unknown)
			continue
		}

		lfmb := mc.GetCurrentMagicBlock()
		if lfmb == nil {
			logging.Logger.Error("[mvc] dkg process: can't get lfmb")
			continue
		}

		logging.Logger.Debug("[mvc] dkg process: run phase function",
			zap.String("name", phaseFuncName),
			zap.String("current_phase", mc.CurrentPhase().String()),
			zap.String("next_phase", pn.Phase.String()),
			zap.Int64("lfb round", lfb.Round))

		txn, err := phaseFunc(ctx, lfb, lfmb)
		if err != nil {
			logging.Logger.Error("[mvc] dkg process: phase func failed",
				zap.String("current_phase", mc.CurrentPhase().String()),
				zap.String("next_phase", pn.Phase.String()),
				zap.Error(err),
			)
			if pn.Phase != minersc.Share {
				continue
			}
			retrySharePhase = true
		}

		if txn == nil || mc.ConfirmTransaction(ctx, txn, 30) {
			hadTxnAndConfirmed = true
			prevPhase := mc.CurrentPhase()
			mc.SetCurrentPhase(pn.Phase)
			phaseStartRound = pn.StartRound
			mb := mc.GetLatestMagicBlock()
			logging.Logger.Debug("[mvc] dkg process: moved phase",
				zap.String("prev_phase", prevPhase.String()),
				zap.String("current_phase", mc.CurrentPhase().String()),
				zap.Int64("magic block number", mb.MagicBlockNumber),
			)
		} else {
			hadTxnAndConfirmed = false
			logging.Logger.Debug("[mvc] dkg process: failed to move phase",
				zap.String("current_phase", mc.CurrentPhase().String()),
				zap.Any("next_phase", pn),
				zap.Any("txn", txn))
		}
	}
}

func getFunctionName(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}

func (vcp *viewChangeProcess) clearViewChange() {
	logging.Logger.Warn("[mvc] clear view change")
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
	*block.MagicBlock) (*httpclientutil.Transaction, error) {

	mc.viewChangeProcess.Lock()
	defer mc.viewChangeProcess.Unlock()

	mc.viewChangeProcess.clearViewChange()
	return nil, nil
}

func (vcp *viewChangeProcess) isNeedCreateSijs() (ok bool) {
	return vcp.viewChangeDKG != nil &&
		vcp.viewChangeDKG.GetSijLen() < vcp.viewChangeDKG.T
}

func (mc *Chain) getMinersMpks(ctx context.Context, lfb *block.Block, mb *block.MagicBlock,
) (mpks *block.Mpks, err error) {
	mpks = block.NewMpks()
	err = mc.GetBlockStateNode(lfb, minersc.MinersMPKKey, mpks)
	if err != nil {
		return
	}

	return mpks, nil
}

func (mc *Chain) getDKGMiners(ctx context.Context, lfb *block.Block, mb *block.MagicBlock) (
	dmn *minersc.DKGMinerNodes, err error) {
	dmn = minersc.NewDKGMinerNodes()
	err = mc.GetBlockStateNode(lfb, minersc.DKGMinersKey, dmn)
	if err != nil {
		return
	}
	return dmn, nil
}

func (mc *Chain) createSijs(ctx context.Context, lfb *block.Block, mb *block.MagicBlock) (err error) {

	if !mc.viewChangeProcess.isDKGSet() {
		return common.NewError("createSijs", "DKG is not set")
	}

	if !mc.viewChangeProcess.isNeedCreateSijs() {
		return // doesn't need to create them
	}

	var mpks *block.Mpks
	if mpks, err = mc.getMinersMpks(ctx, lfb, mb); err != nil {
		logging.Logger.Error("can't share", zap.Error(err))
		return
	}

	var dmn *minersc.DKGMinerNodes
	if dmn, err = mc.getDKGMiners(ctx, lfb, mb); err != nil {
		logging.Logger.Error("can't share", zap.Error(err))
		return
	}

	logging.Logger.Debug("[mvc] createSijs", zap.Int("mpks num", len(mpks.Mpks)))

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
		if err := n.SetPublicKey(v.PublicKey); err != nil {
			return err
		}
		n.Description = v.ShortName
		n.Type = node.NodeTypeMiner
		n.Info.BuildTag = v.BuildTag
		n.SetStatus(node.NodeStatusActive)
		if err := node.Setup(n); err != nil {
			logging.Logger.Error("[mvc] createSijs failed to setup node", zap.Error(err))
			return err
		}
		node.RegisterNode(n)
	}

	mc.viewChangeProcess.mpks = mpks // set

	var foundSelf = false

	for k := range mc.mpks.GetMpks() {
		id := bls.ComputeIDdkg(k)
		share, err := mc.viewChangeDKG.ComputeDKGKeyShare(id)
		if err != nil {
			logging.Logger.Error("can't compute secret share", zap.Error(err))
			return err
		}
		if k == node.Self.Underlying().GetKey() {
			if err := mc.viewChangeDKG.AddSecretShare(id, share.GetHexString(), false); err != nil {
				return err
			}

			if err := StoreDKGKey(ctx, &block.DKGKey{
				KeyShare:      share.GetHexString(),
				MagicBlockNum: mc.viewChangeDKG.MagicBlockNumber}); err != nil {
				logging.Logger.Error("can't store dkg key", zap.Error(err))
				return err
			}
			logging.Logger.Debug("add dkg key to store",
				zap.String("key", share.GetHexString()),
				zap.Int64("mb number", mc.viewChangeDKG.MagicBlockNumber))
			foundSelf = true
		}
	}
	if !foundSelf {
		logging.Logger.Error("failed to add secret key for self",
			zap.Int64("lfb_round", lfb.Round))
	}

	return nil
}

func (vcp *viewChangeProcess) isDKGSet() bool {
	return vcp.viewChangeDKG != nil
}

//
//                                 S H A R E
//

func (mc *Chain) sendSijsPrepare(ctx context.Context, lfb *block.Block, mb *block.MagicBlock) (
	sendTo []string, err error) {
	mc.viewChangeProcess.Lock()
	defer mc.viewChangeProcess.Unlock()

	if !mc.viewChangeProcess.isDKGSet() {
		return nil, common.NewError("dkg_not_set", "send_sijs: DKG is not set")
	}

	var dkgMiners *minersc.DKGMinerNodes
	if dkgMiners, err = mc.getDKGMiners(ctx, lfb, mb); err != nil {
		return // error
	}

	var selfNodeKey = node.Self.Underlying().GetKey()
	if _, ok := dkgMiners.SimpleNodes[selfNodeKey]; !mc.isDKGSet() || !ok {
		logging.Logger.Error("[mvc] failed to send sijs", zap.Bool("dkg_set", mc.isDKGSet()),
			zap.Bool("ok", ok))
		return // (nil, nil)
	}

	if err = mc.createSijs(ctx, lfb, mb); err != nil {
		logging.Logger.Error("[mvc] failed to create sijs", zap.Error(err))
		return // error
	}

	// we haven't to make sure all nodes are registered, because the
	// createSijs registers them; and after a restart ('deregister')
	// we have to restart DKG for this miner, since secret key is lost

	for key := range dkgMiners.SimpleNodes {
		if key == selfNodeKey {
			continue // don't send to self
		}
		if _, ok := mc.viewChangeProcess.shareOrSigns.ShareOrSigns[key]; !ok {
			sendTo = append(sendTo, key)
		}
	}

	return
}

func (mc *Chain) getNodeSij(nodeID hbls.ID) (*hbls.SecretKey, bool) {
	mc.viewChangeProcess.Lock()
	defer mc.viewChangeProcess.Unlock()

	if mc.viewChangeProcess.viewChangeDKG == nil {
		return nil, false
	}

	k, ok := mc.viewChangeProcess.viewChangeDKG.GetKeyShare(nodeID)
	if !ok {
		return nil, false
	}

	return &k, true
}

func (mc *Chain) setSecretShares(shareOrSignSuccess map[string]*bls.DKGKeyShare) {

	mc.viewChangeProcess.Lock()
	defer mc.viewChangeProcess.Unlock()

	for id, share := range shareOrSignSuccess {
		if share != nil {
			mc.viewChangeProcess.shareOrSigns.ShareOrSigns[id] = share
		}
	}
}

func (mc *Chain) GetMagicBlockFromSC(ctx context.Context, lfb *block.Block, mb *block.MagicBlock) (
	magicBlock *block.MagicBlock, err error) {
	magicBlock = block.NewMagicBlock()
	err = mc.GetBlockStateNode(lfb, minersc.MagicBlockKey, magicBlock)
	if err != nil {
		return nil, err
	}

	return
}

func (mc *Chain) waitTransaction(mb *block.MagicBlock) (
	tx *httpclientutil.Transaction, err error) {

	var data = new(httpclientutil.SmartContractTxnData)
	data.Name = scNameWait

	var selfNode = node.Self.Underlying()

	tx = httpclientutil.NewSmartContractTxn(selfNode.GetKey(), mc.ID, selfNode.PublicKey, minersc.ADDRESS)
	err = mc.SendSmartContractTxn(tx, data, mb.Miners.N2NURLs(), mb.Sharders.N2NURLs())
	return
}

// NextViewChange round stored in view change protocol (RAM).
func (vcp *viewChangeProcess) NextViewChange() (round int64) {
	vcp.nvcmx.Lock()
	defer vcp.nvcmx.Unlock()

	return vcp.nextViewChange
}

// SetNextViewChange round into view change protocol (RAM).
func (vcp *viewChangeProcess) SetNextViewChange(round int64) {
	vcp.nvcmx.Lock()
	defer vcp.nvcmx.Unlock()

	vcp.nextViewChange = round
}

// ========================================================================== //
//                            other VC helpers                                //
// ========================================================================== //

// MB save / load

func StoreMagicBlock(ctx context.Context, magicBlock *block.MagicBlock) (
	err error) {

	var (
		data = block.NewMagicBlockData(magicBlock)
		emd  = data.GetEntityMetadata()
		dctx = ememorystore.WithEntityConnection(ctx, emd)
	)
	defer ememorystore.Close(dctx)

	if err = data.Write(dctx); err != nil {
		return
	}

	var connection = ememorystore.GetEntityCon(dctx, emd)
	logging.Logger.Info("[mvc] store mb", zap.Int64("mb number", magicBlock.MagicBlockNumber))
	return connection.Commit()
}

func LoadMagicBlock(ctx context.Context, id string) (mb *block.MagicBlock,
	err error) {

	var mbd = datastore.GetEntity("magicblockdata").(*block.MagicBlockData)
	mbd.ID = id

	var (
		emd  = mbd.GetEntityMetadata()
		dctx = ememorystore.WithEntityConnection(ctx, emd)
	)
	defer ememorystore.Close(dctx)

	if err = mbd.Read(dctx, mbd.GetKey()); err != nil {
		return
	}
	mb = mbd.MagicBlock
	return
}

// DKG save / load

func StoreDKGKey(ctx context.Context, dkgKey *block.DKGKey) error {
	var (
		dkgKeyData     = block.NewDKGKeyData(dkgKey)
		dkgKeyMetadata = dkgKeyData.GetEntityMetadata()
		dctx           = ememorystore.WithEntityConnection(ctx, dkgKeyMetadata)
	)

	defer ememorystore.Close(dctx)

	if err := dkgKeyData.Write(dctx); err != nil {
		return err
	}

	con := ememorystore.GetEntityCon(dctx, dkgKeyMetadata)
	return con.Commit()
}

func LoadDKGKey(ctx context.Context, mbNum int64) (dkgKey *block.DKGKey, err error) {
	id := strconv.FormatInt(mbNum, 10)
	dkgKeyData := datastore.GetEntity("dkgkeydata").(*block.DKGKeyData)
	dkgKeyData.ID = id

	var (
		emd  = dkgKeyData.GetEntityMetadata()
		dctx = ememorystore.WithEntityConnection(ctx, emd)
	)
	defer ememorystore.Close(dctx)

	if err = dkgKeyData.Read(dctx, dkgKeyData.GetKey()); err != nil {
		return
	}

	return dkgKeyData.DKGKey, nil
}

// StoreDKGSummary in DB.
func StoreDKGSummary(ctx context.Context, summary *bls.DKGSummary) (err error) {
	var (
		dkgSummaryMetadata = summary.GetEntityMetadata()
		dctx               = ememorystore.WithEntityConnection(ctx,
			dkgSummaryMetadata)
	)
	defer ememorystore.Close(dctx)

	if err = summary.Write(dctx); err != nil {
		return
	}

	var con = ememorystore.GetEntityCon(dctx, dkgSummaryMetadata)
	return con.Commit()
}

// LoadDKGSummary loads DKG summary by stored DKG (that stores DKG summary).
func LoadDKGSummary(ctx context.Context, id string) (dkgs *bls.DKGSummary,
	err error) {

	dkgs = datastore.GetEntity("dkgsummary").(*bls.DKGSummary)
	dkgs.ID = id
	var (
		dkgSummaryMetadata = dkgs.GetEntityMetadata()
		dctx               = ememorystore.WithEntityConnection(ctx,
			dkgSummaryMetadata)
	)
	defer ememorystore.Close(dctx)
	err = dkgs.Read(dctx, dkgs.GetKey())
	return
}

// ReadDKGSummaryFile obtains dkg summary from JSON file with given path.
func ReadDKGSummaryFile(path string) (dkgs *bls.DKGSummary, err error) {
	dkgs = &bls.DKGSummary{SecretShares: make(map[string]string)}
	if path == "" {
		return nil, common.NewError("Error reading dkg file", "path is blank")
	}

	if ext := filepath.Ext(path); ext != ".json" {
		return nil, common.NewError("Error reading dkg file", fmt.Sprintf("unexpected dkg summary file extension: %q, expected '.json'", ext))
	}

	var b []byte
	if b, err = os.ReadFile(path); err != nil {
		return nil, common.NewError("Error reading dkg file", fmt.Sprintf("reading dkg summary file: %v", err))
	}

	if err = dkgs.Decode(b); err != nil {
		return nil, common.NewError("Error reading dkg file", fmt.Sprintf("decoding dkg summary file: %v", err))
	}

	logging.Logger.Info("read dkg summary file", zap.String("ID", dkgs.ID))
	return
}

//
// Latest MB from store
//

// func LoadLatestMB(ctx context.Context, lfbRound, mbNumber int64) (mb *block.MagicBlock, err error) {
// 	if mbNumber > 0 {
// 		mbStr := strconv.FormatInt(mbNumber, 10)
// 		mb, err = LoadMagicBlock(ctx, mbStr)
// 		if err != nil {
// 			logging.Logger.Error("load_latest_mb", zap.Error(err), zap.Int64("mb number", mbNumber))
// 			return
// 		}
// 		logging.Logger.Info("[mvc] find latest MB by magic bock number", zap.Int64("mb number", mbNumber))
// 		return mb, nil
// 	}

// 	var (
// 		mbemd = datastore.GetEntityMetadata("magicblockdata")
// 		rctx  = ememorystore.WithEntityConnection(ctx, mbemd)
// 		conn  = ememorystore.GetEntityCon(rctx, mbemd)
// 	)
// 	defer ememorystore.Close(rctx)

// 	iter := conn.Conn.NewIterator(conn.ReadOptions)
// 	defer iter.Close()
// 	// the first time the hardfork is happened
// 	var data = mbemd.Instance().(*block.MagicBlockData)
// 	iter.SeekToLast() // from last

// 	if !iter.Valid() {
// 		return nil, util.ErrValueNotPresent
// 	}

// 	if err = datastore.FromJSON(iter.Value().Data(), data); err != nil {
// 		return nil, common.NewErrorf("load_latest_mb",
// 			"decoding error: %v, key: %q", err, string(iter.Key().Data()))
// 	}

// 	mb = data.MagicBlock
// 	logging.Logger.Info("[mvc] seek to the last in MB store", zap.Int64("mb number", mb.MagicBlockNumber))
// 	return
// }

//
// Setup miner (initialization).
//

// SetupLatestAndPreviousMagicBlocks used to be sure miner has latest and
// previous MB and corresponding DKG. The previous MB can be useless in
// some cases but this method just makes sure it is.
func (mc *Chain) SetupLatestAndPreviousMagicBlocks(ctx context.Context) {
	if !mc.ChainConfig.IsViewChangeEnabled() {
		return
	}

	logging.Logger.Info("setup latest and previous fmbs")
	lfmb := mc.GetLatestFinalizedMagicBlock(ctx)
	if lfmb == nil || lfmb.Sharders == nil || lfmb.Miners == nil {
		return
	}

	if err := mc.SetDKGSFromStore(ctx, lfmb.MagicBlock); err != nil {
		logging.Logger.Warn("set dkgs from store failed", zap.Error(err))
	}

	if lfmb.MagicBlockNumber <= 1 {
		mc.UpdateMagicBlocks(lfmb)
		return // no previous MB is expected
	}

	pfmb := mc.GetLatestFinalizedMagicBlockRound(lfmb.StartingRound - 1)
	if pfmb == nil {
		logging.Logger.Error("can't get lfmb")
		return
	}

	if pfmb.MagicBlock.Hash == lfmb.MagicBlock.PreviousMagicBlockHash {
		if err := mc.SetDKGSFromStore(ctx, lfmb.MagicBlock); err != nil {
			logging.Logger.Warn("set dkgs from store failed", zap.Error(err))
		}
		mc.UpdateMagicBlocks(pfmb, lfmb)
		return
	}

	var err error
	pfmb, err = mc.GetBlock(ctx, lfmb.LatestFinalizedMagicBlockHash)
	if err == nil && pfmb.MagicBlock != nil &&
		pfmb.MagicBlock.Hash == lfmb.MagicBlock.PreviousMagicBlockHash {
		if err := mc.SetDKGSFromStore(ctx, pfmb.MagicBlock); err != nil {
			logging.Logger.Warn("set dkgs from store failed", zap.Error(err))
		}

		mc.UpdateMagicBlocks(pfmb, lfmb)
		return
	}

	pfmb, err = httpclientutil.FetchMagicBlockFromSharders(
		ctx, lfmb.Sharders.N2NURLs(), lfmb.MagicBlockNumber-1, func(*block.Block) bool { return true })
	if err != nil {
		logging.Logger.Error("getting previous FMB from sharder", zap.Error(err),
			zap.Int64("num", lfmb.MagicBlockNumber-1))
		return // error
	}

	if pfmb != nil && pfmb.MagicBlock == nil {
		logging.Logger.Error("getting previous FMB from sharder, has no magic block",
			zap.Int64("num", lfmb.MagicBlockNumber-1))
		return
	}

	if pfmb.MagicBlock.GetHash() != lfmb.MagicBlock.PreviousMagicBlockHash {
		logging.Logger.Error("getting previous FMB from sharder",
			zap.String("err", "invalid hash"),
			zap.Int64("num", lfmb.MagicBlockNumber-1))
		return // error
	}

	if err := mc.SetDKGSFromStore(ctx, pfmb.MagicBlock); err != nil {
		logging.Logger.Warn("set dkgs from store", zap.Error(err))
	}
	mc.UpdateMagicBlocks(pfmb, lfmb) // ok
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
		logging.Logger.Error("[mvc] sign share failed, dkg is not set")
		return nil, common.NewError("sign_share", "DKG is not set")
	}

	var (
		mpks        = mc.viewChangeProcess.mpks.GetMpks()
		lmpks, dkgt = len(mpks), mc.viewChangeProcess.viewChangeDKG.T
	)
	if lmpks < dkgt {
		logging.Logger.Error("[mvc] sign share failed, not enough mpks yet",
			zap.Int("mpks num", lmpks), zap.Int("dkg t", dkgt))
		return nil, common.NewErrorf("sign_share", "don't have enough mpks"+
			" yet, l mpks (%d) < dkg t (%d)", lmpks, dkgt)
	}

	var (
		message = datastore.GetEntityMetadata("dkg_share").Instance().(*bls.DKGKeyShare)
		share   bls.Key
	)

	if err = share.SetHexString(secShare); err != nil {
		logging.Logger.Error("[mvc] failed to set hex string", zap.Error(err))
		return nil, common.NewErrorf("sign_share",
			"setting hex string: %v", err)
	}

	mpk, err := bls.ConvertStringToMpk(mpks[nodeID].Mpk)
	if err != nil {
		return nil, err
	}

	if !mc.viewChangeProcess.viewChangeDKG.ValidateShare(mpk, share) {
		logging.Logger.Error("[mvc] failed to verify dkg share",
			zap.String("share", secShare), zap.String("node_id", nodeID))
		return nil, common.NewError("sign_share", "failed to verify DKG share")
	}

	err = mc.viewChangeProcess.viewChangeDKG.AddSecretShare(bls.ComputeIDdkg(nodeID), secShare, false)
	if err != nil {
		logging.Logger.Error("[mvc] failed to add secret share", zap.Error(err))
		return nil, common.NewErrorf("sign_share", "adding secret share: %v", err)
	}

	message.Message = encryption.Hash(secShare)
	message.Sign, err = node.Self.Sign(message.Message)
	if err != nil {
		logging.Logger.Error("[mvc] failed to sign DKG share message", zap.Error(err))
		return nil, common.NewErrorf("sign_share",
			"signing DKG share message: %v", err)
	}

	logging.Logger.Debug("[mvc] sign share request success",
		zap.String("share", secShare),
		zap.String("node_id", nodeID),
		zap.Int("shares num", mc.viewChangeProcess.viewChangeDKG.GetSecretSharesSize()))

	return afterSignShareRequestHandler(message, nodeID)
}

func (mc *Chain) NextViewChangeOfBlock(lfb *block.Block) (round int64, err error) {
	if !mc.ChainConfig.IsViewChangeEnabled() {
		return lfb.LatestFinalizedMagicBlockRound, nil
	}

	// miner SC global node is not created yet, but firs block creates it
	if lfb.Round < 1 {
		return 0, nil
	}

	var gn minersc.GlobalNode
	err = mc.GetBlockStateNode(lfb, minersc.GlobalNodeKey, &gn)
	if err != nil {
		logging.Logger.Error("block_next_vc -- can't get miner SC global node",
			zap.Error(err), zap.Int64("lfb", lfb.Round),
			zap.Bool("is_state", lfb.IsStateComputed()),
			zap.Bool("is_init", lfb.ClientState != nil),
			zap.Any("state", lfb.ClientStateHash))
		return 0, common.NewErrorf("block_next_vc",
			"can't get miner SC global node, lfb: %d, error: %v (%s)",
			lfb.Round, err, lfb.Hash)
	}

	gnb := gn.MustBase()
	logging.Logger.Debug("block_next_vc -- ok", zap.Int64("lfb", lfb.Round),
		zap.Int64("nvc", gnb.ViewChange))

	return gnb.ViewChange, nil // got it
}
