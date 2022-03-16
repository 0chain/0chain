package datastore

import (
	"sync"
	"time"
)

type (
	// CollectionEntity describes a collection members interface.
	// It can optionally override GetCollectionName to provide multiple
	// collections partitioned by some key.
	// For example - transactions and blocks can be partitioned by chain.
	CollectionEntity interface {
		// Entity embedded interface.
		Entity

		// GetCollectionName returns the collection name.
		GetCollectionName() string

		// GetCollectionSize returns the collection size.
		GetCollectionSize() int64

		// GetCollectionDuration returns the collection duration.
		GetCollectionDuration() time.Duration

		// InitCollectionScore inits the collection score.
		InitCollectionScore()

		// SetCollectionScore sets the collection score.
		SetCollectionScore(score int64)

		// GetCollectionScore returns the collection score,
		// larger scores have higher priority
		GetCollectionScore() int64
	}

	// EntityCollection describes an organized entities into collections
	// and provides configuration for those collections.
	EntityCollection struct {
		CollectionName     string
		CollectionSize     int64
		CollectionDuration time.Duration

		mutex sync.RWMutex
	}
)

// Clone returns a copy of the EntityCollection instance.
func (d *EntityCollection) Clone() *EntityCollection {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	ec := EntityCollection{
		CollectionName:     d.CollectionName,
		CollectionSize:     d.CollectionSize,
		CollectionDuration: d.CollectionDuration,
	}

	return &ec
}

// GetCollectionName returns the key for the collection
// by given an partitioning key (such as parent key).
func (d *EntityCollection) GetCollectionName(parent string) string {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	name := d.CollectionName
	if !IsEmpty(parent) {
		name += ":" + parent
	}

	return name
}

// GetCollectionSize implements CollectionEntity.GetCollectionSize method of interface.
func (d *EntityCollection) GetCollectionSize() int64 {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	return d.CollectionSize
}

// GetCollectionDuration implements CollectionEntity.GetCollectionDuration method of interface.
func (d *EntityCollection) GetCollectionDuration() time.Duration {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	return d.CollectionDuration
}
