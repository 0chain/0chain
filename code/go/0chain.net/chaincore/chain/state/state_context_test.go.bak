package state

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"0chain.net/core/datastore"
	"0chain.net/core/encryption"

	"github.com/0chain/common/core/logging"
	"github.com/0chain/common/core/statecache"
	"github.com/0chain/common/core/util"
	"github.com/stretchr/testify/require"
)

func init() {
	logging.InitLogging("development", "")
}

func TestGetItemsByIDs(t *testing.T) {
	type testItem struct {
		ID    string
		Value string
	}

	items := make(map[string]*testItem, 10)
	for i := 1; i <= 10; i++ {
		id := fmt.Sprintf("t%d", i)
		items[id] = &testItem{
			ID:    id,
			Value: fmt.Sprintf("v%d", i),
		}
	}

	type args struct {
		ids      []string
		getItem  GetItemFunc[*testItem]
		balances StateContextI
	}
	tests := []struct {
		name string
		args args
		want []*testItem
		err  error
	}{
		{
			name: "get one item",
			args: args{
				ids: []string{"t1"},
				getItem: func(id string, _ StateContextI) (*testItem, error) {
					return items["t1"], nil
				},
			},
			want: []*testItem{
				{
					ID:    "t1",
					Value: "v1",
				},
			},
		},
		{
			name: "get 5 item",
			args: args{
				ids: []string{"t1", "t2", "t3", "t4", "t5"},
				getItem: func(id string, _ StateContextI) (*testItem, error) {
					return items[id], nil
				},
			},
			want: []*testItem{
				{
					ID:    "t1",
					Value: "v1",
				},
				{
					ID:    "t2",
					Value: "v2",
				},
				{
					ID:    "t3",
					Value: "v3",
				},
				{
					ID:    "t4",
					Value: "v4",
				},
				{
					ID:    "t5",
					Value: "v5",
				},
			},
		},
		{
			name: "get node not found error",
			args: args{
				ids: []string{"t1", "t2", "t3", "t4", "t5"},
				getItem: func(id string, _ StateContextI) (*testItem, error) {
					if id == "t2" {
						return nil, util.ErrNodeNotFound
					}
					return items[id], nil
				},
			},
			err: util.ErrNodeNotFound,
		},
		{
			name: "get node not found and value not present errors",
			args: args{
				ids: []string{"t1", "t2", "t3", "t4", "t5"},
				getItem: func(id string, _ StateContextI) (*testItem, error) {
					if id == "t2" {
						return nil, util.ErrNodeNotFound
					}

					if id == "t1" {
						return nil, util.ErrValueNotPresent
					}

					return items[id], nil
				},
			},
			err: util.ErrNodeNotFound,
		},
		{
			name: "get value not present error",
			args: args{
				ids: []string{"t1", "t2", "t3", "t4", "t5"},
				getItem: func(id string, _ StateContextI) (*testItem, error) {
					if id == "t1" {
						return nil, util.ErrValueNotPresent
					}

					return items[id], nil
				},
			},
			err: fmt.Errorf("could not get item %q: %v", "t1", util.ErrValueNotPresent),
		},
		{
			name: "get multiple value not present errors",
			args: args{
				ids: []string{"t1", "t2", "t3", "t4", "t5"},
				getItem: func(id string, _ StateContextI) (*testItem, error) {
					if id == "t1" {
						return nil, util.ErrValueNotPresent
					}

					if id == "t5" {
						return nil, util.ErrValueNotPresent
					}

					return items[id], nil
				},
			},
			err: fmt.Errorf("could not get item %q: %v", "t1", util.ErrValueNotPresent),
		},
		{
			name: "return nil item without ErrValueNotPresent",
			args: args{
				ids: []string{"t1"},
				getItem: func(id string, _ StateContextI) (*testItem, error) {
					return nil, nil
				},
			},
			err: fmt.Errorf("could not get item %q: %v", "t1", errors.New("nil item returned without ErrValueNotPresent")),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := GetItemsByIDs(tc.args.ids, tc.args.getItem, tc.args.balances)
			require.Equal(t, tc.err, err)
			if err != nil {
				return
			}

			require.Equal(t, tc.want, got)
		})
	}
}

