package storagesc

import (
	cstate "0chain.net/chaincore/chain/state"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"strconv"
	"testing"
	"time"
)

type blobberStakes []state.Balance

const (
	//requestOwner       = "owin"
	errValueNotPresent = "value not present"
)

func TestNewAllocation(t *testing.T) {
	var stakes = blobberStakes{}
	var now = common.Timestamp(10000)
	scYaml = &scConfig{
		MinAllocSize:               1027,
		MinAllocDuration:           5 * time.Minute,
		MaxChallengeCompletionTime: 30 * time.Minute,
		MaxStake:                   zcnToBalance(100.0),
	}
	var blobberYaml = mockBlobberYaml{
		readPrice:               0.01,
		writePrice:              0.10,
		challengeCompletionTime: scYaml.MaxChallengeCompletionTime,
	}

	var request = newAllocationRequest{
		Owner:                      clientId,
		OwnerPublicKey:             "my public key",
		Size:                       scYaml.MinAllocSize,
		DataShards:                 3,
		ParityShards:               5,
		Expiration:                 common.Timestamp(scYaml.MinAllocDuration.Seconds()) + now,
		ReadPriceRange:             PriceRange{0, zcnToBalance(blobberYaml.readPrice) + 1},
		WritePriceRange:            PriceRange{0, zcnToBalance(blobberYaml.writePrice) + 1},
		MaxChallengeCompletionTime: blobberYaml.challengeCompletionTime + 1,
	}
	var goodBlobber = StorageNode{
		Capacity: 536870912,
		Used:     73,
		Terms: Terms{
			MaxOfferDuration:        1000 * scYaml.MinAllocDuration,
			ReadPrice:               zcnToBalance(blobberYaml.readPrice),
			ChallengeCompletionTime: blobberYaml.challengeCompletionTime,
		},
		LastHealthCheck: now - blobberHealthTime,
	}
	var blobbers = new(sortedBlobbers)
	var stake = scYaml.MaxStake
	var writePrice = blobberYaml.writePrice
	for i := 0; i < request.DataShards+request.ParityShards+4; i++ {
		var nextBlobber = goodBlobber
		nextBlobber.ID = strconv.Itoa(i)
		nextBlobber.Terms.WritePrice = zcnToBalance(writePrice)
		writePrice *= 0.9
		blobbers.add(&nextBlobber)
		stakes = append(stakes, stake)
		stake = stake / 10
	}

	t.Run("new allocation", func(t *testing.T) {
		err := testNewAllocation(t, request, *blobbers, *scYaml, blobberYaml, stakes)
		require.NoError(t, err)
	})

	t.Run("new allocation", func(t *testing.T) {
		var request2 = request
		request2.Size = 100 * GB

		err := testNewAllocation(t, request, *blobbers, *scYaml, blobberYaml, stakes)
		require.NoError(t, err)
	})
}

func testNewAllocation(t *testing.T, request newAllocationRequest, blobbers sortedBlobbers,
	scYaml scConfig, blobberYaml mockBlobberYaml, stakes blobberStakes,
) (err error) {
	require.EqualValues(t, len(blobbers), len(stakes))
	var f = formulaeCommitNewAllocation{
		scYaml:      scYaml,
		blobberYaml: blobberYaml,
		request:     request,
		blobbers:    blobbers,
		stakes:      stakes,
	}

	var txn = &transaction.Transaction{
		HashIDField: datastore.HashIDField{
			Hash: datastore.Key(transactionHash),
		},
		Value:        request.Size,
		ClientID:     clientId,
		ToClientID:   storageScId,
		CreationDate: creationDate,
	}
	var ctx = &mockStateContext{
		ctx: *cstate.NewStateContext(
			nil,
			&util.MerklePatriciaTrie{},
			&state.Deserializer{},
			txn,
			nil,
			nil,
			nil,
			nil,
		),
		clientBalance: zcnToBalance(3),
		store:         make(map[datastore.Key]util.Serializable),
	}
	var ssc = &StorageSmartContract{
		&sci.SmartContract{
			ID: storageScId,
		},
	}

	input, err := json.Marshal(request)
	require.NoError(t, err)

	var blobberList = new(StorageNodes)
	blobberList.Nodes = blobbers
	_, err = ctx.InsertTrieNode(ALL_BLOBBERS_KEY, blobberList)
	require.NoError(t, err)

	for i, blobber := range blobbers {
		var stakePool = newStakePool()
		stakePool.Pools["paula"] = &delegatePool{}
		stakePool.Pools["paula"].Balance = stakes[i] //state.Balance(request.Size + 1)
		require.NoError(t, stakePool.save(ssc.ID, blobber.ID, ctx))
	}

	var wPool = writePool{}
	require.NoError(t, wPool.save(ssc.ID, clientId, ctx))

	_, err = ctx.InsertTrieNode(scConfigKey(ssc.ID), &scYaml)
	require.NoError(t, err)

	_, err = ssc.newAllocationRequest(txn, input, ctx)
	if err != nil {
		return err
	}

	allBlobbersList, err := ssc.getBlobbersList(ctx)
	require.NoError(t, err)
	var individualBlobbers = sortedBlobbers{}
	for _, blobber := range allBlobbersList.Nodes {
		var b *StorageNode
		b, err = ssc.getBlobber(blobber.ID, ctx)
		if err != nil && err.Error() == errValueNotPresent {
			continue
		}
		require.NoError(t, err)
		individualBlobbers.add(b)
	}

	var newStakePools = []*stakePool{}
	for _, blobber := range allBlobbersList.Nodes {
		var sp, err = ssc.getStakePool(blobber.ID, ctx)
		require.NoError(t, err)
		newStakePools = append(newStakePools, sp)
	}
	var wp *writePool
	wp, err = ssc.getWritePool(clientId, ctx)
	wp = wp

	confirmTestNewAllocation(t, f, allBlobbersList.Nodes, individualBlobbers, newStakePools, *wp, ctx)

	return nil
}

