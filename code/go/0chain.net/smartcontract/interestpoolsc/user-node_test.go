package interestpoolsc

import (
	"reflect"
	"testing"

	"0chain.net/chaincore/tokenpool"
	"0chain.net/core/datastore"
)

var (
	encodedUserNodeWithClient1 = []byte{
		123, 34, 99, 108, 105, 101, 110, 116, 95, 105, 100, 34, 58, 34, 99, 108, 105, 101,
		110, 116, 95, 49, 34, 44, 34, 112, 111, 111, 108, 115, 34, 58, 123, 125, 125,
	}
	encryptedHash = []byte{
		6, 33, 201, 65, 143, 172, 147, 94, 217, 74, 109, 166, 255, 159, 140,
		46, 45, 231, 217, 67, 191, 16, 11, 174, 137, 96, 1, 114, 7, 125, 75, 235,
	}
)

const (
	encodedUserNodeWithClient1String             = "{\"client_id\":\"client_1\",\"pools\":{}}"
	encodedUserNodeWithClient1StringWthPool      = "{\"client_id\":\"client_1\"}"
	encodedUserNodeWithClient1StringWthCId       = "{\"client_id\":\"client_1\"}"
	encodedUserNodeWithClient1StringWthWrongPool = "{\"client_id\":\"client_1\",\"pools\":123}"

	wrongUserNodeData = "{\test\":123}"
	emptyUserNodeData = "{}"
)

