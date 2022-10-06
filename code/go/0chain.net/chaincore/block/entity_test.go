package block

import (
	"context"
	"encoding/json"
	"reflect"
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"0chain.net/chaincore/client"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/memorystore"
	"0chain.net/core/mocks"
	"github.com/0chain/common/core/logging"
	"github.com/0chain/common/core/util"
)

func init() {
	sp := memorystore.GetStorageProvider()
	SetupEntity(sp)
	SetupBlockSummaryEntity(sp)

	clientEM := datastore.MetadataProvider()
	clientEM.Name = "client"
	clientEM.Provider = client.Provider
	clientEM.Store = sp
	datastore.RegisterEntityMetadata("client", clientEM)

	logging.InitLogging("testing", "")
	config.SetServerChainID("")
}

// NOTE: copyBlock does not copy Block.ClientState and Block.*MagicBlock fields.
func copyBlock(b *Block) *Block {
	if b == nil {
		return nil
	}

	copiedB := Block{
		HashIDField:        b.HashIDField,
		Signature:          b.Signature,
		ChainID:            b.ChainID,
		RoundRank:          b.RoundRank,
		PrevBlock:          copyBlock(b.PrevBlock),
		ClientState:        nil,
		stateStatus:        b.stateStatus,
		blockState:         b.blockState,
		isNotarized:        b.isNotarized,
		verificationStatus: b.verificationStatus,
		RunningTxnCount:    b.RunningTxnCount,
		MagicBlock:         nil,
	}

	copiedB.UnverifiedBlockBody = b.UnverifiedBlockBody
	if b.PrevBlockVerificationTickets != nil {
		copiedB.PrevBlockVerificationTickets = copyVerTickets(b.PrevBlockVerificationTickets)
	}
	if b.Txns != nil {
		copiedB.Txns = make([]*transaction.Transaction, len(b.Txns))
		for i, v := range b.Txns {
			copiedB.Txns[i] = v.Clone()
		}
	}

	if b.VerificationTickets != nil {
		copiedB.VerificationTickets = copyVerTickets(b.VerificationTickets)
	}

	if b.TxnsMap != nil {
		copiedB.TxnsMap = make(map[string]bool)
		for k, v := range b.TxnsMap {
			copiedB.TxnsMap[k] = v
		}
	}

	if b.UniqueBlockExtensions != nil {
		copiedB.UniqueBlockExtensions = make(map[string]bool)
		for k, v := range b.UniqueBlockExtensions {
			copiedB.UniqueBlockExtensions[k] = v
		}
	}

	return &copiedB
}

func copyVerTickets(t []*VerificationTicket) []*VerificationTicket {
	copiedT := make([]*VerificationTicket, len(t))
	for i, v := range t {
		copiedT[i] = &VerificationTicket{
			VerifierID: v.VerifierID,
			Signature:  v.Signature,
		}
	}

	return copiedT
}

func makeTestNode(pbK string) (*node.Node, error) {
	if pbK == "" {
		ss := encryption.NewBLS0ChainScheme()
		ss.GenerateKeys()
		pbK = ss.GetPublicKey()
	}

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

func TestNewBlock(t *testing.T) {
	var r int64 = 1

	type args struct {
		chainID datastore.Key
		round   int64
	}
	tests := []struct {
		name string
		args args
		want *Block
	}{
		{
			name: "OK",
			args: args{round: r},
			want: func() *Block {
				b := datastore.GetEntityMetadata("block").Instance().(*Block)
				b.Round = r
				b.ChainID = ""
				return b
			}(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewBlock(tt.args.chainID, tt.args.round); !assert.Equal(t, tt.want, got) {
				t.Errorf("NewBlock() = %v, want %v", tt.want, got)
			}
		})
	}
}

func TestBlock_GetVerificationTickets(t *testing.T) {
	scheme := encryption.NewBLS0ChainScheme()
	if err := scheme.GenerateKeys(); err != nil {
		t.Fatal(err)
	}
	sign, err := scheme.Sign(encryption.Hash("data"))
	if err != nil {
		t.Fatal(err)
	}

	anotherScheme := encryption.NewBLS0ChainScheme()
	if err := anotherScheme.GenerateKeys(); err != nil {
		t.Fatal(err)
	}
	anotherSign, err := anotherScheme.Sign(encryption.Hash("data"))
	if err != nil {
		t.Fatal(err)
	}

	b := NewBlock("", 1)
	b.VerificationTickets = []*VerificationTicket{
		{
			VerifierID: "123",
			Signature:  sign,
		},
		{
			VerifierID: "124",
			Signature:  anotherSign,
		},
	}

	type fields struct {
		UnverifiedBlockBody   UnverifiedBlockBody
		VerificationTickets   []*VerificationTicket
		HashIDField           datastore.HashIDField
		Signature             string
		ChainID               datastore.Key
		RoundRank             int
		PrevBlock             *Block
		TxnsMap               map[string]bool
		ClientState           util.MerklePatriciaTrieI
		stateStatus           int8
		blockState            int8
		isNotarized           bool
		verificationStatus    int
		RunningTxnCount       int64
		UniqueBlockExtensions map[string]bool
		MagicBlock            *MagicBlock
	}
	var tests = []struct {
		name    string
		fields  fields
		wantVts []*VerificationTicket
	}{
		{
			name: "OK",
			fields: fields{
				VerificationTickets: b.VerificationTickets,
			},
			wantVts: []*VerificationTicket{
				b.VerificationTickets[0].Copy(),
				b.VerificationTickets[1].Copy(),
			},
		},
		{
			name:    "Empty_Tickets_OK",
			fields:  fields{},
			wantVts: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Block{
				UnverifiedBlockBody:   tt.fields.UnverifiedBlockBody,
				VerificationTickets:   tt.fields.VerificationTickets,
				HashIDField:           tt.fields.HashIDField,
				Signature:             tt.fields.Signature,
				ChainID:               tt.fields.ChainID,
				RoundRank:             tt.fields.RoundRank,
				PrevBlock:             tt.fields.PrevBlock,
				TxnsMap:               tt.fields.TxnsMap,
				ClientState:           tt.fields.ClientState,
				stateStatus:           tt.fields.stateStatus,
				blockState:            tt.fields.blockState,
				isNotarized:           tt.fields.isNotarized,
				verificationStatus:    tt.fields.verificationStatus,
				RunningTxnCount:       tt.fields.RunningTxnCount,
				UniqueBlockExtensions: tt.fields.UniqueBlockExtensions,
				MagicBlock:            tt.fields.MagicBlock,
			}
			if gotVts := b.GetVerificationTickets(); !reflect.DeepEqual(gotVts, tt.wantVts) {
				t.Errorf("GetVerificationTickets() = %v, want %v", gotVts, tt.wantVts)
			}
		})
	}
}

