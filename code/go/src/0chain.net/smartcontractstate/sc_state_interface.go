package smartcontractstate

import (
	"context"
)

//SCStateI - interface of the smart contract state
type SCStateI interface {
	SetSCDB(ndb SCDB)
	GetSCDB() SCDB

	GetNode(key Key) (Node, error)
	PutNode(key Key, node Node) error
	DeleteNode(key Key) error
	Iterate(ctx context.Context, handler SCDBIteratorHandler) error

	MultiPutNode(keys []Key, nodes []Node) error
	MultiDeleteNode(keys []Key) error

	//SaveChanges(ctx context.Context, ndb SCDB, includeDeletes bool) error

	// only for testing and debugging
	//PrettyPrint(ctx context.Context, w io.Writer) error
}
