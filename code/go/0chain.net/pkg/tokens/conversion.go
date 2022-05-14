package tokens

import "github.com/shopspring/decimal"

func ZCNToSAS(f float64) int64 {
	dInt := decimal.NewFromFloat(f).Round(10).Mul(decimal.NewFromInt(zcn)).IntPart()
	return dInt
}

func SASToZCN(s int64) float64 {
	val, _ := decimal.NewFromInt(s).Div(decimal.NewFromInt(zcn)).Round(10).Float64()
	return val
}
