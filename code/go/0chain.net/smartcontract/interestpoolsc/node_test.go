package interestpoolsc

import (
	"reflect"
	"testing"
	"time"

	"golang.org/x/crypto/sha3"

	"0chain.net/core/datastore"
	"0chain.net/core/util"
)

const (
	globalNodeJson0       = "{\"min_lock_period\":\"0s\",\"simple_global_node\":{\"max_mint\":0,\"total_minted\":0,\"min_lock\":0,\"apr\":0,\"owner_id\":\"\"}}"
	globalNodeJson10      = "{\"min_lock_period\":\"10s\",\"simple_global_node\":{\"max_mint\":0,\"total_minted\":0,\"min_lock\":0,\"apr\":0,\"owner_id\":\"\"}}"
	wrongGlobalNodeJson10 = "{\"min_lock_period\":\"10\",\"simple_global_node\":{\"max_mint\":0,\"total_minted\":0,\"min_lock\":0,\"apr\":0,\"owner_id\":\"\"}}"
)

var (
	gnHash0  = []byte{}
	gnHash10 = []byte{}

	gnHash0Hex string
)

func init() {
	var hash0 = sha3.New256()
	hash0.Write([]byte(globalNodeJson0))
	var buf0 []byte
	gnHash0 = hash0.Sum(buf0)

	var hash10 = sha3.New256()
	hash10.Write([]byte(globalNodeJson10))
	var buf10 []byte
	gnHash10 = hash10.Sum(buf10)

	gnHash0Hex = util.ToHex(gnHash0)
}

