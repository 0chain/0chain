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

func TestParseMZCN(t *testing.T) {
	type args struct {
		c float64
	}
	tests := []struct {
		name string
		args args
		want Coin
	}{
		{
			name: "less than 7 decimal",
			args: args{c: 1.12},
			want: Coin(11200000),
		},
		{
			name: "more than 7 decimal",
			args: args{c: 1.126854884758765},
			want: Coin(11268548),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseMZCN(tt.args.c); got != tt.want {
				require.Equal(t, tt.want, got)
			}
		})
	}
}

func TestParseUZCN(t *testing.T) {
	type args struct {
		c float64
	}
	tests := []struct {
		name string
		args args
		want Coin
	}{
		{
			name: "less than 4 decimal",
			args: args{c: 1.12},
			want: Coin(11200),
		},
		{
			name: "more than 4 decimal",
			args: args{c: 1.12868578},
			want: Coin(11286),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseUZCN(tt.args.c); got != tt.want {
				require.Equal(t, tt.want, got)
			}
		})
	}
}

func TestParseZCN(t *testing.T) {
	type args struct {
		c float64
	}
	tests := []struct {
		name string
		args args
		want Coin
	}{
		{
			name: "less than 10 decimal",
			args: args{c: 1.12},
			want: Coin(11200000000),
		},
		{
			name: "more than 10 decimal",
			args: args{c: 1.1211119076897},
			want: Coin(11211119076),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseZCN(tt.args.c); got != tt.want {
				require.Equal(t, tt.want, got)
			}
		})
	}
}