type formulaeCommitNewAllocation struct {
	scYaml      scConfig
	blobberYaml mockBlobberYaml
	request     newAllocationRequest
	blobbers    sortedBlobbers
	stakes      blobberStakes
}

func (f formulaeCommitNewAllocation) blobbersUsed() int {
	return f.request.ParityShards + f.request.DataShards
}

func (f formulaeCommitNewAllocation) blobberEarnt(t *testing.T, id string, used []string) int64 {
	var totalWritePrice = 0.0
	var found = false
	for _, bId := range used {
		if bId == id {
			found = true
		}
		b, ok := f.blobbers.get(bId)
		require.True(t, ok)
		totalWritePrice += float64(b.Terms.WritePrice)
	}
	require.True(t, found)

	thisBlobber, ok := f.blobbers.get(id)
	require.True(t, ok)
	var ratio = float64(thisBlobber.Terms.WritePrice) / totalWritePrice
	var sizeOfWrite = float64(f.request.Size)

	return int64(sizeOfWrite * ratio)
}

func (f formulaeCommitNewAllocation) sizePerUsedBlobber() int64 {
	var numBlobbersUsed = int64(f.blobbersUsed())
	var writeSize = f.request.Size

	return (writeSize + numBlobbersUsed - 1) / numBlobbersUsed
}

func (f formulaeCommitNewAllocation) offerBlobber(index int) int64 {
	var amount = sizeInGB(f.sizePerUsedBlobber())
	var writePrice = float64(f.blobbers[index].Terms.WritePrice)

	return int64(amount * writePrice)
}

func (f formulaeCommitNewAllocation) capacityUsedBlobber(t *testing.T, id string) int64 {
	var thisBlobber, ok = f.blobbers.get(id)
	require.True(t, ok)
	var usedAlready = thisBlobber.Used
	var newAllocament = f.sizePerUsedBlobber()

	return usedAlready + newAllocament
}

func (f formulaeCommitNewAllocation) offerExpiration() common.Timestamp {
	var expiration = f.request.Expiration
	var challangeTime = f.request.MaxChallengeCompletionTime

	return expiration + toSeconds(challangeTime)
}

func confirmTestNewAllocation(t *testing.T, f formulaeCommitNewAllocation,
	blobbers1, blobbers2 sortedBlobbers, stakes []*stakePool, wp writePool, ctx cstate.StateContextI,
) {
	var transfers = ctx.GetTransfers()
	require.Len(t, transfers, 1)
	require.EqualValues(t, clientId, transfers[0].ClientID)
	require.EqualValues(t, storageScId, transfers[0].ToClientID)
	require.EqualValues(t, f.request.Size, transfers[0].Amount)

	require.Len(t, wp.Pools, 1)
	require.EqualValues(t, transactionHash, wp.Pools[0].ID)
	require.EqualValues(t, transactionHash, wp.Pools[0].AllocationID)
	require.EqualValues(t, f.request.Size, wp.Pools[0].Balance)
	require.Len(t, wp.Pools[0].Blobbers, f.blobbersUsed())
	var blobbersUsed []string
	for _, blobber := range wp.Pools[0].Blobbers {
		blobbersUsed = append(blobbersUsed, blobber.BlobberID)
	}
	for _, blobber := range wp.Pools[0].Blobbers {
		require.EqualValues(t, f.blobberEarnt(t, blobber.BlobberID, blobbersUsed), blobber.Balance)
	}

	var countUsedBlobbers = 0
	for _, blobber := range blobbers1 {
		b, ok := f.blobbers.get(blobber.ID)
		require.True(t, ok)
		if blobber.Used > b.Used {
			require.EqualValues(t, f.capacityUsedBlobber(t, blobber.ID), blobber.Used)
			countUsedBlobbers++
		}
	}
	require.EqualValues(t, f.blobbersUsed(), countUsedBlobbers)

	require.EqualValues(t, f.blobbersUsed(), len(blobbers2))
	for _, blobber := range blobbers2 {
		require.EqualValues(t, f.capacityUsedBlobber(t, blobber.ID), blobber.Used)
	}

	var countOffers = 0
	for i, stake := range stakes {
		offer, ok := stake.Offers[transactionHash]
		if ok {
			require.EqualValues(t, f.offerBlobber(i), int64(offer.Lock))
			require.EqualValues(t, f.offerExpiration(), offer.Expire)
			countOffers++
		}
	}
	require.EqualValues(t, f.blobbersUsed(), countOffers)
}
