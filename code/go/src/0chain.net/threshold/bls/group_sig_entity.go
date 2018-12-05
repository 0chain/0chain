package bls

import (
	"0chain.net/datastore"
)

/* Todo: Change "Bls" to group_sig to be in sync with lingo */
type Bls struct {
	datastore.IDField
	BLSsignShare string `json:"share"`
	BLSRound     int64  `json:"round"`
}

var blsEntityMetadata *datastore.EntityMetadataImpl

func (bls *Bls) GetEntityMetadata() datastore.EntityMetadata {
	return blsEntityMetadata
}

func BLSProvider() datastore.Entity {
	bls := &Bls{}
	return bls
}

func SetupBLSEntity() {
	blsEntityMetadata = datastore.MetadataProvider()
	blsEntityMetadata.Name = "bls_share"
	blsEntityMetadata.Provider = BLSProvider
	blsEntityMetadata.IDColumnName = "bls_id"
	datastore.RegisterEntityMetadata("bls_share", blsEntityMetadata)
}
