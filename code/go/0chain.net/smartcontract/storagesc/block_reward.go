package storagesc

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"sync"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/core/common"
	"0chain.net/core/encryption"
	"0chain.net/core/logging"
	"0chain.net/core/maths"
	"0chain.net/smartcontract/dbs"
	"0chain.net/smartcontract/dbs/event"
	"0chain.net/smartcontract/partitions"
	"0chain.net/smartcontract/stakepool/spenum"
	"go.uber.org/zap"
)

func (ssc *StorageSmartContract) blobberBlockRewards(
	balances cstate.StateContextI,
) (err error) {
	logging.Logger.Info("blobberBlockRewards started",
		zap.Int64("round", balances.GetBlock().Round),
		zap.String("block_hash", balances.GetBlock().Hash))

	var (
		totalQStake float64
		weight      []float64
		totalWeight float64
	)

	// TODO: move all the maths constants with right name once finalized to the sc.yaml
	const (
		// constants for gamma
		alpha = 1
		A     = 1
		B     = 1

		// constants for zeta
		I  = 1
		K  = 1
		mu = 1
	)

	conf, err := ssc.getConfig(balances, true)
	if err != nil {
		return common.NewError("blobber_block_rewards_failed",
			"cannot get smart contract configurations: "+err.Error())
	}

	if conf.BlockReward.BlockReward == 0 {
		return nil
	}

	bbr, err := getBlockReward(conf.BlockReward.BlockReward, balances.GetBlock().Round,
		conf.BlockReward.BlockRewardChangePeriod, conf.BlockReward.BlockRewardChangeRatio,
		conf.BlockReward.BlobberWeight)
	if err != nil {
		return common.NewError("blobber_block_rewards_failed",
			"cannot get block rewards: "+err.Error())
	}

	allBlobbers, err := getActivePassedBlobbersList(balances, conf.BlockReward.TriggerPeriod)
	if err != nil {
		return common.NewError("blobber_block_rewards_failed",
			"cannot get all blobbers list: "+err.Error())
	}

	hashString := encryption.Hash(balances.GetTransaction().Hash + balances.GetBlock().PrevHash)
	var randomSeed int64
	randomSeed, err = strconv.ParseInt(hashString[0:15], 16, 64)
	if err != nil {
		return common.NewError("blobber_block_rewards_failed",
			"error in creating seed"+err.Error())
	}
	r := rand.New(rand.NewSource(randomSeed))

	blobberPartition, err := allBlobbers.GetRandomSlice(r, balances)
	if err != nil {
		logging.Logger.Info("blobber_block_rewards_failed",
			zap.String("getting random partition", err.Error()))
		return nil
	}

	type spResp struct {
		index int
		sp    *stakePool
	}

	var wg sync.WaitGroup
	errorChan := make(chan error, len(blobberPartition))
	spChan := make(chan spResp, len(blobberPartition))
	for i, b := range blobberPartition {
		wg.Add(1)
		go func(b partitions.PartitionItem, i int) {
			defer wg.Done()
			if sp, err := ssc.getStakePool(b.Name(), balances); err != nil {
				errorChan <- err
			} else {
				spChan <- spResp{
					index: i,
					sp:    sp,
				}
			}
		}(b, i)
	}
	wg.Wait()
	close(errorChan)
	close(spChan)

	for err := range errorChan {
		if err != nil {
			return err
		}
	}

	stakePools := make([]*stakePool, len(blobberPartition))
	for resp := range spChan {
		stakePools[resp.index] = resp.sp
	}

	qualifyingBlobberIds := make([]string, len(blobberPartition))

	for i, b := range blobberPartition {
		var blobber partitions.BlobberRewardNode

		err = blobber.Decode(b.Encode())
		if err != nil {
			return common.NewError("blobber_block_rewards_failed",
				"can't decode blobber reward node: "+err.Error())
		}
		sp := stakePools[i]

		stake := float64(sp.stake())

		gamma := maths.GetGamma(A, B, alpha, blobber.TotalData, blobber.DataRead)
		zeta := maths.GetZeta(I, K, mu, float64(blobber.WritePrice), float64(blobber.ReadPrice))
		qualifyingBlobberIds[i] = blobber.ID
		totalQStake += stake
		blobberWeight := (gamma*zeta*float64(blobber.SuccessChallenges) + 1) * stake
		weight = append(weight, blobberWeight)
		totalWeight += blobberWeight
	}

	if totalWeight == 0 {
		totalWeight = 1
	}

	for i, qsp := range stakePools {
		weightRatio := weight[i] / totalWeight
		if weightRatio > 0 && weightRatio <= 1 {
			reward := bbr * weightRatio
			logging.Logger.Info("blobber_block_rewards_pass",
				zap.Float64("reward", reward),
				zap.String("blobber id", qualifyingBlobberIds[i]),
				zap.Int64("round", balances.GetBlock().Round),
				zap.String("block_hash", balances.GetBlock().Hash))

			if err := qsp.DistributeRewards(reward, qualifyingBlobberIds[i], spenum.Blobber, balances); err != nil {
				return common.NewError("blobber_block_rewards_failed", "minting capacity reward"+err.Error())
			}

		} else {
			logging.Logger.Error("blobber_bloc_rewards - error in weight ratio",
				zap.Any("stake pool", qsp))
			return common.NewError("blobber_block_rewards_failed", "weight ratio out of bound")
		}
	}

	for i, qsp := range stakePools {
		if err = qsp.save(ssc.ID, qualifyingBlobberIds[i], balances); err != nil {
			return common.NewError("blobber_block_rewards_failed",
				"saving stake pool: "+err.Error())
		}
		data, _ := json.Marshal(dbs.DbUpdates{
			Id: qualifyingBlobberIds[i],
			Updates: map[string]interface{}{
				"total_stake": int64(qsp.stake()),
			},
		})
		balances.EmitEvent(event.TypeStats, event.TagUpdateBlobber, qualifyingBlobberIds[i], string(data))

	}

	return nil
}

func getBlockReward(
	br state.Balance,
	currentRound,
	brChangePeriod int64,
	brChangeRatio,
	blobberWeight float64) (float64, error) {
	if brChangeRatio <= 0 || brChangeRatio >= 1 {
		return 0, fmt.Errorf("unexpected block reward change ratio: %f", brChangeRatio)
	}
	changeBalance := 1 - brChangeRatio
	changePeriods := currentRound % brChangePeriod
	return float64(br) * math.Pow(changeBalance, float64(changePeriods)) * blobberWeight, nil
}

func GetCurrentRewardRound(currentRound, period int64) int64 {
	if period > 0 {
		extra := currentRound % period
		return currentRound - extra
	}
	return 0
}

func GetPreviousRewardRound(currentRound, period int64) int64 {
	crr := GetCurrentRewardRound(currentRound, period)
	if crr >= period {
		return crr - period
	} else {
		return 0
	}
}
