package currency

import (
	"github.com/shopspring/decimal"
)

func ParseZCN(c float64) Coin {
	dInt := decimal.NewFromFloat(c).Round(10).Mul(decimal.NewFromInt(int64(ZCN))).IntPart()
	return Coin(dInt)
}

func ParseMZCN(c float64) Coin {
	dInt := decimal.NewFromFloat(c).Round(7).Mul(decimal.NewFromInt(int64(mZCN))).IntPart()
	return Coin(dInt)
}

func ParseUZCN(c float64) Coin {
	dInt := decimal.NewFromFloat(c).Round(4).Mul(decimal.NewFromInt(int64(uZCN))).IntPart()
	return Coin(dInt)
}

func (c Coin) ToZCN() float64 {
	val, _ := decimal.NewFromInt(int64(c)).Div(decimal.NewFromInt(int64(ZCN))).Round(10).Float64()
	return val
}
