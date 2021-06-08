package faucetsc

import (
	"encoding/json"
	"math/rand"
	"strings"
	"testing"
	"time"

	configpkg "0chain.net/chaincore/config"
	"0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/viper"

	"github.com/stretchr/testify/require"
)

func init() {
	rand.Seed(time.Now().UnixNano())
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

func newTransaction(f, t datastore.Key, val state.Balance, now common.Timestamp) (tx *transaction.Transaction) {
	tx = new(transaction.Transaction)
	tx.Hash = randString(32)
	tx.ClientID = string(f)
	tx.ToClientID = string(t)
	tx.Value = int64(val)
	tx.CreationDate = now
	return
}

func newTestFaucetSC() (fc *FaucetSmartContract) {
	fc = new(FaucetSmartContract)
	fc.SmartContract = new(smartcontractinterface.SmartContract)
	fc.ID = ADDRESS
	return
}
