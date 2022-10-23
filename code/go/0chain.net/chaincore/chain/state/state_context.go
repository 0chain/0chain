package state

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"

	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"

	"0chain.net/core/common"

	"0chain.net/chaincore/currency"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/smartcontract/dbs/event"
	"github.com/0chain/common/core/util"
)

//msgp:ignore StateContext, TimedQueryStateContext
//go:generate msgp -io=false -tests=false -v

type ApprovedMinter int

const (
	MinterMiner ApprovedMinter = iota
	MinterStorage
	MinterZcn
)

var (
	approvedMinters = []string{
		"6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9", // miner SC
		"6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7", // storage SC
		"6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712e0"} //zcn SC
)

func GetMinter(minter ApprovedMinter) (string, error) {
	if int(minter) >= len(approvedMinters) {
		return "", fmt.Errorf("invalid minter %v", minter)
	}
	return approvedMinters[minter], nil
}

/*
* The state context is available to the smart contract logic.
* The smart contract logic can use
*    GetClientBalance - to get the balance of a client at the beginning of executing the transaction.
*    AddTransfer - to add transfer of tokens from one client to another.
*  Restrictions:
*    1) The total transfer out from the txn.ClientID should be <= txn.Value
*    2) The only from clients valid are txn.ClientID and txn.ToClientID (which will be the smart contract's client id)
 */

type CommonStateContextI interface {
	GetTrieNode(key datastore.Key, v util.MPTSerializable) error
	InsertTrieNode(key datastore.Key, v util.MPTSerializable) (datastore.Key, error)
	GetBlock() *block.Block
	GetLatestFinalizedBlock() *block.Block
	GetConfig(smartcontract string) (*SCConfig, error)
	SetConfig(smartcontract string, config SCConfig) error
}

//go:generate mockery --case underscore --name=QueryStateContextI --output=./mocks
type QueryStateContextI interface {
	CommonStateContextI
	GetEventDB() *event.EventDb
}

//go:generate mockery --case underscore --name=QueryStateContextI --output=./mocks
type TimedQueryStateContextI interface {
	QueryStateContextI
	Now() common.Timestamp
}

type SCConfig interface{}

type Appender func(events []event.Event, current event.Event) []event.Event

// StateContextI - a state context interface. These interface are available for the smart contract
//
//go:generate mockery --case underscore --name=StateContextI --output=./mocks
type StateContextI interface {
	QueryStateContextI
	GetLastestFinalizedMagicBlock() *block.Block
	GetChainCurrentMagicBlock() *block.MagicBlock
	GetMagicBlock(round int64) *block.MagicBlock
	SetMagicBlock(block *block.MagicBlock)    // cannot use in smart contracts or REST endpoints
	GetState() util.MerklePatriciaTrieI       // cannot use in smart contracts or REST endpoints
	GetTransaction() *transaction.Transaction // cannot use in smart contracts or REST endpoints
	GetClientBalance(clientID datastore.Key) (currency.Coin, error)
	SetStateContext(st *state.State) error // cannot use in smart contracts or REST endpoints
	DeleteTrieNode(key datastore.Key) (datastore.Key, error)
	AddTransfer(t *state.Transfer) error
	AddSignedTransfer(st *state.SignedTransfer)
	AddMint(m *state.Mint) error
	GetTransfers() []*state.Transfer // cannot use in smart contracts or REST endpoints
	GetSignedTransfers() []*state.SignedTransfer
	GetMints() []*state.Mint // cannot use in smart contracts or REST endpoints
	Validate() error
	GetSignatureScheme() encryption.SignatureScheme
	GetLatestFinalizedBlock() *block.Block
	EmitEvent(eventType event.EventType, eventTag event.EventTag, index string, data interface{}, appender ...Appender)
	EmitError(error)
	GetEvents() []event.Event // cannot use in smart contracts or REST endpoints
	GetInvalidStateErrors() []error
}

// StateContext - a context object used to manipulate global state
type StateContext struct {
	block                         *block.Block
	state                         util.MerklePatriciaTrieI
	txn                           *transaction.Transaction
	transfers                     []*state.Transfer
	signedTransfers               []*state.SignedTransfer
	mints                         []*state.Mint
	events                        []event.Event
	getLastestFinalizedMagicBlock func() *block.Block
	getLatestFinalizedBlock       func() *block.Block
	getMagicBlock                 func(round int64) *block.MagicBlock
	getChainCurrentMagicBlock     func() *block.MagicBlock
	getSignature                  func() encryption.SignatureScheme
	eventDb                       *event.EventDb
	mutex                         *sync.Mutex
	storagescConfig               *SCConfig
	invalidStateErrors            []error
}

