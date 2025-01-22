package chain

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"0chain.net/core/config"
	"github.com/0chain/common/core/currency"
	"github.com/0chain/common/core/statecache"

	"0chain.net/chaincore/node"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/threshold/bls"
	"0chain.net/smartcontract/dbs/event"

	metrics "github.com/rcrowley/go-metrics"
	"go.uber.org/zap"

	"0chain.net/chaincore/block"
	bcstate "0chain.net/chaincore/chain/state"
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/smartcontract"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/smartcontract/minersc"
	"github.com/0chain/common/core/logging"
	"github.com/0chain/common/core/util"
)

// SmartContractExecutionTimer - a metric that tracks the time it takes to execute a smart contract txn
var SmartContractExecutionTimer metrics.Timer
var StateComputationTimer metrics.Histogram
var EventsComputationTimer metrics.Histogram

func init() {
	SmartContractExecutionTimer = metrics.GetOrRegisterTimer("sc_execute_timer", nil)
	StateComputationTimer = metrics.NewHistogram(metrics.NewUniformSample(1024))
	EventsComputationTimer = metrics.NewHistogram(metrics.NewUniformSample(1024))
}

var ErrWrongNonce = common.NewError("wrong_nonce", "nonce of sender is not valid")

/*ComputeState - compute the state for the block */
func (c *Chain) ComputeState(ctx context.Context, b *block.Block, waitC ...chan struct{}) (err error) {
	return c.ComputeBlockStateWithLock(ctx, func() error {
		//check whether we already computed it
		if b.IsStateComputed() {
			return nil
		}
		return c.computeState(ctx, b, waitC...)
	})
}

// ComputeOrSyncState - try to compute state and if there is an error, just sync it
func (c *Chain) ComputeOrSyncState(ctx context.Context, b *block.Block) error {
	err := c.ComputeState(ctx, b)
	if err != nil {
		bsc, err := c.getBlockStateChange(b)
		if err != nil {
			return err
		}
		if bsc != nil {
			if err = c.ApplyBlockStateChange(b, bsc); err != nil {
				logging.Logger.Error("compute state - applying state change",
					zap.Int64("round", b.Round), zap.String("block", b.Hash),
					zap.Error(err))
				return err
			}
		}
		if !b.IsStateComputed() {
			logging.Logger.Error("compute state - state change error",
				zap.Int64("round", b.Round), zap.String("block", b.Hash),
				zap.Error(err))
			return err
		}
	}
	return nil
}

func (c *Chain) computeState(ctx context.Context, b *block.Block, waitC ...chan struct{}) error {
	timer := time.Now()
	err := b.ComputeState(ctx, c, waitC...)
	StateComputationTimer.Update(time.Since(timer).Microseconds())
	return err
}

// SaveChanges - persist the state changes
func (c *Chain) SaveChanges(ctx context.Context, b *block.Block) error {
	if !b.IsStateComputed() {
		err := errors.New("block state not computed")
		logging.Logger.Error("save changes failed", zap.Error(err),
			zap.Int64("round", b.Round),
			zap.String("hash", b.Hash))
		return err
	}
	cctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	return b.SaveChanges(cctx, c)
}

func (c *Chain) rebaseState(lfb *block.Block) {
	if lfb.ClientState == nil {
		return
	}
	c.stateMutex.Lock()
	defer c.stateMutex.Unlock()
	ndb := lfb.ClientState.GetNodeDB()
	if ndb != c.stateDB {
		lfb.ClientState.SetNodeDB(c.stateDB)
		logging.Logger.Debug("finalize round - rebased current state db",
			zap.Int64("round", lfb.Round), zap.String("block", lfb.Hash),
			zap.String("state hash", util.ToHex(lfb.ClientState.GetRoot())))
	}
}

