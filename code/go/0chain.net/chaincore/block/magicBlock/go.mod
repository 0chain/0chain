module magicBlock

replace 0chain.net/chaincore => ../../../chaincore

replace 0chain.net/mocks => ../../../mocks

replace 0chain.net/core => ../../../core

replace 0chain.net/smartcontract => ../../../smartcontract

replace 0chain.net/conductor => ../../../conductor

require (
	0chain.net/chaincore v0.0.0
	0chain.net/core v0.0.0
	gopkg.in/yaml.v2 v2.2.4
)

go 1.13
