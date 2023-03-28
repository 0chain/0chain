package state

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/0chain/common/core/util"
)

func TestPartialState_SaveState(t *testing.T) {
	t.Parallel()

	ps := &PartialState{
		Hash: util.Key("key"),
	}
	ps.mndb = util.NewMemoryNodeDB()
	db := util.NewMemoryNodeDB()
	err := db.PutNode(util.Key("node key"), util.NewFullNode(&util.SecureSerializableValue{Buffer: []byte("data")}))
	require.NoError(t, err)

	type fields struct {
		Hash      util.Key
		Version   string
		StartRoot util.Key
		Nodes     []util.Node
		mndb      *util.MemoryNodeDB
		root      util.Node
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
	var err error
	ps.mndb, err = ps.newNodeDB()
	require.NoError(t, err)
	ps.root, err = ps.mndb.ComputeRoot()
	require.NoError(t, err)
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
		err    error
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
			err: errors.New("partial state root hash mismatch"),
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

			err := ps.ComputeProperties()
			require.Equal(t, tt.err, err)
			if err != nil {
				return
			}
			assert.Equal(t, tt.want, ps)
		})
	}
}
