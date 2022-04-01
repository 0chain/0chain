package partitions

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/mocks"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
)

func Test_PartitionItemList_getByIndex(t *testing.T) {
	type (
		args struct {
			item PartitionItem
			idx  int
		}

		caseData struct {
			name    string
			list    PartitionItemList
			args    args
			wantErr error
		}
	)

	var (
		length = 2
	)
	cases := []caseData{
		{
			name: "itemList_OK",
			list: mockItemList(length),
			args: args{
				item: func() PartitionItem {
					item := mockStringItem()
					return &item
				}(),
				idx: length,
			},
		},
		{
			name: "itemList_NegativeIndex_ERR",
			list: mockItemList(length),
			args: args{
				idx: -1,
			},
			wantErr: IndexOutOfBounds,
		},
		{
			name: "itemList_IndexOverLength_ERR",
			list: mockItemList(length),
			args: args{
				idx: length + 1,
			},
			wantErr: IndexOutOfBounds,
		},
		{
			name: "validatorsItemList",
			list: mockValidatorsList(length),
			args: args{
				item: func() PartitionItem {
					item := mockValidatorItem()
					return &item
				}(),
				idx: length,
			},
		},
		{
			name: "validatorsItemList_NegativeIndex_ERR",
			list: mockValidatorsList(length),
			args: args{
				idx: -1,
			},
			wantErr: IndexOutOfBounds,
		},
		{
			name: "validatorsItemList_IndexOverLength_ERR",
			list: mockValidatorsList(length),
			args: args{
				idx: length + 1,
			},
			wantErr: IndexOutOfBounds,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if testCase.args.item != nil {
				testCase.list.add(testCase.args.item)
			}

			got, err := testCase.list.getByIndex(testCase.args.idx)
			require.ErrorIs(t, err, testCase.wantErr)
			require.Equal(t, testCase.args.item, got)
		})
	}
}

func Test_PartitionItemList_set(t *testing.T) {
	type (
		args struct {
			item PartitionItem
			idx  int
		}

		caseData struct {
			name    string
			list    PartitionItemList
			args    args
			wantErr error
		}
	)

	var (
		length = 2
	)
	cases := []caseData{
		{
			name: "itemList_OK",
			list: mockItemList(length),
			args: args{
				item: func() PartitionItem {
					item := mockStringItem()
					return &item
				}(),
				idx: 0,
			},
		},
		{
			name: "itemList_NegativeIndex_ERR",
			list: mockItemList(length),
			args: args{
				idx: -1,
			},
			wantErr: IndexOutOfBounds,
		},
		{
			name: "itemList_IndexOverLength_ERR",
			list: mockItemList(length),
			args: args{
				idx: length + 1,
			},
			wantErr: IndexOutOfBounds,
		},
		{
			name: "validatorsItemList_OK",
			list: mockValidatorsList(length),
			args: args{
				item: func() PartitionItem {
					item := mockValidatorItem()
					return &item
				}(),
				idx: 0,
			},
		},
		{
			name: "validatorsItemList_NegativeIndex_ERR",
			list: mockValidatorsList(length),
			args: args{
				idx: -1,
			},
			wantErr: IndexOutOfBounds,
		},
		{
			name: "validatorsItemList_IndexOverLength_ERR",
			list: mockValidatorsList(length),
			args: args{
				idx: length + 1,
			},
			wantErr: IndexOutOfBounds,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			err := testCase.list.set(testCase.args.idx, testCase.args.item)
			require.ErrorIs(t, err, testCase.wantErr)

			got, err := testCase.list.getByIndex(testCase.args.idx)
			require.ErrorIs(t, err, testCase.wantErr)
			require.Equal(t, testCase.args.item, got)
		})
	}
}

