package block

import (
	"context"
	"encoding/json"
	"strconv"

	"0chain.net/core/datastore"
)

/*MagicBlockSummary - the summary of the transaction */
type MagicBlockMap struct {
	datastore.IDField
	Hash       string `json:"hash"`
	BlockRound int64  `json:"block_round"`
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
func (mb *MagicBlockMap) GetScore() (int64, error) {
	return 0, nil
}

/*Write - store read */
func (mb *MagicBlockMap) Write(ctx context.Context) error {
	return mb.GetEntityMetadata().GetStore().Write(ctx, mb)
}

/*Delete - store read */
func (mb *MagicBlockMap) Delete(ctx context.Context) error {
	return mb.GetEntityMetadata().GetStore().Delete(ctx, mb)
}

// UnmarshalJSON decodes the magic block map data
// we implement this because the ID field in cql is `bigint` type,
// while we have string in the MagicBlockMap itself
func (mb *MagicBlockMap) UnmarshalJSON(data []byte) error {
	type Alias MagicBlockMap
	var v = struct {
		ID int64 `json:"id"`
		*Alias
	}{
		Alias: (*Alias)(mb),
	}

	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	if v.ID > 0 {
		mb.ID = strconv.FormatInt(v.ID, 10)
	}
	return nil
}

// MarshalJSON encodes the magic block map
func (mb *MagicBlockMap) MarshalJSON() ([]byte, error) {
	var id int64
	if mb.ID != "" {
		var err error
		id, err = strconv.ParseInt(mb.ID, 10, 64)
		if err != nil {
			return nil, err
		}
	}
	type Alias MagicBlockMap
	return json.Marshal(&struct {
		ID int64 `json:"id"`
		*Alias
	}{
		ID:    id,
		Alias: (*Alias)(mb),
	})
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
