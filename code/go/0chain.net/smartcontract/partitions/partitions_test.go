package partitions

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/chain/state/mocks"
	"0chain.net/core/common"
	"github.com/0chain/common/core/util"
	"github.com/stretchr/testify/require"
)

type StringItem string

func (si StringItem) GetID() string {
	return string(si)
}

func (si *StringItem) MarshalMsg(o []byte) ([]byte, error) {
	return []byte(*si), nil
}

func (si *StringItem) UnmarshalMsg(b []byte) ([]byte, error) {
	*si = StringItem(b)
	return nil, nil
}

func (si *StringItem) Msgsize() int {
	return len(*si)
}

func ItemFromString(name string) PartitionItem {
	v := StringItem(name)
	return &v
}

type mockStateContextI struct {
	*mocks.StateContextI
	data map[string][]byte
}

func (m *mockStateContextI) GetTrieNode(key string, v util.MPTSerializable) error {
	d, ok := m.data[key]
	if !ok {
		return util.ErrValueNotPresent
	}

	_, err := v.UnmarshalMsg(d)
	return err
}

func (m *mockStateContextI) InsertTrieNode(key string, node util.MPTSerializable) (string, error) {
	d, err := node.MarshalMsg(nil)
	if err != nil {
		return "", err
	}
	m.data[key] = d
	return "", nil
}

func (m *mockStateContextI) DeleteTrieNode(key string) (string, error) {
	delete(m.data, key)
	return "", nil
}

type testItem struct {
	ID string
	V  string
}

func (ti *testItem) GetID() string {
	return ti.ID
}

func (ti *testItem) MarshalMsg(o []byte) ([]byte, error) {
	return json.Marshal(ti)
}

func (ti *testItem) UnmarshalMsg(b []byte) ([]byte, error) {
	return nil, json.Unmarshal(b, ti)
}

func (ti *testItem) Msgsize() int {
	d, _ := ti.MarshalMsg(nil)
	return len(d)
}

func TestPartitionsSave(t *testing.T) {
	balances := &mockStateContextI{data: make(map[string][]byte)}
	parts, err := newPartitions("test_rs", 10)
	require.NoError(t, err)

	for i := 0; i < 20; i++ {
		k := fmt.Sprintf("k%d", i)
		v := fmt.Sprintf("v%d", i)
		it := testItem{ID: k, V: v}
		_, err = parts.Add(balances, &it)
		require.NoError(t, err)
	}

	err = parts.Save(balances)
	require.NoError(t, err)

	p1, err := GetPartitions(balances, "test_rs")
	require.NoError(t, err)

	var v testItem
	err = p1.Get(balances, "k15", &v)
	require.NoError(t, err)
	require.Equal(t, "v15", v.V)

	p2, err := GetPartitions(balances, "test_rs")
	require.NoError(t, err)
	require.Equal(t, 2, p2.NumPartitions)

	// updateItem item
	err = p1.UpdateItem(balances, &testItem{"k10", "vv10"})
	require.NoError(t, err)
	require.NoError(t, p1.Save(balances))

	p3, err := GetPartitions(balances, "test_rs")
	require.NoError(t, err)
	require.Equal(t, 2, p3.NumPartitions)

	var vv testItem
	err = p3.Get(balances, "k10", &vv)
	require.NoError(t, err)
	require.Equal(t, "vv10", vv.V)
}

func TestPartitionsForeach(t *testing.T) {
	balances := &mockStateContextI{data: make(map[string][]byte)}
	parts, err := newPartitions("test_rs", 10)
	require.NoError(t, err)

	for i := 0; i < 20; i++ {
		k := fmt.Sprintf("k%d", i)
		v := fmt.Sprintf("v%d", i)
		it := testItem{ID: k, V: v}
		_, err = parts.Add(balances, &it)
		require.NoError(t, err)
	}

	err = parts.Save(balances)
	require.NoError(t, err)

	p1, err := GetPartitions(balances, "test_rs")
	require.NoError(t, err)

	err = p1.foreach(balances, func(key string, data []byte, _ int) ([]byte, bool, error) {
		if key == "k1" {
			n := testItem{}
			_, err := n.UnmarshalMsg(data)
			require.NoError(t, err)

			n.V = "new item"

			d, err := n.MarshalMsg(nil)
			require.NoError(t, err)

			return d, false, nil
		}

		return data, false, nil
	})
	require.NoError(t, err)

	err = p1.Save(balances)
	require.NoError(t, err)

	p2, err := GetPartitions(balances, "test_rs")
	require.NoError(t, err)
	vv := testItem{}
	err = p2.Get(balances, "k1", &vv)
	require.NoError(t, err)
	require.Equal(t, "new item", vv.V)
}

