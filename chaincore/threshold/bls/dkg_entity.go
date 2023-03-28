package bls

import (
	"encoding/json"

	"0chain.net/core/datastore"
)

//go:generate msgp -io=false

type DKGKeyShare struct {
	datastore.IDField
	Message string `json:"message"`
	Share   string `json:"share"`
	Sign    string `json:"sign"`
}

func (dks *DKGKeyShare) Encode() []byte {
	buff, _ := json.Marshal(dks)
	return buff
}

func (dks *DKGKeyShare) Decode(input []byte) error {
	return json.Unmarshal(input, dks)
}

var dkgsEntityMetadata *datastore.EntityMetadataImpl

func (dkgs *DKGKeyShare) GetEntityMetadata() datastore.EntityMetadata {
	return dkgsEntityMetadata
}

func DKGProvider() datastore.Entity {
	dkgs := &DKGKeyShare{}
	return dkgs
}

func SetupDKGEntity() {
	dkgsEntityMetadata = datastore.MetadataProvider()
	dkgsEntityMetadata.Name = "dkg_share"
	dkgsEntityMetadata.Provider = DKGProvider
	dkgsEntityMetadata.IDColumnName = "id"
	datastore.RegisterEntityMetadata("dkg_share", dkgsEntityMetadata)
}
