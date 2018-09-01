package miner

import (
	"0chain.net/block"
	"0chain.net/datastore"
)

/*Notarization - A list of valid block verification tickets for the given block
that are good enough to get notarization */
type Notarization struct {
	datastore.NOIDField
	VerificationTickets []*block.VerificationTicket
	BlockID             datastore.Key `json:"block_id"`
	Round               int64
}

var notarizationEntityMetadata *datastore.EntityMetadataImpl

/*GetEntityMetadata - implementing the interface */
func (notarization *Notarization) GetEntityMetadata() datastore.EntityMetadata {
	return notarizationEntityMetadata
}

/*GetKey - overwrites the interface to return the block id */
func (notarization *Notarization) GetKey() datastore.Key {
	return datastore.ToKey(notarization.BlockID)
}

/*NotarizationProvider - entity provider for block_notarization object */
func NotarizationProvider() datastore.Entity {
	notarization := &Notarization{}
	return notarization
}

/*SetupNotarizationEntity - setup the entity */
func SetupNotarizationEntity() {
	notarizationEntityMetadata = datastore.MetadataProvider()
	notarizationEntityMetadata.Name = "block_notarization"
	notarizationEntityMetadata.Provider = NotarizationProvider
	notarizationEntityMetadata.IDColumnName = "block_id"

	datastore.RegisterEntityMetadata("block_notarization", notarizationEntityMetadata)
}