type GetNow func() common.Timestamp

type TimedQueryStateContext struct {
	StateContextI
	now GetNow
}

func (t TimedQueryStateContext) Now() common.Timestamp {
	return t.now()
}

func NewTimedQueryStateContext(i StateContextI, now GetNow) TimedQueryStateContext {
	return TimedQueryStateContext{
		StateContextI: i,
		now:           now,
	}
}

// NewStateContext - create a new state context
func NewStateContext(
	b *block.Block,
	s util.MerklePatriciaTrieI,
	t *transaction.Transaction,
	getMagicBlock func(int64) *block.MagicBlock,
	getLastestFinalizedMagicBlock func() *block.Block,
	getChainCurrentMagicBlock func() *block.MagicBlock,
	getChainSignature func() encryption.SignatureScheme,
	getLatestFinalizedBlock func() *block.Block,
	eventDb *event.EventDb,
) (
	balances *StateContext,
) {
	return &StateContext{
		block:                         b,
		state:                         s,
		txn:                           t,
		getMagicBlock:                 getMagicBlock,
		getLastestFinalizedMagicBlock: getLastestFinalizedMagicBlock,
		getLatestFinalizedBlock:       getLatestFinalizedBlock,
		getChainCurrentMagicBlock:     getChainCurrentMagicBlock,
		getSignature:                  getChainSignature,
		eventDb:                       eventDb,
		mutex:                         new(sync.Mutex),
	}
}

// GetBlock - get the block associated with this state context
func (sc *StateContext) GetBlock() *block.Block {
	return sc.block
}

func (sc *StateContext) SetMagicBlock(block *block.MagicBlock) {
	sc.block.MagicBlock = block
}

// GetState - get the state MPT associated with this state context
func (sc *StateContext) GetState() util.MerklePatriciaTrieI {
	return sc.state
}

// GetTransaction - get the transaction associated with this context
func (sc *StateContext) GetTransaction() *transaction.Transaction {
	return sc.txn
}

// AddTransfer - add the transfer
func (sc *StateContext) AddTransfer(t *state.Transfer) error {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()
	if t.ClientID != sc.txn.ClientID && t.ClientID != sc.txn.ToClientID {
		return state.ErrInvalidTransfer
	}
	sc.transfers = append(sc.transfers, t)

	return nil
}

// AddSignedTransfer - add the signed transfer
func (sc *StateContext) AddSignedTransfer(st *state.SignedTransfer) {
	// Signature on the signed transfer will be checked on call to sc.Validate()
	sc.signedTransfers = append(sc.signedTransfers, st)
}

// AddMint - add the mint
func (sc *StateContext) AddMint(m *state.Mint) error {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()
	if !sc.isApprovedMinter(m) {
		return state.ErrInvalidMint
	}
	sc.mints = append(sc.mints, m)

	return nil
}

func (sc *StateContext) isApprovedMinter(m *state.Mint) bool {
	for _, minter := range approvedMinters {
		if m.Minter == minter && sc.txn.ToClientID == minter {
			return true
		}
	}
	return false
}

// GetTransfers - get all the transfers
func (sc *StateContext) GetTransfers() []*state.Transfer {
	return sc.transfers
}

// GetSignedTransfers - get all the signed transfers
func (sc *StateContext) GetSignedTransfers() []*state.SignedTransfer {
	return sc.signedTransfers
}

// GetMints - get all the mints and fight bad breath
func (sc *StateContext) GetMints() []*state.Mint {
	return sc.mints
}

func (sc *StateContext) EmitEvent(eventType event.EventType, tag event.EventTag, index string, data interface{}, appenders ...Appender) {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()
	if index == "" {
		logging.Logger.Error("error emitting event: empty index",
			zap.Any("event_type", eventType),
			zap.Any("tag", tag),
			zap.Any("data", data))
	}
	e := event.Event{
		BlockNumber: sc.block.Round,
		TxHash:      sc.txn.Hash,
		Type:        int(eventType),
		Tag:         int(tag),
		Index:       index,
		Data:        data,
	}
	if len(appenders) != 0 {
		sc.events = appenders[0](sc.events, e)
	} else {
		sc.events = append(sc.events, e)
	}
}

