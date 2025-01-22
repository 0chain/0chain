package storagesc

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"0chain.net/core/config"
	"0chain.net/smartcontract/provider"
	"0chain.net/smartcontract/stakepool/spenum"

	"0chain.net/chaincore/threshold/bls"
	"github.com/0chain/common/core/currency"

	chainState "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/smartcontract/partitions"
	"github.com/0chain/common/core/logging"
	"github.com/0chain/common/core/util"

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
	terms        Terms
	cap          int64
	isRestricted bool
	isEnterprise bool

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

func newClientWithBalance(balance currency.Coin, balances chainState.StateContextI) (
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
	sn := &StorageNode{}
	sne := &storageNodeV4{
		IsRestricted:   new(bool),
		IsEnterprise:   new(bool),
		ManagingWallet: new(string),
	}
	sne.Provider.ProviderType = spenum.Blobber
	sne.ID = c.id
	sne.PublicKey = c.pk
	sne.BaseURL = getBlobberURL(c.id)
	sne.Terms = c.terms
	sne.Capacity = c.cap
	sne.Allocated = 0
	sne.LastHealthCheck = 0
	sne.StakePoolSettings.MaxNumDelegates = 100
	sne.StakePoolSettings.ServiceChargeRatio = 0.30 // 30%
	sne.StakePoolSettings.DelegateWallet = "rand_delegate_wallet"
	*sne.IsRestricted = c.isRestricted
	*sne.IsEnterprise = c.isEnterprise
	*sne.ManagingWallet = "rand_delegate_wallet"
	sn.SetEntity(sne)

	return mustEncode(t, &sn)
}

func (c *Client) stakeLockRequest(t testing.TB) []byte {
	spr := stakePoolRequest{
		ProviderType: spenum.Blobber,
		ProviderID:   c.id,
	}

	return mustEncode(t, &spr)
}

func (c *Client) addValidatorRequest(t testing.TB) []byte {
	var vn = newValidator(c.id)
	vn.ProviderType = spenum.Validator
	vn.BaseURL = getValidatorURL(c.id)
	vn.StakePoolSettings.MaxNumDelegates = 100
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
	tx.PublicKey = c.pk
	balances.(*testBalances).setTransaction(t, tx)
	var input = c.addBlobRequest(t)
	return ssc.addBlobber(tx, input, balances)
}

func (c *Client) callAddValidator(t testing.TB, ssc *StorageSmartContract,
	now int64, balances chainState.StateContextI) (resp string, err error) {

	var tx = newTransaction(c.id, ADDRESS, 0, now)
	balances.(*testBalances).setTransaction(t, tx)
	blobber := &StorageNode{}
	b := &storageNodeV3{
		Provider: provider.Provider{
			ID:           c.id,
			ProviderType: spenum.Blobber,
		},
	}
	blobber.SetEntity(b)

	//_, err = balances.InsertTrieNode(blobber.GetKey(ssc.ID), blobber)
	//require.NoError(t, err)
	var input = c.addValidatorRequest(t)
	return ssc.addValidator(tx, input, balances)
}

func updateBlobberUsingAddBlobber(t testing.TB, blob *StorageNode, value currency.Coin, now int64,
	ssc *StorageSmartContract, balances chainState.StateContextI) (
	resp string, err error) {

	var (
		input = blob.Encode()
		tx    = newTransaction(blob.Id(), ADDRESS, value, now)
	)
	balances.(*testBalances).setTransaction(t, tx)
	return ssc.addBlobber(tx, input, balances)
}

func updateBlobber(t testing.TB, blob *StorageNode, value currency.Coin, now int64,
	ssc *StorageSmartContract, balances chainState.StateContextI) (
	resp string, err error) {

	var (
		input = blob.Encode()
		tx    = newTransaction("rand_delegate_wallet", ADDRESS, value, now)
	)
	balances.(*testBalances).setTransaction(t, tx)
	return ssc.updateBlobberSettings(tx, input, balances)
}

func healthCheckBlobber(t testing.TB, blob *StorageNode, value currency.Coin, now int64, ssc *StorageSmartContract, balances chainState.StateContextI) (
	resp string, err error) {

	var (
		input = blob.Encode()
		tx    = newTransaction(blob.Id(), ADDRESS, value, now)
	)
	balances.(*testBalances).setTransaction(t, tx)
	resp, err = ssc.blobberHealthCheck(tx, input, balances)
	require.NoError(t, err)
	b, err := ssc.getBlobber(blob.Id(), balances)
	require.NoError(t, err)
	require.Equal(t, b.mustBase().LastHealthCheck, tx.CreationDate)
	return resp, err
}

