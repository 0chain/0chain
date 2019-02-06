package smartcontractstate

import (
	"context"
	"strings"
)

/*SCState - wrapper to interact with the smart contract state */
type SCState struct {
	DB        SCDB
	SCAddress string
}

/*NewSCState - create a new state for smart contracts */
func NewSCState(db SCDB, scAddress string) *SCState {
	scs := &SCState{DB: db, SCAddress: scAddress}
	return scs
}

/*SetSCDB - implement interface */
func (scs *SCState) SetSCDB(ndb SCDB) {
	scs.DB = ndb
}

/*GetSCDB - implement interface */
func (scs *SCState) GetSCDB() SCDB {
	return scs.DB
}

func (scs *SCState) GetSCKeyPrefix() string {
	return scs.SCAddress + Separator
}

/*GetNode - get the value for a given path */
func (scs *SCState) GetNode(key Key) (Node, error) {
	keyString := string(key)
	keyString = scs.GetSCKeyPrefix() + keyString
	return scs.DB.GetNode(Key(keyString))
}

/*PutNode - inserts the key into DB */
func (scs *SCState) PutNode(key Key, value Node) error {
	keyString := string(key)
	keyString = scs.GetSCKeyPrefix() + keyString
	return scs.DB.PutNode(Key(keyString), value)
}

/*DeleteNode - delete a value from the db */
func (scs *SCState) DeleteNode(key Key) error {
	keyString := string(key)
	keyString = scs.GetSCKeyPrefix() + keyString
	return scs.DB.DeleteNode(Key(keyString))
}

/*Iterate - iterate the entire smart contract state */
func (scs *SCState) Iterate(ctx context.Context, handler SCDBIteratorHandler) error {
	iterHandler := func(ctx context.Context, key Key, node Node) error {
		keyString := string(key)
		if strings.HasPrefix(keyString, scs.GetSCKeyPrefix()) {
			return handler(ctx, key, node)
		}
		return nil
	}
	err := scs.DB.Iterate(ctx, iterHandler)
	if err != nil {
		return err
	}
	return nil
}

func (scs *SCState) MultiPutNode(keys []Key, nodes []Node) error {
	keyStrings := make([]Key, len(keys))
	for idx, key := range keys {
		keyString := string(key)
		keyString = scs.GetSCKeyPrefix() + keyString
		keyStrings[idx] = Key(keyString)
	}

	return scs.DB.MultiPutNode(keyStrings, nodes)
}

func (scs *SCState) MultiDeleteNode(keys []Key) error {
	keyStrings := make([]Key, len(keys))
	for idx, key := range keys {
		keyString := string(key)
		keyString = scs.GetSCKeyPrefix() + keyString
		keyStrings[idx] = Key(keyString)
	}

	return scs.DB.MultiDeleteNode(keyStrings)
}

/*PrettyPrint - print this state */
// func (scs *SCState) PrettyPrint(ctx context.Context, w io.Writer) error {
// 	if pndb, ok := scs.GetSCDB().(*PSCDB); ok {
// 		Logger.Info("SmartContractState: about to print")
// 		handler := func(ctx context.Context, key Key, node Node) error {
// 			Logger.Info("SmartContractState: ", zap.Any("key", key), zap.Any("value", node))
// 			return nil
// 		}
// 		pndb.Iterate(ctx, handler)
// 	}
// 	return nil
// }
