package datastore

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"0chain.net/common"
	"github.com/vmihailenco/msgpack"
)

const (
	CodecJSON    = 0
	CodecMsgpack = 1
)

/*ToJSON - given an entity, get the json of that entity as a buffer */
func ToJSON(entity Entity) *bytes.Buffer {
	buffer := bytes.NewBuffer(make([]byte, 0, 256))
	json.NewEncoder(buffer).Encode(entity)
	return buffer
}

/*ToMsgpack - msgpack encoding */
func ToMsgpack(entity Entity) *bytes.Buffer {
	buffer := bytes.NewBuffer(make([]byte, 0, 256))
	encoder := msgpack.NewEncoder(buffer)
	encoder.UseJSONTag(true)
	encoder.Encode(entity)
	return buffer
}

func ToJSONPipe(entity Entity) *io.PipeReader {
	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()
		err := json.NewEncoder(pw).Encode(entity)
		if err != nil {
			fmt.Printf("TOJSONPipe:error: %v\n", err)
		}
	}()
	return pr
}

/*FromJSON - read data into an entity */
func FromJSON(data interface{}, entity Entity) error {
	switch jsondata := data.(type) {
	case []byte:
		return json.Unmarshal(jsondata, entity)
	case string:
		return json.Unmarshal([]byte(jsondata), entity)
	case io.Reader:
		return json.NewDecoder(jsondata).Decode(entity)
	default:
		return common.NewError("unknown_data_type", fmt.Sprintf("unknown data type for reading entity from json: %T, %v\n", data, data))
	}
}

/*FromMsgpack - read data into an entity */
func FromMsgpack(data interface{}, entity Entity) error {
	switch jsondata := data.(type) {
	case []byte:
		decoder := msgpack.NewDecoder(bytes.NewBuffer(jsondata))
		decoder.UseJSONTag(true)
		return decoder.Decode(entity)
	case string:
		decoder := msgpack.NewDecoder(bytes.NewBuffer([]byte(jsondata)))
		decoder.UseJSONTag(true)
		return decoder.Decode(entity)
	case io.Reader:
		decoder := msgpack.NewDecoder(jsondata)
		decoder.UseJSONTag(true)
		return decoder.Decode(entity)
	default:
		return common.NewError("unknown_data_type", fmt.Sprintf("unknown data type for reading entity from json: %T, %v\n", data, data))
	}
}
