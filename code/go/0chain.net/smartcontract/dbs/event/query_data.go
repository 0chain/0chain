package event

import "fmt"

func (edb *EventDb) GetQueryData(fields string, table interface{}) (interface{}, error) {
	var result interface{}
	switch t := table.(type) {
	case *Miner:
		var miners []Miner
		err := edb.Get().Model(&Miner{}).Select(fields).Find(&miners).Error
		if err != nil {
			return nil, err
		}
		result = miners
	case *Blobber:
		var blobbers []Blobber
		err := edb.Get().Model(&Blobber{}).Select(fields).Find(&blobbers).Error
		if err != nil {
			return nil, err
		}
		result = blobbers
	case *Sharder:
		var sharders []Sharder
		err := edb.Get().Model(&Sharder{}).Select(fields).Find(&sharders).Error
		if err != nil {
			return nil, err
		}
		result = sharders
	case *Authorizer:
		var authorizers []Authorizer
		err := edb.Get().Model(&Authorizer{}).Select(fields).Find(&authorizers).Error
		if err != nil {
			return nil, err
		}
		result = authorizers
	case *Validator:
		var validators []Validator
		err := edb.Get().Model(&Validator{}).Select(fields).Find(&validators).Error
		if err != nil {
			return nil, err
		}
		result = validators
	case *User:
		var users []User
		err := edb.Get().Model(&User{}).Select(fields).Find(&users).Error
		if err != nil {
			return nil, err
		}
		result = users
	case *UserSnapshot:
		var userSnapshots []UserSnapshot
		err := edb.Get().Model(&UserSnapshot{}).Select(fields).Find(&userSnapshots).Error
		if err != nil {
			return nil, err
		}
		result = userSnapshots
	case *MinerSnapshot:
		var minerSnapshots []MinerSnapshot
		err := edb.Get().Model(&MinerSnapshot{}).Select(fields).Find(&minerSnapshots).Error
		if err != nil {
			return nil, err
		}
		result = minerSnapshots
	case *BlobberSnapshot:
		var blobberSnapshots []BlobberSnapshot
		err := edb.Get().Model(&BlobberSnapshot{}).Select(fields).Find(&blobberSnapshots).Error
		if err != nil {
			return nil, err
		}
		result = blobberSnapshots
	case *SharderSnapshot:
		var sharderSnapshots []SharderSnapshot
		err := edb.Get().Model(&SharderSnapshot{}).Select(fields).Find(&sharderSnapshots).Error
		if err != nil {
			return nil, err
		}
		result = sharderSnapshots
	case *ValidatorSnapshot:
		var validatorSnapshots []ValidatorSnapshot
		err := edb.Get().Model(&ValidatorSnapshot{}).Select(fields).Find(&validatorSnapshots).Error
		if err != nil {
			return nil, err
		}
		result = validatorSnapshots
	case *AuthorizerSnapshot:
		var authorizerSnapshots []AuthorizerSnapshot
		err := edb.Get().Model(&AuthorizerSnapshot{}).Select(fields).Find(&authorizerSnapshots).Error
		if err != nil {
			return nil, err
		}
		result = authorizerSnapshots
	case *ProviderRewards:
		var providerRewards []ProviderRewards
		err := edb.Get().Model(&ProviderRewards{}).Select(fields).Find(&providerRewards).Error
		if err != nil {
			return nil, err
		}
		result = providerRewards
	default:
		return nil, fmt.Errorf("unsupported type: %T", t)
	}
	return result, nil
}
