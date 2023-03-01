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

func (edb *EventDb) GetLatestUserAggregates() (map[string]*UserAggregate, error) {
	var ua []UserAggregate

	result := edb.Store.Get().
		Raw(`SELECT user_id, max(round), collected_reward, payed_fees, total_stake, read_pool_total, write_pool_total 
	FROM user_aggregates 
	GROUP BY user_id, collected_reward, payed_fees, total_stake, read_pool_total, write_pool_total`).
		Scan(&ua)
	if result.Error != nil {
		return nil, result.Error
	}

	var mappedAggrs = make(map[string]*UserAggregate, len(ua))

	for _, aggr := range ua {
		mappedAggrs[aggr.UserID] = &aggr
	}

	return mappedAggrs, nil
}

func (edb *EventDb) update(lua map[string]*UserAggregate, round int64, evs []Event) {
	for _, event := range evs {
		switch event.Tag {
		case TagLockReadPool:
			rpls, ok := fromEvent[[]ReadPoolLock](event.Data)
			if !ok {
				logging.Logger.Error("user aggregate",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
			}
			for _, rpl := range *rpls {
				if aggr, ok := lua[rpl.Client]; ok {
					aggr.ReadPoolTotal += rpl.Amount
					continue
				}
				lua[rpl.Client] = &UserAggregate{
					ReadPoolTotal: rpl.Amount,
				}
			}
		case TagUnlockReadPool:
			rpls, ok := fromEvent[[]ReadPoolLock](event.Data)
			if !ok {
				logging.Logger.Error("user aggregate unlock read pool",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
			}
			for _, rpl := range *rpls {
				if aggr, ok := lua[rpl.Client]; ok {
					aggr.ReadPoolTotal -= rpl.Amount
					continue
				}
				lua[rpl.Client] = &UserAggregate{
					ReadPoolTotal: -rpl.Amount,
				}
			}
		case TagLockWritePool:
			wpls, ok := fromEvent[[]WritePoolLock](event.Data)
			if !ok {
				logging.Logger.Error("user aggregate lock write pool",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
			}
			for _, wpl := range *wpls {
				if aggr, ok := lua[wpl.Client]; ok {
					aggr.WritePoolTotal += wpl.Amount
					continue
				}
				lua[wpl.Client] = &UserAggregate{
					WritePoolTotal: wpl.Amount,
				}
			}
		case TagUnlockWritePool:
			wpls, ok := fromEvent[[]WritePoolLock](event.Data)
			if !ok {
				logging.Logger.Error("user aggregate unlock stake pool",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
			}
			for _, wpl := range *wpls {
				if aggr, ok := lua[wpl.Client]; ok {
					aggr.WritePoolTotal -= wpl.Amount
					continue
				}
				lua[wpl.Client] = &UserAggregate{
					WritePoolTotal: -wpl.Amount,
				}
			}
		case TagLockStakePool:
			dpls, ok := fromEvent[[]DelegatePoolLock](event.Data)
			if !ok {
				logging.Logger.Error("user aggregate lock stake pool",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
			}
			for _, dpl := range *dpls {
				if aggr, ok := lua[dpl.Client]; ok {
					aggr.TotalStake += dpl.Amount
					continue
				}
				lua[dpl.Client] = &UserAggregate{
					TotalStake: dpl.Amount,
				}
			}
		case TagUnlockStakePool:
			dpls, ok := fromEvent[[]DelegatePoolLock](event.Data)
			if !ok {
				logging.Logger.Error("user aggregate",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
			}
			for _, dpl := range *dpls {
				if aggr, ok := lua[dpl.Client]; ok {
					aggr.TotalStake -= dpl.Amount
					continue
				}
				lua[dpl.Client] = &UserAggregate{
					TotalStake: -dpl.Amount,
				}
			}
		case TagUpdateUserPayedFees:
			users, ok := fromEvent[[]UserAggregate](event.Data)
			if !ok {
				logging.Logger.Error("user aggregate",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
			}
			for _, u := range *users {
				if aggr, ok := lua[u.UserID]; ok {
					aggr.PayedFees += u.PayedFees
					continue
				}
				lua[u.UserID] = &UserAggregate{
					PayedFees: u.PayedFees,
				}
			}
		case TagUpdateUserCollectedRewards:
			users, ok := fromEvent[[]UserAggregate](event.Data)
			if !ok {
				logging.Logger.Error("user aggregate",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
			}
			for _, u := range *users {
				if aggr, ok := lua[u.UserID]; ok {
					aggr.CollectedReward += u.CollectedReward
					continue
				}
				lua[u.UserID] = &UserAggregate{
					CollectedReward: u.CollectedReward,
				}
			}
		default:
			continue
		}
	}
	for key, aggr := range lua {
		aggr.Round = round
		aggr.UserID = key
		err := edb.addUserAggregate(aggr)
		if err != nil {
			logging.Logger.Error("saving user aggregate failed", zap.Error(err))
		}
	}
}

func (edb *EventDb) addUserAggregate(ua *UserAggregate) error {
	if result := edb.Store.Get().Create(ua); result.Error != nil {
		return result.Error
	}
	return nil
}
