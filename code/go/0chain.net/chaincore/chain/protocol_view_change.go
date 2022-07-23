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
	"time"

	"0chain.net/chaincore/currency"

	"go.uber.org/zap"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/client"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/httpclientutil"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/logging"
	"0chain.net/core/memorystore"
	"0chain.net/core/util"
	"0chain.net/core/viper"
	"0chain.net/smartcontract/minersc"
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

// RegisterClient registers client on BC.
func (c *Chain) RegisterClient() {
	if node.Self.Underlying().Type == node.NodeTypeMiner {
		var (
			clientMetadataProvider = datastore.GetEntityMetadata("client")
			ctx                    = memorystore.WithEntityConnection(
				common.GetRootContext(), clientMetadataProvider)
		)
		defer memorystore.Close(ctx)
		ctx = datastore.WithAsyncChannel(ctx, client.ClientEntityChannel)
		_, err := client.PutClient(ctx, &node.Self.Underlying().Client)
		if err != nil {
			panic(err)
		}
	}

	nodeBytes, err := json.Marshal(node.Self.Underlying().Client.Clone())
	if err != nil {
		logging.Logger.DPanic("Encode self node failed", zap.Error(err))
	}

	var (
		mb               = c.GetCurrentMagicBlock()
		miners           = mb.Miners.CopyNodesMap()
		registered       = 0
		thresholdByCount = config.GetThresholdCount()
		consensus        = int(math.Ceil((float64(thresholdByCount) / 100) *
			float64(len(miners))))
	)

	if consensus > len(miners) {
		logging.Logger.DPanic(fmt.Sprintf("number of miners %d is not enough"+
			" relative to the threshold parameter %d%%(%d)", len(miners),
			thresholdByCount, consensus))
	}

	for registered < consensus {
		for key, miner := range miners {
			body, err := httpclientutil.SendPostRequest(
				miner.GetN2NURLBase()+httpclientutil.RegisterClient, nodeBytes,
				"", "", nil,
			)
			if err != nil {
				logging.Logger.Error("error in register client",
					zap.Error(err),
					zap.Any("body", body),
					zap.Int("registered", registered),
					zap.Int("consensus", consensus))
			} else {
				delete(miners, key)
				registered++
			}
		}
		time.Sleep(httpclientutil.SleepBetweenRetries * time.Millisecond)
	}

	logging.Logger.Info("register client success",
		zap.Int("registered", registered),
		zap.Int("consensus", consensus))
}

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
		allNodesList = &minersc.MinerNodes{}
		selfNode     = node.Self.Underlying()
		selfNodeKey  = selfNode.GetKey()
	)

	if c.IsActiveInChain() && !remote {

		var (
			sp  = getStatePath(selfNode)
			err = c.GetBlockStateNode(c.GetLatestFinalizedBlock(), sp, allNodesList)
		)

		if err != nil {
			logging.Logger.Error("failed to get block state node",
				zap.Any("error", err), zap.String("path", sp))
			return false
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
			logging.Logger.Error("is registered", zap.Any("error", err))
			return false
		}
	}

	for _, miner := range allNodesList.Nodes {
		if miner == nil {
			continue
		}

		if miner.ID == selfNodeKey {
			return true
		}
	}

	return false
}

