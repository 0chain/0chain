package state

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"0chain.net/core/util"
)

func TestPartialState_GetNodeDB(t *testing.T) {
	t.Parallel()

	type fields struct {
		Hash    util.Key
		Version string
		Nodes   []util.Node
		mndb    *util.MemoryNodeDB
		root    util.Node
	}
	tests := []struct {
		name   string
		fields fields
		want   util.NodeDB
	}{
		{
			name: "OK",
			fields: fields{
				mndb: util.NewMemoryNodeDB(),
			},
			want: util.NewMemoryNodeDB(),
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
				mndb:    tt.fields.mndb,
				root:    tt.fields.root,
			}
			ps.ComputeProperties()
			if got := ps.GetNodeDB(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetNodeDB() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPartialState_SaveState(t *testing.T) {
	t.Parallel()

	ps := NewPartialState(util.Key("key"))
	ps.mndb = util.NewMemoryNodeDB()
	db := util.NewMemoryNodeDB()
	err := db.PutNode(util.Key("node key"), util.NewFullNode(&util.SecureSerializableValue{Buffer: []byte("data")}))
	require.NoError(t, err)

	type fields struct {
		Hash    util.Key
		Version string
		Nodes   []util.Node
		mndb    *util.MemoryNodeDB
		root    util.Node
	}
	type args struct {
		ctx     context.Context
		stateDB util.NodeDB
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
			fields:  fields(*ps),
			args:    args{stateDB: db},
			wantErr: false,
			want: func() *PartialState {
				mndb := *ps.mndb
				err := util.MergeState(context.TODO(), &mndb, db)
				require.NoError(t, err)

				ps := *ps
				ps.mndb = &mndb
				return &ps
			}(),
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
				mndb:    tt.fields.mndb,
				root:    tt.fields.root,
			}
			if err := ps.SaveState(tt.args.ctx, tt.args.stateDB); (err != nil) != tt.wantErr {
				t.Errorf("SaveState() error = %v, wantErr %v", err, tt.wantErr)
			}
			assert.Equal(t, tt.want, ps)
		})
	}
}

func TestPartialState_ComputeProperties(t *testing.T) {
	t.Parallel()

	ps := PartialState{
		Nodes: []util.Node{
			util.NewFullNode(&util.SecureSerializableValue{Buffer: []byte("value")}),
		},
	}
	ps.mndb = ps.newNodeDB()
	ps.root = ps.mndb.ComputeRoot()
	ps.Hash = ps.root.GetHashBytes()

	type fields struct {
		Hash    util.Key
		Version string
		Nodes   []util.Node
		mndb    *util.MemoryNodeDB
		root    util.Node
	}
	tests := []struct {
		name   string
		fields fields
		want   *PartialState
	}{
		{
			name: "OK",
			fields: fields{
				Hash:  ps.Hash,
				Nodes: ps.Nodes,
			},
			want: &ps,
		},
		{
			name: "OK2",
			fields: fields{
				Nodes: ps.Nodes,
			},
			want: &PartialState{
				Nodes: ps.Nodes,
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
				mndb:    tt.fields.mndb,
				root:    tt.fields.root,
			}

			ps.ComputeProperties()
			assert.Equal(t, tt.want, ps)
		})
	}
}

func TestPartialState_Validate(t *testing.T) {
	t.Parallel()

	ps := PartialState{
		Nodes: []util.Node{
			util.NewFullNode(&util.SecureSerializableValue{Buffer: []byte("value")}),
		},
	}
	ps.mndb = ps.newNodeDB()
	ps.root = ps.mndb.ComputeRoot()

	type fields struct {
		Hash    util.Key
		Version string
		Nodes   []util.Node
		mndb    *util.MemoryNodeDB
		root    util.Node
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
			fields:  fields(ps),
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
				mndb:    tt.fields.mndb,
				root:    tt.fields.root,
			}
			if err := ps.Validate(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
