package state

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"reflect"
	"strconv"
	"testing"

	"0chain.net/core/datastore"
	"0chain.net/core/mocks"
	"0chain.net/core/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/vmihailenco/msgpack/v5"
)

func init() {
	setupNodesDBMock()
}

func setupNodesDBMock() {
	store := mocks.Store{}
	store.On("Read", context.Context(nil), "", mock.AnythingOfType("*state.Nodes")).Return(
		func(_ context.Context, _ string, _ datastore.Entity) error {
			return nil
		},
	)
	store.On("Write", context.Context(nil), mock.AnythingOfType("*state.Nodes")).Return(
		func(_ context.Context, _ datastore.Entity) error {
			return nil
		},
	)
	store.On("Delete", context.Context(nil), mock.AnythingOfType("*state.Nodes")).Return(
		func(_ context.Context, _ datastore.Entity) error {
			return nil
		},
	)
	SetupStateNodes(&store)
}

func makeTestStateNodes() *Nodes {
	sn := NewStateNodes()
	sn.Nodes = make([]util.Node, 0)
	for i := 0; i < 2; i++ {
		value := util.SecureSerializableValue{Buffer: []byte("node" + strconv.Itoa(i))}
		node := util.NewFullNode(&value)
		sn.Nodes = append(sn.Nodes, node)
	}

	return sn
}