func (c *Chain) ConfirmTransaction(ctx context.Context, t *httpclientutil.Transaction) bool {
	var (
		active = c.IsActiveInChain()
		mb     = c.GetCurrentMagicBlock()

		found, pastTime bool
		urls            []string
		cctx, cancel    = context.WithTimeout(ctx, time.Duration(transaction.TXN_TIME_TOLERANCE)*time.Second)
	)

	defer cancel()

	for _, sharder := range mb.Sharders.CopyNodesMap() {
		if !active || sharder.GetStatus() == node.NodeStatusActive {
			urls = append(urls, sharder.GetN2NURLBase())
		}
	}

	for !found && !pastTime {
		select {
		case <-cctx.Done():
			return false
		default:
		}

		txn, err := httpclientutil.GetTransactionStatus(t.Hash, urls, 1)
		if active {
			lfb := c.GetLatestFinalizedBlock()
			pastTime = lfb != nil &&
				!common.WithinTime(int64(lfb.CreationDate), int64(t.CreationDate), transaction.TXN_TIME_TOLERANCE)
		} else {
			blockSummary, err := httpclientutil.GetBlockSummaryCall(urls, 1, false)
			if err != nil {
				logging.Logger.Info("confirm transaction", zap.Any("confirmation", false))
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

func (c *Chain) RegisterNode() (*httpclientutil.Transaction, error) {
	selfNode := node.Self.Underlying()
	txn := httpclientutil.NewTransactionEntity(selfNode.GetKey(),
		c.ID, selfNode.PublicKey)

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

	var err error
	mn.Settings.MinStake, err = currency.ParseZCN(viper.GetFloat64("min_stake"))
	if err != nil {
		return nil, err
	}
	mn.Settings.MaxStake, err = currency.ParseZCN(viper.GetFloat64("max_stake"))
	if err != nil {
		return nil, err
	}
	mn.Geolocation = minersc.SimpleNodeGeolocation{
		Latitude:  viper.GetFloat64("latitude"),
		Longitude: viper.GetFloat64("longitude"),
	}
	scData := &httpclientutil.SmartContractTxnData{}
	if selfNode.Type == node.NodeTypeMiner {
		scData.Name = scNameAddMiner
	} else if selfNode.Type == node.NodeTypeSharder {
		scData.Name = scNameAddSharder
	}

	scData.InputArgs = mn

	txn.ToClientID = minersc.ADDRESS
	txn.PublicKey = selfNode.PublicKey
	mb := c.GetCurrentMagicBlock()
	var minerUrls = mb.Miners.N2NURLs()
	logging.Logger.Debug("Register nodes to", zap.Strings("urls", minerUrls))
	err = httpclientutil.SendSmartContractTxn(txn, minersc.ADDRESS, 0, 0, scData, minerUrls, mb.Sharders.N2NURLs())
	return txn, err
}

func (c *Chain) RegisterSharderKeep() (result *httpclientutil.Transaction, err2 error) {
	selfNode := node.Self.Underlying()
	if selfNode.Type != node.NodeTypeSharder {
		return nil, errors.New("only sharder")
	}
	txn := httpclientutil.NewTransactionEntity(selfNode.GetKey(),
		c.ID, selfNode.PublicKey)

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

	txn.ToClientID = minersc.ADDRESS
	txn.PublicKey = selfNode.PublicKey
	mb := c.GetCurrentMagicBlock()
	var minerUrls = mb.Miners.N2NURLs()
	err := httpclientutil.SendSmartContractTxn(txn, minersc.ADDRESS, 0, 0, scData, minerUrls, mb.Sharders.N2NURLs())
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
//     - address    -- SC address
//     - relative   -- REST API relative path (e.g. handler name)
//     - sharders   -- list of sharders to request from (N2N URLs)
//     - newFunc    -- factory to create new value of type you want to request
//     - rejectFunc -- filter to reject some values, can't be nil (feel free
//                     to modify)
//     - highFunc   -- function that returns value highness; used to choose
//                     highest values
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
		logging.Logger.Debug("GetFromSharders", zap.Any("duration", time.Since(t)))
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
func (c *Chain) PhaseEvents() (pe chan PhaseEvent) {
	return c.phaseEvents
}

// The sendPhase optimistically sends given phase to phase trackers.
// It never blocks. Skipping event if no one can accept it at this time.
func (c *Chain) sendPhase(pn minersc.PhaseNode, sharders bool) {
	select {
	case c.phaseEvents <- PhaseEvent{Phase: pn, Sharders: sharders}:
	default:
		// never block here, be optimistic
	}
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
	c.sendPhase(*phase, isGivenFromSharders)
}

// The GetPhaseOfBlock extracts and returns Miner SC phase node for given block.
func (c *Chain) GetPhaseOfBlock(b *block.Block) (pn minersc.PhaseNode,
	err error) {

	err = c.GetBlockStateNode(b, minersc.PhaseKey, &pn)
	if err != nil && err != util.ErrValueNotPresent {
		err = fmt.Errorf("get_block_phase -- can't get: %v, block %d",
			err, b.Round)
		return
	}

	if err == util.ErrValueNotPresent {
		err = nil // not a real error, Miner SC just is not started (yet)
		return
	}

	return // ok
}