func TestPartitionsAdd(t *testing.T) {
	tt := []struct {
		name      string
		size      int
		num       int
		it        testItem
		expectErr error
		expectLoc int
	}{
		{
			name:      "add one item to empty partition",
			size:      10,
			num:       0,
			it:        testItem{ID: "k1", V: "v1"},
			expectLoc: 0,
			expectErr: nil,
		},
		{
			name:      "add one item to non-empty partition",
			size:      10,
			num:       1,
			it:        testItem{ID: "k1", V: "v1"},
			expectLoc: 0,
			expectErr: nil,
		},
		{
			name:      "add item - partition is full",
			size:      10,
			num:       10,
			it:        testItem{ID: "k11", V: "v11"},
			expectLoc: 1,
			expectErr: nil,
		},
		{
			name:      "add item - to second partition",
			size:      10,
			num:       11,
			it:        testItem{ID: "k12", V: "v12"},
			expectLoc: 1,
			expectErr: nil,
		},
		{
			name:      "item already exists",
			size:      10,
			num:       10,
			it:        testItem{ID: "k1", V: "v1"},
			expectErr: common.NewError(errItemExistCode, "k1"),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			partsName := "test_pa"
			s := prepareState(t, partsName, tc.size, tc.num)
			p, err := GetPartitions(s, partsName)
			require.NoError(t, err)

			loc, err := p.Add(s, &tc.it)
			require.Equal(t, tc.expectErr, err)
			if err != nil {
				return
			}

			require.Equal(t, tc.expectLoc, loc)
			err = p.Save(s)
			require.NoError(t, err)

			p, err = GetPartitions(s, partsName)
			require.NoError(t, err)

			var it testItem
			err = p.Get(s, tc.it.ID, &it)
			require.NoError(t, err)
			require.Equal(t, tc.it, it)
		})
	}
}

func TestPartitionsRemove(t *testing.T) {
	tt := []struct {
		name      string
		size      int
		num       int
		removeIdx int
		expectErr error
	}{
		{
			name:      "1 partition, 1 item, remove head",
			size:      10,
			num:       1,
			removeIdx: 0,
		},
		{
			name:      "1 partition, 10 item, remove head",
			size:      10,
			num:       10,
			removeIdx: 0,
		},
		{
			name:      "1 partition, 10 item, remove middle",
			size:      10,
			num:       10,
			removeIdx: 5,
		},
		{
			name:      "1 partition, 10 item, remove end",
			size:      10,
			num:       10,
			removeIdx: 9,
		},
		{
			name:      "1 partition, 5 item, remove end",
			size:      10,
			num:       5,
			removeIdx: 4,
		},
		{
			name:      "1 partition, not found",
			size:      10,
			num:       5,
			removeIdx: 5,
			expectErr: common.NewError(errItemNotFoundCode, fmt.Sprintf("k%d", 5)),
		},
		{
			name:      "1 partition, remove beyond partition size, not found",
			size:      10,
			num:       5,
			removeIdx: 15,
			expectErr: common.NewError(errItemNotFoundCode, fmt.Sprintf("k%d", 15)),
		},
		{
			name:      "2 partition, remove from 2, head",
			size:      10,
			num:       11,
			removeIdx: 10,
		},
		{
			name:      "2 partition, remove middle",
			size:      10,
			num:       20,
			removeIdx: 15,
		},
		{
			name:      "2 partition, remove from 2, end",
			size:      10,
			num:       20,
			removeIdx: 19,
		},
		{
			name:      "2 partition, remove from 1",
			size:      10,
			num:       20,
			removeIdx: 9,
		},
		{
			name:      "2 partition, remove from 1, cut 2 tail",
			size:      10,
			num:       11,
			removeIdx: 9,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			pn := "test_pr"
			balances := prepareState(t, pn, tc.size, tc.num)

			p, err := GetPartitions(balances, pn)
			require.NoError(t, err)
			k := fmt.Sprintf("k%d", tc.removeIdx)
			err = p.Remove(balances, k)
			require.Equal(t, tc.expectErr, err)
			if err != nil {
				return
			}

			// assert the item is removed before committing, i.e p.Save()
			verify := func() {
				var it testItem
				err = p.Get(balances, k, &it)
				require.Equal(t, common.NewError(errItemNotFoundCode, k), err)

				// all remaining items should exist
				for i := 0; i < tc.num; i++ {
					if i == tc.removeIdx {
						continue
					}

					it = testItem{}
					k := fmt.Sprintf("k%d", i)
					err = p.Get(balances, k, &it)
					require.NoError(t, err)
					require.Equal(t, &testItem{ID: k, V: fmt.Sprintf("v%d", i)}, &it)
				}
			}

			verify()

			// commit
			err = p.Save(balances)
			require.NoError(t, err)

			// verify after commit and reload
			p, err = GetPartitions(balances, pn)
			require.NoError(t, err)
			verify()
		})
	}
}

