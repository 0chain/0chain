package datastore

import (
	"context"
	"fmt"
	"time"
)

/*CollectionIteratorHandler is a collection iteration handler function type */
type CollectionIteratorHandler func(ctx context.Context, ce CollectionEntity) bool

/*CollectionEntity - An entity can implement the CollectionEntity interface by including a CollectionMemberField
*It can optionally override GetCollectionName to provide multiple collections partitioned by some key
* Example - transactions and blocks can be partioned by chain
 */
type CollectionEntity interface {
	Entity
	GetCollectionName() string
	GetCollectionSize() int64
	GetCollectionDuration() time.Duration
	InitCollectionScore()
	SetCollectionScore(score int64)
	GetCollectionScore() int64 // larger scores have higher priority
}

/*EntityCollection - Entities can be organized into collections. EntityCollection provides configuration for those collections */
type EntityCollection struct {
	CollectionName     string
	CollectionSize     int64
	CollectionDuration time.Duration
}

/*GetCollectionName - Given an partitioning key (such as parent key), returns the key for the collection */
func (eq *EntityCollection) GetCollectionName(parent Key) string {
	if IsEmpty(parent) {
		return eq.CollectionName
	}
	return fmt.Sprintf("%s:%s", eq.CollectionName, parent)
}

/*CollectionMemberField - An entity with a CollectionMemberField will automatically put that entity into a collection */
type CollectionMemberField struct {
	//IDField
	EntityCollection *EntityCollection `json:"-"`
	CollectionScore  int64             `json:"-"`
}

/*GetCollectionName - default implementation for CollectionEntity interface
* Entities can override this method to provide collections partitioned by some key
**/
func (cf *CollectionMemberField) GetCollectionName() string {
	return cf.EntityCollection.CollectionName
}

/*GetCollectionSize - get the max size of the collection before trimming */
func (cf *CollectionMemberField) GetCollectionSize() int64 {
	return cf.EntityCollection.CollectionSize
}

/*GetCollectionDuration - get the max duration beyond which the collection should be trimmed */
func (cf *CollectionMemberField) GetCollectionDuration() time.Duration {
	return cf.EntityCollection.CollectionDuration
}

/*GetCollectionScore - override */
func (cf *CollectionMemberField) GetCollectionScore() int64 {
	return cf.CollectionScore
}

/*SetCollectionScore - override */
func (cf *CollectionMemberField) SetCollectionScore(score int64) {
	cf.CollectionScore = score
}

/*InitCollectionScore - override */
func (cf *CollectionMemberField) InitCollectionScore() {
	cf.SetCollectionScore(GetCollectionScore(time.Now()))
}

/*GetCollectionScore - Get collection score */
func GetCollectionScore(ts time.Time) int64 {
	// score := time.Now().UniqNano() // nano seconds (10^18)
	// score := time.Now().Unix() // seconds (10^9)
	return ts.UnixNano() / int64(time.Millisecond) // 10^12
}

func AddToCollection(ce CollectionEntity, ctx context.Context) error {
	entityMetadata := ce.GetEntityMetadata()
	store := entityMetadata.GetStore()
	return store.AddToCollection(ctx, ce)
}
