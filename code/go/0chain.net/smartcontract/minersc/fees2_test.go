package minersc

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"0chain.net/core/config"
	"0chain.net/core/config/mocks"
	"0chain.net/smartcontract/provider"
	"0chain.net/smartcontract/stakepool/spenum"

	"github.com/0chain/common/core/currency"

	"0chain.net/smartcontract/stakepool"

	"0chain.net/chaincore/block"
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/client"
	"0chain.net/chaincore/node"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"github.com/0chain/common/core/util"
	"github.com/rcrowley/go-metrics"
	"github.com/stretchr/testify/require"
)

const (
	blockHash           = datastore.Key("myHash")
	signatureSchemeType = encryption.SignatureSchemeEd25519
	minerPk             = datastore.Key("25206bf74fb1afa8045acd269ef76890d8a1e34d89eb681c042ac58dbc080e30")
	selfId              = datastore.Key("mySelfId")
	delegateId          = "delegate"
	maxDelegates        = 1000
	errEpsilon          = 0.1
	errPayFee           = "pay_fee"
	errJumpedBackInTime = "jumped back in time"
)

var sharderPKs = []datastore.Key{
	"76b77d4efdaa2320244a86d864a5fbd35eecd1bb21dd062f083187ed6b9e14a1",
	"a9fa275bca2d8ee06d3f5de6f0a1900d5e4ee13f48e08cec17273031fc154cbc",
	"c7ecdf1a8d16717d2cf345845bf6c5effb26430087c8c7dbf9be7eddd6e9db63",
	"a107c2a8ee0f60806ba53f77a94641c6a0782054e371c7a86dd8272ccae12566",
	"53af72f0fcdc1c7f1beb6091d487835c723351846e9aaf4a72a788edd10b72d1",
}

type mockScYaml struct {
	startRound        int64
	rewardRate        float64
	blockReward       float64
	epoch             int64
	rewardDeclineRate float64
	shareRatio        float64
	maxMint           float64
	rewardRoundPeriod int64
}

type mock0ChainYaml struct {
	viewChange    bool
	ServiceCharge float64
}

type runtimeValues struct {
	lastRound      int64
	blockRound     int64
	runRound       int64
	phase          Phase
	phaseRound     int64
	nextViewChange int64
	minted         currency.Coin
	fees           []currency.Coin
}

type MinerDelegates []float64
type SharderDelegates [][]float64

var (
	approvedMinters = []string{
		"6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9", // miner SC
		"cf8d0df9bd8cc637a4ff4e792ffe3686da6220c45f0e1103baa609f3f1751ef4", // interest SC
		"6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7", // storage SC
	}
	minerScId = approvedMinters[0]

	scYaml = mockScYaml{
		startRound:        50,
		rewardRate:        1.0,
		blockReward:       0.21,
		epoch:             15000000,
		rewardDeclineRate: 0.1,
		shareRatio:        0.8,
		maxMint:           4000000.0,
		rewardRoundPeriod: 250,
	}
	zChainYaml = mock0ChainYaml{
		viewChange:    false,
		ServiceCharge: 0.10,
	}

	runValues = runtimeValues{
		lastRound:      50,
		blockRound:     53,
		runRound:       53,
		phase:          4,
		phaseRound:     35,
		nextViewChange: 100,
		minted:         0,
		fees:           []currency.Coin{200, 300, 500},
	}
)

func TestMain(m *testing.M) {
	// Initialise global variables
	PhaseRounds = make(map[Phase]int64)
	node.Self = &node.SelfNode{
		Node: &node.Node{
			Client: client.Client{
				IDField: datastore.IDField{
					ID: selfId,
				},
			},
		},
	}

	os.Exit(m.Run())
}

