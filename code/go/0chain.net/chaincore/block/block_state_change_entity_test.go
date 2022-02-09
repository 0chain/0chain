package block

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"0chain.net/chaincore/state"
	"0chain.net/core/datastore"
	"0chain.net/core/memorystore"
	"0chain.net/core/mocks"
	"0chain.net/core/util"
	"github.com/stretchr/testify/require"
	"github.com/vmihailenco/msgpack/v5"
)

func init() {
	SetupStateChange(memorystore.GetStorageProvider())
}

func newBSC(state util.MerklePatriciaTrieI) *StateChange {
	bsc := datastore.GetEntityMetadata("block_state_change").Instance().(*StateChange)
	var changes []*util.NodeChange
	bsc.Hash, changes, _, bsc.StartRoot = state.GetChanges()
	bsc.Nodes = make([]util.Node, len(changes))
	for idx, change := range changes {
		bsc.Nodes[idx] = change.New
	}
	bsc.ComputeProperties()
	return bsc
}

func TestStateChangeComputeRoot(t *testing.T) {
	initPathValues := [][]string{
		{"1234", "1234"},
		{"12", "12"},
		{"1235", "1235"},
		{"123", "123"},
	}

	newPathValues := [][]string{
		{"12345", "12345"},
		{"1235", "1235A"},
	}

	clientState := util.NewMerklePatriciaTrie(util.NewMemoryNodeDB(), 1, nil)
	for _, pv := range initPathValues {
		_, err := clientState.Insert(util.Path(pv[0]), &util.SecureSerializableValue{Buffer: []byte(pv[1])})
		require.NoError(t, err)
	}

	bsc := newBSC(clientState)
	require.Equal(t, bsc.GetRoot().GetHash(), util.ToHex(clientState.GetRoot()))

	// apply new updates
	newClientState := util.NewMerklePatriciaTrie(clientState.GetNodeDB(), 2, clientState.GetRoot())
	for _, pv := range newPathValues {
		_, err := newClientState.Insert(util.Path(pv[0]), &util.SecureSerializableValue{Buffer: []byte(pv[1])})
		require.NoError(t, err)
	}

	bsc = newBSC(newClientState)
	require.Equal(t, bsc.GetRoot().GetHash(), util.ToHex(newClientState.GetRoot()))

}