func TestPartitionsUpdateItem(t *testing.T) {
	tt := []struct {
		name      string
		size      int
		num       int
		update    testItem
		expectErr error
	}{
		{
			name:   "1 partition, update head",
			size:   10,
			num:    10,
			update: testItem{ID: "k0", V: "v10"},
		},
		{
			name:   "1 partition, update middle",
			size:   10,
			num:    10,
			update: testItem{ID: "k5", V: "v15"},
		},
		{
			name:   "1 partition, update end",
			size:   10,
			num:    10,
			update: testItem{ID: "k9", V: "v90"},
		},
		{
			name:   "2 partition, update 1 head",
			size:   10,
			num:    20,
			update: testItem{ID: "k0", V: "v10"},
		},
		{
			name:   "2 partition, update 1 middle",
			size:   10,
			num:    20,
			update: testItem{ID: "k5", V: "v15"},
		},
		{
			name:   "2 partition, update 1 end",
			size:   10,
			num:    20,
			update: testItem{ID: "k9", V: "v90"},
		},
		{
			name:   "2 partition, update 2 head",
			size:   10,
			num:    20,
			update: testItem{ID: "k10", V: "v100"},
		},
		{
			name:   "2 partition, update 2 middle",
			size:   10,
			num:    20,
			update: testItem{ID: "k15", V: "v150"},
		},
		{
			name:   "2 partition, update 2 end",
			size:   10,
			num:    20,
			update: testItem{ID: "k19", V: "v190"},
		},
		{
			name:   "2 partition, update 2 head, one item",
			size:   10,
			num:    11,
			update: testItem{ID: "k10", V: "v100"},
		},
		{
			name:      "item not found",
			size:      10,
			num:       10,
			update:    testItem{ID: "k100", V: "v100"},
			expectErr: common.NewError(errItemNotFoundCode, "k100"),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			pn := "test_pu"
			s := prepareState(t, pn, tc.size, tc.num)
			p, err := GetPartitions(s, pn)
			require.NoError(t, err)

			err = p.UpdateItem(s, &tc.update)
			require.Equal(t, tc.expectErr, err)
			if err != nil {
				return
			}

			verify := func() {
				var it testItem
				err = p.Get(s, tc.update.ID, &it)
				require.NoError(t, err)
				require.Equal(t, &tc.update, &it)
			}

			// verify before committing
			verify()

			// verify after committing
			err = p.Save(s)
			require.NoError(t, err)

			// verify after commit and reload
			p, err = GetPartitions(s, pn)
			require.NoError(t, err)
			verify()
		})
	}
}

