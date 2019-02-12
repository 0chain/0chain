package datastore

import (
	"bytes"
	"io"

	"0chain.net/core/common"
)

const (
	CodecJSON    = 0
	CodecMsgpack = 1
)

/*ToJSON - given an entity, get the json of that entity as a buffer */
func ToJSON(entity Entity) *bytes.Buffer {
	return common.ToJSON(entity)
}

/*WriteJSON - writes the entity json to a stream */
func WriteJSON(w io.Writer, entity Entity) error {
	return common.WriteJSON(w, entity)
}

/*ToMsgpack - msgpack encoding */
func ToMsgpack(entity Entity) *bytes.Buffer {
	return common.ToMsgpack(entity)
}

/*FromJSON - read data into an entity */
func FromJSON(data interface{}, entity Entity) error {
	err := common.FromJSON(data, entity)
	if err != nil {
		return err
	}
	entity.ComputeProperties()
	return nil
}

/*ReadJSON - read entity json from a stream */
func ReadJSON(r io.Reader, entity Entity) error {
	return common.ReadJSON(r, entity)
}

/*FromMsgpack - read data into an entity */
func FromMsgpack(data interface{}, entity Entity) error {
	err := common.FromMsgpack(data, entity)
	if err != nil {
		return err
	}
	entity.ComputeProperties()
	return nil
}
