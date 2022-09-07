package storagesc

import (
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"sync"

	"0chain.net/chaincore/currency"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/core/common"
	"0chain.net/core/encryption"
	"0chain.net/core/maths"
	"0chain.net/smartcontract/dbs"
	"0chain.net/smartcontract/dbs/event"
	"0chain.net/smartcontract/stakepool/spenum"
	"github.com/0chain/common/core/logging"
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

	activePassedBlobberRewardPart, err := getActivePassedBlobberRewardsPartitions(balances, conf.BlockReward.TriggerPeriod)
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

	var blobberRewards []BlobberRewardNode
	if err := activePassedBlobberRewardPart.GetRandomItems(balances, r, &blobberRewards); err != nil {
		logging.Logger.Info("blobber_block_rewards_failed",
			zap.String("getting random partition", err.Error()))
		return nil
	}

	type spResp struct {
		index int
		sp    *stakePool
	}

	var wg sync.WaitGroup
	errorChan := make(chan error, len(blobberRewards))
	spChan := make(chan spResp, len(blobberRewards))
	for i, br := range blobberRewards {
		wg.Add(1)
		go func(b BlobberRewardNode, i int) {
			defer wg.Done()
			if sp, err := ssc.getStakePool(b.ID, balances); err != nil {
				errorChan <- err
			} else {
				spChan <- spResp{
					index: i,
					sp:    sp,
				}
			}
		}(br, i)
	}
	wg.Wait()
	close(errorChan)
	close(spChan)

	for err := range errorChan {
		if err != nil {
			return err
		}
	}

	stakePools := make([]*stakePool, len(blobberRewards))
	for resp := range spChan {
		stakePools[resp.index] = resp.sp
	}

	qualifyingBlobberIds := make([]string, len(blobberRewards))

	for i, br := range blobberRewards {
		sp := stakePools[i]

		staked, err := sp.stake()
		if err != nil {
			return err
		}

		stake := float64(staked)

		gamma := maths.GetGamma(
			conf.BlockReward.Gamma.A,
			conf.BlockReward.Gamma.B,
			conf.BlockReward.Gamma.Alpha,
			br.TotalData,
			br.DataRead,
		)
		zeta := maths.GetZeta(
			conf.BlockReward.Zeta.I,
			conf.BlockReward.Zeta.K,
			conf.BlockReward.Zeta.Mu,
			float64(br.WritePrice),
			float64(br.ReadPrice),
		)
		qualifyingBlobberIds[i] = br.ID
		totalQStake += stake
		blobberWeight := ((gamma * zeta) + 1) * stake * float64(br.SuccessChallenges)
		weight = append(weight, blobberWeight)
		totalWeight += blobberWeight
	}

	if totalWeight == 0 {
		totalWeight = 1
	}

	rewardBal := bbr
	for i, qsp := range stakePools {
		if rewardBal == 0 {
			break
		}
		weightRatio := weight[i] / totalWeight
		if weightRatio > 0 && weightRatio <= 1 {
			fBBR, err := bbr.Float64()
			if err != nil {
				return err
			}
			reward, err := currency.Float64ToCoin(fBBR * weightRatio)
			if err != nil {
				return err
			}
			if reward > rewardBal {
				reward = rewardBal
				rewardBal = 0
			} else {
				rewardBal -= reward
			}
			logging.Logger.Info("blobber_block_rewards_pass",
				zap.Uint64("reward", uint64(reward)),
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

	if rewardBal > 0 {
		rShare, rl, err := currency.DistributeCoin(rewardBal, int64(len(stakePools)))
		if err != nil {
			return err
		}

		if rShare > 0 {
			for i := range stakePools {
				if err := stakePools[i].DistributeRewards(rShare, qualifyingBlobberIds[i], spenum.Blobber, balances); err != nil {
					return common.NewError("blobber_block_rewards_failed", "minting capacity reward"+err.Error())
				}
			}
		}

		if rl > 0 {
			for i := 0; i < int(rl); i++ {
				if err := stakePools[i].DistributeRewards(1, qualifyingBlobberIds[i], spenum.Blobber, balances); err != nil {
					return common.NewError("blobber_block_rewards_failed", "minting capacity reward"+err.Error())
				}
			}
		}

	}

	for i, qsp := range stakePools {
		if err = qsp.save(ssc.ID, qualifyingBlobberIds[i], balances); err != nil {
			return common.NewError("blobber_block_rewards_failed",
				"saving stake pool: "+err.Error())
		}
		staked, err := qsp.stake()
		if err != nil {
			return common.NewError("blobber_block_rewards_failed",
				"getting stake pool stake: "+err.Error())
		}

		data := dbs.DbUpdates{
			Id: qualifyingBlobberIds[i],
			Updates: map[string]interface{}{
				"total_stake": int64(staked),
			},
		}
		balances.EmitEvent(event.TypeStats, event.TagUpdateBlobber, qualifyingBlobberIds[i], data)

	}

	return nil
}

func getBlockReward(
	br currency.Coin,
	currentRound,
	brChangePeriod int64,
	brChangeRatio,
	blobberWeight float64) (currency.Coin, error) {
	if brChangeRatio <= 0 || brChangeRatio >= 1 {
		return 0, fmt.Errorf("unexpected block reward change ratio: %f", brChangeRatio)
	}
	changeBalance := 1 - brChangeRatio
	changePeriods := currentRound / brChangePeriod

	factor := math.Pow(changeBalance, float64(changePeriods)) * blobberWeight
	return currency.MultFloat64(br, factor)
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
