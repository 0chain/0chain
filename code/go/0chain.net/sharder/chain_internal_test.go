package sharder

import (
	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/core/cache"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/ememorystore"
	"0chain.net/core/encryption"
	"0chain.net/sharder/blockstore"
	"context"
	"encoding/binary"
	"github.com/0chain/gorocksdb"
	"reflect"
	"testing"
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

func TestChain_setupLatestBlocks(t *testing.T) {
	sc := GetSharderChain()

	// case 1
	var (
		r1    = round.NewRound(1)
		lfmb1 = block.NewBlock("", r1.Number)
	)

	pbK, _, err := encryption.GenerateKeys()
	if err != nil {
		t.Fatal(err)
	}
	mb1, err := makeTestMagicBlock(pbK)
	if err != nil {
		t.Fatal(err)
	}
	lfmb1.MagicBlock = mb1
	bl1 := &blocksLoaded{
		lfb:   block.NewBlock("", r1.Number),
		lfmb:  lfmb1,
		r:     r1,
		nlfmb: block.NewBlock("", r1.Number),
	}

	// case 2
	var (
		r2     = round.NewRound(1)
		hash   = encryption.Hash("data")
		lfb2   = block.NewBlock("", r2.Number)
		lfmb2  = block.NewBlock("", r2.Number)
		nlfmb2 = block.NewBlock("", 3)
	)
	pbK, prK, err := encryption.GenerateKeys()
	if err != nil {
		t.Fatal(err)
	}
	sign, err := encryption.Sign(prK, hash)
	if err != nil {
		t.Fatal(err)
	}
	lfb2.VerificationTickets = []*block.VerificationTicket{
		&block.VerificationTicket{
			VerifierID: encryption.Hash(pbK),
			Signature:  sign,
		},
	}
	mb2, err := makeTestMagicBlock(pbK)
	if err != nil {
		t.Fatal(err)
	}
	lfmb2.MagicBlock = mb2
	lfb2.Hash = hash
	nlfmb2.MagicBlock = mb1
	bl2 := &blocksLoaded{
		lfb:   lfb2,
		lfmb:  lfmb2,
		r:     r2,
		nlfmb: nlfmb2,
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
		bl  *blocksLoaded
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Test_Chain_setupLatestBlocks_No_Verification_OK",
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
				bl:  bl1,
			},
			wantErr: false,
		},
		{
			name: "Test_Chain_setupLatestBlocks_OK",
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
				bl:  bl2,
			},
			wantErr: false,
		},
		{
			name: "Test_Chain_setupLatestBlocks_Update_MB_ERR",
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
				mb := *mb1
				mb.Miners = nil
				lfmb := *lfmb1
				lfmb.MagicBlock = &mb

				return args{
					ctx: common.GetRootContext(),
					bl: &blocksLoaded{
						lfb:   bl1.lfb,
						lfmb:  &lfmb,
						r:     bl1.r,
						nlfmb: bl1.nlfmb,
					},
				}
			}(),
			wantErr: true,
		},
		{
			name: "Test_Chain_setupLatestBlocks_Update_MB_ERR2",
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
				mb := *mb2
				mb.Miners = nil
				nlfmb := *nlfmb2
				nlfmb.MagicBlock = &mb

				return args{
					ctx: common.GetRootContext(),
					bl: &blocksLoaded{
						lfb:   bl2.lfb,
						lfmb:  bl2.lfmb,
						r:     bl2.r,
						nlfmb: &nlfmb,
					},
				}
			}(),
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
			if err := sc.setupLatestBlocks(tt.args.ctx, tt.args.bl); (err != nil) != tt.wantErr {
				t.Errorf("setupLatestBlocks() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestChain_loadLatestFinalizedMagicBlockFromStore(t *testing.T) {
	sc := GetSharderChain()

	// case 1
	var (
		r1    = round.NewRound(1)
		lfb1  = block.NewBlock("", r1.Number)
		lfmb1 = block.NewBlock("", 1)
	)
	pbK, _, err := encryption.GenerateKeys()
	if err != nil {
		t.Fatal(err)
	}
	mb1, err := makeTestMagicBlock(pbK)
	if err != nil {
		t.Fatal(err)
	}
	lfb1.MagicBlock = mb1
	lfmb1.Hash = encryption.Hash("LatestFinalizedMagicBlockHash data")
	lfmb1.Round = r1.Number
	lfmb1.MagicBlock = mb1
	lfb1.LatestFinalizedMagicBlockHash = lfmb1.Hash
	lfb1.LatestFinalizedMagicBlockRound = lfmb1.Round
	if err := blockstore.GetStore().Write(lfmb1); err != nil {
		t.Fatal(err)
	}

	// case 2
	lfb2 := *lfb1
	lfb2.Hash = encryption.Hash("lfb2 data")
	lfb2.LatestFinalizedMagicBlockHash = lfb2.Hash

	// case 3
	var (
		r3    = round.NewRound(3)
		lfb3  = block.NewBlock("", r3.Number)
		lfmb3 = block.NewBlock("", r3.Number)
	)
	pbK, _, err = encryption.GenerateKeys()
	if err != nil {
		t.Fatal(err)
	}
	mb3, err := makeTestMagicBlock(pbK)
	if err != nil {
		t.Fatal(err)
	}
	lfb3.MagicBlock = mb3
	lfmb3.Hash = encryption.Hash("LatestFinalizedMagicBlockHash3 data")
	lfmb3.Round = r3.Number
	lfmb3.MagicBlock = mb3
	lfb3.LatestFinalizedMagicBlockHash = lfmb3.Hash
	lfb3.LatestFinalizedMagicBlockRound = lfmb3.Round

	// case 4
	var (
		r4    = round.NewRound(4)
		lfb4  = block.NewBlock("", r4.Number)
		lfmb4 = block.NewBlock("", r4.Number)
	)
	pbK, _, err = encryption.GenerateKeys()
	if err != nil {
		t.Fatal(err)
	}
	mb4, err := makeTestMagicBlock(pbK)
	if err != nil {
		t.Fatal(err)
	}
	lfb4.MagicBlock = mb4
	lfmb4.Hash = encryption.Hash("LatestFinalizedMagicBlockHash4 data")
	lfmb4.Round = r4.Number
	lfmb4.MagicBlock = nil
	lfb4.LatestFinalizedMagicBlockHash = lfmb4.Hash
	lfb4.LatestFinalizedMagicBlockRound = lfmb4.Round

	if err := blockstore.GetStore().Write(lfmb4); err != nil {
		t.Fatal(err)
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
		lfb *block.Block
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		wantLfmb *block.Block
		wantErr  bool
	}{
		{
			name: "Test_Chain_loadLatestFinalizedMagicBlockFromStore_OK",
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
				lfb: lfb1,
			},
			wantLfmb: lfmb1,
			wantErr:  false,
		},
		{
			name: "Test_Chain_loadLatestFinalizedMagicBlockFromStore_Empty_LFMB_Hash_ERR",
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
				lfb: func() *block.Block {
					lfb := *lfb1
					lfb.LatestFinalizedMagicBlockHash = ""

					return &lfb
				}(),
			},
			wantErr: true,
		},
		{
			name: "Test_Chain_loadLatestFinalizedMagicBlockFromStore_LFB_Hash_Equals_LFMB_Hash_And_Nil_MB_ERR",
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
				lfb: func() *block.Block {
					lfb := *lfb1
					lfb.Hash = encryption.Hash("lfb data")
					lfb.LatestFinalizedMagicBlockHash = lfb.Hash
					lfb.MagicBlock = nil

					return &lfb
				}(),
			},
			wantErr: true,
		},
		{
			name: "Test_Chain_loadLatestFinalizedMagicBlockFromStore_LFB_Hash_Equals_LFMB_Hash_OK",
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
				lfb: &lfb2,
			},
			wantLfmb: &lfb2,
			wantErr:  false,
		},
		{
			name: "Test_Chain_loadLatestFinalizedMagicBlockFromStore_LFMB_Not_Existing_In_Store_ERR",
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
				lfb: lfb3,
			},
			wantErr: true,
		},
		{
			name: "Test_Chain_loadLatestFinalizedMagicBlockFromStore_LFMB_From_Store_Nil_MB_ERR",
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
				lfb: lfb4,
			},
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
			gotLfmb, err := sc.loadLatestFinalizedMagicBlockFromStore(tt.args.ctx, tt.args.lfb)
			if (err != nil) != tt.wantErr {
				t.Errorf("loadLatestFinalizedMagicBlockFromStore() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotLfmb, tt.wantLfmb) {
				t.Errorf("loadLatestFinalizedMagicBlockFromStore() gotLfmb = %v, want %v", gotLfmb, tt.wantLfmb)
			}
		})
	}
}

