package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	. "0chain.net/core/logging"
	"github.com/vmihailenco/msgpack"
	"go.uber.org/zap"
)

var ErrInvalidData = NewError("invalid_data", "Invalid data")

const (
	CodecJSON    = 0
	CodecMsgpack = 1
)

/*ToJSON - given an entity, get the json of that entity as a buffer */
func ToJSON(entity interface{}) *bytes.Buffer {
	buffer := bytes.NewBuffer(make([]byte, 0, 256))
	json.NewEncoder(buffer).Encode(entity)
	return buffer
}

/*WriteJSON - writes the entity json to a stream */
func WriteJSON(w io.Writer, entity interface{}) error {
	return json.NewEncoder(w).Encode(entity)
}

/*ToMsgpack - msgpack encoding */
func ToMsgpack(entity interface{}) *bytes.Buffer {
	buffer := bytes.NewBuffer(make([]byte, 0, 256))
	encoder := msgpack.NewEncoder(buffer)
	encoder.UseJSONTag(true)
	if impl, ok := entity.(ReadLockable); ok {
		impl.DoReadLock()
		defer impl.DoReadUnlock()
	}
	encoder.Encode(entity)
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

/*FromMsgpack - read data into an entity */
func FromMsgpack(data interface{}, entity interface{}) error {
	var err error
	switch jsondata := data.(type) {
	case []byte:
		decoder := msgpack.NewDecoder(bytes.NewBuffer(jsondata))
		decoder.UseJSONTag(true)
		err = decoder.Decode(entity)
	case string:
		decoder := msgpack.NewDecoder(bytes.NewBuffer([]byte(jsondata)))
		decoder.UseJSONTag(true)
		err = decoder.Decode(entity)
	case io.Reader:
		decoder := msgpack.NewDecoder(jsondata)
		decoder.UseJSONTag(true)
		err = decoder.Decode(entity)
	default:
		return NewError("unknown_data_type", fmt.Sprintf("unknown data type for reading entity from json: %T, %v\n", data, data))
	}
	if err != nil {
		return err
	}
	return nil
}
