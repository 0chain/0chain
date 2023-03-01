package event

import (
	"fmt"
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
	var updatedAggrs []UserAggregate
	for _, event := range evs {
		logging.Logger.Debug("update user aggregate",
			zap.String("tag", event.Tag.String()))
		switch event.Tag {
		case TagLockReadPool:
			rpls, ok := fromEvent[[]ReadPoolLock](event.Data)
			if !ok {
				logging.Logger.Error("user aggregate",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				continue

			}
			for _, rpl := range *rpls {
				if aggr, ok := lua[rpl.Client]; ok {
					aggr.ReadPoolTotal += rpl.Amount
					updatedAggrs = append(updatedAggrs, *aggr)
					continue
				}
				lua[rpl.Client] = &UserAggregate{
					UserID:        rpl.Client,
					ReadPoolTotal: rpl.Amount,
				}
				updatedAggrs = append(updatedAggrs, *lua[rpl.Client])
			}
		case TagUnlockReadPool:
			rpls, ok := fromEvent[[]ReadPoolLock](event.Data)
			if !ok {
				logging.Logger.Error("user aggregate unlock read pool",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				continue

			}
			for _, rpl := range *rpls {
				if aggr, ok := lua[rpl.Client]; ok {
					aggr.ReadPoolTotal -= rpl.Amount
					updatedAggrs = append(updatedAggrs, *aggr)
					continue
				}
				lua[rpl.Client] = &UserAggregate{
					UserID:        rpl.Client,
					ReadPoolTotal: -rpl.Amount,
				}
				updatedAggrs = append(updatedAggrs, *lua[rpl.Client])
			}
		case TagLockWritePool:
			wpls, ok := fromEvent[[]WritePoolLock](event.Data)
			if !ok {
				logging.Logger.Error("user aggregate lock write pool",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				continue

			}
			for _, wpl := range *wpls {
				if aggr, ok := lua[wpl.Client]; ok {
					aggr.WritePoolTotal += wpl.Amount
					updatedAggrs = append(updatedAggrs, *aggr)
					continue
				}
				lua[wpl.Client] = &UserAggregate{
					UserID:         wpl.Client,
					WritePoolTotal: wpl.Amount,
				}
				updatedAggrs = append(updatedAggrs, *lua[wpl.Client])
			}
		case TagUnlockWritePool:
			wpls, ok := fromEvent[[]WritePoolLock](event.Data)
			if !ok {
				logging.Logger.Error("user aggregate unlock stake pool",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				continue

			}
			for _, wpl := range *wpls {
				if aggr, ok := lua[wpl.Client]; ok {
					aggr.WritePoolTotal -= wpl.Amount
					updatedAggrs = append(updatedAggrs, *aggr)
					continue
				}
				lua[wpl.Client] = &UserAggregate{
					UserID:         wpl.Client,
					WritePoolTotal: -wpl.Amount,
				}
				updatedAggrs = append(updatedAggrs, *lua[wpl.Client])
			}
		case TagLockStakePool:
			dpls, ok := fromEvent[[]DelegatePoolLock](event.Data)
			if !ok {
				logging.Logger.Error("user aggregate lock stake pool",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				continue

			}
			for _, dpl := range *dpls {
				if aggr, ok := lua[dpl.Client]; ok {
					aggr.TotalStake += dpl.Amount
					updatedAggrs = append(updatedAggrs, *aggr)
					continue
				}
				lua[dpl.Client] = &UserAggregate{
					UserID:     dpl.Client,
					TotalStake: dpl.Amount,
				}
				updatedAggrs = append(updatedAggrs, *lua[dpl.Client])
			}
		case TagUnlockStakePool:
			dpls, ok := fromEvent[[]DelegatePoolLock](event.Data)
			if !ok {
				logging.Logger.Error("user aggregate",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				continue

			}
			for _, dpl := range *dpls {
				if aggr, ok := lua[dpl.Client]; ok {
					aggr.TotalStake -= dpl.Amount
					updatedAggrs = append(updatedAggrs, *aggr)
					continue
				}
				lua[dpl.Client] = &UserAggregate{
					UserID:     dpl.Client,
					TotalStake: -dpl.Amount,
				}
				updatedAggrs = append(updatedAggrs, *lua[dpl.Client])
			}
		case TagUpdateUserPayedFees:
			users, ok := fromEvent[[]UserAggregate](event.Data)
			if !ok {
				logging.Logger.Error("user aggregate",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				continue

			}
			for _, u := range *users {
				if aggr, ok := lua[u.UserID]; ok {
					aggr.PayedFees += u.PayedFees
					updatedAggrs = append(updatedAggrs, *aggr)
					continue
				}
				lua[u.UserID] = &UserAggregate{
					UserID:    u.UserID,
					PayedFees: u.PayedFees,
				}
				updatedAggrs = append(updatedAggrs, *lua[u.UserID])
			}
		case TagUpdateUserCollectedRewards:
			users, ok := fromEvent[[]UserAggregate](event.Data)
			if !ok {
				logging.Logger.Error("user aggregate",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				continue

			}
			for _, u := range *users {
				if aggr, ok := lua[u.UserID]; ok {
					aggr.CollectedReward += u.CollectedReward
					updatedAggrs = append(updatedAggrs, *aggr)
					continue
				}
				lua[u.UserID] = &UserAggregate{
					UserID:          u.UserID,
					CollectedReward: u.CollectedReward,
				}
				updatedAggrs = append(updatedAggrs, *lua[u.UserID])
			}
		default:
			continue
		}
	}
	for _, aggr := range updatedAggrs {
		logging.Logger.Debug("Logging aggrs to be saved", zap.String("reward", fmt.Sprintf(`reward %v`, aggr.CollectedReward)),
			zap.String("fees", fmt.Sprintf(`fees %v`, aggr.PayedFees)),
			zap.String("read pool", fmt.Sprintf(`read pool %v`, aggr.ReadPoolTotal)),
			zap.String("write pool", fmt.Sprintf(`reward %v`, aggr.WritePoolTotal)),
			zap.String("stake pool", fmt.Sprintf(`stake pool %v`, aggr.TotalStake)),
		)
		aggr.Round = round
		err := edb.addUserAggregate(&aggr)
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
