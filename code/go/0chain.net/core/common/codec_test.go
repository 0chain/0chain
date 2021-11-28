package common_test

import (
	"bytes"
	"encoding/json"
	"io"
	"reflect"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmihailenco/msgpack/v5"

	"0chain.net/chaincore/block"
	"0chain.net/core/common"
	"0chain.net/core/memorystore"
)

func init() {
	block.SetupEntity(memorystore.GetStorageProvider())
}

type CodecTestStruct struct {
	Numbers []int `json:"numbers" msgpack:"nums"`
	mutex   sync.RWMutex
}

func (c *CodecTestStruct) getNumbers() []int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	nums := make([]int, len(c.Numbers))
	copy(nums, c.Numbers)
	return nums
}

func (c *CodecTestStruct) setNumbers(numbers []int) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.Numbers = make([]int, len(numbers))
	copy(c.Numbers, numbers)
}

func (c *CodecTestStruct) DoReadLock() {
	c.mutex.RLock()
}

func (c *CodecTestStruct) DoReadUnlock() {
	c.mutex.RUnlock()
}

func TestConcurrentCodec(t *testing.T) {
	var (
		o   CodecTestStruct
		wg  sync.WaitGroup
		num = 10
	)

	for idx := 0; idx < 10; idx++ {
		o.setNumbers(append(o.getNumbers(), 1))
		wg.Add(num)
		for i := 0; i < num; i++ {
			go func(mi int, wg *sync.WaitGroup) {
				nums := o.getNumbers()
				for j := 0; j < 1; j++ {
					nums = append(nums, 100*mi+j)
				}
				o.setNumbers(nums)
				wg.Done()
			}(i, &wg)
		}
		for i := 0; i < 10; i++ {
			_ = common.ToMsgpack(&o)
		}
		wg.Wait()
	}
}

func TestToJSON(t *testing.T) {
	t.Parallel()

	c := CodecTestStruct{Numbers: []int{1, 2, 3}}
	buf := bytes.NewBuffer(make([]byte, 0, 256))
	if err := json.NewEncoder(buf).Encode(&c); err != nil {
		t.Fatal(err)
	}

	type args struct {
		entity *CodecTestStruct
	}
	tests := []struct {
		name string
		args args
		want *bytes.Buffer
	}{
		{
			name: "Test_ToJSON_OK",
			args: args{entity: &c},
			want: buf,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := common.ToJSON(tt.args.entity)
			require.NoError(t, err)
			if !bytes.Equal(got.Bytes(), tt.want.Bytes()) {
				t.Errorf("ToJSON() = %v, want %v", got.Bytes(), tt.want.Bytes())
			}
		})
	}
}

func TestWriteJSON(t *testing.T) {
	t.Parallel()

	c := CodecTestStruct{Numbers: []int{1, 2, 3}}
	buf := bytes.NewBuffer(make([]byte, 0, 256))
	if err := json.NewEncoder(buf).Encode(&c); err != nil {
		t.Fatal(err)
	}

	type args struct {
		entity *CodecTestStruct
	}
	tests := []struct {
		name    string
		args    args
		wantW   string
		wantErr bool
	}{
		{
			name:    "Test_WriteJSON_OK",
			args:    args{entity: &c},
			wantW:   buf.String(),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

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
	t.Parallel()

	entity := block.NewBlock("", 1)
	buf := bytes.NewBuffer(make([]byte, 0, 256))
	encoder := msgpack.NewEncoder(buf)
	encoder.SetCustomStructTag("json")
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
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := common.ToMsgpack(tt.args.entity); !bytes.Equal(got.Bytes(), tt.want.Bytes()) {
				t.Errorf("ToMsgpack() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestToJSONPipe(t *testing.T) {
	t.Parallel()

	c := CodecTestStruct{Numbers: []int{1, 2, 3}}

	pr, pw := io.Pipe()
	go func() {
		err := json.NewEncoder(pw).Encode(&c)
		require.NoError(t, err)
		err = pw.Close()
		require.NoError(t, err)
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
			args: args{entity: &c},
			want: byt,
		},
		{
			name: "Test_ToJSONPipe_ERR",
			args: args{entity: CodecTestStruct{}},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

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
	t.Parallel()

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
		want    *CodecTestStruct
	}{
		{
			name:    "Test_FromJSON_Bytes_OK",
			args:    args{byt},
			wantErr: false,
			want:    &c,
		},
		{
			name:    "Test_FromJSON_String_OK",
			args:    args{string(byt)},
			wantErr: false,
			want:    &c,
		},
		{
			name:    "Test_FromJSON_Reader_OK",
			args:    args{bytes.NewBuffer(byt)},
			wantErr: false,
			want:    &c,
		},
		{
			name:    "Test_FromJSON_Unknown_Type_ERR",
			args:    args{1},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var entity CodecTestStruct

			if err := common.FromJSON(tt.args.data, &entity); (err != nil) != tt.wantErr {
				t.Errorf("FromJSON() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && !assert.Equal(t, &entity, tt.want) {
				t.Errorf("FromJSON() got = %v, want %v", &entity, tt.want)
			}
		})
	}
}

func TestFromJSON_Unmarshall_ERR(t *testing.T) {
	t.Parallel()

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
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if err := common.FromJSON(tt.args.data, tt.args.entity); (err != nil) != tt.wantErr {
				t.Errorf("FromJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestReadJSON(t *testing.T) {
	t.Parallel()

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
		want    *CodecTestStruct
	}{
		{
			name:    "Test_ReadJSON_OK",
			args:    args{r: bytes.NewBuffer(byt)},
			wantErr: false,
			want:    &c,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var entity CodecTestStruct

			if err := common.ReadJSON(tt.args.r, &entity); (err != nil) != tt.wantErr {
				t.Errorf("ReadJSON() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && !reflect.DeepEqual(&entity, tt.want) {
				t.Errorf("ReadJSON() got = %v, want %v", &entity, tt.want)
			}
		})
	}
}

func TestFromMsgpack(t *testing.T) {
	t.Parallel()

	c := CodecTestStruct{Numbers: []int{1, 2, 3}}
	buf := bytes.Buffer{}
	encoder := msgpack.NewEncoder(&buf)
	encoder.SetCustomStructTag("msgpack")
	if err := encoder.Encode(&c); err != nil {
		t.Fatal(err)
	}

	type args struct {
		data interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    *CodecTestStruct
		wantErr bool
	}{
		{
			name:    "Test_FromMsgpack_Bytes_OK",
			args:    args{data: buf.Bytes()},
			want:    &c,
			wantErr: false,
		},
		{
			name:    "Test_FromMsgpack_String_OK",
			args:    args{data: buf.String()},
			want:    &c,
			wantErr: false,
		},
		{
			name:    "Test_FromMsgpack_Reader_OK",
			args:    args{data: bytes.NewBuffer(buf.Bytes())},
			want:    &c,
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
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var entity CodecTestStruct

			if err := common.FromMsgpack(tt.args.data, &entity); (err != nil) != tt.wantErr {
				t.Errorf("FromMsgpack() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && !reflect.DeepEqual(&entity, tt.want) {
				t.Errorf("FromMsgpack() got = %v, want %v", &entity, tt.want)
			}
		})
	}
}
