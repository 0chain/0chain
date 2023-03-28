package datastore

import (
	"context"
	"time"
)

const (
	Ascending Order = iota + 1
	Descending
)

type (
	// CollectionIteratorHandler describes the signature of
	// the collection iterator handler function.
	CollectionIteratorHandler func(ctx context.Context, ce CollectionEntity) (bool, error)

	// Order describes ordering enum.
	Order int8
)

/*GetCollectionScore - Get collection score */
func GetCollectionScore(ts time.Time) int64 {
	// time.Now().Unix() returns amount of seconds followed by 1e9
	// time.Now().UniqNano() returns amount of nanoseconds followed by 1e18
	return -ts.UnixNano() / int64(time.Millisecond) // the score followed by 1e12
}

// AddToCollection appends entity into the collection store.
func AddToCollection(ctx context.Context, ce CollectionEntity) error {
	return ce.GetEntityMetadata().GetStore().AddToCollection(ctx, ce)
}
