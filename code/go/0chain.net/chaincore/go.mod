module 0chain.net/chaincore

replace 0chain.net/core => ../core

replace 0chain.net/smartcontract => ../smartcontract

replace 0chain.net/chaincore => ../chaincore

require (
	0chain.net/core v0.0.0
	0chain.net/smartcontract v0.0.0
	github.com/0chain/gorocksdb v0.0.0-20181010114359-8752a9433481
	github.com/herumi/bls v0.0.0-20190423083323-d414f74643cb
	github.com/rcrowley/go-metrics v0.0.0-20181016184325-3113b8401b8a
	github.com/remeh/sizedwaitgroup v0.0.0-20180822144253-5e7302b12cce
	github.com/spf13/viper v1.3.2
	github.com/stretchr/testify v1.3.0
	go.uber.org/zap v1.9.1
	gopkg.in/yaml.v2 v2.2.2
)
