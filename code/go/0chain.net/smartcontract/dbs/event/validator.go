package event

import (
	"context"
	"fmt"

	"0chain.net/chaincore/state"
)

type ValidationNode struct {
	grom.Model
	ValidatorID string `json:"validator_id" gorm:"index:validator_id"`
	BaseUrl     string `json:"url" gorm:"index:url"`
	Stake       int64  `json:"stake" gorm:"index:stake"`

	stakePoolSettings
}

// Overrive default table name to from "validation_nodes" to "validators".
func (ValidationNode) TableName() string {
	return "validators"
}

type stakePoolSettings struct {
	// DelegateWallet for pool owner.
	DelegateWallet string `json:"delegate_wallet"`
	// MinStake allowed.
	MinStake state.Balance `json:"min_stake"`
	// MaxStake allowed.
	MaxStake state.Balance `json:"max_stake"`
	// NumDelegates maximum allowed.
	NumDelegates int `json:"num_delegates"`
	// ServiceCharge of the blobber. The blobber gets this % (actually, value in
	// [0; 1) range). If the ServiceCharge greater than max_charge of the SC
	// then the blobber can't be registered / updated.
	ServiceCharge float64 `json:"service_charge"`
}

func (vn *ValidationNode) exists(edb *EventDb) (bool, error) {
	var count int64
	result := edb.Get().
		Model(&ValidationNode{}).
		Where(&ValidationNode{ValidatorID: vn.ValidatorID}).
		Count(&count)
	if result.Error != nil {
		return false, fmt.Errorf("error searching for ValidationNode %v, error %v",
			vn.ValidatorID, result.Error)
	}
	return count > 0, nil
}

func (edb *EventDb) GetValidationNode(ctx context.Context, validatorID string) (ValidationNode, error) {
	var vn ValidationNode

	result := edb.Store.Get().Model(&ValidationNode{}).Where(&ValidationNode{ValidatorID: validatorID}).First(vn)

	if result.Error != nil {
		return vn, fmt.Errorf("error retriving Validation node with ID %v; error: %v", validatorID, result.Error)
	}

	return vn, nil
}

func (edb *EventDb) overwriteValidationNode(vn ValidationNode) error {

	result := edb.Store.Get().Model(&ValidationNode{}).Where(&ValidationNode{ValidatorID: vn.ValidatorID}).Updates(&vn)
	return result.Error
}

func (edb *EventDb) addOrOverwriteValidationNode(vn ValidationNode) error {
	exists, err := vn.exists(edb)
	if err != nil {
		return err
	}
	if exists {
		return edb.overwriteValidationNode(vn)
	}

	result := edb.Store.Get().Create(&vn)

	return result.Error
}
