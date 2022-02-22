package partitions

import (
	"fmt"
	"math/rand"
	"strconv"
	"testing"

	"0chain.net/chaincore/mocks"

	"0chain.net/chaincore/chain/state"
	"0chain.net/core/util"
	"github.com/stretchr/testify/require"
)

func TestFuzzyRandom(t *testing.T) {
	const (
		mockName               = "fuzzy league table"
		mockSeed         int64 = 0
		fuzzyRunLength         = 10000
		mockDivisionSize       = 100
		addRatio               = 60
		changeRation           = 20
		removeRatio            = 20
	)
	rand.Seed(mockSeed)
	type Action int
	const (
		Add Action = iota
		Remove
		GetRandomPartition
	)

	type methodCall struct {
		action     Action
		item       PartitionItem
		divisionId int
	}

	type fuzzyItem struct {
		item     string
		division int
	}
	items := []fuzzyItem{}

	getAction := func(testId int) methodCall {
		const totalRation = addRatio + changeRation + removeRatio
		var random int
		if len(items) > 0 {
			random = rand.Intn(totalRation)
		}

		var action methodCall
		switch {
		case random < addRatio:
			action = methodCall{
				action: Add,
				item:   ItemFromString("test " + strconv.Itoa(testId)),
			}
		case random < addRatio+removeRatio:
			toRemove := items[rand.Intn(len(items))]
			action = methodCall{
				action:     Remove,
				item:       ItemFromString(toRemove.item),
				divisionId: toRemove.division,
			}
		default:
			action = methodCall{
				action: GetRandomPartition,
			}
		}
		return action
	}

	var mockCallBack ChangePartitionCallback = func(
		item PartitionItem,
		from, to int,
		_ state.StateContextI,
	) error {
		for i := 0; i < len(items); i++ {
			if items[i].item == item.Name() {
				if items[i].division != from {
					require.EqualValues(t, items[i].division, from)
				}
				require.EqualValues(t, items[i].division, from)
				items[i].division = to
				return nil
			}
		}

		return fmt.Errorf("not found %v, %d, %d", item, from, to)
	}

	balances := &mocks.StateContextI{}
	rs := RandomSelector{
		Name:          mockName,
		PartitionSize: mockDivisionSize,
		Callback:      mockCallBack,
	}

	for i := 0; i <= fuzzyRunLength/mockDivisionSize; i++ {
		balances.On(
			"GetTrieNode",
			rs.partitionKey(i),
		).Return(nil, util.ErrValueNotPresent).Maybe()
		balances.On(
			"DeleteTrieNode",
			rs.partitionKey(i),
		).Return("", nil).Maybe()
	}

	r := rand.New(rand.NewSource(mockSeed))
	for i := 0; i < fuzzyRunLength; i++ {
		action := getAction(i)
		switch action.action {
		case Add:
			partition, err := rs.AddRand(action.item, r, balances)
			require.NoError(t, err, fmt.Sprintf("action Add: %v, error: %v", action, err))
			items = append(items, fuzzyItem{
				item:     action.item.Name(),
				division: partition,
			})

		case Remove:
			err := rs.Remove(action.item, action.divisionId, balances)
			require.NoError(t, err, fmt.Sprintf("action Remove: %v, error: %v", action, err))
			for index, fuzzyItem := range items {
				if fuzzyItem.item == action.item.Name() {
					items[index] = items[len(items)-1]
					items = items[:len(items)-1]
					break
				}
			}
		case GetRandomPartition:
			list, err := rs.GetRandomSlice(r, balances)
			require.NoError(t, err, fmt.Sprintf("action Change: %v, error: %v", action, err))
			require.True(t, len(list) <= rs.PartitionSize)
		default:
			require.Fail(t, "action not found")
		}
	}

	var count = 0
	for i := 0; i < len(rs.Partitions); i++ {
		count += rs.Partitions[i].length()
	}
	require.EqualValues(t, count, len(items))

}