// ExecuteSmartContract - executes the smart contract for the transaction
func (c *Chain) ExecuteSmartContract(
	ctx context.Context,
	txn *transaction.Transaction,
	balances bcstate.StateContextI) (string, error) {

	type result struct {
		output string
		err    error
	}

	var (
		ts      = time.Now()
		sct     = time.NewTimer(c.SmartContractTimeout())
		resultC = make(chan result, 1)
	)

	if node.Self.Type == node.NodeTypeSharder {
		// give more times for sharders to compute state, as sharders are required to be run
		// as full node, so each block should not be executed failed due to timeout
		sct = time.NewTimer(3 * time.Minute)
	}

	go func() {
		output, err := smartcontract.ExecuteSmartContract(txn, balances)
		resultC <- result{output: output, err: err}
	}()
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-sct.C:
		return "", transaction.ErrSmartContractContext
	case r := <-resultC:
		SmartContractExecutionTimer.Update(time.Since(ts))
		if len(balances.GetMissingNodeKeys()) > 0 {
			if r.err == nil || !cstate.ErrInvalidState(r.err) {
				logging.Logger.Error("execute smart contract - find missing nodes, not return from calling",
					zap.Any("txn", txn))
			} else {
				logging.Logger.Error("execute smart contract - find missing nodes, return node not found error",
					zap.Error(r.err),
					zap.Any("output", r.output),
					zap.Any("txn", txn))
			}
			return "", util.ErrNodeNotFound
		}

		if cstate.ErrInvalidState(r.err) {
			logging.Logger.Debug("execute smart contract - return node not found error directly",
				zap.Any("txn", txn))
		}

		return r.output, r.err
	}
}

// UpdateState - update the state of the transaction w.r.t the given block.
// Note, don't call this from within state computation logic since their is
// already a lock on StateMutex. This API is for someone reading the state from
// outside the protocol without already holding a lock on StateMutex. The block
// starts off with the state from the prior block and as transactions are
// processed into a block, the state gets updated. If a state can't be updated
// (e.g low balance), then a false is returned so that the transaction will not
// make it into the block.
func (c *Chain) UpdateState(ctx context.Context,
	b *block.Block,
	bState util.MerklePatriciaTrieI,
	txn *transaction.Transaction,
	blockStateCache *statecache.BlockCache,
	waitC ...chan struct{},
) ([]event.Event, error) {
	c.stateMutex.Lock()
	defer c.stateMutex.Unlock()
	return c.updateState(ctx, b, bState, txn, blockStateCache, waitC...)
}

type SyncReplyC struct {
	sync   bool
	replyC []chan struct{}
}

// SyncNodesOption function for setting node syncing option
type SyncNodesOption func(*SyncReplyC)

// WithSync enable synching missing nodes if any
func WithSync() SyncNodesOption {
	return func(s *SyncReplyC) {
		s.sync = true
	}
}

// WithNotifyC subscribe to channel that will be notified when missing nodes syncing is done
func WithNotifyC(replyC ...chan struct{}) SyncNodesOption {
	return func(s *SyncReplyC) {
		s.replyC = replyC
	}
}

func (c *Chain) EstimateTransactionCost(ctx context.Context,
	b *block.Block, txn *transaction.Transaction, opts ...SyncNodesOption) (int, error) {
	var (
		qbc         = statecache.NewQueryBlockCache(c.GetStateCache(), b.Hash)
		tbc         = statecache.NewTransactionCache(qbc)
		clientState = CreateTxnMPT(b.ClientState, tbc) // begin transaction
		sctx        = c.NewStateContext(b, clientState, txn, nil)
	)

	switch txn.TransactionType {

	case transaction.TxnTypeSmartContract:
		var scData sci.SmartContractTransactionData
		dataBytes := []byte(txn.TransactionData)
		err := json.Unmarshal(dataBytes, &scData)
		if err != nil {
			logging.Logger.Error("Error while decoding the JSON from transaction",
				zap.String("input", txn.TransactionData), zap.Error(err))
			return math.MaxInt32, err
		}

		cost, err := smartcontract.EstimateTransactionCost(txn, scData, sctx)
		if missingKeys := sctx.GetMissingNodeKeys(); len(missingKeys) > 0 {
			syncOpts := &SyncReplyC{}
			for _, opt := range opts {
				opt(syncOpts)
			}

			logging.Logger.Error("Internal error while estimate transaction cost",
				zap.Error(util.ErrNodeNotFound),
				zap.Int64("round", b.Round),
				zap.String("block", b.Hash))
			if syncOpts.sync {
				c.SyncMissingNodes(b.Round, missingKeys, syncOpts.replyC...)
			}
			return math.MaxInt32, util.ErrNodeNotFound
		}

		return cost, err

	case transaction.TxnTypeSend:
		return c.ChainConfig.TxnTransferCost(), nil

	case transaction.TxnTypeLockIn:
		return 0, nil

	case transaction.TxnTypeData:
		return 0, nil

	case transaction.TxnTypeStorageWrite:
		return 0, nil

	case transaction.TxnTypeStorageRead:
		return 0, nil

	default:
		logging.Logger.Error("Invalid transaction type", zap.Int("txn type", txn.TransactionType))
		return math.MaxInt32, fmt.Errorf("invalid transaction type: %v", txn.TransactionType)
	}
}

