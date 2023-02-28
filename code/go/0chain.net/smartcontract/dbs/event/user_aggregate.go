package event

import (
	"time"

	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
)

type UserAggregate struct {
	UserID          string `json:"user_id" gorm:"uniqueIndex"`
	Round           int64  `json:"round"`
	CollectedReward int64  `json:"collected_reward"`
	TotalStake      int64  `json:"total_stake"`
	ReadPoolTotal   int64  `json:"read_pool_total"`
	WritePoolTotal  int64  `json:"write_pool_total"`
	PayedFees       int64  `json:"payed_fees"`
	CreatedAt       time.Time
}

func (edb *EventDb) updateUserAggregate(round int64, evs []Event) error {
	userAggrs := map[string]*UserAggregate{}
	for _, event := range evs {
		switch event.Tag {
		case TagLockReadPool:
			rpls, ok := fromEvent[[]ReadPoolLock](event.Data)
			if !ok {
				return ErrInvalidEventData
			}
			for _, rpl := range *rpls {
				userAggrs[rpl.Client].ReadPoolTotal += rpl.Amount
			}
			break
		case TagUnlockReadPool:
			rpls, ok := fromEvent[[]ReadPoolLock](event.Data)
			if !ok {
				return ErrInvalidEventData
			}
			for _, rpl := range *rpls {
				userAggrs[rpl.Client].ReadPoolTotal -= rpl.Amount
			}
			break
		case TagLockWritePool:
			wpls, ok := fromEvent[[]WritePoolLock](event.Data)
			if !ok {
				return ErrInvalidEventData
			}
			for _, wpl := range *wpls {
				userAggrs[wpl.Client].WritePoolTotal = wpl.Amount
			}
			break
		case TagUnlockWritePool:
			wpls, ok := fromEvent[[]WritePoolLock](event.Data)
			if !ok {
				return ErrInvalidEventData
			}
			for _, wpl := range *wpls {
				userAggrs[wpl.Client].WritePoolTotal -= wpl.Amount
			}
			break
		case TagLockStakePool:
			dpls, ok := fromEvent[[]DelegatePoolLock](event.Data)
			if !ok {
				return ErrInvalidEventData
			}
			for _, dpl := range *dpls {
				userAggrs[dpl.Client].TotalStake += dpl.Amount
			}
			break
		case TagUnlockStakePool:
			dpls, ok := fromEvent[[]DelegatePoolLock](event.Data)
			if !ok {
				return ErrInvalidEventData
			}
			for _, dpl := range *dpls {
				userAggrs[dpl.Client].TotalStake -= dpl.Amount
			}
			break
		case TagUpdateUserPayedFees:
			users, ok := fromEvent[[]User](event.Data)
			if !ok {
				return ErrInvalidEventData
			}
			for _, u := range *users {
				userAggrs[u.UserID].PayedFees += u.PayedFees
			}
			break
		case TagUpdateUserCollectedRewards:
			users, ok := fromEvent[[]User](event.Data)
			if !ok {
				return ErrInvalidEventData
			}
			for _, u := range *users {
				userAggrs[u.UserID].CollectedReward += u.CollectedReward
			}
			break
		default:
			continue
		}
	}
	for key, aggr := range userAggrs {
		aggr.Round = round
		aggr.UserID = key
		err := edb.addUserAggregate(aggr)
		if err != nil {
			logging.Logger.Error("saving user aggregate failed", zap.Error(err))
			return err
		}
	}
	return nil
}

func (edb *EventDb) addUserAggregate(ua *UserAggregate) error {
	if result := edb.Store.Get().Create(ua); result.Error != nil {
		return result.Error
	}
	return nil
}
