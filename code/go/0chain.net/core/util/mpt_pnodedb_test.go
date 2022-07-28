package util

import (
	"context"
	"encoding/binary"
	"testing"

	"github.com/0chain/gorocksdb"
	"github.com/stretchr/testify/require"
)

func TestPNodeDB_Iterate(t *testing.T) {
	t.Parallel()

	type args struct {
		ctx     context.Context
		handler NodeDBIteratorHandler
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Test_PNodeDB_Iterate_Err_Reading_OK",
			args: args{
				handler: func(_ context.Context, k Key, _ Node) error {
					return nil
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {

			pndb, cleanUp := newPNodeDB(t)
			defer cleanUp()

			wo := gorocksdb.NewDefaultWriteOptions()
			for i := 0; i < 1000; i++ {
				key := make([]byte, 8)
				binary.BigEndian.PutUint64(key, uint64(i))

				err := pndb.db.Put(wo, key, []byte{NodeTypeValueNode})
				require.NoError(t, err)
			}
			err := pndb.db.Put(wo, []byte("key"), make([]byte, 0))
			require.NoError(t, err)

			if err := pndb.Iterate(tt.args.ctx, tt.args.handler); (err != nil) != tt.wantErr {
				t.Errorf("Iterate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
