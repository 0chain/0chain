package partitions

import (
	"fmt"
	"math/rand"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/mocks"
	"0chain.net/core/util"
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
	rs := randomSelector{
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

func Test_randomSelector_swap(t *testing.T) {
	partition1, partition2 := mockValidatorsList(2), mockValidatorsList(2)

	itemA, itemB := 0, 1

	wantPartition1, wantPartition2 := copyValidatorsList(partition1), copyValidatorsList(partition2)
	wantPartition1.Items[itemA], wantPartition2.Items[itemB] = wantPartition2.Items[itemB], wantPartition1.Items[itemA]

	type (
		fields struct {
			Name          string
			PartitionSize int
			NumPartitions int
			Partitions    []PartitionItemList
			Callback      ChangePartitionCallback
			ItemType      ItemType
		}

		args struct {
			partitionA int
			itemA      int
			partitionB int
			itemB      int
		}
	)
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *randomSelector
		wantErr error
	}{
		{
			name: "OK",
			fields: fields{
				Partitions: []PartitionItemList{
					partition1,
					partition2,
				},
			},
			args: args{
				partitionA: 0,
				itemA:      itemA,
				partitionB: 1,
				itemB:      itemB,
			},
			want: &randomSelector{
				Partitions: []PartitionItemList{
					wantPartition1,
					wantPartition2,
				},
			},
		},
		{
			name: "partitionA_NegativeIndex_ERR",
			fields: fields{
				Partitions: []PartitionItemList{
					partition1,
					partition2,
				},
			},
			args: args{
				partitionA: -1,
				itemA:      itemA,
				partitionB: 1,
				itemB:      itemB,
			},
			wantErr: IndexOutOfBounds,
		},
		{
			name: "itemA_NegativeIndex_ERR",
			fields: fields{
				Partitions: []PartitionItemList{
					partition1,
					partition2,
				},
			},
			args: args{
				partitionA: 0,
				itemA:      -1,
				partitionB: 1,
				itemB:      itemB,
			},
			wantErr: IndexOutOfBounds,
		},
		{
			name: "partitionB_NegativeIndex_ERR",
			fields: fields{
				Partitions: []PartitionItemList{
					partition1,
					partition2,
				},
			},
			args: args{
				partitionA: 0,
				itemA:      itemA,
				partitionB: -1,
				itemB:      itemB,
			},
			wantErr: IndexOutOfBounds,
		},
		{
			name: "itemB_NegativeIndex_ERR",
			fields: fields{
				Partitions: []PartitionItemList{
					partition1,
					partition2,
				},
			},
			args: args{
				partitionA: 0,
				itemA:      itemA,
				partitionB: 1,
				itemB:      -1,
			},
			wantErr: IndexOutOfBounds,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rs := &randomSelector{
				Name:          tt.fields.Name,
				PartitionSize: tt.fields.PartitionSize,
				NumPartitions: tt.fields.NumPartitions,
				Partitions:    tt.fields.Partitions,
				Callback:      tt.fields.Callback,
				ItemType:      tt.fields.ItemType,
			}
			err := rs.swap(tt.args.partitionA, tt.args.itemA, tt.args.partitionB, tt.args.itemB)
			require.ErrorIs(t, err, tt.wantErr)
			if tt.wantErr == nil {
				require.Equal(t, tt.want, rs)
			}
		})
	}
}
