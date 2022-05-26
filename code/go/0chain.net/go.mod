module 0chain.net

go 1.16

require (
	github.com/0chain/gorocksdb v0.0.0-20220406081817-640f6b0a3abb
	github.com/alicebob/miniredis/v2 v2.14.3
	github.com/didip/tollbooth v4.0.2+incompatible
	github.com/facebookgo/ensure v0.0.0-20160127193407-b4ab57deab51 // indirect
	github.com/facebookgo/stack v0.0.0-20160209184415-751773369052 // indirect
	github.com/facebookgo/subset v0.0.0-20150612182917-8dac2c3c4870 // indirect
	github.com/fsnotify/fsnotify v1.5.3 // indirect
	github.com/go-ini/ini v1.66.4 // indirect
	github.com/go-openapi/runtime v0.24.0
	github.com/go-playground/validator/v10 v10.10.1
	github.com/gocql/gocql v1.0.0
	github.com/golang/snappy v0.0.4
	github.com/gomodule/redigo v1.8.8
	github.com/guregu/null v4.0.0+incompatible
	github.com/hashicorp/golang-lru v0.5.4
	github.com/herumi/bls v0.0.0-20220327072144-7ec09c557eef
	github.com/herumi/mcl v0.0.0-20210601112215-5faedff92a72
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/koding/cache v0.0.0-20161222233018-4a3175c6b2fe
	github.com/minio/minio-go v6.0.14+incompatible
	github.com/mitchellh/mapstructure v1.5.0
	github.com/patrickmn/go-cache v2.1.0+incompatible // indirect
	github.com/pelletier/go-toml v1.9.5 // indirect
	github.com/pkg/errors v0.9.1
	github.com/rcrowley/go-metrics v0.0.0-20201227073835-cf1acfcdf475
	github.com/remeh/sizedwaitgroup v1.0.0
	github.com/selvatico/go-mocket v1.0.7
	github.com/shopspring/decimal v1.3.1 // indirect
	github.com/spf13/cast v1.4.1
	github.com/spf13/cobra v1.4.0
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.11.0
	github.com/stretchr/objx v0.3.0 // indirect
	github.com/stretchr/testify v1.7.1
	github.com/tinylib/msgp v1.1.6
	github.com/valyala/gozstd v1.16.0
	github.com/vmihailenco/msgpack/v5 v5.3.5
	go.uber.org/atomic v1.9.0
	go.uber.org/multierr v1.8.0 // indirect
	go.uber.org/zap v1.21.0
	golang.org/x/crypto v0.0.0-20220411220226-7b82a4e95df4
	golang.org/x/net v0.0.0-20220425223048-2871e0cb64e4
	golang.org/x/sys v0.0.0-20220422013727-9388b58f7150 // indirect
	golang.org/x/time v0.0.0-20220411224347-583f2d630306 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
	gopkg.in/yaml.v2 v2.4.0
	gorm.io/driver/postgres v1.3.5
	gorm.io/gorm v1.23.4
)

replace github.com/tinylib/msgp => github.com/0chain/msgp v1.1.62