func healthCheckValidator(t testing.TB, validator *ValidationNode, value currency.Coin, now int64, ssc *StorageSmartContract, balances chainState.StateContextI) (
	resp string, err error) {

	var (
		input = validator.Encode()
		tx    = newTransaction(validator.ID, ADDRESS, value, now)
	)
	balances.(*testBalances).setTransaction(t, tx)
	resp, err = ssc.validatorHealthCheck(tx, input, balances)
	require.NoError(t, err)
	v, err := ssc.getValidator(validator.ID, balances)
	require.NoError(t, err)
	require.Equal(t, v.LastHealthCheck, tx.CreationDate)
	return resp, err
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
	terms Terms, balance currency.Coin, balances chainState.StateContextI, isRestricted, isEnterprise bool) (
	blob *Client) {

	var scheme = encryption.NewBLS0ChainScheme()
	scheme.GenerateKeys() //nolint

	blob = new(Client)
	blob.terms = terms
	blob.cap = cap
	blob.balance = balance
	blob.scheme = scheme

	blob.isRestricted = isRestricted
	blob.isEnterprise = isEnterprise

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
	b, err := hex.DecodeString(valid.pk)
	require.NoError(t, err)
	valid.id = encryption.Hash(b)

	_, err = valid.callAddValidator(t, ssc, now, balances)
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
	ReadPrice:  1 * x10,
	WritePrice: 5 * x10,
}

// add allocation and 20 blobbers
func addAllocation(t testing.TB, ssc *StorageSmartContract, client *Client,
	now, allocSize, blobberCapacity int64, blobberBalance, lockTokens currency.Coin, nblobs int, balances chainState.StateContextI, preStakeTokens, isRestricted, IsEnterpriseAllocation bool, terms ...Terms) (
	allocID string, blobs []*Client) {

	if nblobs <= 0 {
		nblobs = 30
	}

	if lockTokens == 0 {
		lockTokens = 1000 * x10
	}

	if blobberCapacity == 0 {
		blobberCapacity = 2 * GB
	}

	if blobberBalance == 0 {
		blobberBalance = 50 * x10
	}

	setConfig(t, balances)

	var nar = new(newAllocationRequest)
	nar.DataShards = 10
	nar.ParityShards = 10
	nar.Owner = client.id
	nar.OwnerPublicKey = client.pk
	nar.ReadPriceRange = PriceRange{1 * x10, 10 * x10}
	nar.WritePriceRange = PriceRange{0 * x10, 20 * x10}

	nar.IsEnterprise = IsEnterpriseAllocation
	nar.AuthRoundExpiry = 1000000000

	if allocSize == 0 {
		nar.Size = 1 * GB // 20 GB
	} else {
		nar.Size = allocSize
	}

	for i := 0; i < nblobs; i++ {
		blobberTerms := avgTerms
		if len(terms) > 0 {
			blobberTerms = terms[0]
		}
		var b = addBlobber(t, ssc, blobberCapacity, now, blobberTerms, blobberBalance, balances, isRestricted, IsEnterpriseAllocation)
		nar.Blobbers = append(nar.Blobbers, b.id)

		if isRestricted || IsEnterpriseAllocation {
			blobberAuthTicket, err := b.scheme.Sign(encryption.Hash(fmt.Sprintf("%s_%d", client.id, 1000000000)))
			require.NoError(t, err)
			nar.BlobberAuthTickets = append(nar.BlobberAuthTickets, blobberAuthTicket)
		} else {
			nar.BlobberAuthTickets = append(nar.BlobberAuthTickets, "")
		}

		blobs = append(blobs, b)
	}

	if preStakeTokens {
		for i := 0; i < nblobs; i++ {
			sp, err := ssc.getStakePool(spenum.Blobber, blobs[i].id, balances)
			require.NoError(t, err)
			require.EqualValues(t, 0, sp.TotalOffers)
		}
	}

	var resp, err = nar.callNewAllocReq(t, client.id, lockTokens, ssc, now,
		balances)
	require.NoError(t, err)

	var deco StorageAllocation
	require.NoError(t, deco.Decode([]byte(resp)))

	return deco.mustBase().ID, blobs
}

func mustSave(t testing.TB, key datastore.Key, val util.MPTSerializable,
	balances chainState.StateContextI) {

	var _, err = balances.InsertTrieNode(key, val)
	require.NoError(t, err)
}

