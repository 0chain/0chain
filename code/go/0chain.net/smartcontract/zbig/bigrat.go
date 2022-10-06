package zbig

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"math/big"

	"github.com/tinylib/msgp/msgp"
)

//go:generate msgp -io=false -tests=false -v

const BigRatMsgpExtensionType = 98

func init() {
	msgp.RegisterExtension(BigRatMsgpExtensionType, func() msgp.Extension { return new(BigRat) })
}

type BigRat struct {
	*big.Rat `msg:"-"`
}

func NewBigRat(r big.Rat) *BigRat {
	return &BigRat{&r}
}

func BigRatFromFloat64(f float64) BigRat {
	rat := new(big.Rat)
	return BigRat{
		Rat: rat.SetFloat64(f),
	}
}

func BigRatFromInt64(i int64) BigRat {
	rat := new(big.Rat)
	return BigRat{
		Rat: rat.SetInt64(i),
	}
}

var ZeroBigRat = big.NewRat(0, 1)
var OneBigRat = big.NewRat(1, 1)

func (br *BigRat) Float64() float64 {
	if br.Rat == nil {
		return 0.0
	}
	f, _ := br.Rat.Float64()
	return f
}

// impment Extension interface for mgsp
func (br *BigRat) ExtensionType() int8 {
	return BigRatMsgpExtensionType
}

func (br *BigRat) Len() int {
	if br.Rat == nil {
		return 0
	}
	return len(br.String())
}

func (br *BigRat) MarshalBinaryTo(output []byte) error {
	if br.Rat == nil {
		output = nil
		return nil
	}
	copy(output, br.String())
	return nil
}

func (br *BigRat) UnmarshalBinary(input []byte) error {
	return br.Scan(input)
}

func (br *BigRat) MarshalJSON() ([]byte, error) {
	if br.Rat == nil {
		br.Rat = big.NewRat(0, 1)
	}
	f, _ := br.Rat.Float64()
	return []byte(fmt.Sprintf("%f", f)), nil
}

func (br *BigRat) UnmarshalJSON(input []byte) error {
	return br.Scan(input)
}

func (br *BigRat) Scan(value interface{}) error {
	if value == nil {
		return errors.New("scanning nil value")
	}
	if br.Rat == nil {
		br.Rat = big.NewRat(0, 1)
	}

	var ok bool
	switch src := value.(type) {
	case string:
		_, ok = br.SetString(src)
	case []byte:
		_, ok = br.SetString(string(src))
	default:
		return fmt.Errorf("cannot scan %T", src)
	}
	if !ok {
		return fmt.Errorf("%v not recognised as a big.Rat", value)
	}
	return nil
}

// Value must not use a pointer receiver
func (br BigRat) Value() (driver.Value, error) {
	if br.Rat == nil {

		br.Rat = big.NewRat(0, 1)
	}
	return br.String(), nil
}
