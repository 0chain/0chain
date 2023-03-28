package datastore

import (
	"sync"
	"time"
)

// CollectionMemberField describes an entity with a list of
// EntityCollection that will automatically put that entity into the list.
type CollectionMemberField struct {
	EntityCollection *EntityCollection `json:"-" msgpack:"-"`
	CollectionScore  int64             `json:"-" msgpack:"-"`

	mutex sync.RWMutex
}

// GetCollectionName implements CollectionEntity.GetCollectionName method of interface.
func (d *CollectionMemberField) GetCollectionName() string {
	return d.EntityCollection.GetCollectionName("")
}

// GetCollectionSize implements CollectionEntity.GetCollectionSize method of interface.
func (d *CollectionMemberField) GetCollectionSize() int64 {
	return d.EntityCollection.GetCollectionSize()
}

// GetCollectionDuration implements CollectionEntity.GetCollectionDuration method of interface.
func (d *CollectionMemberField) GetCollectionDuration() time.Duration {
	return d.EntityCollection.GetCollectionDuration()
}

// GetCollectionScore implements CollectionEntity.GetCollectionScore method of interface.
func (d *CollectionMemberField) GetCollectionScore() int64 {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	return d.CollectionScore
}

// SetCollectionScore implements CollectionEntity.SetCollectionScore method of interface.
func (d *CollectionMemberField) SetCollectionScore(score int64) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	d.CollectionScore = score
}

// InitCollectionScore implements CollectionEntity.InitCollectionScore method of interface.
func (d *CollectionMemberField) InitCollectionScore() {
	now := time.Now()
	score := GetCollectionScore(now)

	d.SetCollectionScore(score)
}

// Clone returns a clone of this collection
func (d *CollectionMemberField) Clone() *CollectionMemberField {
	return &CollectionMemberField{
		EntityCollection: d.EntityCollection.Clone(),
		CollectionScore:  d.CollectionScore,
	}
}
