package event

import (
	"time"

	"github.com/0chain/common/core/logging"
	"github.com/lib/pq"
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
				Round:         event.BlockNumber,
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
				Round:         event.BlockNumber,
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
				Round:          event.BlockNumber,
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
				Round:          event.BlockNumber,
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
				Round:      event.BlockNumber,
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
				Round:      event.BlockNumber,
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
				Round:     event.BlockNumber,
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
				Round:           event.BlockNumber,
				UserID:          u.UserID,
				CollectedReward: u.CollectedReward,
			})
		}
		return
	},
}

func (edb *EventDb) GetLatestUserAggregates(ids map[string]interface{}) (map[string]*UserAggregate, error) {
	var ua []UserAggregate
	var mappedAggrs = make(map[string]*UserAggregate, len(ua))

	var idlist []string
	for id := range ids {
		idlist = append(idlist, id)
	}

	if len(idlist) == 0 {
		logging.Logger.Info("empty aggregates list")
		return mappedAggrs, nil
	}
	result := edb.Store.Get().
		Raw(`SELECT user_id, max(round), collected_reward, payed_fees, total_stake, read_pool_total, write_pool_total 
	FROM user_aggregates 
	WHERE user_id IN (SELECT unnest(?::text[]))
	GROUP BY user_id, collected_reward, payed_fees, total_stake, read_pool_total, write_pool_total`, pq.Array(idlist)).
		Scan(&ua)
	if result.Error != nil {
		logging.Logger.Error("can't select aggregates", zap.Error(result.Error))
		return nil, result.Error
	}

	for _, aggr := range ua {
		mappedAggrs[aggr.UserID] = &aggr
	}

	return mappedAggrs, nil
}

func (edb *EventDb) updateUserAggregates(e *blockEvents) error {
	var updatedAggrs []UserAggregate
	for _, ev := range e.events {
		if h := handlers[ev.Tag]; h != nil {
			aggrs := h(ev)
			updatedAggrs = append(updatedAggrs, aggrs...)
		}
	}

	ids := make(map[string]interface{})
	for _, aggr := range updatedAggrs {
		aggr.Round = e.round
		ids[aggr.UserID] = struct{}{}
	}

	latest, err := edb.GetLatestUserAggregates(ids)
	if err != nil {
		logging.Logger.Error("can't load latest aggregates", zap.Error(err))
		return err
	}

	for _, aggr := range updatedAggrs {
		u := aggr
		if aggr.Round == e.round {
			logging.Logger.Error("duplicate round, not sure why", zap.Any("aggr", aggr), zap.Any("latest", latest[u.UserID]))
		}
		a, ok := latest[u.UserID]
		if !ok {
			latest[u.UserID] = &u
			continue
		}
		merge(a, &u)
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
