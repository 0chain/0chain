package tokens

import "github.com/shopspring/decimal"

func ZCNToSAS(f float64) SAS {
	dInt := decimal.NewFromFloat(f).Round(10).Mul(decimal.NewFromInt(int64(ZCN))).IntPart()
	return SAS(dInt)
}

func SASToZCN(s SAS) float64 {
	val, _ := decimal.NewFromInt(int64(s)).Div(decimal.NewFromInt(int64(ZCN))).Round(10).Float64()
	return val
}