func TestPartitionsUpdate(t *testing.T) {
	tt := []struct {
		name      string
		size      int
		num       int
		update    testItem
		expectErr error
	}{
		{
			name:   "1 partition, update head",
			size:   10,
			num:    10,
			update: testItem{ID: "k0", V: "v10"},
		},
		{
			name:   "1 partition, update middle",
			size:   10,
			num:    10,
			update: testItem{ID: "k5", V: "v15"},
		},
		{
			name:   "1 partition, update end",
			size:   10,
			num:    10,
			update: testItem{ID: "k9", V: "v90"},
		},
		{
			name:   "2 partition, update 1 head",
			size:   10,
			num:    20,
			update: testItem{ID: "k0", V: "v10"},
		},
		{
			name:   "2 partition, update 1 middle",
			size:   10,
			num:    20,
			update: testItem{ID: "k5", V: "v15"},
		},
		{
			name:   "2 partition, update 1 end",
			size:   10,
			num:    20,
			update: testItem{ID: "k9", V: "v90"},
		},
		{
			name:   "2 partition, update 2 head",
			size:   10,
			num:    20,
			update: testItem{ID: "k10", V: "v100"},
		},
		{
			name:   "2 partition, update 2 middle",
			size:   10,
			num:    20,
			update: testItem{ID: "k15", V: "v150"},
		},
		{
			name:   "2 partition, update 2 end",
			size:   10,
			num:    20,
			update: testItem{ID: "k19", V: "v190"},
		},
		{
			name:   "2 partition, update 2 head, one item",
			size:   10,
			num:    11,
			update: testItem{ID: "k10", V: "v100"},
		},
		{
			name:      "item not found",
			size:      10,
			num:       10,
			update:    testItem{ID: "k100", V: "v100"},
			expectErr: common.NewError(errItemNotFoundCode, "k100"),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			pn := "test_pu"
			s := prepareState(t, pn, tc.size, tc.num)
			p, err := GetPartitions(s, pn)
			require.NoError(t, err)

			err = p.Update(s, tc.update.ID, func(data []byte) ([]byte, error) {
				var it testItem
				_, err = it.UnmarshalMsg(data)
				require.NoError(t, err)

				it.V = tc.update.V
				return it.MarshalMsg(nil)
			})

			require.Equal(t, tc.expectErr, err)
			if err != nil {
				return
			}

			verify := func() {
				var it testItem
				err = p.Get(s, tc.update.ID, &it)
				require.NoError(t, err)
				require.Equal(t, &tc.update, &it)
			}

			// verify before committing
			verify()

			// verify after committing
			err = p.Save(s)
			require.NoError(t, err)

			// verify after commit and reload
			p, err = GetPartitions(s, pn)
			require.NoError(t, err)
			verify()
		})
	}
}

func TestPartitionSize(t *testing.T) {
	tt := []struct {
		name   string
		size   int
		num    int
		expect int
	}{
		{
			name:   "0",
			size:   10,
			num:    0,
			expect: 0,
		},
		{
			name:   "1",
			size:   10,
			num:    1,
			expect: 1,
		},
		{
			name:   "10",
			size:   10,
			num:    10,
			expect: 10,
		},
		{
			name:   "11",
			size:   10,
			num:    11,
			expect: 11,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			pn := "test_ps"
			s := prepareState(t, pn, tc.size, tc.num)

			p, err := GetPartitions(s, pn)
			require.NoError(t, err)
			l, err := p.Size(s)
			require.NoError(t, err)
			require.Equal(t, tc.expect, l)
		})
	}
}

func TestPartitionExist(t *testing.T) {
	tt := []struct {
		name   string
		size   int
		num    int
		checkK int
		expect bool
	}{
		{
			name:   "1 partition, exist, head",
			size:   10,
			num:    10,
			checkK: 0,
			expect: true,
		},
		{
			name:   "1 partition, exist, middle",
			size:   10,
			num:    10,
			checkK: 5,
			expect: true,
		},
		{
			name:   "1 partition, exist, end",
			size:   10,
			num:    10,
			checkK: 9,
			expect: true,
		},
		{
			name:   "1 partition, not exist",
			size:   10,
			num:    10,
			checkK: 10,
			expect: false,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			pn := "test_ps"
			s := prepareState(t, pn, tc.size, tc.num)
			p, err := GetPartitions(s, pn)
			require.NoError(t, err)
			find, err := p.Exist(s, fmt.Sprintf("k%d", tc.checkK))
			require.NoError(t, err)
			require.Equal(t, tc.expect, find)
		})
	}
}

