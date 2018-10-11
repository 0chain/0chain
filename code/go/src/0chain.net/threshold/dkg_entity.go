package threshold

import (
	"0chain.net/datastore"
)

type Dkg struct {
	datastore.NOIDField
	Share string
	ID    int64 `json:"dkg_id"`
}

var dkgEntityMetadata *datastore.EntityMetadataImpl

func (*Dkg) GetEntityMetadata() datastore.EntityMetadata {
	return dkgEntityMetadata
}

func DKGProvider() datastore.Entity {
	dkg := &Dkg{}
	return dkg
}

func SetupDKGEntity() {
	dkgEntityMetadata = datastore.MetadataProvider()
	dkgEntityMetadata.Name = "dkg_share"
	dkgEntityMetadata.Provider = DKGProvider
	dkgEntityMetadata.IDColumnName = "dkg_id"
	datastore.RegisterEntityMetadata("dkg_share", dkgEntityMetadata)
}