func TestChain_walkDownLookingForLFB(t *testing.T) {
	sc := GetSharderChain()

	var (
		remd = datastore.GetEntityMetadata("round")
		rctx = ememorystore.WithEntityConnection(common.GetRootContext(), remd)
		conn = ememorystore.GetEntityCon(rctx, remd)
	)

	key := make([]byte, 8)
	binary.BigEndian.PutUint64(key, 123)
	if err := conn.Conn.Put(key, []byte(`}{`)); err != nil {
		t.Fatal(err)
	}

	r := round.NewRound(124)
	key2 := make([]byte, 8)
	binary.BigEndian.PutUint64(key2, 124)
	if err := conn.Conn.Put(key2, datastore.ToJSON(r).Bytes()); err != nil {
		t.Fatal(err)
	}

	iter := conn.Conn.NewIterator(conn.ReadOptions)
	iter.SeekToLast()

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
		iter *gorocksdb.Iterator
		r    *round.Round
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantLfb *block.Block
		wantErr bool
	}{
		{
			name: "Test_Chain_walkDownLookingForLFB_JSON_ERR",
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
				iter: iter,
				r:    &round.Round{},
			},
			wantErr: true,
		},
		{
			name: "Test_Chain_walkDownLookingForLFB_Invalid_Iter_ERR",
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
				iter: func() *gorocksdb.Iterator {
					emd := datastore.GetEntityMetadata("round")
					conn := ememorystore.GetEntityCon(ememorystore.WithEntityConnection(common.GetRootContext(), emd), remd)

					return conn.Conn.NewIterator(conn.ReadOptions)
				}(),
				r: &round.Round{},
			},
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
			gotLfb, err := sc.walkDownLookingForLFB(tt.args.iter, tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("walkDownLookingForLFB() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotLfb, tt.wantLfb) {
				t.Errorf("walkDownLookingForLFB() gotLfb = %v, want %v", gotLfb, tt.wantLfb)
			}
		})
	}

	if err := conn.Conn.Rollback(); err != nil {
		t.Error(err)
	}
}

