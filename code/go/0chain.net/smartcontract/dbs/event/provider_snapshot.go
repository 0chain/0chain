package event

import (
	"fmt"

	"0chain.net/core/common"
	"0chain.net/smartcontract/stakepool/spenum"
)

func (edb *EventDb) CreateNewProviderSnapshots(providers ProvidersMap, round int64) error {
	blobbers := make([]*Blobber, 0, len(providers[spenum.Blobber]))
	for _, p := range providers[spenum.Blobber] {
		b, ok := p.(*Blobber)
		if !ok {
			return common.NewError("invalid blobber provider type", fmt.Sprintf("%v", p))
		}
		blobbers = append(blobbers, b)
	}
	err := edb.addBlobberSnapshot(blobbers, round)
	if err != nil {
		return common.NewError("failed to create blobber aggregates", err.Error())
	}

	miners := make([]*Miner, 0, len(providers[spenum.Miner]))
	for _, p := range providers[spenum.Miner] {
		m, ok := p.(*Miner)
		if !ok {
			return common.NewError("invalid miner provider type", fmt.Sprintf("%v", p))
		}
		miners = append(miners, m)
	}
	err = edb.addMinerSnapshot(miners, round)
	if err != nil {
		return common.NewError("failed to create miner aggregates", err.Error())
	}

	sharders := make([]*Sharder, 0, len(providers[spenum.Sharder]))
	for _, p := range providers[spenum.Sharder] {
		s, ok := p.(*Sharder)
		if !ok {
			return common.NewError("invalid sharder provider type", fmt.Sprintf("%v", p))
		}
		sharders = append(sharders, s)
	}
	err = edb.addSharderSnapshot(sharders, round)
	if err != nil {
		return common.NewError("failed to create sharder aggregates", err.Error())
	}

	authorizers := make([]*Authorizer, 0, len(providers[spenum.Authorizer]))
	for _, p := range providers[spenum.Authorizer] {
		a, ok := p.(*Authorizer)
		if !ok {
			return common.NewError("invalid authorizer provider type", fmt.Sprintf("%v", p))
		}
		authorizers = append(authorizers, a)
	}
	err = edb.addAuthorizerSnapshot(authorizers, round)
	if err != nil {
		return common.NewError("failed to create authorizer aggregates", err.Error())
	}

	validators := make([]*Validator, 0, len(providers[spenum.Validator]))
	for _, p := range providers[spenum.Validator] {
		v, ok := p.(*Validator)
		if !ok {
			return common.NewError("invalid validator provider type", fmt.Sprintf("%v", p))
		}
		validators = append(validators, v)
	}
	err = edb.addValidatorSnapshot(validators, round)
	if err != nil {
		return common.NewError("failed to create validator aggregates", err.Error())
	}

	return nil
} 