func (c *Chain) EstimateTransactionFeeLFB(ctx context.Context,
	txn *transaction.Transaction,
	opts ...SyncNodesOption) (currency.Coin, error) {
	lfb := c.GetLatestFinalizedBlock()
	if lfb == nil {
		return 0, errors.New("LFB not ready yet")
	}
	lfb = lfb.Clone()

	_, fee, err := c.EstimateTransactionCostFee(ctx, lfb, txn, opts...)
	return fee, err
}

func (c *Chain) EstimateTransactionCostFee(ctx context.Context,
	b *block.Block,
	txn *transaction.Transaction,
	opts ...SyncNodesOption) (int, currency.Coin, error) {
	cost, err := c.EstimateTransactionCost(ctx, b, txn, opts...)
	if err != nil {
		return 0, 0, err
	}

	if txn.SmartContractData == nil {
		logging.Logger.Warn("txn properties not computed", zap.Any("txn", txn))
		if err := txn.ComputeProperties(); err != nil {
			return 0, 0, err
		}
	}

	if _, ok := c.ChainConfig.TxnExempt()[txn.FunctionName]; ok {
		txn.IsExempt = true
		return cost, 0, nil
	}

	logging.Logger.Debug("estimate transaction cost fee",
		zap.Int("cost", cost),
		zap.String("txn hash", txn.Hash),
		zap.String("txn", txn.TransactionData))

	maxFee := c.ChainConfig.MaxTxnFee()

	zcn := float64(cost) / float64(c.ChainConfig.TxnCostFeeCoeff())
	parseZCN, err := currency.ParseZCN(zcn)
	if err != nil {
		return cost, maxFee, nil
	}

	if maxFee > 0 && parseZCN > maxFee {
		return cost, maxFee, nil
	}

	return cost, parseZCN, nil
}

func (c *Chain) GetTransactionCostFeeTable(ctx context.Context,
	b *block.Block,
	opts ...SyncNodesOption) map[string]map[string]int64 {

	var (
		qbc         = statecache.NewQueryBlockCache(c.GetStateCache(), b.Hash)
		tbc         = statecache.NewTransactionCache(qbc)
		clientState = CreateTxnMPT(b.ClientState, tbc) // begin transaction
		sctx        = c.NewStateContext(b, clientState, &transaction.Transaction{}, nil)
	)

	table := smartcontract.GetTransactionCostTable(sctx)

	table["transfer"] = map[string]int{"transfer": c.ChainConfig.TxnTransferCost()}

	for _, t := range table {
		for name := range c.ChainConfig.TxnExempt() {
			if _, ok := t[name]; ok {
				t[name] = 0
			}
		}
	}

	fees := make(map[string]map[string]int64)
	for sc, t := range table {
		fees[sc] = make(map[string]int64, len(t))
		for f, cost := range t {
			zcn := float64(cost) / float64(c.ChainConfig.TxnCostFeeCoeff())
			parseZCN, err := currency.ParseZCN(zcn)
			if err != nil {
				fees[sc][f] = int64(c.ChainConfig.MaxTxnFee())
				continue
			}

			if c.ChainConfig.MaxTxnFee() > 0 && parseZCN > c.ChainConfig.MaxTxnFee() {
				fees[sc][f] = int64(c.ChainConfig.MaxTxnFee())
			} else {
				fees[sc][f] = int64(parseZCN)
			}

		}
	}

	return fees
}

