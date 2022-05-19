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
	// ErrUint64OverflowsFloat64 is returned if when converting a uint64 to a float64 overflow float64
	ErrUint64OverflowsFloat64 = errors.New("uint64 overflows float64")
	// ErrInt64UnderflowsUint64 is returned if when converting an int64 to a uint64 underflow uint64
	ErrInt64UnderflowsUint64 = errors.New("int64 underflows uint64")
	// ErrFloat64UnderflowsUint64 is returned if when converting an float6464 to a uint64 underflow uint64
	ErrFloat64UnderflowsUint64 = errors.New("float64 underflows uint64")
)

var maxDecimal decimal.Decimal

func init() {
	maxDecimal = decimal.NewFromInt(math.MaxInt64)
}

//go:generate msgp -io=false -tests=false -v
//Coin - any quantity that is represented as an integer in the lowest denomination
type Coin uint64

func ParseZCN(c float64) (Coin, error) {
	d := decimal.NewFromFloat(c)
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

func (c Coin) ToZCN() (float64, error) {
	if c > math.MaxInt64 {
		return 0.0, ErrTooLarge
	}

	f, _ := decimal.New(int64(c), -ZCNExponent).Float64()
	return f, nil
}

// Int64 converts a uint64 Coin to an int64, returning an error if the uint64 value overflows int64
func (c Coin) Int64() (int64, error) {
	b := int64(c)
	if b < 0 {
		return 0, ErrUint64OverflowsInt64
	}
	return b, nil
}

// Float64 converts a uint64 Coin to an float64, returning an error if the uint64 value overflows float64
func (c Coin) Float64() (float64, error) {
	b := float64(c)
	if b < 0 {
		return 0, ErrUint64OverflowsFloat64
	}
	return b, nil
}

// MultUint64 multiplies Coin a by b, returning an error if the values overflow
func (c Coin) MultUint64(b uint64) (Coin, error) {
	a := uint64(c) * b
	if a != 0 && a/uint64(c) != b {
		return 0, ErrUint64MultOverflow
	}
	return Coin(a), nil
}

// AddCoin adds a and b, returning an error if the values overflow
func (c Coin) AddCoin(b Coin) (Coin, error) {
	sum := c + b
	if sum < c || sum < b {
		return 0, ErrUint64AddOverflow
	}
	return sum, nil
}

func (c Coin) DivideCurrency(a int64) (oCur, bal Coin, err error) {
	d, err := Int64ToCoin(a)
	if err != nil {
		return
	}
	oCur = c / d
	bal = c % d
	return
}

// Int64ToCoin converts an int64 to a uint64 Coin, returning an error if the int64 value underflows uint64
func Int64ToCoin(a int64) (Coin, error) {
	if a < 0 {
		return 0, ErrInt64UnderflowsUint64
	}
	return Coin(a), nil
}

// Float64ToCoin converts an float64 to a uint64 Coin, returning an error if the float64 value underflows uint64
func Float64ToCoin(a float64) (Coin, error) {
	if a < 0 {
		return 0, ErrFloat64UnderflowsUint64
	}
	return Coin(a), nil
}

func Min(a, b Coin) (c Coin) {
	if a < b {
		return a
	}
	return b
}
