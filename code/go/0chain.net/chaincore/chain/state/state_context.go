package state

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"

	"0chain.net/core/config"
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"

	"0chain.net/core/common"

	"github.com/0chain/common/core/currency"

	"github.com/0chain/common/core/statecache"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/threshold/bls"
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
		"6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712e0", //zcn SC
		"6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d3"} //faucet SC
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

type Appender func(events []event.Event, current event.Event) []event.Event

// StateContextI - a state context interface. These interface are available for the smart contract
//
//go:generate mockery --case underscore --name=StateContextI --output=./mocks
type StateContextI interface {
	QueryStateContextI
	GetLastestFinalizedMagicBlock() *block.Block
	GetChainCurrentMagicBlock() *block.MagicBlock
	GetMagicBlock(round int64) *block.MagicBlock
	LoadDKGSummary(magicBlockNum int64) (*bls.DKGSummary, error)
	SetMagicBlock(block *block.MagicBlock) // cannot use in smart contracts or REST endpoints
	SetDKG(dkg *bls.DKG) error
	GetState() util.MerklePatriciaTrieI       // cannot use in smart contracts or REST endpoints
	GetTransaction() *transaction.Transaction // cannot use in smart contracts or REST endpoints
	GetClientState(clientID datastore.Key) (*state.State, error)
	SetClientState(clientID datastore.Key, s *state.State) (util.Key, error)
	GetClientBalance(clientID datastore.Key) (currency.Coin, error)
	SetStateContext(st *state.State) error // cannot use in smart contracts or REST endpoints
	DeleteTrieNode(key datastore.Key) (datastore.Key, error)
	AddTransfer(t *state.Transfer) error
	AddSignedTransfer(st *state.SignedTransfer)
	GetTransfers() []*state.Transfer // cannot use in smart contracts or REST endpoints
	GetSignedTransfers() []*state.SignedTransfer
	Validate() error
	GetSignatureScheme() encryption.SignatureScheme
	GetLatestFinalizedBlock() *block.Block
	EmitEvent(eventType event.EventType, eventTag event.EventTag, index string, data interface{}, appender ...Appender)
	EmitEventWithVersion(eventVersion event.EventVersion, eventType event.EventType, eventTag event.EventTag, index string, data interface{}, appender ...Appender)
	EmitError(error)
	GetEvents() []event.Event // cannot use in smart contracts or REST endpoints
	GetMissingNodeKeys() []util.Key
	Cache() *statecache.TransactionCache
}

