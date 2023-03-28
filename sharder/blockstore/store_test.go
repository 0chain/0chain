package blockstore

import (
	"reflect"
	"testing"
)

func TestGetStore(t *testing.T) {
	t.Skip("need protect Store global to avoid races")
	t.Parallel()

	fsbs, cleanUp := makeTestFSBlockStore(t)
	defer cleanUp()

	tests := []struct {
		name string
		want BlockStore
	}{
		{
			name: "Test_GetStore_OK",
			want: fsbs,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			Store = fsbs

			if got := GetStore(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetStore() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSetupStore(t *testing.T) {
	t.Parallel()

	fsbs, cleanUp := makeTestFSBlockStore(t)
	defer cleanUp()

	type args struct {
		store BlockStore
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Test_SetupStore_OK",
			args: args{store: fsbs},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			SetupStore(tt.args.store)

			if !reflect.DeepEqual(Store, tt.args.store) {
				t.Errorf("got setted = %v, want %v", Store, tt.args.store)
			}
		})
	}
}
