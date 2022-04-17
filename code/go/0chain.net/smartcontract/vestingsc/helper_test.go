package vestingsc

import (
	"encoding/json"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

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
	"0chain.net/core/viper"
)

func init() {
	rand.Seed(time.Now().UnixNano())
	chain.ServerChain = new(chain.Chain)
	data := &chain.ConfigData{ClientSignatureScheme: "bls0chain"}
	chain.ServerChain.Config = chain.NewConfigImpl(data)

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
	if err := scheme.GenerateKeys(); err != nil {
		panic(err)
	}

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

func (c *Client) add(t *testing.T, vsc *VestingSmartContract,
	ar *addRequest, value state.Balance, now common.Timestamp,
	balances chainstate.StateContextI) (resp string, err error) {

	var tx = newTransaction(c.id, ADDRESS, value, now)
	balances.(*testBalances).txn = tx
	return vsc.add(tx, mustEncode(t, ar), balances)
}
