package block

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"sync"
	"testing"

	"0chain.net/chaincore/client"
	"0chain.net/chaincore/node"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"github.com/0chain/common/core/util"
	"github.com/stretchr/testify/require"
)

func TestNewMagicBlock(t *testing.T) {
	tests := []struct {
		name string
		want *MagicBlock
	}{
		{
			name: "OK",
			want: &MagicBlock{Mpks: NewMpks(), ShareOrSigns: NewGroupSharesOrSigns()},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewMagicBlock(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewMagicBlock() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMagicBlock_GetShareOrSigns(t *testing.T) {
	type fields struct {
		HashIDField            datastore.HashIDField
		PreviousMagicBlockHash datastore.Key
		MagicBlockNumber       int64
		StartingRound          int64
		Miners                 *node.Pool
		Sharders               *node.Pool
		ShareOrSigns           *GroupSharesOrSigns
		Mpks                   *Mpks
		T                      int
		K                      int
		N                      int
	}
	tests := []struct {
		name   string
		fields fields
		want   *GroupSharesOrSigns
	}{
		{
			name: "OK",
			want: NewGroupSharesOrSigns(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mb := &MagicBlock{
				HashIDField:            tt.fields.HashIDField,
				mutex:                  sync.RWMutex{},
				PreviousMagicBlockHash: tt.fields.PreviousMagicBlockHash,
				MagicBlockNumber:       tt.fields.MagicBlockNumber,
				StartingRound:          tt.fields.StartingRound,
				Miners:                 tt.fields.Miners,
				Sharders:               tt.fields.Sharders,
				ShareOrSigns:           tt.fields.ShareOrSigns,
				Mpks:                   tt.fields.Mpks,
				T:                      tt.fields.T,
				K:                      tt.fields.K,
				N:                      tt.fields.N,
			}

			mb.SetShareOrSigns(tt.want)
			if got := mb.GetShareOrSigns(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetShareOrSigns() = %v, want %v", got, tt.want)
			}
		})
	}
}

func createMiners(np *node.Pool) {
	sd := node.Node{Host: "127.0.0.1", Port: 7071, Type: node.NodeTypeMiner, Status: node.NodeStatusActive}
	sigScheme1 := encryption.NewBLS0ChainScheme()
	err := sigScheme1.GenerateKeys()
	if err != nil {
		panic(err)
	}
	sd.SetSignatureScheme(sigScheme1)
	np.AddNode(&sd)

	sb := node.Node{Host: "127.0.0.2", Port: 7070, Type: node.NodeTypeMiner, Status: node.NodeStatusActive}
	sigScheme2 := encryption.NewBLS0ChainScheme()
	err = sigScheme2.GenerateKeys()
	if err != nil {
		panic(err)
	}
	sb.SetSignatureScheme(sigScheme2)
	np.AddNode(&sb)

	ns := node.Node{Host: "127.0.0.3", Port: 7070, Type: node.NodeTypeMiner, Status: node.NodeStatusActive}
	sigScheme3 := encryption.NewBLS0ChainScheme()
	err = sigScheme3.GenerateKeys()
	if err != nil {
		panic(err)
	}
	np.AddNode(&ns)
}

func createSharders(np *node.Pool) {
	sd := node.Node{Host: "127.0.0.1", Port: 7171, Type: node.NodeTypeSharder, Status: node.NodeStatusActive}
	sigScheme1 := encryption.NewBLS0ChainScheme()
	err := sigScheme1.GenerateKeys()
	if err != nil {
		panic(err)
	}
	sd.SetSignatureScheme(sigScheme1)
	np.AddNode(&sd)

	sb := node.Node{Host: "127.0.0.2", Port: 7172, Type: node.NodeTypeSharder, Status: node.NodeStatusActive}
	sigScheme2 := encryption.NewBLS0ChainScheme()
	err = sigScheme2.GenerateKeys()
	if err != nil {
		panic(err)
	}
	sb.SetSignatureScheme(sigScheme2)
	np.AddNode(&sb)

	ns := node.Node{Host: "127.0.0.3", Port: 7173, Type: node.NodeTypeSharder, Status: node.NodeStatusActive}
	sigScheme3 := encryption.NewBLS0ChainScheme()
	err = sigScheme3.GenerateKeys()
	if err != nil {
		panic(err)
	}
	np.AddNode(&ns)
}

func TestMagicBlockClone(t *testing.T) {
	b := NewBlock("", 1)
	mb := NewMagicBlock()
	b.MagicBlock = mb
	mpool := node.NewPool(node.NodeTypeMiner)
	createMiners(mpool)
	mb.Miners = mpool

	mb.Sharders = node.NewPool(node.NodeTypeSharder)
	createSharders(mb.Sharders)

	clone := b.Clone()
	if clone.Sharders == nil {
		t.Fatal("clone sharders is nil")
	}

	t.Log("clone sharders size", clone.Sharders.Size())

	if clone.Sharders.Size() != b.Sharders.Size() {
		t.Fatal("clone sharders size is not equal to original sharders size")
	}

	for _, n := range clone.Sharders.CopyNodesMap() {
		if _, ok := mb.Sharders.CopyNodesMap()[n.GetKey()]; !ok {
			t.Fatal("clone sharders doesn't contain original sharder")
		}
	}
}

func TestMagicBlock_Encode(t *testing.T) {
	mb := NewMagicBlock()
	mpool := node.NewPool(node.NodeTypeMiner)
	createMiners(mpool)
	mb.Miners = mpool

	blob, err := json.Marshal(mb)
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		HashIDField            datastore.HashIDField
		PreviousMagicBlockHash datastore.Key
		MagicBlockNumber       int64
		StartingRound          int64
		Miners                 *node.Pool
		Sharders               *node.Pool
		ShareOrSigns           *GroupSharesOrSigns
		Mpks                   *Mpks
		T                      int
		K                      int
		N                      int
	}
	tests := []struct {
		name   string
		fields fields
		want   []byte
	}{
		{
			name: "OK",
			fields: fields{
				HashIDField:            mb.HashIDField,
				PreviousMagicBlockHash: mb.PreviousMagicBlockHash,
				MagicBlockNumber:       mb.MagicBlockNumber,
				StartingRound:          mb.StartingRound,
				Miners:                 mb.Miners,
				Sharders:               mb.Sharders,
				ShareOrSigns:           mb.ShareOrSigns,
				Mpks:                   mb.Mpks,
				T:                      mb.T,
				K:                      mb.K,
				N:                      mb.N,
			},
			want: blob,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mb := &MagicBlock{
				HashIDField:            tt.fields.HashIDField,
				mutex:                  sync.RWMutex{},
				PreviousMagicBlockHash: tt.fields.PreviousMagicBlockHash,
				MagicBlockNumber:       tt.fields.MagicBlockNumber,
				StartingRound:          tt.fields.StartingRound,
				Miners:                 tt.fields.Miners,
				Sharders:               tt.fields.Sharders,
				ShareOrSigns:           tt.fields.ShareOrSigns,
				Mpks:                   tt.fields.Mpks,
				T:                      tt.fields.T,
				K:                      tt.fields.K,
				N:                      tt.fields.N,
			}
			// if got := mb.Encode(); !reflect.DeepEqual(got, tt.want) {
			// 	t.Errorf("Encode() = %v, want %v", got, tt.want)
			// }

			b, err := mb.MarshalMsg(nil)
			require.NoError(t, err)
			nmb := NewMagicBlock()
			_, err = nmb.UnmarshalMsg(b)
			require.NoError(t, err)
			fmt.Println(nmb.Miners.Size())
		})
	}
}

func TestMagicBlock_Decode(t *testing.T) {
	mb := NewMagicBlock()
	blob, err := json.Marshal(mb)
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		HashIDField            datastore.HashIDField
		PreviousMagicBlockHash datastore.Key
		MagicBlockNumber       int64
		StartingRound          int64
		Miners                 *node.Pool
		Sharders               *node.Pool
		ShareOrSigns           *GroupSharesOrSigns
		Mpks                   *Mpks
		T                      int
		K                      int
		N                      int
	}
	type args struct {
		input []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		want    *MagicBlock
	}{
		{
			name: "OK",
			fields: fields{
				HashIDField:            mb.HashIDField,
				PreviousMagicBlockHash: mb.PreviousMagicBlockHash,
				MagicBlockNumber:       mb.MagicBlockNumber,
				StartingRound:          mb.StartingRound,
				Miners:                 mb.Miners,
				Sharders:               mb.Sharders,
				ShareOrSigns:           mb.ShareOrSigns,
				Mpks:                   mb.Mpks,
				T:                      mb.T,
				K:                      mb.K,
				N:                      mb.N,
			},
			args: args{input: blob},
			want: mb,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mb := &MagicBlock{
				HashIDField:            tt.fields.HashIDField,
				mutex:                  sync.RWMutex{},
				PreviousMagicBlockHash: tt.fields.PreviousMagicBlockHash,
				MagicBlockNumber:       tt.fields.MagicBlockNumber,
				StartingRound:          tt.fields.StartingRound,
				Miners:                 tt.fields.Miners,
				Sharders:               tt.fields.Sharders,
				ShareOrSigns:           tt.fields.ShareOrSigns,
				Mpks:                   tt.fields.Mpks,
				T:                      tt.fields.T,
				K:                      tt.fields.K,
				N:                      tt.fields.N,
			}
			if err := mb.Decode(tt.args.input); (err != nil) != tt.wantErr {
				t.Errorf("Decode() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(mb, tt.want) {
				t.Errorf("Decode() = %v, want %v", mb, tt.want)
			}
		})
	}
}

func TestMagicBlock_GetHash(t *testing.T) {
	client.SetClientSignatureScheme("ed25519")
	pbK, _, err := encryption.GenerateKeys()
	require.NoError(t, err)
	mb := NewMagicBlock()
	mb.MagicBlockNumber = 10
	mb.PreviousMagicBlockHash = encryption.Hash("prev mb")
	mb.StartingRound = 1
	mb.Miners = node.NewPool(1)
	n, err := makeTestNode(pbK)
	if err != nil {
		t.Fatal(err)
	}
	mb.Miners.AddNode(n)
	mb.Sharders = node.NewPool(1)
	mb.Sharders.AddNode(n)
	mb.Mpks.Mpks = map[string]*MPK{
		"key": nil,
	}
	mb.T = 1
	mb.N = 1

	data := []byte(strconv.FormatInt(mb.MagicBlockNumber, 10))
	data = append(data, []byte(mb.PreviousMagicBlockHash)...)
	data = append(data, []byte(strconv.FormatInt(mb.StartingRound, 10))...)
	var minerKeys, sharderKeys, mpkKeys []string
	minerKeys = mb.Miners.Keys()
	sort.Strings(minerKeys)
	for _, v := range minerKeys {
		data = append(data, []byte(v)...)
	}
	sharderKeys = mb.Sharders.Keys()
	sort.Strings(sharderKeys)
	for _, v := range sharderKeys {
		data = append(data, []byte(v)...)
	}
	shareBytes, _ := hex.DecodeString(mb.GetShareOrSigns().GetHash())
	data = append(data, shareBytes...)
	for k := range mb.Mpks.Mpks {
		mpkKeys = append(mpkKeys, k)
	}
	sort.Strings(mpkKeys)
	for _, v := range mpkKeys {
		data = append(data, []byte(v)...)
	}
	data = append(data, []byte(strconv.Itoa(mb.T))...)
	data = append(data, []byte(strconv.Itoa(mb.N))...)
	hash := encryption.RawHash(data)

	type fields struct {
		HashIDField            datastore.HashIDField
		PreviousMagicBlockHash datastore.Key
		MagicBlockNumber       int64
		StartingRound          int64
		Miners                 *node.Pool
		Sharders               *node.Pool
		ShareOrSigns           *GroupSharesOrSigns
		Mpks                   *Mpks
		T                      int
		K                      int
		N                      int
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "OK",
			fields: fields{
				HashIDField:            mb.HashIDField,
				PreviousMagicBlockHash: mb.PreviousMagicBlockHash,
				MagicBlockNumber:       mb.MagicBlockNumber,
				StartingRound:          mb.StartingRound,
				Miners:                 mb.Miners,
				Sharders:               mb.Sharders,
				ShareOrSigns:           mb.ShareOrSigns,
				Mpks:                   mb.Mpks,
				T:                      mb.T,
				K:                      mb.K,
				N:                      mb.N,
			},
			want: util.ToHex(hash),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mb := &MagicBlock{
				HashIDField:            tt.fields.HashIDField,
				mutex:                  sync.RWMutex{},
				PreviousMagicBlockHash: tt.fields.PreviousMagicBlockHash,
				MagicBlockNumber:       tt.fields.MagicBlockNumber,
				StartingRound:          tt.fields.StartingRound,
				Miners:                 tt.fields.Miners,
				Sharders:               tt.fields.Sharders,
				ShareOrSigns:           tt.fields.ShareOrSigns,
				Mpks:                   tt.fields.Mpks,
				T:                      tt.fields.T,
				K:                      tt.fields.K,
				N:                      tt.fields.N,
			}
			if got := mb.GetHash(); got != tt.want {
				t.Errorf("GetHash() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMagicBlock_IsActiveNode(t *testing.T) {
	mb := NewMagicBlock()
	mb.Miners = node.NewPool(1)
	n, err := makeTestNode("")
	if err != nil {
		t.Fatal(err)
	}
	mb.Miners.AddNode(n)
	mb.Sharders = node.NewPool(1)
	n, err = makeTestNode("")
	if err != nil {
		t.Fatal(err)
	}
	mb.Sharders.AddNode(n)

	type fields struct {
		HashIDField            datastore.HashIDField
		PreviousMagicBlockHash datastore.Key
		MagicBlockNumber       int64
		StartingRound          int64
		Miners                 *node.Pool
		Sharders               *node.Pool
		ShareOrSigns           *GroupSharesOrSigns
		Mpks                   *Mpks
		T                      int
		K                      int
		N                      int
	}
	type args struct {
		id    string
		round int64
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "Nil_Miners_FALSE",
			fields: fields{
				HashIDField:            mb.HashIDField,
				PreviousMagicBlockHash: mb.PreviousMagicBlockHash,
				MagicBlockNumber:       mb.MagicBlockNumber,
				StartingRound:          mb.StartingRound,
				Miners:                 nil,
				Sharders:               mb.Sharders,
				ShareOrSigns:           mb.ShareOrSigns,
				Mpks:                   mb.Mpks,
				T:                      mb.T,
				K:                      mb.K,
				N:                      mb.N,
			},
			want: false,
		},
		{
			name: "Has_Node_And_Round_Greater_Than_Starting_Round_TRUE",
			fields: fields{
				HashIDField:            mb.HashIDField,
				PreviousMagicBlockHash: mb.PreviousMagicBlockHash,
				MagicBlockNumber:       mb.MagicBlockNumber,
				StartingRound:          0,
				Miners:                 mb.Miners,
				Sharders:               mb.Sharders,
				ShareOrSigns:           mb.ShareOrSigns,
				Mpks:                   mb.Mpks,
				T:                      mb.T,
				K:                      mb.K,
				N:                      mb.N,
			},
			args: args{id: n.GetKey(), round: 1},
			want: true,
		},
		{
			name: "Node_Doesnt_Exist_In_Sharders_Pool_FALSE",
			fields: fields{
				HashIDField:            mb.HashIDField,
				PreviousMagicBlockHash: mb.PreviousMagicBlockHash,
				MagicBlockNumber:       mb.MagicBlockNumber,
				StartingRound:          0,
				Miners:                 mb.Miners,
				Sharders:               mb.Sharders,
				ShareOrSigns:           mb.ShareOrSigns,
				Mpks:                   mb.Mpks,
				T:                      mb.T,
				K:                      mb.K,
				N:                      mb.N,
			},
			args: args{round: 1},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mb := &MagicBlock{
				HashIDField:            tt.fields.HashIDField,
				mutex:                  sync.RWMutex{},
				PreviousMagicBlockHash: tt.fields.PreviousMagicBlockHash,
				MagicBlockNumber:       tt.fields.MagicBlockNumber,
				StartingRound:          tt.fields.StartingRound,
				Miners:                 tt.fields.Miners,
				Sharders:               tt.fields.Sharders,
				ShareOrSigns:           tt.fields.ShareOrSigns,
				Mpks:                   tt.fields.Mpks,
				T:                      tt.fields.T,
				K:                      tt.fields.K,
				N:                      tt.fields.N,
			}
			if got := mb.IsActiveNode(tt.args.id, tt.args.round); got != tt.want {
				t.Errorf("IsActiveNode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMagicBlock_VerifyMinersSignatures(t *testing.T) {
	pbK, prK, err := encryption.GenerateKeys()
	if err != nil {
		t.Fatal(err)
	}
	client.SetClientSignatureScheme("ed25519")
	mb := NewMagicBlock()
	mb.Miners = node.NewPool(1)
	n, err := makeTestNode(pbK)
	if err != nil {
		t.Fatal(err)
	}
	mb.Miners.AddNode(n)

	type fields struct {
		HashIDField            datastore.HashIDField
		PreviousMagicBlockHash datastore.Key
		MagicBlockNumber       int64
		StartingRound          int64
		Miners                 *node.Pool
		Sharders               *node.Pool
		ShareOrSigns           *GroupSharesOrSigns
		Mpks                   *Mpks
		T                      int
		K                      int
		N                      int
	}
	type args struct {
		b *Block
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "Nil_Node_FALSE",
			fields: fields{
				HashIDField:            mb.HashIDField,
				PreviousMagicBlockHash: mb.PreviousMagicBlockHash,
				MagicBlockNumber:       mb.MagicBlockNumber,
				StartingRound:          0,
				Miners:                 mb.Miners,
				Sharders:               mb.Sharders,
				ShareOrSigns:           mb.ShareOrSigns,
				Mpks:                   mb.Mpks,
				T:                      mb.T,
				K:                      mb.K,
				N:                      mb.N,
			},
			args: args{
				b: func() *Block {
					b := NewBlock("", 1)
					b.VerificationTickets = []*VerificationTicket{
						{
							VerifierID: "unknown id",
						},
					}
					return b
				}(),
			},
			want: false,
		},
		{
			name: "Invalid_Sign_And_Hash_FALSE",
			fields: fields{
				HashIDField:            mb.HashIDField,
				PreviousMagicBlockHash: mb.PreviousMagicBlockHash,
				MagicBlockNumber:       mb.MagicBlockNumber,
				StartingRound:          0,
				Miners:                 mb.Miners,
				Sharders:               mb.Sharders,
				ShareOrSigns:           mb.ShareOrSigns,
				Mpks:                   mb.Mpks,
				T:                      mb.T,
				K:                      mb.K,
				N:                      mb.N,
			},
			args: args{
				b: func() *Block {
					b := NewBlock("", 1)
					b.VerificationTickets = []*VerificationTicket{
						{
							VerifierID: n.GetKey(),
						},
					}
					return b
				}(),
			},
			want: false,
		},
		{
			name: "TRUE",
			fields: fields{
				HashIDField:            mb.HashIDField,
				PreviousMagicBlockHash: mb.PreviousMagicBlockHash,
				MagicBlockNumber:       mb.MagicBlockNumber,
				StartingRound:          0,
				Miners:                 mb.Miners,
				Sharders:               mb.Sharders,
				ShareOrSigns:           mb.ShareOrSigns,
				Mpks:                   mb.Mpks,
				T:                      mb.T,
				K:                      mb.K,
				N:                      mb.N,
			},
			args: args{
				b: func() *Block {
					b := NewBlock("", 1)
					b.HashBlock()
					sign, err := encryption.Sign(prK, b.Hash)
					if err != nil {
						t.Fatal(err)
					}

					b.VerificationTickets = []*VerificationTicket{
						{
							VerifierID: n.GetKey(),
							Signature:  sign,
						},
					}
					return b
				}(),
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mb := &MagicBlock{
				HashIDField:            tt.fields.HashIDField,
				mutex:                  sync.RWMutex{},
				PreviousMagicBlockHash: tt.fields.PreviousMagicBlockHash,
				MagicBlockNumber:       tt.fields.MagicBlockNumber,
				StartingRound:          tt.fields.StartingRound,
				Miners:                 tt.fields.Miners,
				Sharders:               tt.fields.Sharders,
				ShareOrSigns:           tt.fields.ShareOrSigns,
				Mpks:                   tt.fields.Mpks,
				T:                      tt.fields.T,
				K:                      tt.fields.K,
				N:                      tt.fields.N,
			}
			if got := mb.VerifyMinersSignatures(tt.args.b); got != tt.want {
				t.Errorf("VerifyMinersSignatures() = %v, want %v", got, tt.want)
			}
		})
	}
}
