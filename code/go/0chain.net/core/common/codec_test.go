package common_test

import (
	"0chain.net/chaincore/block"
	"0chain.net/core/common"
	"0chain.net/core/memorystore"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/vmihailenco/msgpack"
	"io"
	"reflect"
	"sync"
	"testing"
	//"encoding/hex"
)

func init() {
	block.SetupEntity(memorystore.GetStorageProvider())
}

type CodecTestStruct struct {
	Numbers []int `json:"numbers" msgpack:"nums"`
}

func TestConcurrentCodec(t *testing.T) {
	var o CodecTestStruct
	var wg sync.WaitGroup
	count := 0
	for idx := 0; idx < 100; idx++ {
		o.Numbers = append(o.Numbers, 1)
		for i := 0; i < 100; i++ {
			var mi = i
			go func() {
				wg.Add(1)
				nums := o.Numbers
				for j := 0; j < 1; j++ {
					nums = append(nums, 100*mi+j)
				}
				o.Numbers = nums
				wg.Done()
			}()
		}
		for i := 0; i < 100; i++ {
			encoded := common.ToMsgpack(o)
			if encoded.Len() > 16 {
				count++
			}
			//fmt.Printf("encoded: %v %v\n",len(o.Numbers),hex.EncodeToString(encoded.Bytes())[:16])
		}
		wg.Wait()
		fmt.Printf("all done: %v\n", count)
	}
}

func TestToJSON(t *testing.T) {
	c := CodecTestStruct{Numbers: []int{1, 2, 3}}
	buf := bytes.NewBuffer(make([]byte, 0, 256))
	if err := json.NewEncoder(buf).Encode(c); err != nil {
		t.Fatal(err)
	}

	type args struct {
		entity interface{}
	}
	tests := []struct {
		name string
		args args
		want *bytes.Buffer
	}{
		{
			name: "Test_ToJSON_OK",
			args: args{entity: c},
			want: buf,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := common.ToJSON(tt.args.entity); !bytes.Equal(got.Bytes(), tt.want.Bytes()) {
				t.Errorf("ToJSON() = %v, want %v", got.Bytes(), tt.want.Bytes())
			}
		})
	}
}

func TestWriteJSON(t *testing.T) {
	c := CodecTestStruct{Numbers: []int{1, 2, 3}}
	buf := bytes.NewBuffer(make([]byte, 0, 256))
	if err := json.NewEncoder(buf).Encode(c); err != nil {
		t.Fatal(err)
	}

	type args struct {
		entity interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantW   string
		wantErr bool
	}{
		{
			name:    "Test_WriteJSON_OK",
			args:    args{entity: c},
			wantW:   buf.String(),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &bytes.Buffer{}
			err := common.WriteJSON(w, tt.args.entity)
			if (err != nil) != tt.wantErr {
				t.Errorf("WriteJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotW := w.String(); gotW != tt.wantW {
				t.Errorf("WriteJSON() gotW = %v, want %v", gotW, tt.wantW)
			}
		})
	}
}

