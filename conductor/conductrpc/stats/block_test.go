package stats

import (
	"reflect"
	"testing"
)

func TestBlockRequests_GetByHash(t *testing.T) {
	blockRequest := mockBlockRequest()
	blockRequests := NewBlockRequests()
	blockRequests.Add(blockRequest)

	type args struct {
		hash string
	}
	tests := []struct {
		name string
		br   *BlockRequests
		args args
		want *BlockRequest
	}{
		{
			name: "OK",
			br:   blockRequests,
			args: args{
				hash: blockRequest.Hash,
			},
			want: blockRequest,
		},
		{
			name: "Not_Found_NIL",
			br:   blockRequests,
			args: args{
				hash: "unknown hash",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.br.GetByHash(tt.args.hash); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetByHash() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBlockRequests_GetByHashOrRound(t *testing.T) {
	onlyHashList := NewBlockRequests()
	onlyHashRequest := mockBlockRequest()
	onlyHashRequest.Round = 0
	onlyHashList.Add(onlyHashRequest)

	onlyRoundList := NewBlockRequests()
	onlyRoundRequest := mockBlockRequest()
	onlyRoundRequest.Hash = ""
	onlyRoundList.Add(onlyRoundRequest)

	hashAndRoundList := NewBlockRequests()
	hashAndRoundRequest := mockBlockRequest()
	hashAndRoundList.Add(hashAndRoundRequest)

	type args struct {
		hash  string
		round int
	}
	tests := []struct {
		name string
		br   *BlockRequests
		args args
		want *BlockRequest
	}{
		{
			name: "Only_Hash_OK",
			br:   onlyHashList,
			args: args{
				hash: onlyHashRequest.Hash,
			},
			want: onlyHashRequest,
		},
		{
			name: "Only_Round_OK",
			br:   onlyRoundList,
			args: args{
				round: onlyRoundRequest.Round,
			},
			want: onlyRoundRequest,
		},
		{
			name: "Hash_And_Round_OK",
			br:   hashAndRoundList,
			args: args{
				hash:  hashAndRoundRequest.Hash,
				round: hashAndRoundRequest.Round,
			},
			want: hashAndRoundRequest,
		},
		{
			name: "Not_Found_NIL",
			br:   NewBlockRequests(),
			args: args{
				hash: "unknown hash",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.br.GetByHashOrRound(tt.args.hash, tt.args.round); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetByHashOrRound() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBlockRequests_GetBySenderIDAndHash(t *testing.T) {
	blockRequest := mockBlockRequest()
	blockRequests := NewBlockRequests()
	blockRequests.Add(blockRequest)

	type args struct {
		senderID string
		hash     string
	}
	tests := []struct {
		name string
		br   *BlockRequests
		args args
		want *BlockRequest
	}{
		{
			name: "OK",
			br:   blockRequests,
			args: args{
				senderID: blockRequest.SenderID,
				hash:     blockRequest.Hash,
			},
			want: blockRequest,
		},
		{
			name: "Not_Found_NIL",
			br:   blockRequests,
			args: args{
				hash: "unknown hash",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.br.GetBySenderIDAndHash(tt.args.senderID, tt.args.hash); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetBySenderIDAndHash() = %v, want %v", got, tt.want)
			}
		})
	}
}
