package blockstore

import (
	"context"
	"fmt"
	"time"

	"0chain.net/core/datastore"
)

// unmovedBlockRecord Unmoved blocks; If cold tiering is enabled then record of unmoved blocks will be kept inside unmovedBlockRecord bucket.
// Some worker will query for the unmoved block and if it is within the date range then it will be moved to the cold storage.
type unmovedBlockRecord struct {
	CreatedAt time.Time `json:"crAt"`
	Hash      string    `json:"hs"`
	datastore.CollectionMemberField
}

func NewUnmovedBlockRecord(hash string, createdAt time.Time) *unmovedBlockRecord {
	b := &unmovedBlockRecord{Hash: hash, CreatedAt: createdAt}
	b.SetKey(b.Hash)
	b.EntityCollection = unmovedBlockRecordEntityCollection
	return b
}

func DefaultUnmovedBlockRecord() *unmovedBlockRecord {
	b := &unmovedBlockRecord{}
	b.EntityCollection = unmovedBlockRecordEntityCollection
	return b
}

func (ubr *unmovedBlockRecord) GetEntityMetadata() datastore.EntityMetadata {
	return unmovedBlockRecordEntityMetadata
}

func (ubr *unmovedBlockRecord) SetKey(key datastore.Key) {
	ubr.Hash = datastore.ToString(key)
}

func (ubr *unmovedBlockRecord) GetKey() datastore.Key {
	return datastore.ToKey(ubr.Hash)
}

func (ubr *unmovedBlockRecord) GetScore() int64 {
	return ubr.GetCollectionScore()
}

func (ubr *unmovedBlockRecord) ComputeProperties() {
	// Not implemented
}

func (ubr *unmovedBlockRecord) Validate(ctx context.Context) error {
	return nil // Not implemented
}

func (ubr *unmovedBlockRecord) Read(ctx context.Context, key datastore.Key) error {
	return nil // Not implemented
}

func (ubr *unmovedBlockRecord) Write(ctx context.Context) error {
	endTime := time.Date(
		ubr.CreatedAt.Year(),
		ubr.CreatedAt.Month(),
		ubr.CreatedAt.Day(),
		ubr.CreatedAt.Hour(),
		ubr.CreatedAt.Minute(),
		ubr.CreatedAt.Second(),
		ubr.CreatedAt.Nanosecond(),
		time.Local,
	)
	difference := endTime.Sub(startTime)
	ubr.SetCollectionScore(difference.Milliseconds())
	return ubr.GetEntityMetadata().GetStore().AddToCollection(ctx, ubr)
}

func (ubr *unmovedBlockRecord) Delete(ctx context.Context) error {
	return ubr.GetEntityMetadata().GetStore().DeleteFromCollection(ctx, ubr)
}

// GetUnmovedBlocks returns the number of blocks = count from the range [0,lastBlock).
func GetUnmovedBlocks(lastBlock, count int64) []*unmovedBlockRecord {
	ubr := &unmovedBlockRecord{}
	var ubrs []datastore.Entity
	err := ubr.GetEntityMetadata().GetStore().GetRangeFromCollection(
		context.Background(),
		ubr,
		ubrs,
		false,
		false,
		"0",
		fmt.Sprintf("%v", lastBlock),
		0,
		count,
	)
	if err != nil {
		return nil
	}

	var res []*unmovedBlockRecord
	for _, uI := range ubrs {
		u := uI.(*unmovedBlockRecord)
		res = append(res, u)
	}

	return res
}

var unmovedBlockRecordEntityMetadata *datastore.EntityMetadataImpl

// providerUnmovedBlockRecord - entity provider for client object
func providerUnmovedBlockRecord() datastore.Entity {
	b := &unmovedBlockRecord{}
	b.EntityCollection = unmovedBlockRecordEntityCollection
	return b
}

// setupEntityUnmovedBlockRecord - setup the entity
func setupEntityUnmovedBlockRecord(store datastore.Store) {
	unmovedBlockRecordEntityMetadata = datastore.MetadataProvider()
	unmovedBlockRecordEntityMetadata.Name = "ubr"
	unmovedBlockRecordEntityMetadata.DB = "MetaRecordDB"
	unmovedBlockRecordEntityMetadata.Provider = providerUnmovedBlockRecord
	unmovedBlockRecordEntityMetadata.Store = store

	datastore.RegisterEntityMetadata("ubr", unmovedBlockRecordEntityMetadata)
	unmovedBlockRecordEntityCollection = &datastore.EntityCollection{
		CollectionName:     "collection." + sortedSetUnmovedBlock,
		CollectionSize:     60000000,
		CollectionDuration: time.Hour,
	}
}

var unmovedBlockRecordEntityCollection *datastore.EntityCollection
