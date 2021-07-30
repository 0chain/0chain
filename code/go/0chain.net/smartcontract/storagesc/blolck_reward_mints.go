package storagesc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/core/common"
	"0chain.net/core/util"
	"encoding/json"
	"fmt"
)

type blockRewardMints struct {
	MintedRewards  float64 `json:"minted_rewards"`
	MaxMintRewards float64 `json:"max_mint_reward"`
	// the max mint check is made before adding to UnMintedBlockRewards map
	UnProcessedMints map[string]float64 `json:"un_minted_block_rewards"`
}

func newBlockRewardsMints() *blockRewardMints {
	return &blockRewardMints{
		UnProcessedMints: make(map[string]float64),
	}
}

func (mi *blockRewardMints) Encode() []byte {
	var b, err = json.Marshal(mi)
	if err != nil {
		panic(err) // must never happens
	}
	return b
}

func (mi *blockRewardMints) Decode(p []byte) error {
	return json.Unmarshal(p, mi)
}

func (mi *blockRewardMints) mintRewardsForBlobber(
	sp *stakePool,
	blobberId string,
	balances cstate.StateContextI,
) error {
	unMinted, ok := mi.UnProcessedMints[blobberId]
	if ok && unMinted > 0 {
		minted, err := mintReward(sp, unMinted, balances)
		if err != nil {
			return fmt.Errorf("error miniting block rewards: %v", err)
		}
		mi.UnProcessedMints[blobberId] = unMinted - float64(minted)
	}
	return nil
}

func (mi *blockRewardMints) addMint(blobberId string, amount float64, config *scConfig) error {
	if mi.MintedRewards+amount > mi.MaxMintRewards {
		return fmt.Errorf("minted rewards exceed max allowed: %f", mi.MaxMintRewards)
	}
	mi.UnProcessedMints[blobberId] += amount
	mi.MintedRewards += amount
	config.Minted += state.Balance(amount)
	return nil
}

func (mi *blockRewardMints) populate(
	ssc *StorageSmartContract,
	balances cstate.StateContextI,
) error {
	conf, err := ssc.getConfig(balances, true)
	if err != nil {
		return common.NewErrorf("allocation_creation_failed",
			"can't get config: %v", err)
	}
	mi.MaxMintRewards = float64(conf.BlockReward.MaxMintRewards)

	return nil
}

func (mi *blockRewardMints) save(balances cstate.StateContextI) error {
	_, err := balances.InsertTrieNode(BLOCK_REWARD_MINTS, mi)
	return err
}

func getBlockRewardMintsBytes(balances cstate.StateContextI) ([]byte, error) {
	var val util.Serializable
	val, err := balances.GetTrieNode(BLOCK_REWARD_MINTS)
	if err != nil {
		return nil, err
	}
	return val.Encode(), nil
}

func getBlockRewardMints(
	ssc *StorageSmartContract,
	balances cstate.StateContextI,
) (*blockRewardMints, error) {
	var bsBytes []byte
	var err error
	mi := newBlockRewardsMints()
	if bsBytes, err = getBlockRewardMintsBytes(balances); err != nil {
		if err != util.ErrValueNotPresent {
			return nil, err
		}
		if err := mi.populate(ssc, balances); err != nil {
			return nil, err
		}
		return mi, nil
	}
	err = mi.Decode(bsBytes)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", common.ErrDecoding, err)
	}
	return mi, nil
}