func TestNewBlockStateChange(t *testing.T) {
	b := NewBlock("", 1)
	b.HashBlock()
	b.ClientState = util.NewMerklePatriciaTrie(util.NewMemoryNodeDB(), 1, nil)
	_, err := b.ClientState.Insert(util.Path("path"), &util.SecureSerializableValue{Buffer: []byte("value")})
	if err != nil {
		t.Fatal(err)
	}

	bsc := datastore.GetEntityMetadata("block_state_change").Instance().(*StateChange)
	bsc.Block = b.Hash
	var changes []*util.NodeChange
	bsc.Hash, changes, _, bsc.StartRoot = b.ClientState.GetChanges()
	bsc.Nodes = make([]util.Node, len(changes))
	for idx, change := range changes {
		bsc.Nodes[idx] = change.New
	}
	bsc.ComputeProperties()

	type args struct {
		b *Block
	}
	tests := []struct {
		name string
		args args
		want *StateChange
	}{
		{
			name: "OK",
			args: args{b: b},
			want: bsc,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewBlockStateChange(tt.args.b); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewBlockStateChange() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStateChange_Read(t *testing.T) {
	sm := mocks.Store{}
	stateChangeEntityMetadata.Store = &sm
	sm.On("Read", context.Context(nil), "", new(StateChange)).Return(
		func(_ context.Context, _ string, _ datastore.Entity) error {
			return nil
		},
	)

	type fields struct {
		PartialState state.PartialState
		Block        string
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
		t.Run(tt.name, func(t *testing.T) {
			sc := &StateChange{
				PartialState: tt.fields.PartialState,
				Block:        tt.fields.Block,
			}
			if err := sc.Read(tt.args.ctx, tt.args.key); (err != nil) != tt.wantErr {
				t.Errorf("Read() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStateChange_Write(t *testing.T) {
	sm := mocks.Store{}
	stateChangeEntityMetadata.Store = &sm
	sm.On("Write", context.Context(nil), new(StateChange)).Return(
		func(_ context.Context, _ datastore.Entity) error {
			return nil
		},
	)

	type fields struct {
		PartialState state.PartialState
		Block        string
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
		t.Run(tt.name, func(t *testing.T) {
			sc := &StateChange{
				PartialState: tt.fields.PartialState,
				Block:        tt.fields.Block,
			}
			if err := sc.Write(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("Write() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStateChange_Delete(t *testing.T) {
	sm := mocks.Store{}
	stateChangeEntityMetadata.Store = &sm
	sm.On("Delete", context.Context(nil), new(StateChange)).Return(
		func(_ context.Context, _ datastore.Entity) error {
			return nil
		},
	)

	type fields struct {
		PartialState state.PartialState
		Block        string
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
		t.Run(tt.name, func(t *testing.T) {
			sc := &StateChange{
				PartialState: tt.fields.PartialState,
				Block:        tt.fields.Block,
			}
			if err := sc.Delete(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStateChange_MarshalJSON(t *testing.T) {
	b := NewBlock("", 1)
	b.ClientState = util.NewMerklePatriciaTrie(util.NewMemoryNodeDB(), 1, nil)
	_, err := b.ClientState.Insert(util.Path("path"), &util.SecureSerializableValue{Buffer: []byte("value")})
	if err != nil {
		t.Fatal(err)
	}
	b.HashBlock()
	sc := NewBlockStateChange(b)
	var data = make(map[string]interface{})
	data["block"] = sc.Block
	want, err := sc.MarshalPartialStateJSON(data)
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		PartialState state.PartialState
		Block        string
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
				PartialState: sc.PartialState,
				Block:        sc.Block,
			},
			want:    want,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := &StateChange{
				PartialState: tt.fields.PartialState,
				Block:        tt.fields.Block,
			}
			got, err := sc.MarshalJSON()
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

func TestStateChange_UnmarshalJSON(t *testing.T) {
	b := NewBlock("", 1)
	b.ClientState = util.NewMerklePatriciaTrie(util.NewMemoryNodeDB(), 1, nil)
	_, err := b.ClientState.Insert(util.Path("path"), &util.SecureSerializableValue{Buffer: []byte("value")})
	if err != nil {
		t.Fatal(err)
	}
	b.HashBlock()
	sc := NewBlockStateChange(b)
	blob, err := sc.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		PartialState state.PartialState
		Block        string
	}
	type args struct {
		data []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "OK",
			fields: fields{
				PartialState: sc.PartialState,
				Block:        sc.Block,
			},
			args:    args{data: blob},
			wantErr: false,
		},
		{
			name:    "Invalid_Data_ERR",
			args:    args{data: []byte("}{")},
			wantErr: true,
		},
		{
			name: "Missing_Block_ERR",
			args: args{
				data: func() []byte {
					m := map[string]interface{}{
						"some data key": "some data value",
					}

					blob, err := json.Marshal(m)
					if err != nil {
						t.Fatal(err)
					}

					return blob
				}(),
			},
			wantErr: true,
		},
		{
			name: "Invalid_Block_ERR",
			args: args{
				data: func() []byte {
					m := map[string]interface{}{
						"block": 123,
					}

					blob, err := json.Marshal(m)
					if err != nil {
						t.Fatal(err)
					}

					return blob
				}(),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := &StateChange{
				PartialState: tt.fields.PartialState,
				Block:        tt.fields.Block,
			}
			if err := sc.UnmarshalJSON(tt.args.data); (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStateChange_UnmarshalMsgpack(t *testing.T) {
	b := NewBlock("", 1)
	b.ClientState = util.NewMerklePatriciaTrie(util.NewMemoryNodeDB(), 1, nil)
	_, err := b.ClientState.Insert(util.Path("path"), &util.SecureSerializableValue{Buffer: []byte("value")})
	if err != nil {
		t.Fatal(err)
	}
	b.HashBlock()
	sc := NewBlockStateChange(b)
	blob, err := msgpack.Marshal(sc)
	if err != nil {
		t.Fatal(err)
	}

	nsc := StateChange{}
	err = msgpack.Unmarshal(blob, &nsc)
	require.NoError(t, err)

	require.Equal(t, nsc.Block, b.Hash)
	require.Equal(t, nsc.StartRoot, sc.StartRoot)
	require.Equal(t, nsc.Hash, sc.Hash)
	require.Equal(t, nsc.Version, sc.Version)
}
