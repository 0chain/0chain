package common

import (
	"bytes"
	"fmt"
	"io"

	"encoding/json"

	. "github.com/0chain/common/core/logging"
	"github.com/vmihailenco/msgpack/v5"
	"go.uber.org/zap"
)

var ErrInvalidData = NewError("invalid_data", "Invalid data")

const (
	CodecJSON    = 0
	CodecMsgpack = 1
)

/*ToJSON - given an entity, get the json of that entity as a buffer */
func ToJSON(entity interface{}) (*bytes.Buffer, error) {
	buffer := bytes.NewBuffer(make([]byte, 0, 256))
	if err := json.NewEncoder(buffer).Encode(entity); err != nil {
		return nil, err
	}

	return buffer, nil
}

/*WriteJSON - writes the entity json to a stream */
func WriteJSON(w io.Writer, entity interface{}) error {
	return json.NewEncoder(w).Encode(entity)
}

func WriteMsgpack(w io.Writer, entity interface{}) error {
	return msgpack.NewEncoder(w).Encode(entity)
}

/*ToMsgpack - msgpack encoding */
func ToMsgpack(entity interface{}) *bytes.Buffer {
	buffer := bytes.NewBuffer(make([]byte, 0, 256))
	encoder := msgpack.NewEncoder(buffer)
	encoder.SetCustomStructTag("json")
	if impl, ok := entity.(ReadLockable); ok {
		impl.DoReadLock()
		defer impl.DoReadUnlock()
	}
	if err := encoder.Encode(entity); err != nil {
		Logger.Error("msgpack encode failed", zap.Error(err))
	}
	return buffer
}

func ToJSONPipe(entity interface{}) *io.PipeReader {
	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()
		err := json.NewEncoder(pw).Encode(entity)
		if err != nil {
			Logger.Error("TOJSONPipe:error", zap.Error(err))
		}
	}()
	return pr
}

/*FromJSON - read data into an entity */
func FromJSON(data interface{}, entity interface{}) error {
	var err error
	switch jsondata := data.(type) {
	case []byte:
		err = json.Unmarshal(jsondata, entity)
	case string:
		err = json.Unmarshal([]byte(jsondata), entity)
	case io.Reader:
		err = json.NewDecoder(jsondata).Decode(entity)
	default:
		return NewError("unknown_data_type", fmt.Sprintf("unknown data type for reading entity from json: %T, %v\n", data, data))
	}
	if err != nil {
		return err
	}
	return nil
}

/*ReadJSON - read entity json from a stream */
func ReadJSON(r io.Reader, entity interface{}) error {
	return json.NewDecoder(r).Decode(entity)
}

func ReadMsgpack(r io.Reader, entity interface{}) error {
	return msgpack.NewDecoder(r).Decode(entity)
}

/*FromMsgpack - read data into an entity */
func FromMsgpack(data interface{}, entity interface{}) error {
	var err error
	switch jsondata := data.(type) {
	case []byte:
		decoder := msgpack.NewDecoder(bytes.NewBuffer(jsondata))
		decoder.SetCustomStructTag("json")
		err = decoder.Decode(entity)
	case string:
		decoder := msgpack.NewDecoder(bytes.NewBuffer([]byte(jsondata)))
		decoder.SetCustomStructTag("json")
		err = decoder.Decode(entity)
	case io.Reader:
		decoder := msgpack.NewDecoder(jsondata)
		decoder.SetCustomStructTag("json")
		err = decoder.Decode(entity)
	default:
		return NewError("unknown_data_type", fmt.Sprintf("unknown data type for reading entity from json: %T, %v\n", data, data))
	}
	if err != nil {
		return err
	}
	return nil
}
