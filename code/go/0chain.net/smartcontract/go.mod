module 0chain.net/smartcontract

replace 0chain.net/core => ../core

replace 0chain.net/chaincore => ../chaincore

replace 0chain.net/smartcontract => ../smartcontract

replace 0chain.net/conductor => ../conductor

require (
	0chain.net/chaincore v0.0.0
	0chain.net/conductor v0.0.0-00010101000000-000000000000
	0chain.net/core v0.0.0
	github.com/asaskevich/govalidator v0.0.0-20180720115003-f9ffefc3facf
	github.com/rcrowley/go-metrics v0.0.0-20181016184325-3113b8401b8a
	github.com/spf13/viper v1.3.2
	github.com/stretchr/testify v1.3.0
	github.com/valyala/gorpc v0.0.0-20160519171614-908281bef774 // indirect
	go.uber.org/zap v1.9.1
	golang.org/x/crypto v0.0.0-20200323165209-0ec3e9974c59 // indirect
)

go 1.13