// NewStateContext creation helper.
func (c *Chain) NewStateContext(
	b *block.Block,
	s util.MerklePatriciaTrieI,
	txn *transaction.Transaction,
	eventDb *event.EventDb,
) (balances *bcstate.StateContext) {
	return bcstate.NewStateContext(b, s, txn,
		c.GetMagicBlock,
		func() *block.Block {
			return c.GetLatestFinalizedMagicBlock(context.Background())
		},
		c.GetCurrentMagicBlock,
		c.GetSignatureScheme,
		c.GetLatestFinalizedBlock,
		c.getDKGSummary,
		c.SetDKG,
		eventDb,
	)
}

func (c *Chain) getDKGSummary(magicBlockNum int64) (*bls.DKGSummary, error) {
	return LoadDKGSummary(common.GetRootContext(), magicBlockNum)
}

func (c *Chain) updateState(ctx context.Context,
	b *block.Block,
	bState util.MerklePatriciaTrieI,
	txn *transaction.Transaction,
	blockStateCache *statecache.BlockCache,
	waitC ...chan struct{}) (es []event.Event, err error) {
	// check if the block's ClientState has root value
	_, err = bState.GetNodeDB().GetNode(bState.GetRoot())
	if err != nil {
		return nil, common.NewErrorf("update_state_failed",
			"block state root is incorrect, err: %v, block hash: %v, state hash: %v, root: %v, round: %d",
			err, b.Hash, util.ToHex(b.ClientStateHash), util.ToHex(bState.GetRoot()), b.Round)
	}

	if txn.Value > config.MaxTokenSupply {
		return nil, errors.New("invalid transaction value, exceeds max token supply")
	}

	var (
		txnStateCache = statecache.NewTransactionCache(blockStateCache)
		clientState   = CreateTxnMPT(bState, txnStateCache) // begin transaction
		sctx          = c.NewStateContext(b, clientState, txn, nil)
		startRoot     = sctx.GetState().GetRoot()
	)

	defer func() {
		if err == nil {
			// commit transaction state cache
			txnStateCache.Commit()
		}

		if bcstate.ErrInvalidState(err) {
			c.SyncMissingNodes(b.Round, sctx.GetMissingNodeKeys(), waitC...)
		}
	}()

	if err = c.validateNonce(sctx, txn.ClientID, txn.Nonce); err != nil {
		return nil, err
	}

	// checks if the client has enough funds to pay for transaction before heavy computations are executed
	if err = sctx.Validate(); err != nil {
		return nil, err
	}

	switch txn.TransactionType {
	case transaction.TxnTypeSmartContract:
		t := time.Now()
		output, err := c.ExecuteSmartContract(ctx, txn, sctx)
		switch err {
		//internal errors
		case context.DeadlineExceeded, transaction.ErrSmartContractContext, util.ErrNodeNotFound:
			logging.Logger.Error("Error executing the SC, internal error",
				zap.Error(err),
				zap.String("scname", txn.FunctionName),
				zap.String("block", b.Hash),
				zap.String("begin client state", util.ToHex(startRoot)),
				zap.String("prev block", b.PrevBlock.Hash),
				zap.Duration("time_spent", time.Since(t)),
				zap.Any("txn", txn))
			//return original error, to handle upwards
			return nil, err
		case context.Canceled:
			logging.Logger.Debug("Error executing the SC, internal error",
				zap.Error(err),
				zap.String("scname", txn.FunctionName),
				zap.String("block", b.Hash),
				zap.String("begin client state", util.ToHex(startRoot)),
				zap.String("prev block", b.PrevBlock.Hash),
				zap.Duration("time_spent", time.Since(t)),
				zap.Any("txn", txn))
			//return original error, to handle upwards
			return nil, err
		default:
			if err != nil {
				if bcstate.ErrInvalidState(err) {
					logging.Logger.Error("Error executing the SC, internal error",
						zap.Error(err),
						zap.String("scname", txn.FunctionName),
						zap.String("block", b.Hash),
						zap.String("begin client state", util.ToHex(startRoot)),
						zap.String("prev block", b.PrevBlock.Hash),
						zap.Duration("time_spent", time.Since(t)),
						zap.Any("txn", txn))
					return nil, err
				}

				logging.Logger.Debug("Error executing the SC, chargeable error",
					zap.Error(err),
					zap.String("client id", txn.ClientID),
					zap.String("block", b.Hash),
					zap.String("begin client state", util.ToHex(startRoot)),
					zap.String("prev block", b.PrevBlock.Hash),
					zap.Duration("time_spent", time.Since(t)),
					zap.Any("txn", txn))

				//refresh client state context, so all changes made by broken smart contract are rejected, it will be used to add fee
				txnStateCache = statecache.NewTransactionCache(blockStateCache)
				clientState = CreateTxnMPT(bState, txnStateCache) // begin transaction
				sctx = c.NewStateContext(b, clientState, txn, nil)
				// records chargeable error event
				sctx.EmitError(err)

				output = err.Error()
				txn.Status = transaction.TxnError
			}
		}
		txn.TransactionOutput = output
		if _, ok := StartToFinalizeTxnTypeTimer[txn.FunctionName]; !ok {
			StartToFinalizeTxnTypeTimer[txn.FunctionName] = metrics.GetOrRegisterTimer(txn.FunctionName, nil)
		}
		StartToFinalizeTxnTypeTimer[txn.FunctionName].Update(time.Since(t))
		mptCacheHits, mptCacheMiss := txnStateCache.Stats()
		logging.Logger.Info("SC executed",
			zap.String("client id", txn.ClientID),
			zap.String("block", b.Hash),
			zap.Int64("round", b.Round),
			zap.String("prev_state_hash", util.ToHex(b.PrevBlock.ClientStateHash)),
			zap.String("txn_hash", txn.Hash),
			zap.Int64("txn_nonce", txn.Nonce),
			zap.String("txn_func", txn.FunctionName),
			zap.Int("txn_status", txn.Status),
			zap.Duration("txn_exec_time", time.Since(t)),
			zap.String("begin client state", util.ToHex(startRoot)),
			zap.String("current_root", util.ToHex(sctx.GetState().GetRoot())),
			zap.Int64("mpt_cache_hit", mptCacheHits),
			zap.Int64("mpt_cache_miss", mptCacheMiss),
			zap.String("output", output))
	case transaction.TxnTypeData:
	case transaction.TxnTypeSend:
		// check src balance
		balance, err := sctx.GetClientBalance(txn.ClientID)
		if err != nil {
			return nil, err
		}

		if balance < txn.Fee+txn.Value {
			return nil, errors.New("insufficient balance to send")
		}

		err = sctx.AddTransfer(state.NewTransfer(txn.ClientID, txn.ToClientID, txn.Value))
		if err != nil {
			logging.Logger.Error("Failed to add transfer",
				zap.Int("txn type", txn.TransactionType),
				zap.String("transaction_ClientID", txn.ClientID),
				zap.String("minersc_address", minersc.ADDRESS),
				zap.Any("state_balance", txn.Fee),
				zap.Any("current_root", sctx.GetState().GetRoot()))
			return nil, err
		}
	default:
		logging.Logger.Error("Invalid transaction type", zap.Int("txn type", txn.TransactionType))
		return nil, fmt.Errorf("invalid transaction type: %v", txn.TransactionType)
	}

	if c.ChainConfig.IsFeeEnabled() {
		err = sctx.AddTransfer(state.NewTransfer(txn.ClientID, minersc.ADDRESS, txn.Fee))
		if err != nil {
			logging.Logger.Error("Failed to add transfer",
				zap.Int("txn type", txn.TransactionType),
				zap.String("transaction_ClientID", txn.ClientID),
				zap.String("minersc_address", minersc.ADDRESS),
				zap.Any("state_balance", txn.Fee))
			return nil, err
		}
	}

	ue := make(map[string]*event.User)
	for _, transfer := range sctx.GetTransfers() {
		tEvents, err := c.transferAmountWithAssert(sctx, transfer.ClientID, transfer.ToClientID, transfer.Amount)
		if err != nil {
			logging.Logger.Error("Failed to transfer amount",
				zap.Int("txn type", txn.TransactionType),
				zap.String("txn data", txn.TransactionData),
				zap.String("transfer_ClientID", transfer.ClientID),
				zap.String("to_ClientID", transfer.ToClientID),
				zap.Any("amount", transfer.Amount),
				zap.Error(err))
			return nil, err
		}
		for _, e := range tEvents {
			ue[e.UserID] = e
		}
	}

	for _, signedTransfer := range sctx.GetSignedTransfers() {
		tEvents, err := c.transferAmountWithAssert(sctx, signedTransfer.ClientID,
			signedTransfer.ToClientID, signedTransfer.Amount)
		if err != nil {
			logging.Logger.Error("Failed to process signed transfer",
				zap.String("signedTransfer_ClientID", signedTransfer.ClientID),
				zap.String("signedTransfer_to_ClientID", signedTransfer.ToClientID),
				zap.Any("signedTransfer_amount", signedTransfer.Amount))
			return nil, err
		}
		for _, e := range tEvents {
			ue[e.UserID] = e
		}
	}

	u, err := c.incrementNonce(sctx, txn.ClientID)
	if err != nil {
		logging.Logger.Error("update nonce error", zap.Error(err),
			zap.Any("transaction", txn),
			zap.String("clientID", txn.ClientID))
		return nil, err
	}

	if u != nil {
		ue[u.UserID] = u
	}

	for _, e := range ue {
		c.emitUserEvent(sctx, e)
	}

	// commit transaction
	if err = bState.MergeMPTChanges(clientState); err != nil {
		if state.DebugTxn() {
			logging.Logger.DPanic("update state - merge mpt error",
				zap.Int64("round", b.Round), zap.String("block", b.Hash),
				zap.Any("txn", txn), zap.Error(err))
		}

		logging.Logger.Error("error committing txn", zap.Error(err))
		return nil, err
	}

	//if status is not set
	if txn.Status == 0 {
		txn.Status = transaction.TxnSuccess
	}

	return sctx.GetEvents(), nil
}

