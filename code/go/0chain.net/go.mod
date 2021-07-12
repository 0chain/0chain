module 0chain.net

go 1.16

require (
	github.com/0chain/gorocksdb v0.0.0-20181010114359-8752a9433481
	github.com/0chain/gosdk v1.2.77-0.20210709165701-d434fe9194b3 // indirect
	github.com/alicebob/miniredis/v2 v2.15.1
	github.com/asaskevich/govalidator v0.0.0-20210307081110-f21760c49a8d
	github.com/bitly/go-hostpool v0.1.0 // indirect
	github.com/bmizerany/assert v0.0.0-20160611221934-b7ed37b82869 // indirect
	github.com/coreos/bbolt v1.3.2 // indirect
	github.com/coreos/etcd v3.3.13+incompatible // indirect
	github.com/coreos/pkg v0.0.0-20180928190104-399ea9e2e55f // indirect
	github.com/didip/tollbooth v4.0.2+incompatible
	github.com/facebookgo/ensure v0.0.0-20200202191622-63f1cf65ac4c // indirect
	github.com/facebookgo/stack v0.0.0-20160209184415-751773369052 // indirect
	github.com/facebookgo/subset v0.0.0-20200203212716-c811ad88dec4 // indirect
	github.com/go-ini/ini v1.62.0 // indirect
	github.com/gocql/gocql v0.0.0-20210707082121-9a3953d1826d
	github.com/golang/snappy v0.0.4
	github.com/go-playground/validator/v10 v10.6.1
	github.com/gomodule/redigo v2.0.0+incompatible
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/hashicorp/golang-lru v0.5.4
	github.com/herumi/bls v0.0.0-20210511012341-3f3850a6eac7
	github.com/herumi/mcl v0.0.0-20210601112215-5faedff92a72
	github.com/koding/cache v0.0.0-20161222233018-4a3175c6b2fe
	github.com/kr/pretty v0.2.1 // indirect
	github.com/leodido/go-urn v1.2.1 // indirect
	github.com/minio/minio-go v6.0.14+incompatible
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/mapstructure v1.4.1
	github.com/patrickmn/go-cache v2.1.0+incompatible // indirect
	github.com/pelletier/go-toml v1.9.3 // indirect
	github.com/prometheus/tsdb v0.7.1 // indirect
	github.com/rcrowley/go-metrics v0.0.0-20201227073835-cf1acfcdf475
	github.com/remeh/sizedwaitgroup v1.0.0
	github.com/spf13/afero v1.6.0 // indirect
	github.com/spf13/cast v1.3.1
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.8.1
	github.com/stretchr/testify v1.7.0
	github.com/tmc/grpc-websocket-proxy v0.0.0-20190109142713-0ad062ec5ee5 // indirect
	github.com/valyala/gozstd v1.11.0
	github.com/vmihailenco/msgpack v4.0.4+incompatible
	go.uber.org/atomic v1.8.0
	go.uber.org/zap v1.18.1
	golang.org/x/crypto v0.0.0-20210711020723-a769d52b0f97
	golang.org/x/exp v0.0.0-20210709195130-ecdcf02a369a
	golang.org/x/net v0.0.0-20210614182718-04defd469f4e // indirect
	golang.org/x/sys v0.0.0-20200202164722-d101bd2416d5 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
	gopkg.in/yaml.v2 v2.4.0
)

replace github.com/0chain/gosdk => ../../../../gosdk
