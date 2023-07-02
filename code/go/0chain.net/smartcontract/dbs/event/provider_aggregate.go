package event

import (
	"errors"
	"reflect"

	"0chain.net/smartcontract/dbs"
	"0chain.net/smartcontract/stakepool/spenum"
)

type EventProvidersIdsExtractorsMap map[EventTag]func(e Event) ([]string, error)
type ProviderAggregateCreator func(edb *EventDb, providers interface{}, round int64) (error)

type IProvider interface {
	GetID() string
}

var ProviderTextMapping = map[reflect.Type]string{
	reflect.TypeOf(Blobber{}): "blobber",
	reflect.TypeOf(Sharder{}): "sharder",
	reflect.TypeOf(Miner{}):  "miner",
	reflect.TypeOf(Validator{}): "validator",
	reflect.TypeOf(Authorizer{}): "authorizer",
}

var providerEventHandlers = map[reflect.Type]EventProvidersIdsExtractorsMap{
	reflect.TypeOf(Blobber{}): {
		TagAddBlobber: extractProvidersIdsFromProvider[Blobber],
		TagUpdateBlobber: extractProvidersIdsFromProvider[Blobber],
		TagUpdateBlobberAllocatedSavedHealth: extractProvidersIdsFromProvider[Blobber],
		TagUpdateBlobberTotalStake: extractProvidersIdsFromProvider[Blobber],
		TagUpdateBlobberTotalOffers: extractProvidersIdsFromProvider[Blobber],
		TagStakePoolReward: extractProvidersIdsFromSPUs,
		TagStakePoolPenalty: extractProvidersIdsFromSPUs,
		TagMintReward: extractProviderIdFromRewards[Blobber],
		TagUpdateBlobberOpenChallenges: extractProvidersIdsFromProvider[Blobber],
		TagUpdateBlobberChallenge: extractProvidersIdsFromProvider[Blobber],
		TagUpdateBlobberStat: extractProvidersIdsFromProvider[Blobber],
		TagBlobberHealthCheck: extractProvidersIdsFromHealthCheck,
		TagShutdownProvider: extractProvidersIdsFromDbsProviderId[Blobber],
		TagKillProvider: extractProvidersIdsFromDbsProviderId[Blobber],
		TagCollectProviderReward: extractProviderIdFromEventIndex,
	},
	reflect.TypeOf(Sharder{}): {
		TagAddSharder: extractProvidersIdsFromProvider[Sharder],
		TagUpdateSharder: extractProvidersIdsFromProvider[Sharder],
		TagUpdateSharderTotalStake: extractProvidersIdsFromProvider[Sharder],
		TagStakePoolReward: extractProvidersIdsFromSPUs,
		TagStakePoolPenalty: extractProvidersIdsFromSPUs,
		TagMintReward: extractProviderIdFromRewards[Sharder],
		TagCollectProviderReward: extractProviderIdFromEventIndex,
		TagSharderHealthCheck: extractProvidersIdsFromHealthCheck,
		TagShutdownProvider: extractProvidersIdsFromDbsProviderId[Sharder],
		TagKillProvider: extractProvidersIdsFromDbsProviderId[Sharder],
	},
	reflect.TypeOf(Miner{}): {
		TagAddMiner: extractProvidersIdsFromProvider[Miner],
		TagUpdateMiner: extractProvidersIdsFromProvider[Miner],
		TagUpdateMinerTotalStake: extractProvidersIdsFromProvider[Miner],
		TagStakePoolReward: extractProvidersIdsFromSPUs,
		TagStakePoolPenalty: extractProvidersIdsFromSPUs,
		TagMintReward: extractProviderIdFromRewards[Miner],
		TagCollectProviderReward: extractProviderIdFromEventIndex,
		TagMinerHealthCheck: extractProvidersIdsFromHealthCheck,
		TagShutdownProvider: extractProvidersIdsFromDbsProviderId[Miner],
		TagKillProvider: extractProvidersIdsFromDbsProviderId[Miner],
	},
	reflect.TypeOf(Validator{}): {
		TagAddOrOverwiteValidator: extractProvidersIdsFromProvider[Validator],
		TagUpdateValidator: extractProvidersIdsFromProvider[Validator],
		TagUpdateValidatorStakeTotal: extractProvidersIdsFromProvider[Validator],
		TagStakePoolReward: extractProvidersIdsFromSPUs,
		TagStakePoolPenalty: extractProvidersIdsFromSPUs,
		TagMintReward: extractProviderIdFromRewards[Validator],
		TagCollectProviderReward: extractProviderIdFromEventIndex,
		TagShutdownProvider: extractProvidersIdsFromDbsProviderId[Validator],
		TagKillProvider: extractProvidersIdsFromDbsProviderId[Validator],
	},
	reflect.TypeOf(Authorizer{}): {
		TagAddAuthorizer: extractProvidersIdsFromProvider[Authorizer],
		TagUpdateAuthorizer: extractProvidersIdsFromProvider[Authorizer],
		TagUpdateAuthorizerTotalStake: extractProvidersIdsFromProvider[Authorizer],
		TagStakePoolReward: extractProvidersIdsFromSPUs,
		TagStakePoolPenalty: extractProvidersIdsFromSPUs,
		TagMintReward: extractProviderIdFromRewards[Authorizer],
		TagCollectProviderReward: extractProviderIdFromEventIndex,
		TagShutdownProvider: extractProvidersIdsFromDbsProviderId[Authorizer],
		TagKillProvider: extractProvidersIdsFromDbsProviderId[Authorizer],
	},
}

