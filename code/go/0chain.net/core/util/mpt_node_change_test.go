package util

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"

	"github.com/0chain/gorocksdb"
	"github.com/stretchr/testify/require"
)

func TestChangeCollector_AddChange(t *testing.T) {
	t.Parallel()

	var (
		newNode = NewValueNode()
		oldNode = NewFullNode(&AState{balance: 2})
	)

	type fields struct {
		Changes map[string]*NodeChange
		Deletes map[string]Node
	}
	type args struct {
		oldNode Node
		newNode Node
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "Test_ChangeCollector_AddChange_Nil_Old_Node_OK",
			fields: fields{
				Changes: map[string]*NodeChange{
					oldNode.GetHash(): {Old: newNode},
				},
				Deletes: map[string]Node{
					newNode.GetHash(): newNode,
				},
			},
			args: args{newNode: newNode},
		},
		{
			name: "Test_ChangeCollector_AddChange_OK",
			fields: fields{
				Changes: map[string]*NodeChange{
					oldNode.GetHash(): {Old: newNode},
				},
				Deletes: map[string]Node{
					newNode.GetHash(): newNode,
				},
			},
			args: args{newNode: newNode, oldNode: oldNode},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cc := &ChangeCollector{
				Changes: tt.fields.Changes,
				Deletes: tt.fields.Deletes,
			}

			cc.AddChange(tt.args.oldNode, tt.args.newNode)
		})
	}
}

func TestChangeCollector_UpdateChanges(t *testing.T) {
	pndb, cleanup := newPNodeDB(t)
	defer cleanup()

	pndb.wo = gorocksdb.NewDefaultWriteOptions()
	pndb.wo.DisableWAL(true)
	pndb.wo.SetSync(true)

	type fields struct {
		Changes map[string]*NodeChange
		Deletes map[string]Node
	}
	type args struct {
		ndb            NodeDB
		origin         Sequence
		includeDeletes bool
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Test_ChangeCollector_UpdateChanges_OK",
			fields: fields{
				Changes: func() map[string]*NodeChange {
					ch := make(map[string]*NodeChange)
					n := NewValueNode()
					ch[n.GetHash()] = &NodeChange{New: n}
					return ch
				}(),
			},
			args:    args{ndb: pndb},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cc := &ChangeCollector{
				Changes: tt.fields.Changes,
				Deletes: tt.fields.Deletes,
			}
			if err := cc.UpdateChanges(tt.args.ndb, tt.args.origin, tt.args.includeDeletes); (err != nil) != tt.wantErr {
				t.Errorf("UpdateChanges() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestChangeCollector_PrintChanges(t *testing.T) {
	t.Parallel()

	n := NewValueNode()

	type fields struct {
		Changes map[string]*NodeChange
		Deletes map[string]Node
	}
	tests := []struct {
		name   string
		fields fields
		wantW  string
	}{
		{
			name: "Test_ChangeCollector_PrintChanges_OK",
			fields: fields{
				Changes: map[string]*NodeChange{
					n.GetHash(): {Old: n, New: n},
				},
			},
			wantW: func() string {
				w := &bytes.Buffer{}
				_, err := fmt.Fprintf(w, "cc(%v): nn=%v on=%v\n", "", n.GetHash(), n.GetHash())
				require.NoError(t, err)

				return w.String()
			}(),
		},
		{
			name: "Test_ChangeCollector_PrintChanges_OK2",
			fields: fields{
				Changes: map[string]*NodeChange{
					n.GetHash(): {New: n},
				},
				Deletes: make(map[string]Node),
			},
			wantW: func() string {
				w := &bytes.Buffer{}
				_, err := fmt.Fprintf(w, "cc(%v): nn=%v\n", "", n.GetHash())
				require.NoError(t, err)

				return w.String()
			}(),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cc := &ChangeCollector{
				Changes: tt.fields.Changes,
				Deletes: tt.fields.Deletes,
			}
			w := &bytes.Buffer{}
			cc.PrintChanges(w)
			if gotW := w.String(); gotW != tt.wantW {
				t.Errorf("PrintChanges() = %v, want %v", gotW, tt.wantW)
			}
		})
	}
}

func TestChangeCollector_Validate(t *testing.T) {
	t.Parallel()

	n := NewValueNode()

	type fields struct {
		Changes map[string]*NodeChange
		Deletes map[string]Node
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "Test_ChangeCollector_Validate_ERR",
			fields: fields{
				Changes: map[string]*NodeChange{
					n.GetHash(): {Old: n, New: n},
				},
				Deletes: map[string]Node{
					n.GetHash(): n,
				},
			},
			wantErr: true,
		},
		{
			name: "Test_ChangeCollector_Validate_OK",
			fields: fields{
				Changes: make(map[string]*NodeChange),
				Deletes: make(map[string]Node),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cc := &ChangeCollector{
				Changes: tt.fields.Changes,
				Deletes: tt.fields.Deletes,
			}
			if err := cc.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestChangeCollector_Clone(t *testing.T) {
	t.Parallel()

	n := NewValueNode()

	type fields struct {
		Changes map[string]*NodeChange
		Deletes map[string]Node
	}
	tests := []struct {
		name   string
		fields fields
		want   ChangeCollectorI
	}{
		{
			name: "Test_ChangeCollector_Clone_OK",
			fields: fields{
				Changes: map[string]*NodeChange{
					n.GetHash(): {Old: n, New: n},
				},
				Deletes: map[string]Node{
					n.GetHash(): n,
				},
			},
			want: &ChangeCollector{
				Changes: map[string]*NodeChange{
					n.GetHash(): {Old: n, New: n},
				},
				Deletes: map[string]Node{
					n.GetHash(): n,
				},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cc := &ChangeCollector{
				Changes: tt.fields.Changes,
				Deletes: tt.fields.Deletes,
			}
			if got := cc.Clone(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Clone() = %v, want %v", got, tt.want)
			}
		})
	}
}
