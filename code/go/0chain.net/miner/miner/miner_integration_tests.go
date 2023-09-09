//go:build integration_tests
// +build integration_tests

package main

import (
	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	crpc "0chain.net/conductor/conductrpc" // integration tests
)

// start lock, where the miner is ready to connect to blockchain (BC)
func initIntegrationsTests() {
	crpc.Init()
}

func registerInConductor(id string) {
	crpc.Client().Register(id)
	go syncCSConfig(id)
}

func shutdownIntegrationTests() {
	crpc.Shutdown()
}

func readMagicBlock(magicBlockConfig string) (*block.MagicBlock, error) {
	magicBlockFromConductor := crpc.Client().MagicBlock()

	if magicBlockFromConductor != "" {
		return chain.ReadMagicBlockFile(magicBlockFromConductor)
	}

	return chain.ReadMagicBlockFile(magicBlockConfig)
}


var latestVersion = 0
func syncCSConfig(id string) {
	for {
		config, err := crpc.Client().GetNodeConfig(id)
		if err != nil {
			logging.Logger.Warn("[conductor] failed synchronizing config", zap.String("node_id", id))
		}

		if config.Version == latestVersion {
			continue
		}

		latestVersion = config.Version

		for k, v := range config.Map {
			viper.Set(k, v)
			c := viper.Get(k)
			typ := "unknown"
			switch c.(type) {
			case string:
				typ = "string"
			case int, int64, int32, int16, int8:
				typ = "int"
			case float32, float64:
				typ = "float"
			}
			logging.Logger.Debug("ebrahim_debug: [conductor] after setting config", zap.String("key", k), zap.Any("set_value", v), zap.Any("cur_value", c), zap.String("type", typ))
		}
	}
}