func TestNewStateNodes(t *testing.T) {
	t.Parallel()

	sn, ok := datastore.GetEntityMetadata("state_nodes").Instance().(*Nodes)
	if !ok {
		t.Error("expected Nodes type")
	}

	tests := []struct {
		name string
		want *Nodes
	}{
		{
			name: "OK",
			want: sn,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := NewStateNodes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewStateNodes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNodes_Read(t *testing.T) {
	t.Parallel()

	type fields struct {
		IDField datastore.IDField
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

			ns := &Nodes{
				IDField: tt.fields.IDField,
				Version: tt.fields.Version,
				Nodes:   tt.fields.Nodes,
			}
			if err := ns.Read(tt.args.ctx, tt.args.key); (err != nil) != tt.wantErr {
				t.Errorf("Read() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNodes_Write(t *testing.T) {
	t.Parallel()

	type fields struct {
		IDField datastore.IDField
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

			ns := &Nodes{
				IDField: tt.fields.IDField,
				Version: tt.fields.Version,
				Nodes:   tt.fields.Nodes,
			}
			if err := ns.Write(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("Write() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNodes_Delete(t *testing.T) {
	t.Parallel()

	type fields struct {
		IDField datastore.IDField
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

			ns := &Nodes{
				IDField: tt.fields.IDField,
				Version: tt.fields.Version,
				Nodes:   tt.fields.Nodes,
			}
			if err := ns.Delete(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNodes_SaveState(t *testing.T) {
	t.Parallel()

	db := util.NewMemoryNodeDB()

	nodes := make([]util.Node, 0)
	keys := make([]util.Key, 0)
	for i := 0; i < 2; i++ {
		value := util.SecureSerializableValue{Buffer: []byte("node" + strconv.Itoa(i))}
		node := util.NewFullNode(&value)
		nodes = append(nodes, node)
		keys = append(keys, node.GetHashBytes())
	}

	if err := db.MultiPutNode(keys, nodes); err != nil {
		t.Fatal(err)
	}

	type fields struct {
		IDField datastore.IDField
		Version string
		Nodes   []util.Node
	}
	type args struct {
		ctx     context.Context
		stateDB util.NodeDB
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantDB  util.NodeDB
		wantErr bool
	}{
		{
			name: "OK",
			fields: fields{
				Nodes: nodes,
			},
			args:    args{stateDB: util.NewMemoryNodeDB()},
			wantDB:  db,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ns := &Nodes{
				IDField: tt.fields.IDField,
				Version: tt.fields.Version,
				Nodes:   tt.fields.Nodes,
			}
			if err := ns.SaveState(tt.args.ctx, tt.args.stateDB); (err != nil) != tt.wantErr {
				t.Errorf("SaveState() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				assert.Equal(t, tt.wantDB, tt.args.stateDB)
			}
		})
	}
}

func TestNodes_MarshalJSON(t *testing.T) {
	t.Parallel()

	nodes := makeTestStateNodes()
	//data := make(map[string]interface{})
	blob, err := json.Marshal(nodes)
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		IDField datastore.IDField
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
				IDField: nodes.IDField,
				Version: nodes.Version,
				Nodes:   nodes.Nodes,
			},
			want: blob,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ns := &Nodes{
				IDField: tt.fields.IDField,
				Version: tt.fields.Version,
				Nodes:   tt.fields.Nodes,
			}
			got, err := ns.MarshalJSON()
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

func TestNodes_MartialStateNodes(t *testing.T) {
	t.Parallel()

	sn := makeTestStateNodes()

	encNodes := make([][]byte, len(sn.Nodes))
	for idx, nd := range sn.Nodes {
		encNodes[idx] = nd.Encode()
	}
	data := map[string]interface{}{
		"version": sn.Version,
		"nodes":   encNodes,
	}
	blob, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		IDField datastore.IDField
		Version string
		Nodes   []util.Node
	}
	type args struct {
		data map[string]interface{}
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "OK",
			fields: fields{
				IDField: sn.IDField,
				Version: sn.Version,
				Nodes:   sn.Nodes,
			},
			args:    args{data: make(map[string]interface{})},
			want:    blob,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ns := &Nodes{
				IDField: tt.fields.IDField,
				Version: tt.fields.Version,
				Nodes:   tt.fields.Nodes,
			}
			got, err := json.Marshal(ns)
			if (err != nil) != tt.wantErr {
				t.Errorf("martialStateNodesJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("martialStateNodesJSON() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNodes_UnmarshalJSON(t *testing.T) {
	t.Parallel()

	sn := makeTestStateNodes()
	blob, err := sn.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		IDField datastore.IDField
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
		want    *Nodes
	}{
		{
			name: "OK",
			args: args{data: blob},
			want: sn,
		},
		{
			name:    "ERR",
			args:    args{data: []byte("}{")},
			want:    sn,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ns := &Nodes{
				IDField: tt.fields.IDField,
				Version: tt.fields.Version,
				Nodes:   tt.fields.Nodes,
			}
			if err := ns.UnmarshalJSON(tt.args.data); (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				assert.Equal(t, tt.want, ns)
			}
		})
	}
}

func TestNodes_UnmarshalStateNodes(t *testing.T) {
	t.Parallel()

	sn := makeTestStateNodes()

	encNodes := make([]interface{}, len(sn.Nodes))
	for idx, nd := range sn.Nodes {
		encNodes[idx] = base64.StdEncoding.EncodeToString(nd.Encode())
	}

	type fields struct {
		IDField datastore.IDField
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
		want    *Nodes
	}{
		{
			name: "Invalid_Version_ERR",
			args: args{
				obj: map[string]interface{}{
					"version": 123,
				},
			},
			wantErr: true,
		},
		{
			name: "Invalid_Nodes_ERR",
			args: args{
				obj: map[string]interface{}{
					"version": sn.Version,
					"nodes":   123,
				},
			},
			wantErr: true,
		},
		{
			name: "Invalid_Node_ERR",
			args: args{
				obj: map[string]interface{}{
					"version": sn.Version,
					"nodes": []interface{}{
						1,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "OK",
			args: args{
				obj: map[string]interface{}{
					"version": sn.Version,
					"nodes":   encNodes,
				},
			},
			want:    sn,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ns := &Nodes{
				IDField: tt.fields.IDField,
				Version: tt.fields.Version,
				Nodes:   tt.fields.Nodes,
			}
			if err := ns.unmarshalStateNodesJSON(tt.args.obj); (err != nil) != tt.wantErr {
				t.Errorf("unmarshalStateNodesJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				assert.Equal(t, tt.want, ns)
			}
		})
	}
}

func TestStateNodesMarshalMsgpack(t *testing.T) {
	nodes := makeTestStateNodes()
	d, err := msgpack.Marshal(nodes)
	require.NoError(t, err)

	nd := &Nodes{}
	err = msgpack.Unmarshal(d, nd)
	require.NoError(t, err)

	require.Equal(t, nodes, nd)
}
