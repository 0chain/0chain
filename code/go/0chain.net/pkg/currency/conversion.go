package currency

import (
	"errors"
	"math"

	"github.com/shopspring/decimal"
)

const (
	ZCNExponent  = 10
	MZCNExponent = 7
	UZCNExponent = 4
)

var (
	// ErrNegativeValue is returned if a float value is a negative number
	ErrNegativeValue = errors.New("negative coin value")
	// ErrTooManyDecimals is returned if a value has more than 10 decimal places
	ErrTooManyDecimals = errors.New("too many decimal places")
	// ErrTooLarge is returned if a value is greater than math.MaxInt64
	ErrTooLarge = errors.New("value is too large")

	// ErrUint64MultOverflow is returned if when multiplying uint64 values overflow uint64
	ErrUint64MultOverflow = errors.New("uint64 multiplication overflow")
	// ErrUint64AddOverflow is returned if when adding uint64 values overflow uint64
	ErrUint64AddOverflow = errors.New("uint64 addition overflow")
	// ErrUint64OverflowsInt64 is returned if when converting a uint64 to an int64 overflow int64
	ErrUint64OverflowsInt64 = errors.New("uint64 overflows int64")
	// ErrInt64UnderflowsUint64 is returned if when converting an int64 to a uint64 underflow uint64
	ErrInt64UnderflowsUint64 = errors.New("int64 underflows uint64")
)

var maxDecimal decimal.Decimal

func init() {
	maxDecimal = decimal.NewFromInt(math.MaxInt64)
}

func ParseZCN(z float64) (Coin, error) {
	d := decimal.NewFromFloat(z)
	if d.Sign() == -1 {
		return 0, ErrNegativeValue
	}

	// ZCN have a maximum of 10 decimal places
	if d.Exponent() < -ZCNExponent {
		return 0, ErrTooManyDecimals
	}

	// Multiply the coin balance by 1e10 to obtain coin amount
	e := d.Shift(ZCNExponent)

	// Check that there are no decimal places remaining. This error should not
	// occur, because of the earlier check of ZCNExponent()
	if e.Exponent() < 0 {
		return 0, ErrTooManyDecimals
	}

	// Values greater than math.MaxInt64 will overflow after conversion to int64
	if e.GreaterThan(maxDecimal) {
		return 0, ErrTooLarge
	}

	return Coin(e.IntPart()), nil
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