func (sc *StateContext) EmitError(err error) {
	sc.events = []event.Event{
		{
			BlockNumber: sc.block.Round,
			TxHash:      sc.txn.Hash,
			Type:        int(event.TypeError),
			Data:        err.Error(),
		},
	}
}

func (sc *StateContext) GetEvents() []event.Event {
	return sc.events
}

func (sc *StateContext) GetEventDB() *event.EventDb {
	return sc.eventDb
}

// Validate - implement interface
func (sc *StateContext) Validate() error {
	var (
		amount currency.Coin
		err    error
	)
	for _, transfer := range sc.transfers {
		if transfer.ClientID == sc.txn.ClientID {
			amount, err = currency.AddCoin(amount, transfer.Amount)
			if err != nil {
				return err
			}
		} else {
			if transfer.ClientID != sc.txn.ToClientID {
				return state.ErrInvalidTransfer
			}
		}
	}

	totalValue := sc.txn.Value
	if config.Configuration().ChainConfig.IsFeeEnabled() {
		totalValue, err = currency.AddCoin(totalValue, sc.txn.Fee)
		if err != nil {
			return err
		}
	}
	if amount > totalValue {
		return state.ErrInvalidTransfer
	}

	for _, signedTransfer := range sc.signedTransfers {
		err := signedTransfer.VerifySignature(true)
		if err != nil {
			return err
		}
		if signedTransfer.Amount <= 0 {
			return state.ErrInvalidTransfer
		}
	}

	return nil
}

func (sc *StateContext) getClientState(clientID string) (*state.State, error) {
	s := &state.State{}
	err := sc.state.GetNodeValue(util.Path(clientID), s)
	if err != nil {
		if err != util.ErrValueNotPresent {
			sc.addInvalidStateError(err)
			return nil, err
		}
		return s, err
	}
	//TODO: should we apply the pending transfers?
	return s, nil
}

// GetClientBalance - get the balance of the client
func (sc *StateContext) GetClientBalance(clientID string) (currency.Coin, error) {
	s, err := sc.getClientState(clientID)
	if err != nil {
		return 0, err
	}
	return s.Balance, nil
}

// GetClientNonce - get the nonce of the client
func (sc *StateContext) GetClientNonce(clientID string) (int64, error) {
	s, err := sc.getClientState(clientID)
	if err != nil {
		return 0, err
	}
	return s.Nonce, nil
}

func (sc *StateContext) GetMagicBlock(round int64) *block.MagicBlock {
	return sc.getMagicBlock(round)
}

func (sc *StateContext) GetLastestFinalizedMagicBlock() *block.Block {
	return sc.getLastestFinalizedMagicBlock()
}

func (sc *StateContext) GetChainCurrentMagicBlock() *block.MagicBlock {
	return sc.getChainCurrentMagicBlock()
}

func (sc *StateContext) GetSignatureScheme() encryption.SignatureScheme {
	return sc.getSignature()
}

func (sc *StateContext) GetTrieNode(key datastore.Key, v util.MPTSerializable) error {
	return sc.getNodeValue(key, v)
}

func (sc *StateContext) InsertTrieNode(key datastore.Key, node util.MPTSerializable) (datastore.Key, error) {
	return sc.setNodeValue(key, node)
}

func (sc *StateContext) DeleteTrieNode(key datastore.Key) (datastore.Key, error) {
	return sc.deleteNode(key)
}

// SetStateContext - set the state context
func (sc *StateContext) SetStateContext(s *state.State) error {
	s.SetRound(sc.block.Round)
	return s.SetTxnHash(sc.txn.Hash)
}

func (sc *StateContext) GetLatestFinalizedBlock() *block.Block {
	return sc.getLatestFinalizedBlock()
}

func (sc *StateContext) GetConfig(smartcontract string) (*SCConfig, error) {
	var config *SCConfig
	switch smartcontract {
	case "storagesc":
		config = sc.storagescConfig
		break
	default:
		return nil, errors.New("invalid smart contract")
	}
	if config == nil {
		return nil, util.ErrValueNotPresent
	}
	return config, nil
}

func (sc *StateContext) SetConfig(smartcontract string, config SCConfig) error {
	switch smartcontract {
	case "storagesc":
		sc.storagescConfig = &config
	default:
		return nil
		// return errors.New("smartcontract not found")
	}
	return nil
}

