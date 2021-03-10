package memorystore_test

import (
	"0chain.net/chaincore/block"
	"0chain.net/core/datastore"
	"0chain.net/core/ememorystore"
	"0chain.net/core/memorystore"
	"testing"
)

func init() {
	block.SetupEntity(ememorystore.GetStorageProvider())
}

func TestGetEntityKey(t *testing.T) {
	t.Parallel()

	type args struct {
		entity datastore.Entity
	}
	tests := []struct {
		name string
		args args
		want datastore.Key
	}{
		{
			name: "Test_GetEntityKey_String_OK",
			args: args{entity: &block.Block{}},
			want: "block" + ":",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := memorystore.GetEntityKey(tt.args.entity); got != tt.want {
				t.Errorf("GetEntityKey() = %v, want %v", got, tt.want)
			}
		})
	}
}
