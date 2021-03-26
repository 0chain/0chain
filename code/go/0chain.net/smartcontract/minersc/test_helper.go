package minersc

import (
	"encoding/json"
	"math/rand"
	"strconv"
	"strings"
	"testing"
	"time"

	// "0chain.net/chaincore/chain"
	"0chain.net/chaincore/block"
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/util"

	"github.com/rcrowley/go-metrics"

	"0chain.net/core/logging"
	"go.uber.org/zap"

	"github.com/stretchr/testify/require"
)

// test helpers

const x10 = 10 * 1000 * 1000 * 1000

func toks(val state.Balance) string {
	return strconv.FormatFloat(float64(val)/float64(x10), 'f', -1, 64)
}

func init() {
	rand.Seed(time.Now().UnixNano())
	// chain.ServerChain = new(chain.Chain)
	// chain.ServerChain.Config = new(chain.Config)
	// chain.ServerChain.ClientSignatureScheme = "bls0chain"

	// node.Self.Node = node.Provider() // stub
	logging.Logger = zap.NewNop() // /dev/null

	moveFunctions[Start] = moveTrue
	moveFunctions[Contribute] = moveTrue
	moveFunctions[Share] = moveTrue
	moveFunctions[Publish] = moveTrue
	moveFunctions[Wait] = moveTrue
}

func moveTrue(balances cstate.StateContextI, pn *PhaseNode, gn *GlobalNode) (
	result bool) {

	return true
}

func randString(n int) string {
	const hexLetters = "abcdef0123456789"

	var sb strings.Builder
	for i := 0; i < n; i++ {
		sb.WriteByte(hexLetters[rand.Intn(len(hexLetters))])
	}
	return sb.String()
}

// Client represents test client. A BC user with his wallet and balance.
type Client struct {
	id      string                     // identifier
	pk      string                     // public key
	scheme  encryption.SignatureScheme // pk/sk
	balance state.Balance              // client wallet balance

	keep state.Balance // keep latest know balance (manual control)
}

func newClient(balance state.Balance, balances cstate.StateContextI) (
	client *Client) {

	var scheme = encryption.NewBLS0ChainScheme()
	scheme.GenerateKeys()

	client = new(Client)
	client.balance = balance
	client.scheme = scheme

	client.pk = scheme.GetPublicKey()
	client.id = encryption.Hash(client.pk)

	balances.(*testBalances).balances[client.id] = balance
	return
}

func newClientWithDelegate(isMiner bool, t *testing.T, msc *MinerSmartContract, now int64,
	balances cstate.StateContextI) (client, delegate *Client) {

	client, delegate = newClient(0, balances), newClient(0, balances)

	var err error
	_, err = client.callAddMinerOrSharder(isMiner, t, msc, now, delegate.id, balances)

	require.NoError(t, err, "add_client")
	return
}

type TestClient struct {
	client   *Client
	delegate *Client
	stakers  []*Client
}

func createLFMB(miners []*TestClient, sharders []*TestClient) (
	b *block.Block) {

	b = new(block.Block)

	b.MagicBlock = block.NewMagicBlock()
	b.MagicBlock.Miners = node.NewPool(node.NodeTypeMiner)
	b.MagicBlock.Sharders = node.NewPool(node.NodeTypeSharder)

	for _, miner := range miners {
		b.MagicBlock.Miners.NodesMap[miner.client.id] = new(node.Node)
	}
	for _, sharder := range sharders {
		b.MagicBlock.Sharders.NodesMap[sharder.client.id] = new(node.Node)
	}
	return
}

// create and add miner/sharder, create stake holders, don't stake
func newClientWithStakers(isMiner bool, t *testing.T, msc *MinerSmartContract,
	now, stakersAmount int64, stakeValue state.Balance,
	balances cstate.StateContextI) (
		client *TestClient) {

	client = new(TestClient)
	client.client, client.delegate = newClientWithDelegate(isMiner, t, msc, now, balances)
	for i := int64(0); i < stakersAmount; i++ {
		client.stakers = append(client.stakers, newClient(stakeValue, balances))
	}
	return
}