func sumOfFromToBalance(sctx bcstate.StateContextI, from, to string) (currency.Coin, error) {
	ofb, err := sctx.GetClientBalance(from)
	if err != nil && err != util.ErrValueNotPresent {
		return 0, err
	}
	otb, err := sctx.GetClientBalance(to)
	if err != nil && err != util.ErrValueNotPresent {
		return 0, err
	}

	bal, err := currency.AddCoin(ofb, otb)
	if err != nil {
		return 0, fmt.Errorf("failed to sum up balances: %v", err)
	}

	return bal, nil
}

func (c *Chain) transferAmountWithAssert(sctx bcstate.StateContextI,
	fromClient, toClient datastore.Key, amount currency.Coin) (eus []*event.User, err error) {
	originBalance, err := sumOfFromToBalance(sctx, fromClient, toClient)
	if err != nil {
		return nil, err
	}

	tEvents, err := c.transferAmount(sctx, fromClient, toClient, amount)
	if err != nil {
		return nil, err
	}

	afterBalance, err := sumOfFromToBalance(sctx, fromClient, toClient)
	if err != nil {
		return nil, err
	}

	if originBalance != afterBalance {
		logging.Logger.Panic("Transfer assertion failed",
			zap.String("txn", sctx.GetTransaction().Hash),
			zap.String("from", fromClient),
			zap.String("to", toClient),
			zap.Any("amount", amount),
			zap.Any("origin_balance", originBalance),
			zap.Any("after_balance", afterBalance))
	}

	return tEvents, nil
}

