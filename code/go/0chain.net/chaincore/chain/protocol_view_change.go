package chain

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"0chain.net/smartcontract/stakepool/spenum"
	"github.com/0chain/common/core/currency"

	"go.uber.org/zap"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/httpclientutil"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/util/orderbuffer"
	"0chain.net/core/viper"
	"0chain.net/smartcontract/minersc"
	"github.com/0chain/common/core/logging"
	"github.com/0chain/common/core/util"
)

const (
	scNameAddMiner    = "add_miner"
	scNameAddSharder  = "add_sharder"
	scNameSharderKeep = "sharder_keep"

	scRestAPIGetPhase           = "/getPhase"
	scRestAPIGetMinerList       = "/getMinerList"
	scRestAPIGetSharderList     = "/getSharderList"
	scRestAPIGetSharderKeepList = "/getSharderKeepList"
)

func (c *Chain) isRegistered(ctx context.Context) (is bool) {
	getStatePathFunc := func(n *node.Node) string {
		switch n.Type {
		case node.NodeTypeMiner:
			return minersc.AllMinersKey
		case node.NodeTypeSharder:
			return minersc.AllShardersKey
		default:
			logging.Logger.Error("isRegistered.getStatePath unknown node type",
				zap.String("type", node.NodeTypeNames[n.Type].Value))
		}

		return ""
	}
	getAPIPathFunc := func(n *node.Node) string {
		switch n.Type {
		case node.NodeTypeMiner:
			return scRestAPIGetMinerList
		case node.NodeTypeSharder:
			return scRestAPIGetSharderList
		default:
			logging.Logger.Error("isRegistered.getAPIPath unknown node type",
				zap.String("type", node.NodeTypeNames[n.Type].Value))
		}
		return ""
	}
	return c.isRegisteredEx(ctx, getStatePathFunc, getAPIPathFunc, false)
}

func (c *Chain) isRegisteredEx(ctx context.Context, getStatePath func(n *node.Node) string,
	getAPIPath func(n *node.Node) string, remote bool) bool {
	var (
		allNodeIDs   = minersc.NodeIDs{}
		allNodesList = &minersc.MinerNodes{}
		selfNode     = node.Self.Underlying()
		selfNodeKey  = selfNode.GetKey()
	)

	if c.IsActiveInChain() && !remote {
		var (
			sp  = getStatePath(selfNode)
			err = c.GetBlockStateNode(c.GetLatestFinalizedBlock(), sp, &allNodeIDs)
		)

		if err != nil {
			logging.Logger.Error("failed to get block state node",
				zap.Error(err), zap.String("path", sp))
			return false
		}

		for _, id := range allNodeIDs {
			if id == selfNodeKey {
				return true
			}
		}
	} else {
		var (
			mb       = c.GetCurrentMagicBlock()
			sharders = mb.Sharders.N2NURLs()
			relPath  = getAPIPath(selfNode)
			err      error
		)

		err = httpclientutil.MakeSCRestAPICall(ctx, minersc.ADDRESS, relPath, nil,
			sharders, allNodesList, 1)
		if err != nil {
			logging.Logger.Error("is registered", zap.Error(err))
			return false
		}

		for _, miner := range allNodesList.Nodes {
			if miner == nil {
				continue
			}

			if miner.ID == selfNodeKey {
				return true
			}
		}
	}

	return false
}

