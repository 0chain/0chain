package chain

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"0chain.net/chaincore/currency"

	"0chain.net/chaincore/node"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/smartcontract/dbs/event"

	metrics "github.com/rcrowley/go-metrics"
	"go.uber.org/zap"

	"0chain.net/chaincore/block"
	bcstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/smartcontract"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/logging"
	"0chain.net/core/util"
	"0chain.net/smartcontract/minersc"
)

//SmartContractExecutionTimer - a metric that tracks the time it takes to execute a smart contract txn
var SmartContractExecutionTimer metrics.Timer

func init() {
	SmartContractExecutionTimer = metrics.GetOrRegisterTimer("sc_execute_timer", nil)
}

var ErrWrongNonce = common.NewError("wrong_nonce", "nonce of sender is not valid")

/*ComputeState - compute the state for the block */
func (c *Chain) ComputeState(ctx context.Context, b *block.Block) (err error) {
	return c.ComputeBlockStateWithLock(ctx, func() error {
		//check whether we already computed it
		if b.IsStateComputed() {
			return nil
		}
		return c.computeState(ctx, b)
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
					zap.Any("round", b.Round), zap.Any("block", b.Hash),
					zap.Error(err))
				return err
			}
		}
		if !b.IsStateComputed() {
			logging.Logger.Error("compute state - state change error",
				zap.Any("round", b.Round), zap.Any("block", b.Hash),
				zap.Error(err))
			return err
		}
	}
	return nil
}

func (c *Chain) computeState(ctx context.Context, b *block.Block) error {
	return b.ComputeState(ctx, c)
}

