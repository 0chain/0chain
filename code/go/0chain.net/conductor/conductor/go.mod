module conductor

replace 0chain.net/conductor => ../../conductor

replace 0chain.net/mocks => ../../mocks

replace 0chain.net/chaincore => ../../chaincore

replace 0chain.net/core => ../../core

replace 0chain.net/smartcontract => ../../smartcontract

go 1.14

require (
	0chain.net/conductor v0.0.0-00010101000000-000000000000
	github.com/kr/pretty v0.2.0
	gopkg.in/yaml.v2 v2.3.0
)
