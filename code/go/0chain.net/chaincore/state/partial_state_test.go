package state

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"reflect"
	"strconv"
	"testing"

	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/logging"
	"0chain.net/core/mocks"
	"0chain.net/core/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/vmihailenco/msgpack/v5"
)

func init() {
	logging.InitLogging("testing", "")

	setupPartialStateDBMocks()
}

func setupPartialStateDBMocks() {
	store := mocks.Store{}
	store.On("Read", context.Context(nil), "", mock.AnythingOfType("*state.PartialState")).Return(
		func(_ context.Context, _ string, _ datastore.Entity) error {
			return nil
		},
	)
	store.On("Write", context.Context(nil), mock.AnythingOfType("*state.PartialState")).Return(
		func(_ context.Context, _ datastore.Entity) error {
			return nil
		},
	)
	store.On("Delete", context.Context(nil), mock.AnythingOfType("*state.PartialState")).Return(
		func(_ context.Context, _ datastore.Entity) error {
			return nil
		},
	)

	SetupPartialState(&store)
}

func TestNewPartialState(t *testing.T) {
	t.Parallel()

	var (
		key util.Key = []byte("key")
		ps           = datastore.GetEntityMetadata("partial_state").Instance().(*PartialState)
	)
	ps.Hash = key
	ps.ComputeProperties()

	type args struct {
		key util.Key
	}
	tests := []struct {
		name string
		args args
		want *PartialState
	}{
		{
			name: "OK",
			args: args{key: key},
			want: ps,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := NewPartialState(tt.args.key); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewPartialState() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPartialState_GetKey(t *testing.T) {
	t.Parallel()

	type fields struct {
		Hash    util.Key
		Version string
		Nodes   []util.Node
	}
	tests := []struct {
		name   string
		fields fields
		want   datastore.Key
	}{
		{
			name: "Hex_Key_OK",
			want: "!key!",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ps := &PartialState{
				Hash:    tt.fields.Hash,
				Version: tt.fields.Version,
				Nodes:   tt.fields.Nodes,
			}

			ps.SetKey(tt.want)
			tt.want = datastore.ToKey(ps.Hash)
			if got := ps.GetKey(); got != tt.want {
				t.Errorf("GetKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPartialState_Read(t *testing.T) {
	t.Parallel()

	type fields struct {
		Hash    util.Key
		Version string
		Nodes   []util.Node
	}
	type args struct {
		ctx context.Context
		key datastore.Key
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:    "OK",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ps := &PartialState{
				Hash:    tt.fields.Hash,
				Version: tt.fields.Version,
				Nodes:   tt.fields.Nodes,
			}
			if err := ps.Read(tt.args.ctx, tt.args.key); (err != nil) != tt.wantErr {
				t.Errorf("Read() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPartialState_GetScore(t *testing.T) {
	t.Parallel()

	type fields struct {
		Hash    util.Key
		Version string
		Nodes   []util.Node
	}
	tests := []struct {
		name   string
		fields fields
		want   int64
	}{
		{
			name: "OK",
			want: 0, // not implemented
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ps := &PartialState{
				Hash:    tt.fields.Hash,
				Version: tt.fields.Version,
				Nodes:   tt.fields.Nodes,
			}
			if got := ps.GetScore(); got != tt.want {
				t.Errorf("GetScore() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPartialState_Write(t *testing.T) {
	t.Parallel()

	type fields struct {
		Hash    util.Key
		Version string
		Nodes   []util.Node
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:    "OK",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ps := &PartialState{
				Hash:    tt.fields.Hash,
				Version: tt.fields.Version,
				Nodes:   tt.fields.Nodes,
			}
			if err := ps.Write(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("Write() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPartialState_Delete(t *testing.T) {
	t.Parallel()

	type fields struct {
		Hash    util.Key
		Version string
		Nodes   []util.Node
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:    "OK",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ps := &PartialState{
				Hash:    tt.fields.Hash,
				Version: tt.fields.Version,
				Nodes:   tt.fields.Nodes,
			}
			if err := ps.Delete(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPartialState_GetRoot(t *testing.T) {
	t.Parallel()

	ps := NewPartialState([]byte("key"))
	root := util.NewValueNode()
	ps.AddNode(root)

	type fields struct {
		Hash    util.Key
		Version string
		Nodes   []util.Node
	}
	tests := []struct {
		name   string
		fields fields
		want   util.Node
	}{
		{
			name: "OK",
			fields: fields{
				Nodes: ps.Nodes,
			},
			want: root,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ps := &PartialState{
				Hash:    tt.fields.Hash,
				Version: tt.fields.Version,
				Nodes:   tt.fields.Nodes,
			}
			ps.ComputeProperties()
			if got := ps.GetRoot(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetRoot() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPartialState_UnmarshalJSON(t *testing.T) {
	t.Parallel()

	ps := NewPartialState([]byte("key"))
	ps.Nodes = []util.Node{}
	blob, err := json.Marshal(ps)
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		Hash    util.Key
		Version string
		Nodes   []util.Node
	}
	type args struct {
		data []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		want    *PartialState
	}{
		{
			name:    "OK",
			args:    args{data: blob},
			want:    ps,
			wantErr: false,
		},
		{
			name:    "ERR",
			args:    args{data: []byte("}{")},
			want:    ps,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ps := &PartialState{
				Hash:    tt.fields.Hash,
				Version: tt.fields.Version,
				Nodes:   tt.fields.Nodes,
			}
			if err := ps.UnmarshalJSON(tt.args.data); (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				assert.Equal(t, tt.want, ps)
			}
		})
	}
}

func TestPartialState_UnmarshalPartialState(t *testing.T) {
	t.Parallel()

	ps := PartialState{
		Version: "1",
	}
	ps.SetKey(encryption.Hash("data"))

	type fields struct {
		Hash    util.Key
		Version string
		Nodes   []util.Node
	}
	type args struct {
		obj map[string]interface{}
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Invalid_Root_ERR",
			args: args{
				obj: map[string]interface{}{
					"root": 124,
				},
			},
			wantErr: true,
		},
		{
			name: "No_Root_ERR",
			args: args{
				obj: map[string]interface{}{},
			},
			wantErr: true,
		},
		{
			name: "No_Version_ERR",
			args: args{
				obj: map[string]interface{}{
					"root": ps.Hash,
				},
			},
			wantErr: true,
		},
		{
			name: "No_Nodes_ERR",
			args: args{
				obj: map[string]interface{}{
					"root":    ps.Hash,
					"version": ps.Version,
				},
			},
			wantErr: true,
		},
		{
			name: "Invalid_Nodes_ERR",
			args: args{
				obj: map[string]interface{}{
					"root":    ps.Hash,
					"version": ps.Version,
					"nodes": []interface{}{
						1,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Node_Decoding_ERR",
			args: args{
				obj: map[string]interface{}{
					"root":    []byte("root"),
					"version": ps.Version,
					"nodes": []interface{}{
						"!",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "OK",
			args: args{
				obj: map[string]interface{}{
					"root":    ps.Hash,
					"version": ps.Version,
					"nodes": []interface{}{
						base64.StdEncoding.EncodeToString([]byte(strconv.Itoa(util.NodeTypeValueNode) + "node")),
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ps := &PartialState{
				Hash:    tt.fields.Hash,
				Version: tt.fields.Version,
				Nodes:   tt.fields.Nodes,
			}
			if err := ps.UnmarshalPartialStateJSON(tt.args.obj); (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalPartialStateJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPartialState_MarshalJSON(t *testing.T) {
	t.Parallel()

	ps := PartialState{
		Version: "1",
		Nodes: []util.Node{
			util.NewValueNode(),
		},
	}
	ps.SetKey(encryption.Hash("data"))

	mapPS := map[string]interface{}{
		"root":    util.ToHex(ps.Hash),
		"version": ps.Version,
		"nodes": [][]byte{
			ps.Nodes[0].Encode(),
		},
	}

	blob, err := json.Marshal(mapPS)
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		Hash    util.Key
		Version string
		Nodes   []util.Node
	}
	tests := []struct {
		name    string
		fields  fields
		want    []byte
		wantErr bool
	}{
		{
			name: "OK",
			fields: fields{
				Hash:    ps.Hash,
				Version: ps.Version,
				Nodes:   ps.Nodes,
			},
			want:    blob,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ps := &PartialState{
				Hash:    tt.fields.Hash,
				Version: tt.fields.Version,
				Nodes:   tt.fields.Nodes,
			}

			got, err := ps.MarshalJSON()
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MarshalJSON() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPartialState_AddNode(t *testing.T) {
	t.Parallel()

	nodes := []util.Node{
		util.NewValueNode(),
	}
	node := util.NewValueNode()

	type fields struct {
		Hash    util.Key
		Version string
		Nodes   []util.Node
	}
	type args struct {
		node util.Node
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *PartialState
	}{
		{
			name: "OK",
			fields: fields{
				Nodes: nodes,
			},
			args: args{node: node},
			want: &PartialState{
				Nodes: append(nodes, node),
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ps := &PartialState{
				Hash:    tt.fields.Hash,
				Version: tt.fields.Version,
				Nodes:   tt.fields.Nodes,
			}

			ps.AddNode(tt.args.node)
			assert.Equal(t, tt.want, ps)
		})
	}
}

type valueNode struct {
	Version string
	Value   int64
}

func (v *valueNode) Encode() []byte {
	d, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return d
}

func (v *valueNode) Decode(b []byte) error {
	return json.Unmarshal(b, v)
}

func newValueNode(version string, v int64) *valueNode {
	return &valueNode{
		Version: version,
		Value:   v,
	}
}

func TestPartialUnmarshalMsgpack(t *testing.T) {
	n := util.NewValueNode()
	n.OriginTracker.SetOrigin(100)
	n.Value = newValueNode("1000", 2022)

	ps := PartialState{
		Version: "1",
		Nodes: []util.Node{
			n,
		},
	}
	ps.SetKey(encryption.Hash("data"))

	d, err := msgpack.Marshal(&ps)
	require.NoError(t, err)

	var ps2 PartialState
	if err := msgpack.Unmarshal(d, &ps2); err != nil {
		panic(err)
	}

	for _, nd := range ps2.Nodes {
		require.Equal(t, util.Sequence(100), nd.GetOriginTracker().GetVersion())
		nv := nd.(*util.ValueNode).Value
		vv := valueNode{}
		if err := vv.Decode(nv.Encode()); err != nil {
			panic(err)
		}

		require.Equal(t, "1000", vv.Version)
		require.Equal(t, int64(2022), vv.Value)
	}
}