// ConfirmTransaction adding a new parameter timeout as we're not sure what all it can break
// without making a lot of changes, to fix a confirmTransaction in SetupSC a new param timeout is added
// if value 0 is passed it'll work like earlier, but anything apart from 0 will result in setting that as timeout
func (c *Chain) ConfirmTransaction(ctx context.Context, t *httpclientutil.Transaction, timeoutSec int64) bool {
	if timeoutSec == 0 {
		timeoutSec = transaction.TXN_TIME_TOLERANCE
	}
	var (
		active = c.IsActiveInChain()
		mb     = c.GetCurrentMagicBlock()

		found, pastTime, notPendingTxn bool
		urls                           []string
		minerUrls                      = make([]string, 0, mb.Miners.Size())
		cctx, cancel                   = context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
	)

	defer func() {
		cancel()
		if !found {
			logging.Logger.Debug("[mvc] invalid txn", zap.String("txn", t.Hash))
		} else {
			logging.Logger.Debug("[mvc] txn confirmed", zap.String("txn", t.Hash))
		}
	}()

	for _, sharder := range mb.Sharders.CopyNodesMap() {
		if !active || sharder.GetStatus() == node.NodeStatusActive {
			urls = append(urls, sharder.GetN2NURLBase())
		}
	}

	for _, m := range mb.Miners.CopyNodesMap() {
		if !active || m.GetStatus() == node.NodeStatusActive {
			minerUrls = append(minerUrls, m.GetN2NURLBase())
		}
	}

	txnPoolCheckingTime := time.NewTicker(3 * time.Second)
	for !found && !pastTime {
		select {
		case <-cctx.Done():
			return false
		case <-txnPoolCheckingTime.C:
			if !node.Self.IsSharder() {
				txn, err := transaction.GetTransactionByHash(ctx, t.Hash)
				if err != nil {
					logging.Logger.Error("[mvc] txn pool checking", zap.Error(err))
					notPendingTxn = true
				} else {
					logging.Logger.Debug("[mvc] txn in pool", zap.Any("txn", txn))
				}
			} else {
				txn, err := httpclientutil.GetTransactionPendingStatus(t.Hash, minerUrls)
				if err != nil {
					logging.Logger.Error("[mvc] txn pool checking", zap.Error(err))
					notPendingTxn = true
				} else {
					logging.Logger.Debug("[mvc] txn in pool", zap.Any("txn", txn))
				}
			}
			// default:
		}

		if !notPendingTxn {
			// in the txn pool, pending
			continue
		}

		txn, err := httpclientutil.GetTransactionStatus(t.Hash, urls, 1)
		if active {
			lfb := c.GetLatestFinalizedBlock()
			pastTime = lfb != nil &&
				!common.WithinTime(int64(lfb.CreationDate), int64(t.CreationDate), transaction.TXN_TIME_TOLERANCE)
		} else {
			blockSummary, err := httpclientutil.GetBlockSummaryCall(urls, 1, false)
			if err != nil {
				logging.Logger.Info("confirm transaction", zap.Bool("confirmation", false))
				return false
			}
			pastTime = blockSummary != nil && !common.WithinTime(int64(blockSummary.CreationDate), int64(t.CreationDate), transaction.TXN_TIME_TOLERANCE)
		}

		found = err == nil && txn != nil
		if found {
			return true
		}

		if notPendingTxn {
			logging.Logger.Error("[mvc] confirm invalid transaction", zap.String("txn", t.Hash))
			// reset the local nonce, set to -1 so that next will be 0 and hence cause nonce sync
			node.Self.SetNonce(-1)
			logging.Logger.Debug("[mvc] nonce, reset nonce after confirming invalid txn", zap.String("txn", t.Hash))
			return false
		}
	}

	logging.Logger.Debug("[mvc] confirm txn", zap.Bool("success", found), zap.Bool("timeout", pastTime))
	return found
}

func (c *Chain) RegisterNode() (*httpclientutil.Transaction, error) {
	selfNode := node.Self.Underlying()
	mn := minersc.NewMinerNode()
	mn.ID = selfNode.GetKey()
	mn.N2NHost = selfNode.N2NHost
	mn.Host = selfNode.Host
	mn.Port = selfNode.Port
	mn.Path = selfNode.Path
	mn.PublicKey = selfNode.PublicKey
	mn.ShortName = selfNode.Description
	mn.BuildTag = selfNode.Info.BuildTag

	// miner SC configurations
	mn.Settings.DelegateWallet = viper.GetString("delegate_wallet")
	mn.Settings.ServiceChargeRatio = viper.GetFloat64("service_charge")
	mn.Settings.MaxNumDelegates = viper.GetInt("number_of_delegates")

	scData := &httpclientutil.SmartContractTxnData{}
	if selfNode.Type == node.NodeTypeMiner {
		mn.ProviderType = spenum.Miner
		scData.Name = scNameAddMiner
	} else if selfNode.Type == node.NodeTypeSharder {
		mn.ProviderType = spenum.Sharder
		scData.Name = scNameAddSharder
	}

	scData.InputArgs = mn

	txn := httpclientutil.NewSmartContractTxn(selfNode.GetKey(), c.ID, selfNode.PublicKey, minersc.ADDRESS)

	mb := c.GetCurrentMagicBlock()
	var minerUrls = mb.Miners.N2NURLs()
	logging.Logger.Debug("Register nodes to",
		zap.Strings("urls", minerUrls),
		zap.String("id", mn.ID))
	err := c.SendSmartContractTxn(txn, scData, minerUrls, mb.Sharders.N2NURLs())
	return txn, err
}

