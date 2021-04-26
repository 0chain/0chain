module sharder

replace 0chain.net/core => ../../core

replace 0chain.net/chaincore => ../../chaincore

replace 0chain.net/smartcontract => ../../smartcontract

replace 0chain.net/sharder => ../../sharder

replace 0chain.net/conductor => ../../conductor

// replace 0chain.net/conductor/conductrpc => ../../conductor/conductrpc

require (
	0chain.net/chaincore v0.0.0
	0chain.net/conductor v0.0.0-00010101000000-000000000000
	0chain.net/core v0.0.0
	0chain.net/sharder v0.0.0
	0chain.net/smartcontract v0.0.0
	github.com/spf13/viper v1.7.0
	go.uber.org/zap v1.10.0
)

go 1.13
