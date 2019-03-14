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
	github.com/spf13/afero v1.2.1 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect

	github.com/spf13/viper v1.3.1
	github.com/stretchr/testify v1.3.0 // indirect
	go.uber.org/zap v1.9.1
	golang.org/x/net v0.0.0-20190213061140-3a22650c66bd // indirect
	golang.org/x/sys v0.0.0-20190306171555-70f529850638 // indirect
	gopkg.in/check.v1 v1.0.0-20180628173108-788fd7840127 // indirect
)