func Test_Partition_Shuffle(t *testing.T) {
	type (
		args struct {
			firstItemIdx          int
			firstPartitionSection int
			r                     *rand.Rand
			balances              state.StateContextI
		}

		caseData struct {
			name    string
			p       Partition
			args    args
			want    Partition
			wantErr error
		}
	)

	var (
		listLen    = 2
		list1      = mockValidatorsList(listLen)
		list2      = mockValidatorsList(listLen)
		partitions = []PartitionItemList{list1, list2}

		firstPartitionSection, firstItemIdx = 0, 0

		seed = time.Now().UnixNano()

		balances = mockStateContextI()
	)
	cases := []caseData{
		{
			name: "randomSelector_OK",
			p:    mockRandomSelector(partitions),
			args: args{
				firstItemIdx:          firstItemIdx,
				firstPartitionSection: firstPartitionSection,
				r:                     rand.New(rand.NewSource(seed)),
				balances:              balances,
			},
			want: func() Partition {
				var (
					r = rand.New(rand.NewSource(seed))

					list1Copy, list2Copy = copyValidatorsList(list1), copyValidatorsList(list2)

					secondPartitionIdx, secondItemIdx = r.Intn(len(partitions)), r.Intn(listLen)
				)

				rs := mockRandomSelector([]PartitionItemList{list1Copy, list2Copy})
				rs.NumPartitions = 2

				err := rs.swap(firstPartitionSection, firstItemIdx, secondPartitionIdx, secondItemIdx)
				require.NoError(t, err)

				return rs
			}(),
		},
		{
			name: "randomSelector_firstItemIdx_Negative_ERR",
			p:    mockRandomSelector(partitions),
			args: args{
				firstItemIdx:          -1,
				firstPartitionSection: firstPartitionSection,
				r:                     rand.New(rand.NewSource(seed)),
				balances:              balances,
			},
			wantErr: IndexOutOfBounds,
		},
		{
			name: "randomSelector_firstItemIdx_OutOfBounds_ERR",
			p:    mockRandomSelector(partitions),
			args: args{
				firstItemIdx:          firstItemIdx,
				firstPartitionSection: listLen,
				r:                     rand.New(rand.NewSource(seed)),
				balances:              balances,
			},
			wantErr: IndexOutOfBounds,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			args := testCase.args
			err := testCase.p.Shuffle(args.firstItemIdx, args.firstPartitionSection, args.r, args.balances)
			require.ErrorIs(t, err, testCase.wantErr)
			if err == nil {
				require.Equal(t, testCase.want, testCase.p)
			}
		})
	}
}

func mockValidatorsList(len int) *validatorItemList {
	items := make([]ValidationNode, len)
	for idx := range items {
		items[idx] = mockValidatorItem()
	}

	return &validatorItemList{
		Items:   items,
		Changed: true,
	}
}

func mockValidatorItem() ValidationNode {
	source := rand.NewSource(time.Now().UnixNano())
	r := rand.New(source)

	return ValidationNode{
		Id:  fmt.Sprintf("id-%d", r.Int63()),
		Url: fmt.Sprintf("url-%d", r.Int63()),
	}
}

func mockItemList(len int) *itemList {
	items := make([]StringItem, len)
	for idx := range items {
		items[idx] = mockStringItem()
	}

	return &itemList{
		Items:   items,
		Changed: true,
	}
}

func mockStringItem() StringItem {
	source := rand.NewSource(time.Now().UnixNano())
	r := rand.New(source)

	return StringItem{
		Item: fmt.Sprintf("item-%d", r.Int63()),
	}
}

func mockRandomSelector(partitions []PartitionItemList) *randomSelector {
	return &randomSelector{
		PartitionSize: 50,
		Partitions:    partitions,
	}
}

func mockStateContextI() state.StateContextI {
	type stateContext struct {
		state.StateContextI

		storage map[datastore.Key]util.MPTSerializable
	}

	stateContextMock := &mocks.StateContextI{}

	m := stateContext{
		StateContextI: stateContextMock,
		storage:       make(map[datastore.Key]util.MPTSerializable),
	}

	stateContextMock.On("InsertTrieNode", mock.Anything, mock.Anything).Return(
		func(key datastore.Key, data util.MPTSerializable) datastore.Key {
			m.storage[key] = data
			return key
		},
		func(key datastore.Key, data util.MPTSerializable) error {
			return nil
		},
	)

	return &m
}

func copyValidatorsList(list *validatorItemList) *validatorItemList {
	items := make([]ValidationNode, 0, len(list.Items))
	for _, item := range list.Items {
		items = append(items, item)
	}

	return &validatorItemList{
		Key:     list.Key,
		Items:   items,
		Changed: list.Changed,
	}
}