func Test_newUserNode(t *testing.T) {
	type args struct {
		clientID datastore.Key
	}
	tests := []struct {
		name string
		args args
		want *UserNode
	}{
		{
			name: "new user node",
			args: args{clientID: clientID1},
			want: &UserNode{
				ClientID: clientID1,
				Pools:    make(map[datastore.Key]*interestPool),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newUserNode(tt.args.clientID); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newUserNode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUserNode_Encode(t *testing.T) {
	type fields struct {
		ClientID datastore.Key
		Pools    map[datastore.Key]*interestPool
	}
	tests := []struct {
		name   string
		fields fields
		want   []byte
	}{
		{
			name: "ok encoding",
			fields: fields{
				ClientID: clientID1,
				Pools:    make(map[datastore.Key]*interestPool),
			},
			want: encodedUserNodeWithClient1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			un := &UserNode{
				ClientID: tt.fields.ClientID,
				Pools:    tt.fields.Pools,
			}
			if got := un.Encode(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Encode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUserNode_Decode(t *testing.T) {
	type fields struct {
		ClientID datastore.Key
		Pools    map[datastore.Key]*interestPool
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
			name: "decode ok",
			fields: fields{
				ClientID: clientID1,
				Pools:    make(map[datastore.Key]*interestPool),
			},
			args:    args{input: encodedUserNodeWithClient1},
			wantErr: false,
		},
		{
			name: "decoding without pool",
			fields: fields{
				ClientID: clientID1,
				Pools:    make(map[datastore.Key]*interestPool),
			},
			args:    args{input: []byte(encodedUserNodeWithClient1StringWthPool)},
			wantErr: false,
		},
		{
			name: "decoding without clientid",
			fields: fields{
				ClientID: clientID1,
				Pools:    make(map[datastore.Key]*interestPool),
			},
			args:    args{input: []byte(encodedUserNodeWithClient1StringWthCId)},
			wantErr: false,
		},
		{
			name: "decoding Error : invalid character",
			fields: fields{
				ClientID: clientID1,
				Pools:    make(map[datastore.Key]*interestPool),
			},
			args:    args{input: []byte(wrongUserNodeData)},
			wantErr: true,
		},
		{
			name: "decoding Error : invalid character",
			fields: fields{
				ClientID: clientID1,
				Pools:    make(map[datastore.Key]*interestPool),
			},
			args:    args{input: []byte(emptyUserNodeData)},
			wantErr: false,
		},
		{
			name: "decoding Error :  cannot unmarshal",
			fields: fields{
				ClientID: clientID1,
				Pools:    make(map[datastore.Key]*interestPool),
			},
			args:    args{input: []byte(encodedUserNodeWithClient1StringWthWrongPool)},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			un := &UserNode{
				ClientID: tt.fields.ClientID,
				Pools:    tt.fields.Pools,
			}
			if err := un.Decode(tt.args.input); (err != nil) != tt.wantErr {
				t.Errorf("Decode() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUserNode_getKey(t *testing.T) {
	type fields struct {
		ClientID datastore.Key
		Pools    map[datastore.Key]*interestPool
	}
	type args struct {
		globalKey string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   datastore.Key
	}{
		{
			name: "getKey ok",
			fields: fields{
				ClientID: "123",
				Pools:    nil,
			},
			args: args{globalKey: "456"},
			want: datastore.Key("456123"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			un := &UserNode{
				ClientID: tt.fields.ClientID,
				Pools:    tt.fields.Pools,
			}
			if got := un.getKey(tt.args.globalKey); got != tt.want {
				t.Errorf("getKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUserNode_GetHashBytes(t *testing.T) {
	type fields struct {
		ClientID datastore.Key
		Pools    map[datastore.Key]*interestPool
	}
	tests := []struct {
		name   string
		fields fields
		want   []byte
	}{
		{
			name: "getHashBytes with pools",
			fields: fields{
				ClientID: "client_1",
				Pools:    map[datastore.Key]*interestPool{},
			},
			want: encryptedHash,
		},
		{
			name: "getHashBytes without pools",
			fields: fields{
				ClientID: "client_1",
			},
			want: []byte{108, 167, 13, 54, 213, 102, 209, 219, 69, 169, 228, 221,
				167, 210, 170, 114, 45, 105, 99, 118, 50, 55, 204, 44, 103, 202,
				78, 167, 173, 143, 11, 98},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			un := &UserNode{
				ClientID: tt.fields.ClientID,
				Pools:    tt.fields.Pools,
			}
			if got := un.GetHashBytes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetHashBytes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUserNode_GetHash(t *testing.T) {
	type fields struct {
		ClientID datastore.Key
		Pools    map[datastore.Key]*interestPool
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "GetHash with pools",
			fields: fields{
				ClientID: "client_1",
				Pools:    map[datastore.Key]*interestPool{},
			},
			want: "0621c9418fac935ed94a6da6ff9f8c2e2de7d943bf100bae89600172077d4beb",
		},
		{
			name: "GetHash without pools",
			fields: fields{
				ClientID: "client_1",
			},
			want: "6ca70d36d566d1db45a9e4dda7d2aa722d6963763237cc2c67ca4ea7ad8f0b62",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			un := &UserNode{
				ClientID: tt.fields.ClientID,
				Pools:    tt.fields.Pools,
			}
			if got := un.GetHash(); got != tt.want {
				t.Errorf("GetHash() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUserNode_hasPool(t *testing.T) {
	type fields struct {
		ClientID datastore.Key
		Pools    map[datastore.Key]*interestPool
	}
	type args struct {
		poolID datastore.Key
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "if has pool",
			fields: fields{
				ClientID: "client_1",
				Pools: map[datastore.Key]*interestPool{
					"client_1": &interestPool{},
				},
			},
			args: args{
				poolID: "client_1",
			},
			want: true,
		},
		{
			name: "if hasn't pool",
			fields: fields{
				ClientID: "client_1",
				Pools:    map[datastore.Key]*interestPool{},
			},
			args: args{
				poolID: "client_1",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			un := &UserNode{
				ClientID: tt.fields.ClientID,
				Pools:    tt.fields.Pools,
			}
			if got := un.hasPool(tt.args.poolID); got != tt.want {
				t.Errorf("hasPool() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUserNode_getPool(t *testing.T) {
	type fields struct {
		ClientID datastore.Key
		Pools    map[datastore.Key]*interestPool
	}
	type args struct {
		poolID datastore.Key
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *interestPool
	}{
		{
			name: "get pool when nil",
			fields: fields{
				ClientID: clientID1,
				Pools: map[datastore.Key]*interestPool{
					"client_id": nil,
				},
			},
			args: args{poolID: clientID1},
			want: nil,
		},
		{
			name: "get pool when has not pool",
			fields: fields{
				ClientID: clientID1,
				Pools:    map[datastore.Key]*interestPool{},
			},
			args: args{poolID: clientID1},
			want: nil,
		},
		{
			name: "get pool when has pool",
			fields: fields{
				ClientID: clientID1,
				Pools: map[datastore.Key]*interestPool{
					clientID1: &interestPool{
						ZcnLockingPool: nil,
						APR:            10,
						TokensEarned:   0,
					},
				},
			},
			args: args{poolID: clientID1},
			want: &interestPool{
				ZcnLockingPool: nil,
				APR:            10,
				TokensEarned:   0,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			un := &UserNode{
				ClientID: tt.fields.ClientID,
				Pools:    tt.fields.Pools,
			}
			if got := un.getPool(tt.args.poolID); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getPool() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUserNode_addPool(t *testing.T) {
	type fields struct {
		ClientID datastore.Key
		Pools    map[datastore.Key]*interestPool
	}
	type args struct {
		ip *interestPool
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "error : pool already exists",
			fields: fields{
				ClientID: clientID1,
				Pools: map[datastore.Key]*interestPool{
					name: &interestPool{
						ZcnLockingPool: &tokenpool.ZcnLockingPool{
							ZcnPool: tokenpool.ZcnPool{
								tokenpool.TokenPool{ID: name},
							},
							TokenLockInterface: nil,
						},
						APR:          10,
						TokensEarned: 0,
					},
				},
			},
			args: args{
				ip: &interestPool{
					ZcnLockingPool: &tokenpool.ZcnLockingPool{
						ZcnPool: tokenpool.ZcnPool{
							tokenpool.TokenPool{ID: name},
						},
						TokenLockInterface: nil,
					},
					APR:          10,
					TokensEarned: 0,
				},
			},
			wantErr: true,
		},
		{
			name: "add pool ok",
			fields: fields{
				ClientID: clientID1,
				Pools:    map[datastore.Key]*interestPool{},
			},
			args: args{ip: &interestPool{
				ZcnLockingPool: &tokenpool.ZcnLockingPool{
					ZcnPool: tokenpool.ZcnPool{
						tokenpool.TokenPool{ID: name},
					},
					TokenLockInterface: nil,
				},
				APR:          10,
				TokensEarned: 0,
			}},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			un := &UserNode{
				ClientID: tt.fields.ClientID,
				Pools:    tt.fields.Pools,
			}
			if err := un.addPool(tt.args.ip); (err != nil) != tt.wantErr {
				t.Errorf("addPool() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUserNode_deletePool(t *testing.T) {
	type fields struct {
		ClientID datastore.Key
		Pools    map[datastore.Key]*interestPool
	}
	type args struct {
		poolID datastore.Key
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "can't delete pool, pool doesnt exist",
			fields: fields{
				ClientID: clientID1,
				Pools:    map[datastore.Key]*interestPool{},
			},
			args:    args{poolID: name},
			wantErr: true,
		},
		{
			name: "deleting pool",
			fields: fields{
				ClientID: clientID1,
				Pools: map[datastore.Key]*interestPool{
					name: newInterestPool(),
				},
			},
			args:    args{poolID: name},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			un := &UserNode{
				ClientID: tt.fields.ClientID,
				Pools:    tt.fields.Pools,
			}
			if err := un.deletePool(tt.args.poolID); (err != nil) != tt.wantErr {
				t.Errorf("deletePool() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