func (sc *StateContext) getNodeValue(key datastore.Key, v util.MPTSerializable) error {
	if err := sc.state.GetNodeValue(util.Path(encryption.Hash(key)), v); err != nil {
		if err != util.ErrValueNotPresent {
			sc.addInvalidStateError(err)
		}
		return err
	}
	return nil
}

func (sc *StateContext) setNodeValue(key datastore.Key, node util.MPTSerializable) (datastore.Key, error) {
	newKey, err := sc.state.Insert(util.Path(encryption.Hash(key)), node)
	if err != nil {
		if err != util.ErrValueNotPresent {
			sc.addInvalidStateError(err)
		}
		return "", err
	}

	return datastore.Key(newKey), nil
}

func (sc *StateContext) deleteNode(key datastore.Key) (datastore.Key, error) {
	newKey, err := sc.state.Delete(util.Path(encryption.Hash(key)))
	if err != nil {
		if err != util.ErrValueNotPresent {
			sc.addInvalidStateError(err)
		}
		return "", err
	}

	return datastore.Key(newKey), nil
}

func (sc *StateContext) addInvalidStateError(err error) {
	sc.mutex.Lock()
	sc.invalidStateErrors = append(sc.invalidStateErrors, err)
	sc.mutex.Unlock()
}

// GetInvalidStateErrors returns invalid state errors if any
func (sc *StateContext) GetInvalidStateErrors() []error {
	sc.mutex.Lock()
	errs := make([]error, len(sc.invalidStateErrors))
	copy(errs, sc.invalidStateErrors)
	sc.mutex.Unlock()
	return errs
}

// ErrInvalidState checks if the error is an invalid state error
func ErrInvalidState(err error) bool {
	return strings.Contains(err.Error(), util.ErrNodeNotFound.Error())
}

type errorIndex struct {
	err   error
	index int
}

type GetItemFunc[T any] func(id string, balance CommonStateContextI) (T, error)

type rspIndex struct {
	index int
	item  interface{}
}

// GetItemsByIDs read items by ids from MPT concurrently and safely with consistent values
// Note: the GetItemFunc should not return custom error that wraps the error returned from
// StateContextI
func GetItemsByIDs[T any](ids []string, getItem GetItemFunc[*T], balances CommonStateContextI) ([]*T, error) {
	var (
		itemC     = make(chan rspIndex, len(ids))
		stateErrC = make(chan error, len(ids))
		errC      = make(chan errorIndex, len(ids))
		wg        sync.WaitGroup
	)

	for i, id := range ids {
		wg.Add(1)
		go func(idx int, id string) {
			defer wg.Done()
			item, err := getItem(id, balances)
			if err != nil {
				if err != util.ErrValueNotPresent {
					stateErrC <- err
					return
				}

				errC <- errorIndex{
					err:   err,
					index: idx,
				}
				return
			}

			if reflect.ValueOf(item).IsNil() {
				errC <- errorIndex{
					err:   fmt.Errorf("nil item returned without ErrValueNotPresent"),
					index: idx,
				}
				return
			}

			itemC <- rspIndex{
				index: idx,
				item:  item,
			}
		}(i, id)
	}
	wg.Wait()
	close(itemC)
	close(errC)

	// check internal error first
	select {
	case err := <-stateErrC:
		return nil, err
	default:
	}

	errIdxs := make([]errorIndex, 0, len(ids))
	for ei := range errC {
		errIdxs = append(errIdxs, ei)
	}

	if len(errIdxs) > 0 {
		sort.SliceStable(errIdxs, func(i, j int) bool {
			return errIdxs[i].index < errIdxs[j].index
		})

		// we would only return one 'value not present' error (the first one) to avoid too much
		// error data added to transaction output.
		logging.Logger.Error("could not get items", zap.Any("errors", errIdxs))
		retErr := errIdxs[0]
		return nil, fmt.Errorf("could not get item %q: %v", ids[retErr.index], retErr.err)
	}

	//ensure original ordering
	items := make([]*T, len(ids))
	for item := range itemC {
		v, ok := item.item.(*T)
		if !ok {
			return nil, fmt.Errorf("invalid item type: %v", reflect.TypeOf(item.item))
		}

		items[item.index] = v
	}

	return items, nil
}
