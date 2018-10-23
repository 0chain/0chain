package smartcontractstate

import (
	"context"
	"io"

	"0chain.net/util"
)

//SCStateI - interface of the smart contract state
type SCStateI interface {
	SetSCDB(ndb SCDB)
	GetSCDB() SCDB

	GetNodeValue(key Key) (util.Serializable, error)
	Insert(key Key, value util.Serializable) (Key, error)
	Delete(key Key) (Key, error)

	Iterate(ctx context.Context, handler SCDBIteratorHandler) error

	SaveChanges(ctx context.Context, ndb SCDB, includeDeletes bool) error

	// only for testing and debugging
	PrettyPrint(ctx context.Context, w io.Writer) error
}