// stake a miner or a sharder
func (c *Client) callAddToDelegatePool(t *testing.T, msc *MinerSmartContract,
	now, value int64, nodeId string, balances cstate.StateContextI) (
		resp string, err error) {

	t.Helper()

	var tx = newTransaction(c.id, ADDRESS, value, now)
	balances.(*testBalances).txn = tx

	var pool delegatePool
	pool.ConsensusNodeID = nodeId

	var (
		input  = mustEncode(t, &pool)
		global *GlobalNode
	)
	global, err = msc.getGlobalNode(balances)
	require.NoError(t, err, "missing global node")
	return msc.addToDelegatePool(tx, input, global, balances)
}

func (c *Client) callAddMinerOrSharder(isMiner bool, t *testing.T,
	msc *MinerSmartContract, now int64, delegateWallet string,
	balances cstate.StateContextI) (
		resp string, err error) {

	var tx = newTransaction(c.id, ADDRESS, 0, now)
	balances.(*testBalances).txn = tx
	var (
		input  = c.addNodeRequest(t, delegateWallet)
		global *GlobalNode
	)
	global, err = msc.getGlobalNode(balances)
	require.NoError(t, err, "missing global node")

	if isMiner {
		return msc.AddMiner(tx, input, global, balances)
	} else {
		return msc.AddSharder(tx, input, global, balances)
	}
}

// add_miner or add_sharder transaction data
func (c *Client) addNodeRequest(t *testing.T, delegateWallet string) []byte {
	var node = NewConsensusNode()
	node.ID = c.id
	node.N2NHost = "http://" + c.id + ":9081/api/v1"
	node.Host = c.id + ".host.miners"
	node.Port = 9081
	node.PublicKey = c.pk
	node.ShortName = "test_miner(" + c.id + ")"
	node.BuildTag = "commit"
	node.DelegateWallet = delegateWallet
	node.ServiceCharge = 0.5
	node.NumberOfDelegates = 100
	node.MinStake = 1
	node.MaxStake = 100e10

	return mustEncode(t, node)
}

func newTransaction(f, t string, val, now int64) (tx *transaction.Transaction) {
	tx = new(transaction.Transaction)
	tx.Hash = randString(32)
	tx.ClientID = f
	tx.ToClientID = t
	tx.Value = val
	tx.CreationDate = common.Timestamp(now)
	return
}

func mustEncode(t *testing.T, val interface{}) []byte {
	var err error
	b, err := json.Marshal(val)
	require.NoError(t, err)
	return b
}

func mustSave(t *testing.T, key datastore.Key, val util.Serializable,
	balances cstate.StateContextI) {

	var _, err = balances.InsertTrieNode(key, val)
	require.NoError(t, err)
}

func setMagicBlock(t *testing.T, miners []*Client, sharders []*Client,
	balances cstate.StateContextI) {

	var mb = block.NewMagicBlock()
	mb.Miners = node.NewPool(node.NodeTypeMiner)
	mb.Sharders = node.NewPool(node.NodeTypeSharder)

	for _, miner := range miners {
		var n = node.Provider()
		n.SetID(miner.id)

		n.Type = node.NodeTypeMiner
		mb.Miners.AddNode(n)
	}
	for _, sharder := range sharders {
		var n = node.Provider()
		n.SetID(sharder.id)

		n.Type = node.NodeTypeSharder
		mb.Sharders.AddNode(n)
	}

	var err error
	_, err = balances.InsertTrieNode(MagicBlockKey, mb)
	require.NoError(t, err, "setting magic block")
}

func newTestMinerSC() (msc *MinerSmartContract) {
	msc = new(MinerSmartContract)
	msc.SmartContract = new(smartcontractinterface.SmartContract)
	msc.ID = ADDRESS
	msc.SmartContractExecutionStats = make(map[string]interface{})
	msc.SmartContractExecutionStats["mintedTokens"] =
		metrics.GetOrRegisterCounter("mintedTokens", nil)
	return
}
