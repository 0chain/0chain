package currency

//go:generate msgp -io=false -tests=false -v
//Coin - any quantity that is represented as an integer in the lowest denomination
type Coin int64

const (
	SAS  Coin = 1
	ZCN       = 1e10 * SAS
	mZCN      = 1e7 * SAS
	uZCN      = 1e4 * SAS
)
