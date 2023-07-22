package event

import (
	common2 "0chain.net/smartcontract/common"
	"0chain.net/smartcontract/dbs/model"
	"fmt"
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
	"gorm.io/gorm/clause"

	"0chain.net/core/common"
)

// swagger:model Challenges
type Challenges []Challenge

type Challenge struct {
	model.UpdatableModel
	ChallengeID    string           `json:"challenge_id" gorm:"index:idx_cchallenge_id,unique"`
	CreatedAt      common.Timestamp `json:"created_at" gorm:"index:idx_copen_challenge,priority:1"`
	AllocationID   string           `json:"allocation_id"`
	BlobberID      string           `json:"blobber_id" gorm:"index:idx_copen_challenge,priority:2"`
	ValidatorsID   string           `json:"validators_id"`
	Seed           int64            `json:"seed"`
	AllocationRoot string           `json:"allocation_root"`
	Responded      int64            `json:"responded" gorm:"index:idx_copen_challenge,priority:3"`
	Passed         bool             `json:"passed"`
	RoundResponded int64            `json:"round_responded"`
	ExpiredN       int              `json:"expired_n" gorm:"-"`
	Timestamp      common.Timestamp `json:"timestamp" gorm:"timestamp"`
}

func (edb *EventDb) GetAllChallengesByAllocationID(allocationID string) (Challenges, error) {
	var chs Challenges
	result := edb.Store.Get().Model(&Challenge{}).Where(&Challenge{AllocationID: allocationID}).Find(&chs)
	return chs, result.Error
}

func (edb *EventDb) GetPassedChallengesForBlobberAllocation(allocationID string) (map[string]int, error) {
	result := make(map[string]int)

	edb.Store.Get().Table("challenges").
		Select("blobber_id, count(*) as count").
		Where("allocation_id = ? AND passed = ?", allocationID, true).
		Group("blobber_id").
		Scan(&result)

	return result, nil
}

func (edb *EventDb) GetChallenge(challengeID string) (*Challenge, error) {
	var ch Challenge

	result := edb.Store.Get().Model(&Challenge{}).Where(&Challenge{ChallengeID: challengeID}).First(&ch)
	if result.Error != nil {
		return nil, fmt.Errorf("error retriving Challenge node with ID %v; error: %v", challengeID, result.Error)
	}

	return &ch, nil
}

func (edb *EventDb) GetChallenges(blobberId string, start, end int64) ([]Challenge, error) {
	var chs []Challenge
	result := edb.Store.Get().
		Model(&Challenge{}).
		Where("blobber_id = ? AND round_responded >= ? AND round_responded < ?",
			blobberId, start, end).
		Find(&chs).Debug()
	return chs, result.Error
}

func (edb *EventDb) GetOpenChallengesForBlobber(blobberID string, from, now, cct common.Timestamp,
	limit common2.Pagination, challengeID string) ([]*Challenge, error) {
	var chs []*Challenge
	expiry := now - cct

	logging.Logger.Info("GetOpenChallengesForBlobber", zap.Any("blobberID", blobberID), zap.Any("from", from), zap.Any("now", now), zap.Any("cct", cct), zap.Any("limit", limit), zap.Any("challengeID", challengeID))

	var challengeWithChallengeID *Challenge
	if challengeID != "" {
		challengeWithChallengeID = &Challenge{}
		result := edb.Store.Get().Model(&Challenge{}).Where(&Challenge{ChallengeID: challengeID}).First(&challengeWithChallengeID)
		if result.Error != nil {
			return nil, fmt.Errorf("error retriving Challenge node with ID %v; error: %v", challengeID, result.Error)
		}
	} else {
		challengeWithChallengeID = &Challenge{}
		result := edb.Store.Get().Model(&Challenge{}).Where(&Challenge{BlobberID: blobberID}).First(&challengeWithChallengeID)
		if result.Error != nil {
			return nil, fmt.Errorf("error retriving Challenge node with ID %v; error: %v", challengeID, result.Error)
		}
	}

	if challengeWithChallengeID.CreatedAt < expiry {
		challengeWithChallengeID.CreatedAt = expiry
	}

	logging.Logger.Info("GetOpenChallengesForBlobber", zap.Any("challengeWithChallengeID", challengeWithChallengeID), zap.Any("from", from), zap.Any("now", now), zap.Any("cct", cct), zap.Any("limit", limit), zap.Any("challengeID", challengeID))

	query := edb.Store.Get().Model(&Challenge{}).
		Where("created_at >= ? AND responded = 0", challengeWithChallengeID.CreatedAt).
		Limit(50).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "created_at"},
		}).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "challenge_id"},
		})

	result := query.Find(&chs)
	if result.Error != nil {
		return nil, fmt.Errorf("error retriving open Challenges with blobberid %v; error: %v",
			blobberID, result.Error)
	}

	logging.Logger.Info("GetOpenChallengesForBlobber", zap.Any("result", result), zap.Any("chs", chs))

	return chs, nil
}

func (edb *EventDb) addChallenges(chlgs []Challenge) error {
	return edb.Store.Get().Create(&chlgs).Error
}

func (edb *EventDb) updateChallenges(chs []Challenge) error {
	var (
		challengeIdList    []string
		respondedList      []int64
		roundRespondedList []int64
		passedList         []bool
	)

	for _, ch := range chs {
		challengeIdList = append(challengeIdList, ch.ChallengeID)
		respondedList = append(respondedList, ch.Responded)
		roundRespondedList = append(roundRespondedList, ch.RoundResponded)
		passedList = append(passedList, ch.Passed)
	}

	return CreateBuilder("challenges", "challenge_id", challengeIdList).
		AddUpdate("responded", respondedList).
		AddUpdate("round_responded", roundRespondedList).
		AddUpdate("passed", passedList).Exec(edb).Error
}

func mergeAddChallengesEvents() *eventsMergerImpl[Challenge] {
	return newEventsMerger[Challenge](TagAddChallenge, withUniqueEventOverwrite())
}

func mergeUpdateChallengesEvents() *eventsMergerImpl[Challenge] {
	return newEventsMerger[Challenge](TagUpdateChallenge, withUniqueEventOverwrite())
}
