package datastore

import (
  "context"
  "fmt"

  "0chain.net/common"
)

/*Key - a type for the entity key */
type Key = string

/*IDField - Useful to embed this into all the entities and get consistent behavior */
type IDField struct {
	ID Key `json:"id"`
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
func (k *IDField) ComputeProperties() {

}

/*Read - abstract method for memory store read */
func (k *IDField) Read(ctx context.Context, key string) error {
	return common.NewError("abstract_read", "Calling entity.Read() requires implementing the method")
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

/*GetKey - implementing the interface */
func (nif *NOIDField) GetKey() Key {
	return EmptyKey
}

/*SetKey - implementing the interface */
func (nif *NOIDField) SetKey(key Key) {
}

/*ComputeProperties - implementing the interface */
func (nif *NOIDField) ComputeProperties() {
}

/*ToString - return string representation of the key */
func ToString(key Key) string {
	return string(key)
}

func IsEmpty(key Key) bool {
	return len(key) == 0
}

func IsEqual(key1 Key, key2 Key) bool {
   return key1 == key2
}

/*EmptyKey - Represents an empty key */
var EmptyKey = Key("")

/*ToKey - takes an interface and returns a Key */
func ToKey(key interface{}) Key {
	switch v := key.(type) {
	case string:
		return Key(v)
	case []byte:
		return Key(v)
	default:
		return Key(fmt.Sprintf("%v", v))
	}
}
