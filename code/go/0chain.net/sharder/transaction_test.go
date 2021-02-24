package sharder

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/round"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/cache"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
)

func TestChain_GetTransactionSummary(t *testing.T) {
	t.Parallel()

	type fields struct {
		Chain          *chain.Chain
		BlockChannel   chan *block.Block
		RoundChannel   chan *round.Round
		BlockCache     cache.Cache
		BlockTxnCache  cache.Cache
		SharderStats   Stats
		BlockSyncStats *SyncStats
		TieringStats   *MinioStats
	}
	type args struct {
		ctx  context.Context
		hash string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *transaction.TransactionSummary
		wantErr bool
	}{
		{
			name: "Test_Chain_GetTransactionSummary_OK",
			args: args{
				ctx:  common.GetRootContext(),
				hash: encryption.Hash("data"),
			},
			want:    datastore.GetEntityMetadata("txn_summary").Instance().(*transaction.TransactionSummary),
			wantErr: false,
		},
		{
			name: "Test_Chain_GetTransactionSummary_ERR",
			args: args{
				ctx:  common.GetRootContext(),
				hash: encryption.Hash("data")[:62],
			},
			want:    datastore.GetEntityMetadata("txn_summary").Instance().(*transaction.TransactionSummary),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sc := &Chain{
				Chain:          tt.fields.Chain,
				BlockChannel:   tt.fields.BlockChannel,
				RoundChannel:   tt.fields.RoundChannel,
				BlockCache:     tt.fields.BlockCache,
				BlockTxnCache:  tt.fields.BlockTxnCache,
				SharderStats:   tt.fields.SharderStats,
				BlockSyncStats: tt.fields.BlockSyncStats,
				TieringStats:   tt.fields.TieringStats,
			}
			got, err := sc.GetTransactionSummary(tt.args.ctx, tt.args.hash)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetTransactionSummary() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetTransactionSummary() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_txnSummaryCreateMV(t *testing.T) {
	tarT := "target table"
	srcT := "src table"

	type args struct {
		targetTable string
		srcTable    string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Test_txnSummaryCreateMV_OK",
			args: args{
				targetTable: tarT,
				srcTable:    srcT,
			},
			want: fmt.Sprintf(
				"CREATE MATERIALIZED VIEW IF NOT EXISTS %v AS SELECT ROUND, "+
					"HASH FROM %v WHERE ROUND IS NOT NULL PRIMARY KEY (ROUND, HASH)", tarT, srcT),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := txnSummaryCreateMV(tt.args.targetTable, tt.args.srcTable); got != tt.want {
				t.Errorf("txnSummaryCreateMV() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getCreateIndex(t *testing.T) {
	table := "table"
	column := "column"

	type args struct {
		table  string
		column string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Test_getCreateIndex_OK",
			args: args{table: table, column: column},
			want: fmt.Sprintf("CREATE INDEX IF NOT EXISTS ON %v(%v)", table, column),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getCreateIndex(tt.args.table, tt.args.column); got != tt.want {
				t.Errorf("getCreateIndex() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getSelectCountTxn(t *testing.T) {
	table := "table"
	column := "column"

	type args struct {
		table  string
		column string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Test_getSelectCountTxn_OK",
			args: args{table: table, column: column},
			want: fmt.Sprintf("SELECT COUNT(*) FROM %v where %v=?", table, column),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getSelectCountTxn(tt.args.table, tt.args.column); got != tt.want {
				t.Errorf("getSelectCountTxn() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getSelectTxn(t *testing.T) {
	table := "table"
	column := "column"

	type args struct {
		table  string
		column string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Test_getSelectTxn_OK",
			args: args{table: table, column: column},
			want: fmt.Sprintf("SELECT round FROM %v where %v=?", table, column),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getSelectTxn(tt.args.table, tt.args.column); got != tt.want {
				t.Errorf("getSelectTxn() = %v, want %v", got, tt.want)
			}
		})
	}
}