func TestGetRandomItems(t *testing.T) {
	seed := int64(7777777)
	tt := []struct {
		name  string
		size  int
		num   int
		randN int
		err   error
	}{
		{
			name: "1 partition, num > size",
			size: 10,
			num:  10,
		},
		{
			name: "1 partition, num < size",
			size: 10,
			num:  5,
		},
		{
			name: "2 partition",
			size: 10,
			num:  20,
		},
		{
			name: "empty partitions",
			size: 10,
			num:  0,
			err:  errors.New("empty list, no items to return"),
		},
		{
			name: "2 partitions, fill from 1",
			size: 10,
			num:  15,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			pn := "test_ps"
			r := rand.New(rand.NewSource(seed))
			s := prepareState(t, pn, tc.size, tc.num)
			p, err := GetPartitions(s, pn)
			require.NoError(t, err)

			var its []testItem
			err = p.GetRandomItems(s, r, &its)
			require.Equal(t, tc.err, err)
			if err != nil {
				return
			}

			if tc.num > tc.size {
				require.Len(t, its, tc.size)
			} else {
				require.Len(t, its, tc.num)
			}

			for _, it := range its {
				var sit testItem
				err = p.Get(s, it.ID, &sit)
				require.NoError(t, err)
				require.Equal(t, sit, it)
			}
		})
	}
}

func FuzzAdd(f *testing.F) {
	rand.Seed(time.Now().UnixNano())
	f.Add(10)
	f.Fuzz(func(t *testing.T, n int) {
		if n <= 0 {
			return
		}

		n = n % 100

		var s state.StateContextI
		var num int
		var ks int
		partsName := "test_fr"
		if n == 0 {
			s = prepareState(t, partsName, 10, 0)
		} else {
			num = rand.Intn(n)
			s = prepareState(t, partsName, 10, num)
			ks = rand.Intn(n)
		}
		k := fmt.Sprintf("k%d", ks)

		p, err := GetPartitions(s, partsName)
		require.NoError(t, err)

		_, err = p.Add(s, &testItem{ID: k, V: fmt.Sprintf("v%d", ks)})
		if ks < num {
			// must already exist
			require.Equal(t, common.NewError(errItemExistCode, k), err)
		}

		err = p.Save(s)
		require.NoError(t, err)

		// reload partitions
		p, err = GetPartitions(s, partsName)

		var it testItem
		err = p.Get(s, k, &it)
		require.NoError(t, err)
		require.Equal(t, fmt.Sprintf("v%d", ks), it.V)
	})
}

