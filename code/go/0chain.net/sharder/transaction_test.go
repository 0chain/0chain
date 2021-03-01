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
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
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

func TestChain_storeTransactions(t *testing.T) {
	sc := GetSharderChain()

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
		ctx   context.Context
		sTxns []datastore.Entity
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Test_Chain_storeTransactions_OK",
			fields: fields{
				Chain:          sc.Chain,
				BlockChannel:   sc.BlockChannel,
				RoundChannel:   sc.RoundChannel,
				BlockCache:     sc.BlockCache,
				BlockTxnCache:  sc.BlockTxnCache,
				SharderStats:   sc.SharderStats,
				BlockSyncStats: sc.BlockSyncStats,
				TieringStats:   sc.TieringStats,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
			if err := sc.storeTransactions(tt.args.ctx, tt.args.sTxns); (err != nil) != tt.wantErr {
				t.Errorf("storeTransactions() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestChain_StoreTransactions(t *testing.T) {
	sc := GetSharderChain()

	b := block.NewBlock("", 1)
	b.Txns = []*transaction.Transaction{
		&transaction.Transaction{},
	}

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
		ctx context.Context
		b   *block.Block
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Test_Chain_StoreTransactions_OK",
			fields: fields{
				Chain:          sc.Chain,
				BlockChannel:   sc.BlockChannel,
				RoundChannel:   sc.RoundChannel,
				BlockCache:     sc.BlockCache,
				BlockTxnCache:  sc.BlockTxnCache,
				SharderStats:   sc.SharderStats,
				BlockSyncStats: sc.BlockSyncStats,
				TieringStats:   sc.TieringStats,
			},
			args: args{
				ctx: common.GetRootContext(),
				b:   b,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
			if err := sc.StoreTransactions(tt.args.ctx, tt.args.b); (err != nil) != tt.wantErr {
				t.Errorf("StoreTransactions() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestChain_GetTransactionConfirmation(t *testing.T) {
	sc := GetSharderChain()

	// case 1
	txn := transaction.Transaction{}
	txn.Hash = encryption.Hash("txn data")
	if err := sc.BlockTxnCache.Add(txn.Hash, txn.GetSummary()); err != nil {
		t.Fatal(err)
	}

	// case 2
	var (
		r2   = round.NewRound(2)
		h2   = encryption.Hash("txn2 data")
		txn2 = transaction.Transaction{}
		b2   = block.NewBlock("", r2.Number)
		ts2  = txn2.GetSummary()
	)
	r2.BlockHash = h2
	txn2.Hash = h2
	b2.Hash = h2
	ts2.Round = r2.Number
	if err := sc.BlockTxnCache.Add(h2, ts2); err != nil {
		t.Fatal(err)
	}
	sc.AddRound(r2)

	// case 3
	var (
		r3   = round.NewRound(13235)
		h3   = encryption.Hash("txn3 data")
		txn3 = transaction.Transaction{}
		b3   = block.NewBlock("", r3.Number)
		ts3  = txn3.GetSummary()
	)
	r3.BlockHash = h3
	txn3.Hash = h3
	b3.Hash = h3
	b3.Txns = append(b3.Txns, &txn3)
	ts3.Round = r3.Number
	if err := sc.BlockTxnCache.Add(h3, ts3); err != nil {
		t.Fatal(err)
	}
	sc.AddRound(r3)
	bs3 := b3.GetSummary()
	if err := bs3.GetEntityMetadata().GetStore().Write(common.GetRootContext(), bs3); err != nil {
		t.Fatal(err)
	}
	sc.AddBlock(b3)

	// case 4
	var (
		r4   = round.NewRound(1234)
		h4   = encryption.Hash("txn4 data")
		txn4 = transaction.Transaction{}
		b4   = block.NewBlock("", r4.Number)
		ts4  = txn4.GetSummary()
	)
	r4.BlockHash = h4
	txn4.Hash = h4
	b4.Hash = h4
	b4.Txns = append(b4.Txns, &txn4)
	ts4.Round = r4.Number
	if err := sc.BlockTxnCache.Add(h4, ts4); err != nil {
		t.Fatal(err)
	}
	sc.AddRound(r4)
	if err := sc.BlockCache.Add(h4, b4); err != nil {
		t.Fatal(err)
	}
	sc.AddBlock(b4)

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
		want    *transaction.Confirmation
		wantErr bool
	}{
		{
			name: "Test_Chain_GetTransactionConfirmation_TS_No_Cache_No_Store_ERR",
			fields: fields{
				Chain:          sc.Chain,
				BlockChannel:   sc.BlockChannel,
				RoundChannel:   sc.RoundChannel,
				BlockCache:     sc.BlockCache,
				BlockTxnCache:  sc.BlockTxnCache,
				SharderStats:   sc.SharderStats,
				BlockSyncStats: sc.BlockSyncStats,
				TieringStats:   sc.TieringStats,
			},
			args: args{
				ctx:  common.GetRootContext(),
				hash: txn.Hash[:62],
			},
			wantErr: true,
		},
		{
			name: "Test_Chain_GetTransactionConfirmation_TS_In_Cache_But_Not_In_Store_ERR",
			fields: fields{
				Chain:          sc.Chain,
				BlockChannel:   sc.BlockChannel,
				RoundChannel:   sc.RoundChannel,
				BlockCache:     sc.BlockCache,
				BlockTxnCache:  sc.BlockTxnCache,
				SharderStats:   sc.SharderStats,
				BlockSyncStats: sc.BlockSyncStats,
				TieringStats:   sc.TieringStats,
			},
			args: args{
				ctx:  common.GetRootContext(),
				hash: txn.Hash,
			},
			wantErr: true,
		},
		{
			name: "Test_Chain_GetTransactionConfirmation_Block_No_Cache_No_Store_ERR",
			fields: fields{
				Chain:          sc.Chain,
				BlockChannel:   sc.BlockChannel,
				RoundChannel:   sc.RoundChannel,
				BlockCache:     sc.BlockCache,
				BlockTxnCache:  sc.BlockTxnCache,
				SharderStats:   sc.SharderStats,
				BlockSyncStats: sc.BlockSyncStats,
				TieringStats:   sc.TieringStats,
			},
			args: args{
				ctx:  common.GetRootContext(),
				hash: txn2.Hash,
			},
			wantErr: true,
		},
		{
			name: "Test_Chain_GetTransactionConfirmation_Block_No_Cache_But_In_Store_OK",
			fields: fields{
				Chain:          sc.Chain,
				BlockChannel:   sc.BlockChannel,
				RoundChannel:   sc.RoundChannel,
				BlockCache:     sc.BlockCache,
				BlockTxnCache:  sc.BlockTxnCache,
				SharderStats:   sc.SharderStats,
				BlockSyncStats: sc.BlockSyncStats,
				TieringStats:   sc.TieringStats,
			},
			args: args{
				ctx:  common.GetRootContext(),
				hash: txn3.Hash,
			},
			wantErr: false,
			want: func() *transaction.Confirmation {
				c := &transaction.Confirmation{
					Version:           "1.0",
					Hash:              h3,
					BlockHash:         h3,
					Round:             r3.Number,
					Transaction:       &txn3,
					CreationDateField: datastore.CreationDateField{},
				}

				mt := b3.GetMerkleTree()
				c.MerkleTreeRoot = mt.GetRoot()
				c.MerkleTreePath = mt.GetPath(c)
				rmt := b3.GetReceiptsMerkleTree()
				c.ReceiptMerkleTreeRoot = rmt.GetRoot()
				c.ReceiptMerkleTreePath = rmt.GetPath(transaction.NewTransactionReceipt(&txn3))

				return c
			}(),
		},
		{
			name: "Test_Chain_GetTransactionConfirmation_Block_In_Cache_OK",
			fields: fields{
				Chain:          sc.Chain,
				BlockChannel:   sc.BlockChannel,
				RoundChannel:   sc.RoundChannel,
				BlockCache:     sc.BlockCache,
				BlockTxnCache:  sc.BlockTxnCache,
				SharderStats:   sc.SharderStats,
				BlockSyncStats: sc.BlockSyncStats,
				TieringStats:   sc.TieringStats,
			},
			args: args{
				ctx:  common.GetRootContext(),
				hash: txn4.Hash,
			},
			wantErr: false,
			want: func() *transaction.Confirmation {
				c := &transaction.Confirmation{
					Version:           "1.0",
					Hash:              h4,
					BlockHash:         h4,
					Round:             b4.Round,
					Transaction:       &txn4,
					CreationDateField: datastore.CreationDateField{},
				}

				mt := b4.GetMerkleTree()
				c.MerkleTreeRoot = mt.GetRoot()
				c.MerkleTreePath = mt.GetPath(c)
				rmt := b4.GetReceiptsMerkleTree()
				c.ReceiptMerkleTreeRoot = rmt.GetRoot()
				c.ReceiptMerkleTreePath = rmt.GetPath(transaction.NewTransactionReceipt(&txn4))

				return c
			}(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
			got, err := sc.GetTransactionConfirmation(tt.args.ctx, tt.args.hash)
			if !tt.wantErr && got != nil {
				got.CreationDateField = datastore.CreationDateField{}
				got.RoundRandomSeed = 0
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("GetTransactionConfirmation() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetTransactionConfirmation() got = %#v, want %#v", got, tt.want)
			}
		})
	}
}
