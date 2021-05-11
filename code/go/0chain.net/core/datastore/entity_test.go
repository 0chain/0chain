package datastore_test

import (
	"reflect"
	"testing"

	"0chain.net/chaincore/block"
	"0chain.net/core/datastore"
)

func TestAllocateEntities(t *testing.T) {
	t.Parallel()

	b := block.NewBlock("", 1)

	type args struct {
		size           int
		entityMetadata datastore.EntityMetadata
	}
	tests := []struct {
		name string
		args args
		want []datastore.Entity
	}{
		{
			name: "Test_AllocateEntities_OK",
			args: args{size: 1, entityMetadata: b.GetEntityMetadata()},
			want: []datastore.Entity{
				b.GetEntityMetadata().Instance(),
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := datastore.AllocateEntities(tt.args.size, tt.args.entityMetadata); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AllocateEntities() = %v, want %v", got, tt.want)
			}
		})
	}
}