func (c *Chain) estimateTxnFee(txn *httpclientutil.Transaction) (currency.Coin, error) {
	tTxn := &transaction.Transaction{
		TransactionType: txn.TransactionType,
		TransactionData: txn.TransactionData,
		CreationDate:    txn.CreationDate,
		ToClientID:      txn.ToClientID,
		PublicKey:       txn.PublicKey,
	}
	if err := tTxn.ComputeProperties(); err != nil {
		return 0, err
	}

	lfb := c.GetLatestFinalizedBlock()
	if lfb == nil || lfb.ClientState == nil {
		err := errors.New("could not get latest finalized block")
		logging.Logger.Error("could not register miner", zap.Error(err))
		return 0, err
	}

	lfb = lfb.Clone()

	_, fee, err := c.EstimateTransactionCostFee(common.GetRootContext(), lfb, tTxn)
	if err != nil {
		logging.Logger.Error("estimate transaction cost fee failed", zap.Error(err))
		return 0, err
	}

	return fee, nil
}

var txnSendCount int64

func incTxnSendCount(num int64) {
	cout := atomic.AddInt64(&txnSendCount, num)
	logging.Logger.Debug("[mvc] current send txn count", zap.Int64("count", cout))
}

func (c *Chain) SendSmartContractTxn(txn *httpclientutil.Transaction,
	scData *httpclientutil.SmartContractTxnData,
	minerUrls []string,
	sharderUrls []string) error {

	logging.Logger.Info("Jayash SendSmartContractTxn", zap.Any("txn", txn), zap.Any("scData", scData), zap.Any("minerUrls", minerUrls), zap.Any("sharderUrls", sharderUrls))

	// if !httpclientutil.AcquireTxnLock(time.Second) {
	// 	return httpclientutil.ErrTxnSendBusy
	// }
	// logging.Logger.Debug("[mvc] acquire txn lock")
	// incTxnSendCount(1)

	txn.TransactionType = httpclientutil.TxnTypeSmartContract
	if txn.Fee == 0 {
		scBytes, err := json.Marshal(scData)
		if err != nil {
			return err
		}

		txn.TransactionData = string(scBytes)
		fee, err := c.estimateTxnFee(txn)
		if err != nil {
			return err
		}

		txn.Fee = int64(fee)
	}

	nextNonce := node.Self.GetNextNonce()
	if nextNonce == 0 {
		// try get nonce from LFB
		lfb := c.GetLatestFinalizedBlock()
		if lfb != nil {
			var err error
			nextNonce, err = c.GetCurrentSelfNonce(node.Self.Underlying().GetKey(), lfb.ClientState)
			if err != nil && state.ErrInvalidState(err) {
				return err
			}
		}

		logging.Logger.Debug("[mvc] nonce, set lfb nonce in send smart txn", zap.Int64("nonce", nextNonce))
	}
	logging.Logger.Debug("[mvc] nonce, send txn with nonce", zap.Int64("nonce", nextNonce))
	txn.Nonce = nextNonce

	return httpclientutil.SendSmartContractTxn(txn, minerUrls, sharderUrls)
}

func (c *Chain) GetCurrentSelfNonce(minerId datastore.Key, bState util.MerklePatriciaTrieI) (int64, error) {
	s, err := GetStateById(bState, minerId)
	if err != nil {
		if err != util.ErrValueNotPresent {
			logging.Logger.Error("can't get nonce", zap.Error(err))
			return 0, err
		}

		return 1, nil
	}
	logging.Logger.Debug("[mvc] nonce, set nonce in getCurrentSelfNonce", zap.Int64("nonce", s.Nonce))
	node.Self.SetNonce(s.Nonce)
	return node.Self.GetNextNonce(), nil
}

func (c *Chain) RegisterSharderKeep() (result *httpclientutil.Transaction, err2 error) {
	selfNode := node.Self.Underlying()
	if selfNode.Type != node.NodeTypeSharder {
		return nil, errors.New("only sharder")
	}
	mn := minersc.NewMinerNode()
	mn.ID = selfNode.GetKey()
	mn.N2NHost = selfNode.N2NHost
	mn.Host = selfNode.Host
	mn.Port = selfNode.Port
	mn.PublicKey = selfNode.PublicKey
	mn.ShortName = selfNode.Description
	mn.BuildTag = selfNode.Info.BuildTag

	scData := &httpclientutil.SmartContractTxnData{}
	scData.Name = scNameSharderKeep
	scData.InputArgs = mn

	mb := c.GetCurrentMagicBlock()
	var minerUrls = mb.Miners.N2NURLs()

	txn := httpclientutil.NewSmartContractTxn(selfNode.GetKey(), c.ID, selfNode.PublicKey, minersc.ADDRESS)
	err := c.SendSmartContractTxn(txn, scData, minerUrls, mb.Sharders.N2NURLs())
	return txn, err
}

