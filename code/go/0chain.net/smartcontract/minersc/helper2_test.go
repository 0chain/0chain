package minersc

import (
	"0chain.net/chaincore/block"
	cstate "0chain.net/chaincore/chain/state"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/tokenpool"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/util"
	"fmt"
	"github.com/stretchr/testify/require"
	"math"
	"strconv"
	"strings"
	"testing"
)

type mockStateContext struct {
	ctx                        cstate.StateContext
	block                      *block.Block
	store                      map[datastore.Key]util.Serializable
	sharders                   []string
	LastestFinalizedMagicBlock *block.Block
}

func (sc *mockStateContext) SetMagicBlock(_ *block.MagicBlock)                       { return }
func (sc *mockStateContext) GetState() util.MerklePatriciaTrieI                      { return nil }
func (sc *mockStateContext) GetTransaction() *transaction.Transaction                { return nil }
func (sc *mockStateContext) GetSignedTransfers() []*state.SignedTransfer             { return nil }
func (sc *mockStateContext) Validate() error                                         { return nil }
func (sc *mockStateContext) GetSignatureScheme() encryption.SignatureScheme          { return nil }
func (sc *mockStateContext) AddSignedTransfer(_ *state.SignedTransfer)               { return }
func (sc *mockStateContext) DeleteTrieNode(_ datastore.Key) (datastore.Key, error)   { return "", nil }
func (sc *mockStateContext) GetClientBalance(_ datastore.Key) (state.Balance, error) { return 0, nil }

func (sc *mockStateContext) GetTransfers() []*state.Transfer {
	return sc.ctx.GetTransfers()
}

func (sc *mockStateContext) GetMints() []*state.Mint {
	return sc.ctx.GetMints()
}

func (sc *mockStateContext) GetLastestFinalizedMagicBlock() *block.Block {
	return sc.LastestFinalizedMagicBlock
}

func (sc *mockStateContext) GetBlockSharders(_ *block.Block) []string {
	return sc.sharders
}

func (sc *mockStateContext) GetBlock() *block.Block {
	return sc.block
}

func (sc *mockStateContext) SetStateContext(_ *state.State) error { return nil }

func (sc *mockStateContext) GetTrieNode(key datastore.Key) (util.Serializable, error) {
	return sc.store[key], nil
}

func (sc *mockStateContext) InsertTrieNode(key datastore.Key, node util.Serializable) (datastore.Key, error) {
	sc.store[key] = node
	return key, nil
}

func (sc *mockStateContext) AddTransfer(t *state.Transfer) error {
	return sc.ctx.AddTransfer(t)
}

func (sc *mockStateContext) AddMint(m *state.Mint) error {

	return sc.ctx.AddMint(m)
}

func zcnToBalance(token float64) state.Balance {
	return state.Balance(token * float64(x10))
}

func populateDelegates(t *testing.T, cNodes []*MinerNode, minerDelegates []float64, sharderDelegates [][]float64) {
	var delegates = [][]float64{}
	delegates = append(delegates, minerDelegates)
	delegates = append(delegates, sharderDelegates...)
	require.True(t, len(cNodes) <= len(delegates))
	var count = 0
	for i, node := range cNodes {
		node.Active = make(map[string]*sci.DelegatePool)
		var staked int64 = 0
		for j, delegate := range delegates[i] {
			count++
			node.Active[strconv.Itoa(j)] = &sci.DelegatePool{
				PoolStats: &sci.PoolStats{
					DelegateID: datastore.Key(delegateId + " " + strconv.Itoa(i*maxDelegates+j)),
				},
				ZcnLockingPool: &tokenpool.ZcnLockingPool{
					ZcnPool: tokenpool.ZcnPool{
						TokenPool: tokenpool.TokenPool{
							ID:      strconv.Itoa(i*maxDelegates + j),
							Balance: zcnToBalance(delegate),
						},
					},
				},
			}
			staked += int64(zcnToBalance(delegate))
		}
		node.TotalStaked = staked
	}
}