func setConfig(t testing.TB, balances chainState.StateContextI) (
	conf *Config) {

	conf = newConfig()

	conf.TimeUnit = 720 * time.Hour // use one hour as the time unit in the tests
	conf.ChallengeEnabled = true
	conf.ValidatorsPerChallenge = 10
	conf.MaxBlobbersPerAllocation = 10
	conf.MinAllocSize = 1 * KB
	conf.MinBlobberCapacity = 1 * GB
	conf.ValidatorReward = 0.025
	conf.BlobberSlash = 0.1
	conf.MaxReadPrice = 100e10  // 100 tokens per GB max allowed (by 64 KB)
	conf.MaxWritePrice = 100e10 // 100 tokens per GB max allowed
	conf.MinWritePrice = 0      // 0 tokens per GB min allowed
	conf.MaxDelegates = 200
	conf.MaxChallengeCompletionRounds = 720
	config.SmartContractConfig.Set("max_challenge_completion_rounds", 720)
	conf.MaxCharge = 0.50   // 50%
	conf.MinStake = 0.0     // 0 toks
	conf.MaxStake = 1000e10 // 100 toks
	conf.MaxBlobbersPerAllocation = 50
	conf.HealthCheckPeriod = time.Hour
	conf.ReadPool = &readPoolConfig{
		MinLock: 10,
	}
	conf.WritePool = &writePoolConfig{
		MinLock: 10,
	}

	conf.StakePool = &stakePoolConfig{}

	conf.BlockReward = &blockReward{
		BlockReward:             18 * 1e9,
		BlockRewardChangePeriod: 125000000,
		BlockRewardChangeRatio:  0.1,
		TriggerPeriod:           30,
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
		QualifyingStake: 1,
	}

	conf.CancellationCharge = 0.2
	conf.MaxIndividualFreeAllocation = 1000000
	conf.MaxTotalFreeAllocation = 100000000000000000

	conf.FreeAllocationSettings = freeAllocationSettings{
		DataShards:   4,
		ParityShards: 2,
		Size:         2147483648,
		WritePriceRange: PriceRange{
			Min: 0,
			Max: 100,
		},
		ReadPriceRange: PriceRange{
			Min: 0,
			Max: 100,
		},
		ReadPoolFraction: 0,
	}

	conf.NumValidatorsRewarded = 10

	mustSave(t, scConfigKey(ADDRESS), conf, balances)
	return
}

func genChall(t testing.TB, ssc *StorageSmartContract, now, roundCreatedAt int64, challID string, seed int64,
	valids *partitions.Partitions, allocID string,
	blobber *StorageNode, balances chainState.StateContextI) {

	sa, err := ssc.getAllocation(allocID, balances)
	require.NoError(t, err)
	alloc := sa.mustBase()

	allocChall, err := ssc.getAllocationChallenges(allocID, balances)
	if err != nil && !errors.Is(err, util.ErrValueNotPresent) {
		t.Fatal("unexpected error:", err)
	}
	if errors.Is(err, util.ErrValueNotPresent) {
		allocChall = new(AllocationChallenges)
		allocChall.AllocationID = allocID
	}
	var storChall = new(StorageChallenge)
	storChall.Created = common.Timestamp(now)
	storChall.RoundCreatedAt = roundCreatedAt
	storChall.ID = challID
	var valSlice []ValidationPartitionNode
	err = valids.GetRandomItems(balances, rand.New(rand.NewSource(seed)), &valSlice)
	valIDs := make([]string, len(valSlice))
	for i := range valSlice {
		valIDs[i] = valSlice[i].Id
	}
	storChall.TotalValidators = len(valSlice)
	storChall.ValidatorIDs = valIDs
	storChall.ValidatorIDMap = make(map[string]struct{}, len(storChall.ValidatorIDs))
	for _, vID := range storChall.ValidatorIDs {
		storChall.ValidatorIDMap[vID] = struct{}{}
	}

	storChall.AllocationID = allocID
	storChall.BlobberID = blobber.Id()

	require.True(t, allocChall.addChallenge(storChall))
	_, err = balances.InsertTrieNode(allocChall.GetKey(ssc.ID), allocChall)
	require.NoError(t, err)

	_, err = balances.InsertTrieNode(storChall.GetKey(ssc.ID), storChall)
	require.NoError(t, err)

	conf := setConfig(t, balances)
	conf.TimeUnit = 2 * time.Minute

	ba, ok := alloc.BlobberAllocsMap[blobber.Id()]
	if !ok {
		ba = newBlobberAllocation(alloc.bSize(), alloc, blobber.mustBase(), conf, common.Timestamp(now))
	}

	ba.Stats.OpenChallenges++
	ba.Stats.TotalChallenges++

	alloc.Stats.OpenChallenges++
	alloc.Stats.TotalChallenges++

	err = sa.save(balances, ssc.ID)
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
