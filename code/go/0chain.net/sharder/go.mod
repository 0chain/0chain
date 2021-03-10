module 0chain.net/sharder

go 1.14

replace 0chain.net/core => ../core

replace 0chain.net/mocks => ../mocks

replace 0chain.net/chaincore => ../chaincore

replace 0chain.net/smartcontract => ../smartcontract

replace 0chain.net/conductor => ../conductor

require (
	0chain.net/chaincore v0.0.0
	0chain.net/core v0.0.0
	0chain.net/smartcontract v0.0.0
	github.com/0chain/gorocksdb v0.0.0-20181010114359-8752a9433481
	github.com/go-ini/ini v1.55.0 // indirect
	github.com/minio/minio-go v6.0.14+incompatible
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/rcrowley/go-metrics v0.0.0-20181016184325-3113b8401b8a
	github.com/remeh/sizedwaitgroup v0.0.0-20180822144253-5e7302b12cce
	github.com/spf13/viper v1.7.0
	go.uber.org/zap v1.10.0
)
