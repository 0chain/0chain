package ememorystore

import (
	"0chain.net/core/common"
	"0chain.net/core/datastore"
)

type CounterMergeOperator struct {
	KeyFieldName     string
	CounterFieldName string
}

func NewCounterMergeOperator(keyFieldName, counterFieldName string) *CounterMergeOperator {
	return &CounterMergeOperator{
		KeyFieldName:     keyFieldName,
		CounterFieldName: counterFieldName,
	}
}

func (m *CounterMergeOperator) Name() string { return "counter_merger" }

func (m *CounterMergeOperator) FullMerge(key, existingValue []byte, operands [][]byte) ([]byte, bool) {

	curCounterEntity := make(map[string]interface{})
	if string(existingValue) == "" {
		curCounterEntity[m.KeyFieldName] = datastore.ToKey(key)
		curCounterEntity[m.CounterFieldName] = float64(0)
	} else {
		err := common.FromJSON(existingValue, &curCounterEntity)
		if err != nil {
			return nil, false
		}
	}

	var (
		result float64
		ok     bool
	)

	if result, ok = curCounterEntity[m.CounterFieldName].(float64); !ok {
		return nil, false
	}

	for _, operand := range operands {
		operandEntity := make(map[string]interface{})
		err := common.FromJSON(operand, &operandEntity)
		if err != nil {
			return nil, false
		}
		var delta float64
		if delta, ok = operandEntity[m.CounterFieldName].(float64); !ok {
			return nil, false
		}
		result += delta
	}

	curCounterEntity[m.CounterFieldName] = int64(result)
	buf, err := common.ToJSON(curCounterEntity)
	if err != nil {
		return nil, false
	}
	return buf.Bytes(), true
}
