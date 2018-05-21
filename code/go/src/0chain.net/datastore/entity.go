package datastore

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"0chain.net/common"
)

var (
	/*EntityNotFound code should be used to check whether an entity is found or not */
	EntityNotFound = "entity_not_found"
	/*EntityDuplicate codee should be used to check if an entity is already present */
	EntityDuplicate = "duplicate_entity"
)

/*Entity - interface that reads and writes any implementing structure as JSON into the store */
type Entity interface {
	GetEntityName() string
	SetKey(key interface{})
	GetKey() interface{}
	Validate(ctx context.Context) error
	Read(ctx context.Context, key string) error
	Write(ctx context.Context) error
	Delete(ctx context.Context) error
	ComputeProperties()
}

/*IDField - Useful to embed this into all the entities and get consistent behavior */
type IDField struct {
	ID interface{} `json:"id"`
}

/*SetKey sets the key */
func (k *IDField) SetKey(key interface{}) {
	k.ID = key
}

/*GetKey returns the key for the entity */
func (k *IDField) GetKey() interface{} {
	return k.ID
}

func (k *IDField) Validate(ctx context.Context) error {
	return nil
}

/*ComputeProperties - default dummy implementation so only entities that need this can implement */
func (k *IDField) ComputeProperties() {

}

func (k *IDField) Read(ctx context.Context, key string) error {
	return common.NewError("abstract_read", "Calling entity.Read() requires implementing the method")
}

func (k *IDField) Write(ctx context.Context) error {
	return common.NewError("abstract_write", "Calling entity.Write() requires implementing the method")
}

type CreationTrackable interface {
	GetCreationTime() common.Time
}

/*CreationDateField - Can be used to add a creation date functionality to an entity */
type CreationDateField struct {
	CreationDate common.Time `json:"creation_date"`
}

/*InitializeCreationDate sets the creation date to current time */
func (cd *CreationDateField) InitializeCreationDate() {
	cd.CreationDate = common.Now()
}

func (cd *CreationDateField) GetCreationTime() common.Time {
	return cd.CreationDate
}

/*GetEntityKey = entity name + entity id */
func GetEntityKey(entity Entity) string {
	return fmt.Sprintf("%s:%s", entity.GetEntityName(), entity.GetKey())
}

/*Read an entity from the store by providing the key */
func Read(ctx context.Context, key interface{}, entity Entity) error {
	entity.SetKey(key)
	redisKey := GetEntityKey(entity)
	c := GetCon(ctx)
	c.Send("GET", redisKey)
	c.Flush()
	data, err := c.Receive()
	if err != nil {
		return err
	}

	if data == nil {
		return common.NewError(EntityNotFound, fmt.Sprintf("%v not found with id = %v", entity.GetEntityName(), redisKey))
	}
	jsondata, ok := data.([]byte)
	if !ok {
		return fmt.Errorf("not a valid entity json: (%v): %T: %v", redisKey, data, data)
	}
	err = json.Unmarshal(jsondata, entity)
	if err != nil {
		return err
	}
	entity.ComputeProperties()
	return nil
}

/*Write an entity to the store */
func Write(ctx context.Context, entity Entity) error {
	return writeAux(ctx, entity, true)
}

func writeAux(ctx context.Context, entity Entity, overwrite bool) error {
	buffer := bytes.NewBuffer(make([]byte, 0, 256))
	json.NewEncoder(buffer).Encode(entity)
	redisKey := GetEntityKey(entity)
	c := GetCon(ctx)
	if overwrite {
		c.Send("SET", redisKey, buffer)
	} else {
		c.Send("SETNX", redisKey, buffer)
	}
	c.Flush()
	data, err := c.Receive()
	if err != nil {
		return err
	}
	if val, ok := data.(int64); ok && val == 0 {
		return common.NewError("duplicate_entity", fmt.Sprintf("%v with key %v already exists", entity.GetEntityName(), entity.GetKey()))
	}
	ce, ok := entity.(CollectionEntity)
	if !ok {
		return nil
	}
	if ce.GetCollectionScore() == 0 {
		ce.InitCollectionScore()
	}
	err = ce.AddToCollection(ctx, ce.GetCollectionName())
	return err
}

/*InsertIfNE - insert an entity only if it doesn't already exist in the store */
func InsertIfNE(ctx context.Context, entity Entity) error {
	return writeAux(ctx, entity, false)
}

/*Delete an entity from the store
*  Given an entity id, the pattern is as follows
* entity.SetKey(id)
* datastore.Delete(ctx,entity)
 */
func Delete(ctx context.Context, entity Entity) error {
	redisKey := GetEntityKey(entity)
	c := GetCon(ctx)
	c.Send("DEL", redisKey)
	c.Flush()
	_, err := c.Receive()
	return err
}

func AllocateEntities(size int, entityProvider common.EntityProvider) ([]Entity, error) {
	entities := make([]Entity, size)
	for i := 0; i < size; i++ {
		entity, ok := entityProvider().(Entity)
		if !ok {
			return nil, common.NewError("invalid_entity_provider", "Could not type cast to entity")
		}
		entities[i] = entity
	}
	return entities, nil
}

/*MultiRead - allows reading multiple entities at the same time */
func MultiRead(ctx context.Context, keys []interface{}, entities []Entity) error {
	rkeys := make([]interface{}, len(keys))
	for idx, key := range keys {
		entity := entities[idx]
		entity.SetKey(key)
		rkeys[idx] = GetEntityKey(entity)
	}
	c := GetCon(ctx)
	c.Send("MGET", rkeys...)
	c.Flush()
	data, err := c.Receive()
	if err != nil {
		return err
	}
	array, ok := data.([]interface{})
	if !ok {
		return fmt.Errorf("not a valid entity json: (%v)", rkeys)
	}
	for idx, ae := range array {
		if ae == nil {
			/* not setting this to nil so it's possible to reuse the same array used for block processing
			instead setting key to nil
			entities[idx] = nil
			*/
			entities[idx].SetKey(nil)
			continue
		}
		entity := entities[idx]
		err = json.Unmarshal(ae.([]byte), entity)
		if err != nil {
			return err
		}
		entity.ComputeProperties()
	}
	return nil
}

/*MultiWrite allows writing multiple entities to the datastore
* If the entities belong to a collection, then all entities should belong to
* the same collection (including partitioning)
 */
func MultiWrite(ctx context.Context, entities ...Entity) error {
	kvpair := make([]interface{}, 2*len(entities))
	hasCollectionEntity := false
	for idx, entity := range entities {
		if !hasCollectionEntity {
			_, hasCollectionEntity = entity.(CollectionEntity)
		}
		/*
			entity.ComputeProperties()
			if err := entity.Validate(ctx); err != nil {
				return err
			} */
		kvpair[2*idx] = GetEntityKey(entity)
		kvpair[2*idx+1] = bytes.NewBuffer(make([]byte, 0, 256))
		json.NewEncoder(kvpair[2*idx+1].(*bytes.Buffer)).Encode(entity)
	}
	c := GetCon(ctx)
	c.Send("MSET", kvpair...)
	c.Flush()
	_, err := c.Receive()
	if err != nil {
		return err
	}
	if hasCollectionEntity {
		err = MultiAddToCollection(ctx, entities)
	}
	return err
}
