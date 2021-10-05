package partitions

import (
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"0chain.net/core/common"

	"github.com/stretchr/testify/mock"

	"0chain.net/chaincore/mocks"
	"0chain.net/chaincore/state"
	"0chain.net/core/util"
)

const (
	minReadPrice               = 0
	maxReadPrice               = 0.04 * 1e10
	minWritePrice              = 0
	maxWritePrice              = 0.1 * 1e10
	minOfferDuration           = 10
	maxOfferDuration           = 24 * 365 * 10
	minBlobberCapacity         = 1
	maxBlobberCapacity         = 1000
	maxBlobberUsed             = maxBlobberCapacity
	maxChallengeCompletionTime = 30
	minAllocationSize          = 1
	maxAllocationSize          = 500
	mockNow                    = 10000
	maxShards                  = 2
	partitionsPerField         = 3
)

type MockTerms struct {
	ReadPrice               state.Balance `json:"read_price"`
	WritePrice              state.Balance `json:"write_price"`
	MinLockDemand           float64       `json:"min_lock_demand"`
	MaxOfferDuration        time.Duration `json:"max_offer_duration"`
	ChallengeCompletionTime time.Duration `json:"challenge_completion_time"`
}
type MockStorageNode struct {
	ID              string    `json:"id"`
	Terms           MockTerms `json:"terms"`    // terms
	Capacity        int64     `json:"capacity"` // total blobber capacity
	Used            int64     `json:"used"`     // allocated capacity
	LastHealthCheck common.Timestamp
}

func (sn *MockStorageNode) Name() string {
	return sn.ID
}

func (sn *MockStorageNode) randomise() {
	sn.Terms.ReadPrice = state.Balance(minReadPrice + rand.Intn(maxReadPrice-minReadPrice))
	sn.Terms.WritePrice = sn.Terms.ReadPrice + state.Balance(rand.Intn(maxWritePrice-int(sn.Terms.ReadPrice)))
	sn.Terms.MaxOfferDuration = time.Duration(rand.Intn(maxOfferDuration - minOfferDuration))
	sn.Terms.ChallengeCompletionTime = time.Duration(rand.Intn(maxChallengeCompletionTime))
	sn.Capacity = int64(minBlobberCapacity + rand.Intn(maxBlobberCapacity-minBlobberCapacity))
	sn.Used = int64(rand.Intn(int(sn.Capacity)))
	sn.LastHealthCheck = common.Timestamp(rand.Intn(mockNow))
}

func (sn *MockStorageNode) hash() string {
	var hash string
	hash += "read price " + convertToRangeHash(minReadPrice, maxReadPrice, int64(sn.Terms.ReadPrice)) + ","
	hash += "write price " + convertToRangeHash(minWritePrice, maxWritePrice, int64(sn.Terms.WritePrice)) + ","
	hash += "offer dur " + convertToRangeHash(minOfferDuration, maxOfferDuration, int64(sn.Terms.MaxOfferDuration)) + ","
	hash += "cct " + convertToRangeHash(0, maxChallengeCompletionTime, int64(sn.Terms.ChallengeCompletionTime)) + ","
	hash += "capacity " + convertToRangeHash(minBlobberCapacity, maxBlobberCapacity, sn.Capacity) + ","
	if sn.Capacity-sn.Used < maxAllocationSize {
		hash += "free " + convertToRangeHash(0, maxAllocationSize, sn.Capacity-sn.Used) + ","
	} else {
		hash += "free" + strconv.Itoa(partitionsPerField+1)
	}

	return hash
}

var convertToRangeHash = func(min, max, value int64) string {
	partitionSize := (max - min) / partitionsPerField
	partition := int((value - min) / partitionSize)
	return strconv.Itoa(partition)
}

type MockSnRange struct {
	maxReadPrice               state.Balance
	maxWritePrice              state.Balance
	maxOfferDuration           time.Duration
	spaceWanted                int64
	maxChallengeCompletionTime time.Duration
}

func (snr *MockSnRange) randomise() {
	snr.maxReadPrice = state.Balance(minReadPrice + rand.Intn(maxReadPrice-minReadPrice))
	snr.maxWritePrice = snr.maxReadPrice + state.Balance(rand.Intn(maxWritePrice-int(snr.maxReadPrice)))
	snr.maxOfferDuration = time.Duration(rand.Intn(maxOfferDuration - minOfferDuration))
	snr.spaceWanted = int64(minAllocationSize + rand.Intn(maxAllocationSize-minAllocationSize))
	snr.maxChallengeCompletionTime = time.Duration(rand.Intn(maxChallengeCompletionTime))
}

type mockPartitionIterator struct {
	snRange MockSnRange
	node    MockStorageNode
	at      int
	sizes   []int64
}

func (pi *mockPartitionIterator) Start(partitionRange PartitionRange) error {
	var ok bool
	pi.snRange, ok = partitionRange.(MockSnRange)
	if !ok {
		return fmt.Errorf("%v not MockSnRange", partitionRange)
	}
	pi.at = 0
	pi.node = MockStorageNode{
		Terms: MockTerms{
			ReadPrice:               pi.snRange.maxReadPrice,
			WritePrice:              pi.snRange.maxWritePrice,
			MaxOfferDuration:        pi.snRange.maxOfferDuration,
			ChallengeCompletionTime: pi.snRange.maxChallengeCompletionTime,
		},
	}
	spacePartition := int64((maxAllocationSize - minAllocationSize) / partitionsPerField)
	partitionAt := int(pi.snRange.spaceWanted / spacePartition)
	pi.sizes = []int64{}
	for i := 0; i < partitionsPerField-partitionAt; i++ {
		pi.sizes = append(pi.sizes, pi.snRange.spaceWanted+int64(i)*spacePartition)
	}
	return nil
}

