package minersc

import (
	"context"
	"encoding/json"
	"math/rand"
	"net/url"
	"strings"
	"testing"
	"time"

	"0chain.net/chaincore/threshold/bls"
	"github.com/0chain/common/core/currency"

	"0chain.net/chaincore/block"
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"github.com/0chain/common/core/util"

	"github.com/rcrowley/go-metrics"

	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"

	"github.com/stretchr/testify/require"
)

// test helpers

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
	result error) {

	return nil
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
	balance currency.Coin              // client wallet balance
}

func newClient(balance currency.Coin, balances cstate.StateContextI) (client *Client) {

	var scheme = encryption.NewBLS0ChainScheme()
	if err := scheme.GenerateKeys(); err != nil {
		panic(err)
	}

	client = new(Client)
	client.balance = balance
	client.scheme = scheme

	client.pk = scheme.GetPublicKey()
	pub := bls.PublicKey{}
	pub.DeserializeHexStr(client.pk)
	client.id = encryption.Hash(pub.Serialize())

	balances.(*testBalances).balances[client.id] = balance
	return
}

const minerServiceCharge = 0.5

// add_miner or add_sharder transaction data
func (c *Client) addNodeRequest(delegateWallet string) *MinerNode {
	var mn = NewMinerNode()
	mn.ID = c.id
	mn.N2NHost = "http://" + c.id + ":9081/api/v1"
	mn.Host = c.id + ".host.miners"
	mn.Port = 9081
	mn.PublicKey = c.pk
	mn.ShortName = "test_miner(" + c.id + ")"
	mn.BuildTag = "commit"
	mn.Settings.DelegateWallet = delegateWallet
	mn.Settings.ServiceChargeRatio = minerServiceCharge
	mn.Settings.MaxNumDelegates = 10
	return mn
}

func newTransaction(f, t string, val currency.Coin, now int64) (tx *transaction.Transaction) {
	tx = new(transaction.Transaction)
	tx.Hash = randString(32)
	tx.ClientID = f
	tx.ToClientID = t
	tx.Value = val
	tx.CreationDate = common.Timestamp(now)
	return
}

func (c *Client) callAddMiner(msc *MinerSmartContract,
	now int64, mn *MinerNode, balances cstate.StateContextI) (
	resp string, err error) {

	var tx = newTransaction(c.id, ADDRESS, 0, now)
	balances.(*testBalances).txn = tx
	input := mustEncode(mn)
	var gn *GlobalNode
	gn, err = getGlobalNode(balances)
	if err != nil {
		return
	}
	rsp, err := msc.AddMiner(tx, input, gn, balances)
	if err != nil {
		return
	}
	return rsp, nil
}

func (c *Client) callAddSharder(msc *MinerSmartContract,
	now int64, sn *MinerNode, balances cstate.StateContextI) (
	resp string, err error) {

	var tx = newTransaction(c.id, ADDRESS, 0, now)
	balances.(*testBalances).txn = tx
	input := mustEncode(sn)
	var gn *GlobalNode
	gn, err = getGlobalNode(balances)
	if err != nil {
		return
	}

	resp, err = msc.AddSharder(tx, input, gn, balances)
	if err != nil {
		return
	}
	return resp, nil
}

func newMiner(t *testing.T, saveToMB bool, balances cstate.StateContextI) *miner {
	c, delegate := newClient(0, balances), newClient(0, balances)
	if saveToMB {
		nd := node.Node{}
		nd.ID = c.id
		nd.Type = node.NodeTypeMiner
		nd.PublicKey = c.pk
		err := balances.(*testBalances).magicBlock.Miners.AddNode(&nd)
		require.NoError(t, err)
	}

	mn := c.addNodeRequest(delegate.id)

	m := new(miner)
	m.miner = c
	m.delegate = delegate
	m.node = mn
	return m
}

func addMiner(t *testing.T, msc *MinerSmartContract, now int64, saveToMB bool,
	balances cstate.StateContextI) (*miner, error) {

	m := newMiner(t, saveToMB, balances)
	_, err := m.execAddMinerTxn(msc, now, balances)
	if err != nil {
		return nil, err
	}

	return m, nil
}

