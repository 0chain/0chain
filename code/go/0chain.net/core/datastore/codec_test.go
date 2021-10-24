package datastore_test

import (
	"bytes"
	"io"
	"reflect"
	"testing"

	"encoding/json"

	"github.com/stretchr/testify/require"
	"github.com/vmihailenco/msgpack/v5"

	"0chain.net/chaincore/block"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/memorystore"
)

func init() {
	block.SetupEntity(memorystore.GetStorageProvider())
}

func TestToJSON(t *testing.T) {
	t.Parallel()

	b := block.NewBlock("", 1)

	type args struct {
		entity datastore.Entity
	}
	tests := []struct {
		name string
		args args
		want *bytes.Buffer
	}{
		{
			name: "Test_ToJSON_OK",
			args: args{entity: b},
			want: func() *bytes.Buffer {
				b, err := common.ToJSON(b)
				require.NoError(t, err)
				return b
			}(),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := datastore.ToJSON(tt.args.entity); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ToJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWriteJSON(t *testing.T) {
	t.Parallel()

	b := block.NewBlock("", 1)
	buf := new(bytes.Buffer)
	if err := common.WriteJSON(buf, b); err != nil {
		t.Error(err)
	}

	type args struct {
		entity datastore.Entity
	}
	tests := []struct {
		name    string
		args    args
		wantW   string
		wantErr bool
	}{
		{
			name:    "Test_WriteJSON_OK",
			args:    args{entity: b},
			wantW:   buf.String(),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			w := &bytes.Buffer{}
			err := datastore.WriteJSON(w, tt.args.entity)
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

	b := block.NewBlock("", 1)
	buf := datastore.ToMsgpack(b)

	type args struct {
		entity datastore.Entity
	}
	tests := []struct {
		name string
		args args
		want *bytes.Buffer
	}{
		{
			name: "Test_ToMsgpack_OK",
			args: args{entity: b},
			want: buf,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := datastore.ToMsgpack(tt.args.entity); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ToMsgpack() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFromJSON(t *testing.T) {
	t.Parallel()

	b := block.NewBlock("", 1)
	byt, err := json.Marshal(b)
	if err != nil {
		t.Fatal(err)
	}

	type args struct {
		data   interface{}
		entity datastore.Entity
	}
	tests := []struct {
		name    string
		args    args
		wantB   datastore.Entity
		wantErr bool
	}{
		{
			name:    "Test_FromJSON_OK",
			args:    args{data: byt, entity: &block.Block{}},
			wantB:   b,
			wantErr: false,
		},
		{
			name:    "Test_FromJSON_ERR",
			args:    args{data: "}{", entity: &block.Block{}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if err := datastore.FromJSON(tt.args.data, tt.args.entity); (err != nil) != tt.wantErr {
				t.Errorf("FromJSON() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				got := tt.args.entity.(*block.Block)
				want := tt.wantB.(*block.Block)
				if !reflect.DeepEqual(got.Hash, want.Hash) {
					t.Errorf("FromJSON() got = %#v, want %#v", got.Hash, want.Hash)
				}
			}
		})
	}
}

func TestReadJSON(t *testing.T) {
	t.Parallel()

	b := block.NewBlock("", 1)
	byt, err := json.Marshal(b)
	if err != nil {
		t.Fatal(err)
	}

	type args struct {
		r      io.Reader
		entity datastore.Entity
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		wantB   datastore.Entity
	}{
		{
			name:    "Test_ReadJSON_OK",
			args:    args{r: bytes.NewBuffer(byt), entity: &block.Block{}},
			wantErr: false,
			wantB:   b,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if err := datastore.ReadJSON(tt.args.r, tt.args.entity); (err != nil) != tt.wantErr {
				t.Errorf("ReadJSON() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				got := tt.args.entity.(*block.Block)
				want := tt.wantB.(*block.Block)
				if !reflect.DeepEqual(got.Hash, want.Hash) {
					t.Errorf("ReadJSON() got = %#v, want %#v", got.Hash, want.Hash)
				}
			}
		})
	}
}

func TestFromMsgpack(t *testing.T) {
	t.Parallel()

	b := block.NewBlock("", 1)
	buf := bytes.Buffer{}
	encoder := msgpack.NewEncoder(&buf)
	encoder.SetCustomStructTag("json")
	if err := encoder.Encode(b); err != nil {
		t.Fatal(err)
	}

	type args struct {
		data   interface{}
		entity datastore.Entity
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		wantB   datastore.Entity
	}{
		{
			name: "Test_FromMsgpack_OK",
			args: args{
				data:   buf.Bytes(),
				entity: &block.Block{},
			},
			wantErr: false,
			wantB:   b,
		},
		{
			name: "Test_FromMsgpack_ERR",
			args: args{
				data:   1,
				entity: &block.Block{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if err := datastore.FromMsgpack(tt.args.data, tt.args.entity); (err != nil) != tt.wantErr {
				t.Errorf("FromMsgpack() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				got := tt.args.entity.(*block.Block)
				want := tt.wantB.(*block.Block)
				if !reflect.DeepEqual(got.Hash, want.Hash) {
					t.Errorf("FromMsgpack() got = %#v, want %#v", got.Hash, want.Hash)
				}
			}
		})
	}
}