func (c *Chain) IsRegisteredSharderKeep(ctx context.Context, remote bool) bool {
	return c.isRegisteredEx(ctx,
		func(n *node.Node) string {
			if typ := n.Type; typ == node.NodeTypeSharder {
				return minersc.ShardersKeepKey
			}
			return ""
		},
		func(n *node.Node) string {
			if typ := n.Type; typ == node.NodeTypeSharder {
				return scRestAPIGetSharderKeepList
			}
			return ""
		}, remote)
}

//
// DKG Phase tracking
//

// PhaseEvent represents DKG phase event.
type PhaseEvent struct {
	Phase    minersc.PhaseNode // current phase node
	Sharders bool              // is obtained from sharders or another sharders
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
}

// The isZero returns true if given value is zero. It can be replaced with
// reflect.IsZero, but Golang updating required (there is some problems in
// docker builds).
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

// The seriHigh represents util.Serializable value with it 'highness'.
type seriHigh struct {
	seri util.Serializable // value
	high int64             // value highness
}

// The mostConsensus chooses from the list value with most repetitions.
func mostConsensus(list []seriHigh) (elem util.Serializable) {

	type valueCount struct {
		value util.Serializable
		count int
	}

	var cons = make(map[string]*valueCount, len(list))
	for _, sh := range list {
		var str = string(sh.seri.Encode())
		if vc, ok := cons[str]; ok {
			vc.count++
		} else {
			cons[str] = &valueCount{value: sh.seri, count: 1}
		}
	}

	var most int
	for _, vc := range cons {
		if vc.count > most {
			elem = vc.value // choose this value
			most = vc.count // update the most
		}
	}

	return
}

// The getHighestOnly receive unordered list of seriHigh values and extracts
// only top highest values from it (e.g. values with top, the same, highness).
func getHighestOnly(list []seriHigh) []seriHigh {

	// sort by the highness first
	sort.Slice(list, func(i, j int) bool {
		return list[i].high > list[j].high
	})

	// exclude all except top highness
	var top int64 = math.MinInt64
	for i, sh := range list {
		if sh.high > top {
			top = sh.high
		}
		if sh.high < top {
			list = list[:i] // keep the list until previous value only
			break
		}
	}

	return list
}

// The makeSCRESTAPICall is internal for GetFromSharders.
func makeSCRESTAPICall(ctx context.Context, address, relative string, sharder string,
	seri util.Serializable, collect chan util.Serializable) {

	if err := httpclientutil.MakeSCRestAPICall(ctx, address, relative, nil,
		[]string{sharder}, seri, 1); err != nil {
		logging.Logger.Error("requesting phase node from sharder",
			zap.String("sharder", sharder),
			zap.Error(err))
		collect <- nil
		return
	}
	collect <- seri // regardless error
}

// The GetFromSharders used to obtains an information from sharders using REST
// API interface of a SC. About the arguments:
//
//   - address    -- SC address
//   - relative   -- REST API relative path (e.g. handler name)
//   - sharders   -- list of sharders to request from (N2N URLs)
//   - newFunc    -- factory to create new value of type you want to request
//   - rejectFunc -- filter to reject some values, can't be nil (feel free
//     to modify)
//   - highFunc   -- function that returns value highness; used to choose
//     highest values
//
// TODO (sfxdx): to trust or not to trust, that is the question
//
// Security note. Following its initial design we are using REST API call here.
// And chooses highest (first) and most consensus (second) value. Thus, a one
// of sharders in the list can provide illegal highest value to break all
// the mechanics here.
//
// Other side, for 3 sharders, two of which is behind the active one. The two
// sharders behind will never give correct resultC (but give most consensus).
// Thus we can't use most consensual response. Pew-pew.
//
// Probably, block requesting verifying, and syncing its state and then
// extracting phase can help. But it's not 100% (slow or doesn't work for now ?).
func GetFromSharders(ctx context.Context, address, relative string, sharders []string,
	newFunc func() util.Serializable,
	rejectFunc func(seri util.Serializable) bool,
	highFunc func(seri util.Serializable) int64) (
	got util.Serializable) {

	t := time.Now()
	defer func() {
		logging.Logger.Debug("GetFromSharders", zap.Duration("duration", time.Since(t)))
	}()

	wg := &sync.WaitGroup{}
	var collect = make(chan util.Serializable, len(sharders))
	for _, sharder := range sharders {
		wg.Add(1)
		go func(sh string) {
			defer wg.Done()
			cctx, cancel := context.WithTimeout(ctx, 3*time.Second)
			defer cancel()
			makeSCRESTAPICall(cctx, address, relative, sh, newFunc(), collect)
		}(sharder)
	}

	wg.Wait()
	close(collect)
	var list = make([]seriHigh, 0, len(sharders))
	for val := range collect {
		if !isZero(val) && !rejectFunc(val) {
			list = append(list, seriHigh{
				seri: val,
				high: highFunc(val),
			})
		}
	}

	return mostConsensus(getHighestOnly(list))
}

