package cmd

import (
	"github.com/spf13/viper"
)

func GetViper(path string) *viper.Viper {
	var vi = viper.New()
	vi.SetConfigType("yaml")
	vi.SetConfigName("benchmark")
	vi.AddConfigPath("../config/")
	vi.AddConfigPath("./testdata/")
	vi.AddConfigPath("./config/")
	vi.AddConfigPath(".")
	vi.AddConfigPath("..")
	err := vi.ReadInConfig()
	if err != nil {
		panic(err)
	}
	return vi
}