func TestPayFees(t *testing.T) {
	t.Run("one sharder no delegates", func(t *testing.T) {
		//t.Skip()
		var minerStakes = MinerDelegates{}
		var sharderStakes = SharderDelegates{[]float64{}}
		err := testPayFees(t, minerStakes, sharderStakes, runValues)
		require.NoError(t, err)
	})

	t.Run("one sharder one delegate each", func(t *testing.T) {
		//t.Skip()
		var minerStakes = MinerDelegates{0.2}
		var sharderStakes = SharderDelegates{[]float64{0.3}}
		runValues.blockRound = scYaml.rewardRoundPeriod
		runValues.runRound = scYaml.rewardRoundPeriod
		runValues.lastRound = scYaml.rewardRoundPeriod - 2
		zChainYaml.viewChange = true
		err := testPayFees(t, minerStakes, sharderStakes, runValues)
		require.NoError(t, err)
	})

	t.Run("three sharders multiple delegates", func(t *testing.T) {
		//t.Skip()
		var minerStakes = MinerDelegates{0.2, 0.011}
		var sharderStakes = SharderDelegates{
			[]float64{0.3, 0.5},
			[]float64{0.5, 0.15},
			[]float64{0.7, 1.7, 0.1, 0.23}}
		err := testPayFees(t, minerStakes, sharderStakes, runValues)
		require.NoError(t, err)
	})

	t.Run("view change round, one sharder no delegates", func(t *testing.T) {
		//t.Skip()
		var minerStakes = MinerDelegates{}
		var sharderStakes = SharderDelegates{[]float64{}}
		runValues.blockRound = 2 * scYaml.rewardRoundPeriod
		runValues.runRound = 2 * scYaml.rewardRoundPeriod
		runValues.lastRound = 2*scYaml.rewardRoundPeriod - 1
		zChainYaml.viewChange = true
		err := testPayFees(t, minerStakes, sharderStakes, runValues)
		require.NoError(t, err)
	})

	t.Run("reward round, three sharders multiple delegates ", func(t *testing.T) {
		//t.Skip()
		var minerStakes = MinerDelegates{0.2, 0.011}
		var sharderStakes = SharderDelegates{
			[]float64{0.1, 0.5},
			[]float64{0.2, 0.12},
			[]float64{0.6, 1.777, 0.19, 0.1123}}
		runValues.blockRound = 3 * scYaml.rewardRoundPeriod
		runValues.runRound = 3 * scYaml.rewardRoundPeriod
		runValues.lastRound = 3*scYaml.rewardRoundPeriod - 1
		zChainYaml.viewChange = false
		err := testPayFees(t, minerStakes, sharderStakes, runValues)
		require.NoError(t, err)
	})

	t.Run("new epoch, three sharders multiple delegates", func(t *testing.T) {
		//t.Skip()
		var minerStakes = MinerDelegates{0.2, 0.011}
		var sharderStakes = SharderDelegates{
			[]float64{0.1, 0.5},
			[]float64{0.2, 0.12},
			[]float64{0.6, 1.777, 0.19, 0.1123}}
		runValues.blockRound = 3 * scYaml.epoch
		runValues.runRound = 3 * scYaml.epoch
		runValues.lastRound = 3*scYaml.epoch - 1
		zChainYaml.viewChange = true
		err := testPayFees(t, minerStakes, sharderStakes, runValues)
		require.NoError(t, err)
	})

	t.Run("errJumpedBackInTime", func(t *testing.T) {
		var minerStakes = MinerDelegates{}
		var sharderStakes = SharderDelegates{[]float64{}}
		runValues.runRound -= 1
		err := testPayFees(t, minerStakes, sharderStakes, runValues)
		require.Error(t, err)
		require.True(t, strings.Contains(err.Error(), errPayFee))
	})
}

