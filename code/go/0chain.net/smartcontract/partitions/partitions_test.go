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
