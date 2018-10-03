package block

import (
	"context"

	"0chain.net/common"
	"0chain.net/datastore"
)

/*VerificationTicket - verification ticket for the block */
type VerificationTicket struct {
	VerifierID datastore.Key `json:"verifier_id" msgpack:"v_id"`
	Signature  string        `json:"signature" msgpack:"sig"`
}

/*BlockVerificationTicket - verification ticket with the block id.
* As VerificationTickets are contained in a block, it doesn't need to have a reference to a block
* However, when the verifiers verify and send the tickets, they need to indicate what block the
* verification ticket is for. So, this wrapper data strcuture is used for that.
 */
type BlockVerificationTicket struct {
	datastore.NOIDField
	VerificationTicket
	Round   int64         `json:"round"`
	BlockID datastore.Key `json:"block_id"`
}

var bvtEntityMetadata *datastore.EntityMetadataImpl

/*GetEntityMetadata - implementing the interface */
func (bvt *BlockVerificationTicket) GetEntityMetadata() datastore.EntityMetadata {
	return bvtEntityMetadata
}

/*GetKey - returning the block id as the key */
func (bvt *BlockVerificationTicket) GetKey() datastore.Key {
	return datastore.ToKey(bvt.BlockID)
}

/*Validate - implementing the interface */
func (bvt *BlockVerificationTicket) Validate(ctx context.Context) error {
	if datastore.IsEmpty(bvt.VerifierID) {
		return common.InvalidRequest("block_verification_ticket id is required")
	}
	return nil
}

/*BVTProvider - entity provider for block_verification_ticket object */
func BVTProvider() datastore.Entity {
	bvt := &BlockVerificationTicket{}
	return bvt
}

/*SetupBVTEntity - setup the entity */
func SetupBVTEntity() {
	bvtEntityMetadata = datastore.MetadataProvider()
	bvtEntityMetadata.Name = "block_verification_ticket"
	bvtEntityMetadata.Provider = BVTProvider
	bvtEntityMetadata.IDColumnName = "block_id"
	datastore.RegisterEntityMetadata("block_verification_ticket", bvtEntityMetadata)
}

/*GetBlockVerificationTicket - Get Block Verification Ticket */
func (vt *VerificationTicket) GetBlockVerificationTicket(b *Block) *BlockVerificationTicket {
	bvt := BVTProvider().(*BlockVerificationTicket)
	bvt.VerifierID = vt.VerifierID
	bvt.Signature = vt.Signature
	bvt.BlockID = b.Hash
	return bvt
}