func FuzzRemove(f *testing.F) {
	rand.Seed(time.Now().UnixNano())
	f.Add(10)
	f.Fuzz(func(t *testing.T, n int) {
		partsName := "test_fr"
		if n <= 0 {
			return
		}

		// limit the item number to 100
		n = n % 100

		var (
			s   state.StateContextI
			num int
			ks  int
		)

		if n == 0 {
			s = prepareState(t, partsName, 10, 0)
		} else {
			num = rand.Intn(n)
			s = prepareState(t, partsName, 10, num)
			ks = rand.Intn(n)
		}

		k := fmt.Sprintf("k%d", ks)

		p, err := GetPartitions(s, partsName)
		require.NoError(t, err)

		// empty partitions
		if n == 0 || num == 0 {
			require.Equal(t, 0, p.partitionsNum())
			err = p.Remove(s, k)
			require.Equal(t, common.NewError(errItemNotFoundCode, k), err)
			return
		}

		// remove item that does not exist in the partition
		if ks >= num {
			err = p.Remove(s, k)
			require.Equal(t, common.NewError(errItemNotFoundCode, k), err)
			return
		}

		// verify the last replaced item is moved or removed properly
		lastLoc := p.partitionsNum() - 1
		lastPart, err := p.getPartition(s, lastLoc)
		require.NoError(t, err)
		lastItem := lastPart.Items[len(lastPart.Items)-1]

		loc, ok, err := p.getItemPartIndex(s, k)
		require.NoError(t, err)
		require.True(t, ok, fmt.Sprintf("num: %d, n: %d, k: %s", num, n, k))

		err = p.Remove(s, k)
		require.NoError(t, err)

		err = p.Save(s)
		require.NoError(t, err)

		// reload partitions
		p, err = GetPartitions(s, partsName)

		_, ok, err = p.getItemPartIndex(s, k)
		require.NoError(t, err)
		require.False(t, ok)

		// if the item is not the last item in last part, then the last item must has been moved
		if lastLoc != loc {
			movedLoc, ok, err := p.getItemPartIndex(s, lastItem.ID)
			require.NoError(t, err)
			require.True(t, ok)
			require.Equal(t, movedLoc, loc)
		}
	})
}

func FuzzPartitionsAddRemove(f *testing.F) {
	rand.Seed(time.Now().UnixNano())
	f.Add(20, 10, 5)
	f.Fuzz(func(t *testing.T, initN, addN, removeN int) {
		var (
			partsName = "test_fr"
			maxNum    = 100
			maxAdd    = 200
		)
		if addN <= 0 {
			return
		}

		addN = addN % maxAdd

		if removeN < 0 {
			return
		}
		removeN = removeN % maxNum

		if initN < 0 {
			return
		}

		var (
			s        state.StateContextI
			itemsMap = make(map[string]struct{})
		)

		if initN == 0 {
			s = prepareState(t, partsName, 10, 0)
		} else {
			// init state with randN size, and randN number of items
			size := rand.Intn(initN)
			num := rand.Intn(initN)
			s = prepareState(t, partsName, size, num)
			for i := 0; i < num; i++ {
				itemsMap[fmt.Sprintf("k%d", i)] = struct{}{}
			}
			t.Logf("init size: %d", num)
		}

		p, err := GetPartitions(s, partsName)
		require.NoError(t, err)

		addFunc := func() {
			ks := rand.Intn(addN)
			k := fmt.Sprintf("k%d", ks)
			_, ok := itemsMap[k]
			_, err = p.Add(s, &testItem{ID: k, V: fmt.Sprintf("v%d", ks)})
			if !ok {
				itemsMap[k] = struct{}{}
				require.NoError(t, err)
			} else {
				require.Equal(t, common.NewError(errItemExistCode, k), err)
			}
		}

		for i := 0; i < addN; i++ {
			addFunc()
		}

		err = p.Save(s)
		require.NoError(t, err)

		// remove items
		var removed []string
		removeFunc := func() {
			ks := rand.Intn(removeN)
			k := fmt.Sprintf("k%d", ks)

			_, ok := itemsMap[k]
			err = p.Remove(s, k)
			if !ok {
				require.Equal(t, common.NewError(errItemNotFoundCode, k), err, p.locations)
			} else {
				// remove item not exist
				delete(itemsMap, k)
				require.NoError(t, err)
				removed = append(removed, k)
			}
		}
		p, err = GetPartitions(s, partsName)
		require.NoError(t, err)

		for i := 0; i < removeN; i++ {
			removeFunc()
		}

		err = p.Save(s)
		require.NoError(t, err)
	})
}

