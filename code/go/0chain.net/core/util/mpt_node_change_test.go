package util

import (
	"bytes"
	"fmt"
	"github.com/0chain/gorocksdb"
	"reflect"
	"testing"
)

func TestChangeCollector_AddChange(t *testing.T) {
	cc := &ChangeCollector{
		Changes: make(map[string]*NodeChange),
		Deletes: make(map[string]Node),
	}

	nn := NewValueNode()
	cc.Deletes[nn.GetHash()] = nn

	on := NewFullNode(&AState{balance: 2})
	cc.Changes[on.GetHash()] = &NodeChange{Old: nn}

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
				Changes: cc.Changes,
				Deletes: cc.Deletes,
			},
			args: args{newNode: nn},
		},
		{
			name: "Test_ChangeCollector_AddChange_OK",
			fields: fields{
				Changes: cc.Changes,
				Deletes: cc.Deletes,
			},
			args: args{newNode: nn, oldNode: on},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cc := &ChangeCollector{
				Changes: tt.fields.Changes,
				Deletes: tt.fields.Deletes,
			}

			cc.AddChange(tt.args.oldNode, tt.args.newNode)
		})
	}
}

func TestChangeCollector_UpdateChanges(t *testing.T) {
	cc := &ChangeCollector{
		Changes: make(map[string]*NodeChange),
		Deletes: make(map[string]Node),
	}

	n := NewValueNode()
	cc.Changes[n.GetHash()] = &NodeChange{New: n}

	pndb, err := NewPNodeDB(dataDir, "")
	if err != nil {
		t.Fatal(err)
	}
	defer pndb.Close()

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
				Changes: cc.Changes,
				Deletes: cc.Deletes,
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

	if err := cleanUp(); err != nil {
		t.Fatal(err)
	}
}

func TestChangeCollector_PrintChanges(t *testing.T) {
	cc := &ChangeCollector{
		Changes: make(map[string]*NodeChange),
		Deletes: make(map[string]Node),
	}

	n := NewValueNode()
	cc.Changes[n.GetHash()] = &NodeChange{Old: n, New: n}

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
				Changes: cc.Changes,
				Deletes: cc.Deletes,
			},
			wantW: func() string {
				w := &bytes.Buffer{}
				fmt.Fprintf(w, "cc(%v): nn=%v on=%v\n", "", n.GetHash(), n.GetHash())

				return w.String()
			}(),
		},
		{
			name: "Test_ChangeCollector_PrintChanges_OK2",
			fields: fields{
				Changes: func() map[string]*NodeChange {
					m := make(map[string]*NodeChange)
					m[n.GetHash()] = &NodeChange{New: n}

					return m
				}(),
				Deletes: make(map[string]Node),
			},
			wantW: func() string {
				w := &bytes.Buffer{}
				fmt.Fprintf(w, "cc(%v): nn=%v\n", "", n.GetHash())

				return w.String()
			}(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
	cc := &ChangeCollector{
		Changes: make(map[string]*NodeChange),
		Deletes: make(map[string]Node),
	}

	n := NewValueNode()
	cc.Changes[n.GetHash()] = &NodeChange{Old: n, New: n}
	cc.Deletes[n.GetHash()] = n

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
				Changes: cc.Changes,
				Deletes: cc.Deletes,
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
		t.Run(tt.name, func(t *testing.T) {
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
	cc := &ChangeCollector{
		Changes: make(map[string]*NodeChange),
		Deletes: make(map[string]Node),
	}

	n := NewValueNode()
	cc.Changes[n.GetHash()] = &NodeChange{Old: n, New: n}
	cc.Deletes[n.GetHash()] = n

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
				Changes: cc.Changes,
				Deletes: cc.Deletes,
			},
			want: cc,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
