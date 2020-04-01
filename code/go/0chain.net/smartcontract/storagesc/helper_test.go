package storagesc

import (
	"math/rand"
	"strings"
	"testing"
	"time"

	chainState "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/util"

	// "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	rand.Seed(time.Now().UnixNano())
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
	id     string                     // identifier
	pk     string                     // public key
	scheme encryption.SignatureScheme // pk/sk

	// blobber
	terms Terms
	cap   int64

	// user or blobber
	balance state.Balance
}

func newClient(balance state.Balance, balances chainState.StateContextI) (
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

func (c *Client) addBlobRequest(t *testing.T) []byte {
	var sn StorageNode
	sn.ID = c.id
	sn.BaseURL = "http://" + c.id + ":9081/api/v1"
	sn.Terms = c.terms
	sn.Capacity = c.cap
	sn.Used = 0
	sn.LastHealthCheck = 0
	return mustEncode(t, &sn)
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

func (c *Client) callAddBlobber(t *testing.T, ssc *StorageSmartContract,
	now int64, balances chainState.StateContextI) (resp string, err error) {

	var tx = newTransaction(c.id, ADDRESS,
		int64(float64(c.terms.WritePrice)*sizeInGB(c.cap)), now)
	balances.(*testBalances).txn = tx
	var input = c.addBlobRequest(t)
	return ssc.addBlobber(tx, input, balances)
}

// addBlobber to SC
func addBlobber(t *testing.T, ssc *StorageSmartContract, cap, now int64,
	terms Terms, balacne state.Balance, balances chainState.StateContextI) (
	blob *Client) {

	var scheme = encryption.NewBLS0ChainScheme()
	scheme.GenerateKeys()

	blob = new(Client)
	blob.terms = terms
	blob.cap = cap
	blob.balance = balacne
	blob.scheme = scheme

	blob.pk = scheme.GetPublicKey()
	blob.id = encryption.Hash(blob.pk)

	balances.(*testBalances).balances[blob.id] = balacne

	var _, err = blob.callAddBlobber(t, ssc, now, balances)
	require.NoError(t, err)
	return
}

func (nar *newAllocationRequest) callNewAllocReq(t *testing.T, clientID string,
	value int64, ssc *StorageSmartContract, now int64,
	balances chainState.StateContextI) (resp string, err error) {

	var (
		input = mustEncode(t, nar)
		tx    = newTransaction(clientID, ADDRESS, value, now)
	)
	balances.(*testBalances).txn = tx
	return ssc.newAllocationRequest(tx, input, balances)
}

func (uar *updateAllocationRequest) callUpdateAllocReq(t *testing.T,
	clientID string, value int64, ssc *StorageSmartContract,
	balances chainState.StateContextI) (resp string, err error) {

	var input = mustEncode(t, uar)

	var tx transaction.Transaction
	tx.Hash = randString(32)
	tx.ClientID = clientID
	tx.ToClientID = ADDRESS
	tx.Value = value

	return ssc.newAllocationRequest(&tx, input, balances)
}

var avgTerms = Terms{
	//
}

// add allocation and 20 blobbers
func addAllocation(t *testing.T, ssc *StorageSmartContract, client *Client,
	now, exp int64, balances chainState.StateContextI) (allocID string,
	blobs []*Client) {

	nar = new(newAllocationRequest)
	nar.DataShards = 10
	nar.ParityShards = 10
	nar.Expiration = exp
	nar.Owner = client.id
	nar.OwnerPublicKey = client.pk
	nar.ReadPriceRange = PriceRange{1 * 1000, 10 * 1000}
	nar.WritePriceRange = PriceRange{20 * 1000, 200 * 1000}
	nar.Size = 2 * GB

	for i := 0; i < 20; i++ {
		addBlobber(t, ssc, 2*GB, now, avgTerms, 50*1000, balances)
	}

}

func mustSave(t *testing.T, key datastore.Key, val util.Serializable,
	balances chainState.StateContextI) {

	var _, err = balances.InsertTrieNode(key, val)
	require.NoError(t, err)
}

func setConfig(t *testing.T, balances chainState.StateContextI) (
	conf *scConfig) {

	conf = new(scConfig)

	conf.ChallengeEnabled = true
	conf.ChallengeRatePerMBMin = 1
	conf.MinAllocSize = 1 * GB
	conf.MinAllocDuration = 1 * time.Minute
	conf.MaxChallengeCompletionTime = 15 * time.Second
	conf.MinOfferDuration = 1 * time.Minute
	conf.MinBlobberCapacity = 1 * GB
	conf.ValidatorReward = 0.025
	conf.BlobberSlash = 0.1

	conf.ReadPool = &readPoolConfig{
		MinLock:       10,
		MinLockPeriod: 5 * time.Second,
		MaxLockPeriod: 20 * time.Minute,
	}
	conf.WritePool = &writePoolConfig{
		MinLock: 10,
	}

	mustSave(t, scConfigKey(ADDRESS), conf, balances)
	return
}
