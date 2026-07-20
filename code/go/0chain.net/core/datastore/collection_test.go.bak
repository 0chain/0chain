package datastore_test

import (
	"reflect"
	"testing"
	"time"

	"0chain.net/core/datastore"
)

func TestEntityCollection_Copy(t *testing.T) {
	t.Parallel()

	c := datastore.EntityCollection{
		CollectionName:     "name",
		CollectionSize:     1,
		CollectionDuration: 1,
	}

	type fields struct {
		CollectionName     string
		CollectionSize     int64
		CollectionDuration time.Duration
	}
	tests := []struct {
		name   string
		fields fields
		wantCp *datastore.EntityCollection
	}{
		{
			name: "Test_EntityCollection_Copy_OK",
			fields: fields{
				CollectionName:     c.CollectionName,
				CollectionSize:     c.CollectionSize,
				CollectionDuration: c.CollectionDuration,
			},
			wantCp: &c,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ec := &datastore.EntityCollection{
				CollectionName:     tt.fields.CollectionName,
				CollectionSize:     tt.fields.CollectionSize,
				CollectionDuration: tt.fields.CollectionDuration,
			}
			if gotCp := ec.Clone(); !reflect.DeepEqual(gotCp, tt.wantCp) {
				t.Errorf("Clone() = %v, want %v", gotCp, tt.wantCp)
			}
		})
	}
}

func TestEntityCollection_GetCollectionName(t *testing.T) {
	t.Parallel()

	type fields struct {
		CollectionName     string
		CollectionSize     int64
		CollectionDuration time.Duration
	}
	type args struct {
		parent datastore.Key
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{
		{
			name:   "Test_EntityCollection_GetCollectionName_OK",
			fields: fields{CollectionName: "name"},
			args:   args{parent: "parent"},
			want:   "name:parent",
		},
		{
			name:   "Test_EntityCollection_GetCollectionName_Empty_Parent_OK",
			fields: fields{CollectionName: "name"},
			args:   args{parent: ""},
			want:   "name",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			eq := &datastore.EntityCollection{
				CollectionName:     tt.fields.CollectionName,
				CollectionSize:     tt.fields.CollectionSize,
				CollectionDuration: tt.fields.CollectionDuration,
			}
			if got := eq.GetCollectionName(tt.args.parent); got != tt.want {
				t.Errorf("GetCollectionName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCollectionMemberField_Get(t *testing.T) {
	t.Parallel()

	c := datastore.CollectionMemberField{
		EntityCollection: &datastore.EntityCollection{
			CollectionName:     "name",
			CollectionSize:     1,
			CollectionDuration: 2,
		},
		CollectionScore: 3,
	}

	type fields struct {
		EntityCollection *datastore.EntityCollection
		CollectionScore  int64
	}
	tests := []struct {
		name         string
		fields       fields
		wantName     string
		wantSize     int64
		WantDuration time.Duration
		wantScore    int64
	}{
		{
			name: "TestCollectionMemberField_Get_OK",
			fields: fields{
				EntityCollection: c.EntityCollection,
				CollectionScore:  c.CollectionScore,
			},
			wantName:     c.EntityCollection.CollectionName,
			wantSize:     c.EntityCollection.CollectionSize,
			WantDuration: c.EntityCollection.CollectionDuration,
			wantScore:    c.CollectionScore,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cf := &datastore.CollectionMemberField{
				EntityCollection: tt.fields.EntityCollection,
				CollectionScore:  tt.fields.CollectionScore,
			}

			if got := cf.GetCollectionName(); got != tt.wantName {
				t.Errorf("GetCollectionName() = %v, want %v", got, tt.wantName)
			}
			if got := cf.GetCollectionSize(); got != tt.wantSize {
				t.Errorf("GetCollectionSize() = %v, want %v", got, tt.wantSize)
			}
			if got := cf.GetCollectionDuration(); got != tt.WantDuration {
				t.Errorf("GetCollectionDuration() = %v, want %v", got, tt.WantDuration)
			}
			if got := cf.GetCollectionScore(); got != tt.wantScore {
				t.Errorf("GetCollectionScore() = %v, want %v", got, tt.wantScore)
			}
		})
	}
}

func TestCollectionMemberField_SetCollectionScore(t *testing.T) {
	t.Parallel()

	c := datastore.CollectionMemberField{
		EntityCollection: &datastore.EntityCollection{
			CollectionName:     "name",
			CollectionSize:     1,
			CollectionDuration: 2,
		},
		CollectionScore: 3,
	}

	type fields struct {
		EntityCollection *datastore.EntityCollection
		CollectionScore  int64
	}
	type args struct {
		score int64
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "Test_CollectionMemberField_SetCollectionScore_OK",
			fields: fields{
				EntityCollection: c.EntityCollection,
				CollectionScore:  c.CollectionScore,
			},
			args: args{score: 123},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cf := &datastore.CollectionMemberField{
				EntityCollection: tt.fields.EntityCollection,
				CollectionScore:  tt.fields.CollectionScore,
			}

			cf.SetCollectionScore(tt.args.score)

			if cf.CollectionScore != tt.args.score {
				t.Errorf("SetCollectionScore() not setted corresponding field")
			}
		})
	}
}

func TestCollectionMemberField_InitCollectionScore(t *testing.T) {
	t.Parallel()

	c := datastore.CollectionMemberField{
		EntityCollection: &datastore.EntityCollection{
			CollectionName:     "name",
			CollectionSize:     1,
			CollectionDuration: 2,
		},
		CollectionScore: 3,
	}

	type fields struct {
		EntityCollection *datastore.EntityCollection
		CollectionScore  int64
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "Test_CollectionMemberField_InitCollectionScore_OK",
			fields: fields{
				EntityCollection: c.EntityCollection,
				CollectionScore:  c.CollectionScore,
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cf := &datastore.CollectionMemberField{
				EntityCollection: tt.fields.EntityCollection,
				CollectionScore:  tt.fields.CollectionScore,
			}

			cf.InitCollectionScore()
		})
	}
}

func TestGetCollectionScore(t *testing.T) {
	t.Parallel()

	now := time.Now()

	type args struct {
		ts time.Time
	}
	tests := []struct {
		name string
		args args
		want int64
	}{
		{
			name: "TestGetCollectionScore_OK",
			args: args{ts: now},
			want: -now.UnixNano() / int64(time.Millisecond),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := datastore.GetCollectionScore(tt.args.ts); got != tt.want {
				t.Errorf("GetCollectionScore() = %v, want %v", got, tt.want)
			}
		})
	}
}
