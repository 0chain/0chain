package sharder

import (
	"context"

	"0chain.net/block"
	"0chain.net/datastore"
	"0chain.net/persistencestore"
)

type Sharder struct {
	block.Block
	Block_hash      string `json:"block_hash"`
	Block_signature string `json:"block_signature"`
	Round           string `json:"round"`
}

var blockEntityMetadata = &datastore.EntityMetadataImpl{Name: "sharder", Provider: Provider}

func (s *Sharder) GetEntityName() string {
	return "sharder"
}

func (s *Sharder) PWrite(ctx context.Context) error {
	return persistencestore.PWrite(ctx, s)
}

func (s *Sharder) PDelete(ctx context.Context) error {
	return persistencestore.PDelete(ctx, s)
}

func (s *Sharder) PRead(ctx context.Context, key datastore.Key) error {
	return persistencestore.PRead(ctx, key, s)
}

/*Provider - entity provider for block object */
func Provider() datastore.Entity {
	b := &Sharder{}
	return b
}
