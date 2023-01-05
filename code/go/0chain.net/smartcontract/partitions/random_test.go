package partitions

import (
	"encoding/json"
	"fmt"
	"testing"

	"0chain.net/chaincore/chain/state/mocks"
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
	rs, err := newRandomSelector("test_rs", 10)

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
	rs, err := newRandomSelector("test_rs", 10)

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
	err = loadRs.foreach(balances, func(id string, data []byte, idx int) ([]byte, bool, error) {
		var it testItem
		_, err := it.UnmarshalMsg(data)
		if err != nil {
			return nil, false, err
		}
		retItems = append(retItems, it)
		return data, false, nil
	})

	require.NoError(t, err)
	require.Equal(t, items, retItems)
}