func TestBlock_VerificationTicketsSize(t *testing.T) {
	b := NewBlock("", 1)
	b.VerificationTickets = []*VerificationTicket{
		{
			VerifierID: "123",
		},
		{
			VerifierID: "124",
		},
	}

	type fields struct {
		UnverifiedBlockBody   UnverifiedBlockBody
		VerificationTickets   []*VerificationTicket
		HashIDField           datastore.HashIDField
		Signature             string
		ChainID               datastore.Key
		RoundRank             int
		PrevBlock             *Block
		TxnsMap               map[string]bool
		ClientState           util.MerklePatriciaTrieI
		stateStatus           int8
		blockState            int8
		isNotarized           bool
		verificationStatus    int
		RunningTxnCount       int64
		UniqueBlockExtensions map[string]bool
		MagicBlock            *MagicBlock
	}
	tests := []struct {
		name   string
		fields fields
		want   int
	}{
		{
			name: "OK",
			fields: fields{
				VerificationTickets: b.VerificationTickets,
			},
			want: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Block{
				UnverifiedBlockBody:   tt.fields.UnverifiedBlockBody,
				VerificationTickets:   tt.fields.VerificationTickets,
				HashIDField:           tt.fields.HashIDField,
				Signature:             tt.fields.Signature,
				ChainID:               tt.fields.ChainID,
				RoundRank:             tt.fields.RoundRank,
				PrevBlock:             tt.fields.PrevBlock,
				TxnsMap:               tt.fields.TxnsMap,
				ClientState:           tt.fields.ClientState,
				stateStatus:           tt.fields.stateStatus,
				blockState:            tt.fields.blockState,
				isNotarized:           tt.fields.isNotarized,
				verificationStatus:    tt.fields.verificationStatus,
				RunningTxnCount:       tt.fields.RunningTxnCount,
				UniqueBlockExtensions: tt.fields.UniqueBlockExtensions,
				MagicBlock:            tt.fields.MagicBlock,
			}
			if got := b.VerificationTicketsSize(); got != tt.want {
				t.Errorf("VerificationTicketsSize() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBlock_GetEntityMetadata(t *testing.T) {
	type fields struct {
		UnverifiedBlockBody   UnverifiedBlockBody
		VerificationTickets   []*VerificationTicket
		HashIDField           datastore.HashIDField
		Signature             string
		ChainID               datastore.Key
		RoundRank             int
		PrevBlock             *Block
		TxnsMap               map[string]bool
		ClientState           util.MerklePatriciaTrieI
		stateStatus           int8
		blockState            int8
		isNotarized           bool
		verificationStatus    int
		RunningTxnCount       int64
		UniqueBlockExtensions map[string]bool
		MagicBlock            *MagicBlock
	}
	tests := []struct {
		name   string
		fields fields
		want   datastore.EntityMetadata
	}{
		{
			name: "OK",
			want: blockEntityMetadata,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Block{
				UnverifiedBlockBody:   tt.fields.UnverifiedBlockBody,
				VerificationTickets:   tt.fields.VerificationTickets,
				HashIDField:           tt.fields.HashIDField,
				Signature:             tt.fields.Signature,
				ChainID:               tt.fields.ChainID,
				RoundRank:             tt.fields.RoundRank,
				PrevBlock:             tt.fields.PrevBlock,
				TxnsMap:               tt.fields.TxnsMap,
				ClientState:           tt.fields.ClientState,
				stateStatus:           tt.fields.stateStatus,
				blockState:            tt.fields.blockState,
				isNotarized:           tt.fields.isNotarized,
				verificationStatus:    tt.fields.verificationStatus,
				RunningTxnCount:       tt.fields.RunningTxnCount,
				UniqueBlockExtensions: tt.fields.UniqueBlockExtensions,
				MagicBlock:            tt.fields.MagicBlock,
			}
			if got := b.GetEntityMetadata(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetEntityMetadata() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBlock_ComputeProperties(t *testing.T) {
	t.Parallel()

	b := NewBlock("", 1)
	txn := new(transaction.Transaction)

	scheme := encryption.NewBLS0ChainScheme()
	err := scheme.GenerateKeys()
	require.NoError(t, err)
	txn.PublicKey = scheme.GetPublicKey()

	b.Txns = []*transaction.Transaction{txn}

	tests := []struct {
		name   string
		fields *Block
		want   *Block
	}{
		{
			name:   "OK",
			fields: copyBlock(b),
			want: func() *Block {
				want := NewBlock("", 1)
				want.Txns = b.Txns
				want.ChainID = datastore.ToKey(config.GetServerChainID())
				want.TxnsMap = make(map[string]bool, len(want.Txns))
				for _, txn := range want.Txns {
					err := txn.ComputeProperties()
					require.NoError(t, err)
					want.TxnsMap[txn.Hash] = true
				}

				return want
			}(),
		},
		// duplicating tests to expose race errors
		{
			name:   "OK",
			fields: copyBlock(b),
			want: func() *Block {
				want := NewBlock("", 1)
				want.Txns = b.Txns
				want.ChainID = datastore.ToKey(config.GetServerChainID())
				want.TxnsMap = make(map[string]bool, len(want.Txns))
				for _, txn := range want.Txns {
					err := txn.ComputeProperties()
					require.NoError(t, err)
					want.TxnsMap[txn.Hash] = true
				}

				return want
			}(),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			b := &Block{
				UnverifiedBlockBody:   tt.fields.UnverifiedBlockBody,
				VerificationTickets:   tt.fields.VerificationTickets,
				HashIDField:           tt.fields.HashIDField,
				Signature:             tt.fields.Signature,
				ChainID:               tt.fields.ChainID,
				RoundRank:             tt.fields.RoundRank,
				PrevBlock:             tt.fields.PrevBlock,
				TxnsMap:               tt.fields.TxnsMap,
				ClientState:           tt.fields.ClientState,
				stateStatus:           tt.fields.stateStatus,
				blockState:            tt.fields.blockState,
				isNotarized:           tt.fields.isNotarized,
				verificationStatus:    tt.fields.verificationStatus,
				RunningTxnCount:       tt.fields.RunningTxnCount,
				UniqueBlockExtensions: tt.fields.UniqueBlockExtensions,
				MagicBlock:            tt.fields.MagicBlock,
			}

			err := b.ComputeProperties()
			require.NoError(t, err)

			assert.Equal(t, tt.want, b)
		})
	}
}

func TestBlock_Decode(t *testing.T) {
	t.Parallel()

	b := NewBlock("", 1)
	byt, err := json.Marshal(b)
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		UnverifiedBlockBody   UnverifiedBlockBody
		VerificationTickets   []*VerificationTicket
		HashIDField           datastore.HashIDField
		Signature             string
		ChainID               datastore.Key
		RoundRank             int
		PrevBlock             *Block
		TxnsMap               map[string]bool
		ClientState           util.MerklePatriciaTrieI
		stateStatus           int8
		blockState            int8
		isNotarized           bool
		verificationStatus    int
		RunningTxnCount       int64
		UniqueBlockExtensions map[string]bool
		MagicBlock            *MagicBlock
	}
	type args struct {
		input []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *Block
		wantErr bool
	}{
		{
			name:    "OK",
			args:    args{input: byt},
			want:    copyBlock(b),
			wantErr: false,
		},
		// duplicating tests to expose race errors
		{
			name:    "OK",
			args:    args{input: byt},
			want:    copyBlock(b),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			b := &Block{
				UnverifiedBlockBody:   tt.fields.UnverifiedBlockBody,
				VerificationTickets:   tt.fields.VerificationTickets,
				HashIDField:           tt.fields.HashIDField,
				Signature:             tt.fields.Signature,
				ChainID:               tt.fields.ChainID,
				RoundRank:             tt.fields.RoundRank,
				PrevBlock:             tt.fields.PrevBlock,
				TxnsMap:               tt.fields.TxnsMap,
				ClientState:           tt.fields.ClientState,
				stateStatus:           tt.fields.stateStatus,
				blockState:            tt.fields.blockState,
				isNotarized:           tt.fields.isNotarized,
				verificationStatus:    tt.fields.verificationStatus,
				RunningTxnCount:       tt.fields.RunningTxnCount,
				UniqueBlockExtensions: tt.fields.UniqueBlockExtensions,
				MagicBlock:            tt.fields.MagicBlock,
			}
			if err := b.Decode(tt.args.input); (err != nil) != tt.wantErr {
				t.Errorf("Decode() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && !assert.Equal(t, tt.want, b) {
				t.Errorf("Decode() got = %v, want = %v", b, tt.want)
			}
		})
	}
}

func TestBlock_Validate(t *testing.T) {
	pbK, prK, err := encryption.GenerateKeys()
	if err != nil {
		t.Fatal(err)
	}
	client.SetClientSignatureScheme("ed25519")

	n, err := makeTestNode(pbK)
	if err != nil {
		t.Fatal(err)
	}
	node.RegisterNode(n)

	type fields struct {
		UnverifiedBlockBody   UnverifiedBlockBody
		VerificationTickets   []*VerificationTicket
		HashIDField           datastore.HashIDField
		Signature             string
		ChainID               datastore.Key
		RoundRank             int
		PrevBlock             *Block
		TxnsMap               map[string]bool
		ClientState           util.MerklePatriciaTrieI
		stateStatus           int8
		blockState            int8
		isNotarized           bool
		verificationStatus    int
		RunningTxnCount       int64
		UniqueBlockExtensions map[string]bool
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
			name: "Invalid_Chain_ERR",
			fields: func() fields {
				b := NewBlock("", 1)
				b.ChainID = "unknown id"

				return fields{
					UnverifiedBlockBody:   b.UnverifiedBlockBody,
					VerificationTickets:   b.VerificationTickets,
					HashIDField:           b.HashIDField,
					Signature:             b.Signature,
					ChainID:               b.ChainID,
					RoundRank:             b.RoundRank,
					PrevBlock:             b.PrevBlock,
					TxnsMap:               b.TxnsMap,
					ClientState:           b.ClientState,
					stateStatus:           b.stateStatus,
					blockState:            b.blockState,
					isNotarized:           b.isNotarized,
					verificationStatus:    b.verificationStatus,
					RunningTxnCount:       b.RunningTxnCount,
					UniqueBlockExtensions: b.UniqueBlockExtensions,
					MagicBlock:            b.MagicBlock,
				}
			}(),
			wantErr: true,
		},
		{
			name: "Empty_Hash_ERR",
			fields: func() fields {
				b := NewBlock("", 1)
				b.ChainID = config.ServerChainID

				return fields{
					UnverifiedBlockBody:   b.UnverifiedBlockBody,
					VerificationTickets:   b.VerificationTickets,
					HashIDField:           b.HashIDField,
					Signature:             b.Signature,
					ChainID:               b.ChainID,
					RoundRank:             b.RoundRank,
					PrevBlock:             b.PrevBlock,
					TxnsMap:               b.TxnsMap,
					ClientState:           b.ClientState,
					stateStatus:           b.stateStatus,
					blockState:            b.blockState,
					isNotarized:           b.isNotarized,
					verificationStatus:    b.verificationStatus,
					RunningTxnCount:       b.RunningTxnCount,
					UniqueBlockExtensions: b.UniqueBlockExtensions,
					MagicBlock:            b.MagicBlock,
				}
			}(),
			wantErr: true,
		},
		{
			name: "Empty_MinerID_ERR",
			fields: func() fields {
				b := NewBlock("", 1)
				b.ChainID = config.ServerChainID
				b.Hash = b.ComputeHash()

				return fields{
					UnverifiedBlockBody:   b.UnverifiedBlockBody,
					VerificationTickets:   b.VerificationTickets,
					HashIDField:           b.HashIDField,
					Signature:             b.Signature,
					ChainID:               b.ChainID,
					RoundRank:             b.RoundRank,
					PrevBlock:             b.PrevBlock,
					TxnsMap:               b.TxnsMap,
					ClientState:           b.ClientState,
					stateStatus:           b.stateStatus,
					blockState:            b.blockState,
					isNotarized:           b.isNotarized,
					verificationStatus:    b.verificationStatus,
					RunningTxnCount:       b.RunningTxnCount,
					UniqueBlockExtensions: b.UniqueBlockExtensions,
					MagicBlock:            b.MagicBlock,
				}
			}(),
			wantErr: true,
		},
		{
			name: "Nil_Node_For_Miner_ERR",
			fields: func() fields {
				b := NewBlock("", 1)
				b.ChainID = config.ServerChainID
				b.Hash = b.ComputeHash()
				b.MinerID = "miner id"

				return fields{
					UnverifiedBlockBody:   b.UnverifiedBlockBody,
					VerificationTickets:   b.VerificationTickets,
					HashIDField:           b.HashIDField,
					Signature:             b.Signature,
					ChainID:               b.ChainID,
					RoundRank:             b.RoundRank,
					PrevBlock:             b.PrevBlock,
					TxnsMap:               b.TxnsMap,
					ClientState:           b.ClientState,
					stateStatus:           b.stateStatus,
					blockState:            b.blockState,
					isNotarized:           b.isNotarized,
					verificationStatus:    b.verificationStatus,
					RunningTxnCount:       b.RunningTxnCount,
					UniqueBlockExtensions: b.UniqueBlockExtensions,
					MagicBlock:            b.MagicBlock,
				}
			}(),
			wantErr: true,
		},
		{
			name: "Chain_Weight_Greater_Than_Round_ERR",
			fields: func() fields {
				b := NewBlock("", 1)
				b.ChainID = config.ServerChainID
				b.Hash = b.ComputeHash()
				b.MinerID = n.ID

				return fields{
					UnverifiedBlockBody:   b.UnverifiedBlockBody,
					VerificationTickets:   b.VerificationTickets,
					HashIDField:           b.HashIDField,
					Signature:             b.Signature,
					ChainID:               b.ChainID,
					RoundRank:             b.RoundRank,
					PrevBlock:             b.PrevBlock,
					TxnsMap:               b.TxnsMap,
					ClientState:           b.ClientState,
					stateStatus:           b.stateStatus,
					blockState:            b.blockState,
					isNotarized:           b.isNotarized,
					verificationStatus:    b.verificationStatus,
					RunningTxnCount:       b.RunningTxnCount,
					UniqueBlockExtensions: b.UniqueBlockExtensions,
					MagicBlock:            b.MagicBlock,
				}
			}(),
			wantErr: true,
		},
		{
			name: "Duplicate_Transactions_ERR",
			fields: func() fields {
				b := NewBlock("", 1)
				b.ChainID = config.ServerChainID
				b.Hash = b.ComputeHash()
				b.MinerID = n.ID
				b.TxnsMap = map[string]bool{
					"txn1": false,
				}

				return fields{
					UnverifiedBlockBody:   b.UnverifiedBlockBody,
					VerificationTickets:   b.VerificationTickets,
					HashIDField:           b.HashIDField,
					Signature:             b.Signature,
					ChainID:               b.ChainID,
					RoundRank:             b.RoundRank,
					PrevBlock:             b.PrevBlock,
					TxnsMap:               b.TxnsMap,
					ClientState:           b.ClientState,
					stateStatus:           b.stateStatus,
					blockState:            b.blockState,
					isNotarized:           b.isNotarized,
					verificationStatus:    b.verificationStatus,
					RunningTxnCount:       b.RunningTxnCount,
					UniqueBlockExtensions: b.UniqueBlockExtensions,
					MagicBlock:            b.MagicBlock,
				}
			}(),
			wantErr: true,
		},
		{
			name: "Diff_Hashes_ERR",
			fields: func() fields {
				b := NewBlock("", 1)
				b.ChainID = config.ServerChainID
				b.Hash = encryption.Hash("another data")
				b.MinerID = n.ID

				return fields{
					UnverifiedBlockBody:   b.UnverifiedBlockBody,
					VerificationTickets:   b.VerificationTickets,
					HashIDField:           b.HashIDField,
					Signature:             b.Signature,
					ChainID:               b.ChainID,
					RoundRank:             b.RoundRank,
					PrevBlock:             b.PrevBlock,
					TxnsMap:               b.TxnsMap,
					ClientState:           b.ClientState,
					stateStatus:           b.stateStatus,
					blockState:            b.blockState,
					isNotarized:           b.isNotarized,
					verificationStatus:    b.verificationStatus,
					RunningTxnCount:       b.RunningTxnCount,
					UniqueBlockExtensions: b.UniqueBlockExtensions,
					MagicBlock:            b.MagicBlock,
				}
			}(),
			wantErr: true,
		},
		{
			name: "Wrong_Signature_ERR",
			fields: func() fields {
				b := NewBlock("", 1)
				b.ChainID = config.ServerChainID
				b.MinerID = n.ID
				b.Hash = b.ComputeHash()

				return fields{
					UnverifiedBlockBody:   b.UnverifiedBlockBody,
					VerificationTickets:   b.VerificationTickets,
					HashIDField:           b.HashIDField,
					Signature:             b.Signature,
					ChainID:               b.ChainID,
					RoundRank:             b.RoundRank,
					PrevBlock:             b.PrevBlock,
					TxnsMap:               b.TxnsMap,
					ClientState:           b.ClientState,
					stateStatus:           b.stateStatus,
					blockState:            b.blockState,
					isNotarized:           b.isNotarized,
					verificationStatus:    b.verificationStatus,
					RunningTxnCount:       b.RunningTxnCount,
					UniqueBlockExtensions: b.UniqueBlockExtensions,
					MagicBlock:            b.MagicBlock,
				}
			}(),
			wantErr: true,
		},
		{
			name: "Invalid_Signature_ERR",
			fields: func() fields {
				b := NewBlock("", 1)
				b.ChainID = config.ServerChainID
				b.MinerID = n.ID
				b.Hash = b.ComputeHash()
				b.Signature = "!"

				return fields{
					UnverifiedBlockBody:   b.UnverifiedBlockBody,
					VerificationTickets:   b.VerificationTickets,
					HashIDField:           b.HashIDField,
					Signature:             b.Signature,
					ChainID:               b.ChainID,
					RoundRank:             b.RoundRank,
					PrevBlock:             b.PrevBlock,
					TxnsMap:               b.TxnsMap,
					ClientState:           b.ClientState,
					stateStatus:           b.stateStatus,
					blockState:            b.blockState,
					isNotarized:           b.isNotarized,
					verificationStatus:    b.verificationStatus,
					RunningTxnCount:       b.RunningTxnCount,
					UniqueBlockExtensions: b.UniqueBlockExtensions,
					MagicBlock:            b.MagicBlock,
				}
			}(),
			wantErr: true,
		},
		{
			name: "OK",
			fields: func() fields {
				b := NewBlock("", 1)
				b.ChainID = config.ServerChainID
				b.MinerID = n.ID
				b.Hash = b.ComputeHash()

				var err error
				if b.Signature, err = encryption.Sign(prK, b.Hash); err != nil {
					t.Fatal(err)
				}

				return fields{
					UnverifiedBlockBody:   b.UnverifiedBlockBody,
					VerificationTickets:   b.VerificationTickets,
					HashIDField:           b.HashIDField,
					Signature:             b.Signature,
					ChainID:               b.ChainID,
					RoundRank:             b.RoundRank,
					PrevBlock:             b.PrevBlock,
					TxnsMap:               b.TxnsMap,
					ClientState:           b.ClientState,
					stateStatus:           b.stateStatus,
					blockState:            b.blockState,
					isNotarized:           b.isNotarized,
					verificationStatus:    b.verificationStatus,
					RunningTxnCount:       b.RunningTxnCount,
					UniqueBlockExtensions: b.UniqueBlockExtensions,
					MagicBlock:            b.MagicBlock,
				}
			}(),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Block{
				UnverifiedBlockBody:   tt.fields.UnverifiedBlockBody,
				VerificationTickets:   tt.fields.VerificationTickets,
				HashIDField:           tt.fields.HashIDField,
				Signature:             tt.fields.Signature,
				ChainID:               tt.fields.ChainID,
				RoundRank:             tt.fields.RoundRank,
				PrevBlock:             tt.fields.PrevBlock,
				TxnsMap:               tt.fields.TxnsMap,
				ClientState:           tt.fields.ClientState,
				stateStatus:           tt.fields.stateStatus,
				blockState:            tt.fields.blockState,
				isNotarized:           tt.fields.isNotarized,
				verificationStatus:    tt.fields.verificationStatus,
				RunningTxnCount:       tt.fields.RunningTxnCount,
				UniqueBlockExtensions: tt.fields.UniqueBlockExtensions,
				MagicBlock:            tt.fields.MagicBlock,
			}
			if err := b.Validate(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBlock_Read(t *testing.T) {
	store := mocks.Store{}
	store.On("Read", context.Context(nil), "", new(Block)).Return(
		func(_ context.Context, _ string, _ datastore.Entity) error {
			return nil
		},
	)

	blockEntityMetadata = &datastore.EntityMetadataImpl{
		Store: &store,
	}

	type fields struct {
		UnverifiedBlockBody   UnverifiedBlockBody
		VerificationTickets   []*VerificationTicket
		HashIDField           datastore.HashIDField
		Signature             string
		ChainID               datastore.Key
		RoundRank             int
		PrevBlock             *Block
		TxnsMap               map[string]bool
		ClientState           util.MerklePatriciaTrieI
		stateStatus           int8
		blockState            int8
		isNotarized           bool
		verificationStatus    int
		RunningTxnCount       int64
		UniqueBlockExtensions map[string]bool
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
			b := &Block{
				UnverifiedBlockBody:   tt.fields.UnverifiedBlockBody,
				VerificationTickets:   tt.fields.VerificationTickets,
				HashIDField:           tt.fields.HashIDField,
				Signature:             tt.fields.Signature,
				ChainID:               tt.fields.ChainID,
				RoundRank:             tt.fields.RoundRank,
				PrevBlock:             tt.fields.PrevBlock,
				TxnsMap:               tt.fields.TxnsMap,
				ClientState:           tt.fields.ClientState,
				stateStatus:           tt.fields.stateStatus,
				blockState:            tt.fields.blockState,
				isNotarized:           tt.fields.isNotarized,
				verificationStatus:    tt.fields.verificationStatus,
				RunningTxnCount:       tt.fields.RunningTxnCount,
				UniqueBlockExtensions: tt.fields.UniqueBlockExtensions,
				MagicBlock:            tt.fields.MagicBlock,
			}
			if err := b.Read(tt.args.ctx, tt.args.key); (err != nil) != tt.wantErr {
				t.Errorf("Read() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBlock_GetScore(t *testing.T) {
	type fields struct {
		UnverifiedBlockBody   UnverifiedBlockBody
		VerificationTickets   []*VerificationTicket
		HashIDField           datastore.HashIDField
		Signature             string
		ChainID               datastore.Key
		RoundRank             int
		PrevBlock             *Block
		TxnsMap               map[string]bool
		ClientState           util.MerklePatriciaTrieI
		stateStatus           int8
		blockState            int8
		isNotarized           bool
		verificationStatus    int
		RunningTxnCount       int64
		UniqueBlockExtensions map[string]bool
		MagicBlock            *MagicBlock
	}
	tests := []struct {
		name   string
		fields fields
		want   int64
	}{
		{
			name:   "OK",
			fields: fields{UnverifiedBlockBody: UnverifiedBlockBody{Round: 1}},
			want:   1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Block{
				UnverifiedBlockBody:   tt.fields.UnverifiedBlockBody,
				VerificationTickets:   tt.fields.VerificationTickets,
				HashIDField:           tt.fields.HashIDField,
				Signature:             tt.fields.Signature,
				ChainID:               tt.fields.ChainID,
				RoundRank:             tt.fields.RoundRank,
				PrevBlock:             tt.fields.PrevBlock,
				TxnsMap:               tt.fields.TxnsMap,
				ClientState:           tt.fields.ClientState,
				stateStatus:           tt.fields.stateStatus,
				blockState:            tt.fields.blockState,
				isNotarized:           tt.fields.isNotarized,
				verificationStatus:    tt.fields.verificationStatus,
				RunningTxnCount:       tt.fields.RunningTxnCount,
				UniqueBlockExtensions: tt.fields.UniqueBlockExtensions,
				MagicBlock:            tt.fields.MagicBlock,
			}
			got, err := b.GetScore()
			require.NoError(t, err)
			if got != tt.want {
				t.Errorf("GetScore() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBlock_Write(t *testing.T) {
	store := mocks.Store{}
	store.On("Write", context.Context(nil), new(Block)).Return(
		func(_ context.Context, _ datastore.Entity) error {
			return nil
		},
	)

	blockEntityMetadata = &datastore.EntityMetadataImpl{
		Store: &store,
	}

	type fields struct {
		UnverifiedBlockBody   UnverifiedBlockBody
		VerificationTickets   []*VerificationTicket
		HashIDField           datastore.HashIDField
		Signature             string
		ChainID               datastore.Key
		RoundRank             int
		PrevBlock             *Block
		TxnsMap               map[string]bool
		ClientState           util.MerklePatriciaTrieI
		stateStatus           int8
		blockState            int8
		isNotarized           bool
		verificationStatus    int
		RunningTxnCount       int64
		UniqueBlockExtensions map[string]bool
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
			b := &Block{
				UnverifiedBlockBody:   tt.fields.UnverifiedBlockBody,
				VerificationTickets:   tt.fields.VerificationTickets,
				HashIDField:           tt.fields.HashIDField,
				Signature:             tt.fields.Signature,
				ChainID:               tt.fields.ChainID,
				RoundRank:             tt.fields.RoundRank,
				PrevBlock:             tt.fields.PrevBlock,
				TxnsMap:               tt.fields.TxnsMap,
				ClientState:           tt.fields.ClientState,
				stateStatus:           tt.fields.stateStatus,
				blockState:            tt.fields.blockState,
				isNotarized:           tt.fields.isNotarized,
				verificationStatus:    tt.fields.verificationStatus,
				RunningTxnCount:       tt.fields.RunningTxnCount,
				UniqueBlockExtensions: tt.fields.UniqueBlockExtensions,
				MagicBlock:            tt.fields.MagicBlock,
			}
			if err := b.Write(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("Write() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBlock_Delete(t *testing.T) {
	store := mocks.Store{}
	store.On("Delete", context.Context(nil), new(Block)).Return(
		func(_ context.Context, _ datastore.Entity) error {
			return nil
		},
	)

	blockEntityMetadata = &datastore.EntityMetadataImpl{
		Store: &store,
	}

	type fields struct {
		UnverifiedBlockBody   UnverifiedBlockBody
		VerificationTickets   []*VerificationTicket
		HashIDField           datastore.HashIDField
		Signature             string
		ChainID               datastore.Key
		RoundRank             int
		PrevBlock             *Block
		TxnsMap               map[string]bool
		ClientState           util.MerklePatriciaTrieI
		stateStatus           int8
		blockState            int8
		isNotarized           bool
		verificationStatus    int
		RunningTxnCount       int64
		UniqueBlockExtensions map[string]bool
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
			b := &Block{
				UnverifiedBlockBody:   tt.fields.UnverifiedBlockBody,
				VerificationTickets:   tt.fields.VerificationTickets,
				HashIDField:           tt.fields.HashIDField,
				Signature:             tt.fields.Signature,
				ChainID:               tt.fields.ChainID,
				RoundRank:             tt.fields.RoundRank,
				PrevBlock:             tt.fields.PrevBlock,
				TxnsMap:               tt.fields.TxnsMap,
				ClientState:           tt.fields.ClientState,
				stateStatus:           tt.fields.stateStatus,
				blockState:            tt.fields.blockState,
				isNotarized:           tt.fields.isNotarized,
				verificationStatus:    tt.fields.verificationStatus,
				RunningTxnCount:       tt.fields.RunningTxnCount,
				UniqueBlockExtensions: tt.fields.UniqueBlockExtensions,
				MagicBlock:            tt.fields.MagicBlock,
			}
			if err := b.Delete(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBlock_SetPreviousBlock(t *testing.T) {
	b := NewBlock("", 2)
	prevB := NewBlock("", 1)

	type fields struct {
		UnverifiedBlockBody   UnverifiedBlockBody
		VerificationTickets   []*VerificationTicket
		HashIDField           datastore.HashIDField
		Signature             string
		ChainID               datastore.Key
		RoundRank             int
		PrevBlock             *Block
		TxnsMap               map[string]bool
		ClientState           util.MerklePatriciaTrieI
		stateStatus           int8
		blockState            int8
		isNotarized           bool
		verificationStatus    int
		RunningTxnCount       int64
		UniqueBlockExtensions map[string]bool
		MagicBlock            *MagicBlock
	}
	type args struct {
		prevBlock *Block
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *Block
	}{
		{
			name: "OK",
			fields: fields{
				UnverifiedBlockBody:   b.UnverifiedBlockBody,
				VerificationTickets:   b.VerificationTickets,
				HashIDField:           b.HashIDField,
				Signature:             b.Signature,
				ChainID:               b.ChainID,
				RoundRank:             b.RoundRank,
				PrevBlock:             b.PrevBlock,
				TxnsMap:               b.TxnsMap,
				ClientState:           b.ClientState,
				stateStatus:           b.stateStatus,
				blockState:            b.blockState,
				isNotarized:           b.isNotarized,
				verificationStatus:    b.verificationStatus,
				RunningTxnCount:       b.RunningTxnCount,
				UniqueBlockExtensions: b.UniqueBlockExtensions,
				MagicBlock:            b.MagicBlock,
			},
			args: args{prevBlock: prevB},
			want: func() *Block {
				b := NewBlock("", 2)
				b.PrevBlock = copyBlock(prevB)
				b.PrevHash = prevB.Hash
				b.Round = prevB.Round + 1
				if len(b.PrevBlockVerificationTickets) == 0 {
					b.PrevBlockVerificationTickets = prevB.GetVerificationTickets()
				}
				return b
			}(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Block{
				UnverifiedBlockBody:   tt.fields.UnverifiedBlockBody,
				VerificationTickets:   tt.fields.VerificationTickets,
				HashIDField:           tt.fields.HashIDField,
				Signature:             tt.fields.Signature,
				ChainID:               tt.fields.ChainID,
				RoundRank:             tt.fields.RoundRank,
				PrevBlock:             tt.fields.PrevBlock,
				TxnsMap:               tt.fields.TxnsMap,
				ClientState:           tt.fields.ClientState,
				stateStatus:           tt.fields.stateStatus,
				blockState:            tt.fields.blockState,
				isNotarized:           tt.fields.isNotarized,
				verificationStatus:    tt.fields.verificationStatus,
				RunningTxnCount:       tt.fields.RunningTxnCount,
				UniqueBlockExtensions: tt.fields.UniqueBlockExtensions,
				MagicBlock:            tt.fields.MagicBlock,
			}

			b.SetPreviousBlock(tt.args.prevBlock)

			if !assert.Equal(t, tt.want, b) {
				t.Errorf("SetPreviousBlock() got = %v, want = %v", b, tt.want)
			}
		})
	}
}

func TestBlock_SetStateDB_Debug_True(t *testing.T) {
	state.SetDebugLevel(1)

	type fields struct {
		UnverifiedBlockBody   UnverifiedBlockBody
		VerificationTickets   []*VerificationTicket
		HashIDField           datastore.HashIDField
		Signature             string
		ChainID               datastore.Key
		RoundRank             int
		PrevBlock             *Block
		TxnsMap               map[string]bool
		ClientState           util.MerklePatriciaTrieI
		stateStatus           int8
		blockState            int8
		isNotarized           bool
		verificationStatus    int
		RunningTxnCount       int64
		UniqueBlockExtensions map[string]bool
		MagicBlock            *MagicBlock
	}
	type args struct {
		prevBlock *Block
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		want      *Block
		wantPanic bool
	}{
		{
			name:      "Debug_PANIC",
			wantPanic: true,
		},
		// duplicating tests to expose race errors
		{
			name:      "Debug_PANIC",
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
					t.Errorf("SetStateDB() want panic  = %v, but got = %v", tt.wantPanic, got)
				}
			}()

			b := &Block{
				UnverifiedBlockBody:   tt.fields.UnverifiedBlockBody,
				VerificationTickets:   tt.fields.VerificationTickets,
				HashIDField:           tt.fields.HashIDField,
				Signature:             tt.fields.Signature,
				ChainID:               tt.fields.ChainID,
				RoundRank:             tt.fields.RoundRank,
				PrevBlock:             tt.fields.PrevBlock,
				TxnsMap:               tt.fields.TxnsMap,
				ClientState:           tt.fields.ClientState,
				stateStatus:           tt.fields.stateStatus,
				blockState:            tt.fields.blockState,
				isNotarized:           tt.fields.isNotarized,
				verificationStatus:    tt.fields.verificationStatus,
				RunningTxnCount:       tt.fields.RunningTxnCount,
				UniqueBlockExtensions: tt.fields.UniqueBlockExtensions,
				MagicBlock:            tt.fields.MagicBlock,
			}
			b.SetStateDB(tt.args.prevBlock, util.NewMemoryNodeDB())

			b.ClientState = nil
			tt.want.ClientState = nil
			if !assert.Equal(t, tt.want, b) {
				assert.Equal(t, tt.want.ClientState, b.ClientState)
				t.Errorf("SetStateDB() got = %v, want = %v", b, tt.want)
			}
		})
	}
}

func TestBlock_SetStateDB_Debug_False(t *testing.T) {
	state.SetDebugLevel(0)

	b := NewBlock("", 1)
	prevB := NewBlock("", 0)
	cs := util.NewMerklePatriciaTrie(util.NewMemoryNodeDB(), util.Sequence(b.Round), nil)

	type fields struct {
		UnverifiedBlockBody   UnverifiedBlockBody
		VerificationTickets   []*VerificationTicket
		HashIDField           datastore.HashIDField
		Signature             string
		ChainID               datastore.Key
		RoundRank             int
		PrevBlock             *Block
		TxnsMap               map[string]bool
		ClientState           util.MerklePatriciaTrieI
		stateStatus           int8
		stateStatusMutex      *sync.RWMutex
		StateMutex            *sync.RWMutex
		blockState            int8
		isNotarized           bool
		verificationStatus    int
		RunningTxnCount       int64
		UniqueBlockExtensions map[string]bool
		MagicBlock            *MagicBlock
	}
	type args struct {
		prevBlock *Block
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		want      *Block
		wantPanic bool
	}{
		{
			name: "OK",
			fields: fields{
				UnverifiedBlockBody:   b.UnverifiedBlockBody,
				VerificationTickets:   b.VerificationTickets,
				HashIDField:           b.HashIDField,
				Signature:             b.Signature,
				ChainID:               b.ChainID,
				RoundRank:             b.RoundRank,
				PrevBlock:             b.PrevBlock,
				TxnsMap:               b.TxnsMap,
				ClientState:           b.ClientState,
				stateStatus:           b.stateStatus,
				blockState:            b.blockState,
				isNotarized:           b.isNotarized,
				verificationStatus:    b.verificationStatus,
				RunningTxnCount:       b.RunningTxnCount,
				UniqueBlockExtensions: b.UniqueBlockExtensions,
				MagicBlock:            b.MagicBlock,
			},
			args: args{prevBlock: prevB},
			want: func() *Block {
				b := NewBlock("", 1)
				pndb := util.NewMemoryNodeDB()
				rootHash := prevB.ClientStateHash
				b.CreateState(pndb, rootHash)

				return b
			}(),
		},
		{
			name: "Non_Nil_Client_State",
			fields: fields{
				UnverifiedBlockBody:   b.UnverifiedBlockBody,
				VerificationTickets:   b.VerificationTickets,
				HashIDField:           b.HashIDField,
				Signature:             b.Signature,
				ChainID:               b.ChainID,
				RoundRank:             b.RoundRank,
				PrevBlock:             b.PrevBlock,
				TxnsMap:               b.TxnsMap,
				ClientState:           b.ClientState,
				stateStatus:           b.stateStatus,
				blockState:            b.blockState,
				isNotarized:           b.isNotarized,
				verificationStatus:    b.verificationStatus,
				RunningTxnCount:       b.RunningTxnCount,
				UniqueBlockExtensions: b.UniqueBlockExtensions,
				MagicBlock:            b.MagicBlock,
			},
			args: args{
				prevBlock: func() *Block {
					prevB := NewBlock("", 0)
					prevB.ClientState = cs
					return prevB
				}(),
			},

			want: func() *Block {
				b := NewBlock("", 1)
				pndb := cs.GetNodeDB()
				rootHash := prevB.ClientStateHash
				b.CreateState(pndb, rootHash)

				return b
			}(),
		},
		// duplicating tests to expose race errors
		{
			name: "OK",
			fields: fields{
				UnverifiedBlockBody:   b.UnverifiedBlockBody,
				VerificationTickets:   b.VerificationTickets,
				HashIDField:           b.HashIDField,
				Signature:             b.Signature,
				ChainID:               b.ChainID,
				RoundRank:             b.RoundRank,
				PrevBlock:             b.PrevBlock,
				TxnsMap:               b.TxnsMap,
				ClientState:           b.ClientState,
				stateStatus:           b.stateStatus,
				blockState:            b.blockState,
				isNotarized:           b.isNotarized,
				verificationStatus:    b.verificationStatus,
				RunningTxnCount:       b.RunningTxnCount,
				UniqueBlockExtensions: b.UniqueBlockExtensions,
				MagicBlock:            b.MagicBlock,
			},
			args: args{prevBlock: prevB},
			want: func() *Block {
				b := NewBlock("", 1)
				pndb := util.NewMemoryNodeDB()
				rootHash := prevB.ClientStateHash
				b.CreateState(pndb, rootHash)

				return b
			}(),
		},
		{
			name: "Non_Nil_Client_State",
			fields: fields{
				UnverifiedBlockBody:   b.UnverifiedBlockBody,
				VerificationTickets:   b.VerificationTickets,
				HashIDField:           b.HashIDField,
				Signature:             b.Signature,
				ChainID:               b.ChainID,
				RoundRank:             b.RoundRank,
				PrevBlock:             b.PrevBlock,
				TxnsMap:               b.TxnsMap,
				ClientState:           b.ClientState,
				stateStatus:           b.stateStatus,
				blockState:            b.blockState,
				isNotarized:           b.isNotarized,
				verificationStatus:    b.verificationStatus,
				RunningTxnCount:       b.RunningTxnCount,
				UniqueBlockExtensions: b.UniqueBlockExtensions,
				MagicBlock:            b.MagicBlock,
			},
			args: args{
				prevBlock: func() *Block {
					prevB := NewBlock("", 0)
					prevB.ClientState = cs
					return prevB
				}(),
			},

			want: func() *Block {
				b := NewBlock("", 1)
				pndb := cs.GetNodeDB()
				rootHash := prevB.ClientStateHash
				b.CreateState(pndb, rootHash)

				return b
			}(),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			defer func() {
				got := recover()
				if (got != nil) != tt.wantPanic {
					t.Errorf("SetStateDB() want panic  = %v, but got = %v", tt.wantPanic, got)
				}
			}()

			b := &Block{
				UnverifiedBlockBody:   tt.fields.UnverifiedBlockBody,
				VerificationTickets:   tt.fields.VerificationTickets,
				HashIDField:           tt.fields.HashIDField,
				Signature:             tt.fields.Signature,
				ChainID:               tt.fields.ChainID,
				RoundRank:             tt.fields.RoundRank,
				PrevBlock:             tt.fields.PrevBlock,
				TxnsMap:               tt.fields.TxnsMap,
				ClientState:           tt.fields.ClientState,
				stateStatus:           tt.fields.stateStatus,
				blockState:            tt.fields.blockState,
				isNotarized:           tt.fields.isNotarized,
				verificationStatus:    tt.fields.verificationStatus,
				RunningTxnCount:       tt.fields.RunningTxnCount,
				UniqueBlockExtensions: tt.fields.UniqueBlockExtensions,
				MagicBlock:            tt.fields.MagicBlock,
			}
			b.SetStateDB(tt.args.prevBlock, util.NewMemoryNodeDB())

			b.ClientState = nil
			tt.want.ClientState = nil

			if !assert.Equal(t, tt.want, b) {
				assert.Equal(t, tt.want.ClientState, b.ClientState)
				t.Errorf("SetStateDB() got = %v, want = %v", b, tt.want)
			}
		})
	}
}

func TestBlock_InitStateDB(t *testing.T) {
	key := util.Key("key")
	n := util.NewValueNode()

	type fields struct {
		UnverifiedBlockBody   UnverifiedBlockBody
		VerificationTickets   []*VerificationTicket
		HashIDField           datastore.HashIDField
		Signature             string
		ChainID               datastore.Key
		RoundRank             int
		PrevBlock             *Block
		TxnsMap               map[string]bool
		ClientState           util.MerklePatriciaTrieI
		stateStatus           int8
		blockState            int8
		isNotarized           bool
		verificationStatus    int
		RunningTxnCount       int64
		UniqueBlockExtensions map[string]bool
		MagicBlock            *MagicBlock
	}
	type args struct {
		ndb util.NodeDB
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "OK",
			fields: fields{
				UnverifiedBlockBody: UnverifiedBlockBody{ClientStateHash: key},
			},
			args: args{
				ndb: func() util.NodeDB {
					db := util.NewMemoryNodeDB()
					if err := db.PutNode(key, n); err != nil {
						t.Fatal(err)
					}

					return db
				}(),
			},
			wantErr: false,
		},
		{
			name:    "ERR",
			args:    args{ndb: util.NewMemoryNodeDB()},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Block{
				UnverifiedBlockBody:   tt.fields.UnverifiedBlockBody,
				VerificationTickets:   tt.fields.VerificationTickets,
				HashIDField:           tt.fields.HashIDField,
				Signature:             tt.fields.Signature,
				ChainID:               tt.fields.ChainID,
				RoundRank:             tt.fields.RoundRank,
				PrevBlock:             tt.fields.PrevBlock,
				TxnsMap:               tt.fields.TxnsMap,
				ClientState:           tt.fields.ClientState,
				stateStatus:           tt.fields.stateStatus,
				blockState:            tt.fields.blockState,
				isNotarized:           tt.fields.isNotarized,
				verificationStatus:    tt.fields.verificationStatus,
				RunningTxnCount:       tt.fields.RunningTxnCount,
				UniqueBlockExtensions: tt.fields.UniqueBlockExtensions,
				MagicBlock:            tt.fields.MagicBlock,
			}
			if err := b.InitStateDB(tt.args.ndb); (err != nil) != tt.wantErr {
				t.Errorf("InitStateDB() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBlock_AddVerificationTicket(t *testing.T) {
	verID := "id"
	b := NewBlock("", 1)
	b.VerificationTickets = []*VerificationTicket{
		{
			VerifierID: verID,
		},
		{
			VerifierID: verID,
		},
	}

	type fields struct {
		UnverifiedBlockBody   UnverifiedBlockBody
		VerificationTickets   []*VerificationTicket
		HashIDField           datastore.HashIDField
		Signature             string
		ChainID               datastore.Key
		RoundRank             int
		PrevBlock             *Block
		TxnsMap               map[string]bool
		ClientState           util.MerklePatriciaTrieI
		stateStatus           int8
		blockState            int8
		isNotarized           bool
		verificationStatus    int
		RunningTxnCount       int64
		UniqueBlockExtensions map[string]bool
		MagicBlock            *MagicBlock
	}
	type args struct {
		vt *VerificationTicket
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "TRUE",
			fields: fields{
				VerificationTickets: b.VerificationTickets,
			},
			args: args{vt: &VerificationTicket{VerifierID: "unknown id"}},
			want: true,
		},
		{
			name: "FALSE",
			fields: fields{
				VerificationTickets: b.VerificationTickets,
			},
			args: args{vt: &VerificationTicket{VerifierID: verID}},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Block{
				UnverifiedBlockBody:   tt.fields.UnverifiedBlockBody,
				VerificationTickets:   tt.fields.VerificationTickets,
				HashIDField:           tt.fields.HashIDField,
				Signature:             tt.fields.Signature,
				ChainID:               tt.fields.ChainID,
				RoundRank:             tt.fields.RoundRank,
				PrevBlock:             tt.fields.PrevBlock,
				TxnsMap:               tt.fields.TxnsMap,
				ClientState:           tt.fields.ClientState,
				stateStatus:           tt.fields.stateStatus,
				blockState:            tt.fields.blockState,
				isNotarized:           tt.fields.isNotarized,
				verificationStatus:    tt.fields.verificationStatus,
				RunningTxnCount:       tt.fields.RunningTxnCount,
				UniqueBlockExtensions: tt.fields.UniqueBlockExtensions,
				MagicBlock:            tt.fields.MagicBlock,
			}
			if got := b.AddVerificationTicket(tt.args.vt); got != tt.want {
				t.Errorf("AddVerificationTickets() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBlock_MergeVerificationTickets(t *testing.T) {
	verID := "id"
	tickets := []*VerificationTicket{
		{
			VerifierID: verID,
		},
		{
			VerifierID: verID,
		},
	}

	type fields struct {
		UnverifiedBlockBody   UnverifiedBlockBody
		VerificationTickets   []*VerificationTicket
		HashIDField           datastore.HashIDField
		Signature             string
		ChainID               datastore.Key
		RoundRank             int
		PrevBlock             *Block
		TxnsMap               map[string]bool
		ClientState           util.MerklePatriciaTrieI
		stateStatus           int8
		blockState            int8
		isNotarized           bool
		verificationStatus    int
		RunningTxnCount       int64
		UniqueBlockExtensions map[string]bool
		MagicBlock            *MagicBlock
	}
	type args struct {
		vts []*VerificationTicket
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []*VerificationTicket
	}{
		{
			name:   "Received_OK",
			fields: fields{},
			want:   []*VerificationTicket(nil),
		},
		{
			name: "Already_Have_OK",
			fields: fields{
				VerificationTickets: tickets,
			},
			want: tickets,
		},
		{
			name: "Nil_Ticket_OK",
			fields: fields{
				VerificationTickets: tickets,
			},
			args: args{vts: make([]*VerificationTicket, 2)},
			want: tickets,
		},
		{
			name: "Not_Nil_Tickets_But_Duplicate_OK",
			fields: fields{
				VerificationTickets: tickets,
			},
			args: args{
				vts: []*VerificationTicket{
					{
						VerifierID: "another id",
					},
					{
						VerifierID: verID,
					},
				},
			},
			want: append(tickets, &VerificationTicket{VerifierID: "another id"}),
		},
		{

			name: "OK",
			fields: fields{
				VerificationTickets: tickets,
			},
			args: args{
				vts: []*VerificationTicket{
					{
						VerifierID: verID,
					},
				},
			},
			want: tickets,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Block{
				UnverifiedBlockBody:   tt.fields.UnverifiedBlockBody,
				VerificationTickets:   tt.fields.VerificationTickets,
				HashIDField:           tt.fields.HashIDField,
				Signature:             tt.fields.Signature,
				ChainID:               tt.fields.ChainID,
				RoundRank:             tt.fields.RoundRank,
				PrevBlock:             tt.fields.PrevBlock,
				TxnsMap:               tt.fields.TxnsMap,
				ClientState:           tt.fields.ClientState,
				stateStatus:           tt.fields.stateStatus,
				blockState:            tt.fields.blockState,
				isNotarized:           tt.fields.isNotarized,
				verificationStatus:    tt.fields.verificationStatus,
				RunningTxnCount:       tt.fields.RunningTxnCount,
				UniqueBlockExtensions: tt.fields.UniqueBlockExtensions,
				MagicBlock:            tt.fields.MagicBlock,
			}

			b.MergeVerificationTickets(tt.args.vts)
			if !reflect.DeepEqual(b.VerificationTickets, tt.want) {
				t.Errorf("MergeverificationTickets() got = %#v, want = %#v", b.VerificationTickets, tt.want)
			}
		})
	}
}

func TestBlock_GetMerkleTree(t *testing.T) {
	b := NewBlock("", 1)
	hashables := make([]util.Hashable, 0, 3)
	for i := 0; i < 3; i++ {
		txn := transaction.Transaction{OutputHash: encryption.Hash("data" + strconv.Itoa(i))}
		b.Txns = append(b.Txns, &txn)
		hashables = append(hashables, &txn)
	}

	var mt util.MerkleTree
	mt.ComputeTree(hashables)

	type fields struct {
		UnverifiedBlockBody   UnverifiedBlockBody
		VerificationTickets   []*VerificationTicket
		HashIDField           datastore.HashIDField
		Signature             string
		ChainID               datastore.Key
		RoundRank             int
		PrevBlock             *Block
		TxnsMap               map[string]bool
		ClientState           util.MerklePatriciaTrieI
		stateStatus           int8
		blockState            int8
		isNotarized           bool
		verificationStatus    int
		RunningTxnCount       int64
		UniqueBlockExtensions map[string]bool
		MagicBlock            *MagicBlock
	}
	tests := []struct {
		name   string
		fields fields
		want   *util.MerkleTree
	}{
		{
			name: "OK",
			fields: fields{
				UnverifiedBlockBody: b.UnverifiedBlockBody,
			},
			want: &mt,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Block{
				UnverifiedBlockBody:   tt.fields.UnverifiedBlockBody,
				VerificationTickets:   tt.fields.VerificationTickets,
				HashIDField:           tt.fields.HashIDField,
				Signature:             tt.fields.Signature,
				ChainID:               tt.fields.ChainID,
				RoundRank:             tt.fields.RoundRank,
				PrevBlock:             tt.fields.PrevBlock,
				TxnsMap:               tt.fields.TxnsMap,
				ClientState:           tt.fields.ClientState,
				stateStatus:           tt.fields.stateStatus,
				blockState:            tt.fields.blockState,
				isNotarized:           tt.fields.isNotarized,
				verificationStatus:    tt.fields.verificationStatus,
				RunningTxnCount:       tt.fields.RunningTxnCount,
				UniqueBlockExtensions: tt.fields.UniqueBlockExtensions,
				MagicBlock:            tt.fields.MagicBlock,
			}
			if got := b.GetMerkleTree(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetMerkleTree() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBlock_HashBlock(t *testing.T) {
	b := NewBlock("", 1)
	for i := 0; i < 3; i++ {
		txn := transaction.Transaction{OutputHash: encryption.Hash("data" + strconv.Itoa(i))}
		b.Txns = append(b.Txns, &txn)
	}
	b.MinerID = "miner id"
	b.PrevHash = "prev hash"
	b.StateChangesCount = 10

	mt := b.GetMerkleTree()
	merkleRoot := mt.GetRoot()
	rmt := b.GetReceiptsMerkleTree()
	rMerkleRoot := rmt.GetRoot()
	hashData := b.MinerID + ":" + b.PrevHash + ":" + common.TimeToString(b.CreationDate) + ":" +
		strconv.FormatInt(b.Round, 10) + ":" + strconv.FormatInt(b.GetRoundRandomSeed(), 10) + ":" +
		strconv.Itoa(b.StateChangesCount) + ":" + merkleRoot + ":" + rMerkleRoot
	hash := encryption.Hash(hashData)

	type fields struct {
		UnverifiedBlockBody   UnverifiedBlockBody
		VerificationTickets   []*VerificationTicket
		HashIDField           datastore.HashIDField
		Signature             string
		ChainID               datastore.Key
		RoundRank             int
		PrevBlock             *Block
		TxnsMap               map[string]bool
		ClientState           util.MerklePatriciaTrieI
		stateStatus           int8
		blockState            int8
		isNotarized           bool
		verificationStatus    int
		RunningTxnCount       int64
		UniqueBlockExtensions map[string]bool
		StateChangesCount     int
		MagicBlock            *MagicBlock
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "OK",
			fields: fields{
				UnverifiedBlockBody:   b.UnverifiedBlockBody,
				VerificationTickets:   b.VerificationTickets,
				HashIDField:           b.HashIDField,
				Signature:             b.Signature,
				ChainID:               b.ChainID,
				RoundRank:             b.RoundRank,
				PrevBlock:             b.PrevBlock,
				TxnsMap:               b.TxnsMap,
				ClientState:           b.ClientState,
				stateStatus:           b.stateStatus,
				blockState:            b.blockState,
				isNotarized:           b.isNotarized,
				verificationStatus:    b.verificationStatus,
				RunningTxnCount:       b.RunningTxnCount,
				UniqueBlockExtensions: b.UniqueBlockExtensions,
				StateChangesCount:     b.StateChangesCount,
				MagicBlock:            b.MagicBlock,
			},
			want: hash,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Block{
				UnverifiedBlockBody:   tt.fields.UnverifiedBlockBody,
				VerificationTickets:   tt.fields.VerificationTickets,
				HashIDField:           tt.fields.HashIDField,
				Signature:             tt.fields.Signature,
				ChainID:               tt.fields.ChainID,
				RoundRank:             tt.fields.RoundRank,
				PrevBlock:             tt.fields.PrevBlock,
				TxnsMap:               tt.fields.TxnsMap,
				ClientState:           tt.fields.ClientState,
				stateStatus:           tt.fields.stateStatus,
				blockState:            tt.fields.blockState,
				isNotarized:           tt.fields.isNotarized,
				verificationStatus:    tt.fields.verificationStatus,
				RunningTxnCount:       tt.fields.RunningTxnCount,
				UniqueBlockExtensions: tt.fields.UniqueBlockExtensions,
				StateChangesCount:     tt.fields.StateChangesCount,
				MagicBlock:            tt.fields.MagicBlock,
			}
			b.HashBlock()
			require.Equal(t, tt.want, b.Hash)
		})
	}
}

func TestBlock_ComputeTxnMap(t *testing.T) {
	b := NewBlock("", 1)
	for i := 0; i < 3; i++ {
		txn := transaction.Transaction{OutputHash: encryption.Hash("data" + strconv.Itoa(i))}
		b.Txns = append(b.Txns, &txn)
	}

	type fields struct {
		UnverifiedBlockBody   UnverifiedBlockBody
		VerificationTickets   []*VerificationTicket
		HashIDField           datastore.HashIDField
		Signature             string
		ChainID               datastore.Key
		RoundRank             int
		PrevBlock             *Block
		TxnsMap               map[string]bool
		ClientState           util.MerklePatriciaTrieI
		stateStatus           int8
		blockState            int8
		isNotarized           bool
		verificationStatus    int
		RunningTxnCount       int64
		UniqueBlockExtensions map[string]bool
		MagicBlock            *MagicBlock
	}
	tests := []struct {
		name   string
		fields fields
		want   map[string]bool
	}{
		{
			name: "OK",
			fields: fields{
				UnverifiedBlockBody:   b.UnverifiedBlockBody,
				VerificationTickets:   b.VerificationTickets,
				HashIDField:           b.HashIDField,
				Signature:             b.Signature,
				ChainID:               b.ChainID,
				RoundRank:             b.RoundRank,
				PrevBlock:             b.PrevBlock,
				TxnsMap:               b.TxnsMap,
				ClientState:           b.ClientState,
				stateStatus:           b.stateStatus,
				blockState:            b.blockState,
				isNotarized:           b.isNotarized,
				verificationStatus:    b.verificationStatus,
				RunningTxnCount:       b.RunningTxnCount,
				UniqueBlockExtensions: b.UniqueBlockExtensions,
				MagicBlock:            b.MagicBlock,
			},
			want: func() map[string]bool {
				tm := make(map[string]bool, len(b.Txns))
				for _, txn := range b.Txns {
					tm[txn.Hash] = true
				}

				return tm
			}(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Block{
				UnverifiedBlockBody:   tt.fields.UnverifiedBlockBody,
				VerificationTickets:   tt.fields.VerificationTickets,
				HashIDField:           tt.fields.HashIDField,
				Signature:             tt.fields.Signature,
				ChainID:               tt.fields.ChainID,
				RoundRank:             tt.fields.RoundRank,
				PrevBlock:             tt.fields.PrevBlock,
				TxnsMap:               tt.fields.TxnsMap,
				ClientState:           tt.fields.ClientState,
				stateStatus:           tt.fields.stateStatus,
				blockState:            tt.fields.blockState,
				isNotarized:           tt.fields.isNotarized,
				verificationStatus:    tt.fields.verificationStatus,
				RunningTxnCount:       tt.fields.RunningTxnCount,
				UniqueBlockExtensions: tt.fields.UniqueBlockExtensions,
				MagicBlock:            tt.fields.MagicBlock,
			}

			b.ComputeTxnMap()
			if !reflect.DeepEqual(b.TxnsMap, tt.want) {
				t.Errorf("ComputeTxnMap() = %v, want %v", b.TxnsMap, tt.want)
			}
		})
	}
}

func TestBlock_HasTransaction(t *testing.T) {
	b := NewBlock("", 1)
	txn := transaction.Transaction{HashIDField: datastore.HashIDField{Hash: encryption.Hash("data")}}
	b.Txns = append(b.Txns, &txn)
	b.AddTransaction(&txn)
	b.TxnsMap = make(map[string]bool)
	b.TxnsMap[txn.Hash] = true

	type fields struct {
		UnverifiedBlockBody   UnverifiedBlockBody
		VerificationTickets   []*VerificationTicket
		HashIDField           datastore.HashIDField
		Signature             string
		ChainID               datastore.Key
		RoundRank             int
		PrevBlock             *Block
		TxnsMap               map[string]bool
		ClientState           util.MerklePatriciaTrieI
		stateStatus           int8
		blockState            int8
		isNotarized           bool
		verificationStatus    int
		RunningTxnCount       int64
		UniqueBlockExtensions map[string]bool
		MagicBlock            *MagicBlock
	}
	type args struct {
		hash string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "TRUE",
			fields: fields{
				UnverifiedBlockBody:   b.UnverifiedBlockBody,
				VerificationTickets:   b.VerificationTickets,
				HashIDField:           b.HashIDField,
				Signature:             b.Signature,
				ChainID:               b.ChainID,
				RoundRank:             b.RoundRank,
				PrevBlock:             b.PrevBlock,
				TxnsMap:               b.TxnsMap,
				ClientState:           b.ClientState,
				stateStatus:           b.stateStatus,
				blockState:            b.blockState,
				isNotarized:           b.isNotarized,
				verificationStatus:    b.verificationStatus,
				RunningTxnCount:       b.RunningTxnCount,
				UniqueBlockExtensions: b.UniqueBlockExtensions,
				MagicBlock:            b.MagicBlock,
			},
			args: args{hash: b.Txns[0].Hash},
			want: true,
		},
		{
			name: "FALSE",
			fields: fields{
				UnverifiedBlockBody:   b.UnverifiedBlockBody,
				VerificationTickets:   b.VerificationTickets,
				HashIDField:           b.HashIDField,
				Signature:             b.Signature,
				ChainID:               b.ChainID,
				RoundRank:             b.RoundRank,
				PrevBlock:             b.PrevBlock,
				TxnsMap:               b.TxnsMap,
				ClientState:           b.ClientState,
				stateStatus:           b.stateStatus,
				blockState:            b.blockState,
				isNotarized:           b.isNotarized,
				verificationStatus:    b.verificationStatus,
				RunningTxnCount:       b.RunningTxnCount,
				UniqueBlockExtensions: b.UniqueBlockExtensions,
				MagicBlock:            b.MagicBlock,
			},
			args: args{hash: "unknown hash"},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Block{
				UnverifiedBlockBody:   tt.fields.UnverifiedBlockBody,
				VerificationTickets:   tt.fields.VerificationTickets,
				HashIDField:           tt.fields.HashIDField,
				Signature:             tt.fields.Signature,
				ChainID:               tt.fields.ChainID,
				RoundRank:             tt.fields.RoundRank,
				PrevBlock:             tt.fields.PrevBlock,
				TxnsMap:               tt.fields.TxnsMap,
				ClientState:           tt.fields.ClientState,
				stateStatus:           tt.fields.stateStatus,
				blockState:            tt.fields.blockState,
				isNotarized:           tt.fields.isNotarized,
				verificationStatus:    tt.fields.verificationStatus,
				RunningTxnCount:       tt.fields.RunningTxnCount,
				UniqueBlockExtensions: tt.fields.UniqueBlockExtensions,
				MagicBlock:            tt.fields.MagicBlock,
			}
			if got := b.HasTransaction(tt.args.hash); got != tt.want {
				t.Errorf("HasTransaction() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBlock_GetSummary(t *testing.T) {
	b := NewBlock("", 1)
	b.Version = "1"
	b.Hash = b.ComputeHash()
	b.MinerID = "miner id"
	b.RoundRandomSeed = b.GetRoundRandomSeed()
	for i := 0; i < 3; i++ {
		txn := transaction.Transaction{OutputHash: encryption.Hash("data" + strconv.Itoa(i))}
		b.Txns = append(b.Txns, &txn)
	}
	b.ClientStateHash = util.Key("client state hash")
	b.MagicBlock = NewMagicBlock()

	bs := datastore.GetEntityMetadata("block_summary").Instance().(*BlockSummary)
	bs.Version = b.Version
	bs.Hash = b.Hash
	bs.MinerID = b.MinerID
	bs.Round = b.Round
	bs.RoundRandomSeed = b.GetRoundRandomSeed()
	bs.CreationDate = b.CreationDate
	bs.MerkleTreeRoot = b.GetMerkleTree().GetRoot()
	bs.ClientStateHash = b.ClientStateHash
	bs.ReceiptMerkleTreeRoot = b.GetReceiptsMerkleTree().GetRoot()
	bs.NumTxns = len(b.Txns)
	bs.MagicBlock = b.MagicBlock

	type fields struct {
		UnverifiedBlockBody   UnverifiedBlockBody
		VerificationTickets   []*VerificationTicket
		HashIDField           datastore.HashIDField
		Signature             string
		ChainID               datastore.Key
		RoundRank             int
		PrevBlock             *Block
		TxnsMap               map[string]bool
		ClientState           util.MerklePatriciaTrieI
		stateStatus           int8
		blockState            int8
		isNotarized           bool
		verificationStatus    int
		RunningTxnCount       int64
		UniqueBlockExtensions map[string]bool
		MagicBlock            *MagicBlock
	}
	tests := []struct {
		name   string
		fields fields
		want   *BlockSummary
	}{
		{
			name: "OK",
			fields: fields{
				UnverifiedBlockBody:   b.UnverifiedBlockBody,
				VerificationTickets:   b.VerificationTickets,
				HashIDField:           b.HashIDField,
				Signature:             b.Signature,
				ChainID:               b.ChainID,
				RoundRank:             b.RoundRank,
				PrevBlock:             b.PrevBlock,
				TxnsMap:               b.TxnsMap,
				ClientState:           b.ClientState,
				stateStatus:           b.stateStatus,
				blockState:            b.blockState,
				isNotarized:           b.isNotarized,
				verificationStatus:    b.verificationStatus,
				RunningTxnCount:       b.RunningTxnCount,
				UniqueBlockExtensions: b.UniqueBlockExtensions,
				MagicBlock:            b.MagicBlock,
			},
			want: bs,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Block{
				UnverifiedBlockBody:   tt.fields.UnverifiedBlockBody,
				VerificationTickets:   tt.fields.VerificationTickets,
				HashIDField:           tt.fields.HashIDField,
				Signature:             tt.fields.Signature,
				ChainID:               tt.fields.ChainID,
				RoundRank:             tt.fields.RoundRank,
				PrevBlock:             tt.fields.PrevBlock,
				TxnsMap:               tt.fields.TxnsMap,
				ClientState:           tt.fields.ClientState,
				stateStatus:           tt.fields.stateStatus,
				blockState:            tt.fields.blockState,
				isNotarized:           tt.fields.isNotarized,
				verificationStatus:    tt.fields.verificationStatus,
				RunningTxnCount:       tt.fields.RunningTxnCount,
				UniqueBlockExtensions: tt.fields.UniqueBlockExtensions,
				MagicBlock:            tt.fields.MagicBlock,
			}
			if got := b.GetSummary(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetSummary() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBlock_Weight(t *testing.T) {
	type fields struct {
		UnverifiedBlockBody   UnverifiedBlockBody
		VerificationTickets   []*VerificationTicket
		HashIDField           datastore.HashIDField
		Signature             string
		ChainID               datastore.Key
		RoundRank             int
		PrevBlock             *Block
		TxnsMap               map[string]bool
		ClientState           util.MerklePatriciaTrieI
		stateStatus           int8
		blockState            int8
		isNotarized           bool
		verificationStatus    int
		RunningTxnCount       int64
		UniqueBlockExtensions map[string]bool
		MagicBlock            *MagicBlock
	}
	tests := []struct {
		name   string
		fields fields
		want   float64
	}{
		{
			name:   "OK",
			fields: fields{RoundRank: 1},
			want:   0.5,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Block{
				UnverifiedBlockBody:   tt.fields.UnverifiedBlockBody,
				VerificationTickets:   tt.fields.VerificationTickets,
				HashIDField:           tt.fields.HashIDField,
				Signature:             tt.fields.Signature,
				ChainID:               tt.fields.ChainID,
				RoundRank:             tt.fields.RoundRank,
				PrevBlock:             tt.fields.PrevBlock,
				TxnsMap:               tt.fields.TxnsMap,
				ClientState:           tt.fields.ClientState,
				stateStatus:           tt.fields.stateStatus,
				blockState:            tt.fields.blockState,
				isNotarized:           tt.fields.isNotarized,
				verificationStatus:    tt.fields.verificationStatus,
				RunningTxnCount:       tt.fields.RunningTxnCount,
				UniqueBlockExtensions: tt.fields.UniqueBlockExtensions,
				MagicBlock:            tt.fields.MagicBlock,
			}
			if got := b.Weight(); got != tt.want {
				t.Errorf("Weight() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBlock_GetBlockState(t *testing.T) {
	type fields struct {
		UnverifiedBlockBody   UnverifiedBlockBody
		VerificationTickets   []*VerificationTicket
		HashIDField           datastore.HashIDField
		Signature             string
		ChainID               datastore.Key
		RoundRank             int
		PrevBlock             *Block
		TxnsMap               map[string]bool
		ClientState           util.MerklePatriciaTrieI
		stateStatus           int8
		blockState            int8
		isNotarized           bool
		verificationStatus    int
		RunningTxnCount       int64
		UniqueBlockExtensions map[string]bool
		MagicBlock            *MagicBlock
	}
	tests := []struct {
		name   string
		fields fields
		want   int8
	}{
		{
			name: "OK",
			want: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Block{
				UnverifiedBlockBody:   tt.fields.UnverifiedBlockBody,
				VerificationTickets:   tt.fields.VerificationTickets,
				HashIDField:           tt.fields.HashIDField,
				Signature:             tt.fields.Signature,
				ChainID:               tt.fields.ChainID,
				RoundRank:             tt.fields.RoundRank,
				PrevBlock:             tt.fields.PrevBlock,
				TxnsMap:               tt.fields.TxnsMap,
				ClientState:           tt.fields.ClientState,
				stateStatus:           tt.fields.stateStatus,
				blockState:            tt.fields.blockState,
				isNotarized:           tt.fields.isNotarized,
				verificationStatus:    tt.fields.verificationStatus,
				RunningTxnCount:       tt.fields.RunningTxnCount,
				UniqueBlockExtensions: tt.fields.UniqueBlockExtensions,
				MagicBlock:            tt.fields.MagicBlock,
			}

			b.SetBlockState(tt.want)
			if got := b.GetBlockState(); got != tt.want {
				t.Errorf("GetBlockState() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBlock_GetClients(t *testing.T) {
	b := NewBlock("", 1)
	pbK1, _, err := encryption.GenerateKeys()
	if err != nil {
		t.Fatal(err)
	}
	client.SetClientSignatureScheme("ed25519")

	b.Txns = []*transaction.Transaction{
		{},
		{PublicKey: pbK1},
		{PublicKey: pbK1},
	}

	type fields struct {
		UnverifiedBlockBody   UnverifiedBlockBody
		VerificationTickets   []*VerificationTicket
		HashIDField           datastore.HashIDField
		Signature             string
		ChainID               datastore.Key
		RoundRank             int
		PrevBlock             *Block
		TxnsMap               map[string]bool
		ClientState           util.MerklePatriciaTrieI
		stateStatus           int8
		blockState            int8
		isNotarized           bool
		verificationStatus    int
		RunningTxnCount       int64
		UniqueBlockExtensions map[string]bool
		MagicBlock            *MagicBlock
	}
	tests := []struct {
		name   string
		fields fields
		want   []*client.Client
	}{
		{
			name:   "OK",
			fields: fields{UnverifiedBlockBody: b.UnverifiedBlockBody},
			want: func() []*client.Client {
				cl := client.NewClient()
				require.NoError(t, cl.SetPublicKey(b.Txns[1].PublicKey))

				return []*client.Client{
					cl,
				}
			}(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Block{
				UnverifiedBlockBody:   tt.fields.UnverifiedBlockBody,
				VerificationTickets:   tt.fields.VerificationTickets,
				HashIDField:           tt.fields.HashIDField,
				Signature:             tt.fields.Signature,
				ChainID:               tt.fields.ChainID,
				RoundRank:             tt.fields.RoundRank,
				PrevBlock:             tt.fields.PrevBlock,
				TxnsMap:               tt.fields.TxnsMap,
				ClientState:           tt.fields.ClientState,
				stateStatus:           tt.fields.stateStatus,
				blockState:            tt.fields.blockState,
				isNotarized:           tt.fields.isNotarized,
				verificationStatus:    tt.fields.verificationStatus,
				RunningTxnCount:       tt.fields.RunningTxnCount,
				UniqueBlockExtensions: tt.fields.UniqueBlockExtensions,
				MagicBlock:            tt.fields.MagicBlock,
			}
			got, err := b.GetClients()
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestBlock_GetStateStatus(t *testing.T) {
	type fields struct {
		UnverifiedBlockBody   UnverifiedBlockBody
		VerificationTickets   []*VerificationTicket
		HashIDField           datastore.HashIDField
		Signature             string
		ChainID               datastore.Key
		RoundRank             int
		PrevBlock             *Block
		TxnsMap               map[string]bool
		ClientState           util.MerklePatriciaTrieI
		stateStatus           int8
		blockState            int8
		isNotarized           bool
		verificationStatus    int
		RunningTxnCount       int64
		UniqueBlockExtensions map[string]bool
		MagicBlock            *MagicBlock
	}
	tests := []struct {
		name   string
		fields fields
		want   int8
	}{
		{
			name:   "OK",
			fields: fields{stateStatus: 1},
			want:   1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Block{
				UnverifiedBlockBody:   tt.fields.UnverifiedBlockBody,
				VerificationTickets:   tt.fields.VerificationTickets,
				HashIDField:           tt.fields.HashIDField,
				Signature:             tt.fields.Signature,
				ChainID:               tt.fields.ChainID,
				RoundRank:             tt.fields.RoundRank,
				PrevBlock:             tt.fields.PrevBlock,
				TxnsMap:               tt.fields.TxnsMap,
				ClientState:           tt.fields.ClientState,
				stateStatus:           tt.fields.stateStatus,
				blockState:            tt.fields.blockState,
				isNotarized:           tt.fields.isNotarized,
				verificationStatus:    tt.fields.verificationStatus,
				RunningTxnCount:       tt.fields.RunningTxnCount,
				UniqueBlockExtensions: tt.fields.UniqueBlockExtensions,
				MagicBlock:            tt.fields.MagicBlock,
			}
			if got := b.GetStateStatus(); got != tt.want {
				t.Errorf("GetStateStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBlock_IsStateComputed(t *testing.T) {
	type fields struct {
		UnverifiedBlockBody   UnverifiedBlockBody
		VerificationTickets   []*VerificationTicket
		HashIDField           datastore.HashIDField
		Signature             string
		ChainID               datastore.Key
		RoundRank             int
		PrevBlock             *Block
		TxnsMap               map[string]bool
		ClientState           util.MerklePatriciaTrieI
		stateStatus           int8
		blockState            int8
		isNotarized           bool
		verificationStatus    int
		RunningTxnCount       int64
		UniqueBlockExtensions map[string]bool
		MagicBlock            *MagicBlock
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name:   "TRUE",
			fields: fields{stateStatus: StateSuccessful},
			want:   true,
		},
		{
			name:   "FALSE",
			fields: fields{stateStatus: StatePending},
			want:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Block{
				UnverifiedBlockBody:   tt.fields.UnverifiedBlockBody,
				VerificationTickets:   tt.fields.VerificationTickets,
				HashIDField:           tt.fields.HashIDField,
				Signature:             tt.fields.Signature,
				ChainID:               tt.fields.ChainID,
				RoundRank:             tt.fields.RoundRank,
				PrevBlock:             tt.fields.PrevBlock,
				TxnsMap:               tt.fields.TxnsMap,
				ClientState:           tt.fields.ClientState,
				stateStatus:           tt.fields.stateStatus,
				blockState:            tt.fields.blockState,
				isNotarized:           tt.fields.isNotarized,
				verificationStatus:    tt.fields.verificationStatus,
				RunningTxnCount:       tt.fields.RunningTxnCount,
				UniqueBlockExtensions: tt.fields.UniqueBlockExtensions,
				MagicBlock:            tt.fields.MagicBlock,
			}
			if got := b.IsStateComputed(); got != tt.want {
				t.Errorf("IsStateComputed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBlock_GetTransaction(t *testing.T) {
	b := NewBlock("", 1)
	for i := 0; i < 3; i++ {
		txn := transaction.Transaction{}
		txn.Hash = encryption.Hash("data" + strconv.Itoa(i))
		b.Txns = append(b.Txns, &txn)
	}

	type fields struct {
		UnverifiedBlockBody   UnverifiedBlockBody
		VerificationTickets   []*VerificationTicket
		HashIDField           datastore.HashIDField
		Signature             string
		ChainID               datastore.Key
		RoundRank             int
		PrevBlock             *Block
		TxnsMap               map[string]bool
		ClientState           util.MerklePatriciaTrieI
		stateStatus           int8
		blockState            int8
		isNotarized           bool
		verificationStatus    int
		RunningTxnCount       int64
		UniqueBlockExtensions map[string]bool
		MagicBlock            *MagicBlock
	}
	type args struct {
		hash string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *transaction.Transaction
	}{
		{
			name:   "OK",
			fields: fields{UnverifiedBlockBody: b.UnverifiedBlockBody},
			args:   args{hash: b.Txns[2].Hash},
			want:   b.Txns[2],
		},
		{
			name:   "Nil_OK",
			fields: fields{UnverifiedBlockBody: b.UnverifiedBlockBody},
			args:   args{hash: "unknown hash"},
			want:   nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Block{
				UnverifiedBlockBody:   tt.fields.UnverifiedBlockBody,
				VerificationTickets:   tt.fields.VerificationTickets,
				HashIDField:           tt.fields.HashIDField,
				Signature:             tt.fields.Signature,
				ChainID:               tt.fields.ChainID,
				RoundRank:             tt.fields.RoundRank,
				PrevBlock:             tt.fields.PrevBlock,
				TxnsMap:               tt.fields.TxnsMap,
				ClientState:           tt.fields.ClientState,
				stateStatus:           tt.fields.stateStatus,
				blockState:            tt.fields.blockState,
				isNotarized:           tt.fields.isNotarized,
				verificationStatus:    tt.fields.verificationStatus,
				RunningTxnCount:       tt.fields.RunningTxnCount,
				UniqueBlockExtensions: tt.fields.UniqueBlockExtensions,
				MagicBlock:            tt.fields.MagicBlock,
			}
			if got := b.GetTransaction(tt.args.hash); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetTransaction() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBlock_IsBlockNotarized(t *testing.T) {
	type fields struct {
		UnverifiedBlockBody   UnverifiedBlockBody
		VerificationTickets   []*VerificationTicket
		HashIDField           datastore.HashIDField
		Signature             string
		ChainID               datastore.Key
		RoundRank             int
		PrevBlock             *Block
		TxnsMap               map[string]bool
		ClientState           util.MerklePatriciaTrieI
		stateStatus           int8
		blockState            int8
		isNotarized           bool
		verificationStatus    int
		RunningTxnCount       int64
		UniqueBlockExtensions map[string]bool
		MagicBlock            *MagicBlock
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name:   "TRUE",
			fields: fields{},
			want:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Block{
				UnverifiedBlockBody:   tt.fields.UnverifiedBlockBody,
				VerificationTickets:   tt.fields.VerificationTickets,
				HashIDField:           tt.fields.HashIDField,
				Signature:             tt.fields.Signature,
				ChainID:               tt.fields.ChainID,
				RoundRank:             tt.fields.RoundRank,
				PrevBlock:             tt.fields.PrevBlock,
				TxnsMap:               tt.fields.TxnsMap,
				ClientState:           tt.fields.ClientState,
				stateStatus:           tt.fields.stateStatus,
				blockState:            tt.fields.blockState,
				isNotarized:           tt.fields.isNotarized,
				verificationStatus:    tt.fields.verificationStatus,
				RunningTxnCount:       tt.fields.RunningTxnCount,
				UniqueBlockExtensions: tt.fields.UniqueBlockExtensions,
				MagicBlock:            tt.fields.MagicBlock,
			}

			b.SetBlockNotarized()
			if got := b.IsBlockNotarized(); got != tt.want {
				t.Errorf("UpdateBlockNotarization() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBlock_GetVerificationStatus(t *testing.T) {
	type fields struct {
		UnverifiedBlockBody   UnverifiedBlockBody
		VerificationTickets   []*VerificationTicket
		HashIDField           datastore.HashIDField
		Signature             string
		ChainID               datastore.Key
		RoundRank             int
		PrevBlock             *Block
		TxnsMap               map[string]bool
		ClientState           util.MerklePatriciaTrieI
		stateStatus           int8
		blockState            int8
		isNotarized           bool
		verificationStatus    int
		RunningTxnCount       int64
		UniqueBlockExtensions map[string]bool
		MagicBlock            *MagicBlock
	}
	tests := []struct {
		name   string
		fields fields
		want   int
	}{
		{
			name: "OK",
			want: 8,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Block{
				UnverifiedBlockBody:   tt.fields.UnverifiedBlockBody,
				VerificationTickets:   tt.fields.VerificationTickets,
				HashIDField:           tt.fields.HashIDField,
				Signature:             tt.fields.Signature,
				ChainID:               tt.fields.ChainID,
				RoundRank:             tt.fields.RoundRank,
				PrevBlock:             tt.fields.PrevBlock,
				TxnsMap:               tt.fields.TxnsMap,
				ClientState:           tt.fields.ClientState,
				stateStatus:           tt.fields.stateStatus,
				blockState:            tt.fields.blockState,
				isNotarized:           tt.fields.isNotarized,
				verificationStatus:    tt.fields.verificationStatus,
				RunningTxnCount:       tt.fields.RunningTxnCount,
				UniqueBlockExtensions: tt.fields.UniqueBlockExtensions,
				MagicBlock:            tt.fields.MagicBlock,
			}

			b.SetVerificationStatus(tt.want)
			if got := b.GetVerificationStatus(); got != tt.want {
				t.Errorf("GetVerificationStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBlock_UnknownTickets(t *testing.T) {
	tickets := []*VerificationTicket{
		{
			VerifierID: "id 1",
		},
		{
			VerifierID: "id 2",
		},
	}
	newTickets := []*VerificationTicket{
		{
			VerifierID: "id 3",
		},
		{
			VerifierID: "id 4",
		},
	}

	type fields struct {
		UnverifiedBlockBody   UnverifiedBlockBody
		VerificationTickets   []*VerificationTicket
		HashIDField           datastore.HashIDField
		Signature             string
		ChainID               datastore.Key
		RoundRank             int
		PrevBlock             *Block
		TxnsMap               map[string]bool
		ClientState           util.MerklePatriciaTrieI
		stateStatus           int8
		blockState            int8
		isNotarized           bool
		verificationStatus    int
		RunningTxnCount       int64
		UniqueBlockExtensions map[string]bool
		MagicBlock            *MagicBlock
	}
	type args struct {
		vts []*VerificationTicket
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []*VerificationTicket
	}{
		{
			name: "OK",
			fields: fields{
				VerificationTickets: tickets,
			},
			args: args{vts: newTickets},
			want: newTickets,
		},
		{
			name: "Nil_New_Tickets_OK",
			fields: fields{
				VerificationTickets: tickets,
			},
			args: args{vts: make([]*VerificationTicket, 1)},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Block{
				UnverifiedBlockBody:   tt.fields.UnverifiedBlockBody,
				VerificationTickets:   tt.fields.VerificationTickets,
				HashIDField:           tt.fields.HashIDField,
				Signature:             tt.fields.Signature,
				ChainID:               tt.fields.ChainID,
				RoundRank:             tt.fields.RoundRank,
				PrevBlock:             tt.fields.PrevBlock,
				TxnsMap:               tt.fields.TxnsMap,
				ClientState:           tt.fields.ClientState,
				stateStatus:           tt.fields.stateStatus,
				blockState:            tt.fields.blockState,
				isNotarized:           tt.fields.isNotarized,
				verificationStatus:    tt.fields.verificationStatus,
				RunningTxnCount:       tt.fields.RunningTxnCount,
				UniqueBlockExtensions: tt.fields.UniqueBlockExtensions,
				MagicBlock:            tt.fields.MagicBlock,
			}
			if got := b.UnknownTickets(tt.args.vts); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UnknownTickets() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBlock_AddUniqueBlockExtension(t *testing.T) {
	b := NewBlock("", 1)
	b.MinerID = "miner id"
	type fields struct {
		UnverifiedBlockBody   UnverifiedBlockBody
		VerificationTickets   []*VerificationTicket
		HashIDField           datastore.HashIDField
		Signature             string
		ChainID               datastore.Key
		RoundRank             int
		PrevBlock             *Block
		TxnsMap               map[string]bool
		ClientState           util.MerklePatriciaTrieI
		stateStatus           int8
		blockState            int8
		isNotarized           bool
		verificationStatus    int
		RunningTxnCount       int64
		UniqueBlockExtensions map[string]bool
		MagicBlock            *MagicBlock
	}
	type args struct {
		eb *Block
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   map[string]bool
	}{
		{
			name: "OK",
			args: args{eb: b},
			want: map[string]bool{
				b.MinerID: true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Block{
				UnverifiedBlockBody:   tt.fields.UnverifiedBlockBody,
				VerificationTickets:   tt.fields.VerificationTickets,
				HashIDField:           tt.fields.HashIDField,
				Signature:             tt.fields.Signature,
				ChainID:               tt.fields.ChainID,
				RoundRank:             tt.fields.RoundRank,
				PrevBlock:             tt.fields.PrevBlock,
				TxnsMap:               tt.fields.TxnsMap,
				ClientState:           tt.fields.ClientState,
				stateStatus:           tt.fields.stateStatus,
				blockState:            tt.fields.blockState,
				isNotarized:           tt.fields.isNotarized,
				verificationStatus:    tt.fields.verificationStatus,
				RunningTxnCount:       tt.fields.RunningTxnCount,
				UniqueBlockExtensions: tt.fields.UniqueBlockExtensions,
				MagicBlock:            tt.fields.MagicBlock,
			}

			b.AddUniqueBlockExtension(tt.args.eb)
			if !reflect.DeepEqual(b.UniqueBlockExtensions, tt.want) {
				t.Errorf("AddUniqueBlockExtension() got = %v, want = %v", b.UniqueBlockExtensions, tt.want)
			}
		})
	}
}

func TestBlock_GetPrevBlockVerificationTickets(t *testing.T) {
	tickets := []*VerificationTicket{
		{
			VerifierID: "id 1",
		},
		{
			VerifierID: "id 2",
		},
	}

	type fields struct {
		UnverifiedBlockBody   UnverifiedBlockBody
		VerificationTickets   []*VerificationTicket
		HashIDField           datastore.HashIDField
		Signature             string
		ChainID               datastore.Key
		RoundRank             int
		PrevBlock             *Block
		TxnsMap               map[string]bool
		ClientState           util.MerklePatriciaTrieI
		stateStatus           int8
		blockState            int8
		isNotarized           bool
		verificationStatus    int
		RunningTxnCount       int64
		UniqueBlockExtensions map[string]bool
		MagicBlock            *MagicBlock
	}
	tests := []struct {
		name      string
		fields    fields
		wantPbvts []*VerificationTicket
	}{
		{
			name:      "Nil_tickets_OK",
			fields:    fields{},
			wantPbvts: nil,
		},
		{
			name: "OK",
			fields: fields{
				UnverifiedBlockBody: UnverifiedBlockBody{PrevBlockVerificationTickets: tickets},
			},
			wantPbvts: tickets,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Block{
				UnverifiedBlockBody:   tt.fields.UnverifiedBlockBody,
				VerificationTickets:   tt.fields.VerificationTickets,
				HashIDField:           tt.fields.HashIDField,
				Signature:             tt.fields.Signature,
				ChainID:               tt.fields.ChainID,
				RoundRank:             tt.fields.RoundRank,
				PrevBlock:             tt.fields.PrevBlock,
				TxnsMap:               tt.fields.TxnsMap,
				ClientState:           tt.fields.ClientState,
				stateStatus:           tt.fields.stateStatus,
				blockState:            tt.fields.blockState,
				isNotarized:           tt.fields.isNotarized,
				verificationStatus:    tt.fields.verificationStatus,
				RunningTxnCount:       tt.fields.RunningTxnCount,
				UniqueBlockExtensions: tt.fields.UniqueBlockExtensions,
				MagicBlock:            tt.fields.MagicBlock,
			}
			if gotPbvts := b.GetPrevBlockVerificationTickets(); !reflect.DeepEqual(gotPbvts, tt.wantPbvts) {
				t.Errorf("GetPrevBlockVerificationTickets() = %v, want %v", gotPbvts, tt.wantPbvts)
			}
		})
	}
}

func TestBlock_PrevBlockVerificationTicketsSize(t *testing.T) {
	b := NewBlock("", 1)
	tickets := []*VerificationTicket{
		{
			VerifierID: "id 1",
		},
		{
			VerifierID: "id 2",
		},
	}
	b.SetPrevBlockVerificationTickets(tickets)

	type fields struct {
		UnverifiedBlockBody   UnverifiedBlockBody
		VerificationTickets   []*VerificationTicket
		HashIDField           datastore.HashIDField
		Signature             string
		ChainID               datastore.Key
		RoundRank             int
		PrevBlock             *Block
		TxnsMap               map[string]bool
		ClientState           util.MerklePatriciaTrieI
		stateStatus           int8
		blockState            int8
		isNotarized           bool
		verificationStatus    int
		RunningTxnCount       int64
		UniqueBlockExtensions map[string]bool
		MagicBlock            *MagicBlock
	}
	tests := []struct {
		name   string
		fields fields
		want   int
	}{
		{
			name: "OK",
			fields: fields{
				UnverifiedBlockBody: b.UnverifiedBlockBody,
			},
			want: len(tickets),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Block{
				UnverifiedBlockBody:   tt.fields.UnverifiedBlockBody,
				VerificationTickets:   tt.fields.VerificationTickets,
				HashIDField:           tt.fields.HashIDField,
				Signature:             tt.fields.Signature,
				ChainID:               tt.fields.ChainID,
				RoundRank:             tt.fields.RoundRank,
				PrevBlock:             tt.fields.PrevBlock,
				TxnsMap:               tt.fields.TxnsMap,
				ClientState:           tt.fields.ClientState,
				stateStatus:           tt.fields.stateStatus,
				blockState:            tt.fields.blockState,
				isNotarized:           tt.fields.isNotarized,
				verificationStatus:    tt.fields.verificationStatus,
				RunningTxnCount:       tt.fields.RunningTxnCount,
				UniqueBlockExtensions: tt.fields.UniqueBlockExtensions,
				MagicBlock:            tt.fields.MagicBlock,
			}
			if got := b.PrevBlockVerificationTicketsSize(); got != tt.want {
				t.Errorf("PrevBlockVerificationTicketsSize() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUnverifiedBlockBody_GetRoundRandomSeed(t *testing.T) {
	type fields struct {
		VersionField                   datastore.VersionField
		CreationDateField              datastore.CreationDateField
		LatestFinalizedMagicBlockHash  string
		LatestFinalizedMagicBlockRound int64
		PrevHash                       string
		PrevBlockVerificationTickets   []*VerificationTicket
		MinerID                        datastore.Key
		Round                          int64
		RoundRandomSeed                int64
		RoundTimeoutCount              int
		ClientStateHash                util.Key
		Txns                           []*transaction.Transaction
	}
	tests := []struct {
		name   string
		fields fields
		want   int64
	}{
		{
			name: "OK",
			want: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &UnverifiedBlockBody{
				VersionField:                   tt.fields.VersionField,
				CreationDateField:              tt.fields.CreationDateField,
				LatestFinalizedMagicBlockHash:  tt.fields.LatestFinalizedMagicBlockHash,
				LatestFinalizedMagicBlockRound: tt.fields.LatestFinalizedMagicBlockRound,
				PrevHash:                       tt.fields.PrevHash,
				PrevBlockVerificationTickets:   tt.fields.PrevBlockVerificationTickets,
				MinerID:                        tt.fields.MinerID,
				Round:                          tt.fields.Round,
				RoundRandomSeed:                tt.fields.RoundRandomSeed,
				RoundTimeoutCount:              tt.fields.RoundTimeoutCount,
				ClientStateHash:                tt.fields.ClientStateHash,
				Txns:                           tt.fields.Txns,
			}

			u.SetRoundRandomSeed(tt.want)
			if got := u.GetRoundRandomSeed(); got != tt.want {
				t.Errorf("GetRoundRandomSeed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBlock_DoReadLock(t *testing.T) {
	type fields struct {
		UnverifiedBlockBody   UnverifiedBlockBody
		VerificationTickets   []*VerificationTicket
		HashIDField           datastore.HashIDField
		Signature             string
		ChainID               datastore.Key
		RoundRank             int
		PrevBlock             *Block
		TxnsMap               map[string]bool
		ClientState           util.MerklePatriciaTrieI
		stateStatus           int8
		blockState            int8
		isNotarized           bool
		verificationStatus    int
		RunningTxnCount       int64
		UniqueBlockExtensions map[string]bool
		MagicBlock            *MagicBlock
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name:   "OK",
			fields: fields{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Block{
				UnverifiedBlockBody:   tt.fields.UnverifiedBlockBody,
				VerificationTickets:   tt.fields.VerificationTickets,
				HashIDField:           tt.fields.HashIDField,
				Signature:             tt.fields.Signature,
				ChainID:               tt.fields.ChainID,
				RoundRank:             tt.fields.RoundRank,
				PrevBlock:             tt.fields.PrevBlock,
				TxnsMap:               tt.fields.TxnsMap,
				ClientState:           tt.fields.ClientState,
				stateStatus:           tt.fields.stateStatus,
				blockState:            tt.fields.blockState,
				isNotarized:           tt.fields.isNotarized,
				verificationStatus:    tt.fields.verificationStatus,
				RunningTxnCount:       tt.fields.RunningTxnCount,
				UniqueBlockExtensions: tt.fields.UniqueBlockExtensions,
				MagicBlock:            tt.fields.MagicBlock,
			}

			b.DoReadLock()
			b.DoReadUnlock()
		})
	}
}
