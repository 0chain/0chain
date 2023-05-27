//go:build !integration_tests
// +build !integration_tests

package event

import (
	"fmt"

	"0chain.net/chaincore/state"
	"0chain.net/smartcontract/dbs"
	"github.com/0chain/common/core/currency"
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
)

func (edb *EventDb) addStatMain(event Event) (err error) {
	switch event.Tag {
	// blobber
	case TagAddBlobber:
		blobbers, ok := fromEvent[[]Blobber](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.addBlobbers(*blobbers)
	case TagUpdateBlobber:
		blobbers, ok := fromEvent[[]Blobber](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.updateBlobber(*blobbers)
	case TagUpdateBlobberAllocatedSavedHealth:
		blobbers, ok := fromEvent[[]Blobber](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.updateBlobbersAllocatedSavedAndHealth(*blobbers)
	case TagUpdateBlobberTotalStake:
		bs, ok := fromEvent[[]Blobber](event.Data)
		if !ok {
			return ErrInvalidEventData
		}

		return edb.updateBlobbersTotalStakes(*bs)
	case TagUpdateBlobberTotalOffers:
		bs, ok := fromEvent[[]Blobber](event.Data)
		if !ok {
			return ErrInvalidEventData
		}

		return edb.updateBlobbersTotalOffers(*bs)
	case TagDeleteBlobber:
		blobberID, ok := fromEvent[string](event.Data)

		if !ok {
			return ErrInvalidEventData
		}
		return edb.deleteBlobber(*blobberID)
	// authorizer
	case TagAddAuthorizer:
		auth, ok := fromEvent[Authorizer](event.Data)

		if !ok {
			return ErrInvalidEventData
		}
		return edb.AddAuthorizer(auth)
	case TagDeleteAuthorizer:
		id, ok := event.Data.(string)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.DeleteAuthorizer(id)
	case TagUpdateAuthorizerTotalStake:
		as, ok := fromEvent[[]Authorizer](event.Data)
		if !ok {
			return ErrInvalidEventData
		}

		return edb.updateAuthorizersTotalStakes(*as)
	case TagAddWriteMarker:
		wms, ok := fromEvent[[]WriteMarker](event.Data)
		if !ok {
			return ErrInvalidEventData
		}

		for i := range *wms {
			(*wms)[i].BlockNumber = event.BlockNumber
		}

		if err := edb.addWriteMarkers(*wms); err != nil {
			return err
		}
		return nil
	case TagAddReadMarker:
		rms, ok := fromEvent[[]ReadMarker](event.Data)
		if !ok {
			return ErrInvalidEventData
		}

		for i := range *rms {
			(*rms)[i].BlockNumber = event.BlockNumber
		}
		return edb.addOrOverwriteReadMarker(*rms)
	case TagAddOrOverwriteUser:
		users, ok := fromEvent[[]User](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.addOrUpdateUsers(*users)
	case TagAddTransactions:
		txns, ok := fromEvent[[]Transaction](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.addTransactions(*txns)
	case TagAddBlock:
		block, ok := fromEvent[Block](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.addOrUpdateBlock(*block)
	case TagFinalizeBlock:
		block, ok := fromEvent[Block](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.addOrUpdateBlock(*block)
	case TagAddOrOverwiteValidator:
		vns, ok := fromEvent[[]Validator](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.addOrOverwriteValidators(*vns)
	case TagUpdateValidator:
		updates, ok := fromEvent[[]Validator](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.updateValidators(*updates)
	case TagUpdateValidatorStakeTotal:
		updates, ok := fromEvent[[]Validator](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.updateValidatorTotalStakes(*updates)
	case TagAddMiner:
		miners, ok := fromEvent[[]Miner](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.addMiner(*miners)
	case TagUpdateMiner:
		updates, ok := fromEvent[dbs.DbUpdates](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.updateMiner(*updates)
	case TagDeleteMiner:
		minerID, ok := fromEvent[string](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.deleteMiner(*minerID)
	case TagAddSharder:
		sharders, ok := fromEvent[[]Sharder](event.Data)
		if !ok {
			return ErrInvalidEventData
		}

		return edb.addSharders(*sharders)
	case TagUpdateMinerTotalStake:
		m, ok := fromEvent[[]Miner](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.updateMinersTotalStakes(*m)
	case TagUpdateSharder:
		updates, ok := fromEvent[dbs.DbUpdates](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.updateSharder(*updates)
	case TagDeleteSharder:
		sharderID, ok := fromEvent[string](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.deleteSharder(*sharderID)
	case TagUpdateSharderTotalStake:
		s, ok := fromEvent[[]Sharder](event.Data)
		if !ok {
			return ErrInvalidEventData
		}

		return edb.updateShardersTotalStakes(*s)
	//stake pool
	case TagAddDelegatePool:
		dps, ok := fromEvent[[]DelegatePool](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.addDelegatePools(*dps)
	case TagUpdateDelegatePool:
		spUpdate, ok := fromEvent[dbs.DelegatePoolUpdate](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.updateDelegatePool(*spUpdate)
	case TagStakePoolReward:
		spus, ok := fromEvent[[]dbs.StakePoolReward](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		if err := edb.rewardUpdate(*spus, event.BlockNumber); err != nil {
			return err
		}
		if err := edb.blobberSpecificRevenue(*spus); err != nil {
			return fmt.Errorf("could not update blobber specific revenue: %v", err)
		}
		return nil
	case TagStakePoolPenalty:
		spus, ok := fromEvent[[]dbs.StakePoolReward](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		err := edb.penaltyUpdate(*spus, event.BlockNumber)
		if err != nil {
			return err
		}
		err = edb.blobberSpecificRevenue(*spus)
		if err != nil {
			return fmt.Errorf("could not update blobber specific revenue: %v", err)
		}
		return nil
	case TagAddAllocation:
		allocs, ok := fromEvent[[]Allocation](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.addAllocations(*allocs)
	case TagUpdateAllocation:
		allocs, ok := fromEvent[[]Allocation](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.updateAllocations(*allocs)
	case TagUpdateAllocationStakes:
		allocs, ok := fromEvent[[]Allocation](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.updateAllocationStakes(*allocs)
	case TagMintReward:
		reward, ok := fromEvent[RewardMint](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.addRewardMint(*reward)
	case TagAddChallenge:
		challenges, ok := fromEvent[[]Challenge](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.addChallenges(*challenges)
	case TagAddChallengeToAllocation:
		as, ok := fromEvent[[]Allocation](event.Data)
		if !ok {
			return ErrInvalidEventData
		}

		return edb.addChallengesToAllocations(*as)
	case TagUpdateBlobberOpenChallenges:
		updates, ok := fromEvent[[]Blobber](event.Data)
		if !ok {
			return ErrInvalidEventData
		}

		return edb.updateOpenBlobberChallenges(*updates)
	case TagUpdateChallenge:
		chs, ok := fromEvent[[]Challenge](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.updateChallenges(*chs)
	case TagUpdateBlobberChallenge:
		bs, ok := fromEvent[[]Blobber](event.Data)
		if !ok {
			return ErrInvalidEventData
		}

		return edb.updateBlobberChallenges(*bs)

	case TagUpdateAllocationChallenge:
		as, ok := fromEvent[[]Allocation](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.updateAllocationChallenges(*as)
	case TagAddOrOverwriteAllocationBlobberTerm:
		updates, ok := fromEvent[[]AllocationBlobberTerm](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.addOrOverwriteAllocationBlobberTerms(*updates)
	case TagUpdateAllocationBlobberTerm:
		updates, ok := fromEvent[[]AllocationBlobberTerm](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.updateAllocationBlobberTerms(*updates)
	case TagDeleteAllocationBlobberTerm:
		updates, ok := fromEvent[[]AllocationBlobberTerm](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.deleteAllocationBlobberTerms(*updates)
	case TagUpdateAllocationStat:
		stats, ok := fromEvent[[]Allocation](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.updateAllocationsStats(*stats)
	case TagUpdateBlobberStat:
		stats, ok := fromEvent[[]Blobber](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.updateBlobbersStats(*stats)
	case TagAddOrUpdateChallengePool:
		// challenge pool
		cps, ok := fromEvent[[]ChallengePool](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.addOrUpdateChallengePools(*cps)
	case TagCollectProviderReward:
		return edb.collectRewards(event.Index)
	case TagMinerHealthCheck:
		healthCheckUpdates, ok := fromEvent[[]dbs.DbHealthCheck](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.updateProvidersHealthCheck(*healthCheckUpdates, MinerTable)
	case TagSharderHealthCheck:
		healthCheckUpdates, ok := fromEvent[[]dbs.DbHealthCheck](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.updateProvidersHealthCheck(*healthCheckUpdates, SharderTable)
	case TagBlobberHealthCheck:
		healthCheckUpdates, ok := fromEvent[[]dbs.DbHealthCheck](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.updateProvidersHealthCheck(*healthCheckUpdates, BlobberTable)
	case TagAuthorizerHealthCheck:
		healthCheckUpdates, ok := fromEvent[[]dbs.DbHealthCheck](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.updateProvidersHealthCheck(*healthCheckUpdates, AuthorizerTable)
	case TagValidatorHealthCheck:
		healthCheckUpdates, ok := fromEvent[[]dbs.DbHealthCheck](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.updateProvidersHealthCheck(*healthCheckUpdates, ValidatorTable)
	case TagAuthorizerBurn:
		b, ok := fromEvent[[]state.Burn](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		logging.Logger.Debug("TagAuthorizerBurn", zap.Any("burns", b))
		return edb.updateAuthorizersTotalBurn(*b)
	case TagAddBurnTicket:
		bt, ok := fromEvent[[]BurnTicket](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		if len(*bt) == 0 {
			return ErrInvalidEventData
		}
		return edb.addBurnTicket((*bt)[0])
	case TagAddBridgeMint:
		// challenge pool
		bms, ok := fromEvent[[]BridgeMint](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		users := make([]User, 0, len(*bms))
		authMint := make(map[string]currency.Coin)
		for _, bm := range *bms {
			users = append(users, User{
				UserID:    bm.UserID,
				MintNonce: bm.MintNonce,
			})

			for _, sig := range bm.Signers {
				mv, ok := authMint[sig]
				if !ok {
					mv = 0
				}
				authMint[sig] = mv + bm.Amount
			}
		}

		mints := make([]state.Mint, 0, len(authMint))
		for auth, amount := range authMint {
			mints = append(mints, state.Mint{
				Minter: auth,
				Amount: amount,
			})
		}

		err := edb.updateUserMintNonce(users)
		if err != nil {
			return err
		}

		err = edb.updateAuthorizersTotalMint(mints)
		if err != nil {
			return err
		}
		return nil

	case TagShutdownProvider:
		u, ok := fromEvent[[]dbs.ProviderID](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.providersSetBoolean(*u, "is_shutdown", true)
	case TagKillProvider:
		u, ok := fromEvent[[]dbs.ProviderID](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.providersSetBoolean(*u, "is_killed", true)
	default:
		return nil
	}
}
