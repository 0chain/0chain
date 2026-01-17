package minersc

import (
	"strconv"
	"testing"

	"github.com/0chain/common/core/currency"

	"0chain.net/smartcontract/stakepool"
	"0chain.net/smartcontract/stakepool/spenum"

	"0chain.net/chaincore/block"
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/smartcontract/dbs/event"
	"github.com/0chain/common/core/util"
	"github.com/stretchr/testify/require"
)

type mockStateContext struct {
	cstate.StateContext
	block                      *block.Block
	store                      map[datastore.Key]util.MPTSerializable
	events                     []event.Event
	LastestFinalizedMagicBlock *block.Block
}

func (sc *mockStateContext) SetMagicBlock(_ *block.MagicBlock)                     {}
func (sc *mockStateContext) GetState() util.MerklePatriciaTrieI                    { return nil }
func (sc *mockStateContext) GetTransaction() *transaction.Transaction              { return nil }
func (sc *mockStateContext) GetSignedTransfers() []*state.SignedTransfer           { return nil }
func (sc *mockStateContext) Validate() error                                       { return nil }
func (sc *mockStateContext) GetSignatureScheme() encryption.SignatureScheme        { return nil }
func (sc *mockStateContext) AddSignedTransfer(_ *state.SignedTransfer)             {}
func (sc *mockStateContext) DeleteTrieNode(_ datastore.Key) (datastore.Key, error) { return "", nil }
func (sc *mockStateContext) GetClientBalance(_ datastore.Key) (currency.Coin, error) {
	return 0, nil
}
func (sc *mockStateContext) GetChainCurrentMagicBlock() *block.MagicBlock { return nil }
func (sc *mockStateContext) EmitEvent(eventType event.EventType, tag event.EventTag, index string, data interface{}, appender ...cstate.Appender) {
	sc.events = append(sc.events, event.Event{
		BlockNumber: sc.block.Round,
		Type:        eventType,
		Tag:         tag,
		Index:       index,
		Data:        data,
	})
}
func (sc *mockStateContext) EmitError(error)                       {}
func (sc *mockStateContext) GetEvents() []event.Event              { return nil }
func (sc *mockStateContext) GetEventDB() *event.EventDb            { return nil }
func (sc *mockStateContext) GetLatestFinalizedBlock() *block.Block { return nil }
func (sc *mockStateContext) GetLastestFinalizedMagicBlock() *block.Block {
	return sc.LastestFinalizedMagicBlock
}

func (sc *mockStateContext) GetBlock() *block.Block {
	return sc.block
}

func (sc *mockStateContext) SetStateContext(_ *state.State) error { return nil }

func (sc *mockStateContext) GetTrieNode(key datastore.Key, v util.MPTSerializable) error {
	vv, ok := sc.store[key]
	if !ok {
		return util.ErrValueNotPresent
	}
	d, err := vv.MarshalMsg(nil)
	if err != nil {
		return err
	}

	_, err = v.UnmarshalMsg(d)
	return err
}

func (sc *mockStateContext) InsertTrieNode(key datastore.Key, node util.MPTSerializable) (datastore.Key, error) {
	sc.store[key] = node
	return key, nil
}

func zcnToBalance(token float64) currency.Coin {
	return currency.Coin(token * float64(x10))
}

func populateDelegates(t *testing.T, cNodes []*MinerNode, minerDelegates []float64, sharderDelegates [][]float64) {
	var delegates [][]float64
	delegates = append(delegates, minerDelegates)
	delegates = append(delegates, sharderDelegates...)
	require.True(t, len(cNodes) <= len(delegates))
	var count = 0
	for i, node := range cNodes {
		node.Pools = make(map[string]*stakepool.DelegatePool)
		var (
			staked currency.Coin
			err    error
		)
		for j, delegate := range delegates[i] {
			count++
			var dp stakepool.DelegatePool
			dp.Balance = zcnToBalance(delegate)
			dp.DelegateID = delegateId + " " + strconv.Itoa(i*maxDelegates+j)
			dp.Status = spenum.Active
			node.Pools[strconv.Itoa(j)] = &dp
			staked, err = currency.AddCoin(staked, zcnToBalance(delegate))
			require.NoError(t, err)
		}
		node.TotalStaked = staked
	}
}