func Test_newGlobalNode(t *testing.T) {
	tests := []struct {
		name string
		want *GlobalNode
	}{
		{
			name: "new_empty_global_node",
			want: &GlobalNode{
				ID:               ADDRESS,
				SimpleGlobalNode: &SimpleGlobalNode{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newGlobalNode(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newGlobalNode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGlobalNode_Encode(t *testing.T) {
	type fields struct {
		ID               datastore.Key
		SimpleGlobalNode *SimpleGlobalNode
		MinLockPeriod    time.Duration
	}
	tests := []struct {
		name   string
		fields fields
		want   []byte
	}{
		{
			name: "encoding_globalNode",
			fields: fields{
				ID:               ADDRESS,
				SimpleGlobalNode: &SimpleGlobalNode{},
				MinLockPeriod:    0,
			},
			want: newGlobalNode().Encode(),
		},
		{
			name: "encoding_globalNode_string",
			fields: fields{
				ID:               "",
				SimpleGlobalNode: &SimpleGlobalNode{},
				MinLockPeriod:    0,
			},
			want: []byte(globalNodeJson0),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gn := &GlobalNode{
				ID:               tt.fields.ID,
				SimpleGlobalNode: tt.fields.SimpleGlobalNode,
				MinLockPeriod:    tt.fields.MinLockPeriod,
			}
			if got := gn.Encode(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Encode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGlobalNode_Decode(t *testing.T) {
	type fields struct {
		ID               datastore.Key
		SimpleGlobalNode *SimpleGlobalNode
		MinLockPeriod    time.Duration
	}
	type args struct {
		input []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "decode input data without min_lock_period value",
			fields: fields{
				ID:               "",
				SimpleGlobalNode: &SimpleGlobalNode{},
				MinLockPeriod:    0,
			},
			args:    args{input: []byte(globalNodeJson0)},
			wantErr: false,
		},
		{
			name: "decode input data with min_lock_period value",
			fields: fields{
				ID:               "",
				SimpleGlobalNode: &SimpleGlobalNode{},
				MinLockPeriod:    10,
			},
			args:    args{input: []byte(globalNodeJson10)},
			wantErr: false,
		},
		{
			name: "decode invalid input data",
			fields: fields{
				ID:               "",
				SimpleGlobalNode: &SimpleGlobalNode{},
				MinLockPeriod:    10,
			},
			args:    args{input: []byte(wrongGlobalNodeJson10)},
			wantErr: true,
		},
		{
			name: "decode invalid input data",
			fields: fields{
				ID:               "",
				SimpleGlobalNode: &SimpleGlobalNode{},
				MinLockPeriod:    10,
			},
			args:    args{input: []byte("{t:0}")},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gn := &GlobalNode{
				ID:               tt.fields.ID,
				SimpleGlobalNode: tt.fields.SimpleGlobalNode,
				MinLockPeriod:    tt.fields.MinLockPeriod,
			}
			if err := gn.Decode(tt.args.input); (err != nil) != tt.wantErr {
				t.Errorf("Decode() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGlobalNode_getKey(t *testing.T) {
	type fields struct {
		ID               datastore.Key
		SimpleGlobalNode *SimpleGlobalNode
		MinLockPeriod    time.Duration
	}
	tests := []struct {
		name   string
		fields fields
		want   datastore.Key
	}{
		{
			name: "wrong_name",
			fields: fields{
				ID:               "ABC",
				SimpleGlobalNode: nil,
				MinLockPeriod:    0,
			},
			want: "ABCABC",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gn := &GlobalNode{
				ID:               tt.fields.ID,
				SimpleGlobalNode: tt.fields.SimpleGlobalNode,
				MinLockPeriod:    tt.fields.MinLockPeriod,
			}
			if got := gn.getKey(); got != tt.want {
				t.Errorf("getKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGlobalNode_GetHashBytes(t *testing.T) {
	type fields struct {
		ID               datastore.Key
		SimpleGlobalNode *SimpleGlobalNode
		MinLockPeriod    time.Duration
	}
	tests := []struct {
		name   string
		fields fields
		want   []byte
	}{
		{
			name: "getting hash of global node when MinLockPeriod=10",
			fields: fields{
				ID:               ADDRESS,
				SimpleGlobalNode: &SimpleGlobalNode{},
				MinLockPeriod:    10 * time.Second,
			},
			want: gnHash10,
		},
		{
			name: "getting hash of global node when MinLockPeriod=0",
			fields: fields{
				ID:               ADDRESS,
				SimpleGlobalNode: &SimpleGlobalNode{},
				MinLockPeriod:    0 * time.Second,
			},
			want: gnHash0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gn := &GlobalNode{
				ID:               tt.fields.ID,
				SimpleGlobalNode: tt.fields.SimpleGlobalNode,
				MinLockPeriod:    tt.fields.MinLockPeriod,
			}
			if got := gn.GetHashBytes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetHashBytes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGlobalNode_GetHash(t *testing.T) {
	type fields struct {
		ID               datastore.Key
		SimpleGlobalNode *SimpleGlobalNode
		MinLockPeriod    time.Duration
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "get_hash",
			fields: fields{
				ID:               "",
				SimpleGlobalNode: &SimpleGlobalNode{},
				MinLockPeriod:    0,
			},
			want: gnHash0Hex,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gn := &GlobalNode{
				ID:               tt.fields.ID,
				SimpleGlobalNode: tt.fields.SimpleGlobalNode,
				MinLockPeriod:    tt.fields.MinLockPeriod,
			}
			if got := gn.GetHash(); got != tt.want {
				t.Errorf("GetHash() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGlobalNode_canMint(t *testing.T) {
	type fields struct {
		ID               datastore.Key
		SimpleGlobalNode *SimpleGlobalNode
		MinLockPeriod    time.Duration
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "can't mint",
			fields: fields{
				ID: "",
				SimpleGlobalNode: &SimpleGlobalNode{
					MaxMint:     15,
					TotalMinted: 10,
					MinLock:     0,
					APR:         0,
				},
				MinLockPeriod: 0,
			},
			want: true,
		},
		{
			name: "can mint",
			fields: fields{
				ID: "",
				SimpleGlobalNode: &SimpleGlobalNode{
					MaxMint:     10,
					TotalMinted: 51,
					MinLock:     0,
					APR:         0,
				},
				MinLockPeriod: 0,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gn := &GlobalNode{
				ID:               tt.fields.ID,
				SimpleGlobalNode: tt.fields.SimpleGlobalNode,
				MinLockPeriod:    tt.fields.MinLockPeriod,
			}
			if got := gn.canMint(); got != tt.want {
				t.Errorf("canMint() = %v, want %v", got, tt.want)
			}
		})
	}
}
