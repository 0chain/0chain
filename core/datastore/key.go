package datastore

import (
	"context"
	"fmt"

	"0chain.net/core/common"
)

//msgp:ignore Key

type Key = string

//go:generate msgp -io=false -tests=false -v
/*IDField - Useful to embed this into all the entities and get consistent behavior */
type IDField struct {
	ID string `json:"id" yaml:"id"`
}

/*SetKey sets the key */
func (k *IDField) SetKey(key Key) {
	k.ID = key
}

/*GetKey returns the key for the entity */
func (k *IDField) GetKey() Key {
	return k.ID
}

/*Validate - just an abstract implementation */
func (k *IDField) Validate(ctx context.Context) error {
	return nil
}

/*ComputeProperties - default dummy implementation so only entities that need this can implement */
func (k *IDField) ComputeProperties() error {
	return nil
}

/*Read - abstract method for memory store read */
func (k *IDField) Read(ctx context.Context, key string) error {
	return common.NewError("abstract_read", "Calling entity.Read() requires implementing the method")
}

/*GetScore - abstract method for score when writing*/
func (k *IDField) GetScore() (int64, error) {
	return 0, nil
}

/*Write - abstract method for memory store write */
func (k *IDField) Write(ctx context.Context) error {
	return common.NewError("abstract_write", "Calling entity.Write() requires implementing the method")
}

/*Delete - abstract method for memory store delete */
func (k *IDField) Delete(ctx context.Context) error {
	return common.NewError("abstract_delete", "Calling entity.Delete() requires implementing the method")
}

/*NOIDFied - used when we just want to create a datastore entity that doesn't
have it's own id (like 1-to-many) that is only required to send it around with the parent key */
type NOIDField struct {
}

/*Read - abstract method for memory store read */
func (nif *NOIDField) Read(ctx context.Context, key string) error {
	return common.NewError("abstract_read", "Calling entity.Read() requires implementing the method")
}

/*GetScore - abstract method for score when writing*/
func (nif *NOIDField) GetScore() (int64, error) {
	return 0, nil
}

/*Write - abstract method for memory store write */
func (nif *NOIDField) Write(ctx context.Context) error {
	return common.NewError("abstract_write", "Calling entity.Write() requires implementing the method")
}

/*Delete - abstract method for memory store delete */
func (nif *NOIDField) Delete(ctx context.Context) error {
	return common.NewError("abstract_delete", "Calling entity.Delete() requires implementing the method")
}

/*GetKey - implementing the interface */
func (nif *NOIDField) GetKey() Key {
	return EmptyKey
}

/*SetKey - implementing the interface */
func (nif *NOIDField) SetKey(key Key) {
}

/*ComputeProperties - implementing the interface */
func (nif *NOIDField) ComputeProperties() error {
	return nil
}

/*Validate - implementing the interface */
func (nif *NOIDField) Validate(ctx context.Context) error {
	return nil
}

/*ToString - return string representation of the key */
func ToString(key Key) string {
	return string(key)
}

func IsEmpty(key Key) bool {
	return len(key) == 0
}

func IsEqual(key1, key2 Key) bool {
	return key1 == key2
}

/*EmptyKey - Represents an empty key */
var EmptyKey = Key("")

/*ToKey - takes an interface and returns a Key */
func ToKey(key interface{}) Key {
	switch v := key.(type) {
	case string:
		return v
	case []byte:
		return Key(v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

/*HashIDField - Useful to embed this into all the entities and get consistent behavior */
type HashIDField struct {
	Hash string `json:"hash" msgpack:"h"`
}

/*GetKey - Entity implementation */
func (h *HashIDField) GetKey() Key {
	return h.Hash
}

/*SetKey - Entity implementation */
func (h *HashIDField) SetKey(key Key) {
	h.Hash = key
}

/*ComputeProperties - Entity implementation */
func (h *HashIDField) ComputeProperties() error {
	return nil
}

/*Validate - Entity implementation */
func (h *HashIDField) Validate(ctx context.Context) error {
	return nil
}
