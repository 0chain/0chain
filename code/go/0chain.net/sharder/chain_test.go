package sharder_test

import (
	"0chain.net/chaincore/node"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/ememorystore"
	"0chain.net/core/encryption"
	"0chain.net/sharder/blockstore"
	"context"
	"encoding/binary"
	"fmt"
	"reflect"
	"testing"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/round"
	"0chain.net/core/cache"
	"0chain.net/sharder"
)

func makeTestMagicBlock(publicKey string) (*block.MagicBlock, error) {
	mb := block.NewMagicBlock()
	mb.Miners = node.NewPool(node.NodeTypeMiner)
	mb.Sharders = node.NewPool(node.NodeTypeMiner)

	id := encryption.Hash(publicKey)
	nc := map[interface{}]interface{}{
		"type":       int8(1),
		"public_ip":  "",
		"n2n_ip":     "",
		"port":       123,
		"public_key": publicKey,
		"id":         id,
	}
	n, err := node.NewNode(nc)
	if err != nil {
		return nil, err
	}
	mb.Miners.NodesMap = map[string]*node.Node{
		id: n,
	}

	return mb, nil
}

func TestChain_GetRoundChannel(t *testing.T) {
	t.Parallel()

	sc := sharder.GetSharderChain()

	type fields struct {
		Chain          *chain.Chain
		BlockChannel   chan *block.Block
		RoundChannel   chan *round.Round
		BlockCache     cache.Cache
		BlockTxnCache  cache.Cache
		SharderStats   sharder.Stats
		BlockSyncStats *sharder.SyncStats
		TieringStats   *sharder.MinioStats
	}
	tests := []struct {
		name   string
		fields fields
		want   chan *round.Round
	}{
		{
			name: "Test_Chain_GetRoundChannel_OK",
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
			want: sc.RoundChannel,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sc := &sharder.Chain{
				Chain:          tt.fields.Chain,
				BlockChannel:   tt.fields.BlockChannel,
				RoundChannel:   tt.fields.RoundChannel,
				BlockCache:     tt.fields.BlockCache,
				BlockTxnCache:  tt.fields.BlockTxnCache,
				SharderStats:   tt.fields.SharderStats,
				BlockSyncStats: tt.fields.BlockSyncStats,
				TieringStats:   tt.fields.TieringStats,
			}
			if got := sc.GetRoundChannel(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetRoundChannel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_GetBlockChannel(t *testing.T) {
	t.Parallel()

	sc := sharder.GetSharderChain()

	type fields struct {
		Chain          *chain.Chain
		BlockChannel   chan *block.Block
		RoundChannel   chan *round.Round
		BlockCache     cache.Cache
		BlockTxnCache  cache.Cache
		SharderStats   sharder.Stats
		BlockSyncStats *sharder.SyncStats
		TieringStats   *sharder.MinioStats
	}
	tests := []struct {
		name   string
		fields fields
		want   chan *block.Block
	}{
		{
			name: "Test_Chain_GetBlockChannel_OK",
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
			want: sc.BlockChannel,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sc := &sharder.Chain{
				Chain:          tt.fields.Chain,
				BlockChannel:   tt.fields.BlockChannel,
				RoundChannel:   tt.fields.RoundChannel,
				BlockCache:     tt.fields.BlockCache,
				BlockTxnCache:  tt.fields.BlockTxnCache,
				SharderStats:   tt.fields.SharderStats,
				BlockSyncStats: tt.fields.BlockSyncStats,
				TieringStats:   tt.fields.TieringStats,
			}
			if got := sc.GetBlockChannel(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetBlockChannel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_GetBlockHash(t *testing.T) {
	sc := sharder.GetSharderChain()

	type fields struct {
		Chain          *chain.Chain
		BlockChannel   chan *block.Block
		RoundChannel   chan *round.Round
		BlockCache     cache.Cache
		BlockTxnCache  cache.Cache
		SharderStats   sharder.Stats
		BlockSyncStats *sharder.SyncStats
		TieringStats   *sharder.MinioStats
	}
	type args struct {
		ctx         context.Context
		roundNumber int64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "Test_Chain_GetBlockHash_ERR",
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
				ctx:         common.GetRootContext(),
				roundNumber: -1,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := &sharder.Chain{
				Chain:          tt.fields.Chain,
				BlockChannel:   tt.fields.BlockChannel,
				RoundChannel:   tt.fields.RoundChannel,
				BlockCache:     tt.fields.BlockCache,
				BlockTxnCache:  tt.fields.BlockTxnCache,
				SharderStats:   tt.fields.SharderStats,
				BlockSyncStats: tt.fields.BlockSyncStats,
				TieringStats:   tt.fields.TieringStats,
			}
			got, err := sc.GetBlockHash(tt.args.ctx, tt.args.roundNumber)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetBlockHash() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetBlockHash() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChain_SetupGenesisBlock(t *testing.T) {
	sc := sharder.GetSharderChain()

	pbK, _, err := encryption.GenerateKeys()
	if err != nil {
		t.Fatal(err)
	}
	mb, err := makeTestMagicBlock(pbK)
	if err != nil {
		t.Fatal(err)
	}

	b := &block.Block{
		HashIDField: datastore.HashIDField{
			Hash: encryption.Hash("data"),
		},
		MagicBlock: mb,
	}

	_, gb := sc.GenerateGenesisBlock(b.Hash, b.MagicBlock)

	type fields struct {
		Chain          *chain.Chain
		BlockChannel   chan *block.Block
		RoundChannel   chan *round.Round
		BlockCache     cache.Cache
		BlockTxnCache  cache.Cache
		SharderStats   sharder.Stats
		BlockSyncStats *sharder.SyncStats
		TieringStats   *sharder.MinioStats
	}
	type args struct {
		hash       string
		magicBlock *block.MagicBlock
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *block.Block
	}{
		{
			name: "Test_Chain_SetupGenesisBlock_OK",
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
			args: args{hash: b.Hash, magicBlock: b.MagicBlock},
			want: gb,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := &sharder.Chain{
				Chain:          tt.fields.Chain,
				BlockChannel:   tt.fields.BlockChannel,
				RoundChannel:   tt.fields.RoundChannel,
				BlockCache:     tt.fields.BlockCache,
				BlockTxnCache:  tt.fields.BlockTxnCache,
				SharderStats:   tt.fields.SharderStats,
				BlockSyncStats: tt.fields.BlockSyncStats,
				TieringStats:   tt.fields.TieringStats,
			}
			if got := sc.SetupGenesisBlock(tt.args.hash, tt.args.magicBlock); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SetupGenesisBlock() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestChain_LoadLatestBlocksFromStore(t *testing.T) {
	var (
		sc = sharder.GetSharderChain()

		remd = datastore.GetEntityMetadata("round")
		con  = ememorystore.GetEntityCon(ememorystore.WithEntityConnection(common.GetRootContext(), remd), remd)

		r    = round.NewRound(2)
		b    = block.NewBlock("", r.Number)
		hash = encryption.Hash(fmt.Sprintf("data: %v", r.Number))
	)

	sc.LatestFinalizedBlock = block.NewBlock("", 1)

	pbK, prK, err := encryption.GenerateKeys()
	if err != nil {
		t.Fatal(err)
	}
	mb, err := makeTestMagicBlock(pbK)
	if err != nil {
		t.Fatal(err)
	}
	if err := sc.MagicBlockStorage.Put(mb, 2); err != nil {
		t.Fatal(err)
	}

	sign, err := encryption.Sign(prK, hash)
	if err != nil {
		t.Fatal(err)
	}
	b.VerificationTickets = []*block.VerificationTicket{
		&block.VerificationTicket{
			VerifierID: encryption.Hash(pbK),
			Signature:  sign,
		},
	}

	b.MagicBlock = mb
	b.Hash = hash
	b.LatestFinalizedMagicBlockHash = hash
	if err := blockstore.GetStore().Write(b); err != nil {
		t.Fatal(err)
	}

	r.BlockHash = b.Hash

	key := make([]byte, 8)
	binary.BigEndian.PutUint64(key, 2)
	if err := con.Conn.Put(key, datastore.ToJSON(r).Bytes()); err != nil {
		t.Fatal(err)
	}
	if err := con.Commit(); err != nil {
		t.Fatal(err)
	}

	type fields struct {
		Chain          *chain.Chain
		BlockChannel   chan *block.Block
		RoundChannel   chan *round.Round
		BlockCache     cache.Cache
		BlockTxnCache  cache.Cache
		SharderStats   sharder.Stats
		BlockSyncStats *sharder.SyncStats
		TieringStats   *sharder.MinioStats
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
			name: "Test_Chain_LoadLatestBlocksFromStore_OK",
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
			args:    args{ctx: common.GetRootContext()},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := &sharder.Chain{
				Chain:          tt.fields.Chain,
				BlockChannel:   tt.fields.BlockChannel,
				RoundChannel:   tt.fields.RoundChannel,
				BlockCache:     tt.fields.BlockCache,
				BlockTxnCache:  tt.fields.BlockTxnCache,
				SharderStats:   tt.fields.SharderStats,
				BlockSyncStats: tt.fields.BlockSyncStats,
				TieringStats:   tt.fields.TieringStats,
			}
			if err := sc.LoadLatestBlocksFromStore(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("LoadLatestBlocksFromStore() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestChain_SaveMagicBlockHandler(t *testing.T) {
	sc := sharder.GetSharderChain()
	b := block.NewBlock("", 1)

	pbK, _, err := encryption.GenerateKeys()
	if err != nil {
		t.Fatal(err)
	}
	mb, err := makeTestMagicBlock(pbK)
	if err != nil {
		t.Fatal(err)
	}
	b.MagicBlock = mb

	type fields struct {
		Chain          *chain.Chain
		BlockChannel   chan *block.Block
		RoundChannel   chan *round.Round
		BlockCache     cache.Cache
		BlockTxnCache  cache.Cache
		SharderStats   sharder.Stats
		BlockSyncStats *sharder.SyncStats
		TieringStats   *sharder.MinioStats
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
			name: "Test_Chain_SaveMagicBlockHandler_OK",
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
			args:    args{ctx: common.GetRootContext(), b: b},
			wantErr: false,
		},
		{
			name: "Test_Chain_SaveMagicBlockHandler_ERR",
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
			args: func() args {
				mb := *b.MagicBlock
				mb.MagicBlockNumber = -1
				b := *b
				b.Hash = encryption.Hash("data")[:62]
				b.MagicBlock = &mb

				return args{ctx: common.GetRootContext(), b: &b}
			}(),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := &sharder.Chain{
				Chain:          tt.fields.Chain,
				BlockChannel:   tt.fields.BlockChannel,
				RoundChannel:   tt.fields.RoundChannel,
				BlockCache:     tt.fields.BlockCache,
				BlockTxnCache:  tt.fields.BlockTxnCache,
				SharderStats:   tt.fields.SharderStats,
				BlockSyncStats: tt.fields.BlockSyncStats,
				TieringStats:   tt.fields.TieringStats,
			}
			if err := sc.SaveMagicBlockHandler(tt.args.ctx, tt.args.b); (err != nil) != tt.wantErr {
				t.Errorf("SaveMagicBlockHandler() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestChain_SaveMagicBlock(t *testing.T) {
	t.Parallel()

	sc := sharder.GetSharderChain()

	type fields struct {
		Chain          *chain.Chain
		BlockChannel   chan *block.Block
		RoundChannel   chan *round.Round
		BlockCache     cache.Cache
		BlockTxnCache  cache.Cache
		SharderStats   sharder.Stats
		BlockSyncStats *sharder.SyncStats
		TieringStats   *sharder.MinioStats
	}
	tests := []struct {
		name   string
		fields fields
		want   chain.MagicBlockSaveFunc
	}{
		{
			name: "Test_Chain_SaveMagicBlock_OK",
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
			want: chain.MagicBlockSaveFunc(sc.SaveMagicBlockHandler),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sc := &sharder.Chain{
				Chain:          tt.fields.Chain,
				BlockChannel:   tt.fields.BlockChannel,
				RoundChannel:   tt.fields.RoundChannel,
				BlockCache:     tt.fields.BlockCache,
				BlockTxnCache:  tt.fields.BlockTxnCache,
				SharderStats:   tt.fields.SharderStats,
				BlockSyncStats: tt.fields.BlockSyncStats,
				TieringStats:   tt.fields.TieringStats,
			}
			if got := sc.SaveMagicBlock(); reflect.ValueOf(got).Pointer() != reflect.ValueOf(tt.want).Pointer() {
				t.Errorf("SaveMagicBlock() = %v, want %v", got, tt.want)
			}
		})
	}
}
