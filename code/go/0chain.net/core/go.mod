module 0chain.net/core

go 1.14

replace 0chain.net/core => ../core

replace 0chain.net/mocks => ../mocks

replace 0chain.net/smartcontract => ../smartcontract

replace 0chain.net/chaincore => ../chaincore

replace 0chain.net/conductor => ../conductor

require (
	0chain.net/chaincore v0.0.0
	0chain.net/mocks v0.0.0
	github.com/0chain/gorocksdb v0.0.0-20181010114359-8752a9433481
	github.com/alicebob/miniredis/v2 v2.14.3
	github.com/didip/tollbooth v4.0.0+incompatible
	github.com/gocql/gocql v0.0.0-20190423091413-b99afaf3b163
	github.com/golang/snappy v0.0.1
	github.com/gomodule/redigo v2.0.0+incompatible
	github.com/hashicorp/golang-lru v0.5.1
	github.com/herumi/bls v0.0.0-20190423083323-d414f74643cb
	github.com/herumi/mcl v0.0.0-20190422075523-7f408a29acdc
	github.com/koding/cache v0.0.0-20161222233018-4a3175c6b2fe
	github.com/spf13/viper v1.7.0
	github.com/stretchr/testify v1.4.0
	github.com/valyala/gozstd v1.4.1
	github.com/vmihailenco/msgpack v4.0.4+incompatible
	go.uber.org/atomic v1.4.0
	go.uber.org/zap v1.10.0
	golang.org/x/crypto v0.0.0-20200323165209-0ec3e9974c59
	golang.org/x/exp v0.0.0-20191030013958-a1ab85dbe136
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
)