// StateContext - a context object used to manipulate global state
type StateContext struct {
	block           *block.Block
	state           util.MerklePatriciaTrieI
	txn             *transaction.Transaction
	transfers       []*state.Transfer
	signedTransfers []*state.SignedTransfer
	events          []event.Event
	// clientStates is the cache for storing client states, usually for storing txn.From and txn.To
	clientStates                  map[string]*state.State
	getLastestFinalizedMagicBlock func() *block.Block
	getLatestFinalizedBlock       func() *block.Block
	getMagicBlock                 func(round int64) *block.MagicBlock
	getChainCurrentMagicBlock     func() *block.MagicBlock
	getDKGSummary                 func(magicBlockNum int64) (*bls.DKGSummary, error)
	setDKG                        func(dkg *bls.DKG) error
	getSignature                  func() encryption.SignatureScheme
	eventDb                       *event.EventDb
	mutex                         *sync.Mutex
	setMagicBlock                 func(mb *block.MagicBlock) error
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
	getDKGSummary func(magicBlockNum int64) (*bls.DKGSummary, error),
	setDKG func(dkg *bls.DKG) error,
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
		getDKGSummary:                 getDKGSummary,
		setDKG:                        setDKG,
		eventDb:                       eventDb,
		clientStates:                  make(map[string]*state.State),
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

func (sc *StateContext) LoadDKGSummary(magicBlockNum int64) (*bls.DKGSummary, error) {
	return sc.getDKGSummary(magicBlockNum)
}

func (sc *StateContext) SetDKG(dkg *bls.DKG) error {
	return sc.setDKG(dkg)
}

// AddTransfer - add the transfer
func (sc *StateContext) AddTransfer(t *state.Transfer) error {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()
	if !encryption.IsHash(t.ToClientID) {
		return errors.New("invalid transaction ToClientID")
	}
	sc.transfers = append(sc.transfers, t)

	return nil
}

// AddSignedTransfer - add the signed transfer
func (sc *StateContext) AddSignedTransfer(st *state.SignedTransfer) {
	// Signature on the signed transfer will be checked on call to sc.Validate()
	sc.signedTransfers = append(sc.signedTransfers, st)
}

// GetTransfers - get all the transfers
func (sc *StateContext) GetTransfers() []*state.Transfer {
	return sc.transfers
}

// GetSignedTransfers - get all the signed transfers
func (sc *StateContext) GetSignedTransfers() []*state.SignedTransfer {
	return sc.signedTransfers
}

func (sc *StateContext) EmitEvent(eventType event.EventType, tag event.EventTag, index string, data interface{}, appenders ...Appender) {
	sc.EmitEventWithVersion(event.Version1, eventType, tag, index, data, appenders...)
}

func (sc *StateContext) EmitEventWithVersion(eventVersion event.EventVersion, eventType event.EventType, tag event.EventTag, index string, data interface{}, appenders ...Appender) {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()
	if index == "" {
		logging.Logger.Error("error emitting event: empty index",
			zap.Any("event_type", eventType),
			zap.Any("tag", tag),
			zap.Any("data", data))
	}
	if len(eventVersion) == 0 || eventVersion == "0" {
		eventVersion = event.Version1
	}
	e := event.Event{
		BlockNumber: sc.block.Round,
		TxHash:      sc.txn.Hash,
		Type:        eventType,
		Tag:         tag,
		Index:       index,
		Data:        data,
		Version:     eventVersion,
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
			Type:        event.TypeError,
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

func (sc *StateContext) GetClientState(clientID string) (*state.State, error) {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()
	if s, ok := sc.clientStates[clientID]; ok {
		return s.Clone(), nil
	}

	s := &state.State{}
	path := util.Path(clientID)
	err := sc.state.GetNodeValue(path, s)
	if err != nil {
		if err != util.ErrValueNotPresent {
			return nil, err
		}
		if er := sc.SetStateContext(s); er != nil {
			return s, er
		}
		return s, err
	}
	//TODO: should we apply the pending transfers?
	sc.clientStates[clientID] = s.Clone()
	return s, nil
}

func (sc *StateContext) SetClientState(clientID string, s *state.State) (util.Key, error) {
	k, err := sc.state.Insert(util.Path(clientID), s)
	if err != nil {
		return nil, err
	}

	sc.mutex.Lock()
	sc.clientStates[clientID] = s.Clone()
	sc.mutex.Unlock()

	return k, nil
}

// GetClientBalance - get the balance of the client
func (sc *StateContext) GetClientBalance(clientID string) (currency.Coin, error) {
	s, err := sc.GetClientState(clientID)
	if err != nil {
		return 0, err
	}
	return s.Balance, nil
}

// GetClientNonce - get the nonce of the client
func (sc *StateContext) GetClientNonce(clientID string) (int64, error) {
	s, err := sc.GetClientState(clientID)
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
	// // // get from MPT
	// if err := sc.getNodeValue(key, v); err != nil {
	// 	// fmt.Println("get node value error", err)
	// 	return err
	// }

	// return nil

	cv, ok := sc.Cache().Get(key)
	if ok {
		ccv, ok := statecache.Copyable(v)
		if !ok {
			panic("state context cache - get trie node not copyable")
		}

		if !ccv.CopyFrom(cv) {
			panic("state context cache - get trie node copy from failed")
		}
		return nil
	}

	// get from MPT
	if err := sc.getNodeValue(key, v); err != nil {
		// fmt.Println("get node value error", err)
		return err
	}

	// cache it if it's cacheable
	if cv, ok := statecache.Cacheable(v); ok {
		sc.Cache().Set(key, cv)
	}
	return nil
}

func (sc *StateContext) InsertTrieNode(key datastore.Key, node util.MPTSerializable) (datastore.Key, error) {
	k, err := sc.setNodeValue(key, node)
	if err != nil {
		return "", err
	}

	vn, ok := statecache.Cacheable(node)
	if ok {
		sc.Cache().Set(key, vn)
	}

	return k, nil
}

func (sc *StateContext) DeleteTrieNode(key datastore.Key) (datastore.Key, error) {
	k, err := sc.deleteNode(key)
	if err != nil {
		return "", err
	}

	sc.Cache().Remove(key)
	return k, nil
}

// SetStateContext - set the state context
func (sc *StateContext) SetStateContext(s *state.State) error {
	s.SetRound(sc.block.Round)
	return s.SetTxnHash(sc.txn.Hash)
}

func (sc *StateContext) GetLatestFinalizedBlock() *block.Block {
	return sc.getLatestFinalizedBlock()
}

func (sc *StateContext) getNodeValue(key datastore.Key, v util.MPTSerializable) error {
	return sc.state.GetNodeValue(util.Path(encryption.Hash(key)), v)
}

func (sc *StateContext) setNodeValue(key datastore.Key, node util.MPTSerializable) (datastore.Key, error) {
	newKey, err := sc.state.Insert(util.Path(encryption.Hash(key)), node)
	if err != nil {
		return "", err
	}

	return datastore.Key(newKey), nil
}

func (sc *StateContext) deleteNode(key datastore.Key) (datastore.Key, error) {
	newKey, err := sc.state.Delete(util.Path(encryption.Hash(key)))
	if err != nil {
		return "", err
	}

	return datastore.Key(newKey), nil
}

// GetMissingNodeKeys returns missing node keys
func (sc *StateContext) GetMissingNodeKeys() []util.Key {
	return sc.state.GetMissingNodeKeys()
}

func (sc *StateContext) Cache() *statecache.TransactionCache {
	return sc.state.Cache()
}

// ErrInvalidState checks if the error is an invalid state error
func ErrInvalidState(err error) bool {
	return err != nil && strings.Contains(err.Error(), util.ErrNodeNotFound.Error())
}

// ErrInvalidState checks if the error is an invalid state error
func ErrValueNotPresent(err error) bool {
	return err != nil && strings.Contains(err.Error(), util.ErrValueNotPresent.Error())
}

type errorIndex struct {
	err   error
	index int
}

type GetItemFunc[T any] func(id string, balance StateContextI) (T, error)

type rspIndex struct {
	index int
	item  interface{}
}

// GetItemsByIDs read items by ids from MPT concurrently and safely with consistent values
// Note: the GetItemFunc should not return custom error that wraps the error returned from
// StateContextI
func GetItemsByIDs[T any](ids []string, getItem GetItemFunc[*T], balances StateContextI) ([]*T, error) {
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
