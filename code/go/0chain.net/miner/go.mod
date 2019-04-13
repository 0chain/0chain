module 0chain.net/miner

replace 0chain.net/core => ../core

replace 0chain.net/chaincore => ../chaincore

replace 0chain.net/smartcontract => ../smartcontract

require (
	0chain.net/chaincore v0.0.0
	0chain.net/core v0.0.0
	0chain.net/smartcontract v0.0.0
	github.com/rcrowley/go-metrics v0.0.0-20181016184325-3113b8401b8a
	github.com/spf13/viper v1.3.2
	go.uber.org/zap v1.9.1
)