func TestToMsgpack(t *testing.T) {
	entity := block.NewBlock("", 1)
	buf := bytes.NewBuffer(make([]byte, 0, 256))
	encoder := msgpack.NewEncoder(buf)
	encoder.UseJSONTag(true)
	if err := encoder.Encode(entity); err != nil {
		t.Fatal(err)
	}

	type args struct {
		entity interface{}
	}
	tests := []struct {
		name string
		args args
		want *bytes.Buffer
	}{
		{
			name: "Test_ToMsgpack_OK",
			args: args{
				entity: entity,
			},
			want: buf,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := common.ToMsgpack(tt.args.entity); !bytes.Equal(got.Bytes(), tt.want.Bytes()) {
				t.Errorf("ToMsgpack() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestToJSONPipe(t *testing.T) {
	c := CodecTestStruct{Numbers: []int{1, 2, 3}}

	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()
		_ = json.NewEncoder(pw).Encode(c)
	}()
	byt := make([]byte, 0)
	if _, err := pr.Read(byt); err != nil {
		t.Fatal(err)
	}

	type args struct {
		entity interface{}
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{
			name: "Test_ToJSONPipe_OK",
			args: args{entity: c},
			want: byt,
		},
		{
			name: "Test_ToJSONPipe_ERR",
			args: args{entity: CodecTestStruct{}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := common.ToJSONPipe(tt.args.entity)
			byt := make([]byte, 0)
			if _, err := got.Read(byt); err != nil {
				t.Fatal(err)
			}

			if !bytes.Equal(byt, tt.want) {
				t.Errorf("ToJSONPipe() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestFromJSON(t *testing.T) {
	c := CodecTestStruct{Numbers: []int{1, 2, 3}}
	byt, err := json.Marshal(&c)
	if err != nil {
		t.Fatal(err)
	}

	type args struct {
		data interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    interface{}
	}{
		{
			name:    "Test_FromJSON_Bytes_OK",
			args:    args{byt},
			wantErr: false,
			want:    c,
		},
		{
			name:    "Test_FromJSON_String_OK",
			args:    args{string(byt)},
			wantErr: false,
			want:    c,
		},
		{
			name:    "Test_FromJSON_Reader_OK",
			args:    args{bytes.NewBuffer(byt)},
			wantErr: false,
			want:    c,
		},
		{
			name:    "Test_FromJSON_Unknown_Type_ERR",
			args:    args{1},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var entity CodecTestStruct

			if err := common.FromJSON(tt.args.data, &entity); (err != nil) != tt.wantErr {
				t.Errorf("FromJSON() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && !reflect.DeepEqual(entity, tt.want) {
				t.Errorf("FromJSON() got = %v, want %v", entity, tt.want)
			}
		})
	}
}

func TestFromJSON_Unmarshall_ERR(t *testing.T) {
	c := CodecTestStruct{Numbers: []int{1, 2, 3}}
	byt, err := json.Marshal(&c)
	if err != nil {
		t.Fatal(err)
	}

	type args struct {
		data   interface{}
		entity interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "Test_FromJSON_Unmarshall_ERR",
			args:    args{data: byt, entity: CodecTestStruct{}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := common.FromJSON(tt.args.data, tt.args.entity); (err != nil) != tt.wantErr {
				t.Errorf("FromJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestReadJSON(t *testing.T) {
	c := CodecTestStruct{Numbers: []int{1, 2, 3}}
	byt, err := json.Marshal(&c)
	if err != nil {
		t.Fatal(err)
	}

	type args struct {
		r io.Reader
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    CodecTestStruct
	}{
		{
			name:    "Test_ReadJSON_OK",
			args:    args{r: bytes.NewBuffer(byt)},
			wantErr: false,
			want:    c,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var entity CodecTestStruct

			if err := common.ReadJSON(tt.args.r, &entity); (err != nil) != tt.wantErr {
				t.Errorf("ReadJSON() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && !reflect.DeepEqual(entity, tt.want) {
				t.Errorf("ReadJSON() got = %v, want %v", entity, tt.want)
			}
		})
	}
}

func TestFromMsgpack(t *testing.T) {
	c := CodecTestStruct{Numbers: []int{1, 2, 3}}
	buf := bytes.Buffer{}
	encoder := msgpack.NewEncoder(&buf)
	encoder.UseJSONTag(true)
	if err := encoder.Encode(c); err != nil {
		t.Fatal(err)
	}

	type args struct {
		data interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    CodecTestStruct
		wantErr bool
	}{
		{
			name:    "Test_FromMsgpack_Bytes_OK",
			args:    args{data: buf.Bytes()},
			want:    c,
			wantErr: false,
		},
		{
			name:    "Test_FromMsgpack_String_OK",
			args:    args{data: buf.String()},
			want:    c,
			wantErr: false,
		},
		{
			name:    "Test_FromMsgpack_Reader_OK",
			args:    args{data: bytes.NewBuffer(buf.Bytes())},
			want:    c,
			wantErr: false,
		},
		{
			name:    "Test_FromMsgpack_Decoding_ERR",
			args:    args{data: `}{`},
			wantErr: true,
		},
		{
			name:    "Test_FromMsgpack_Unknown_Type_ERR",
			args:    args{data: 123},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var entity CodecTestStruct

			if err := common.FromMsgpack(tt.args.data, &entity); (err != nil) != tt.wantErr {
				t.Errorf("FromMsgpack() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && !reflect.DeepEqual(entity, tt.want) {
				t.Errorf("FromMsgpack() got = %v, want %v", entity, tt.want)
			}
		})
	}
}
