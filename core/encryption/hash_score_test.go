package encryption

import (
	"reflect"
	"testing"
)

func TestNewXORHashScorer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want *XORHashScorer
	}{
		{
			name: "Test_NewXORHashScorer_OK",
			want: &XORHashScorer{},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := NewXORHashScorer(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewXORHashScorer() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestXORHashScorer_Score(t *testing.T) {
	t.Parallel()

	type args struct {
		hash1 []byte
		hash2 []byte
	}
	tests := []struct {
		name string
		args args
		want int32
	}{
		{
			name: "Test_XORHashScorer_Score_OK",
			args: args{hash1: []byte("hash1"), hash2: []byte("hash2")},
			want: 2,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			xor := &XORHashScorer{}
			if got := xor.Score(tt.args.hash1, tt.args.hash2); got != tt.want {
				t.Errorf("Score() = %v, want %v", got, tt.want)
			}
		})
	}
}
