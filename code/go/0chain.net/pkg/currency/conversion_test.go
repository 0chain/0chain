package currency

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCoin_Float64(t *testing.T) {
	tests := []struct {
		name string
		c    Coin
		want float64
	}{
		{
			name: "coin to float64",
			c:    Coin(12),
			want: 12.0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.c.Float64(); got != tt.want {
				require.Equal(t, tt.want, got)
			}
		})
	}
}

func TestCoin_Int64(t *testing.T) {
	tests := []struct {
		name string
		c    Coin
		want int64
	}{
		{
			name: "coin to int64",
			c:    Coin(12),
			want: 12,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.c.Int64(); got != tt.want {
				require.Equal(t, tt.want, got)
			}
		})
	}
}

func TestCoin_ToZCN(t *testing.T) {
	tests := []struct {
		name string
		c    Coin
		want float64
	}{
		{
			name: "less than 10 digits",
			c:    Coin(12),
			want: .0000000012,
		},
		{
			name: "more than 10 digits",
			c:    Coin(1285674575869698),
			want: 128567.4575869698,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.c.ToZCN(); got != tt.want {
				require.Equal(t, tt.want, got)
			}
		})
	}
}

func TestParseZCN(t *testing.T) {
	type args struct {
		z float64
	}
	tests := []struct {
		name    string
		args    args
		want    Coin
		wantErr bool
	}{
		{
			name: "less than 10 decimal places",
			args: args{z: 1.2},
			want: 12000000000,
		},
		{
			name:    "more than 10 decimal places",
			args:    args{z: 1.23465934064596734},
			want:    0,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseZCN(tt.args.z)
			require.Equal(t, tt.wantErr, err != nil)
			require.Equal(t, tt.want, got)
		})
	}
}
