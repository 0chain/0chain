package event

import "gorm.io/gorm"

type ValidationNode struct {
	gorm.Model
	ChallengeId uint
	ValidatorID string `json:"id" gorm:"uniqueIndex"`
	BaseURL     string `json:"url"`
}

func (edb *EventDb) getValidationNoes(challengeId uint) ([]ValidationNode, error) {
	var validators []ValidationNode
	result := edb.Store.Get().
		Model(&ValidationNode{}).
		Where("challenge_id", challengeId).
		Find(&validators)
	if result.Error != nil {
		return nil, result.Error
	}
	return validators, nil
}