func testPayFees(t *testing.T, minerStakes []float64, sharderStakes [][]float64, runtime runtimeValues) error {
	minersPool := node.NewPool(node.NodeTypeMiner)
	shardersPool := node.NewPool(node.NodeTypeSharder)

	minerNode := node.Provider()
	minerNode.Type = node.NodeTypeMiner
	minerNode.PublicKey = minerPk
	minerNode.SetSignatureSchemeType(signatureSchemeType)
	minersPool.AddNode(minerNode)
	minerID := minerNode.ID

	var numberOfSharders = len(sharderStakes)
	sharderIDs := make([]string, numberOfSharders)
	for i := 0; i < numberOfSharders; i++ {
		sharderNode := node.Provider()
		sharderNode.Type = node.NodeTypeSharder
		sharderNode.PublicKey = sharderPKs[i]
		sharderNode.SetSignatureSchemeType(signatureSchemeType)
		shardersPool.AddNode(sharderNode)
	}

	for i, s := range shardersPool.Nodes {
		sharderIDs[i] = s.ID
	}

	var f = formulae{
		zChain:           zChainYaml,
		sc:               scYaml,
		runtime:          runValues,
		minerDelegates:   minerStakes,
		sharderDelegates: sharderStakes,
	}

	var globalNodeObjV2 = &globalNodeV2{
		globalNodeV1: globalNodeV1{
			LastRound:                   runtime.lastRound,
			RewardRate:                  scYaml.rewardRate,
			BlockReward:                 zcnToBalance(scYaml.blockReward),
			Epoch:                       scYaml.epoch,
			ShareRatio:                  scYaml.shareRatio,
			RewardRoundFrequency:        scYaml.rewardRoundPeriod,
			NumShardersRewarded:         5,
			NumSharderDelegatesRewarded: 1,
			NumMinerDelegatesRewarded:   10,
		},
	}

	var globalNode GlobalNode
	globalNode.SetEntity(globalNodeObjV2)

	var msc = &MinerSmartContract{
		SmartContract: &sci.SmartContract{
			SmartContractExecutionStats: make(map[string]interface{}),
		},
	}
	msc.SmartContractExecutionStats["feesPaid"] = nil
	msc.SmartContractExecutionStats["mintedTokens"] = metrics.NilCounter{}
	var txn = &transaction.Transaction{
		ClientID:   minerID,
		ToClientID: minerScId,
	}

	var ctx = &mockStateContext{
		StateContext: *cstate.NewStateContext(
			nil,
			&util.MerklePatriciaTrie{},
			txn,
			func(round int64) *block.MagicBlock {
				return &block.MagicBlock{
					Sharders: shardersPool,
				}
			},
			nil,
			nil,
			nil,
			nil,
			nil, nil, nil,
		),
		block: &block.Block{
			UnverifiedBlockBody: block.UnverifiedBlockBody{
				MinerID: minerID,
				Round:   runtime.blockRound,
				Txns:    []*transaction.Transaction{},
			},
			HashIDField: datastore.HashIDField{
				Hash: blockHash,
			},
			PrevBlock: &block.Block{},
		},
		store: make(map[datastore.Key]util.MPTSerializable),
		LastestFinalizedMagicBlock: &block.Block{
			MagicBlock: &block.MagicBlock{
				Miners:   minersPool,
				Sharders: shardersPool,
			},
		},
	}
	for _, fee := range runtime.fees {
		ctx.block.Txns = append(ctx.block.Txns, &transaction.Transaction{Fee: fee})
	}
	var phaseNode = &PhaseNode{
		Phase:      runtime.phase,
		StartRound: scYaml.startRound,
	}
	PhaseRounds[phaseNode.Phase] = runtime.phaseRound
	_, err := ctx.InsertTrieNode(phaseNode.GetKey(), phaseNode)
	require.NoError(t, err)

	var self = NewMinerNode()
	self.ID = selfId
	self.ProviderType = spenum.Miner
	_, err = ctx.InsertTrieNode(self.GetKey(), self)
	require.NoError(t, err)

	var miner = &MinerNode{
		SimpleNode: &SimpleNode{
			Provider: provider.Provider{
				ID:           minerID,
				ProviderType: spenum.Miner,
			},
			TotalStaked: 100,
		},
		StakePool: stakepool.NewStakePool(),
	}
	miner.Settings.ServiceChargeRatio = zChainYaml.ServiceCharge
	miner.Settings.DelegateWallet = minerID
	miner.StakePool.Settings.ServiceChargeRatio = zChainYaml.ServiceCharge
	var allMiners = &MinerNodes{
		Nodes: []*MinerNode{miner},
	}

	err = updateMinersList(ctx, allMiners)
	require.NoError(t, err)

	var sharders []*MinerNode
	for i := 0; i < numberOfSharders; i++ {
		sharder := &MinerNode{
			SimpleNode: &SimpleNode{
				Provider: provider.Provider{
					ID:           sharderIDs[i],
					ProviderType: spenum.Sharder,
				},
				TotalStaked: 100,
			},
			StakePool: stakepool.NewStakePool(),
		}
		miner.Settings.ServiceChargeRatio = zChainYaml.ServiceCharge
		miner.Settings.DelegateWallet = minerID
		miner.StakePool.Settings.ServiceChargeRatio = zChainYaml.ServiceCharge
		sharders = append(sharders, sharder)
	}

	populateDelegates(t, append([]*MinerNode{miner}, sharders...), minerStakes, sharderStakes)
	_, err = ctx.InsertTrieNode(miner.GetKey(), miner)

	require.NoError(t, err)
	for i := 0; i < numberOfSharders; i++ {
		_, err = ctx.InsertTrieNode(sharders[i].GetKey(), sharders[i])
		require.NoError(t, err)
	}
	var allSharders = &MinerNodes{
		Nodes: sharders,
	}
	err = updateAllShardersList(ctx, allSharders)
	require.NoError(t, err)

	mockChainConfig := mocks.NewChainConfig(t)
	mockChainConfig.On("IsViewChangeEnabled").Return(false)
	// Add information only relevant to view change rounds
	config.Configuration().ChainConfig = mockChainConfig

	globalNode.MustBase().ViewChange = 100
	if runValues.blockRound == runValues.nextViewChange {
		var allMinersList = &MinerNodes{}
		err = updateAllShardersList(ctx, allMinersList)
	}

	inp := &PayFeesInput{Round: runValues.runRound}
	marshal, err := json.Marshal(inp)
	if err != nil {
		return err
	}
	_, err = msc.payFees(txn, marshal, &globalNode, ctx)
	if err != nil {
		return err
	}
	require.NoError(t, err)

	mn, err := getMinerNode(txn.ClientID, ctx)
	require.NoError(t, err)

	confirmResults(t, globalNode, runtime, f, mn, ctx)

	return err
}
