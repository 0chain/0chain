package partitions

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"
	"testing"

	"0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/chain/state/mocks"
	"github.com/0chain/common/core/util"
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
		id string,
		item []byte,
		from, to int,
		_ state.StateContextI,
	) error {
		for i := 0; i < len(items); i++ {
			if items[i].item == id {
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
			partition, err := rs.AddRand(balances, action.item, r)
			require.NoError(t, err, fmt.Sprintf("action AddItem: %v, error: %v", action, err))
			items = append(items, fuzzyItem{
				item:     action.item.GetID(),
				division: partition,
			})

		case Remove:
			err := rs.RemoveItem(balances, action.item.GetID(), action.divisionId)
			require.NoError(t, err, fmt.Sprintf("action Remove: %v, error: %v", action, err))
			for index, fuzzyItem := range items {
				if fuzzyItem.item == action.item.GetID() {
					items[index] = items[len(items)-1]
					items = items[:len(items)-1]
					break
				}
			}
		case GetRandomPartition:
			var strItems []StringItem
			err := rs.GetRandomItems(balances, r, &strItems)
			require.NoError(t, err, fmt.Sprintf("action Change: %v, error: %v", action, err))
			require.True(t, len(strItems) <= rs.PartitionSize)
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

func Test_randomSelector_Save(t *testing.T) {
	balances := &mockStateContextI{data: make(map[string][]byte)}
	rs, err := newRandomSelector("test_rs", 10, nil)

	for i := 0; i < 20; i++ {
		k := fmt.Sprintf("k%d", i)
		v := fmt.Sprintf("v%d", i)
		it := testItem{ID: k, V: v}
		_, err = rs.Add(balances, &it)
		require.NoError(t, err)
	}

	err = rs.Save(balances)
	require.NoError(t, err)

	var loadRs randomSelector
	err = balances.GetTrieNode("test_rs", &loadRs)
	require.NoError(t, err)
	var v testItem
	err = loadRs.GetItem(balances, 1, "k15", &v)
	require.NoError(t, err)
	require.Equal(t, "v15", v.V)

	err = loadRs.Save(balances)
	require.NoError(t, err)

	var loadRs2 randomSelector
	err = balances.GetTrieNode("test_rs", &loadRs2)
	require.NoError(t, err)
	require.Equal(t, 2, loadRs2.NumPartitions)

	// update item
	err = loadRs.UpdateItem(balances, 1, &testItem{"k10", "vv10"})
	require.NoError(t, err)
	require.NoError(t, loadRs.Save(balances))

	var loadRs3 randomSelector
	err = balances.GetTrieNode("test_rs", &loadRs3)
	require.NoError(t, err)
	require.Equal(t, 2, loadRs3.NumPartitions)

	var vv testItem
	err = loadRs3.GetItem(balances, 1, "k10", &vv)
	require.NoError(t, err)
	require.Equal(t, "vv10", vv.V)
}

func Test_randomSelector_Foreach(t *testing.T) {
	balances := &mockStateContextI{data: make(map[string][]byte)}
	rs, err := newRandomSelector("test_rs", 10, nil)

	items := make([]testItem, 0, 20)
	for i := 0; i < 20; i++ {
		k := fmt.Sprintf("k%d", i)
		v := fmt.Sprintf("v%d", i)
		it := testItem{ID: k, V: v}
		items = append(items, it)
		_, err = rs.Add(balances, &it)
		require.NoError(t, err)
	}

	err = rs.Save(balances)
	require.NoError(t, err)

	var loadRs randomSelector
	err = balances.GetTrieNode("test_rs", &loadRs)
	require.NoError(t, err)

	retItems := make([]testItem, 0, 20)
	err = loadRs.foreach(balances, func(id string, data []byte) error {
		var it testItem
		_, err := it.UnmarshalMsg(data)
		if err != nil {
			return err
		}
		retItems = append(retItems, it)
		return nil
	})

	require.NoError(t, err)
	require.Equal(t, items, retItems)
}
