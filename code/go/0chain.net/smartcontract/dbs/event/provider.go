package event

import (
	"fmt"
	"math/big"
	"reflect"
	"sort"
	"time"

	"github.com/0chain/common/core/logging"
	"github.com/lib/pq"
	"go.uber.org/zap"

	"0chain.net/smartcontract/stakepool/spenum"

	"0chain.net/core/common"
	"0chain.net/smartcontract/dbs"
	"github.com/0chain/common/core/currency"
	"gorm.io/gorm"
)

type IProvider interface {
	GetID() string
	TableName() string
}

type ProviderIdsMap map[spenum.Provider]map[string]interface{}
type ProvidersMap map[spenum.Provider]map[string]IProvider

var ProviderTextMapping = map[reflect.Type]string{
	reflect.TypeOf(Blobber{}):    "blobber",
	reflect.TypeOf(Sharder{}):    "sharder",
	reflect.TypeOf(Miner{}):      "miner",
	reflect.TypeOf(Validator{}):  "validator",
	reflect.TypeOf(Authorizer{}): "authorizer",
}

type Provider struct {
	ID              string `gorm:"primaryKey"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DelegateWallet  string           `json:"delegate_wallet"`
	NumDelegates    int              `json:"num_delegates"`
	ServiceCharge   float64          `json:"service_charge"`
	TotalStake      currency.Coin    `json:"total_stake"`
	Rewards         ProviderRewards  `json:"rewards" gorm:"foreignKey:ProviderID"`
	Downtime        uint64           `json:"downtime"`
	LastHealthCheck common.Timestamp `json:"last_health_check"`
	IsKilled        bool             `json:"is_killed"`
	IsShutdown      bool             `json:"is_shutdown"`
}

type ProviderAggregate interface {
	GetTotalStake() currency.Coin
	GetServiceCharge() float64
	GetTotalRewards() currency.Coin
	SetTotalStake(value currency.Coin)
	SetServiceCharge(value float64)
	SetTotalRewards(value currency.Coin)
}

func (p *Provider) GetID() string {
	return p.ID
}

func (p *Provider) IsOffline() bool {
	return p.IsKilled || p.IsShutdown
}

func (p *Provider) BeforeCreate(tx *gorm.DB) (err error) {
	intID := new(big.Int)
	intID.SetString(p.ID, 16)

	return
}

func (edb *EventDb) updateProviderTotalStakes(providers []Provider, tablename string) error {
	var ids []string
	var stakes []int64
	for _, m := range providers {
		ids = append(ids, m.ID)
		i, err := m.TotalStake.Int64()
		if err != nil {
			return err
		}
		stakes = append(stakes, i)
	}

	return CreateBuilder(tablename, "id", ids).
		AddUpdate("total_stake", stakes).Exec(edb).Error
}

func (edb *EventDb) updateProvidersHealthCheck(updates []dbs.DbHealthCheck, tableName ProviderTable) error {
	table := string(tableName)

	var ids []string
	var lastHealthCheck []int64
	var downtime []int64
	for _, u := range updates {
		ids = append(ids, u.ID)
		lastHealthCheck = append(lastHealthCheck, int64(u.LastHealthCheck))
		downtime = append(downtime, int64(u.Downtime))
	}

	return CreateBuilder(table, "id", ids).
		AddUpdate("downtime", downtime, table+".downtime + t.downtime").
		AddUpdate("last_health_check", lastHealthCheck).Exec(edb).Error
}

func (edb *EventDb) BuildChangedProvidersMapFromEvents(events []Event) (ProvidersMap, error) {
	ids, err := extractIdsFromEvents(events)
	if err != nil {
		return nil, err
	}

	providers := ProvidersMap{
		spenum.Blobber:    make(map[string]IProvider),
		spenum.Miner:      make(map[string]IProvider),
		spenum.Sharder:    make(map[string]IProvider),
		spenum.Authorizer: make(map[string]IProvider),
		spenum.Validator:  make(map[string]IProvider),
	}

	idsLists := map[spenum.Provider][]string{
		spenum.Blobber:    make([]string, 0, len(ids[spenum.Blobber])),
		spenum.Miner:      make([]string, 0, len(ids[spenum.Miner])),
		spenum.Sharder:    make([]string, 0, len(ids[spenum.Sharder])),
		spenum.Authorizer: make([]string, 0, len(ids[spenum.Authorizer])),
		spenum.Validator:  make([]string, 0, len(ids[spenum.Validator])),
	}

	for provider, pids := range ids {
		for id := range pids {
			idsLists[provider] = append(idsLists[provider], id)
		}
	}

	providersLists := make(map[spenum.Provider][]IProvider)
	for provider, ids := range idsLists {
		providersLists[provider], err = edb.GetProvidersByIds(provider, ids)
		if err != nil {
			return nil, err
		}
	}
	for ptype, providersList := range providersLists {
		for _, provider := range providersList {
			providers[ptype][provider.GetID()] = provider
		}
	}

	return providers, nil
}

func (edb *EventDb) GetProvidersByIds(ptype spenum.Provider, ids []string) ([]IProvider, error) {
	switch ptype {
	case spenum.Blobber:
		return getProvidersById[*Blobber](edb, ids)
	case spenum.Miner:
		return getProvidersById[*Miner](edb, ids)
	case spenum.Sharder:
		return getProvidersById[*Sharder](edb, ids)
	case spenum.Authorizer:
		return getProvidersById[*Authorizer](edb, ids)
	case spenum.Validator:
		return getProvidersById[*Validator](edb, ids)
	}

	return nil, common.NewError("invalid_provider_type", "invalid provider type")
}

func getProvidersById[P IProvider](edb *EventDb, ids []string) ([]IProvider, error) {
	var (
		model     P
		providers []P
		tableName = model.TableName()
	)

	err := edb.Get().
		Model(&model).
		Joins("Rewards").
		Joins(fmt.Sprintf("INNER JOIN unnest(?::text[]) as ids(id) on ids.id = %v.id", tableName), pq.Array(ids)).
		Find(&providers).Debug().Error
	if err != nil {
		return nil, err
	}

	ips := make([]IProvider, len(providers))
	for i, p := range providers {
		ips[i] = p
	}

	return ips, nil
}

func extractIdsFromEvents(events []Event) (ProviderIdsMap, error) {
	ids := map[spenum.Provider]map[string]interface{}{
		spenum.Blobber:    {},
		spenum.Miner:      {},
		spenum.Sharder:    {},
		spenum.Validator:  {},
		spenum.Authorizer: {},
	}
	for _, event := range events {
		switch event.Tag {
		case TagAddBlobber,
			TagUpdateBlobber,
			TagUpdateBlobberAllocatedSavedHealth,
			TagUpdateBlobberTotalStake,
			TagUpdateBlobberTotalOffers,
			TagUpdateBlobberStat:
			blobbers, ok := fromEvent[[]Blobber](event.Data)
			if !ok {
				logging.Logger.Error("snapshot",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				return nil, common.NewError("update_snapshot", fmt.Sprintf("invalid data for event %s", event.Tag.String()))
			}
			for _, b := range *blobbers {
				ids[spenum.Blobber][b.ID] = nil
			}
		case TagAddMiner,
			TagUpdateMinerTotalStake:
			miners, ok := fromEvent[[]Miner](event.Data)
			if !ok {
				logging.Logger.Error("snapshot",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				return nil, common.NewError("update_snapshot", fmt.Sprintf("invalid data for event %s", event.Tag.String()))
			}
			for _, m := range *miners {
				ids[spenum.Miner][m.ID] = nil
			}
		case TagUpdateMiner:
			updates, ok := fromEvent[[]dbs.DbUpdates](event.Data)
			if !ok {
				logging.Logger.Error("snapshot",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				return nil, common.NewError("update_snapshot", fmt.Sprintf("invalid data for event %s", event.Tag.String()))
			}
			for _, update := range *updates {
				ids[spenum.Miner][update.Id] = nil
			}
		case TagAddSharder,
			TagUpdateSharderTotalStake:
			sharders, ok := fromEvent[[]Sharder](event.Data)
			if !ok {
				logging.Logger.Error("snapshot",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				return nil, common.NewError("update_snapshot", fmt.Sprintf("invalid data for event %s", event.Tag.String()))
			}
			for _, s := range *sharders {
				ids[spenum.Sharder][s.ID] = nil
			}
		case TagUpdateSharder:
			updates, ok := fromEvent[[]dbs.DbUpdates](event.Data)
			if !ok {
				logging.Logger.Error("snapshot",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				return nil, common.NewError("update_snapshot", fmt.Sprintf("invalid data for event %s", event.Tag.String()))
			}

			for _, update := range *updates {
				ids[spenum.Sharder][update.Id] = nil
			}

		case TagAddAuthorizer,
			TagUpdateAuthorizer,
			TagUpdateAuthorizerTotalStake:
			authorizers, ok := fromEvent[[]Authorizer](event.Data)
			if !ok {
				logging.Logger.Error("snapshot",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				return nil, common.NewError("update_snapshot", fmt.Sprintf("invalid data for event %s", event.Tag.String()))
			}
			for _, a := range *authorizers {
				ids[spenum.Authorizer][a.ID] = nil
			}
		case TagAddOrOverwiteValidator,
			TagUpdateValidator,
			TagUpdateValidatorStakeTotal:
			validators, ok := fromEvent[[]Validator](event.Data)
			if !ok {
				logging.Logger.Error("snapshot",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				return nil, common.NewError("update_snapshot", fmt.Sprintf("invalid data for event %s", event.Tag.String()))
			}
			for _, v := range *validators {
				ids[spenum.Validator][v.ID] = nil
			}
		case TagUpdateBlobberChallenge:
			challenges, ok := fromEvent[[]ChallengeStatsDeltas](event.Data)
			if !ok {
				logging.Logger.Error("snapshot",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				return nil, common.NewError("update_snapshot",
					fmt.Sprintf("invalid data for event %s", event.Tag.String()))
			}
			for _, c := range *challenges {
				ids[spenum.Blobber][c.Id] = nil
			}
		case TagStakePoolReward:
			spus, ok := fromEvent[[]dbs.StakePoolReward](event.Data)
			if !ok {
				logging.Logger.Error("snapshot",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				return nil, common.NewError("update_snapshot", fmt.Sprintf("invalid data for event %s", event.Tag.String()))
			}
			for _, spu := range *spus {
				idsMap, ok := ids[spu.ProviderID.Type]
				if !ok {
					logging.Logger.Warn("BuildChangedProvidersMapFromEvents - StakePoolUpdate ignored, unknown provider type",
						zap.String("provider_id", spu.ProviderID.ID),
						zap.Any("provider_type", spu.ProviderID.Type))
					continue
				}

				idsMap[spu.ProviderID.ID] = nil
			}
		case TagStakePoolPenalty:
			spus, ok := fromEvent[[]dbs.StakePoolReward](event.Data)
			if !ok {
				logging.Logger.Error("snapshot",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				return nil, common.NewError("update_snapshot", fmt.Sprintf("invalid data for event %s", event.Tag.String()))
			}
			for _, spu := range *spus {
				idsMap, ok := ids[spu.ProviderID.Type]
				if !ok {
					logging.Logger.Warn("BuildChangedProvidersMapFromEvents - StakePoolUpdate ignored, unknown provider type",
						zap.String("provider_id", spu.ProviderID.ID),
						zap.Any("provider_type", spu.ProviderID.Type))
					continue
				}

				idsMap[spu.ProviderID.ID] = nil
			}
		case TagCollectProviderReward:
			// Since we don't know the type, we'll need to add it to all maps
			pid, ok := event.Data.(dbs.ProviderID)
			if !ok {
				logging.Logger.Error("snapshot",
					zap.Any("event", event.Data), zap.Error(ErrInvalidEventData))
				return nil, common.NewError("update_snapshot", fmt.Sprintf("invalid data for event %s", event.Tag.String()))
			}

			idMap, ok := ids[pid.Type]
			if !ok {
				logging.Logger.Warn("BuildChangedProvidersMapFromEvents - CollectProviderReward ignored, unknown provider type",
					zap.String("provider_id", pid.ID),
					zap.Any("provider_type", pid.Type))
				continue
			}

			idMap[pid.ID] = nil
		case TagBlobberHealthCheck:
			healthCheckUpdates, ok := fromEvent[[]dbs.DbHealthCheck](event.Data)
			if !ok {
				logging.Logger.Error("snapshot",
					zap.Int("tag", event.Tag.Int()), zap.Any("data", event.Data), zap.Error(ErrInvalidEventData))
				return nil, common.NewError("update_snapshot", fmt.Sprintf("invalid data for event %s", event.Tag.String()))
			}
			idMap := ids[spenum.Blobber]
			for _, hcu := range *healthCheckUpdates {
				idMap[hcu.ID] = nil
			}
		case TagMinerHealthCheck:
			healthCheckUpdates, ok := fromEvent[[]dbs.DbHealthCheck](event.Data)
			if !ok {
				logging.Logger.Error("snapshot",
					zap.Int("tag", event.Tag.Int()), zap.Any("data", event.Data), zap.Error(ErrInvalidEventData))
				return nil, common.NewError("update_snapshot", fmt.Sprintf("invalid data for event %s", event.Tag.String()))
			}
			idMap := ids[spenum.Miner]
			for _, hcu := range *healthCheckUpdates {
				idMap[hcu.ID] = nil
			}
		case TagSharderHealthCheck:
			healthCheckUpdates, ok := fromEvent[[]dbs.DbHealthCheck](event.Data)
			if !ok {
				logging.Logger.Error("snapshot",
					zap.Int("tag", event.Tag.Int()), zap.Any("data", event.Data), zap.Error(ErrInvalidEventData))
				return nil, common.NewError("update_snapshot", fmt.Sprintf("invalid data for event %s", event.Tag.String()))
			}
			idMap := ids[spenum.Sharder]
			for _, hcu := range *healthCheckUpdates {
				idMap[hcu.ID] = nil
			}
		case TagAuthorizerHealthCheck:
			healthCheckUpdates, ok := fromEvent[[]dbs.DbHealthCheck](event.Data)
			if !ok {
				logging.Logger.Error("snapshot",
					zap.Int("tag", event.Tag.Int()), zap.Any("data", event.Data), zap.Error(ErrInvalidEventData))
				return nil, common.NewError("update_snapshot", fmt.Sprintf("invalid data for event %s", event.Tag.String()))
			}
			idMap := ids[spenum.Authorizer]
			for _, hcu := range *healthCheckUpdates {
				idMap[hcu.ID] = nil
			}
		case TagValidatorHealthCheck:
			healthCheckUpdates, ok := fromEvent[[]dbs.DbHealthCheck](event.Data)
			if !ok {
				logging.Logger.Error("snapshot",
					zap.Int("tag", event.Tag.Int()), zap.Any("data", event.Data), zap.Error(ErrInvalidEventData))
				return nil, common.NewError("update_snapshot", fmt.Sprintf("invalid data for event %s", event.Tag.String()))
			}
			idMap := ids[spenum.Validator]
			for _, hcu := range *healthCheckUpdates {
				idMap[hcu.ID] = nil
			}
		case TagShutdownProvider:
			pids, ok := fromEvent[[]dbs.ProviderID](event.Data)
			if !ok {
				return nil, common.NewError("update_snapshot", fmt.Sprintf("invalid data for event %s", event.Tag.String()))
			}
			for _, pid := range *pids {
				idMap, ok := ids[pid.Type]
				if !ok {
					logging.Logger.Warn("BuildChangedProvidersMapFromEvents - ShutdownProvider ignored, unknown provider type",
						zap.String("provider_id", pid.ID),
						zap.Any("provider_type", pid.Type))
					continue
				}
				idMap[pid.ID] = nil
			}
		case TagKillProvider:
			pids, ok := fromEvent[[]dbs.ProviderID](event.Data)
			if !ok {
				return nil, common.NewError("update_snapshot", fmt.Sprintf("invalid data for event %s", event.Tag.String()))
			}
			for _, pid := range *pids {
				idMap, ok := ids[pid.Type]
				if !ok {
					logging.Logger.Warn("BuildChangedProvidersMapFromEvents - KillProvider ignored, unknown provider type",
						zap.String("provider_id", pid.ID),
						zap.Any("provider_type", pid.Type))
					continue
				}
				idMap[pid.ID] = nil
			}

		}
	}

	return ids, nil
}

func providerToTableName(pType spenum.Provider) string {
	return pType.String() + "s"
}

func mapProviders(
	providers []dbs.ProviderID,
) map[spenum.Provider][]string {
	idSlices := make(map[spenum.Provider][]string, 5)
	for _, provider := range providers {
		var ids []string
		ids = idSlices[provider.Type]
		ids = append(ids, provider.ID)
		idSlices[provider.Type] = ids
	}
	return idSlices
}

func (edb *EventDb) providersSetBoolean(providers []dbs.ProviderID, field string, value bool) error {
	mappedProviders := mapProviders(providers)
	sortedTypes := sortProviderTypes(mappedProviders)
	for _, pType := range sortedTypes {
		ids := mappedProviders[pType]
		table := providerToTableName(pType)
		var values []bool
		for i := 0; i < len(ids); i++ {
			values = append(values, value)
		}
		if err := edb.setBoolean(table, ids, field, values); err != nil {
			logging.Logger.Error("updating boolean field "+table+"."+field,
				zap.Error(err))
		}
	}
	return nil
}

func (edb *EventDb) setBoolean(
	table string,
	ids []string,
	column string,
	values []bool,
) error {
	return CreateBuilder(table, "id", ids).
		AddUpdate(column, values).
		Exec(edb).Error
}

func sortProviderTypes(m map[spenum.Provider][]string) []spenum.Provider {
	var keys []spenum.Provider
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})
	return keys
}
