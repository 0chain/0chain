module 0chain.net/smartcontract/multisigsc/test

replace 0chain.net/core => ../../../core

replace 0chain.net/mocks => ../../../mocks

replace 0chain.net/chaincore => ../../../chaincore

replace 0chain.net/smartcontract => ../../../smartcontract

replace 0chain.net/conductor => ../../../conductor

require (
	0chain.net/chaincore v0.0.0
	0chain.net/core v0.0.0
	0chain.net/smartcontract v0.0.0
	github.com/armon/consul-api v0.0.0-20180202201655-eb2c6b5be1b6 // indirect
	github.com/coreos/go-etcd v2.0.0+incompatible // indirect
	github.com/spf13/viper v1.7.0
	github.com/ugorji/go/codec v0.0.0-20181204163529-d75b2dcb6bc8 // indirect
	github.com/xordataexchange/crypt v0.0.3-0.20170626215501-b2862e3d0a77 // indirect
	go.uber.org/zap v1.10.0
)

go 1.13
