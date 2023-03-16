package storagesc

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"math"
	"math/rand"
	"strconv"
	"sync"

	"github.com/0chain/common/core/currency"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/core/common"
	"0chain.net/core/encryption"
	"0chain.net/core/maths"
	"0chain.net/smartcontract/dbs/event"
	"0chain.net/smartcontract/stakepool/spenum"
	"github.com/0chain/common/core/logging"
	"github.com/0chain/common/core/util"
	"go.uber.org/zap"
)

func (ssc *StorageSmartContract) blobberBlockRewards(
	balances cstate.StateContextI,
) (err error) {
	// generate random unique id for logging
	uniqueIdForLogging := uuid.New().String()

	logging.Logger.Info("blobberBlockRewards started"+uniqueIdForLogging,
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

	// convert bbr to string and log it
	bbrString, err := json.Marshal(bbr)
	logging.Logger.Debug("bbrString"+uniqueIdForLogging, zap.String("bbrString", string(bbrString)))

	activePassedBlobberRewardPart, err := getActivePassedBlobberRewardsPartitions(balances, conf.BlockReward.TriggerPeriod)
	if err != nil {
		return common.NewError("blobber_block_rewards_failed",
			"cannot get all blobbers list: "+err.Error())
	}

	// convert activePassedBlobberRewardPart to string and log it
	activePassedBlobberRewardPartString, err := json.Marshal(activePassedBlobberRewardPart)
	logging.Logger.Debug("activePassedBlobberRewardPartString"+uniqueIdForLogging, zap.String("activePassedBlobberRewardPartString", string(activePassedBlobberRewardPartString)))

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
		logging.Logger.Info("blobber_block_rewards_failed"+uniqueIdForLogging,
			zap.String("getting random partition", err.Error()))
		if err != util.ErrValueNotPresent {
			return err
		}
		return nil
	}

	// read all data of blobberRewards and log it
	blobberRewardsString, _ := json.Marshal(blobberRewards)
	logging.Logger.Info("jayashA blobberRewards"+uniqueIdForLogging, zap.String("blobberRewards", string(blobberRewardsString)))

	type spResp struct {
		index int
		sp    *stakePool
	}

	var wg sync.WaitGroup
	errC := make(chan error, len(blobberRewards))
	spC := make(chan spResp, len(blobberRewards))
	for i, br := range blobberRewards {
		wg.Add(1)
		go func(b BlobberRewardNode, i int) {
			defer wg.Done()
			if sp, err := ssc.getStakePool(spenum.Blobber, b.ID, balances); err != nil {
				errC <- err
			} else {
				spC <- spResp{
					index: i,
					sp:    sp,
				}
			}
		}(br, i)
	}
	wg.Wait()
	close(spC)

	select {
	case err := <-errC:
		return err
	default:
	}

	stakePools := make([]*stakePool, len(blobberRewards))

	before := make([]currency.Coin, len(blobberRewards))
	for resp := range spC {
		stakePools[resp.index] = resp.sp
		stake, err := resp.sp.stake()
		if err != nil {
			return err
		}
		before[resp.index] = stake
	}

	// read all data of stakePools and log it
	stakePoolsString, _ := json.Marshal(stakePools)
	logging.Logger.Info("jayash stakePools"+uniqueIdForLogging, zap.String("stakePools", string(stakePoolsString)))

	qualifyingBlobberIds := make([]string, len(blobberRewards))

	// read all data of blobberRewards and log it
	blobberRewardsString, _ = json.Marshal(blobberRewards)
	logging.Logger.Info("jayashB blobberRewards"+uniqueIdForLogging, zap.String("blobberRewards", string(blobberRewardsString)))

	for i, br := range blobberRewards {
		sp := stakePools[i]

		staked, err := sp.stake()

		logging.Logger.Debug("jayashB staked"+uniqueIdForLogging, zap.Any("staked", staked))

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

		// log the values of gamma
		logging.Logger.Info("jayashB gamma"+uniqueIdForLogging, zap.Float64("gamma", gamma))

		zeta := maths.GetZeta(
			conf.BlockReward.Zeta.I,
			conf.BlockReward.Zeta.K,
			conf.BlockReward.Zeta.Mu,
			float64(br.WritePrice),
			float64(br.ReadPrice),
		)

		// log the values of zeta
		logging.Logger.Info("jayashB zeta"+uniqueIdForLogging, zap.Float64("zeta", zeta))

		qualifyingBlobberIds[i] = br.ID
		totalQStake += stake
		blobberWeight := ((gamma * zeta) + 1) * stake * float64(br.SuccessChallenges)
		weight = append(weight, blobberWeight)
		totalWeight += blobberWeight

		// log the values of blobberWeight

		logging.Logger.Info("jayashB blobberWeight"+uniqueIdForLogging, zap.Float64("blobberWeight", blobberWeight))
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
			logging.Logger.Info("blobber_block_rewards_pass"+uniqueIdForLogging,
				zap.Uint64("reward", uint64(reward)),
				zap.String("blobber id", qualifyingBlobberIds[i]),
				zap.Int64("round", balances.GetBlock().Round),
				zap.String("block_hash", balances.GetBlock().Hash))

			if err := qsp.DistributeRewards(
				reward, qualifyingBlobberIds[i], spenum.Blobber, spenum.BlockRewardBlobber, balances); err != nil {
				return common.NewError("blobber_block_rewards_failed", "minting capacity reward"+err.Error())
			}

		} else {
			logging.Logger.Error("blobber_bloc_rewards - error in weight ratio"+uniqueIdForLogging,
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
				if err := stakePools[i].DistributeRewards(rShare, qualifyingBlobberIds[i], spenum.Blobber, spenum.BlockRewardBlobber, balances); err != nil {
					return common.NewError("blobber_block_rewards_failed"+uniqueIdForLogging, "minting capacity reward"+err.Error())
				}
			}
		}

		if rl > 0 {
			for i := 0; i < int(rl); i++ {
				if err := stakePools[i].DistributeRewards(1, qualifyingBlobberIds[i], spenum.Blobber, spenum.BlockRewardBlobber, balances); err != nil {
					return common.NewError("blobber_block_rewards_failed"+uniqueIdForLogging, "minting capacity reward"+err.Error())
				}
			}
		}

	}

	for i, qsp := range stakePools {
		if err = qsp.Save(spenum.Blobber, qualifyingBlobberIds[i], balances); err != nil {
			return common.NewError("blobber_block_rewards_failed"+uniqueIdForLogging,
				"saving stake pool: "+err.Error())
		}
		staked, err := qsp.stake()
		if err != nil {
			return common.NewError("blobber_block_rewards_failed"+uniqueIdForLogging,
				"getting stake pool stake: "+err.Error())
		}

		bid := qualifyingBlobberIds[i]
		tag, data := event.NewUpdateBlobberTotalStakeEvent(bid, staked)
		balances.EmitEvent(event.TypeStats, tag, bid, data)
		if blobberRewards[i].WritePrice > 0 {
			stake, err := qsp.stake()
			if err != nil {
				return err
			}
			balances.EmitEvent(event.TypeStats, event.TagAllocBlobberValueChange, qualifyingBlobberIds[i], event.AllocationBlobberValueChanged{
				FieldType:    event.Staked,
				AllocationId: "",
				BlobberId:    qualifyingBlobberIds[i],
				Delta:        int64((stake - before[i]) / blobberRewards[i].WritePrice),
			})
		}
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
