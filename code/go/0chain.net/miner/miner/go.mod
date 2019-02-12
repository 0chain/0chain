module miner

replace 0chain.net/core => ../../core

replace 0chain.net/chaincore => ../../chaincore

replace 0chain.net/smartcontract => ../../smartcontract

replace 0chain.net/miner => ../../miner

require (
	0chain.net/chaincore v0.0.0
	0chain.net/core v0.0.0
	0chain.net/miner v0.0.0
	0chain.net/smartcontract v0.0.0

	github.com/spf13/viper v1.3.1
	go.uber.org/zap v1.9.1
)
