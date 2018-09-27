package datastore

import (
	"bytes"
	"io"
	"time"

	"0chain.net/common"
	. "0chain.net/logging"
	"go.uber.org/zap"
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
func ToMsgpack(entity Entity) (*bytes.Buffer, error) {
	buffer, err := common.ToMsgpack(entity)
	if err != nil {
		Logger.Error("failed to encode entity", zap.Any("entity", entity.GetKey()))
		time.Sleep(time.Millisecond * 10)
		buffer, err = common.ToMsgpack(entity)
		if err != nil {
			Logger.Error("failed retry to encode entity", zap.Any("entity", entity.GetKey()))
		}
	}
	return buffer, err
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