/*
* transferAmount - transfers balance from one account to another
*   when there is an error getting the state of the from or to account (other than no value), the error is simply returned back
*   when there is an error inserting/deleting the state of the from or to account, this results in fatal error when state is enabled
 */
func (c *Chain) transferAmount(sctx bcstate.StateContextI, fromClient, toClient datastore.Key,
	amount currency.Coin) (eus []*event.User, err error) {
	if amount == 0 {
		return nil, nil
	}
	if fromClient == toClient {
		return nil, common.InvalidRequest("from and to client should be different for balance transfer")
	}

	defer func() {
		if bcstate.ErrInvalidState(err) {
			c.SyncMissingNodes(sctx.GetBlock().Round, sctx.GetMissingNodeKeys())
		}
	}()

	var (
		b   = sctx.GetBlock()
		txn = sctx.GetTransaction()
	)

	fs, err := sctx.GetClientState(fromClient)
	if !isValid(err) {
		return nil, err
	}

	if fs.Balance < amount {
		logging.Logger.Error("transfer amount - insufficient balance",
			zap.Any("balance", fs.Balance),
			zap.Any("transfer", amount),
			zap.String("from", fromClient))
		return nil, transaction.ErrInsufficientBalance
	}

	ts, err := sctx.GetClientState(toClient)
	if !isValid(err) {
		return nil, err
	}

	if err := sctx.SetStateContext(fs); err != nil {
		logging.Logger.Error("transfer amount - set state context failed",
			zap.Int64("round", b.Round),
			zap.String("state txn hash", fs.TxnHash),
			zap.Error(err))
		return nil, err
	}

	fromBalance, err := currency.MinusCoin(fs.Balance, amount)
	if err != nil {
		return nil, fmt.Errorf("transfer tokens from client failed: %v", err)
	}
	fs.Balance = fromBalance
	_, err = sctx.SetClientState(fromClient, fs)
	if err != nil {
		return nil, err
	}

	if err := sctx.SetStateContext(ts); err != nil {
		logging.Logger.Error("transfer amount - set state context failed",
			zap.Int64("round", b.Round),
			zap.String("state txn hash", fs.TxnHash),
			zap.Error(err))
		return nil, err
	}

	toBalance, err := currency.AddCoin(ts.Balance, amount)
	if err != nil {
		logging.Logger.Error("transfer amount - error",
			zap.Int64("round", b.Round),
			zap.String("block", b.Hash),
			zap.Any("txn", txn), zap.Error(err))
		return nil, fmt.Errorf("transfer tokens to client failed: %v", err)
	}

	ts.Balance = toBalance

	_, err = sctx.SetClientState(toClient, ts)
	if err != nil {
		return nil, err
	}

	return []*event.User{stateToUser(fromClient, fs), stateToUser(toClient, ts)}, nil
}

