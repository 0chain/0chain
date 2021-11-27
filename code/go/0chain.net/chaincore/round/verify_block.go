package round

import (
	"0chain.net/chaincore/block"
	"0chain.net/core/datastore"
)

var verifyBlockEntityMetadata *datastore.EntityMetadataImpl

// VerifyBlockProvider - entity provider for block_verification_ticket object */
func VerifyBlockProvider() datastore.Entity {
	return &VerifyBlock{}
}

// VerifyBlock represents the block verify proposal message struct
type VerifyBlock struct {
	*block.Block
	VRFShares map[string]*VRFShare
}

// NewVerifyBlock returns a new VerifyBlock instance
func NewVerifyBlock(b *block.Block, vrfShares map[string]*VRFShare) *VerifyBlock {
	return &VerifyBlock{
		Block:     b,
		VRFShares: vrfShares,
	}
}

// GetEntityMetadata returns the verifyBlockEntityMetadata
func (vbe *VerifyBlock) GetEntityMetadata() datastore.EntityMetadata {
	return verifyBlockEntityMetadata
}

// SetupVerifyBlockEntity setup and register verify block entity metadata
func SetupVerifyBlockEntity() {
	name := "verify_block"
	verifyBlockEntityMetadata = datastore.MetadataProvider()
	verifyBlockEntityMetadata.Name = name
	verifyBlockEntityMetadata.Provider = VerifyBlockProvider
	verifyBlockEntityMetadata.IDColumnName = "hash"
	datastore.RegisterEntityMetadata(name, verifyBlockEntityMetadata)
}