func TestChain_iterateRoundsLookingForLFB_WalkDownErr(t *testing.T) {
	sc := GetSharderChain()

	var (
		remd = datastore.GetEntityMetadata("round")
		rctx = ememorystore.WithEntityConnection(common.GetRootContext(), remd)
		conn = ememorystore.GetEntityCon(rctx, remd)
	)

	key := make([]byte, 8)
	binary.BigEndian.PutUint64(key, 123)
	if err := conn.Conn.Put(key, []byte(`}{`)); err != nil {
		t.Fatal(err)
	}

	if err := conn.Commit(); err != nil {
		t.Fatal(err)
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
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		wantBl *blocksLoaded
	}{
		{
			name: "Test_Chain_iterateRoundsLookingForLFB_WalkDownErr_OK",
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
			args: args{ctx: common.GetRootContext()},
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
			if gotBl := sc.iterateRoundsLookingForLFB(tt.args.ctx); !reflect.DeepEqual(gotBl, tt.wantBl) {
				t.Errorf("iterateRoundsLookingForLFB() = %v, want %v", gotBl, tt.wantBl)
			}
		})
	}

	remd = datastore.GetEntityMetadata("round")
	rctx = ememorystore.WithEntityConnection(common.GetRootContext(), remd)
	conn = ememorystore.GetEntityCon(rctx, remd)

	if err := conn.Conn.Delete(key); err != nil {
		t.Error(err)
	}
	if err := conn.Commit(); err != nil {
		t.Error(err)
	}
}

func TestChain_iterateRoundsLookingForLFB_StoreErr(t *testing.T) {
	sc := GetSharderChain()

	var (
		remd = datastore.GetEntityMetadata("round")
		rctx = ememorystore.WithEntityConnection(common.GetRootContext(), remd)
		conn = ememorystore.GetEntityCon(rctx, remd)
		b    = block.NewBlock("", 124)
	)

	b.Hash = encryption.Hash("data")
	b.Round = 124
	r := round.NewRound(124)
	r.BlockHash = b.Hash
	key := make([]byte, 8)
	binary.BigEndian.PutUint64(key, 124)

	if err := conn.Conn.Put(key, datastore.ToJSON(r).Bytes()); err != nil {
		t.Fatal(err)
	}
	if err := conn.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := blockstore.GetStore().Write(b); err != nil {
		t.Fatal(err)
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
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		wantBl *blocksLoaded
	}{
		{
			name: "Test_Chain_iterateRoundsLookingForLFB_StoreErr_OK",
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
			args: args{ctx: common.GetRootContext()},
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
			if gotBl := sc.iterateRoundsLookingForLFB(tt.args.ctx); !reflect.DeepEqual(gotBl, tt.wantBl) {
				t.Errorf("iterateRoundsLookingForLFB() = %v, want %v", gotBl, tt.wantBl)
			}
		})
	}

	remd = datastore.GetEntityMetadata("round")
	rctx = ememorystore.WithEntityConnection(common.GetRootContext(), remd)
	conn = ememorystore.GetEntityCon(rctx, remd)

	if err := conn.Conn.Delete(key); err != nil {
		t.Error(err)
	}
	if err := conn.Commit(); err != nil {
		t.Error(err)
	}
}
