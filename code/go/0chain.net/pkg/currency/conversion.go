package currency

import (
	"github.com/shopspring/decimal"
)

func ParseZCN(c float64) Coin {
	dInt := decimal.NewFromFloat(c).Truncate(10).Mul(decimal.NewFromInt(int64(ZCN))).IntPart()
	return Coin(dInt)
}

func ParseMZCN(c float64) Coin {
	dInt := decimal.NewFromFloat(c).Truncate(7).Mul(decimal.NewFromInt(int64(mZCN))).IntPart()
	return Coin(dInt)
}

func ParseUZCN(c float64) Coin {
	dInt := decimal.NewFromFloat(c).Truncate(4).Mul(decimal.NewFromInt(int64(uZCN))).IntPart()
	return Coin(dInt)
}

func (c Coin) ToZCN() float64 {
	val, _ := decimal.NewFromInt(int64(c)).Div(decimal.NewFromInt(int64(ZCN))).Float64()
	return val
}

func (c Coin) Int64() int64 {
	return int64(c)
}

func (c Coin) Float64() float64 {
	return float64(c)
}