// PhaseEvents notifications channel.
func (c *Chain) PhaseEvents() *orderbuffer.OrderBuffer {
	return c.phaseEvents
}

// The SendPhaseNode optimistically sends given phase to phase trackers.
// It never blocks. Skipping event if no one can accept it at this time.
func (c *Chain) SendPhaseNode(ctx context.Context, pe PhaseEvent) {
	c.phaseEvents.Add(pe.Phase.StartRound, pe)
}

// The GetPhaseFromSharders obtains minersc.PhaseNode from sharders and sends
// it to phases events channel. It chooses highest most consensus phase.
// E.g. phase with highest starting round (the highness is in priority) and
// it there is collisions, then it chooses phase with most consensus.
//
// The methods optimistically (non-blocking) sends the resultC to internal
// phaseEvetns channel.
//
// There is no a worker uses the GetPhaseFromSharders in the chaincore/chain.
// Both, miners and sharders should trigger it themselves.
func (c *Chain) GetPhaseFromSharders(ctx context.Context) {

	var (
		cmb = c.GetCurrentMagicBlock()
		got util.Serializable
	)

	brief := c.GetLatestFinalizedMagicBlockBrief()
	if brief == nil {
		return
	}
	shardersN2NURLs := brief.ShardersN2NURLs
	got = GetFromSharders(ctx, minersc.ADDRESS, scRestAPIGetPhase,
		shardersN2NURLs, func() util.Serializable {
			return new(minersc.PhaseNode)
		}, func(val util.Serializable) bool {
			if pn, ok := val.(*minersc.PhaseNode); ok {
				return pn.StartRound < cmb.StartingRound
			}
			return true // reject
		}, func(val util.Serializable) (high int64) {
			if pn, ok := val.(*minersc.PhaseNode); ok {
				return pn.StartRound // its start round is the highness
			}
			return // zero
		})

	var phase, ok = got.(*minersc.PhaseNode)
	if !ok {
		logging.Logger.Error("get_dkg_phase_from_sharders -- no phases given")
		return
	}

	logging.Logger.Debug("dkg_process -- phase from sharders",
		zap.String("phase", phase.Phase.String()),
		zap.Int64("start_round", phase.StartRound),
		zap.Int64("restarts", phase.Restarts))

	const isGivenFromSharders = true // it is given from sharders 100%
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	c.SendPhaseNode(ctx, PhaseEvent{Phase: *phase, Sharders: isGivenFromSharders})
}

// The GetPhaseOfBlock extracts and returns Miner SC phase node for given block.
func (c *Chain) GetPhaseOfBlock(b *block.Block) (*minersc.PhaseNode, error) {
	var pn minersc.PhaseNode
	err := c.GetBlockStateNode(b, minersc.PhaseKey, &pn)
	if err != nil {
		return nil, err
	}

	return &pn, nil
}

func (c *Chain) GetRegisterNodesList(b *block.Block) (minersc.NodeIDs, error) {
	var mids minersc.NodeIDs
	minerKey, ok := minersc.GetRegisterNodeKey(spenum.Miner)
	if !ok {
		return nil, fmt.Errorf("invalid node type: %s", spenum.Miner)
	}

	err := c.GetBlockStateNode(b, minerKey, &mids)
	if err != nil && err != util.ErrValueNotPresent {
		return nil, err
	}

	sharderKey, ok := minersc.GetRegisterNodeKey(spenum.Sharder)
	if !ok {
		return nil, fmt.Errorf("invalid node type: %s", spenum.Sharder)
	}

	var sids minersc.NodeIDs
	err = c.GetBlockStateNode(b, sharderKey, &sids)
	if err != nil && err != util.ErrValueNotPresent {
		return nil, err
	}

	return append(mids, sids...), nil
}
