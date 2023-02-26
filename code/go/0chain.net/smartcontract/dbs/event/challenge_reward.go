package event

import (
	"0chain.net/smartcontract/dbs/model"
	"github.com/0chain/common/core/currency"
)

// swagger:model RewardProvider
type ValidatorRewardHistory struct {
	model.UpdatableModel
	Amount      currency.Coin `json:"amount"`
	ValidatorID string        `json:"validator_id"`
	ChallengeID string        `json:"challenge_id"`
	Success     bool          `json:"success"`
}

func (edb *EventDb) InsertValidatorRewardHistory(history ValidatorRewardHistory) error {
	historyDB := ValidatorRewardHistory{
		Amount:      history.Amount,
		ValidatorID: history.ValidatorID,
		ChallengeID: history.ChallengeID,
		Success:     history.Success,
	}
	return edb.Get().Create(&historyDB).Error
}

func (edb *EventDb) GetValidatorRewardHistor(validatorID string) ([]ValidatorRewardHistory, error) {
	var histories []ValidatorRewardHistory
	err := edb.Get().Where("validator_id = ?", validatorID).Find(&histories).Error
	return histories, err
}
