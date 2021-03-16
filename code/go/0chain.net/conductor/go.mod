module 0chain.net/conductor

go 1.14

replace 0chain.net/core => ../core

replace 0chain.net/mocks => ../mocks

replace 0chain.net/chaincore => ../chaincore

replace 0chain.net/smartcontract => ../smartcontract

replace 0chain.net/conductor => ./

require (
	0chain.net/chaincore v0.0.0
	0chain.net/core v0.0.0
	github.com/mitchellh/mapstructure v1.3.1
	github.com/spf13/viper v1.7.0
	github.com/valyala/gorpc v0.0.0-20160519171614-908281bef774
)
