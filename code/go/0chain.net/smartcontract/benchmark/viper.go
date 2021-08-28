package benchmark

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func getViper(b *testing.B, path string) *viper.Viper {
	var vi = viper.New()
	vi.SetConfigType("yaml")
	vi.SetConfigName("benchmark")
	vi.AddConfigPath("./testdata/")
	require.NoError(b, vi.ReadInConfig())
	return vi
}