func FuzzPartitionsUpdateItem(f *testing.F) {
	rand.Seed(time.Now().UnixNano())
	f.Add(10, 5)
	f.Fuzz(func(t *testing.T, initN, updateK int) {
		if initN < 0 {
			return
		}
		if updateK < 0 {
			return
		}

		var (
			pn     = "test_pu"
			maxNum = 100
			s      state.StateContextI
			num    int
		)

		initN = initN % maxNum
		if initN == 0 {
			s = prepareState(t, pn, 10, 0)
		} else {
			// init state with randN size, and randN number of items
			size := rand.Intn(initN)
			num = rand.Intn(initN)
			s = prepareState(t, pn, size, num)
		}

		p, err := GetPartitions(s, pn)
		require.NoError(t, err)

		k := fmt.Sprintf("k%d", updateK)
		err = p.UpdateItem(s, &testItem{ID: k, V: fmt.Sprintf("v%d", updateK+100)})
		if updateK < num {
			require.NoError(t, err)
			// verify the item is updated
			verify := func() {
				var it testItem
				err = p.Get(s, k, &it)
				require.NoError(t, err)
				require.Equal(t, fmt.Sprintf("v%d", updateK+100), it.V)
			}

			verify()
			// verify after commit
			err = p.Save(s)
			require.NoError(t, err)
			verify()
		} else {
			// item not exist
			require.Equal(t, common.NewError(errItemNotFoundCode, k), err)
		}
	})
}

func FuzzPartitionsUpdate(f *testing.F) {
	rand.Seed(time.Now().UnixNano())
	f.Add(10, 5)
	f.Fuzz(func(t *testing.T, initN, updateK int) {
		if initN < 0 {
			return
		}
		if updateK < 0 {
			return
		}

		var (
			pn     = "test_pu"
			maxNum = 100
			s      state.StateContextI
			num    int
		)

		initN = initN % maxNum
		if initN == 0 {
			s = prepareState(t, pn, 10, 0)
		} else {
			// init state with randN size, and randN number of items
			size := rand.Intn(initN)
			num = rand.Intn(initN)
			s = prepareState(t, pn, size, num)
		}

		p, err := GetPartitions(s, pn)
		require.NoError(t, err)

		k := fmt.Sprintf("k%d", updateK)
		err = p.Update(s, k, func(data []byte) ([]byte, error) {
			var it testItem
			_, err = it.UnmarshalMsg(data)
			require.NoError(t, err)
			it.V = fmt.Sprintf("v%d", updateK+100)
			return it.MarshalMsg(nil)
		})
		if updateK < num {
			require.NoError(t, err)
			// verify the item is updated
			verify := func() {
				var it testItem
				err = p.Get(s, k, &it)
				require.NoError(t, err)
				require.Equal(t, fmt.Sprintf("v%d", updateK+100), it.V)
			}

			verify()
			// verify after commit
			err = p.Save(s)
			require.NoError(t, err)
			verify()
		} else {
			// item not exist
			require.Equal(t, common.NewError(errItemNotFoundCode, k), err)
		}
	})
}

func FuzzPartitionsGetRandomItems(f *testing.F) {
	rand.Seed(time.Now().UnixNano())
	f.Add(10, 5)
	f.Fuzz(func(t *testing.T, initN, size int) {
		if initN < 0 {
			return
		}

		if size <= 0 {
			return
		}

		var (
			pn     = "test_get_rand_items"
			maxNum = 100
			s      state.StateContextI
		)

		initN = initN % maxNum
		s = prepareState(t, pn, size, initN)

		p, err := GetPartitions(s, pn)
		require.NoError(t, err)

		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		var its []testItem
		err = p.GetRandomItems(s, r, &its)

		if initN == 0 {
			require.Equal(t, errors.New("empty list, no items to return"), err)
		} else {
			require.NoError(t, err)

			for _, it := range its {
				var sit testItem
				err = p.Get(s, it.ID, &sit)
				require.NoError(t, err)
				require.Equal(t, it, sit)
			}
		}
	})
}

func prepareState(t *testing.T, name string, size, num int) state.StateContextI {
	s := &mockStateContextI{data: make(map[string][]byte)}
	parts, err := newPartitions(name, size)
	require.NoError(t, err)

	for i := 0; i < num; i++ {
		k := fmt.Sprintf("k%d", i)
		v := fmt.Sprintf("v%d", i)
		it := testItem{ID: k, V: v}
		_, err = parts.Add(s, &it)
		require.NoError(t, err)
	}

	err = parts.Save(s)
	require.NoError(t, err)
	return s
}
