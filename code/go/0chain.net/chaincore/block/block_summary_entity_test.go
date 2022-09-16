package block

import (
	"context"
	"encoding/json"
	"reflect"
	"strconv"
	"testing"

	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/memorystore"
	"0chain.net/core/mocks"
	"github.com/0chain/common/core/util"
)

func init() {
	SetupMagicBlockMapEntity(memorystore.GetStorageProvider())
}

func TestBlockSummary_GetEntityMetadata(t *testing.T) {
	type fields struct {
		VersionField          datastore.VersionField
		CreationDateField     datastore.CreationDateField
		NOIDField             datastore.NOIDField
		Hash                  string
		MinerID               datastore.Key
		Round                 int64
		RoundRandomSeed       int64
		MerkleTreeRoot        string
		ClientStateHash       util.Key
		ReceiptMerkleTreeRoot string
		NumTxns               int
		MagicBlock            *MagicBlock
	}
	tests := []struct {
		name   string
		fields fields
		want   datastore.EntityMetadata
	}{
		{
			name: "OK",
			want: blockSummaryEntityMetadata,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &BlockSummary{
				VersionField:          tt.fields.VersionField,
				CreationDateField:     tt.fields.CreationDateField,
				NOIDField:             tt.fields.NOIDField,
				Hash:                  tt.fields.Hash,
				MinerID:               tt.fields.MinerID,
				Round:                 tt.fields.Round,
				RoundRandomSeed:       tt.fields.RoundRandomSeed,
				MerkleTreeRoot:        tt.fields.MerkleTreeRoot,
				ClientStateHash:       tt.fields.ClientStateHash,
				ReceiptMerkleTreeRoot: tt.fields.ReceiptMerkleTreeRoot,
				NumTxns:               tt.fields.NumTxns,
				MagicBlock:            tt.fields.MagicBlock,
			}
			if got := b.GetEntityMetadata(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetEntityMetadata() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBlockSummary_GetKey(t *testing.T) {
	type fields struct {
		VersionField          datastore.VersionField
		CreationDateField     datastore.CreationDateField
		NOIDField             datastore.NOIDField
		Hash                  string
		MinerID               datastore.Key
		Round                 int64
		RoundRandomSeed       int64
		MerkleTreeRoot        string
		ClientStateHash       util.Key
		ReceiptMerkleTreeRoot string
		NumTxns               int
		MagicBlock            *MagicBlock
	}
	tests := []struct {
		name   string
		fields fields
		want   datastore.Key
	}{
		{
			name: "OK",
			want: encryption.Hash("data"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &BlockSummary{
				VersionField:          tt.fields.VersionField,
				CreationDateField:     tt.fields.CreationDateField,
				NOIDField:             tt.fields.NOIDField,
				Hash:                  tt.fields.Hash,
				MinerID:               tt.fields.MinerID,
				Round:                 tt.fields.Round,
				RoundRandomSeed:       tt.fields.RoundRandomSeed,
				MerkleTreeRoot:        tt.fields.MerkleTreeRoot,
				ClientStateHash:       tt.fields.ClientStateHash,
				ReceiptMerkleTreeRoot: tt.fields.ReceiptMerkleTreeRoot,
				NumTxns:               tt.fields.NumTxns,
				MagicBlock:            tt.fields.MagicBlock,
			}
			b.SetKey(tt.want)
			if got := b.GetKey(); got != tt.want {
				t.Errorf("GetKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBlockSummary_Read(t *testing.T) {
	sm := mocks.Store{}
	blockSummaryEntityMetadata.Store = &sm
	sm.On("Read", context.Context(nil), "", new(BlockSummary)).Return(
		func(_ context.Context, _ string, _ datastore.Entity) error {
			return nil
		},
	)

	type fields struct {
		VersionField          datastore.VersionField
		CreationDateField     datastore.CreationDateField
		NOIDField             datastore.NOIDField
		Hash                  string
		MinerID               datastore.Key
		Round                 int64
		RoundRandomSeed       int64
		MerkleTreeRoot        string
		ClientStateHash       util.Key
		ReceiptMerkleTreeRoot string
		NumTxns               int
		MagicBlock            *MagicBlock
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
		t.Run(tt.name, func(t *testing.T) {
			b := &BlockSummary{
				VersionField:          tt.fields.VersionField,
				CreationDateField:     tt.fields.CreationDateField,
				NOIDField:             tt.fields.NOIDField,
				Hash:                  tt.fields.Hash,
				MinerID:               tt.fields.MinerID,
				Round:                 tt.fields.Round,
				RoundRandomSeed:       tt.fields.RoundRandomSeed,
				MerkleTreeRoot:        tt.fields.MerkleTreeRoot,
				ClientStateHash:       tt.fields.ClientStateHash,
				ReceiptMerkleTreeRoot: tt.fields.ReceiptMerkleTreeRoot,
				NumTxns:               tt.fields.NumTxns,
				MagicBlock:            tt.fields.MagicBlock,
			}
			if err := b.Read(tt.args.ctx, tt.args.key); (err != nil) != tt.wantErr {
				t.Errorf("Read() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBlockSummary_Write(t *testing.T) {
	sm := mocks.Store{}
	blockSummaryEntityMetadata.Store = &sm
	sm.On("Write", context.Context(nil), new(BlockSummary)).Return(
		func(_ context.Context, _ datastore.Entity) error {
			return nil
		},
	)

	type fields struct {
		VersionField          datastore.VersionField
		CreationDateField     datastore.CreationDateField
		NOIDField             datastore.NOIDField
		Hash                  string
		MinerID               datastore.Key
		Round                 int64
		RoundRandomSeed       int64
		MerkleTreeRoot        string
		ClientStateHash       util.Key
		ReceiptMerkleTreeRoot string
		NumTxns               int
		MagicBlock            *MagicBlock
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
		t.Run(tt.name, func(t *testing.T) {
			b := &BlockSummary{
				VersionField:          tt.fields.VersionField,
				CreationDateField:     tt.fields.CreationDateField,
				NOIDField:             tt.fields.NOIDField,
				Hash:                  tt.fields.Hash,
				MinerID:               tt.fields.MinerID,
				Round:                 tt.fields.Round,
				RoundRandomSeed:       tt.fields.RoundRandomSeed,
				MerkleTreeRoot:        tt.fields.MerkleTreeRoot,
				ClientStateHash:       tt.fields.ClientStateHash,
				ReceiptMerkleTreeRoot: tt.fields.ReceiptMerkleTreeRoot,
				NumTxns:               tt.fields.NumTxns,
				MagicBlock:            tt.fields.MagicBlock,
			}
			if err := b.Write(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("Write() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBlockSummary_Delete(t *testing.T) {
	sm := mocks.Store{}
	blockSummaryEntityMetadata.Store = &sm
	sm.On("Delete", context.Context(nil), new(BlockSummary)).Return(
		func(_ context.Context, _ datastore.Entity) error {
			return nil
		},
	)

	type fields struct {
		VersionField          datastore.VersionField
		CreationDateField     datastore.CreationDateField
		NOIDField             datastore.NOIDField
		Hash                  string
		MinerID               datastore.Key
		Round                 int64
		RoundRandomSeed       int64
		MerkleTreeRoot        string
		ClientStateHash       util.Key
		ReceiptMerkleTreeRoot string
		NumTxns               int
		MagicBlock            *MagicBlock
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
		t.Run(tt.name, func(t *testing.T) {
			b := &BlockSummary{
				VersionField:          tt.fields.VersionField,
				CreationDateField:     tt.fields.CreationDateField,
				NOIDField:             tt.fields.NOIDField,
				Hash:                  tt.fields.Hash,
				MinerID:               tt.fields.MinerID,
				Round:                 tt.fields.Round,
				RoundRandomSeed:       tt.fields.RoundRandomSeed,
				MerkleTreeRoot:        tt.fields.MerkleTreeRoot,
				ClientStateHash:       tt.fields.ClientStateHash,
				ReceiptMerkleTreeRoot: tt.fields.ReceiptMerkleTreeRoot,
				NumTxns:               tt.fields.NumTxns,
				MagicBlock:            tt.fields.MagicBlock,
			}
			if err := b.Delete(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBlockSummary_Encode(t *testing.T) {
	bs := NewBlock("", 1).GetSummary()
	blob, err := json.Marshal(bs)
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		VersionField          datastore.VersionField
		CreationDateField     datastore.CreationDateField
		NOIDField             datastore.NOIDField
		Hash                  string
		MinerID               datastore.Key
		Round                 int64
		RoundRandomSeed       int64
		MerkleTreeRoot        string
		ClientStateHash       util.Key
		ReceiptMerkleTreeRoot string
		NumTxns               int
		MagicBlock            *MagicBlock
	}
	tests := []struct {
		name   string
		fields fields
		want   []byte
	}{
		{
			name: "OK",
			fields: fields{
				VersionField:          bs.VersionField,
				CreationDateField:     bs.CreationDateField,
				NOIDField:             bs.NOIDField,
				Hash:                  bs.Hash,
				MinerID:               bs.MinerID,
				Round:                 bs.Round,
				RoundRandomSeed:       bs.RoundRandomSeed,
				MerkleTreeRoot:        bs.MerkleTreeRoot,
				ClientStateHash:       bs.ClientStateHash,
				ReceiptMerkleTreeRoot: bs.ReceiptMerkleTreeRoot,
				NumTxns:               bs.NumTxns,
				MagicBlock:            bs.MagicBlock,
			},
			want: blob,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &BlockSummary{
				VersionField:          tt.fields.VersionField,
				CreationDateField:     tt.fields.CreationDateField,
				NOIDField:             tt.fields.NOIDField,
				Hash:                  tt.fields.Hash,
				MinerID:               tt.fields.MinerID,
				Round:                 tt.fields.Round,
				RoundRandomSeed:       tt.fields.RoundRandomSeed,
				MerkleTreeRoot:        tt.fields.MerkleTreeRoot,
				ClientStateHash:       tt.fields.ClientStateHash,
				ReceiptMerkleTreeRoot: tt.fields.ReceiptMerkleTreeRoot,
				NumTxns:               tt.fields.NumTxns,
				MagicBlock:            tt.fields.MagicBlock,
			}
			if got := b.Encode(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Encode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBlockSummary_Decode(t *testing.T) {
	bs := NewBlock("", 1).GetSummary()
	blob, err := json.Marshal(bs)
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		VersionField          datastore.VersionField
		CreationDateField     datastore.CreationDateField
		NOIDField             datastore.NOIDField
		Hash                  string
		MinerID               datastore.Key
		Round                 int64
		RoundRandomSeed       int64
		MerkleTreeRoot        string
		ClientStateHash       util.Key
		ReceiptMerkleTreeRoot string
		NumTxns               int
		MagicBlock            *MagicBlock
	}
	type args struct {
		input []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *BlockSummary
		wantErr bool
	}{
		{
			name: "OK",
			args: args{input: blob},
			want: bs,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &BlockSummary{
				VersionField:          tt.fields.VersionField,
				CreationDateField:     tt.fields.CreationDateField,
				NOIDField:             tt.fields.NOIDField,
				Hash:                  tt.fields.Hash,
				MinerID:               tt.fields.MinerID,
				Round:                 tt.fields.Round,
				RoundRandomSeed:       tt.fields.RoundRandomSeed,
				MerkleTreeRoot:        tt.fields.MerkleTreeRoot,
				ClientStateHash:       tt.fields.ClientStateHash,
				ReceiptMerkleTreeRoot: tt.fields.ReceiptMerkleTreeRoot,
				NumTxns:               tt.fields.NumTxns,
				MagicBlock:            tt.fields.MagicBlock,
			}
			if err := b.Decode(tt.args.input); (err != nil) != tt.wantErr {
				t.Errorf("Decode() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(b, tt.want) {
				t.Errorf("Decode() got = %v, want %v", b, tt.want)
			}
		})
	}
}

func TestBlockSummary_GetMagicBlockMap(t *testing.T) {
	bs := NewBlock("", 1).GetSummary()
	bs.MagicBlock = NewMagicBlock()
	bs.MagicBlockNumber = 2
	bs.Hash = encryption.Hash("data")
	bs.Round = 2

	mbm := datastore.GetEntityMetadata("magic_block_map").Instance().(*MagicBlockMap)
	mbm.ID = strconv.FormatInt(bs.MagicBlockNumber, 10)
	mbm.Hash = bs.Hash
	mbm.BlockRound = bs.Round

	type fields struct {
		VersionField          datastore.VersionField
		CreationDateField     datastore.CreationDateField
		NOIDField             datastore.NOIDField
		Hash                  string
		MinerID               datastore.Key
		Round                 int64
		RoundRandomSeed       int64
		MerkleTreeRoot        string
		ClientStateHash       util.Key
		ReceiptMerkleTreeRoot string
		NumTxns               int
		MagicBlock            *MagicBlock
	}
	tests := []struct {
		name   string
		fields fields
		want   *MagicBlockMap
	}{
		{
			name: "OK",
			fields: fields{
				VersionField:          bs.VersionField,
				CreationDateField:     bs.CreationDateField,
				NOIDField:             bs.NOIDField,
				Hash:                  bs.Hash,
				MinerID:               bs.MinerID,
				Round:                 bs.Round,
				RoundRandomSeed:       bs.RoundRandomSeed,
				MerkleTreeRoot:        bs.MerkleTreeRoot,
				ClientStateHash:       bs.ClientStateHash,
				ReceiptMerkleTreeRoot: bs.ReceiptMerkleTreeRoot,
				NumTxns:               bs.NumTxns,
				MagicBlock:            bs.MagicBlock,
			},
			want: mbm,
		},
		{
			name: "Nil_MBM_OK",
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &BlockSummary{
				VersionField:          tt.fields.VersionField,
				CreationDateField:     tt.fields.CreationDateField,
				NOIDField:             tt.fields.NOIDField,
				Hash:                  tt.fields.Hash,
				MinerID:               tt.fields.MinerID,
				Round:                 tt.fields.Round,
				RoundRandomSeed:       tt.fields.RoundRandomSeed,
				MerkleTreeRoot:        tt.fields.MerkleTreeRoot,
				ClientStateHash:       tt.fields.ClientStateHash,
				ReceiptMerkleTreeRoot: tt.fields.ReceiptMerkleTreeRoot,
				NumTxns:               tt.fields.NumTxns,
				MagicBlock:            tt.fields.MagicBlock,
			}
			if got := b.GetMagicBlockMap(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetMagicBlockMap() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSetupBlockSummaryDB_Panic(t *testing.T) {
	tests := []struct {
		name      string
		wantPanic bool
	}{
		{
			name:      "Panic",
			wantPanic: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				got := recover()
				if (got != nil) != tt.wantPanic {
					t.Errorf("SetupBlockSummaryDB() want panic  = %v, but got = %v", tt.wantPanic, got)
				}
			}()

			SetupBlockSummaryDB("")
		})
	}
}
