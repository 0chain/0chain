module magicBlock

replace 0chain.net/chaincore => ../../../chaincore

replace 0chain.net/core => ../../../core

replace 0chain.net/smartcontract => ../../../smartcontract

replace 0chain.net/conductor => ../../../conductor

require (
	gopkg.in/yaml.v2 v2.2.2
)

go 1.13