//nolint:unused
func (c *Chain) mintAmountWithAssert(sctx bcstate.StateContextI, toClient datastore.Key, amount currency.Coin) (eu *event.User, err error) {
	originBalance, err := sctx.GetClientBalance(toClient)
	if err != nil && err != util.ErrValueNotPresent {
		return nil, err
	}

	tEvent, err := c.mintAmount(sctx, toClient, amount)
	if err != nil {
		return nil, err
	}

	afterBalance, err := sctx.GetClientBalance(toClient)
	if err != nil {
		return nil, err
	}

	if originBalance+amount != afterBalance {
		logging.Logger.Panic("Mint assertion failed",
			zap.String("txn", sctx.GetTransaction().Hash),
			zap.String("to", toClient),
			zap.Any("amount", amount),
			zap.Any("origin_balance", originBalance),
			zap.Any("after_balance", afterBalance))
	}

	return tEvent, nil
}

//nolint:unused
func (c *Chain) mintAmount(sctx bcstate.StateContextI, toClient datastore.Key, amount currency.Coin) (eu *event.User, err error) {
	if amount == 0 {
		return nil, nil
	}

	var (
		b   = sctx.GetBlock()
		txn = sctx.GetTransaction()
	)

	defer func() {
		if bcstate.ErrInvalidState(err) {
			c.SyncMissingNodes(sctx.GetBlock().Round, sctx.GetMissingNodeKeys())
		}
	}()

	ts, err := sctx.GetClientState(toClient)
	if !isValid(err) {
		return nil, common.NewError("mint_amount - get state", err.Error())
	}

	if err := sctx.SetStateContext(ts); err != nil {
		logging.Logger.Error("transfer amount - set state context failed",
			zap.String("txn hash", ts.TxnHash),
			zap.Error(err))
		return nil, err
	}

	toBalance, err := currency.AddCoin(ts.Balance, amount)
	if err != nil {
		logging.Logger.Error("transfer amount - error",
			zap.Int64("round", b.Round),
			zap.String("block", b.Hash),
			zap.Any("txn", txn),
			zap.Error(err))
		return nil, err
	}
	ts.Balance = toBalance

	_, err = sctx.SetClientState(toClient, ts)
	if err != nil {
		return nil, common.NewError("mint_amount - insert", err.Error())
	}

	return stateToUser(toClient, ts), nil
}

