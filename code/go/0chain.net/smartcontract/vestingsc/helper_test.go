package vestingsc

import (
	"encoding/json"
	"math/rand"
	"strconv"
	"strings"
	"testing"
	"time"

	"0chain.net/chaincore/chain"
	chainstate "0chain.net/chaincore/chain/state"
	configpkg "0chain.net/chaincore/config"
	"0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/logging"
	"0chain.net/core/util"

	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/stretchr/testify/require"
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

func mustSave(t *testing.T, key datastore.Key, val util.Serializable,
	balances chainstate.StateContextI) {

	var _, err = balances.InsertTrieNode(key, val)
	require.NoError(t, err)
}

func avgConfig() (conf *config) {
	conf = new(config)
	conf.MinLock = 1
	conf.MinDuration = 1 * time.Second
	conf.MaxDuration = 1 * time.Hour
	conf.MinFriquency = 1 * time.Second
	conf.MaxFriquency = 1 * time.Hour
	conf.MaxDestinations = 2
	conf.MaxDescriptionLength = 20
	return
}

func setConfig(t *testing.T, balances chainstate.StateContextI) (conf *config) {
	conf = avgConfig()
	mustSave(t, configKey(ADDRESS), conf, balances)
	return
}

func (c *Client) trigger(t *testing.T, vsc *VestingSmartContract,
	poolID datastore.Key, now common.Timestamp,
	balances chainstate.StateContextI) (resp string, err error) {

	var (
		tx = newTransaction(c.id, ADDRESS, 0, now)
		tr lockRequest
	)
	balances.(*testBalances).txn = tx
	tr.PoolID = poolID
	return vsc.trigger(tx, mustEncode(t, &tr), balances)
}

func (c *Client) lock(t *testing.T, vsc *VestingSmartContract,
	poolID datastore.Key, value state.Balance, now common.Timestamp,
	balances chainstate.StateContextI) (resp string, err error) {

	var (
		tx = newTransaction(c.id, ADDRESS, value, now)
		lr lockRequest
	)
	balances.(*testBalances).txn = tx
	lr.PoolID = poolID
	return vsc.lock(tx, mustEncode(t, &lr), balances)
}

func (c *Client) unlock(t *testing.T, vsc *VestingSmartContract,
	poolID datastore.Key, now common.Timestamp,
	balances chainstate.StateContextI) (resp string, err error) {

	var (
		tx = newTransaction(c.id, ADDRESS, 0, now)
		ur lockRequest
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
		dr lockRequest
	)
	balances.(*testBalances).txn = tx
	dr.PoolID = poolID
	return vsc.unlock(tx, mustEncode(t, &dr), balances)
}

func (c *Client) updateConfig(t *testing.T, vsc *VestingSmartContract,
	conf *config, now common.Timestamp, balances chainstate.StateContextI) (
	resp string, err error) {

	var tx = newTransaction(c.id, ADDRESS, 0, now)
	balances.(*testBalances).txn = tx
	return vsc.add(tx, mustEncode(t, conf), balances)
}