var providerAggregatesCreators = map[reflect.Type]ProviderAggregateCreator {
	reflect.TypeOf(Blobber{}): func(edb *EventDb, providers interface{}, round int64) error {
		blobbers, ok := providers.([]Blobber)
		if !ok {
			return errors.New("invalid providers")
		}

		return edb.CreateBlobberAggregates(blobbers, round)
	},
	reflect.TypeOf(Sharder{}): func(edb *EventDb, providers interface{}, round int64) error {
		sharders, ok := providers.([]Sharder)
		if !ok {
			return errors.New("invalid providers")
		}

		return edb.CreateSharderAggregates(sharders, round)
	},
	reflect.TypeOf(Miner{}): func(edb *EventDb, providers interface{}, round int64) error {
		miners, ok := providers.([]Miner)
		if !ok {
			return errors.New("invalid providers")
		}

		return edb.CreateMinerAggregates(miners, round)
	},
}

func updateProviderAggregates[P any](edb *EventDb, e *blockEvents) error {
	var pModel P

	// 1. Scan events for target provider Ids
	idExtractors := providerEventHandlers[reflect.TypeOf(pModel)]
	var targetIds []string
	for _, event := range e.events {
		if extractor, ok := idExtractors[event.Tag]; ok {
			ids, err := extractor(event)
			if err != nil {
				return err
			}
			targetIds = append(targetIds, ids...)
		}
	}

	// 2. Get actual provider data from the db, will return only unique ones so no need to remove duplicates from ids
	var providers []P
	err := edb.Get().Model(&pModel).
		Where("id IN (?)", targetIds).
		Find(&providers).Error;
	if err != nil {
		return err
	}
	
	// 3. Create and store aggregates
	return providerAggregatesCreators[reflect.TypeOf(pModel)](edb, providers, e.round)
}

func extractProviderIdFromEventIndex(e Event) ([]string, error) {
	return []string{e.Index}, nil
}

func extractProvidersIdsFromProvider[P IProvider](e Event) ([]string, error) {
	providers, ok := e.Data.([]P)
	if !ok {
		return nil, ErrInvalidEventData
	}
	ids := make([]string, 0, len(providers))
	for _, b := range providers {
		ids = append(ids, b.GetID())
	}
	return ids, nil
}

func extractProvidersIdsFromSPUs(e Event) ([]string, error) {
	spus, ok := e.Data.([]dbs.StakePoolReward)
	if !ok {
		return nil, ErrInvalidEventData
	}
	ids := make([]string, 0, len(spus))
	for _, spu := range spus {
		ids = append(ids, spu.ProviderID.ID)
	}
	return ids, nil
}

func extractProviderIdFromRewards[T any](e Event) ([]string, error) {
	reward, ok := e.Data.(RewardMint)
	if !ok {
		return nil, ErrInvalidEventData
	}

	var model T
	if reward.ProviderType != ProviderTextMapping[reflect.TypeOf(model)] {
		return nil, nil
	}
	return []string{reward.ProviderID}, nil
}

func extractProvidersIdsFromHealthCheck(e Event) ([]string, error) {
	healthCheck, ok := e.Data.([]dbs.DbHealthCheck)
	if !ok {
		return nil, ErrInvalidEventData
	}
	ids := make([]string, 0, len(healthCheck))
	for _, hc := range healthCheck {
		ids = append(ids, hc.ID)
	}
	return ids, nil
}

func extractProvidersIdsFromDbsProviderId[T any](e Event) ([]string, error) {
	providerIds, ok := e.Data.([]dbs.ProviderID)
	if !ok {
		return nil, ErrInvalidEventData
	}

	var model T
	ptype := spenum.ToProviderType(ProviderTextMapping[reflect.TypeOf(model)])

	ids := make([]string, 0, len(providerIds))
	for _, pid := range providerIds {
		if pid.Type == ptype {
			ids = append(ids, pid.ID)
		}
	}
	return ids, nil
}
