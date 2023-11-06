package stats

import "fmt"

type Aggregate map[string]any

type AggregateStore map[ProviderType]map[string][]Aggregate

type ProviderType string
const (
	Sharder ProviderType = "sharder"
	Miner ProviderType = "miner"
	Blobber ProviderType = "blobber"
	Validator ProviderType = "validator"
	Authorizer ProviderType = "authorizer"
	User ProviderType = "user"
	Global ProviderType = "global" // For Global, the id will be always "global".
)

type Monotonicity string
const (
	Increase Monotonicity = "increase"
	Decrease Monotonicity = "decrease"
)

type Comparison string
const (
	EQ Comparison = "eq"
	LT Comparison = "lt"
	LTE Comparison = "lte"
	GT Comparison = "gt"
	GTE Comparison = "gte"
)

var store AggregateStore

func init() {
	store = make(AggregateStore)
	store[Sharder] = make(map[string][]Aggregate)
	store[Miner] = make(map[string][]Aggregate)
	store[Blobber] = make(map[string][]Aggregate)
	store[Validator] = make(map[string][]Aggregate)
	store[Authorizer] = make(map[string][]Aggregate)
	store[User] = make(map[string][]Aggregate)
	store[Global] = make(map[string][]Aggregate)
}

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

func AddAggregate(agg Aggregate, ptype ProviderType, pid string) error {
	_, err := getProviderIdStore(ptype, pid, false) // Used to check if it exists
	if err != nil {
		return err
	}

	store[ptype][pid] = append(store[ptype][pid], agg)
	return nil
}

func CheckAggregateValueChange(ptype ProviderType, pid string, key string, monotonicity Monotonicity) (bool, error) {
	aggProviderIdStore, err := getProviderIdStore(ptype, pid, false)
	if err != nil {
		return false, err
	}

	if len(aggProviderIdStore) < 2 {
		return false, nil
	}

	prev, ok := aggProviderIdStore[0][key]
	if !ok {
		return false, fmt.Errorf("key not found in the aggregates of this provider: %v, %v, %v", ptype.String(), pid, key)
	}

	prevInt64, ok := prev.(int64)
	if !ok {
		return false, fmt.Errorf("key found in the aggregates of this provider but value not int64: %v, %v, %v, %T %v", ptype.String(), pid, key, prev, prev)
	}

	for i := 1; i < len(aggProviderIdStore); i++ {
		cur, ok := aggProviderIdStore[i][key]
		if !ok {
			return false, fmt.Errorf("key not found in the aggregates of this provider: %v, %v, %v", ptype.String(), pid, key)
		}

		switch curInt64 := cur.(type) {
		case int64:
			var check bool
			switch monotonicity {
			case Increase:
				check = curInt64 > prevInt64
			case Decrease:
				check = curInt64 < prevInt64
			default:
				return false, fmt.Errorf("unknown monotonicity")
			}
	
			if check {
				return true, nil
			}
	
			prevInt64 = curInt64
		default:
			return false, fmt.Errorf("key found in the aggregates of this provider but value not int64: %v, %v, %v, %T %v", ptype.String(), pid, key, cur, cur)
		}
	}

	return false, nil
}

func GetLatestAggregateValue(ptype ProviderType, pid string, key string) (int64, error) {
	aggProviderIdStore, err := getProviderIdStore(ptype, pid, false)
	if err != nil {
		return 0, err
	}

	val, ok := aggProviderIdStore[len(aggProviderIdStore) - 1][key]
	if !ok {
		return 0, fmt.Errorf("key not found in the aggregates of this provider: %v, %v, %v", ptype.String(), pid, key)
	}

	valInt64, ok := val.(int64)
	if !ok {
		return 0, fmt.Errorf("key found in the aggregates of this provider but value not int64: %v, %v, %v, %T %v", ptype.String(), pid, key, val, val)
	}

	return valInt64, nil
}

func CompareAggregateValue(ptype ProviderType, pid string, key string, comparison Comparison, rvalue int64) (bool, error) {
	aggProviderIdStore, err := getProviderIdStore(ptype, pid, false)
	if err != nil {
		return false, err
	}

	if len(aggProviderIdStore) == 0 {
		return false, fmt.Errorf("no aggregates for this provider: %v, %v", ptype.String(), pid)
	}

	latestVal, ok := aggProviderIdStore[len(aggProviderIdStore)-1][key]
	if !ok {
		return false, fmt.Errorf("key not found in the aggregates of this provider: %v, %v, %v", ptype.String(), pid, key)
	}

	latestValInt64, ok := latestVal.(int64)
	if !ok {
		return false, fmt.Errorf("key found in the aggregates of this provider but value not int64: %v, %v, %v, %T %v", ptype.String(), pid, key, latestVal, latestVal)
	}

	switch comparison {
	case EQ:
		return latestValInt64 == rvalue, nil
	case LT:
		return latestValInt64 < rvalue, nil
	case LTE:
		return latestValInt64 <= rvalue, nil
	case GT:
		return latestValInt64 > rvalue, nil
	case GTE:
		return latestValInt64 >= rvalue, nil
	default:
		return false, fmt.Errorf("unknown comparison")
	}
}

func getProviderIdStore(ptype ProviderType, pid string, mustGet bool) ([]Aggregate, error) {
	aggProviderTypeStore, ok := store[ptype]
	if !ok {
		return nil, fmt.Errorf("unknown aggregate provider type")
	}

	aggProviderIdStore, ok := aggProviderTypeStore[pid]
	if !ok {
		if mustGet {
			return nil, fmt.Errorf("provider id has no stored aggregates")
		}
		aggProviderIdStore = make([]Aggregate, 0)
		store[ptype][pid] = aggProviderIdStore
	}

	return aggProviderIdStore, nil
}