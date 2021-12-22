package stats

import (
	"reflect"
	"testing"
)

func TestBlockInfos_ContainsHashOrRound(t *testing.T) {
	hash, round, path := "d0cab02dd0f094eaa2d136fa335d4fbb7858832caebc416982187b2c9b58cecc", 5, "path"

	hashBI := BlockInfo{
		Hash: hash,
	}
	roundBI := BlockInfo{
		Round: round,
	}
	hashAndRoundBI := BlockInfo{
		Hash:  hash,
		Round: round,
	}

	type args struct {
		hash  string
		round int
	}
	tests := []struct {
		name  string
		bi    BlockInfos
		args  args
		want  bool
		want1 BlockReport
	}{
		{
			name: "ContainsHash_TRUE",
			bi: BlockInfos{
				path: map[BlockInfo]int{
					hashBI: 1,
				},
			},
			args: args{
				hash: hash,
			},
			want: true,
			want1: BlockReport{
				BlockInfo: hashBI,
				Handler:   path,
			},
		},
		{
			name: "ContainsRound_TRUE",
			bi: BlockInfos{
				path: map[BlockInfo]int{
					roundBI: 1,
				},
			},
			args: args{
				round: round,
			},
			want: true,
			want1: BlockReport{
				BlockInfo: roundBI,
				Handler:   path,
			},
		},
		{
			name: "ContainsHashAndRound_TRUE",
			bi: BlockInfos{
				path: map[BlockInfo]int{
					hashAndRoundBI: 1,
				},
			},
			args: args{
				hash:  hash,
				round: round,
			},
			want: true,
			want1: BlockReport{
				BlockInfo: hashAndRoundBI,
				Handler:   path,
			},
		},
		{
			name: "FALSE",
			bi: BlockInfos{
				path: map[BlockInfo]int{
					hashAndRoundBI: 1,
				},
			},
			args: args{
				hash:  "unknown hash",
				round: round + 1,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := tt.bi.ContainsHashOrRound(tt.args.hash, tt.args.round)
			if got != tt.want {
				t.Errorf("ContainsHashOrRound() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("ContainsHashOrRound() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
