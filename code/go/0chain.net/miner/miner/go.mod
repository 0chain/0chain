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
	github.com/gopherjs/gopherjs v0.0.0-20181103185306-d547d1d9531e // indirect
	github.com/kisielk/gotool v1.0.0 // indirect
	github.com/neelance/astrewrite v0.0.0-20160511093645-99348263ae86 // indirect
	github.com/neelance/sourcemap v0.0.0-20151028013722-8c68805598ab // indirect
	github.com/shurcooL/httpfs v0.0.0-20181222201310-74dc9339e414 // indirect
	github.com/spf13/cobra v0.0.3 // indirect

	github.com/spf13/viper v1.3.1
	go.uber.org/zap v1.9.1
	golang.org/x/tools v0.0.0-20190304215341-0f64db555a9c // indirect
)