func TestWrongGetItemFunc(t *testing.T) {
	// NOTE: uncomment the following code to see compiler error

	//type testItem struct {
	//	ID    string
	//	Value string
	//}
	//ids := []string{"t1"}
	//_, err := GetItemsByIDs(ids, func(id string, _ CommonStateContextI) (testItem, error) {
	//	return testItem{}, nil
	//}, nil)
	//require.NoError(t, err)
}

type testCacheValue struct {
	Value           string
	isMarshalCalled bool
}

// MarshalMsg implements util.MPTSerializable.
func (v *testCacheValue) MarshalMsg([]byte) ([]byte, error) {
	v.isMarshalCalled = true
	return []byte(v.Value), nil
}

// UnmarshalMsg implements util.MPTSerializable.
func (v *testCacheValue) UnmarshalMsg(d []byte) ([]byte, error) {
	v.isMarshalCalled = true
	v.Value = string(d)
	return d, nil
}

func (v *testCacheValue) Clone() statecache.Value {
	return &testCacheValue{
		Value: v.Value,
	}
}

func (v *testCacheValue) CopyFrom(src interface{}) bool {
	if reflect.TypeOf(src) != reflect.TypeOf(v) {
		return false
	}

	v.Value = src.(*testCacheValue).Value
	return true
}

type testCacheValueNotCopyable struct {
	Value           string
	isMarshalCalled bool
}

// MarshalMsg implements util.MPTSerializable.
func (v *testCacheValueNotCopyable) MarshalMsg([]byte) ([]byte, error) {
	v.isMarshalCalled = true
	return []byte(v.Value), nil
}

// UnmarshalMsg implements util.MPTSerializable.
func (v *testCacheValueNotCopyable) UnmarshalMsg(d []byte) ([]byte, error) {
	v.isMarshalCalled = true
	v.Value = string(d)
	return d, nil
}

func (v *testCacheValueNotCopyable) Clone() statecache.Value {
	return &testCacheValueNotCopyable{
		Value: v.Value,
	}
}

func (v *testCacheValueNotCopyable) CopyFrom(src interface{}) bool {
	return false
}

type mockStateContext struct {
	*StateContext
}

func (msc *mockStateContext) getNodeValue(key datastore.Key, v util.MPTSerializable) error {
	tv, ok := v.(*testCacheValue)
	if !ok {
		return errors.New("unexpected type")
	}
	tv.Value = "mptValue3"
	return nil
}

func TestGetTrieNode(t *testing.T) {
	sc := &mockStateContext{
		StateContext: &StateContext{
			state: util.NewMerklePatriciaTrie(util.NewMemoryNodeDB(), 0, nil, statecache.NewEmpty()),
		},
	}

	t.Run("cache hit - copyable", func(t *testing.T) {
		key := "key1"
		cacheValue := &testCacheValue{Value: "cacheValue1"}
		sc.Cache().Set(key, cacheValue)

		var vv testCacheValue
		err := sc.GetTrieNode(key, &vv)
		require.NoError(t, err)
		require.Equal(t, cacheValue.Value, vv.Value)
		require.False(t, vv.isMarshalCalled)
	})

	t.Run("cache hit - not copyable", func(t *testing.T) {
		key := "key2"
		cacheValue := &testCacheValueNotCopyable{Value: "cacheValue2"}
		sc.Cache().Set(key, cacheValue)

		var v testCacheValueNotCopyable
		require.Panics(t, func() {
			sc.GetTrieNode(key, &v)
		}, "should panic")
	})

	t.Run("cache miss", func(t *testing.T) {
		key := "key3"
		mptValue := &testCacheValue{Value: "mptValue3"}
		// insert the value to MPT
		_, err := sc.state.Insert(util.Path(encryption.Hash(key)), &testCacheValue{Value: "mptValue3"})
		require.NoError(t, err)

		// verify that the value is not cached
		_, ok := sc.Cache().Get(key)
		require.False(t, ok)

		var v testCacheValue
		err = sc.GetTrieNode(key, &v)
		require.NoError(t, err)
		require.Equal(t, mptValue.Value, v.Value)
		require.True(t, v.isMarshalCalled)

		// Verify that the value is cached
		cachedValue, ok := sc.Cache().Get(key)
		require.True(t, ok)
		require.Equal(t, mptValue, cachedValue)
	})
}
