package tokens

//go:generate msgp -io=false -tests=false -v
//SAS - any quantity that is represented as an integer in the lowest denomination
type SAS int64

const (
	ZCN  SAS = 1e10
	mZCN SAS = 1e7
	uZCN SAS = 1e4
)
