package vestingsc

import (
	"encoding/json"
	"math/rand"
	"strings"
	"testing"
	"time"

	"0chain.net/chaincore/currency"
	"0chain.net/chaincore/threshold/bls"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	chainstate "0chain.net/chaincore/chain/state"
	configpkg "0chain.net/chaincore/config"
	"0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/viper"
	"github.com/0chain/common/core/logging"
)

func init() {
	rand.Seed(time.Now().UnixNano())
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
	balance currency.Coin              // user or blobber
}

func newClient(balance currency.Coin, balances chainstate.StateContextI) (
	client *Client) {

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

func mustEncode(t *testing.T, val interface{}) (b []byte) {
	var err error
	b, err = json.Marshal(val)
	require.NoError(t, err)
	return
}

func newTransaction(f, t datastore.Key, val currency.Coin,
	now common.Timestamp) (tx *transaction.Transaction) {

	tx = new(transaction.Transaction)
	tx.Hash = randString(32)
	tx.ClientID = f
	tx.ToClientID = t
	tx.Value = val
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
	ar *addRequest, value currency.Coin, now common.Timestamp,
	balances chainstate.StateContextI) (resp string, err error) {

	var tx = newTransaction(c.id, ADDRESS, value, now)
	balances.(*testBalances).txn = tx
	return vsc.add(tx, mustEncode(t, ar), balances)
}
