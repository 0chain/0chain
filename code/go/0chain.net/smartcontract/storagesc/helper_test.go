package storagesc

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"testing"
	"time"

	"0chain.net/chaincore/chain"
	chainState "0chain.net/chaincore/chain/state"
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
	chain.ServerChain = new(chain.Chain)
	chain.ServerChain.Config = new(chain.Config)
	chain.ServerChain.ClientSignatureScheme = "bls0chain"

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

func (c *Client) stakeLockRequest(t *testing.T) []byte {
	var spr stakePoolRequest
	spr.BlobberID = c.id
	return mustEncode(t, &spr)
}

func (c *Client) addValidatorRequest(t *testing.T) []byte {
	var vn ValidationNode
	vn.ID = c.id
	vn.BaseURL = "http://" + c.id + ":10291/api/v1"
	return mustEncode(t, &vn)
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

func (c *Client) callAddValidator(t *testing.T, ssc *StorageSmartContract,
	now int64, balances chainState.StateContextI) (resp string, err error) {

	var tx = newTransaction(c.id, ADDRESS, 0, now)
	balances.(*testBalances).txn = tx
	var input = c.addValidatorRequest(t)
	return ssc.addValidator(tx, input, balances)
}

func updateBlobber(t *testing.T, blob *StorageNode, value, now int64,
	ssc *StorageSmartContract, balances chainState.StateContextI) (
	resp string, err error) {

	var (
		input = blob.Encode()
		tx    = newTransaction(blob.ID, ADDRESS, value, now)
	)
	balances.(*testBalances).txn = tx
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

	// add stake for the blobber as blobber owner
	var tx = newTransaction(blob.id, ADDRESS,
		int64(float64(terms.WritePrice)*sizeInGB(cap)), now)
	balances.(*testBalances).txn = tx
	_, err = ssc.stakePoolLock(tx, blob.stakeLockRequest(t), balances)
	require.NoError(t, err)
	return
}

// addValidator to SC
func addValidator(t *testing.T, ssc *StorageSmartContract, now int64,
	balances chainState.StateContextI) (valid *Client) {

	var scheme = encryption.NewBLS0ChainScheme()
	scheme.GenerateKeys()

	valid = new(Client)
	valid.scheme = scheme

	valid.pk = scheme.GetPublicKey()
	valid.id = encryption.Hash(valid.pk)

	var _, err = valid.callAddValidator(t, ssc, now, balances)
	require.NoError(t, err)
	return
}

func (c *Client) validTicket(t *testing.T, challID, blobID string, ok bool,
	now int64) (vt *ValidationTicket) {

	vt = new(ValidationTicket)
	vt.ChallengeID = challID
	vt.BlobberID = blobID
	vt.ValidatorID = c.id
	vt.ValidatorKey = c.pk
	vt.Result = ok
	vt.Message = ""
	vt.MessageCode = ""
	vt.Timestamp = common.Timestamp(now)

	var data = fmt.Sprintf("%v:%v:%v:%v:%v:%v", vt.ChallengeID, vt.BlobberID,
		vt.ValidatorID, vt.ValidatorKey, vt.Result, vt.Timestamp)
	var (
		hash = encryption.Hash(data)
		err  error
	)
	vt.Signature, err = c.scheme.Sign(hash)
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
	clientID string, value, now int64, ssc *StorageSmartContract,
	balances chainState.StateContextI) (resp string, err error) {

	var (
		input = mustEncode(t, uar)
		tx    = newTransaction(clientID, ADDRESS, value, now)
	)
	balances.(*testBalances).txn = tx
	return ssc.updateAllocationRequest(tx, input, balances)
}

var avgTerms = Terms{
	ReadPrice:               1 * x10,
	WritePrice:              5 * x10,
	MinLockDemand:           0.1,
	MaxOfferDuration:        1 * time.Hour,
	ChallengeCompletionTime: 200 * time.Second,
}

// add allocation and 20 blobbers
func addAllocation(t *testing.T, ssc *StorageSmartContract, client *Client,
	now, exp int64, balances chainState.StateContextI) (allocID string,
	blobs []*Client) {

	setConfig(t, balances)

	var nar = new(newAllocationRequest)
	nar.DataShards = 10
	nar.ParityShards = 10
	nar.Expiration = common.Timestamp(exp)
	nar.Owner = client.id
	nar.OwnerPublicKey = client.pk
	nar.ReadPriceRange = PriceRange{1 * x10, 10 * x10}
	nar.WritePriceRange = PriceRange{2 * x10, 20 * x10}
	nar.Size = 2 * GB // 2 GB
	nar.MaxChallengeCompletionTime = 200 * time.Hour

	for i := 0; i < 30; i++ {
		var b = addBlobber(t, ssc, 2*GB, now, avgTerms, 50*x10, balances)
		blobs = append(blobs, b)
	}

	var resp, err = nar.callNewAllocReq(t, client.id, 15*x10, ssc, now,
		balances)
	require.NoError(t, err)

	var deco StorageAllocation
	require.NoError(t, deco.Decode([]byte(resp)))

	return deco.ID, blobs
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
	conf.ChallengeGenerationRate = 1
	conf.FailedChallengesToCancel = 100
	conf.FailedChallengesToRevokeMinLock = 50
	conf.MinAllocSize = 1 * GB
	conf.MinAllocDuration = 1 * time.Minute
	conf.MaxChallengeCompletionTime = 15 * time.Second
	conf.MinOfferDuration = 1 * time.Minute
	conf.MinBlobberCapacity = 1 * GB
	conf.ValidatorReward = 0.025
	conf.BlobberSlash = 0.1
	conf.MaxReadPrice = 100e10  // 100 tokens per GB max allowed (by 64 KB)
	conf.MaxWritePrice = 100e10 // 100 tokens per GB max allowed

	conf.ReadPool = &readPoolConfig{
		MinLock:       10,
		MinLockPeriod: 5 * time.Second,
		MaxLockPeriod: 20 * time.Minute,
	}
	conf.WritePool = &writePoolConfig{
		MinLock:       10,
		MinLockPeriod: 5 * time.Second,
		MaxLockPeriod: 20 * time.Minute,
	}

	conf.StakePool = &stakePoolConfig{
		MinLock:          10,
		InterestRate:     0.01,
		InterestInterval: 5 * time.Second,
	}

	mustSave(t, scConfigKey(ADDRESS), conf, balances)
	return
}

func genChall(t *testing.T, ssc *StorageSmartContract,
	blobberID string, now int64, prevID, challID string, seed int64,
	valids []*ValidationNode, allocID string, blobber *StorageNode,
	allocRoot string, balances chainState.StateContextI) {

	var blobberChall, err = ssc.getBlobberChallenge(blobberID, balances)
	if err != nil && err != util.ErrValueNotPresent {
		t.Fatal("unexpected error:", err)
	}
	if err == util.ErrValueNotPresent {
		blobberChall = new(BlobberChallenge)
		blobberChall.BlobberID = blobberID
	}
	var storChall = new(StorageChallenge)
	storChall.Created = common.Timestamp(now)
	storChall.ID = challID
	storChall.PrevID = prevID
	storChall.Validators = valids
	storChall.RandomNumber = seed
	storChall.AllocationID = allocID
	storChall.Blobber = blobber
	storChall.AllocationRoot = allocRoot

	require.True(t, blobberChall.addChallenge(storChall))
	_, err = balances.InsertTrieNode(blobberChall.GetKey(ssc.ID), blobberChall)
	require.NoError(t, err)
	return
}

func newTestStorageSC() (ssc *StorageSmartContract) {
	ssc = new(StorageSmartContract)
	ssc.SmartContract = new(smartcontractinterface.SmartContract)
	ssc.ID = ADDRESS
	return
}