//SaveChanges - persist the state changes
func (c *Chain) SaveChanges(ctx context.Context, b *block.Block) error {
	if !b.IsStateComputed() {
		err := errors.New("block state not computed")
		logging.Logger.Error("save changes failed", zap.Error(err),
			zap.Int64("round", b.Round),
			zap.String("hash", b.Hash))
		return err
	}
	cctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
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

//ExecuteSmartContract - executes the smart contract for the transaction
func (c *Chain) ExecuteSmartContract(
	ctx context.Context,
	t *transaction.Transaction,
	scData *sci.SmartContractTransactionData,
	balances bcstate.StateContextI) (string, error) {

	var output string
	var err error
	ts := time.Now()
	done := make(chan bool, 1)

	sct := time.NewTimer(c.SmartContractTimeout())
	if node.Self.Type == node.NodeTypeSharder {
		// give more times for sharders to compute state, as sharders are required to be run
		// as full node, so each block should not be executed failed due to timeout
		sct = time.NewTimer(3 * time.Minute)
	}

	go func() {
		output, err = smartcontract.ExecuteSmartContract(t, scData, balances)
		done <- true
	}()
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-sct.C:
		return "", transaction.ErrSmartContractContext
	case <-done:
		SmartContractExecutionTimer.Update(time.Since(ts))
		return output, err
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
func (c *Chain) UpdateState(ctx context.Context, b *block.Block, bState util.MerklePatriciaTrieI, txn *transaction.Transaction) ([]event.Event, error) {
	c.stateMutex.Lock()
	defer c.stateMutex.Unlock()
	return c.updateState(ctx, b, bState, txn)
}

func (c *Chain) EstimateTransactionCost(ctx context.Context,
	b *block.Block,
	bState util.MerklePatriciaTrieI,
	txn *transaction.Transaction) (int, error) {
	var (
		clientState = CreateTxnMPT(bState) // begin transaction
		sctx        = c.NewStateContext(b, clientState, txn, nil)
	)

	if txn.TransactionType == transaction.TxnTypeSmartContract {
		var scData sci.SmartContractTransactionData
		dataBytes := []byte(txn.TransactionData)
		err := json.Unmarshal(dataBytes, &scData)
		if err != nil {
			logging.Logger.Error("Error while decoding the JSON from transaction",
				zap.Any("input", txn.TransactionData), zap.Any("error", err))
			return math.MaxInt32, err
		}
		cost, err := smartcontract.EstimateTransactionCost(txn, scData, sctx)
		return cost, err
	}

	return 0, nil
}

// NewStateContext creation helper.
func (c *Chain) NewStateContext(
	b *block.Block,
	s util.MerklePatriciaTrieI,
	txn *transaction.Transaction,
	eventDb *event.EventDb,
) (balances *bcstate.StateContext) {
	return bcstate.NewStateContext(b, s, txn,
		c.GetBlockSharders,
		func() *block.Block {
			return c.GetLatestFinalizedMagicBlock(context.Background())
		},
		c.GetCurrentMagicBlock,
		c.GetSignatureScheme,
		c.GetLatestFinalizedBlock,
		eventDb,
	)
}

func (c *Chain) updateState(ctx context.Context, b *block.Block, bState util.MerklePatriciaTrieI, txn *transaction.Transaction) (events []event.Event, err error) {
	// check if the block's ClientState has root value
	_, err = bState.GetNodeDB().GetNode(bState.GetRoot())
	if err != nil {
		return nil, common.NewErrorf("update_state_failed",
			"block state root is incorrect, block hash: %v, state hash: %v, root: %v, round: %d",
			b.Hash, util.ToHex(b.ClientStateHash), util.ToHex(bState.GetRoot()), b.Round)
	}

	var (
		clientState = CreateTxnMPT(bState) // begin transaction
		sctx        = c.NewStateContext(b, clientState, txn, nil)
		startRoot   = sctx.GetState().GetRoot()
	)
	defer func() { events = sctx.GetEvents() }()

	if err := c.validateNonce(sctx, txn.ClientID, txn.Nonce); err != nil {
		return nil, err
	}

	//we should check that client has enough funds to pay for transaction before heavy computations are executed
	if err = sctx.Validate(); err != nil {
		return
	}

	switch txn.TransactionType {

	case transaction.TxnTypeSmartContract:
		var output string

		var scData sci.SmartContractTransactionData
		dataBytes := []byte(txn.TransactionData)
		err = json.Unmarshal(dataBytes, &scData)
		if err != nil {
			logging.Logger.Error("Error while decoding the JSON from transaction",
				zap.Any("input", txn.TransactionData), zap.Any("error", err))
			return nil, err
		}

		t := time.Now()
		output, err = c.ExecuteSmartContract(ctx, txn, &scData, sctx)
		switch err {
		//internal errors
		case context.DeadlineExceeded, context.Canceled, transaction.ErrSmartContractContext, util.ErrNodeNotFound:
			sctx.EmitError(err)
			logging.Logger.Error("Error executing the SC, internal error",
				zap.Error(err),
				zap.String("scname", scData.FunctionName),
				zap.String("block", b.Hash),
				zap.String("begin client state", util.ToHex(startRoot)),
				zap.String("prev block", b.PrevBlock.Hash),
				zap.Duration("time_spent", time.Since(t)),
				zap.Any("txn", txn))
			//return original error, to handle upwards
			return events, err
		default:
			if err != nil {
				sctx.EmitError(err)

				if strings.Contains(err.Error(), "node not found") {
					logging.Logger.Error("Error executing the SC, internal error",
						zap.Error(err),
						zap.String("scname", scData.FunctionName),
						zap.String("block", b.Hash),
						zap.String("begin client state", util.ToHex(startRoot)),
						zap.String("prev block", b.PrevBlock.Hash),
						zap.Duration("time_spent", time.Since(t)),
						zap.Any("txn", txn))
					return events, err
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
				clientState = CreateTxnMPT(bState) // begin transaction
				sctx = c.NewStateContext(b, clientState, txn, nil)

				output = err.Error()
				txn.Status = transaction.TxnError
			}
		}
		txn.TransactionOutput = output
		logging.Logger.Info("SC executed",
			zap.String("client id", txn.ClientID),
			zap.String("block", b.Hash),
			zap.Int64("round", b.Round),
			zap.String("prev_state_hash", util.ToHex(b.PrevBlock.ClientStateHash)),
			zap.String("txn_hash", txn.Hash),
			zap.Int64("txn_nonce", txn.Nonce),
			zap.String("txn_func", scData.FunctionName),
			zap.Int("txn_status", txn.Status),
			zap.Duration("txn_exec_time", time.Since(t)),
			zap.String("begin client state", util.ToHex(startRoot)),
			zap.String("current_root", util.ToHex(sctx.GetState().GetRoot())))

	case transaction.TxnTypeData:

	case transaction.TxnTypeSend:
		err = sctx.AddTransfer(state.NewTransfer(txn.ClientID, txn.ToClientID, txn.Value))
		if err != nil {
			logging.Logger.Error("Failed to add transfer",
				zap.Any("txn type", txn.TransactionType),
				zap.Any("transaction_ClientID", txn.ClientID),
				zap.Any("minersc_address", minersc.ADDRESS),
				zap.Any("state_balance", txn.Fee),
				zap.Any("current_root", sctx.GetState().GetRoot()))
			return
		}
	default:
		logging.Logger.Error("Invalid transaction type", zap.Int("txn type", txn.TransactionType))
		return nil, fmt.Errorf("invalid transaction type: %v", txn.TransactionType)
	}

	if c.ChainConfig.IsFeeEnabled() {
		err = sctx.AddTransfer(state.NewTransfer(txn.ClientID, minersc.ADDRESS,
			txn.Fee))
		if err != nil {
			logging.Logger.Error("Failed to add transfer",
				zap.Any("txn type", txn.TransactionType),
				zap.Any("transaction_ClientID", txn.ClientID),
				zap.Any("minersc_address", minersc.ADDRESS),
				zap.Any("state_balance", txn.Fee))
			return
		}
	}

	ue := make(map[string]*event.User)
	for _, transfer := range sctx.GetTransfers() {
		tEvents, er := c.transferAmount(sctx, transfer.ClientID, transfer.ToClientID, transfer.Amount)
		if er != nil {
			logging.Logger.Error("Failed to transfer amount",
				zap.Any("txn type", txn.TransactionType),
				zap.String("txn data", txn.TransactionData),
				zap.Any("transfer_ClientID", transfer.ClientID),
				zap.Any("to_ClientID", transfer.ToClientID),
				zap.Any("amount", transfer.Amount),
				zap.Error(err))
			return
		}
		for _, e := range tEvents {
			ue[e.UserID] = e
		}
	}

	for _, signedTransfer := range sctx.GetSignedTransfers() {
		tEvents, er := c.transferAmount(sctx, signedTransfer.ClientID,
			signedTransfer.ToClientID, signedTransfer.Amount)
		if er != nil {
			logging.Logger.Error("Failed to process signed transfer",
				zap.Any("signedTransfer_ClientID", signedTransfer.ClientID),
				zap.Any("signedTransfer_to_ClientID", signedTransfer.ToClientID),
				zap.Any("signedTransfer_amount", signedTransfer.Amount))
			return
		}
		for _, e := range tEvents {
			ue[e.UserID] = e
		}
	}

	for _, mint := range sctx.GetMints() {
		u, er := c.mintAmount(sctx, mint.ToClientID, mint.Amount)
		if er != nil {
			logging.Logger.Error("mint error", zap.Any("error", err),
				zap.Any("transaction", txn.Hash),
				zap.String("to clientID", mint.ToClientID))
			return
		}
		if u != nil {
			ue[u.UserID] = u
		}
	}

	u, err := c.incrementNonce(sctx, txn.ClientID)
	if err != nil {
		logging.Logger.Error("update nonce error", zap.Any("error", err),
			zap.Any("transaction", txn.Hash),
			zap.String("clientID", txn.ClientID))
		return
	}
	if u != nil {
		ue[u.UserID] = u
	}

	for _, e := range ue {
		if err = c.emitUserEvent(sctx, e); err != nil {
			logging.Logger.Error("could not emit event", zap.Any("error", err),
				zap.Any("transaction", txn.Hash),
				zap.String("clientID", txn.ClientID))
			return
		}
	}
	// commit transaction
	if err = bState.MergeMPTChanges(clientState); err != nil {
		if state.DebugTxn() {
			logging.Logger.DPanic("update state - merge mpt error",
				zap.Int64("round", b.Round), zap.String("block", b.Hash),
				zap.Any("txn", txn), zap.Error(err))
		}

		logging.Logger.Error("error committing txn", zap.Any("error", err))
		return
	}

	//logging.Logger.Debug("update state - root",
	//	zap.String("block", b.Hash),
	//	zap.String("txn", txn.Hash),
	//	zap.String("prev_state_hash", util.ToHex(b.PrevBlock.ClientStateHash)),
	//	zap.String("begin", util.ToHex(startRoot)),
	//	zap.String("root", util.ToHex(clientState.GetRoot())))

	if state.DebugTxn() {
		// TODO: fix me, the b does not has the state changes
		if err = block.ValidateState(context.TODO(), b, startRoot); err != nil {
			logging.Logger.DPanic("update state - state validation failure",
				zap.Any("txn", txn), zap.Error(err))
		}
		var os *state.State
		os, err = c.GetStateById(bState, c.OwnerID())
		if err != nil || os == nil || os.Balance == 0 {
			logging.Logger.DPanic("update state - owner account",
				zap.Int64("round", b.Round), zap.String("block", b.Hash),
				zap.Any("txn", txn), zap.Any("os", os), zap.Error(err))
		}
	}

	//if status is not set
	if txn.Status == 0 {
		txn.Status = transaction.TxnSuccess
	}
	return
}

/*
* transferAmount - transfers balance from one account to another
*   when there is an error getting the state of the from or to account (other than no value), the error is simply returned back
*   when there is an error inserting/deleting the state of the from or to account, this results in fatal error when state is enabled
 */
func (c *Chain) transferAmount(sctx bcstate.StateContextI, fromClient, toClient datastore.Key, amount currency.Coin) ([]*event.User, error) {
	if amount == 0 {
		return nil, nil
	}
	if fromClient == toClient {
		return nil, common.InvalidRequest("from and to client should be different for balance transfer")
	}
	b := sctx.GetBlock()
	clientState := sctx.GetState()
	txn := sctx.GetTransaction()
	fs, err := c.GetStateById(clientState, fromClient)
	if !isValid(err) {
		if state.DebugTxn() {
			logging.Logger.Error("transfer amount - client get", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("prev_block", b.PrevHash), zap.Any("txn", datastore.ToJSON(txn)), zap.Error(err))
			for _, txn := range b.Txns {
				if txn == nil {
					break
				}
				fmt.Fprintf(block.StateOut, "transfer amount r=%v b=%v t=%+v\n", b.Round, b.Hash, txn)
			}
			fmt.Fprintf(block.StateOut, "transfer amount - error getting state value: %v %+v %v\n", fromClient, txn, err)
			block.PrintStates(clientState, b.ClientState)
			logging.Logger.DPanic(fmt.Sprintf("transfer amount - error getting state value: %v %v", fromClient, err))
		}
		return nil, err
	}
	if fs.Balance < amount {
		logging.Logger.Error("transfer amount - insufficient balance",
			zap.Any("balance", fs.Balance),
			zap.Any("transfer", amount))
		return nil, transaction.ErrInsufficientBalance
	}
	ts, err := c.GetStateById(clientState, toClient)
	if !isValid(err) {
		if state.DebugTxn() {
			logging.Logger.Error("transfer amount - to_client get", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("prev_block", b.PrevHash), zap.Any("txn", datastore.ToJSON(txn)), zap.Error(err))
			for _, txn := range b.Txns {
				if txn == nil {
					break
				}
				fmt.Fprintf(block.StateOut, "transfer amount r=%v b=%v t=%+v\n", b.Round, b.Hash, txn)
			}
			fmt.Fprintf(block.StateOut, "transfer amount - error getting state value: %v %+v %v\n", toClient, txn, err)
			block.PrintStates(clientState, b.ClientState)
			logging.Logger.DPanic(fmt.Sprintf("transfer amount - error getting state value: %v %v", toClient, err))
		}
		return nil, err
	}

	if err := sctx.SetStateContext(fs); err != nil {
		logging.Logger.Error("transfer amount - set state context failed",
			zap.Int64("round", b.Round),
			zap.String("state txn hash", fs.TxnHash),
			zap.Error(err))
		return nil, err
	}
	fs.Balance -= amount
	_, err = clientState.Insert(util.Path(fromClient), fs)
	if err != nil {
		if state.DebugTxn() {
			if c.ChainConfig.IsStateEnabled() {
				logging.Logger.DPanic("transfer amount - error", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Any("txn", txn), zap.Error(err))
			}
			if state.Debug() {
				logging.Logger.Error("transfer amount - error", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Any("txn", txn), zap.Error(err))
			}
		}
		return nil, err
	}
	if err := sctx.SetStateContext(ts); err != nil {
		logging.Logger.Error("transfer amount - set state context failed",
			zap.Int64("round", b.Round),
			zap.String("state txn hash", fs.TxnHash),
			zap.Error(err))
		return nil, err
	}
	ts.Balance += amount
	_, err = clientState.Insert(util.Path(toClient), ts)
	if err != nil {
		if state.DebugTxn() {
			if c.ChainConfig.IsStateEnabled() {
				logging.Logger.DPanic("transfer amount - error", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Any("txn", txn), zap.Error(err))
			}
			if state.Debug() {
				logging.Logger.Error("transfer amount - error", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Any("txn", txn), zap.Error(err))
			}
		}
		return nil, err
	}

	return []*event.User{stateToUser(fromClient, fs), stateToUser(toClient, ts)}, nil
}

func (c *Chain) mintAmount(sctx bcstate.StateContextI, toClient datastore.Key, amount currency.Coin) (*event.User, error) {
	if amount == 0 {
		return nil, nil
	}
	b := sctx.GetBlock()
	clientState := sctx.GetState()
	txn := sctx.GetTransaction()
	ts, err := c.GetStateById(clientState, toClient)
	if !isValid(err) {
		if state.DebugTxn() {
			logging.Logger.Error("transfer amount - to_client get", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("prev_block", b.PrevHash), zap.Any("txn", datastore.ToJSON(txn)), zap.Error(err))
			for _, txn := range b.Txns {
				if txn == nil {
					break
				}
				_, _ = fmt.Fprintf(block.StateOut, "transfer amount r=%v b=%v t=%+v\n", b.Round, b.Hash, txn)
			}
			_, _ = fmt.Fprintf(block.StateOut, "transfer amount - error getting state value: %v %+v %v\n", toClient, txn, err)
			block.PrintStates(clientState, b.ClientState)
			logging.Logger.DPanic(fmt.Sprintf("transfer amount - error getting state value: %v %v", toClient, err))
		}
		if state.Debug() {
			logging.Logger.Error("transfer amount - error", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Any("txn", txn), zap.Error(err))
		}
		return nil, common.NewError("mint_amount - get state", err.Error())
	}
	if err := sctx.SetStateContext(ts); err != nil {
		logging.Logger.Error("transfer amount - set state context failed",
			zap.String("txn hash", ts.TxnHash),
			zap.Error(err))
		return nil, err
	}

	ts.Balance += amount
	_, err = clientState.Insert(util.Path(toClient), ts)
	if err != nil {
		if state.DebugTxn() {
			if c.ChainConfig.IsStateEnabled() {
				logging.Logger.Error("transfer amount - to_client get", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("prev_block", b.PrevHash), zap.Any("txn", datastore.ToJSON(txn)), zap.Error(err))
				for _, txn := range b.Txns {
					if txn == nil {
						break
					}
					_, _ = fmt.Fprintf(block.StateOut, "transfer amount r=%v b=%v t=%+v\n", b.Round, b.Hash, txn)
				}
				_, _ = fmt.Fprintf(block.StateOut, "transfer amount - error getting state value: %v %+v %v\n", toClient, txn, err)
				block.PrintStates(clientState, b.ClientState)
				logging.Logger.DPanic("transfer amount - error", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Any("txn", txn), zap.Error(err))
			}
		}
		if state.Debug() {
			logging.Logger.Error("transfer amount - error", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Any("txn", txn), zap.Error(err))
		}
		return nil, common.NewError("mint_amount - insert", err.Error())
	}

	return stateToUser(toClient, ts), nil
}

func (c *Chain) validateNonce(sctx bcstate.StateContextI, fromClient datastore.Key, txnNonce int64) error {
	s, err := c.GetStateById(sctx.GetState(), fromClient)
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
			zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Any("txn_nonce", txnNonce),
			zap.Any("local_nonce", s.Nonce), zap.Error(err))
		return ErrWrongNonce
	}

	return nil
}

func (c *Chain) incrementNonce(sctx bcstate.StateContextI, fromClient datastore.Key) (*event.User, error) {
	sc := sctx.GetState()
	s, err := c.GetStateById(sc, fromClient)
	if !isValid(err) {
		return nil, err
	}
	if s == nil {
		s = &state.State{}
	}
	if err := sctx.SetStateContext(s); err != nil {
		return nil, err
	}
	s.Nonce += 1
	if _, err := sc.Insert(util.Path(fromClient), s); err != nil {
		return nil, err
	}
	logging.Logger.Debug("Updating nonce", zap.String("client", fromClient), zap.Int64("new_nonce", s.Nonce))

	return stateToUser(fromClient, s), nil
}

func CreateTxnMPT(mpt util.MerklePatriciaTrieI) util.MerklePatriciaTrieI {
	tdb := util.NewLevelNodeDB(util.NewMemoryNodeDB(), mpt.GetNodeDB(), false)
	tmpt := util.NewMerklePatriciaTrie(tdb, mpt.GetVersion(), mpt.GetRoot())
	return tmpt
}

func (c *Chain) GetStateById(clientState util.MerklePatriciaTrieI, clientID string) (*state.State, error) {
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

/*GetState - Get the state of a client w.r.t a block. Note, don't call this from within state computation logic
since block.GetStateValue uses a RLock on the StateMutex. This API is for someone reading the state from outside
the protocol without already holding a lock on StateMutex */
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

func (c *Chain) emitUserEvent(sc bcstate.StateContextI, usr *event.User) error {
	if c.GetEventDb() == nil {
		return nil
	}

	sc.EmitEvent(event.TypeStats, event.TagAddOrOverwriteUser, "", usr, func(events []event.Event, current event.Event) []event.Event {
		return append([]event.Event{current}, events...)
	})
	return nil
}
