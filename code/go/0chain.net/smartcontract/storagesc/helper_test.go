package storagesc

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"0chain.net/chaincore/config"
	"0chain.net/chaincore/currency"
	"0chain.net/chaincore/threshold/bls"

	chainState "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/logging"
	"0chain.net/core/util"
	"0chain.net/smartcontract/partitions"

	"go.uber.org/zap"

	"github.com/stretchr/testify/require"
)

// test helpers

func init() {
	rand.Seed(time.Now().UnixNano())

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
	balance currency.Coin
}

func newClient(balance currency.Coin, balances chainState.StateContextI) (
	client *Client) {

	var scheme = encryption.NewBLS0ChainScheme()
	scheme.GenerateKeys() //nolint

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

func getBlobberURL(id string) string {
	return "http://" + id + ":9081/api/v1"
}

func blobberIDByURL(url string) (id string) {
	return url[len("http://") : len(url)-len(":9081/api/v1")]
}

func getValidatorURL(id string) string {
	return "http://" + id + ":10291/api/v1"
}

func (c *Client) addBlobRequest(t testing.TB) []byte {
	var sn StorageNode
	sn.ID = c.id
	sn.BaseURL = getBlobberURL(c.id)
	sn.Terms = c.terms
	sn.Capacity = c.cap
	sn.Allocated = 0
	sn.LastHealthCheck = 0
	sn.StakePoolSettings.MaxNumDelegates = 100
	sn.StakePoolSettings.MinStake = 0
	sn.StakePoolSettings.MaxStake = 1000e10
	sn.StakePoolSettings.ServiceChargeRatio = 0.30 // 30%
	return mustEncode(t, &sn)
}

func (c *Client) stakeLockRequest(t testing.TB) []byte {
	var spr stakePoolRequest
	spr.BlobberID = c.id
	return mustEncode(t, &spr)
}

func (c *Client) addValidatorRequest(t testing.TB) []byte {
	var vn ValidationNode
	vn.ID = c.id
	vn.BaseURL = getValidatorURL(c.id)
	vn.StakePoolSettings.MaxNumDelegates = 100
	vn.StakePoolSettings.MinStake = 0
	vn.StakePoolSettings.MaxStake = 1000e10
	return mustEncode(t, &vn)
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

func (c *Client) callAddBlobber(t testing.TB, ssc *StorageSmartContract,
	now int64, balances chainState.StateContextI) (resp string, err error) {

	txVal, err := currency.Float64ToCoin(float64(c.terms.WritePrice) * sizeInGB(c.cap))
	require.NoError(t, err)
	var tx = newTransaction(c.id, ADDRESS, txVal, now)
	balances.(*testBalances).setTransaction(t, tx)
	var input = c.addBlobRequest(t)
	return ssc.addBlobber(tx, input, balances)
}

func (c *Client) callAddValidator(t testing.TB, ssc *StorageSmartContract,
	now int64, balances chainState.StateContextI) (resp string, err error) {

	var tx = newTransaction(c.id, ADDRESS, 0, now)
	balances.(*testBalances).setTransaction(t, tx)
	blobber := new(StorageNode)
	blobber.ID = c.id
	_, err = balances.InsertTrieNode(blobber.GetKey(ssc.ID), blobber)
	require.NoError(t, err)
	var input = c.addValidatorRequest(t)
	return ssc.addValidator(tx, input, balances)
}

func updateBlobber(t testing.TB, blob *StorageNode, value currency.Coin, now int64,
	ssc *StorageSmartContract, balances chainState.StateContextI) (
	resp string, err error) {

	var (
		input = blob.Encode()
		tx    = newTransaction(blob.ID, ADDRESS, value, now)
	)
	balances.(*testBalances).setTransaction(t, tx)
	return ssc.addBlobber(tx, input, balances)
}

// pseudo-random IPv4 address by ID (never used)
//
// func blobAddress(t *testing.T, id string) (ip string) {
// 	t.Helper()
// 	require.True(t, len(id) < 8)
// 	var seed int64
// 	fmt.Sscanf(id[:8], "%x", &seed)
// 	var rnd = rand.New(rand.NewSource(seed))
// 	ip = fmt.Sprintf("http://%d.%d.%d.%d/api", rnd.Int63n(255), rnd.Int63n(255),
// 		rnd.Int63n(255), rnd.Int63n(255))
// 	return
// }

// addBlobber to SC
func addBlobber(t testing.TB, ssc *StorageSmartContract, cap, now int64,
	terms Terms, balance currency.Coin, balances chainState.StateContextI) (
	blob *Client) {

	var scheme = encryption.NewBLS0ChainScheme()
	scheme.GenerateKeys() //nolint

	blob = new(Client)
	blob.terms = terms
	blob.cap = cap
	blob.balance = balance
	blob.scheme = scheme

	blob.pk = scheme.GetPublicKey()
	blob.id = encryption.Hash(blob.pk)

	balances.(*testBalances).balances[blob.id] = balance

	var _, err = blob.callAddBlobber(t, ssc, now, balances)
	require.NoError(t, err)

	txVal, err := currency.Float64ToCoin(float64(terms.WritePrice) * sizeInGB(cap))
	require.NoError(t, err)
	// add stake for the blobber as blobber owner
	var tx = newTransaction(blob.id, ADDRESS, txVal, now)
	balances.(*testBalances).setTransaction(t, tx)
	_, err = ssc.stakePoolLock(tx, blob.stakeLockRequest(t), balances)
	require.NoError(t, err)
	return
}

// addValidator to SC
func addValidator(t testing.TB, ssc *StorageSmartContract, now int64,
	balances chainState.StateContextI) (valid *Client) {

	var scheme = encryption.NewBLS0ChainScheme()
	scheme.GenerateKeys() //nolint

	valid = new(Client)
	valid.scheme = scheme

	valid.pk = scheme.GetPublicKey()
	valid.id = encryption.Hash(valid.pk)

	var _, err = valid.callAddValidator(t, ssc, now, balances)
	require.NoError(t, err)
	return
}

func (c *Client) validTicket(t testing.TB, challID, blobID string, ok bool,
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

func (nar *newAllocationRequest) callNewAllocReq(t testing.TB, clientID string,
	value currency.Coin, ssc *StorageSmartContract, now int64,
	balances chainState.StateContextI) (resp string, err error) {

	var (
		input = mustEncode(t, nar)
		tx    = newTransaction(clientID, ADDRESS, value, now)
	)
	balances.(*testBalances).setTransaction(t, tx)
	return ssc.newAllocationRequest(tx, input, balances, nil)
}

func (uar *updateAllocationRequest) callUpdateAllocReq(t testing.TB,
	clientID string, value currency.Coin, now int64, ssc *StorageSmartContract,
	balances chainState.StateContextI) (resp string, err error) {

	var (
		input = mustEncode(t, uar)
		tx    = newTransaction(clientID, ADDRESS, value, now)
	)
	balances.(*testBalances).setTransaction(t, tx)
	return ssc.updateAllocationRequest(tx, input, balances)
}

var avgTerms = Terms{
	ReadPrice:        1 * x10,
	WritePrice:       5 * x10,
	MinLockDemand:    0.1,
	MaxOfferDuration: 1 * time.Hour,
}

// add allocation and 20 blobbers
func addAllocation(t testing.TB, ssc *StorageSmartContract, client *Client,
	now, exp int64, nblobs int, balances chainState.StateContextI) (
	allocID string, blobs []*Client) {

	if nblobs <= 0 {
		nblobs = 30
	}

	setConfig(t, balances)

	var nar = new(newAllocationRequest)
	nar.DataShards = 10
	nar.ParityShards = 10
	nar.Expiration = common.Timestamp(exp)
	nar.Owner = client.id
	nar.OwnerPublicKey = client.pk
	nar.ReadPriceRange = PriceRange{1 * x10, 10 * x10}
	nar.WritePriceRange = PriceRange{2 * x10, 20 * x10}
	nar.Size = 1 * GB // 2 GB

	for i := 0; i < nblobs; i++ {
		var b = addBlobber(t, ssc, 2*GB, now, avgTerms, 50*x10, balances)
		nar.Blobbers = append(nar.Blobbers, b.id)
		blobs = append(blobs, b)
	}

	var resp, err = nar.callNewAllocReq(t, client.id, 15*x10, ssc, now,
		balances)
	require.NoError(t, err)

	var deco StorageAllocation
	require.NoError(t, deco.Decode([]byte(resp)))

	return deco.ID, blobs
}

func mustSave(t testing.TB, key datastore.Key, val util.MPTSerializable,
	balances chainState.StateContextI) {

	var _, err = balances.InsertTrieNode(key, val)
	require.NoError(t, err)
}

func setConfig(t testing.TB, balances chainState.StateContextI) (
	conf *Config) {

	conf = new(Config)

	conf.TimeUnit = 48 * time.Hour // use one hour as the time unit in the tests
	conf.ChallengeEnabled = true
	conf.ChallengeGenerationRate = 1
	conf.MaxChallengesPerGeneration = 100
	conf.ValidatorsPerChallenge = 10
	conf.MaxBlobbersPerAllocation = 10
	conf.FailedChallengesToCancel = 100
	conf.FailedChallengesToRevokeMinLock = 50
	conf.MinAllocSize = 1 * GB
	conf.MinAllocDuration = 1 * time.Minute
	conf.MinOfferDuration = 1 * time.Minute
	conf.MinBlobberCapacity = 1 * GB
	conf.ValidatorReward = 0.025
	conf.BlobberSlash = 0.1
	conf.MaxReadPrice = 100e10  // 100 tokens per GB max allowed (by 64 KB)
	conf.MaxWritePrice = 100e10 // 100 tokens per GB max allowed
	conf.MinWritePrice = 0      // 100 tokens per GB max allowed
	conf.MaxDelegates = 200
	conf.MaxChallengeCompletionTime = 5 * time.Minute
	config.SmartContractConfig.Set(confMaxChallengeCompletionTime, "5m")

	conf.MaxCharge = 0.50   // 50%
	conf.MinStake = 0.0     // 0 toks
	conf.MaxStake = 1000e10 // 100 toks
	conf.MaxMint = 100e10
	conf.MaxBlobbersPerAllocation = 50

	conf.ReadPool = &readPoolConfig{
		MinLock: 10,
	}
	conf.WritePool = &writePoolConfig{
		MinLock: 10,
	}

	conf.StakePool = &stakePoolConfig{
		MinLock: 10,
	}

	conf.BlockReward = &blockReward{
		BlockReward:             1000,
		BlockRewardChangePeriod: 1000,
		BlockRewardChangeRatio:  0.1,
		TriggerPeriod:           30,
		BlobberWeight:           0.5,
		Gamma: blockRewardGamma{
			Alpha: 0.2,
			A:     10,
			B:     9,
		},
		Zeta: blockRewardZeta{
			Mu: 0.2,
			I:  1,
			K:  0.9,
		},
	}

	mustSave(t, scConfigKey(ADDRESS), conf, balances)
	return
}

func genChall(t testing.TB, ssc *StorageSmartContract,
	blobberID string, now int64, prevID, challID string, seed int64,
	valids *partitions.Partitions, allocID string, blobber *StorageNode,
	allocRoot string, balances chainState.StateContextI) {

	allocChall, err := ssc.getAllocationChallenges(allocID, balances)
	if err != nil && err != util.ErrValueNotPresent {
		t.Fatal("unexpected error:", err)
	}
	if err == util.ErrValueNotPresent {
		allocChall = new(AllocationChallenges)
		allocChall.AllocationID = allocID
	}
	var storChall = new(StorageChallenge)
	storChall.Created = common.Timestamp(now)
	storChall.ID = challID
	var valSlice []ValidationPartitionNode
	err = valids.GetRandomItems(balances, rand.New(rand.NewSource(seed)), &valSlice)
	valIDs := make([]string, len(valSlice))
	for i := range valSlice {
		valIDs[i] = valSlice[i].Id
	}
	storChall.TotalValidators = len(valSlice)
	storChall.ValidatorIDs = valIDs

	storChall.AllocationID = allocID
	storChall.BlobberID = blobber.ID

	require.True(t, allocChall.addChallenge(storChall))
	_, err = balances.InsertTrieNode(allocChall.GetKey(ssc.ID), allocChall)
	require.NoError(t, err)

	_, err = balances.InsertTrieNode(storChall.GetKey(ssc.ID), storChall)
	require.NoError(t, err)
	return
}

func newTestStorageSC() (ssc *StorageSmartContract) {
	ssc = new(StorageSmartContract)
	ssc.SmartContract = new(smartcontractinterface.SmartContract)
	ssc.ID = ADDRESS
	return
}

func stakePoolTotal(sp *stakePool) (total currency.Coin, err error) {
	for _, id := range sp.OrderedPoolIds() {
		total, err = currency.AddCoin(total, sp.Pools[id].Balance)
		if err != nil {
			return 0, err
		}
	}
	return
}
