package block

import (
	"context"
	"encoding/json"

	"0chain.net/core/datastore"
)

/*MagicBlockSummary - the summary of the transaction */
type MagicBlockMap struct {
	datastore.IDField
	Hash string `json:"hash"`
}

var magicBlockMapEntityMetadata *datastore.EntityMetadataImpl

//MagicBlockSummaryProvider - factory method
func MagicBlockMapProvider() datastore.Entity {
	mb := &MagicBlockMap{}
	return mb
}

//GetEntityMetadata - implement interface
func (mb *MagicBlockMap) GetEntityMetadata() datastore.EntityMetadata {
	return magicBlockMapEntityMetadata
}

//GetKey - implement interface
func (mb *MagicBlockMap) GetKey() datastore.Key {
	return datastore.ToKey(mb.ID)
}

//SetKey - implement interface
func (mb *MagicBlockMap) SetKey(key datastore.Key) {
	mb.ID = datastore.ToString(key)
}

/*Read - store read */
func (mb *MagicBlockMap) Read(ctx context.Context, key datastore.Key) error {
	return mb.GetEntityMetadata().GetStore().Read(ctx, key, mb)
}

/*GetScore - score for write*/
func (mb *MagicBlockMap) GetScore() int64 {
	return 0
}

/*Write - store read */
func (mb *MagicBlockMap) Write(ctx context.Context) error {
	return mb.GetEntityMetadata().GetStore().Write(ctx, mb)
}

/*Delete - store read */
func (mb *MagicBlockMap) Delete(ctx context.Context) error {
	return mb.GetEntityMetadata().GetStore().Delete(ctx, mb)
}

func (mb *MagicBlockMap) Encode() []byte {
	buff, _ := json.Marshal(mb)
	return buff
}

func (mb *MagicBlockMap) Decode(input []byte) error {
	return json.Unmarshal(input, mb)
}

/*SetupTxnSummaryEntity - setup the txn summary entity */
func SetupMagicBlockMapEntity(store datastore.Store) {
	magicBlockMapEntityMetadata = datastore.MetadataProvider()
	magicBlockMapEntityMetadata.Name = "magic_block_map"
	magicBlockMapEntityMetadata.Provider = MagicBlockMapProvider
	magicBlockMapEntityMetadata.Store = store
	magicBlockMapEntityMetadata.IDColumnName = "id"
	datastore.RegisterEntityMetadata("magic_block_map", magicBlockMapEntityMetadata)
}
