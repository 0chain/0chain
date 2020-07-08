package minersc

import (
	"math/rand"
	"strconv"
	"strings"
	"testing"
	"time"

	// "0chain.net/chaincore/chain"
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/logging"
	"0chain.net/core/util"

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

	logging.Logger = zap.NewNop()
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

// func (c *Client) addBlobRequest(t *testing.T) []byte {
// 	var sn StorageNode
// 	sn.ID = c.id
// 	sn.BaseURL = "http://" + c.id + ":9081/api/v1"
// 	sn.Terms = c.terms
// 	sn.Capacity = c.cap
// 	sn.Used = 0
// 	sn.LastHealthCheck = 0
// 	sn.StakePoolSettings.NumDelegates = 100
// 	sn.StakePoolSettings.MinStake = 0
// 	sn.StakePoolSettings.MaxStake = 1000e10
// 	return mustEncode(t, &sn)
// }

func newTransaction(f, t string, val, now int64) (tx *transaction.Transaction) {
	tx = new(transaction.Transaction)
	tx.Hash = randString(32)
	tx.ClientID = f
	tx.ToClientID = t
	tx.Value = val
	tx.CreationDate = common.Timestamp(now)
	return
}

// func (c *Client) callAddBlobber(t *testing.T, msc *MinerSmartContract,
// 	now int64, balances cstate.StateContextI) (resp string, err error) {
//
// 	var tx = newTransaction(c.id, ADDRESS,
// 		int64(float64(c.terms.WritePrice)*sizeInGB(c.cap)), now)
// 	balances.(*testBalances).txn = tx
// 	var input = c.addBlobRequest(t)
// 	return msc.addBlobber(tx, input, balances)
// }

// // addBlobber to SC
// func addBlobber(t *testing.T, msc *MinerSmartContract, cap, now int64,
// 	terms Terms, balacne state.Balance, balances cstate.StateContextI) (
// 	blob *Client) {
//
// 	var scheme = encryption.NewBLS0ChainScheme()
// 	scheme.GenerateKeys()
//
// 	blob = new(Client)
// 	blob.terms = terms
// 	blob.cap = cap
// 	blob.balance = balacne
// 	blob.scheme = scheme
//
// 	blob.pk = scheme.GetPublicKey()
// 	blob.id = encryption.Hash(blob.pk)
//
// 	balances.(*testBalances).balances[blob.id] = balacne
//
// 	var _, err = blob.callAddBlobber(t, ssc, now, balances)
// 	require.NoError(t, err)
//
// 	// add stake for the blobber as blobber owner
// 	var tx = newTransaction(blob.id, ADDRESS,
// 		int64(float64(terms.WritePrice)*sizeInGB(cap)), now)
// 	balances.(*testBalances).txn = tx
// 	_, err = ssc.stakePoolLock(tx, blob.stakeLockRequest(t), balances)
// 	require.NoError(t, err)
// 	return
// }

func mustSave(t *testing.T, key datastore.Key, val util.Serializable,
	balances cstate.StateContextI) {

	var _, err = balances.InsertTrieNode(key, val)
	require.NoError(t, err)
}

func setConfig(t *testing.T, balances cstate.StateContextI) (
	gn *globalNode) {

	gn = new(globalNode)
	gn.ViewChange = 0
	gn.MaxN = 100
	gn.MinN = 3
	gn.MaxS = 30
	gn.MinS = 1
	gn.MaxDelegates = 10 // for tests
	gn.TPercent = 0.51   // %
	gn.KPercent = 0.75   // %
	gn.LastRound = 0
	gn.MaxStake = state.Balance(100.0e10)
	gn.MinStake = state.Balance(0.01e10)
	gn.InterestRate = 0.1
	gn.RewardRate = 1.0
	gn.ShareRatio = 0.10
	gn.BlockReward = state.Balance(0.7e10)
	gn.MaxCharge = 0.5 // %
	gn.Epoch = 15e6    // 15M
	gn.RewardDeclineRate = 0.1
	gn.InterestDeclineRate = 0.1
	gn.MaxMint = state.Balance(4e6 * 1e10)
	gn.Minted = 0

	mustSave(t, GlobalNodeKey, gn, balances)
	return
}

func newTestMinerSC() (msc *MinerSmartContract) {
	msc = new(MinerSmartContract)
	msc.SmartContract = new(smartcontractinterface.SmartContract)
	msc.ID = ADDRESS
	return
}
