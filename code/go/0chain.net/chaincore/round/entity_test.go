package round

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/0chain/common/core/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/node"
	"0chain.net/core/datastore"
	"0chain.net/core/memorystore"
	"0chain.net/core/mocks"
	"github.com/0chain/common/core/logging"
)

var (
	//blsPublicKeysNum = 10
	blsPublicKeys = []string{
		"046a69c7525694e67f5039b2004110b09b362e83cb232379a071a8234a14f41f4118f5e4e0b33c4debb6ac0b626010b501f2463d21b3850fcd5a8bbbe3270221",
		"b15102bec92dababc953437ac46e90f8ef0bb99e4e78613aa5bf01ddc12e6b04b911fd752d35b7fb03546fa5883024d56f1fdc0ee1a836e137da79032d01e408",
		"6280bbf63ab8ad84c9ef72d56705a7a0d3207102ec58fcc6972098be87da342074851b07774924a29c1df253fc7e27571b09c1779e0b461a4216b5e8052fe086",
		"2d065b09841817b00b502ba7df7a0f26fe1c0aae0a1f56f3cec7eea51967c1130bf9cdd935f9a4b72b596c1b6ecfdc4d17d725c9ddd99b76fe06f063dc9a3e88",
		"8650897e1fb58e91d92dff86d1d0bf0833c2960b39f3ca340ed612ff16e30e03bb75fc5e1b9dfbca8a33f8aefed18121366fcf0eb22c7c955045a990d303fd1d",
		"53207759a66f139ad4b15202ca60d5d694a2a122ee0519cf57744e08b6ba940024459ded6d51b81ab2a645ea386bcf11bdabb2b197083287a4c0e5a5cf448415",
		"625fc0c291ff10e1fe647803f4e1a463b010e7b0cdd17d064f2f085760d7c910e923224022d063c3b04d74028cb0f758c5595c15eb08bcd012aa0a2feb8e7493",
		"9603a712393d9b9d6d4291874f9474dbc057035e7f385280c25b7257926b8118711fcc8cf439c7481faadcb8790be851bd12d01882eb18644df925b302444a90",
		"3eb2b0d62136e30c5e3bdb56b2d5e5015554b56ea903fdbae527b613b5338b00a69741dda6fd692011a5f244de27dbb4c9000cb220a9dca36ea333c271875183",
		"35f73f4ef2b79857200ea98a8efe472944cdb758753f1642ecbcc3a5e984c3090aee9d3347c0e8db987674f3b1f4cd9adb888ebc0a241a01891842eb9531f295",
	}
)

func init() {
	logging.InitLogging("development", "")

	sp := memorystore.GetStorageProvider()
	SetupEntity(sp)
	block.SetupEntity(sp)

	setupRoundDBMocks()

	//blsPublicKeys = make([]string, blsPublicKeysNum)
	//for i := 0; i < blsPublicKeysNum; i++ {
	//	ss := encryption.NewBLS0ChainScheme()
	//	ss.GenerateKeys()
	//	blsPublicKeys[i] = ss.GetPublicKey()
	//	fmt.Printf("%q,\n", blsPublicKeys[i])
	//}
}

func setupRoundDBMocks() {
	store := mocks.Store{}
	store.On("Read", context.Context(nil), "", new(Round)).Return(
		func(_ context.Context, _ string, _ datastore.Entity) error {
			return nil
		},
	)
	store.On("Write", context.Context(nil), new(Round)).Return(
		func(_ context.Context, _ datastore.Entity) error {
			return nil
		},
	)
	store.On("Delete", context.Context(nil), new(Round)).Return(
		func(_ context.Context, _ datastore.Entity) error {
			return nil
		},
	)

	roundEntityMetadata.Store = &store
}

func makeTestNode(pbK string) (*node.Node, error) {
	nc := map[interface{}]interface{}{
		"type":       node.NodeTypeSharder,
		"public_ip":  "public ip",
		"n2n_ip":     "n2n_ip",
		"port":       8080,
		"id":         util.ToHex([]byte("miners node id")),
		"public_key": pbK,
	}
	n, err := node.NewNode(nc)
	if err != nil {
		return nil, err
	}

	return n, nil
}