func confirmResults(t *testing.T, global GlobalNode, runtime runtimeValues, f formulae, mn *MinerNode, _ cstate.StateContextI) {
	var epochChangeRound = runtime.blockRound%scYaml.epoch == 0

	if epochChangeRound {
		require.InEpsilon(t, global.MustBase().RewardRate, scYaml.rewardRate*(1.0-scYaml.rewardDeclineRate), errEpsilon)
	} else {
		require.EqualValues(t, global.MustBase().RewardRate, scYaml.rewardRate)
	}

	require.InEpsilon(t, float64(f.minerReward(EtBoth)), float64(mn.Reward), errEpsilon)

}

type EarningsType int

const (
	EtFees EarningsType = iota
	EtBlockReward
	EtBoth
)

// Calculates important 0chain values defined from config
// logs and cli input parameters.
// sc = sc.yaml
// lockFlags input to ./zwallet lock
type formulae struct {
	zChain           mock0ChainYaml
	sc               mockScYaml
	runtime          runtimeValues
	minerDelegates   []float64
	sharderDelegates [][]float64
}

func (f formulae) tokensEarned(et EarningsType) int64 {
	var totalFees int64 = 0
	for _, fee := range f.runtime.fees {
		totalFees += int64(fee)
	}
	var blockReward = f.sc.blockReward * f.sc.rewardRate
	switch et {
	case EtFees:
		return totalFees
	case EtBlockReward:
		return int64(zcnToBalance(blockReward))
	case EtBoth:
		return totalFees + int64(zcnToBalance(blockReward))
	default:
		panic("Invalid earnings type")
	}
}

func (f formulae) minerRevenue(et EarningsType) int64 {
	var totalEarned = float64(f.tokensEarned(et))

	return int64(totalEarned * f.sc.shareRatio)
}

func (f formulae) sharderRevenue(t *testing.T, et EarningsType) int64 {
	var totalEarned = float64(f.tokensEarned(et))
	var ratio = 1 - f.sc.shareRatio
	require.True(t, len(f.sharderDelegates) > 0)
	var numberOfSharders = len(f.sharderDelegates)

	return int64(totalEarned * ratio / float64(numberOfSharders))
}

// miner gets any extra reward from rounding errors after paying delegates
func (f formulae) minerReward(et EarningsType) int64 {
	var minerRevenue = float64(f.minerRevenue(et))
	var areDelegates = len(f.minerDelegates) > 0
	var serviceCharge = f.zChain.ServiceCharge

	if areDelegates {
		return int64(minerRevenue * serviceCharge)
	} else {
		return int64(minerRevenue)
	}
}

// sharders get any extra reward from rounding errors after paying delegates
func (f formulae) sharderReward(t *testing.T, et EarningsType, sharderId int) int64 {
	var sharderRevenue = float64(f.sharderRevenue(t, et))
	var areDelegates = len(f.sharderDelegates[sharderId]) > 0
	var serviceCharge = f.zChain.ServiceCharge

	if areDelegates {
		return int64(sharderRevenue * serviceCharge)
	} else {
		return int64(sharderRevenue)
	}
}

func (f formulae) minerDelegateReward(t *testing.T, et EarningsType, delegateId int) int64 {
	require.True(t, len(f.minerDelegates) > 0)
	var total = 0.0
	for i := 0; i < len(f.minerDelegates); i++ {
		total += float64(zcnToBalance(float64(f.minerDelegates[i])))
	}
	require.True(t, total > 0.0)
	var ratio = float64(zcnToBalance(f.minerDelegates[delegateId])) / total
	var minerRevenue = float64(f.minerRevenue(et))
	var minerReward = float64(f.minerReward(et))

	return int64((minerRevenue - minerReward) * ratio)
}

func (f formulae) sharderDelegateReward(t *testing.T, et EarningsType, delegateId, sharderId int) int64 {
	require.True(t, len(f.sharderDelegates) > sharderId)
	require.True(t, len(f.sharderDelegates[sharderId]) >= delegateId)
	var total = 0.0
	for i := 0; i < len(f.sharderDelegates[sharderId]); i++ {
		total += float64(zcnToBalance(f.sharderDelegates[sharderId][i]))
	}
	require.True(t, total > 0.0)
	var ratio = float64(zcnToBalance(f.sharderDelegates[sharderId][delegateId])) / total
	var sharderRevenue = float64(f.sharderRevenue(t, et))
	var sharderReward = float64(f.sharderReward(t, et, sharderId))

	return int64((sharderRevenue - sharderReward) * ratio)
}