func addSharder(t *testing.T, msc *MinerSmartContract, now int64, saveToMB bool,
	balances cstate.StateContextI) (s *sharder, err error) {
	s = newSharder(t, saveToMB, balances)
	_, err = s.execAddSharderTxn(msc, now, balances)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func newSharder(t *testing.T, saveToMB bool, balances cstate.StateContextI) *sharder {
	c, delegate := newClient(0, balances), newClient(0, balances)
	if saveToMB {
		nd := node.Node{}
		nd.ID = c.id
		nd.Type = node.NodeTypeSharder
		nd.PublicKey = c.pk
		err := balances.(*testBalances).magicBlock.Sharders.AddNode(&nd)
		require.NoError(t, err)
	}

	sn := c.addNodeRequest(delegate.id)

	s := new(sharder)
	s.sharder = c
	s.delegate = delegate
	s.node = sn
	return s
}

func (c *Client) addToDelegatePoolRequest(t *testing.T, nodeID string) []byte {
	var dp deletePool
	dp.ProviderID = nodeID
	return mustEncode(&dp)
}

// stake a miner or a sharder
func (c *Client) callAddToDelegatePool(t *testing.T, msc *MinerSmartContract,
	now int64, val currency.Coin, nodeID string, balances cstate.StateContextI) (resp string,
	err error) {

	t.Helper()
	var tx = newTransaction(c.id, ADDRESS, val, now)
	balances.(*testBalances).txn = tx
	var (
		input = c.addToDelegatePoolRequest(t, nodeID)
		gn    *GlobalNode
	)
	gn, err = getGlobalNode(balances)
	require.NoError(t, err, "missing global node")
	return msc.addToDelegatePool(tx, input, gn, balances)
}

func mustEncode(val interface{}) []byte {
	var err error
	b, err := json.Marshal(val)
	if err != nil {
		panic(err)
	}
	return b
}

func mustSave(t *testing.T, key datastore.Key, val util.MPTSerializable,
	balances cstate.StateContextI) {

	var _, err = balances.InsertTrieNode(key, val)
	require.NoError(t, err)
}

func setConfig(t *testing.T, balances cstate.StateContextI) *GlobalNode {

	gn := new(globalNodeV2)
	gn.ViewChange = 0
	gn.MaxN = 100
	gn.MinN = 3
	gn.MaxS = 30
	gn.MinS = 1
	gn.MaxDelegates = 10 // for tests
	gn.NumMinerDelegatesRewarded = 10
	gn.NumShardersRewarded = 5
	gn.NumSharderDelegatesRewarded = 1
	gn.TPercent = 0.51 // %
	gn.KPercent = 0.75 // %
	gn.LastRound = 0
	gn.MaxStake = currency.Coin(100.0e10)
	gn.MinStake = currency.Coin(0.01e10)
	gn.RewardRate = 1.0
	gn.ShareRatio = 0.10
	gn.BlockReward = currency.Coin(0.7e10)
	gn.MaxCharge = 0.5 // %
	gn.Epoch = 15e6    // 15M
	gn.RewardDeclineRate = 0.1

	res := new(GlobalNode)
	res.SetEntity(gn)

	mustSave(t, GlobalNodeKey, res, balances)
	return res
}

func setMagicBlock(t *testing.T, miners []*Client, sharders []*Client,
	balances cstate.StateContextI) {

	var mb = block.NewMagicBlock()
	mb.Miners = node.NewPool(node.NodeTypeMiner)
	mb.Sharders = node.NewPool(node.NodeTypeSharder)
	for _, mn := range miners {
		var n = node.Provider()
		err := n.SetID(mn.id)
		require.NoError(t, err)
		n.PublicKey = mn.pk
		n.Type = node.NodeTypeMiner
		n.SetSignatureSchemeType(encryption.SignatureSchemeBls0chain)
		mb.Miners.AddNode(n)
	}
	for _, sh := range sharders {
		var n = node.Provider()
		err := n.SetID(sh.id)
		require.NoError(t, err)

		n.PublicKey = sh.pk
		n.Type = node.NodeTypeSharder
		n.SetSignatureSchemeType(encryption.SignatureSchemeBls0chain)
		mb.Sharders.AddNode(n)
	}

	err := updateMagicBlock(balances, mb)
	require.NoError(t, err, "setting magic block")
}

func setRounds(t *testing.T, _ *MinerSmartContract, last, vc int64,
	balances cstate.StateContextI) {

	var gn, err = getGlobalNode(balances)
	require.NoError(t, err, "getting global node")

	gn.MustUpdateBase(func(base *globalNodeBase) error {
		base.LastRound = last
		base.ViewChange = vc
		return nil
	})

	require.NoError(t, gn.save(balances), "saving global node")

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

func (msc *MinerSmartContract) ConfigHandler(
	ctx context.Context,
	values url.Values,
	balances cstate.StateContextI,
) (interface{}, error) {
	return msc.configHandler(ctx, values, balances)
}

func (msc *MinerSmartContract) configHandler(
	_ context.Context,
	_ url.Values,
	balances cstate.StateContextI,
) (interface{}, error) {
	gn, err := getGlobalNode(balances)
	if err != nil {
		return nil, common.NewErrInternal(err.Error())
	}
	return gn.getConfigMap()
}

func (msc *MinerSmartContract) UpdateSettings(
	t *transaction.Transaction,
	inputData []byte,
	gn *GlobalNode,
	balances cstate.StateContextI,
) (resp string, err error) {
	return msc.updateSettings(t, inputData, gn, balances)
}

func (msc *MinerSmartContract) UpdateGlobals(
	txn *transaction.Transaction,
	inputData []byte,
	gn *GlobalNode,
	balances cstate.StateContextI,
) (resp string, err error) {
	return msc.updateGlobals(txn, inputData, gn, balances)
}
