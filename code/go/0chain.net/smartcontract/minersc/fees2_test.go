package minersc

import (
	"0chain.net/chaincore/block"
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/client"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/node"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
	"github.com/rcrowley/go-metrics"
	"github.com/stretchr/testify/require"
	"os"
	"strconv"
	"strings"
	"testing"
)

const (
	blockHash           = datastore.Key("myHash")
	minerId             = datastore.Key("myMiner")
	selfId              = datastore.Key("mySelfId")
	sharderId           = "sharder"
	delegateId          = "delegate"
	maxDelegates        = 1000
	errDelta            = 4 // for testing values with rounding errors
	errEpsilon          = 0.1
	errPayFee           = "pay_fee"
	errJumpedBackInTime = "jumped back in time"
)

type mockScYaml struct {
	startRound          int64
	rewardRate          float64
	blockReward         float64
	epoch               int64
	rewardDeclineRate   float64
	interestDeclineRAte float64
	interestRate        float64
	shareRatio          float64
	maxMint             float64
	rewardRoundPeriod   int64
}

type mock0ChainYaml struct {
	viewChange    bool
	ServiceCharge float64
}

type runtimeValues struct {
	lastRound      int64
	blockRound     int64
	phase          Phase
	phaseRound     int64
	nextViewChange int64
	minted         state.Balance
	fees           []int64
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
		startRound:          50,
		rewardRate:          1.0,
		blockReward:         0.21,
		epoch:               15000000,
		interestRate:        0.000000555, // 0
		rewardDeclineRate:   0.1,
		interestDeclineRAte: 0.1,
		shareRatio:          0.8,
		maxMint:             4000000.0,
		rewardRoundPeriod:   250,
	}
	zChainYaml = mock0ChainYaml{
		viewChange:    false,
		ServiceCharge: 0.10,
	}

	runValues = runtimeValues{
		lastRound:      50,
		blockRound:     53,
		phase:          4,
		phaseRound:     35,
		nextViewChange: 100,
		minted:         0,
		fees:           []int64{200, 300, 500},
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
		runValues.lastRound = 3*scYaml.epoch - 1
		zChainYaml.viewChange = true
		err := testPayFees(t, minerStakes, sharderStakes, runValues)
		require.NoError(t, err)
	})

	t.Run("errJumpedBackInTime", func(t *testing.T) {
		var minerStakes = MinerDelegates{}
		var sharderStakes = SharderDelegates{[]float64{}}
		runValues.lastRound = runValues.blockRound + 1
		err := testPayFees(t, minerStakes, sharderStakes, runValues)
		require.Error(t, err)
		require.True(t, strings.Contains(err.Error(), errPayFee))
		require.True(t, strings.Contains(err.Error(), errJumpedBackInTime))
	})
}

