module 0chain.net/sharder

replace 0chain.net/core => ../core

replace 0chain.net/chaincore => ../chaincore

replace 0chain.net/smartcontract => ../smartcontract

require (
	0chain.net/chaincore v0.0.0
	0chain.net/core v0.0.0
	0chain.net/smartcontract v0.0.0
	github.com/didip/tollbooth v4.0.0+incompatible // indirect
	github.com/gocql/gocql v0.0.0-20190204224311-252acab79f98 // indirect
	github.com/golang/snappy v0.0.0-20180518054509-2e65f85255db // indirect
	github.com/gomodule/redigo v2.0.0+incompatible // indirect
	github.com/hashicorp/golang-lru v0.5.0 // indirect
	github.com/herumi/bls v0.0.0-20190127071025-2423de33a66f // indirect
	github.com/koding/cache v0.0.0-20161222233015-e8a81b0b3f20 // indirect
	github.com/patrickmn/go-cache v2.1.0+incompatible // indirect
	github.com/rcrowley/go-metrics v0.0.0-20181016184325-3113b8401b8a
	github.com/valyala/gozstd v1.2.1 // indirect
	github.com/vmihailenco/msgpack v4.0.2+incompatible // indirect
	go.uber.org/zap v1.9.1
	golang.org/x/crypto v0.0.0-20190131182504-b8fe1690c613 // indirect
	golang.org/x/time v0.0.0-20181108054448-85acf8d2951c // indirect
	gopkg.in/mgo.v2 v2.0.0-20180705113604-9856a29383ce // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.0 // indirect
)
