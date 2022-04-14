module 0chain.net

go 1.16

require (
	github.com/0chain/gorocksdb v0.0.0-20220125141924-9721107d4a29
	github.com/alicebob/miniredis/v2 v2.14.3
	github.com/bitly/go-hostpool v0.0.0-20171023180738-a3a6125de932 // indirect
	github.com/bmizerany/assert v0.0.0-20160611221934-b7ed37b82869 // indirect
	github.com/didip/tollbooth v4.0.0+incompatible
	github.com/facebookgo/ensure v0.0.0-20160127193407-b4ab57deab51 // indirect
	github.com/facebookgo/stack v0.0.0-20160209184415-751773369052 // indirect
	github.com/facebookgo/subset v0.0.0-20150612182917-8dac2c3c4870 // indirect
	github.com/go-ini/ini v1.55.0 // indirect
	github.com/go-playground/validator/v10 v10.10.1
	github.com/gocql/gocql v0.0.0-20190423091413-b99afaf3b163
	github.com/golang/snappy v0.0.4
	github.com/gomodule/redigo v2.0.0+incompatible
	github.com/guregu/null v4.0.0+incompatible
	github.com/hashicorp/golang-lru v0.5.4
	github.com/herumi/bls v0.0.0-20210511012341-3f3850a6eac7
	github.com/herumi/mcl v0.0.0-20210601112215-5faedff92a72
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/koding/cache v0.0.0-20161222233018-4a3175c6b2fe
	github.com/minio/minio-go v6.0.14+incompatible
	github.com/mitchellh/mapstructure v1.4.3
	github.com/patrickmn/go-cache v2.1.0+incompatible // indirect
	github.com/pkg/errors v0.9.1
	github.com/rcrowley/go-metrics v0.0.0-20181016184325-3113b8401b8a
	github.com/remeh/sizedwaitgroup v0.0.0-20180822144253-5e7302b12cce
	github.com/selvatico/go-mocket v1.0.7
	github.com/spf13/cast v1.4.1
	github.com/spf13/cobra v1.2.1
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.11.0
	github.com/stretchr/objx v0.3.0 // indirect
	github.com/stretchr/testify v1.7.1
	github.com/tinylib/msgp v1.1.6
	github.com/valyala/gozstd v1.14.1
	github.com/vmihailenco/msgpack/v5 v5.3.5
	go.uber.org/atomic v1.9.0
	go.uber.org/multierr v1.8.0 // indirect
	go.uber.org/zap v1.21.0
	golang.org/x/crypto v0.0.0-20220411220226-7b82a4e95df4
	golang.org/x/net v0.0.0-20220412020605-290c469a71a5
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
	gopkg.in/yaml.v2 v2.4.0
	gorm.io/driver/postgres v1.3.4
	gorm.io/gorm v1.23.4
)

replace github.com/tinylib/msgp => github.com/0chain/msgp v1.1.62
