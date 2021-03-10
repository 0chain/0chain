module 0chain.net/mocks

replace 0chain.net/core => ../core

replace 0chain.net/mocks => ../mocks

replace 0chain.net/smartcontract => ../smartcontract

replace 0chain.net/chaincore => ../chaincore

replace 0chain.net/conductor => ../conductor

go 1.13

require (
	0chain.net/core v0.0.0
	github.com/gocql/gocql v0.0.0-20190423091413-b99afaf3b163
	github.com/stretchr/testify v1.4.0
)