func TestNewRound(t *testing.T) {
	t.Parallel()

	r := datastore.GetEntityMetadata("round").Instance().(*Round)
	r.Number = 2

	type args struct {
		round int64
	}
	tests := []struct {
		name string
		args args
		want *Round
	}{
		{
			name: "OK",
			args: args{round: r.Number},
			want: r,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := NewRound(tt.args.round); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewRound() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRound_GetKey(t *testing.T) {
	t.Parallel()

	rNum := int64(2)

	type fields struct {
		NOIDField        datastore.NOIDField
		Number           int64
		RandomSeed       int64
		hasRandomSeed    uint32
		Block            *block.Block
		BlockHash        string
		VRFOutput        string
		minerPerm        []int
		phase            Phase
		proposedBlocks   []*block.Block
		notarizedBlocks  []*block.Block
		shares           map[string]*VRFShare
		softTimeoutCount int32
		vrfStartTime     atomic.Value
	}
	tests := []struct {
		name   string
		fields fields
		want   datastore.Key
	}{
		{
			name:   "OK",
			fields: fields{Number: rNum},
			want:   datastore.ToKey(fmt.Sprintf("%v", rNum)),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := &Round{
				NOIDField:        tt.fields.NOIDField,
				Number:           tt.fields.Number,
				RandomSeed:       tt.fields.RandomSeed,
				Block:            tt.fields.Block,
				BlockHash:        tt.fields.BlockHash,
				VRFOutput:        tt.fields.VRFOutput,
				minerPerm:        tt.fields.minerPerm,
				phase:            tt.fields.phase,
				proposedBlocks:   tt.fields.proposedBlocks,
				notarizedBlocks:  tt.fields.notarizedBlocks,
				mutex:            sync.RWMutex{},
				shares:           tt.fields.shares,
				softTimeoutCount: tt.fields.softTimeoutCount,
				vrfStartTime:     tt.fields.vrfStartTime,
				timeoutCounter:   timeoutCounter{},
			}
			if got := r.GetKey(); got != tt.want {
				t.Errorf("GetKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRound_SetRandomSeedForNotarizedBlock(t *testing.T) {
	t.Parallel()

	var (
		r    = NewRound(2)
		seed = int64(4)
	)
	atomic.StoreInt64(&r.RandomSeed, seed)

	type fields struct {
		NOIDField        datastore.NOIDField
		Number           int64
		RandomSeed       int64
		hasRandomSeed    uint32
		Block            *block.Block
		BlockHash        string
		VRFOutput        string
		minerPerm        []int
		phase            Phase
		proposedBlocks   []*block.Block
		notarizedBlocks  []*block.Block
		shares           map[string]*VRFShare
		softTimeoutCount int32
		vrfStartTime     atomic.Value
	}
	type args struct {
		seed      int64
		minersNum int
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *Round
	}{
		{
			name: "OK",
			fields: func() fields {
				r := NewRound(r.Number)
				return fields{
					NOIDField:        r.NOIDField,
					Number:           r.Number,
					RandomSeed:       r.RandomSeed,
					Block:            r.Block,
					BlockHash:        r.BlockHash,
					VRFOutput:        r.VRFOutput,
					minerPerm:        r.minerPerm,
					phase:            r.phase,
					proposedBlocks:   r.proposedBlocks,
					notarizedBlocks:  r.notarizedBlocks,
					shares:           r.shares,
					softTimeoutCount: r.softTimeoutCount,
					vrfStartTime:     r.vrfStartTime,
				}
			}(),
			args: args{seed: seed},
			want: r,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := &Round{
				NOIDField:        tt.fields.NOIDField,
				Number:           tt.fields.Number,
				RandomSeed:       tt.fields.RandomSeed,
				Block:            tt.fields.Block,
				BlockHash:        tt.fields.BlockHash,
				VRFOutput:        tt.fields.VRFOutput,
				minerPerm:        tt.fields.minerPerm,
				phase:            tt.fields.phase,
				proposedBlocks:   tt.fields.proposedBlocks,
				notarizedBlocks:  tt.fields.notarizedBlocks,
				shares:           tt.fields.shares,
				softTimeoutCount: tt.fields.softTimeoutCount,
				vrfStartTime:     tt.fields.vrfStartTime,
			}
			r.timeoutCounter.votes = make(map[string]int)

			r.SetRandomSeedForNotarizedBlock(tt.args.seed, tt.args.minersNum)
			r.minerPerm = nil
			if !assert.Equal(t, r, tt.want) {
				t.Errorf("SetRandomSeedForNotarizedBlock() got = %v, want = %v", r, tt.want)
			}
		})
	}
}

func TestRound_SetRandomSeed(t *testing.T) {
	t.Parallel()

	var (
		r    = NewRound(2)
		seed = int64(4)
	)
	atomic.StoreInt64(&r.RandomSeed, seed)
	r.phase = ShareVRF

	settedSeedR := NewRound(2)

	type fields struct {
		NOIDField        datastore.NOIDField
		Number           int64
		RandomSeed       int64
		hasRandomSeed    uint32
		Block            *block.Block
		BlockHash        string
		VRFOutput        string
		minerPerm        []int
		phase            Phase
		proposedBlocks   []*block.Block
		notarizedBlocks  []*block.Block
		shares           map[string]*VRFShare
		softTimeoutCount int32
		vrfStartTime     atomic.Value
	}
	type args struct {
		seed      int64
		minersNum int
	}
	tests := []struct {
		name         string
		fields       fields
		args         args
		checksGetter bool
		want         *Round
	}{
		{
			name: "Setted_Seed_OK",
			fields: fields{
				NOIDField:        settedSeedR.NOIDField,
				Number:           settedSeedR.Number,
				RandomSeed:       settedSeedR.RandomSeed,
				Block:            settedSeedR.Block,
				BlockHash:        settedSeedR.BlockHash,
				VRFOutput:        settedSeedR.VRFOutput,
				minerPerm:        settedSeedR.minerPerm,
				phase:            settedSeedR.phase,
				proposedBlocks:   settedSeedR.proposedBlocks,
				notarizedBlocks:  settedSeedR.notarizedBlocks,
				shares:           settedSeedR.shares,
				softTimeoutCount: settedSeedR.softTimeoutCount,
				vrfStartTime:     settedSeedR.vrfStartTime,
			},
			want: settedSeedR,
		},
		{
			name: "OK",
			fields: func() fields {
				r := NewRound(r.Number)
				return fields{
					NOIDField:        r.NOIDField,
					Number:           r.Number,
					RandomSeed:       r.RandomSeed,
					Block:            r.Block,
					BlockHash:        r.BlockHash,
					VRFOutput:        r.VRFOutput,
					minerPerm:        r.minerPerm,
					phase:            r.phase,
					proposedBlocks:   r.proposedBlocks,
					notarizedBlocks:  r.notarizedBlocks,
					shares:           r.shares,
					softTimeoutCount: r.softTimeoutCount,
					vrfStartTime:     r.vrfStartTime,
				}
			}(),
			args:         args{seed: seed},
			checksGetter: true,
			want:         r,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := &Round{
				NOIDField:        tt.fields.NOIDField,
				Number:           tt.fields.Number,
				RandomSeed:       tt.fields.RandomSeed,
				Block:            tt.fields.Block,
				BlockHash:        tt.fields.BlockHash,
				VRFOutput:        tt.fields.VRFOutput,
				minerPerm:        tt.fields.minerPerm,
				phase:            tt.fields.phase,
				proposedBlocks:   tt.fields.proposedBlocks,
				notarizedBlocks:  tt.fields.notarizedBlocks,
				shares:           tt.fields.shares,
				softTimeoutCount: tt.fields.softTimeoutCount,
				vrfStartTime:     tt.fields.vrfStartTime,
			}
			r.timeoutCounter.votes = make(map[string]int)

			r.SetRandomSeed(tt.args.seed, tt.args.minersNum)
			r.minerPerm = nil
			if !assert.Equal(t, r, tt.want) {
				t.Errorf("SetRandomSeed() got = %v, want = %v", r, tt.want)
			}

			if tt.checksGetter {
				gotRS := r.GetRandomSeed()
				if gotRS != tt.args.seed {
					t.Errorf("GetrandomSeed() got = %v, want = %v", gotRS, tt.args.seed)
				}
			}
		})
	}
}

func TestRound_GetVRFOutput(t *testing.T) {
	t.Parallel()

	type fields struct {
		NOIDField        datastore.NOIDField
		Number           int64
		RandomSeed       int64
		hasRandomSeed    uint32
		Block            *block.Block
		BlockHash        string
		VRFOutput        string
		minerPerm        []int
		phase            Phase
		proposedBlocks   []*block.Block
		notarizedBlocks  []*block.Block
		shares           map[string]*VRFShare
		softTimeoutCount int32
		vrfStartTime     atomic.Value
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "OK",
			want: "VRF output",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := &Round{
				NOIDField:        tt.fields.NOIDField,
				Number:           tt.fields.Number,
				RandomSeed:       tt.fields.RandomSeed,
				Block:            tt.fields.Block,
				BlockHash:        tt.fields.BlockHash,
				VRFOutput:        tt.fields.VRFOutput,
				minerPerm:        tt.fields.minerPerm,
				phase:            tt.fields.phase,
				proposedBlocks:   tt.fields.proposedBlocks,
				notarizedBlocks:  tt.fields.notarizedBlocks,
				shares:           tt.fields.shares,
				softTimeoutCount: tt.fields.softTimeoutCount,
				vrfStartTime:     tt.fields.vrfStartTime,
			}

			r.SetVRFOutput(tt.want)
			if got := r.GetVRFOutput(); got != tt.want {
				t.Errorf("GetVRFOutput() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRound_AddNotarizedBlock(t *testing.T) {
	t.Parallel()

	b := block.NewBlock("", 1)
	b.HashBlock()
	b.SetRoundRandomSeed(1)

	b2 := block.NewBlock("", 2)
	b2.HashBlock()
	b2.SetBlockNotarized()
	b2.SetRoundRandomSeed(1)

	b3 := block.NewBlock("", 3)
	b3.HashBlock()
	b3.SetBlockNotarized()

	type fields struct {
		NOIDField        datastore.NOIDField
		Number           int64
		RandomSeed       int64
		hasRandomSeed    uint32
		Block            *block.Block
		BlockHash        string
		VRFOutput        string
		minerPerm        []int
		phase            Phase
		proposedBlocks   []*block.Block
		notarizedBlocks  []*block.Block
		shares           map[string]*VRFShare
		softTimeoutCount int32
		vrfStartTime     atomic.Value
	}
	type args struct {
		b *block.Block
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "FALSE",
			fields: fields{
				notarizedBlocks: []*block.Block{
					func() *block.Block {
						// creating new reference for same block
						b := block.NewBlock("", b.Round)
						b.HashBlock()
						b.SetRoundRandomSeed(1)

						return b
					}(),
				},
			},
			args: args{b: b},
		},
		{
			name: "TRUE",
			fields: fields{
				notarizedBlocks: []*block.Block{
					b,
				},
			},
			args: args{b: b2},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := &Round{
				NOIDField:        tt.fields.NOIDField,
				Number:           tt.fields.Number,
				RandomSeed:       tt.fields.RandomSeed,
				Block:            tt.fields.Block,
				BlockHash:        tt.fields.BlockHash,
				VRFOutput:        tt.fields.VRFOutput,
				minerPerm:        tt.fields.minerPerm,
				phase:            tt.fields.phase,
				proposedBlocks:   tt.fields.proposedBlocks,
				notarizedBlocks:  tt.fields.notarizedBlocks,
				shares:           tt.fields.shares,
				softTimeoutCount: tt.fields.softTimeoutCount,
				vrfStartTime:     tt.fields.vrfStartTime,
			}
			r.AddNotarizedBlock(tt.args.b)
			var foundProposed bool
			for _, b := range r.proposedBlocks {
				if b.Hash == tt.args.b.Hash {
					require.Equal(t, tt.args.b, b)
					foundProposed = true
					break
				}
			}

			require.True(t, foundProposed)

			var foundNotarized bool
			for _, b := range r.notarizedBlocks {
				if b.Hash == tt.args.b.Hash {
					require.Equal(t, tt.args.b, b)
					foundNotarized = true
					break
				}
			}
			require.True(t, foundNotarized)
		})
	}
}

func TestRound_GetNotarizedBlocks(t *testing.T) {
	t.Parallel()

	notBlocks := []*block.Block{
		block.NewBlock("", 1),
	}

	type fields struct {
		NOIDField        datastore.NOIDField
		Number           int64
		RandomSeed       int64
		hasRandomSeed    uint32
		Block            *block.Block
		BlockHash        string
		VRFOutput        string
		minerPerm        []int
		phase            Phase
		proposedBlocks   []*block.Block
		notarizedBlocks  []*block.Block
		shares           map[string]*VRFShare
		softTimeoutCount int32
		vrfStartTime     atomic.Value
	}
	tests := []struct {
		name   string
		fields fields
		want   []*block.Block
	}{
		{
			name:   "OK",
			fields: fields{notarizedBlocks: notBlocks},
			want:   notBlocks,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := &Round{
				NOIDField:        tt.fields.NOIDField,
				Number:           tt.fields.Number,
				RandomSeed:       tt.fields.RandomSeed,
				Block:            tt.fields.Block,
				BlockHash:        tt.fields.BlockHash,
				VRFOutput:        tt.fields.VRFOutput,
				minerPerm:        tt.fields.minerPerm,
				phase:            tt.fields.phase,
				proposedBlocks:   tt.fields.proposedBlocks,
				notarizedBlocks:  tt.fields.notarizedBlocks,
				mutex:            sync.RWMutex{},
				shares:           tt.fields.shares,
				softTimeoutCount: tt.fields.softTimeoutCount,
				vrfStartTime:     tt.fields.vrfStartTime,
				timeoutCounter:   timeoutCounter{},
			}
			if got := r.GetNotarizedBlocks(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetNotarizedBlocks() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRound_AddProposedBlock(t *testing.T) {
	t.Parallel()

	b := block.NewBlock("", 2)
	b.HashBlock()

	b2 := block.NewBlock("", 3)
	b2.HashBlock()

	type fields struct {
		NOIDField        datastore.NOIDField
		Number           int64
		RandomSeed       int64
		hasRandomSeed    uint32
		Block            *block.Block
		BlockHash        string
		VRFOutput        string
		minerPerm        []int
		phase            Phase
		proposedBlocks   []*block.Block
		notarizedBlocks  []*block.Block
		shares           map[string]*VRFShare
		softTimeoutCount int32
		vrfStartTime     atomic.Value
	}
	type args struct {
		b *block.Block
	}
	tests := []struct {
		name         string
		fields       fields
		args         args
		want         *block.Block
		wantResult   bool
		wantPrBlocks []*block.Block
	}{
		{
			name: "FALSE",
			fields: fields{
				proposedBlocks: []*block.Block{
					b,
				},
			},
			args:       args{b: b},
			want:       b,
			wantResult: false,
			wantPrBlocks: []*block.Block{
				b,
			},
		},
		{
			name: "TRUE",
			fields: fields{
				proposedBlocks: []*block.Block{
					b2,
				},
			},
			args:       args{b: b},
			want:       b,
			wantResult: true,
			wantPrBlocks: []*block.Block{
				b2,
				b,
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := &Round{
				NOIDField:        tt.fields.NOIDField,
				Number:           tt.fields.Number,
				RandomSeed:       tt.fields.RandomSeed,
				Block:            tt.fields.Block,
				BlockHash:        tt.fields.BlockHash,
				VRFOutput:        tt.fields.VRFOutput,
				minerPerm:        tt.fields.minerPerm,
				phase:            tt.fields.phase,
				proposedBlocks:   tt.fields.proposedBlocks,
				notarizedBlocks:  tt.fields.notarizedBlocks,
				shares:           tt.fields.shares,
				softTimeoutCount: tt.fields.softTimeoutCount,
				vrfStartTime:     tt.fields.vrfStartTime,
			}
			r.AddProposedBlock(tt.args.b)
			gotProposedBlocks := r.GetProposedBlocks()
			require.Equal(t, tt.wantPrBlocks, gotProposedBlocks)
		})
	}
}

func TestRound_GetHeaviestNotarizedBlock(t *testing.T) {
	t.Parallel()

	b := block.NewBlock("", 2)
	b.HashBlock()

	b2 := block.NewBlock("", 3)
	b2.HashBlock()

	type fields struct {
		NOIDField        datastore.NOIDField
		Number           int64
		RandomSeed       int64
		hasRandomSeed    uint32
		Block            *block.Block
		BlockHash        string
		VRFOutput        string
		minerPerm        []int
		phase            Phase
		proposedBlocks   []*block.Block
		notarizedBlocks  []*block.Block
		shares           map[string]*VRFShare
		softTimeoutCount int32
		vrfStartTime     atomic.Value
	}
	tests := []struct {
		name   string
		fields fields
		want   *block.Block
	}{
		{
			name: "Empty_Notarized_Blocks_OK",
			want: nil,
		},
		{
			name: "OK",
			fields: fields{
				notarizedBlocks: []*block.Block{
					b,
					b2,
				},
			},
			want: b,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := &Round{
				NOIDField:        tt.fields.NOIDField,
				Number:           tt.fields.Number,
				RandomSeed:       tt.fields.RandomSeed,
				Block:            tt.fields.Block,
				BlockHash:        tt.fields.BlockHash,
				VRFOutput:        tt.fields.VRFOutput,
				minerPerm:        tt.fields.minerPerm,
				phase:            tt.fields.phase,
				proposedBlocks:   tt.fields.proposedBlocks,
				notarizedBlocks:  tt.fields.notarizedBlocks,
				mutex:            sync.RWMutex{},
				shares:           tt.fields.shares,
				softTimeoutCount: tt.fields.softTimeoutCount,
				vrfStartTime:     tt.fields.vrfStartTime,
				timeoutCounter:   timeoutCounter{},
			}
			if got := r.GetHeaviestNotarizedBlock(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetHeaviestNotarizedBlock() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRound_GetBlocksByRank(t *testing.T) {
	t.Parallel()

	b := block.NewBlock("", 2)
	b.RoundRank = 2
	b.HashBlock()

	b2 := block.NewBlock("", 3)
	b2.RoundRank = 1
	b2.HashBlock()

	type fields struct {
		NOIDField        datastore.NOIDField
		Number           int64
		RandomSeed       int64
		hasRandomSeed    uint32
		Block            *block.Block
		BlockHash        string
		VRFOutput        string
		minerPerm        []int
		phase            Phase
		proposedBlocks   []*block.Block
		notarizedBlocks  []*block.Block
		shares           map[string]*VRFShare
		softTimeoutCount int32
		vrfStartTime     atomic.Value
	}
	type args struct {
		blocks []*block.Block
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []*block.Block
	}{
		{
			name: "OK",
			args: args{
				blocks: []*block.Block{
					b,
					b2,
				},
			},
			want: []*block.Block{
				b2,
				b,
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := &Round{
				NOIDField:        tt.fields.NOIDField,
				Number:           tt.fields.Number,
				RandomSeed:       tt.fields.RandomSeed,
				Block:            tt.fields.Block,
				BlockHash:        tt.fields.BlockHash,
				VRFOutput:        tt.fields.VRFOutput,
				minerPerm:        tt.fields.minerPerm,
				phase:            tt.fields.phase,
				proposedBlocks:   tt.fields.proposedBlocks,
				notarizedBlocks:  tt.fields.notarizedBlocks,
				mutex:            sync.RWMutex{},
				shares:           tt.fields.shares,
				softTimeoutCount: tt.fields.softTimeoutCount,
				vrfStartTime:     tt.fields.vrfStartTime,
				timeoutCounter:   timeoutCounter{},
			}
			if got := r.GetBlocksByRank(tt.args.blocks); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetBlocksByRank() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRound_GetBestRankedNotarizedBlock(t *testing.T) {
	t.Parallel()

	b := block.NewBlock("", 2)
	b.RoundRank = 2
	b.HashBlock()

	b2 := block.NewBlock("", 3)
	b2.RoundRank = 1
	b2.HashBlock()

	type fields struct {
		NOIDField        datastore.NOIDField
		Number           int64
		RandomSeed       int64
		hasRandomSeed    uint32
		Block            *block.Block
		BlockHash        string
		VRFOutput        string
		minerPerm        []int
		phase            Phase
		proposedBlocks   []*block.Block
		notarizedBlocks  []*block.Block
		shares           map[string]*VRFShare
		softTimeoutCount int32
		vrfStartTime     atomic.Value
	}
	tests := []struct {
		name   string
		fields fields
		want   *block.Block
	}{
		{
			name: "Empty_Notarized_Blocks_OK",
			want: nil,
		},
		{
			name: "1_Notarized_Block_OK",
			fields: fields{
				notarizedBlocks: []*block.Block{
					b,
				},
			},
			want: b,
		},
		{
			name: "OK",
			fields: fields{
				notarizedBlocks: []*block.Block{
					b2,
					b,
				},
			},
			want: b2,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := &Round{
				NOIDField:        tt.fields.NOIDField,
				Number:           tt.fields.Number,
				RandomSeed:       tt.fields.RandomSeed,
				Block:            tt.fields.Block,
				BlockHash:        tt.fields.BlockHash,
				VRFOutput:        tt.fields.VRFOutput,
				minerPerm:        tt.fields.minerPerm,
				phase:            tt.fields.phase,
				proposedBlocks:   tt.fields.proposedBlocks,
				notarizedBlocks:  tt.fields.notarizedBlocks,
				mutex:            sync.RWMutex{},
				shares:           tt.fields.shares,
				softTimeoutCount: tt.fields.softTimeoutCount,
				vrfStartTime:     tt.fields.vrfStartTime,
				timeoutCounter:   timeoutCounter{},
			}
			if got := r.GetBestRankedNotarizedBlock(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetBestRankedNotarizedBlock() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRound_Finalize(t *testing.T) {
	t.Parallel()

	r := NewRound(2)
	b := block.NewBlock("", 2)

	type fields struct {
		NOIDField        datastore.NOIDField
		Number           int64
		RandomSeed       int64
		hasRandomSeed    uint32
		Block            *block.Block
		BlockHash        string
		VRFOutput        string
		minerPerm        []int
		state            FinalizingState
		proposedBlocks   []*block.Block
		notarizedBlocks  []*block.Block
		shares           map[string]*VRFShare
		softTimeoutCount int32
		vrfStartTime     atomic.Value
	}
	type args struct {
		b *block.Block
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *Round
	}{
		{
			name: "OK",
			fields: fields{
				NOIDField:        r.NOIDField,
				Number:           r.Number,
				RandomSeed:       r.RandomSeed,
				Block:            r.Block,
				BlockHash:        r.BlockHash,
				VRFOutput:        r.VRFOutput,
				minerPerm:        r.minerPerm,
				state:            r.finalizingState,
				proposedBlocks:   r.proposedBlocks,
				notarizedBlocks:  r.notarizedBlocks,
				shares:           r.shares,
				softTimeoutCount: r.softTimeoutCount,
				vrfStartTime:     r.vrfStartTime,
			},
			args: args{b: b},
			want: func() *Round {
				r := NewRound(r.Number)
				r.setFinalizingPhase(RoundStateFinalized)
				r.Block = b
				r.BlockHash = b.Hash

				return r
			}(),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := &Round{
				NOIDField:        tt.fields.NOIDField,
				Number:           tt.fields.Number,
				RandomSeed:       tt.fields.RandomSeed,
				Block:            tt.fields.Block,
				BlockHash:        tt.fields.BlockHash,
				VRFOutput:        tt.fields.VRFOutput,
				minerPerm:        tt.fields.minerPerm,
				finalizingState:  tt.fields.state,
				proposedBlocks:   tt.fields.proposedBlocks,
				notarizedBlocks:  tt.fields.notarizedBlocks,
				shares:           tt.fields.shares,
				softTimeoutCount: tt.fields.softTimeoutCount,
				vrfStartTime:     tt.fields.vrfStartTime,
			}
			r.timeoutCounter.votes = make(map[string]int)

			if r.Finalize(tt.args.b); !assert.Equal(t, r, tt.want) {
				t.Errorf("Finalize() = %v, want %v", r, tt.want)
			}
		})
	}
}

func TestRound_SetFinalizing(t *testing.T) {
	t.Parallel()

	type fields struct {
		NOIDField        datastore.NOIDField
		Number           int64
		RandomSeed       int64
		hasRandomSeed    uint32
		Block            *block.Block
		BlockHash        string
		VRFOutput        string
		minerPerm        []int
		state            FinalizingState
		proposedBlocks   []*block.Block
		notarizedBlocks  []*block.Block
		shares           map[string]*VRFShare
		softTimeoutCount int32
		vrfStartTime     atomic.Value
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name:   "State_Finalised_FALSE",
			fields: fields{state: RoundStateFinalized},
			want:   false,
		},
		{
			name:   "Round_0_FALSE",
			fields: fields{Number: 0},
			want:   false,
		},
		{
			name:   "Finalising_FALSE",
			fields: fields{state: RoundStateFinalizing},
			want:   false,
		},
		{
			name:   "TRUE",
			fields: fields{Number: 1, state: -1},
			want:   true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := &Round{
				NOIDField:        tt.fields.NOIDField,
				Number:           tt.fields.Number,
				RandomSeed:       tt.fields.RandomSeed,
				Block:            tt.fields.Block,
				BlockHash:        tt.fields.BlockHash,
				VRFOutput:        tt.fields.VRFOutput,
				minerPerm:        tt.fields.minerPerm,
				finalizingState:  tt.fields.state,
				proposedBlocks:   tt.fields.proposedBlocks,
				notarizedBlocks:  tt.fields.notarizedBlocks,
				mutex:            sync.RWMutex{},
				shares:           tt.fields.shares,
				softTimeoutCount: tt.fields.softTimeoutCount,
				vrfStartTime:     tt.fields.vrfStartTime,
				timeoutCounter:   timeoutCounter{},
			}
			if got := r.SetFinalizing(); got != tt.want {
				t.Errorf("SetFinalizing() = %v, want %v", got, tt.want)
			}
			if tt.want && r.finalizingState != RoundStateFinalizing {
				t.Errorf("SetFinalizing() = %v, want %v", r.phase, RoundStateFinalizing)
			}
		})
	}
}

func TestRound_GetVRFShares(t *testing.T) {
	t.Parallel()

	shares := map[string]*VRFShare{
		"1": {},
	}

	type fields struct {
		NOIDField        datastore.NOIDField
		Number           int64
		RandomSeed       int64
		hasRandomSeed    uint32
		Block            *block.Block
		BlockHash        string
		VRFOutput        string
		minerPerm        []int
		phase            Phase
		proposedBlocks   []*block.Block
		notarizedBlocks  []*block.Block
		shares           map[string]*VRFShare
		softTimeoutCount int32
		vrfStartTime     atomic.Value
	}
	tests := []struct {
		name   string
		fields fields
		want   map[string]*VRFShare
	}{
		{
			name:   "OK",
			fields: fields{shares: shares},
			want:   shares,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := &Round{
				NOIDField:        tt.fields.NOIDField,
				Number:           tt.fields.Number,
				RandomSeed:       tt.fields.RandomSeed,
				Block:            tt.fields.Block,
				BlockHash:        tt.fields.BlockHash,
				VRFOutput:        tt.fields.VRFOutput,
				minerPerm:        tt.fields.minerPerm,
				phase:            tt.fields.phase,
				proposedBlocks:   tt.fields.proposedBlocks,
				notarizedBlocks:  tt.fields.notarizedBlocks,
				mutex:            sync.RWMutex{},
				shares:           tt.fields.shares,
				softTimeoutCount: tt.fields.softTimeoutCount,
				vrfStartTime:     tt.fields.vrfStartTime,
				timeoutCounter:   timeoutCounter{},
			}
			if got := r.GetVRFShares(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetVRFShares() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRound_GetState(t *testing.T) {
	t.Parallel()

	type fields struct {
		NOIDField        datastore.NOIDField
		Number           int64
		RandomSeed       int64
		hasRandomSeed    uint32
		Block            *block.Block
		BlockHash        string
		VRFOutput        string
		minerPerm        []int
		phase            Phase
		proposedBlocks   []*block.Block
		notarizedBlocks  []*block.Block
		shares           map[string]*VRFShare
		softTimeoutCount int32
		vrfStartTime     atomic.Value
	}
	tests := []struct {
		name   string
		fields fields
		want   Phase
	}{
		{
			name: "OK",
			want: Notarize,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := &Round{
				NOIDField:        tt.fields.NOIDField,
				Number:           tt.fields.Number,
				RandomSeed:       tt.fields.RandomSeed,
				Block:            tt.fields.Block,
				BlockHash:        tt.fields.BlockHash,
				VRFOutput:        tt.fields.VRFOutput,
				minerPerm:        tt.fields.minerPerm,
				phase:            tt.fields.phase,
				proposedBlocks:   tt.fields.proposedBlocks,
				notarizedBlocks:  tt.fields.notarizedBlocks,
				mutex:            sync.RWMutex{},
				shares:           tt.fields.shares,
				softTimeoutCount: tt.fields.softTimeoutCount,
				vrfStartTime:     tt.fields.vrfStartTime,
				timeoutCounter:   timeoutCounter{},
			}

			r.SetPhase(tt.want)
			if got := r.GetPhase(); got != tt.want {
				t.Errorf("GetPhase() = %v, want %v", got, tt.want)
			}

			r.ResetPhase(tt.want)
			if got := r.GetPhase(); got != tt.want {
				t.Errorf("GetPhase() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRound_IsFinalized(t *testing.T) {
	t.Parallel()

	type fields struct {
		NOIDField        datastore.NOIDField
		Number           int64
		RandomSeed       int64
		hasRandomSeed    uint32
		Block            *block.Block
		BlockHash        string
		VRFOutput        string
		minerPerm        []int
		state            FinalizingState
		proposedBlocks   []*block.Block
		notarizedBlocks  []*block.Block
		shares           map[string]*VRFShare
		softTimeoutCount int32
		vrfStartTime     atomic.Value
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name:   "State_Finalised_TRUE",
			fields: fields{state: RoundStateFinalized},
			want:   true,
		},
		{
			name:   "Round_0_TRUE",
			fields: fields{Number: 0},
			want:   true,
		},
		{
			name:   "TRUE",
			fields: fields{Number: 1, state: -1},
			want:   false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := &Round{
				NOIDField:        tt.fields.NOIDField,
				Number:           tt.fields.Number,
				RandomSeed:       tt.fields.RandomSeed,
				Block:            tt.fields.Block,
				BlockHash:        tt.fields.BlockHash,
				VRFOutput:        tt.fields.VRFOutput,
				minerPerm:        tt.fields.minerPerm,
				finalizingState:  tt.fields.state,
				proposedBlocks:   tt.fields.proposedBlocks,
				notarizedBlocks:  tt.fields.notarizedBlocks,
				mutex:            sync.RWMutex{},
				shares:           tt.fields.shares,
				softTimeoutCount: tt.fields.softTimeoutCount,
				vrfStartTime:     tt.fields.vrfStartTime,
				timeoutCounter:   timeoutCounter{},
			}
			if got := r.IsFinalized(); got != tt.want {
				t.Errorf("IsFinalized() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRound_Read(t *testing.T) {
	t.Parallel()

	type fields struct {
		NOIDField        datastore.NOIDField
		Number           int64
		RandomSeed       int64
		hasRandomSeed    uint32
		Block            *block.Block
		BlockHash        string
		VRFOutput        string
		minerPerm        []int
		phase            Phase
		proposedBlocks   []*block.Block
		notarizedBlocks  []*block.Block
		shares           map[string]*VRFShare
		softTimeoutCount int32
		vrfStartTime     atomic.Value
	}
	type args struct {
		ctx context.Context
		key datastore.Key
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:    "OK",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := &Round{
				NOIDField:        tt.fields.NOIDField,
				Number:           tt.fields.Number,
				RandomSeed:       tt.fields.RandomSeed,
				Block:            tt.fields.Block,
				BlockHash:        tt.fields.BlockHash,
				VRFOutput:        tt.fields.VRFOutput,
				minerPerm:        tt.fields.minerPerm,
				phase:            tt.fields.phase,
				proposedBlocks:   tt.fields.proposedBlocks,
				notarizedBlocks:  tt.fields.notarizedBlocks,
				mutex:            sync.RWMutex{},
				shares:           tt.fields.shares,
				softTimeoutCount: tt.fields.softTimeoutCount,
				vrfStartTime:     tt.fields.vrfStartTime,
				timeoutCounter:   timeoutCounter{},
			}
			if err := r.Read(tt.args.ctx, tt.args.key); (err != nil) != tt.wantErr {
				t.Errorf("Read() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRound_Write(t *testing.T) {
	t.Parallel()

	type fields struct {
		NOIDField        datastore.NOIDField
		Number           int64
		RandomSeed       int64
		hasRandomSeed    uint32
		Block            *block.Block
		BlockHash        string
		VRFOutput        string
		minerPerm        []int
		phase            Phase
		proposedBlocks   []*block.Block
		notarizedBlocks  []*block.Block
		shares           map[string]*VRFShare
		softTimeoutCount int32
		vrfStartTime     atomic.Value
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
			name:    "OK",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := &Round{
				NOIDField:        tt.fields.NOIDField,
				Number:           tt.fields.Number,
				RandomSeed:       tt.fields.RandomSeed,
				Block:            tt.fields.Block,
				BlockHash:        tt.fields.BlockHash,
				VRFOutput:        tt.fields.VRFOutput,
				minerPerm:        tt.fields.minerPerm,
				phase:            tt.fields.phase,
				proposedBlocks:   tt.fields.proposedBlocks,
				notarizedBlocks:  tt.fields.notarizedBlocks,
				mutex:            sync.RWMutex{},
				shares:           tt.fields.shares,
				softTimeoutCount: tt.fields.softTimeoutCount,
				vrfStartTime:     tt.fields.vrfStartTime,
				timeoutCounter:   timeoutCounter{},
			}
			if err := r.Write(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("Write() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRound_Delete(t *testing.T) {
	t.Parallel()

	type fields struct {
		NOIDField        datastore.NOIDField
		Number           int64
		RandomSeed       int64
		hasRandomSeed    uint32
		Block            *block.Block
		BlockHash        string
		VRFOutput        string
		minerPerm        []int
		phase            Phase
		proposedBlocks   []*block.Block
		notarizedBlocks  []*block.Block
		shares           map[string]*VRFShare
		softTimeoutCount int32
		vrfStartTime     atomic.Value
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
			name:    "OK",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := &Round{
				NOIDField:        tt.fields.NOIDField,
				Number:           tt.fields.Number,
				RandomSeed:       tt.fields.RandomSeed,
				Block:            tt.fields.Block,
				BlockHash:        tt.fields.BlockHash,
				VRFOutput:        tt.fields.VRFOutput,
				minerPerm:        tt.fields.minerPerm,
				phase:            tt.fields.phase,
				proposedBlocks:   tt.fields.proposedBlocks,
				notarizedBlocks:  tt.fields.notarizedBlocks,
				mutex:            sync.RWMutex{},
				shares:           tt.fields.shares,
				softTimeoutCount: tt.fields.softTimeoutCount,
				vrfStartTime:     tt.fields.vrfStartTime,
				timeoutCounter:   timeoutCounter{},
			}
			if err := r.Delete(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSetupRoundSummaryDB_Panic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		wantPanic bool
	}{
		{
			name:      "PANIC",
			wantPanic: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			defer func() {
				got := recover()
				if (got != nil) != tt.wantPanic {
					t.Errorf("SetupRoundSummaryDB() want panic  = %v, but got = %v", tt.wantPanic, got)
				}
			}()

			SetupRoundSummaryDB("")
		})
	}
}

func TestRound_IsRanksComputed(t *testing.T) {
	t.Parallel()

	type fields struct {
		NOIDField        datastore.NOIDField
		Number           int64
		RandomSeed       int64
		hasRandomSeed    uint32
		Block            *block.Block
		BlockHash        string
		VRFOutput        string
		minerPerm        []int
		phase            Phase
		proposedBlocks   []*block.Block
		notarizedBlocks  []*block.Block
		shares           map[string]*VRFShare
		softTimeoutCount int32
		vrfStartTime     atomic.Value
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name:   "TRUE",
			fields: fields{minerPerm: make([]int, 1)},
			want:   true,
		},
		{
			name: "FAlSE",
			want: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := &Round{
				NOIDField:        tt.fields.NOIDField,
				Number:           tt.fields.Number,
				RandomSeed:       tt.fields.RandomSeed,
				Block:            tt.fields.Block,
				BlockHash:        tt.fields.BlockHash,
				VRFOutput:        tt.fields.VRFOutput,
				minerPerm:        tt.fields.minerPerm,
				phase:            tt.fields.phase,
				proposedBlocks:   tt.fields.proposedBlocks,
				notarizedBlocks:  tt.fields.notarizedBlocks,
				mutex:            sync.RWMutex{},
				shares:           tt.fields.shares,
				softTimeoutCount: tt.fields.softTimeoutCount,
				vrfStartTime:     tt.fields.vrfStartTime,
				timeoutCounter:   timeoutCounter{},
			}
			if got := r.IsRanksComputed(); got != tt.want {
				t.Errorf("IsRanksComputed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRound_GetMinerRank(t *testing.T) {
	t.Parallel()

	n, err := makeTestNode(blsPublicKeys[0])
	if err != nil {
		t.Fatal(err)
	}
	n.SetIndex = 1

	type fields struct {
		NOIDField        datastore.NOIDField
		Number           int64
		RandomSeed       int64
		hasRandomSeed    uint32
		Block            *block.Block
		BlockHash        string
		VRFOutput        string
		minerPerm        []int
		phase            Phase
		proposedBlocks   []*block.Block
		notarizedBlocks  []*block.Block
		shares           map[string]*VRFShare
		softTimeoutCount int32
		vrfStartTime     atomic.Value
	}
	type args struct {
		miner *node.Node
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		want      int
		wantPanic bool
	}{
		{
			name:      "PANIC",
			wantPanic: true,
		},
		{
			name: "Miner_Set_Index_Greater_Than_Len_Miner_Perm",
			fields: fields{
				minerPerm: []int{},
			},
			args: args{miner: n},
			want: -1,
		},
		{
			name: "OK",
			fields: fields{
				minerPerm: []int{1, 2},
			},
			args: args{miner: n},
			want: 2,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			defer func() {
				got := recover()
				if (got != nil) != tt.wantPanic {
					t.Errorf("GetMinerRank() want panic  = %v, but got = %v", tt.wantPanic, got)
				}
			}()

			r := &Round{
				NOIDField:        tt.fields.NOIDField,
				Number:           tt.fields.Number,
				RandomSeed:       tt.fields.RandomSeed,
				Block:            tt.fields.Block,
				BlockHash:        tt.fields.BlockHash,
				VRFOutput:        tt.fields.VRFOutput,
				minerPerm:        tt.fields.minerPerm,
				phase:            tt.fields.phase,
				proposedBlocks:   tt.fields.proposedBlocks,
				notarizedBlocks:  tt.fields.notarizedBlocks,
				mutex:            sync.RWMutex{},
				shares:           tt.fields.shares,
				softTimeoutCount: tt.fields.softTimeoutCount,
				vrfStartTime:     tt.fields.vrfStartTime,
				timeoutCounter:   timeoutCounter{},
			}
			if got := r.GetMinerRank(tt.args.miner); got != tt.want {
				t.Errorf("GetMinerRank() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRound_GetMinersByRank(t *testing.T) {
	t.Parallel()

	n, err := makeTestNode(blsPublicKeys[0])
	if err != nil {
		t.Fatal(err)
	}
	n.Type = node.NodeTypeMiner

	p := node.NewPool(node.NodeTypeMiner)
	p.AddNode(n)

	n2, err := makeTestNode(blsPublicKeys[1])
	if err != nil {
		t.Fatal(err)
	}
	n2.Type = node.NodeTypeMiner
	p.AddNode(n2)

	type fields struct {
		NOIDField        datastore.NOIDField
		Number           int64
		RandomSeed       int64
		hasRandomSeed    uint32
		Block            *block.Block
		BlockHash        string
		VRFOutput        string
		minerPerm        []int
		phase            Phase
		proposedBlocks   []*block.Block
		notarizedBlocks  []*block.Block
		shares           map[string]*VRFShare
		softTimeoutCount int32
		vrfStartTime     atomic.Value
	}
	type args struct {
		miners []*node.Node
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []*node.Node
	}{
		{
			name: "OK",
			fields: fields{
				minerPerm: []int{0, 2},
			},
			args: args{miners: p.Nodes},
			want: []*node.Node{
				n,
				n2,
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := &Round{
				NOIDField:        tt.fields.NOIDField,
				Number:           tt.fields.Number,
				RandomSeed:       tt.fields.RandomSeed,
				Block:            tt.fields.Block,
				BlockHash:        tt.fields.BlockHash,
				VRFOutput:        tt.fields.VRFOutput,
				minerPerm:        tt.fields.minerPerm,
				phase:            tt.fields.phase,
				proposedBlocks:   tt.fields.proposedBlocks,
				notarizedBlocks:  tt.fields.notarizedBlocks,
				mutex:            sync.RWMutex{},
				shares:           tt.fields.shares,
				softTimeoutCount: tt.fields.softTimeoutCount,
				vrfStartTime:     tt.fields.vrfStartTime,
				timeoutCounter:   timeoutCounter{},
			}
			got := r.GetMinersByRank(tt.args.miners)
			for i, n := range got {
				require.Equal(t, tt.want[i].ID, n.ID,
					fmt.Sprintf("i:%v, set_index:%v, ids:%v",
						i, n.SetIndex, []string{got[0].ID, got[1].ID}))
				require.Equal(t, tt.want[i].PublicKey, n.PublicKey, fmt.Sprintf("i:%v, set_index:%v", i, n.SetIndex))
			}
		})
	}
}

func TestRound_Clear(t *testing.T) {
	t.Parallel()

	type fields struct {
		NOIDField        datastore.NOIDField
		Number           int64
		RandomSeed       int64
		hasRandomSeed    uint32
		Block            *block.Block
		BlockHash        string
		VRFOutput        string
		minerPerm        []int
		phase            Phase
		proposedBlocks   []*block.Block
		notarizedBlocks  []*block.Block
		shares           map[string]*VRFShare
		softTimeoutCount int32
		vrfStartTime     atomic.Value
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "OK", // not implemented
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := &Round{
				NOIDField:        tt.fields.NOIDField,
				Number:           tt.fields.Number,
				RandomSeed:       tt.fields.RandomSeed,
				Block:            tt.fields.Block,
				BlockHash:        tt.fields.BlockHash,
				VRFOutput:        tt.fields.VRFOutput,
				minerPerm:        tt.fields.minerPerm,
				phase:            tt.fields.phase,
				proposedBlocks:   tt.fields.proposedBlocks,
				notarizedBlocks:  tt.fields.notarizedBlocks,
				mutex:            sync.RWMutex{},
				shares:           tt.fields.shares,
				softTimeoutCount: tt.fields.softTimeoutCount,
				vrfStartTime:     tt.fields.vrfStartTime,
				timeoutCounter:   timeoutCounter{},
			}

			r.Clear()
		})
	}
}

func TestRound_Restart(t *testing.T) {
	t.Parallel()

	r := NewRound(2)

	wantR := NewRound(2)
	wantR.initialize()
	wantR.Block = nil
	wantR.resetSoftTimeoutCount()
	wantR.ResetPhase(ShareVRF)

	type fields struct {
		NOIDField        datastore.NOIDField
		Number           int64
		RandomSeed       int64
		hasRandomSeed    uint32
		Block            *block.Block
		BlockHash        string
		VRFOutput        string
		minerPerm        []int
		phase            Phase
		proposedBlocks   []*block.Block
		notarizedBlocks  []*block.Block
		shares           map[string]*VRFShare
		softTimeoutCount int32
		vrfStartTime     atomic.Value
	}
	tests := []struct {
		name   string
		fields fields
		want   *Round
	}{
		{
			name: "OK",
			fields: fields{
				NOIDField:        r.NOIDField,
				Number:           r.Number,
				RandomSeed:       r.RandomSeed,
				Block:            r.Block,
				BlockHash:        r.BlockHash,
				VRFOutput:        r.VRFOutput,
				minerPerm:        r.minerPerm,
				phase:            r.phase,
				proposedBlocks:   r.proposedBlocks,
				notarizedBlocks:  r.notarizedBlocks,
				shares:           r.shares,
				softTimeoutCount: r.softTimeoutCount,
				vrfStartTime:     r.vrfStartTime,
			},
			want: r,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := &Round{
				NOIDField:        tt.fields.NOIDField,
				Number:           tt.fields.Number,
				RandomSeed:       tt.fields.RandomSeed,
				Block:            tt.fields.Block,
				BlockHash:        tt.fields.BlockHash,
				VRFOutput:        tt.fields.VRFOutput,
				minerPerm:        tt.fields.minerPerm,
				phase:            tt.fields.phase,
				proposedBlocks:   tt.fields.proposedBlocks,
				notarizedBlocks:  tt.fields.notarizedBlocks,
				mutex:            sync.RWMutex{},
				shares:           tt.fields.shares,
				softTimeoutCount: tt.fields.softTimeoutCount,
				vrfStartTime:     tt.fields.vrfStartTime,
				timeoutCounter:   timeoutCounter{},
			}
			r.timeoutCounter.votes = make(map[string]int)

			r.Restart()
			if !assert.Equal(t, r, tt.want) {
				t.Errorf("Restart() = %v, want %v", r, tt.want)
			}
		})
	}
}

func TestRound_AddVRFShare(t *testing.T) {
	t.Parallel()

	n, err := makeTestNode(blsPublicKeys[0])
	if err != nil {
		t.Fatal(err)
	}

	share := &VRFShare{
		party: n,
	}

	type fields struct {
		NOIDField        datastore.NOIDField
		Number           int64
		RandomSeed       int64
		hasRandomSeed    uint32
		Block            *block.Block
		BlockHash        string
		VRFOutput        string
		minerPerm        []int
		phase            Phase
		proposedBlocks   []*block.Block
		notarizedBlocks  []*block.Block
		shares           map[string]*VRFShare
		softTimeoutCount int32
		vrfStartTime     atomic.Value
	}
	type args struct {
		share     *VRFShare
		threshold int
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "Len_Of_VRF_Shares_Greater_Than_Threshold_FALSE",
			fields: fields{
				shares: map[string]*VRFShare{
					share.party.GetKey(): share,
				},
			},
			args: args{threshold: 0},
			want: false,
		},
		{
			name: "Known_Share_FALSE",
			fields: fields{
				shares: map[string]*VRFShare{
					share.party.GetKey(): share,
				},
			},
			args: args{share: share, threshold: 2},
			want: false,
		},
		{
			name: "TRUE",
			fields: fields{
				shares: map[string]*VRFShare{
					"key": share,
				},
			},
			args: args{share: share, threshold: 2},
			want: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := &Round{
				NOIDField:        tt.fields.NOIDField,
				Number:           tt.fields.Number,
				RandomSeed:       tt.fields.RandomSeed,
				Block:            tt.fields.Block,
				BlockHash:        tt.fields.BlockHash,
				VRFOutput:        tt.fields.VRFOutput,
				minerPerm:        tt.fields.minerPerm,
				phase:            tt.fields.phase,
				proposedBlocks:   tt.fields.proposedBlocks,
				notarizedBlocks:  tt.fields.notarizedBlocks,
				mutex:            sync.RWMutex{},
				shares:           tt.fields.shares,
				softTimeoutCount: tt.fields.softTimeoutCount,
				vrfStartTime:     tt.fields.vrfStartTime,
				timeoutCounter:   timeoutCounter{},
			}
			if got := r.AddVRFShare(tt.args.share, tt.args.threshold); got != tt.want {
				t.Errorf("AddVRFShare() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRound_HasRandomSeed(t *testing.T) {
	t.Parallel()

	type fields struct {
		NOIDField        datastore.NOIDField
		Number           int64
		RandomSeed       int64
		Block            *block.Block
		BlockHash        string
		VRFOutput        string
		minerPerm        []int
		phase            Phase
		proposedBlocks   []*block.Block
		notarizedBlocks  []*block.Block
		shares           map[string]*VRFShare
		softTimeoutCount int32
		vrfStartTime     atomic.Value
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name:   "TRUE",
			fields: fields{RandomSeed: 1},
			want:   true,
		},
		{
			name:   "FALSE",
			fields: fields{RandomSeed: 0},
			want:   false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := &Round{
				NOIDField:        tt.fields.NOIDField,
				Number:           tt.fields.Number,
				RandomSeed:       tt.fields.RandomSeed,
				Block:            tt.fields.Block,
				BlockHash:        tt.fields.BlockHash,
				VRFOutput:        tt.fields.VRFOutput,
				minerPerm:        tt.fields.minerPerm,
				phase:            tt.fields.phase,
				proposedBlocks:   tt.fields.proposedBlocks,
				notarizedBlocks:  tt.fields.notarizedBlocks,
				mutex:            sync.RWMutex{},
				shares:           tt.fields.shares,
				softTimeoutCount: tt.fields.softTimeoutCount,
				vrfStartTime:     tt.fields.vrfStartTime,
				timeoutCounter:   timeoutCounter{},
			}
			if got := r.HasRandomSeed(); got != tt.want {
				t.Errorf("HasRandomSeed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRound_GetSoftTimeoutCount(t *testing.T) {
	t.Parallel()

	c := 1

	type fields struct {
		NOIDField        datastore.NOIDField
		Number           int64
		RandomSeed       int64
		hasRandomSeed    uint32
		Block            *block.Block
		BlockHash        string
		VRFOutput        string
		minerPerm        []int
		phase            Phase
		proposedBlocks   []*block.Block
		notarizedBlocks  []*block.Block
		shares           map[string]*VRFShare
		softTimeoutCount int32
		vrfStartTime     atomic.Value
	}
	tests := []struct {
		name   string
		fields fields
		want   int
	}{
		{
			name:   "OK",
			fields: fields{softTimeoutCount: int32(c)},
			want:   c,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := &Round{
				NOIDField:        tt.fields.NOIDField,
				Number:           tt.fields.Number,
				RandomSeed:       tt.fields.RandomSeed,
				Block:            tt.fields.Block,
				BlockHash:        tt.fields.BlockHash,
				VRFOutput:        tt.fields.VRFOutput,
				minerPerm:        tt.fields.minerPerm,
				phase:            tt.fields.phase,
				proposedBlocks:   tt.fields.proposedBlocks,
				notarizedBlocks:  tt.fields.notarizedBlocks,
				mutex:            sync.RWMutex{},
				shares:           tt.fields.shares,
				softTimeoutCount: tt.fields.softTimeoutCount,
				vrfStartTime:     tt.fields.vrfStartTime,
				timeoutCounter:   timeoutCounter{},
			}
			if got := r.GetSoftTimeoutCount(); got != tt.want {
				t.Errorf("GetSoftTimeoutCount() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRound_IncSoftTimeoutCount(t *testing.T) {
	t.Parallel()

	type fields struct {
		NOIDField        datastore.NOIDField
		Number           int64
		RandomSeed       int64
		hasRandomSeed    uint32
		Block            *block.Block
		BlockHash        string
		VRFOutput        string
		minerPerm        []int
		phase            Phase
		proposedBlocks   []*block.Block
		notarizedBlocks  []*block.Block
		shares           map[string]*VRFShare
		softTimeoutCount int32
		vrfStartTime     atomic.Value
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "OK",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := &Round{
				NOIDField:        tt.fields.NOIDField,
				Number:           tt.fields.Number,
				RandomSeed:       tt.fields.RandomSeed,
				Block:            tt.fields.Block,
				BlockHash:        tt.fields.BlockHash,
				VRFOutput:        tt.fields.VRFOutput,
				minerPerm:        tt.fields.minerPerm,
				phase:            tt.fields.phase,
				proposedBlocks:   tt.fields.proposedBlocks,
				notarizedBlocks:  tt.fields.notarizedBlocks,
				mutex:            sync.RWMutex{},
				shares:           tt.fields.shares,
				softTimeoutCount: tt.fields.softTimeoutCount,
				vrfStartTime:     tt.fields.vrfStartTime,
				timeoutCounter:   timeoutCounter{},
			}

			before := r.softTimeoutCount
			r.IncSoftTimeoutCount()
			if r.softTimeoutCount != before+1 {
				t.Errorf("IncSoftTimeoutCount() got = %v, want = %v", r.softTimeoutCount, before+1)
			}
		})
	}
}

func TestRound_GetVrfStartTime(t *testing.T) {
	t.Parallel()

	type fields struct {
		NOIDField        datastore.NOIDField
		Number           int64
		RandomSeed       int64
		hasRandomSeed    uint32
		Block            *block.Block
		BlockHash        string
		VRFOutput        string
		minerPerm        []int
		phase            Phase
		proposedBlocks   []*block.Block
		notarizedBlocks  []*block.Block
		shares           map[string]*VRFShare
		softTimeoutCount int32
		vrfStartTime     atomic.Value
	}
	tests := []struct {
		name          string
		fields        fields
		want          time.Time
		storingBefore bool
	}{
		{
			name:          "Nil_OK",
			storingBefore: false,
			want:          time.Time{},
		},
		{
			name:          "OK",
			storingBefore: true,
			want:          time.Now(),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := &Round{
				NOIDField:        tt.fields.NOIDField,
				Number:           tt.fields.Number,
				RandomSeed:       tt.fields.RandomSeed,
				Block:            tt.fields.Block,
				BlockHash:        tt.fields.BlockHash,
				VRFOutput:        tt.fields.VRFOutput,
				minerPerm:        tt.fields.minerPerm,
				phase:            tt.fields.phase,
				proposedBlocks:   tt.fields.proposedBlocks,
				notarizedBlocks:  tt.fields.notarizedBlocks,
				mutex:            sync.RWMutex{},
				shares:           tt.fields.shares,
				softTimeoutCount: tt.fields.softTimeoutCount,
				vrfStartTime:     tt.fields.vrfStartTime,
				timeoutCounter:   timeoutCounter{},
			}
			if tt.storingBefore {
				r.SetVrfStartTime(tt.want)
			}
			if got := r.GetVrfStartTime(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetVrfStartTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRound_IsFinalizing(t *testing.T) {
	t.Parallel()

	type fields struct {
		NOIDField        datastore.NOIDField
		Number           int64
		RandomSeed       int64
		hasRandomSeed    uint32
		Block            *block.Block
		BlockHash        string
		VRFOutput        string
		minerPerm        []int
		state            FinalizingState
		proposedBlocks   []*block.Block
		notarizedBlocks  []*block.Block
		shares           map[string]*VRFShare
		softTimeoutCount int32
		vrfStartTime     atomic.Value
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "TRUE",
			fields: fields{
				state: RoundStateFinalizing,
			},
			want: true,
		},
		{
			name: "FALSE",
			fields: fields{
				state: RoundStateFinalized,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := &Round{
				NOIDField:        tt.fields.NOIDField,
				Number:           tt.fields.Number,
				RandomSeed:       tt.fields.RandomSeed,
				Block:            tt.fields.Block,
				BlockHash:        tt.fields.BlockHash,
				VRFOutput:        tt.fields.VRFOutput,
				minerPerm:        tt.fields.minerPerm,
				finalizingState:  tt.fields.state,
				proposedBlocks:   tt.fields.proposedBlocks,
				notarizedBlocks:  tt.fields.notarizedBlocks,
				mutex:            sync.RWMutex{},
				shares:           tt.fields.shares,
				softTimeoutCount: tt.fields.softTimeoutCount,
				vrfStartTime:     tt.fields.vrfStartTime,
				timeoutCounter:   timeoutCounter{},
			}
			if got := r.IsFinalizing(); got != tt.want {
				t.Errorf("IsFinalizing() = %v, want %v", got, tt.want)
			}
		})
	}
}

func makeTestTimeoutCounter() timeoutCounter {
	return timeoutCounter{
		mutex: sync.RWMutex{},
		prrs:  1,
		perm:  []string{"miner"},
		count: 1,
		votes: make(map[string]int),
	}
}

func Test_timeoutCounter_AddTimeoutVote(t *testing.T) {
	t.Parallel()

	tc := makeTestTimeoutCounter()
	id := "id"
	num := 2
	tc.votes[id] = num

	type fields struct {
		prrs  int64
		perm  []string
		count int
		votes map[string]int
	}
	type args struct {
		num int
		id  string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *timeoutCounter
	}{
		{
			name: "OK",
			fields: fields{
				prrs:  tc.prrs,
				perm:  tc.perm,
				count: tc.count,
				votes: nil,
			},
			args: args{
				num: num,
				id:  id,
			},
			want: &tc,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tc := &timeoutCounter{
				mutex: sync.RWMutex{},
				prrs:  tt.fields.prrs,
				perm:  tt.fields.perm,
				count: tt.fields.count,
				votes: tt.fields.votes,
			}
			tc.AddTimeoutVote(tt.args.num, tt.args.id)
			if !assert.Equal(t, tc, tt.want) {
				t.Errorf("AddTimeoutVote() got = %v, want = %v", tc, tt.want)
			}
		})
	}
}

func Test_timeoutCounter_GetTimeoutCount(t *testing.T) {
	t.Parallel()

	type fields struct {
		prrs  int64
		perm  []string
		count int
		votes map[string]int
	}
	tests := []struct {
		name      string
		fields    fields
		wantCount int
	}{
		{
			name:      "OK",
			wantCount: 0,
		},
		{
			name:      "OK2",
			wantCount: 1,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tc := &timeoutCounter{
				mutex: sync.RWMutex{},
				prrs:  tt.fields.prrs,
				perm:  tt.fields.perm,
				count: tt.fields.count,
				votes: tt.fields.votes,
			}

			tc.SetTimeoutCount(tt.wantCount)
			if gotCount := tc.GetTimeoutCount(); gotCount != tt.wantCount {
				t.Errorf("GetTimeoutCount() = %v, want %v", gotCount, tt.wantCount)
			}
		})
	}
}

func Test_timeoutCounter_IncrementTimeoutCount(t *testing.T) {
	n, err := makeTestNode(blsPublicKeys[0])
	if err != nil {
		t.Fatal(err)
	}
	n.SetIndex = 1
	n.Type = node.NodeTypeMiner
	n2, err := makeTestNode(blsPublicKeys[1])
	if err != nil {
		t.Fatal(err)
	}
	n2.SetIndex = 2
	n2.Type = node.NodeTypeMiner
	n3, err := makeTestNode(blsPublicKeys[2])
	if err != nil {
		t.Fatal(err)
	}
	n3.SetIndex = 3
	n3.Type = node.NodeTypeMiner
	p := node.NewPool(node.NodeTypeMiner)
	p.AddNode(n)
	p.AddNode(n2)

	tc := makeTestTimeoutCounter()
	tc.votes[n2.ID] = 4
	tc.votes[n3.ID] = 4

	p2 := node.NewPool(node.NodeTypeMiner)
	p2.AddNode(n)
	p2.AddNode(n3)

	sortPerm := func(ids []string) []string {
		sort.SliceStable(ids, func(i, j int) bool {
			return ids[i] < ids[j]
		})
		return ids
	}

	type fields struct {
		prrs  int64
		perm  []string
		count int
		votes map[string]int
	}
	type args struct {
		prrs   int64
		miners *node.Pool
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *timeoutCounter
	}{
		{
			name: "Zero_Prrs_OK",
			fields: fields{
				prrs:  tc.prrs,
				perm:  tc.perm,
				count: tc.count,
				votes: nil,
			},
			want: &timeoutCounter{
				prrs:  tc.prrs,
				perm:  tc.perm,
				count: tc.count,
				votes: nil,
			},
		},
		{
			name: "Nil_Votes_OK",
			args: args{prrs: 1},
			fields: fields{
				prrs:  tc.prrs,
				perm:  tc.perm,
				count: tc.count,
				votes: nil,
			},
			want: &timeoutCounter{
				prrs:  tc.prrs,
				perm:  tc.perm,
				count: tc.count + 1,
				votes: make(map[string]int),
			},
		},
		{
			name: "OK",
			fields: fields{
				prrs:  tc.prrs,
				perm:  make([]string, 0, 1),
				count: tc.count,
				votes: tc.votes,
			},
			args: args{prrs: 1, miners: p},
			want: &timeoutCounter{
				prrs: tc.prrs,
				perm: sortPerm([]string{
					n.ID,
					n2.ID,
				}),
				count: 4,
				votes: make(map[string]int),
			},
		},
		{
			name: "OK2",
			fields: fields{
				prrs:  tc.prrs,
				perm:  make([]string, 0, 1),
				count: tc.count,
				votes: map[string]int{
					n3.ID: 5,
					n.ID:  5,
				},
			},
			args: args{prrs: 1, miners: p2},
			want: &timeoutCounter{
				prrs: tc.prrs,
				perm: sortPerm([]string{
					n.ID,
					n3.ID,
				}),
				count: 5,
				votes: make(map[string]int),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := &timeoutCounter{
				mutex: sync.RWMutex{},
				prrs:  tt.fields.prrs,
				perm:  tt.fields.perm,
				count: tt.fields.count,
				votes: tt.fields.votes,
			}

			tc.IncrementTimeoutCount(tt.args.prrs, tt.args.miners)
			if !assert.Equal(t, tt.want, tc) {
				t.Errorf("AddTimeoutVote() got = %v, want = %v", tc, tt.want)
			}
		})
	}
}