func testPayFees(t *testing.T, minerStakes []float64, sharderStakes [][]float64, runtime runtimeValues) error {
	var numberOfSharders = len(sharderStakes)
	var globalNode = &GlobalNode{
		//ViewChange:           runtime.nextViewChange,
		LastRound:            runtime.lastRound,
		RewardRate:           scYaml.rewardRate,
		BlockReward:          zcnToBalance(scYaml.blockReward),
		Epoch:                scYaml.epoch,
		InterestRate:         scYaml.interestRate,
		ShareRatio:           scYaml.shareRatio,
		MaxMint:              zcnToBalance(scYaml.maxMint),
		Minted:               runtime.minted,
		RewardRoundFrequency: scYaml.rewardRoundPeriod,
	}
	var msc = &MinerSmartContract{
		SmartContract: &sci.SmartContract{
			SmartContractExecutionStats: make(map[string]interface{}),
		},
	}
	msc.SmartContractExecutionStats["feesPaid"] = nil
	msc.SmartContractExecutionStats["mintedTokens"] = metrics.NilCounter{}
	var txn = &transaction.Transaction{
		ClientID:   minerId,
		ToClientID: minerScId,
	}
	var ctx = &mockStateContext{
		ctx: *cstate.NewStateContext(
			nil,
			&util.MerklePatriciaTrie{},
			&state.Deserializer{},
			txn,
			nil,
			nil,
			nil,
		),
		block: &block.Block{
			UnverifiedBlockBody: block.UnverifiedBlockBody{
				MinerID: minerId,
				Round:   runtime.blockRound,
				Txns:    []*transaction.Transaction{},
			},
			HashIDField: datastore.HashIDField{
				Hash: blockHash,
			},
			PrevBlock: &block.Block{},
		},
		sharders: []string{},
		store:    make(map[datastore.Key]util.Serializable),
		LastestFinalizedMagicBlock: &block.Block{
			MagicBlock: &block.MagicBlock{
				Miners: &node.Pool{
					Nodes:    make([]*node.Node, 1),
					NodesMap: make(map[string]*node.Node),
				},
				Sharders: &node.Pool{
					Nodes:    make([]*node.Node, numberOfSharders),
					NodesMap: make(map[string]*node.Node),
				},
			},
		},
	}
	for _, fee := range runtime.fees {
		ctx.block.Txns = append(ctx.block.Txns, &transaction.Transaction{Fee: fee})
	}
	for i := 0; i < numberOfSharders; i++ {
		ctx.sharders = append(ctx.sharders, sharderId+" "+strconv.Itoa(i))
	}
	var phaseNode = &PhaseNode{
		Phase:      runtime.phase,
		StartRound: scYaml.startRound,
	}
	PhaseRounds[phaseNode.Phase] = runtime.phaseRound
	_, err := ctx.InsertTrieNode(phaseNode.GetKey(), phaseNode)
	require.NoError(t, err)

	var self = &MinerNode{
		SimpleNode: &SimpleNode{
			ID: selfId,
		},
	}
	_, err = ctx.InsertTrieNode(self.getKey(), self)
	require.NoError(t, err)

	var miner = &MinerNode{
		SimpleNode: &SimpleNode{
			ID:             minerId,
			TotalStaked:    100,
			ServiceCharge:  zChainYaml.ServiceCharge,
			DelegateWallet: datastore.Key(minerId),
		},
		Active: make(map[string]*sci.DelegatePool),
	}
	var allMiners = &MinerNodes{
		Nodes: []*MinerNode{miner},
	}
	_, err = ctx.InsertTrieNode(AllMinersKey, allMiners)
	require.NoError(t, err)

	var sharders = []*MinerNode{}
	for i := 0; i < numberOfSharders; i++ {
		sharders = append(sharders, &MinerNode{
			SimpleNode: &SimpleNode{
				ID:             datastore.Key(sharderId + " " + strconv.Itoa(i)),
				TotalStaked:    100,
				ServiceCharge:  zChainYaml.ServiceCharge,
				DelegateWallet: datastore.Key(sharderId + " " + strconv.Itoa(i)),
			},
			Active: make(map[string]*sci.DelegatePool),
		})
	}

	populateDelegates(t, append([]*MinerNode{miner}, sharders...), minerStakes, sharderStakes)
	_, err = ctx.InsertTrieNode(miner.getKey(), miner)
	require.NoError(t, err)
	for i := 0; i < numberOfSharders; i++ {
		_, err = ctx.InsertTrieNode(sharders[i].getKey(), sharders[i])
		require.NoError(t, err)
		ctx.LastestFinalizedMagicBlock.Sharders.Nodes = append(ctx.LastestFinalizedMagicBlock.Sharders.Nodes, &node.Node{})
		ctx.LastestFinalizedMagicBlock.Sharders.NodesMap[sharders[i].ID] = &node.Node{}
	}
	var allSharders = &MinerNodes{
		Nodes: sharders,
	}
	_, err = ctx.InsertTrieNode(AllShardersKey, allSharders)
	require.NoError(t, err)

	ctx.LastestFinalizedMagicBlock.Miners.Nodes = []*node.Node{{}}
	ctx.LastestFinalizedMagicBlock.Miners.NodesMap[miner.ID] = &node.Node{}

	// Add information only relevant to view change rounds
	config.DevConfiguration.ViewChange = zChainYaml.viewChange
	globalNode.ViewChange = 100
	if runValues.blockRound == runValues.nextViewChange {
		var allMinersList = NewDKGMinerNodes()
		_, err = ctx.InsertTrieNode(DKGMinersKey, allMinersList)
	}

	_, err = msc.payFees(txn, nil, globalNode, ctx)
	if err != nil {
		return err
	}

	var f = formulae{
		zChain:           zChainYaml,
		sc:               scYaml,
		runtime:          runValues,
		minerDelegates:   minerStakes,
		sharderDelegates: sharderStakes,
	}
	confirmResults(t, *globalNode, runtime, f, ctx)

	return err
}