func (c *Chain) validateNonce(sctx bcstate.StateContextI, fromClient datastore.Key, txnNonce int64) error {
	s, err := sctx.GetClientState(fromClient)
	if !isValid(err) {
		return err
	}
	nonce := int64(0)
	if s != nil {
		nonce = s.Nonce
	}
	if nonce+1 != txnNonce {
		b := sctx.GetBlock()
		logging.Logger.Error("validate nonce - error",
			zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Int64("txn_nonce", txnNonce),
			zap.Int64("local_nonce", s.Nonce), zap.Error(err))
		return ErrWrongNonce
	}

	return nil
}

func (c *Chain) incrementNonce(sctx bcstate.StateContextI, fromClient datastore.Key) (*event.User, error) {
	s, err := sctx.GetClientState(fromClient)
	if !isValid(err) {
		return nil, err
	}

	if s == nil {
		s = &state.State{}
	}
	if err := sctx.SetStateContext(s); err != nil {
		return nil, err
	}

	if s.Nonce == 0 {
		c.emitUniqueAddressEvent(sctx, s)
	}

	s.Nonce += 1
	if _, err := sctx.SetClientState(fromClient, s); err != nil {
		return nil, err
	}
	logging.Logger.Debug("Updating nonce",
		zap.String("client", fromClient),
		zap.Int64("new_nonce", s.Nonce))

	return stateToUser(fromClient, s), nil
}

func CreateTxnMPT(mpt util.MerklePatriciaTrieI, cache *statecache.TransactionCache) util.MerklePatriciaTrieI {
	tdb := util.NewLevelNodeDB(util.NewMemoryNodeDB(), mpt.GetNodeDB(), false)
	tmpt := util.NewMerklePatriciaTrie(tdb, mpt.GetVersion(), mpt.GetRoot(), cache)
	return tmpt
}

func GetStateById(clientState util.MerklePatriciaTrieI, clientID string) (*state.State, error) {
	if clientState == nil {
		return nil, common.NewError("GetStateById", "client state does not exist")
	}
	s := &state.State{}
	s.Balance = currency.Coin(0)
	err := clientState.GetNodeValue(util.Path(clientID), s)
	if err != nil {
		if err != util.ErrValueNotPresent {
			return nil, err
		}
		return s, err
	}
	return s, nil
}

/*
GetState - Get the state of a client w.r.t a block. Note, don't call this from within state computation logic
since block.GetStateValue uses a RLock on the StateMutex. This API is for someone reading the state from outside
the protocol without already holding a lock on StateMutex
*/
func (c *Chain) GetState(b *block.Block, clientID string) (*state.State, error) {
	c.stateMutex.RLock()
	defer c.stateMutex.RUnlock()
	st := &state.State{}
	err := b.ClientState.GetNodeValue(util.Path(clientID), st)
	if err != nil {
		if !b.IsStateComputed() {
			return nil, common.NewError("state_not_yet_computed", "State is not yet computed")
		}
		ps := c.GetPruneStats()
		if ps != nil && ps.MissingNodes > 0 {
			return nil, common.NewError("state_not_synched", "State sync is not yet complete")
		}
		return nil, err
	}
	return st, nil
}

func isValid(err error) bool {
	if err == nil {
		return true
	}
	if err == util.ErrValueNotPresent {
		return true
	}
	return false
}

func userToState(u *event.User) *state.State {
	return &state.State{
		TxnHash: u.TxnHash,
		Balance: u.Balance,
		Round:   u.Round,
		Nonce:   u.Nonce,
	}
}

func stateToUser(clientID string, s *state.State) *event.User {
	return &event.User{
		UserID:  clientID,
		TxnHash: s.TxnHash,
		Balance: s.Balance,
		Round:   s.Round,
		Nonce:   s.Nonce,
	}
}

func (c *Chain) emitUserEvent(sc bcstate.StateContextI, usr *event.User) {
	if c.GetEventDb() == nil {
		return
	}

	sc.EmitEvent(event.TypeStats, event.TagAddOrOverwriteUser, usr.UserID, usr,
		func(events []event.Event, current event.Event) []event.Event {
			return append(events, current)
		})
}

func (c *Chain) emitUniqueAddressEvent(sc bcstate.StateContextI, s *state.State) {
	if c.GetEventDb() == nil {
		return
	}
	sc.EmitEvent(event.TypeStats, event.TagUniqueAddress, s.TxnHash, nil)
}
