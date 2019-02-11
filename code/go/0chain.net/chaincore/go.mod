module 0chain.net/chaincore

replace 0chain.net/core => ../core

require (
	0chain.net/core v0.0.0
	github.com/0chain/gorocksdb v0.0.0-20181010114359-8752a9433481
	github.com/gomodule/redigo v2.0.0+incompatible // indirect
	github.com/hashicorp/golang-lru v0.5.0 // indirect
	github.com/koding/cache v0.0.0-20161222233015-e8a81b0b3f20 // indirect
	github.com/rcrowley/go-metrics v0.0.0-20181016184325-3113b8401b8a
	github.com/spf13/viper v1.3.1
	go.uber.org/zap v1.9.1
	golang.org/x/crypto v0.0.0-20190211182817-74369b46fc67 // indirect
	gopkg.in/mgo.v2 v2.0.0-20180705113604-9856a29383ce // indirect
	gopkg.in/yaml.v2 v2.2.2
)
