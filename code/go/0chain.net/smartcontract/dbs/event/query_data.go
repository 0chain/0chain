package event

import "fmt"

func (edb *EventDb) GetQueryData(fields string, table interface{}) (interface{}, error) {
	var result interface{}
	switch t := table.(type) {
	case *Miner:
		var miners []Miner
		err := edb.Get().Model(&t).Select(fields).Find(&miners).Error
		if err != nil {
			return nil, err
		}
		result = miners
	case *Blobber:
		var blobbers []Blobber
		err := edb.Get().Model(&t).Select(fields).Find(&blobbers).Error
		if err != nil {
			return nil, err
		}
		result = blobbers
	case *Sharder:
		var sharders []Sharder
		err := edb.Get().Model(&t).Select(fields).Find(&sharders).Error
		if err != nil {
			return nil, err
		}
		result = sharders
	case *Authorizer:
		var authorizers []Authorizer
		err := edb.Get().Model(&t).Select(fields).Find(&authorizers).Error
		if err != nil {
			return nil, err
		}
		result = authorizers
	case *Validator:
		var validators []Validator
		err := edb.Get().Model(&t).Select(fields).Find(&validators).Error
		if err != nil {
			return nil, err
		}
		result = validators
	default:
		return nil, fmt.Errorf("unsupported type: %T", t)
	}
	return result, nil
}
