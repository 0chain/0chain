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

var handlers = map[EventTag]func(e Event) (updatedAggrs []UserAggregate){
	TagLockReadPool: func(event Event) (updatedAggrs []UserAggregate) {
		rpls, ok := fromEvent[[]ReadPoolLock](event.Data)
		if !ok {
			logging.Logger.Error("user aggregate",
				zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
			return

		}
		for _, rpl := range *rpls {
			updatedAggrs = append(updatedAggrs, UserAggregate{
				UserID:        rpl.Client,
				ReadPoolTotal: rpl.Amount,
			})
		}
		return
	},
	TagUnlockReadPool: func(event Event) (updatedAggrs []UserAggregate) {
		rpls, ok := fromEvent[[]ReadPoolLock](event.Data)
		if !ok {
			logging.Logger.Error("user aggregate unlock read pool",
				zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
			return

		}
		for _, rpl := range *rpls {
			updatedAggrs = append(updatedAggrs, UserAggregate{
				UserID:        rpl.Client,
				ReadPoolTotal: -rpl.Amount,
			})
		}
		return
	},
	TagLockWritePool: func(event Event) (updatedAggrs []UserAggregate) {
		wpls, ok := fromEvent[[]WritePoolLock](event.Data)
		if !ok {
			logging.Logger.Error("user aggregate lock write pool",
				zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
			return

		}
		for _, wpl := range *wpls {
			updatedAggrs = append(updatedAggrs, UserAggregate{
				UserID:         wpl.Client,
				WritePoolTotal: wpl.Amount,
			})
		}
		return
	},
	TagUnlockWritePool: func(event Event) (updatedAggrs []UserAggregate) {
		wpls, ok := fromEvent[[]WritePoolLock](event.Data)
		if !ok {
			logging.Logger.Error("user aggregate unlock stake pool",
				zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
			return

		}
		for _, wpl := range *wpls {
			updatedAggrs = append(updatedAggrs, UserAggregate{
				UserID:         wpl.Client,
				WritePoolTotal: -wpl.Amount,
			})
		}
		return
	},
	TagLockStakePool: func(event Event) (updatedAggrs []UserAggregate) {
		dpls, ok := fromEvent[[]DelegatePoolLock](event.Data)
		if !ok {
			logging.Logger.Error("user aggregate lock stake pool",
				zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
			return

		}
		for _, dpl := range *dpls {
			updatedAggrs = append(updatedAggrs, UserAggregate{
				UserID:     dpl.Client,
				TotalStake: dpl.Amount,
			})
		}
		return
	},
	TagUnlockStakePool: func(event Event) (updatedAggrs []UserAggregate) {
		dpls, ok := fromEvent[[]DelegatePoolLock](event.Data)
		if !ok {
			logging.Logger.Error("user aggregate",
				zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
			return

		}
		for _, dpl := range *dpls {
			updatedAggrs = append(updatedAggrs, UserAggregate{
				UserID:     dpl.Client,
				TotalStake: -dpl.Amount,
			})
		}
		return
	},
	TagUpdateUserPayedFees: func(event Event) (updatedAggrs []UserAggregate) {
		users, ok := fromEvent[[]UserAggregate](event.Data)
		if !ok {
			logging.Logger.Error("user aggregate",
				zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
			return

		}
		for _, u := range *users {
			updatedAggrs = append(updatedAggrs, UserAggregate{
				UserID:    u.UserID,
				PayedFees: u.PayedFees,
			})
		}
		return
	},
	TagUpdateUserCollectedRewards: func(event Event) (updatedAggrs []UserAggregate) {
		users, ok := fromEvent[[]UserAggregate](event.Data)
		if !ok {
			logging.Logger.Error("user aggregate",
				zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
			return
		}
		for _, u := range *users {
			updatedAggrs = append(updatedAggrs, UserAggregate{
				UserID:          u.UserID,
				CollectedReward: u.CollectedReward,
			})
		}
		return
	},
}

func (edb *EventDb) GetLatestUserAggregates(ids map[string]interface{}) (map[string]*UserAggregate, error) {
	var ua []UserAggregate

	exec := edb.Store.Get().Exec("CREATE TEMP TABLE IF NOT EXISTS temp_user_ids (ID text) ON COMMIT DROP")
	if exec.Error != nil {
		return nil, exec.Error
	}

	var idlist []string
	for id := range ids {
		idlist = append(idlist, id)
	}

	r := edb.Store.Get().Exec("INSERT INTO temp_user_ids (ID) VALUES (?)", idlist)
	if r.Error != nil {
		return nil, r.Error
	}

	result := edb.Store.Get().
		Raw(`SELECT user_id, max(round), collected_reward, payed_fees, total_stake, read_pool_total, write_pool_total 
	FROM user_aggregates 
	WHERE user_id IN (SELECT ID from temp_user_ids)
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

func (edb *EventDb) updateUserAggregates(e *blockEvents) error {
	var events []Event

	var updatedAggrs []UserAggregate
	for _, ev := range events {
		if h := handlers[ev.Tag]; h != nil {
			updatedAggrs = append(updatedAggrs, h(ev)...)
		}
	}

	ids := make(map[string]interface{})
	for _, aggr := range updatedAggrs {
		ids[aggr.UserID] = struct{}{}
	}

	latest, err := edb.GetLatestUserAggregates(ids)
	if err != nil {
		logging.Logger.Error("can't load latest aggregates", zap.Error(err))
		return err
	}

	for _, aggr := range updatedAggrs {
		a, ok := latest[aggr.UserID]
		if !ok {
			latest[aggr.UserID] = &aggr
			continue
		}
		merge(a, &aggr)
	}

	err = edb.addUserAggregates(latest)
	if err != nil {
		logging.Logger.Error("saving user aggregate failed", zap.Error(err))
		return err
	}

	return nil
}

func merge(a *UserAggregate, u *UserAggregate) {
	a.Round = u.Round
	a.CollectedReward += u.CollectedReward
	a.PayedFees += u.PayedFees
	a.WritePoolTotal += u.WritePoolTotal
	a.TotalStake += u.TotalStake
	a.ReadPoolTotal += u.ReadPoolTotal
}

func (edb *EventDb) addUserAggregates(mapped map[string]*UserAggregate) error {
	var aggrs []*UserAggregate
	for _, aggr := range mapped {
		aggrs = append(aggrs, aggr)
	}
	if result := edb.Store.Get().Create(aggrs); result.Error != nil {
		return result.Error
	}
	return nil
}