func (pi *mockPartitionIterator) Next() string {
	if pi.at >= len(pi.sizes) {
		return ""
	}
	pi.node.Capacity = pi.sizes[pi.at]
	pi.at++
	return pi.node.hash()
}

func TestFuzzyCrossPartition(t *testing.T) {
	const (
		mockName                = "fuzzy league table"
		mockSeed                = 0
		fuzzyRunLength          = 100
		addRatio                = 60
		changeRatio             = 20
		removeRatio             = 20
		getRatio                = 50
		maxBlobberPerAllocation = 10
	)
	rand.Seed(mockSeed)
	type Action int
	const (
		Add Action = iota
		Remove
		Change
		Get
	)

	type methodCall struct {
		action  Action
		item1   MockStorageNode
		item2   MockStorageNode
		snRange MockSnRange
		count   int
	}

	var itemsMap = make(map[string]MockStorageNode)
	var items []string

	const hashFields = 6
	var startSize = int(math.Pow(hashFields, partitionsPerField) * maxBlobberPerAllocation)

	getAction := func(index int) methodCall {
		const totalRation = addRatio + changeRatio + removeRatio + getRatio
		var random int

		const addUntil = partitionsPerField * hashFields * maxShards
		if index > startSize {
			random = rand.Intn(totalRation)
		}
		var action methodCall
		switch {
		case random < addRatio:
			action = methodCall{
				action: Add,
				item1: MockStorageNode{
					ID: "sn " + strconv.Itoa(len(itemsMap)),
				},
			}
			action.item1.randomise()
			itemsMap[action.item1.ID] = action.item1
			items = append(items, action.item1.ID)
		case random < addRatio+removeRatio:
			index = rand.Intn(len(items))
			action = methodCall{
				action: Remove,
				item1:  itemsMap[items[index]],
			}
			items[index] = items[len(items)-1]
			items = items[:len(items)-1]
			delete(itemsMap, action.item1.ID)
		case random < addRatio+removeRatio+changeRatio:
			action = methodCall{
				action: Change,
				item1:  itemsMap[items[rand.Intn(len(items))]],
				item2:  MockStorageNode{},
			}
			action.item2.randomise()
			action.item2.ID = action.item1.ID
		default:
			action = methodCall{
				action:  Get,
				snRange: MockSnRange{},
				count:   1,
			}
			action.snRange.randomise()
			if len(itemsMap) > maxBlobberPerAllocation {
				action.count = 1 + rand.Intn(maxBlobberPerAllocation-1)
			}
		}
		return action
	}

	var mockItemToPartition ItemToPartition = func(item PartitionItem) (string, error) {
		sn, ok := item.(*MockStorageNode)
		if !ok {
			return "", fmt.Errorf("cannot convert %v to storage node", item)
		}
		return sn.hash(), nil
	}

	var validator = func(_ PartitionRange, id string) bool {
		if itemsMap[id].LastHealthCheck < mockNow*0.1 {
			return false
		}
		return true
	}

	var (
		balances = &mocks.StateContextI{}
		cp       = crossPartition{
			Name:            mockName,
			PartitionMap:    make(map[string]*revolvingPartition),
			ItemToPartition: mockItemToPartition,
			RangeIterator:   &mockPartitionIterator{},
			Validate:        validator,
		}
	)

	errList := []struct {
		testIndex int
		action    methodCall
		err       error
	}{}

	var processGetResult = func(mRange MockSnRange, results []string) {
		for _, result := range results {
			sn := itemsMap[result]
			free := sn.Capacity - sn.Used
			left := free - mRange.spaceWanted
			left = left
			require.True(t, mRange.spaceWanted <= sn.Capacity-sn.Used)
			require.True(t, sn.LastHealthCheck >= mockNow*0.1)
		}
	}

	balances.On(
		"GetTrieNode",
		mock.Anything,
	).Return(nil, util.ErrValueNotPresent).Maybe()
	fmt.Println("start size", startSize, "fuzzy length", fuzzyRunLength, "total", startSize+fuzzyRunLength)
	for i := 0; i < startSize+fuzzyRunLength; i++ {
		var err error
		var list []string
		action := getAction(i)
		if i >= startSize {
			fmt.Println("i", i, "action", action.action, "first", action.item1.ID, "second", action.item2.ID)
		}
		switch action.action {
		case Add:
			err = cp.Add(&action.item1, balances)
			require.NoError(t, err, fmt.Sprintf("action Add: %v, error: %v", action, err))
		case Remove:
			err = cp.Remove(&action.item1, balances)
			require.NoError(t, err, fmt.Sprintf("action Remove: %v, error: %v", action, err))

		case Change:
			err = cp.Change(&action.item1, &action.item2, balances)
			require.NoError(t, err, fmt.Sprintf("action Change: %v, error: %v", action, err))
			itemsMap[action.item1.ID] = action.item2
		case Get:
			list, err = cp.GetItems(action.snRange, action.count, balances)
			processGetResult(action.snRange, list)
			if err != nil {
				errList = append(errList, struct {
					testIndex int
					action    methodCall
					err       error
				}{
					testIndex: i,
					action:    action,
					err:       err,
				})
			}
		default:
			require.Fail(t, "action not found")
		}
	}
	require.True(t, true)

	/*
		sort.Slice(items, func(i, j int) bool {
			return items[i].item.Value > items[j].item.Value
		})
		itemIndex := 0
		for i := 0; i < len(lt.Divisions); i++ {
			for j := 0; j < len(lt.Divisions[i].Members); j++ {
				require.EqualValues(
					t,
					items[itemIndex].item.Value,
					lt.Divisions[i].Members[j].Value,
				)
				itemIndex++
			}
		}
		require.EqualValues(t, itemIndex, len(items))*/
}
