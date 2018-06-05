package miner

import (
	"context"

	"0chain.net/block"
	"0chain.net/datastore"
)

/*Consensus - A list of valid block verification tickets for the given block
that are good enough to reach consensus */
type Consensus struct {
	datastore.NOIDField
	VerificationTickets []*block.VerificationTicket
	BlockID             datastore.Key `json:"block_id"`
}

var consensusEntityMetadata = &datastore.EntityMetadataImpl{Name: "block_consensus", Provider: ConsensusProvider}

/*GetEntityMetadata - implementing the interface */
func (consensus *Consensus) GetEntityMetadata() datastore.EntityMetadata {
	return consensusEntityMetadata
}

/*GetKey - overwrites the interface to return the block id */
func (consensus *Consensus) GetKey() datastore.Key {
	return datastore.ToKey(consensus.BlockID)
}

/*Validate - implementing the interface */
func (consensus *Consensus) Validate(ctx context.Context) error {
	// TODO
	return nil
}

/*ConsensusProvider - entity provider for block_consensus object */
func ConsensusProvider() datastore.Entity {
	consensus := &Consensus{}
	return consensus
}

/*SetupConsensusEntity - setup the entity */
func SetupConsensusEntity() {
	datastore.RegisterEntityMetadata("block_consensus", consensusEntityMetadata)
}
