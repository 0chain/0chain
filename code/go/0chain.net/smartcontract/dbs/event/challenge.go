package event

import (
	"fmt"

	common2 "0chain.net/smartcontract/common"
	"gorm.io/gorm/clause"

	"0chain.net/core/common"
	"gorm.io/gorm"
)

// swagger:model Challenges
type Challenges []Challenge

type Challenge struct {
	gorm.Model
	ChallengeID    string           `json:"challenge_id" gorm:"index:idx_cchallenge_id,unique"`
	CreatedAt      common.Timestamp `json:"created_at" gorm:"index:idx_copen_challenge,priority:1"`
	AllocationID   string           `json:"allocation_id"`
	BlobberID      string           `json:"blobber_id" gorm:"index:idx_copen_challenge,priority:2"`
	ValidatorsID   string           `json:"validators_id"`
	Seed           int64            `json:"seed"`
	AllocationRoot string           `json:"allocation_root"`
	Responded      bool             `json:"responded" gorm:"index:idx_copen_challenge,priority:3"`
	Passed         bool             `json:"passed"`
	RoundResponded int64            `json:"round_responded" gorm:"index"`
	ExpiredN       int              `json:"expired_n" gorm:"-"`
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
		Find(&chs)
	return chs, result.Error
}

func (edb *EventDb) GetOpenChallengesForBlobber(blobberID string, from, now, cct common.Timestamp,
	limit common2.Pagination) ([]*Challenge, error) {
	var chs []*Challenge
	expiry := now - cct
	if from < expiry {
		from = expiry
	}

	query := edb.Store.Get().Model(&Challenge{}).
		Where("created_at > ? AND blobber_id = ? AND responded = ?",
			from, blobberID, false).Limit(limit.Limit).Offset(limit.Offset).Order(clause.OrderByColumn{
		Column: clause.Column{Name: "created_at"},
		Desc:   limit.IsDescending,
	})

	result := query.Find(&chs)
	if result.Error != nil {
		return nil, fmt.Errorf("error retriving open Challenges with blobberid %v; error: %v",
			blobberID, result.Error)
	}

	return chs, nil
}

func (edb *EventDb) addChallenges(chlgs []Challenge) error {
	return edb.Store.Get().Create(&chlgs).Error
}

func (edb *EventDb) updateChallenges(chs []Challenge) error {
	var (
		challengeIdList []string
		respondedList   []bool
		passedList   	[]bool
	)

	for _, ch := range chs {
		challengeIdList = append(challengeIdList, ch.ChallengeID)
		respondedList = append(respondedList, ch.Responded)
		passedList = append(passedList, ch.Passed)
	}

	return CreateBuilder("challenges", "challenge_id", challengeIdList).
		AddUpdate("responded", respondedList).
		AddUpdate("passed", passedList).Exec(edb).Error
}

func mergeAddChallengesEvents() *eventsMergerImpl[Challenge] {
	return newEventsMerger[Challenge](TagAddChallenge, withUniqueEventOverwrite())
}

func mergeUpdateChallengesEvents() *eventsMergerImpl[Challenge] {
	return newEventsMerger[Challenge](TagUpdateChallenge, withUniqueEventOverwrite())
}