func confirmResults(t *testing.T, global GlobalNode, runtime runtimeValues, f formulae, ctx cstate.StateContextI) {

	var viewChangeRound = runtime.blockRound%scYaml.rewardRoundPeriod == 0
	var epochChangeRound = runtime.blockRound%scYaml.epoch == 0

	if epochChangeRound {
		require.InEpsilon(t, global.RewardRate, scYaml.rewardRate*(1.0-scYaml.rewardDeclineRate), errEpsilon)
	} else {
		require.InEpsilon(t, global.InterestRate, scYaml.interestRate*(1.0-scYaml.interestDeclineRAte), errEpsilon)
	}

	if viewChangeRound {
		require.InEpsilon(t, int64(global.Minted),
			int64(runtime.minted)+f.tokensEarned(EtBlockReward)+f.totalInterest(), errEpsilon)
	} else {
		require.InEpsilon(t, int64(global.Minted), int64(runtime.minted)+f.tokensEarned(EtBlockReward), errEpsilon)
	}

	var minerFees, minerBr bool
	var sharderFees = make([]bool, len(f.sharderDelegates))
	var sharderBr = make([]bool, len(f.sharderDelegates))

	var minerDelegateFees = make([]bool, len(f.minerDelegates))
	var minerDelegateBr = make([]bool, len(f.minerDelegates))
	var minerDelegateInt = make([]bool, len(f.minerDelegates))

	sharderDelegatesFees := make([][]bool, len(f.sharderDelegates))
	sharderDelegatesBr := make([][]bool, len(f.sharderDelegates))
	sharderDelegatesInt := make([][]bool, len(f.sharderDelegates))
	for i := range f.sharderDelegates {
		sharderDelegatesFees[i] = make([]bool, len(f.sharderDelegates[i]))
		sharderDelegatesBr[i] = make([]bool, len(f.sharderDelegates[i]))
		sharderDelegatesInt[i] = make([]bool, len(f.sharderDelegates[i]))
	}

	for _, mint := range ctx.GetMints() {
		require.EqualValues(t, minerScId, mint.Minter)
		var wallet = strings.Split(mint.ToClientID, " ")
		switch wallet[0] {
		case sharderId:
			{
				index, err := strconv.Atoi(wallet[1])
				require.NoError(t, err)
				require.True(t, index/maxDelegates < len(f.sharderDelegates))
				require.InDelta(t, f.sharderReward(t, EtBlockReward, index), int64(mint.Amount), errDelta)
				sharderBr[index] = true
				break
			}
		case minerId:
			require.InDelta(t, f.minerReward(EtBlockReward), int64(mint.Amount), errDelta)
			minerBr = true
			break
		case delegateId:
			{
				index, err := strconv.Atoi(wallet[1])
				require.NoError(t, err)
				var node = index / maxDelegates
				var delegate = index % maxDelegates
				require.True(t, node < len(f.sharderDelegates)+1)
				if node == 0 {
					var blockReward = f.minerDelegateReward(t, EtBlockReward, delegate)
					require.True(t, delegate < len(f.minerDelegates))
					if viewChangeRound {
						var interest = f.minerDelegateInterest(delegate)
						if errDelta > math.Abs(float64(interest)-float64(mint.Amount)) {
							require.False(t, minerDelegateInt[delegate])
							minerDelegateInt[delegate] = true
						} else {
							require.False(t, minerDelegateBr[delegate])
							require.InDelta(t, blockReward, int64(mint.Amount), errDelta)
							minerDelegateBr[delegate] = true
						}
					} else {
						require.False(t, minerDelegateBr[delegate])
						require.InDelta(t, blockReward, int64(mint.Amount), errDelta)
						minerDelegateBr[delegate] = true
					}
				} else {
					node--
					var blockReward = f.sharderDelegateReward(t, EtBlockReward, delegate, node)
					if viewChangeRound {
						var interest = f.sharderDelegateInterest(delegate, node)
						if errDelta > math.Abs(float64(interest)-float64(mint.Amount)) {
							require.False(t, sharderDelegatesInt[node][delegate])
							sharderDelegatesInt[node][delegate] = true
						} else {
							require.False(t, sharderDelegatesBr[node][delegate])
							require.InDelta(t, blockReward, int64(mint.Amount), errDelta)
							sharderDelegatesBr[node][delegate] = true
						}
					} else {
						require.False(t, sharderDelegatesBr[node][delegate])
						require.True(t, delegate < len(f.sharderDelegates[node]))
						require.InDelta(t, blockReward, int64(mint.Amount), errDelta)
						sharderDelegatesBr[node][delegate] = true
					}
				}
			}
		default:
			panic(fmt.Sprintf("unknown wallet type %s", wallet[0]))
		}
	}

	for _, transfer := range ctx.GetTransfers() {
		require.EqualValues(t, minerScId, transfer.ClientID)
		var wallet = strings.Split(transfer.ToClientID, " ")
		switch wallet[0] {
		case sharderId:
			{
				index, err := strconv.Atoi(wallet[1])
				require.NoError(t, err)
				require.True(t, index/maxDelegates < len(f.sharderDelegates))
				require.False(t, sharderFees[index])
				require.InDelta(t, f.sharderReward(t, EtFees, index), int64(transfer.Amount), errDelta)
				sharderFees[index] = true
				break
			}
		case minerId:
			require.InDelta(t, f.minerReward(EtFees), int64(transfer.Amount), errDelta)
			minerFees = true
			break
		case delegateId:
			{
				index, err := strconv.Atoi(wallet[1])
				require.NoError(t, err)
				var node = index / maxDelegates
				var delegate = index % maxDelegates
				require.True(t, node < len(f.sharderDelegates)+1)
				if node == 0 {
					require.False(t, minerDelegateFees[delegate])
					require.True(t, delegate < len(f.minerDelegates))
					require.InDelta(t, f.minerDelegateReward(t, EtFees, delegate), int64(transfer.Amount), errDelta)
					minerDelegateFees[delegate] = true
				} else {
					node--
					require.False(t, sharderDelegatesFees[node][delegate])
					require.True(t, delegate < len(f.sharderDelegates[node]))
					require.InDelta(t, f.sharderDelegateReward(t, EtFees, delegate, node), int64(transfer.Amount), errDelta)
					sharderDelegatesFees[node][delegate] = true
				}
			}
		default:
			panic(fmt.Sprintf("unknown wallet type %s", wallet[0]))
		}
	}

	// These tests might be too strong as if te delegate reward is zero due to
	// the relative share being too small then there will not be a matching mint or transfer for it
	require.True(t, minerFees)
	require.True(t, minerBr)
	for i := range minerDelegateFees {
		require.True(t, minerDelegateFees[i])
		require.True(t, minerDelegateBr[i])
		if viewChangeRound {
			require.True(t, minerDelegateInt[i])
		}
	}
	for i := range sharderFees {
		require.True(t, sharderFees[i])
		require.True(t, sharderBr[i])
	}
	for i := range sharderDelegatesFees {
		for j := range sharderDelegatesFees[i] {
			require.True(t, sharderDelegatesFees[i][j])
			require.True(t, sharderDelegatesBr[i][j])
			if viewChangeRound {
				require.True(t, sharderDelegatesInt[i][j])
			}
		}
	}
}
