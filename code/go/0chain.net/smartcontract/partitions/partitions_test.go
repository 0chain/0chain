package partitions

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPartitionsSave(t *testing.T) {
	balances := &mockStateContextI{data: make(map[string][]byte)}
	rs, err := newRandomSelector("test_rs", 10, nil)
	require.NoError(t, err)

	parts := Partitions{rs: rs}

	for i := 0; i < 20; i++ {
		k := fmt.Sprintf("k%d", i)
		v := fmt.Sprintf("v%d", i)
		it := testItem{ID: k, V: v}
		_, err = parts.AddItem(balances, &it)
		require.NoError(t, err)
	}

	err = parts.Save(balances)
	require.NoError(t, err)

	p1, err := GetPartitions(balances, "test_rs")
	require.NoError(t, err)

	var v testItem
	_, err = p1.GetItem(balances, "k15", &v)
	require.NoError(t, err)
	require.Equal(t, "v15", v.V)

	p2, err := GetPartitions(balances, "test_rs")
	require.NoError(t, err)
	require.Equal(t, 2, p2.rs.NumPartitions)

	// update item
	err = p1.UpdateItem(balances, &testItem{"k10", "vv10"})
	require.NoError(t, err)
	require.NoError(t, p1.Save(balances))

	p3, err := GetPartitions(balances, "test_rs")
	require.NoError(t, err)
	require.Equal(t, 2, p3.rs.NumPartitions)

	var vv testItem
	_, err = p3.GetItem(balances, "k10", &vv)
	require.NoError(t, err)
	require.Equal(t, "vv10", vv.V)
}

func Test_randomSelector_UpdateRandomItems(t *testing.T) {
	balances := &mockStateContextI{data: make(map[string][]byte)}
	rs, err := CreateIfNotExists(balances, "test_rs", 10)

	items := make([]testItem, 0, 20)
	for i := 0; i < 15; i++ {
		k := fmt.Sprintf("k%d", i)
		v := fmt.Sprintf("v%d", i)
		it := testItem{ID: k, V: v}
		items = append(items, it)
		_, err = rs.AddItem(balances, &it)
		require.NoError(t, err)
	}

	err = rs.Save(balances)
	require.NoError(t, err)

	rs2, err := GetPartitions(balances, "test_rs")
	require.NoError(t, err)

	r := rand.New(rand.NewSource(time.Now().Unix()))
	var ids []string
	err = rs2.UpdateRandomItems(balances, r, 8, func(key string, data []byte) ([]byte, error) {
		ti := testItem{}
		_, err := ti.UnmarshalMsg(data)
		require.NoError(t, err)

		fmt.Println(ti)

		ti.V = ti.V + ":new added"
		ids = append(ids, ti.ID)
		return ti.MarshalMsg(nil)
	})

	require.NoError(t, err)

	err = rs2.Save(balances)
	require.NoError(t, err)

	rs3, err := GetPartitions(balances, "test_rs")
	require.NoError(t, err)

	for _, id := range ids {
		var item testItem
		_, err = rs3.GetItem(balances, id, &item)
		require.NoError(t, err)
		fmt.Println(item)
	}
}

func TestPartitionsForeach(t *testing.T) {
	balances := &mockStateContextI{data: make(map[string][]byte)}
	rs, err := newRandomSelector("test_rs", 10, nil)
	require.NoError(t, err)

	parts := Partitions{rs: rs}

	for i := 0; i < 20; i++ {
		k := fmt.Sprintf("k%d", i)
		v := fmt.Sprintf("v%d", i)
		it := testItem{ID: k, V: v}
		_, err = parts.AddItem(balances, &it)
		require.NoError(t, err)
	}

	err = parts.Save(balances)
	require.NoError(t, err)

	p1, err := GetPartitions(balances, "test_rs")
	require.NoError(t, err)

	err = p1.Foreach(balances, func(key string, data []byte, _ int) ([]byte, bool, error) {
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
	_, err = p2.GetItem(balances, "k1", &vv)
	require.NoError(t, err)
	require.Equal(t, "new item", vv.V)

	//var v testItem
	//_, err = p1.GetItem(balances, "k15", &v)
	//require.NoError(t, err)
	//require.Equal(t, "v15", v.V)
	//
	//p2, err := GetPartitions(balances, "test_rs")
	//require.NoError(t, err)
	//require.Equal(t, 2, p2.rs.NumPartitions)
	//
	//// update item
	//err = p1.UpdateItem(balances, &testItem{"k10", "vv10"})
	//require.NoError(t, err)
	//require.NoError(t, p1.Save(balances))
	//
	//p3, err := GetPartitions(balances, "test_rs")
	//require.NoError(t, err)
	//require.Equal(t, 2, p3.rs.NumPartitions)
	//
	//var vv testItem
	//_, err = p3.GetItem(balances, "k10", &vv)
	//require.NoError(t, err)
	//require.Equal(t, "vv10", vv.V)
}
