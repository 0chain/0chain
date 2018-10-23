package smartcontractstate

import (
	"context"
	"io"

	. "0chain.net/logging"
	"0chain.net/util"
	"go.uber.org/zap"
)

/*SCState - wrapper to interact with the smart contract state */
type SCState struct {
	DB SCDB
}

/*NewSCState - create a new state for smart contracts */
func NewSCState(db SCDB) *SCState {
	scs := &SCState{DB: db}
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

/*GetNodeValue - get the value for a given path */
func (scs *SCState) GetNodeValue(key Key) (util.Serializable, error) {
	return nil, nil
}

/*Insert - inserts the key into DB */
func (scs *SCState) Insert(key Key, value util.Serializable) (Key, error) {
	return key, nil
}

/*Delete - delete a value from the db */
func (scs *SCState) Delete(key Key) (Key, error) {
	return key, nil
}

/*SaveChanges - implement interface */
func (scs *SCState) SaveChanges(ctx context.Context, ndb SCDB, includeDeletes bool) error {
	var keys []Key
	var nodes []Node
	handler := func(ctx context.Context, key Key, node Node) error {
		Logger.Info("Saving keys to the persistence", zap.Any("key", util.ToHex(key)), zap.Any("value", node))
		keys = append(keys, key)
		nodes = append(nodes, node)
		return nil
	}
	err := scs.DB.Iterate(ctx, handler)
	if err != nil {
		return err
	}

	err = ndb.MultiPutNode(keys, nodes)
	if pndb, ok := ndb.(*PSCDB); ok {
		pndb.Flush()
	}
	return err
}

/*Iterate - iterate the entire smart contract state */
func (scs *SCState) Iterate(ctx context.Context, handler SCDBIteratorHandler) error {
	return nil
}

/*PrettyPrint - print this state */
func (scs *SCState) PrettyPrint(ctx context.Context, w io.Writer) error {
	if pndb, ok := scs.GetSCDB().(*PSCDB); ok {
		Logger.Info("SmartContractState: about to print")
		handler := func(ctx context.Context, key Key, node Node) error {
			Logger.Info("SmartContractState: ", zap.Any("key", key), zap.Any("value", node))
			return nil
		}
		pndb.Iterate(ctx, handler)
	}
	return nil
}
