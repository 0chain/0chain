package types

import "fmt"

type (
	ProviderType string
)

const (
	Sharder ProviderType = "sharder"
	Miner ProviderType = "miner"
	Blobber ProviderType = "blobber"
	Validator ProviderType = "validator"
	Authorizer ProviderType = "authorizer"
	User ProviderType = "user"
	Global ProviderType = "global" // For Global, the id will be always "global"
)

var (
	ErrNoStoredAggregates = fmt.Errorf("provider id has no stored aggregates")
)

func (p ProviderType) String() string {
	switch p {
	case Sharder:
		return "sharder"
	case Miner:
		return "miner"
	case Blobber:
		return "blobber"
	case Validator:
		return "validator"
	case Authorizer:
		return "authorizer"
	case User:
		return "user"
	case Global:
		return "global"
	default:
		return "unknown"
	}
}

type Aggregate map[string]any

type Comparison string
const (
	EQ Comparison = "eq"
	LT Comparison = "lt"
	LTE Comparison = "lte"
	GT Comparison = "gt"
	GTE Comparison = "gte"
)

type Monotonicity string
const (
	Increase Monotonicity = "increase"
	Decrease Monotonicity = "decrease"
)