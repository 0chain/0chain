module 0chain.net/smartcontract

replace 0chain.net/core => ../core

replace 0chain.net/chaincore => ../chaincore

replace 0chain.net/smartcontract => ../smartcontract

require (
	0chain.net/chaincore v0.0.0
	0chain.net/core v0.0.0
	github.com/0chain/gorocksdb v0.0.0-20181010114359-8752a9433481
	github.com/asaskevich/govalidator v0.0.0-20180720115003-f9ffefc3facf
	go.uber.org/zap v1.9.1
)
