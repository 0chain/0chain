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
		logging.Logger.Debug("user_aggregates TagLockReadPool", zap.Int("events", len(*rpls)))
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
		logging.Logger.Debug("user_aggregates TagUnlockReadPool", zap.Int("events", len(*rpls)))
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
		logging.Logger.Debug("user_aggregates TagLockWritePool", zap.Int("events", len(*wpls)))
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
		logging.Logger.Debug("user_aggregates TagUnlockWritePool", zap.Int("events", len(*wpls)))
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
		logging.Logger.Debug("user_aggregates TagLockStakePool", zap.Int("events", len(*dpls)))
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
		logging.Logger.Debug("user_aggregates TagUnlockStakePool", zap.Int("events", len(*dpls)))
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
		logging.Logger.Debug("user_aggregates TagUpdateUserPayedFees", zap.Int("events", len(*users)))
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
		logging.Logger.Debug("user_aggregates TagUpdateUserCollectedRewards", zap.Int("events", len(*users)))
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
	var mappedAggrs = make(map[string]*UserAggregate, len(ids))

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
		u := aggr
		mappedAggrs[u.UserID] = &u
	}

	return mappedAggrs, nil
}

func (edb *EventDb) updateUserAggregates(e *blockEvents) error {
	logging.Logger.Debug("calculating user_aggregates", zap.Int64("round", e.round))
	var updatedAggrs []UserAggregate
	for _, ev := range e.events {
		if h, ok := handlers[ev.Tag]; ok {
			aggrs := h(ev)
			updatedAggrs = append(updatedAggrs, aggrs...)
		}
	}

	idSet := make(map[string]bool)
	for _, aggr := range updatedAggrs {
		idSet[aggr.UserID] = true
	}
	uniqueIds := make([]string, 0, len(idSet))
	for id := range idSet {
		uniqueIds = append(uniqueIds, id)
	}

	// load user snapshots  
	snaps, err := edb.GetUserSnapshotsByIds(uniqueIds)
	if err != nil {
		logging.Logger.Error("can't load latest snapshots", zap.Error(err))
		return err
	}

	snapsMap := make(map[string]UserSnapshot, len(updatedAggrs))
	for _, snap := range snaps {
		snapsMap[snap.UserID] = snap
	}

	for _, aggr := range updatedAggrs {
		snap, ok := snapsMap[aggr.UserID]
		if !ok {
			snapsMap[aggr.UserID] = UserSnapshot{
				UserID:          aggr.UserID,
				CollectedReward: aggr.CollectedReward,
				PayedFees:       aggr.PayedFees,
				TotalStake:      aggr.TotalStake,
				ReadPoolTotal:   aggr.ReadPoolTotal,
				WritePoolTotal:  aggr.WritePoolTotal,
			}
			continue
		}
		merge(&snap, &aggr)
		snap.UpdatedAt = time.Now()
		snapsMap[aggr.UserID] = snap
	}

	newAggregates := make(map[string]*UserAggregate, len(snapsMap))
	for _, snap := range snapsMap {
		newAggregates[snap.UserID] = &UserAggregate{
			Round:           snap.Round,
			UserID:          snap.UserID,
			CollectedReward: snap.CollectedReward,
			PayedFees:       snap.PayedFees,
			TotalStake:      snap.TotalStake,
			ReadPoolTotal:   snap.ReadPoolTotal,
			WritePoolTotal:  snap.WritePoolTotal,
		}
	}
	err = edb.addUserAggregates(newAggregates)
	if err != nil {
		logging.Logger.Error("saving user aggregate failed", zap.Error(err))
		return err
	}

	updatedSnaps := make([]UserSnapshot, 0, len(snapsMap))
	for _, snap := range snapsMap {
		updatedSnaps = append(updatedSnaps, snap)
	}
	err = edb.AddOrOverwriteUserSnapshots(updatedSnaps)
	if err != nil {
		logging.Logger.Error("saving user aggregate snapshots failed", zap.Error(err))
		return err
	}

	return nil
}

func merge(a *UserSnapshot, u *UserAggregate) {
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
