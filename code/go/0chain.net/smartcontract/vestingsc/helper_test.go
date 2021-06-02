package vestingsc

import (
	"encoding/json"
	"math/rand"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/0chain/0chain/code/go/0chain.net/chaincore/chain"
	chainstate "github.com/0chain/0chain/code/go/0chain.net/chaincore/chain/state"
	configpkg "github.com/0chain/0chain/code/go/0chain.net/chaincore/config"
	"github.com/0chain/0chain/code/go/0chain.net/chaincore/smartcontractinterface"
	"github.com/0chain/0chain/code/go/0chain.net/chaincore/state"
	"github.com/0chain/0chain/code/go/0chain.net/chaincore/transaction"
	"github.com/0chain/0chain/code/go/0chain.net/core/common"
	"github.com/0chain/0chain/code/go/0chain.net/core/datastore"
	"github.com/0chain/0chain/code/go/0chain.net/core/encryption"
	"github.com/0chain/0chain/code/go/0chain.net/core/logging"
	"github.com/0chain/0chain/code/go/0chain.net/core/viper"
)

const x10 = 10 * 1000 * 1000 * 1000

func toks(val state.Balance) string {
	return strconv.FormatFloat(float64(val)/float64(x10), 'f', -1, 64)
}

func init() {
	rand.Seed(time.Now().UnixNano())
	chain.ServerChain = new(chain.Chain)
	chain.ServerChain.Config = new(chain.Config)
	chain.ServerChain.ClientSignatureScheme = "bls0chain"

	logging.Logger = zap.NewNop()

	configpkg.SmartContractConfig = viper.New()
}

func randString(n int) string {
	const hexLetters = "abcdef0123456789"
	var sb strings.Builder
	for i := 0; i < n; i++ {
		sb.WriteByte(hexLetters[rand.Intn(len(hexLetters))])
	}
	return sb.String()
}

type Client struct {
	id      string                     // identifier
	pk      string                     // public key
	scheme  encryption.SignatureScheme // pk/sk
	balance state.Balance              // user or blobber
}

func newClient(balance state.Balance, balances chainstate.StateContextI) (
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

func mustEncode(t *testing.T, val interface{}) (b []byte) {
	var err error
	b, err = json.Marshal(val)
	require.NoError(t, err)
	return
}

func mustDecode(t *testing.T, b []byte, val interface{}) {
	require.NoError(t, json.Unmarshal(b, val))
	return
}

func newTransaction(f, t datastore.Key, val state.Balance,
	now common.Timestamp) (tx *transaction.Transaction) {

	tx = new(transaction.Transaction)
	tx.Hash = randString(32)
	tx.ClientID = string(f)
	tx.ToClientID = string(t)
	tx.Value = int64(val)
	tx.CreationDate = now
	return
}

func newTestVestingSC() (vsc *VestingSmartContract) {
	vsc = new(VestingSmartContract)
	vsc.SmartContract = new(smartcontractinterface.SmartContract)
	vsc.ID = ADDRESS
	return
}

func (c *Client) trigger(t *testing.T, vsc *VestingSmartContract,
	poolID datastore.Key, now common.Timestamp,
	balances chainstate.StateContextI) (resp string, err error) {

	var (
		tx = newTransaction(c.id, ADDRESS, 0, now)
		tr poolRequest
	)
	balances.(*testBalances).txn = tx
	tr.PoolID = poolID
	return vsc.trigger(tx, mustEncode(t, &tr), balances)
}

func (c *Client) stop(t *testing.T, vsc *VestingSmartContract,
	poolID, dest datastore.Key, now common.Timestamp,
	balances chainstate.StateContextI) (resp string, err error) {

	var (
		tx = newTransaction(c.id, ADDRESS, 0, now)
		sr stopRequest
	)
	balances.(*testBalances).txn = tx
	sr.PoolID = poolID
	sr.Destination = dest
	return vsc.stop(tx, mustEncode(t, &sr), balances)
}

func (c *Client) unlock(t *testing.T, vsc *VestingSmartContract,
	poolID datastore.Key, now common.Timestamp,
	balances chainstate.StateContextI) (resp string, err error) {

	var (
		tx = newTransaction(c.id, ADDRESS, 0, now)
		ur poolRequest
	)
	balances.(*testBalances).txn = tx
	ur.PoolID = poolID
	return vsc.unlock(tx, mustEncode(t, &ur), balances)
}

func (c *Client) add(t *testing.T, vsc *VestingSmartContract,
	ar *addRequest, value state.Balance, now common.Timestamp,
	balances chainstate.StateContextI) (resp string, err error) {

	var tx = newTransaction(c.id, ADDRESS, value, now)
	balances.(*testBalances).txn = tx
	return vsc.add(tx, mustEncode(t, ar), balances)
}

func (c *Client) delete(t *testing.T, vsc *VestingSmartContract,
	poolID datastore.Key, now common.Timestamp,
	balances chainstate.StateContextI) (resp string, err error) {

	var (
		tx = newTransaction(c.id, ADDRESS, 0, now)
		dr poolRequest
	)
	balances.(*testBalances).txn = tx
	dr.PoolID = poolID
	return vsc.unlock(tx, mustEncode(t, &dr), balances)
}
