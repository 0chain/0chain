package cmd

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func GetViper(b *testing.B, path string) *viper.Viper {
	var vi = viper.New()
	vi.SetConfigType("yaml")
	vi.SetConfigName("benchmark")
	vi.AddConfigPath("../config/")
	vi.AddConfigPath("./testdata/")
	vi.AddConfigPath("./config/")
	vi.AddConfigPath(".")
	vi.AddConfigPath("..")
	require.NoError(b, vi.ReadInConfig())
	return vi
}
