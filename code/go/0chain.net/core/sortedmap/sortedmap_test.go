package sortedmap

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSortedMapGetValues(t *testing.T) {
	tt := []struct {
		name string
		m    map[string]int
		want []int
	}{
		{
			name: "ok",
			m: map[string]int{
				"a": 1,
				"b": 2,
				"c": 3,
			},
			want: []int{1, 2, 3},
		},
		{
			name: "ok 2",
			m: map[string]int{
				"1": 1,
				"3": 3,
				"2": 2,
			},
			want: []int{1, 2, 3},
		},
	}

	for _, tt := range tt {
		t.Run(tt.name, func(t *testing.T) {
			sm := New[string, int]()
			for k, v := range tt.m {
				sm.m[k] = v
			}

			vs := sm.GetValues()
			require.Equal(t, tt.want, vs)

			sm2 := NewFromMap(tt.m)
			vs2 := sm2.GetValues()
			require.Equal(t, tt.want, vs2)
		})
	}
}

func TestSortedMapGetKeys(t *testing.T) {
	tt := []struct {
		name string
		m    map[string]int
		want []string
	}{
		{
			name: "ok",
			m: map[string]int{
				"a": 1,
				"b": 2,
				"c": 3,
			},
			want: []string{"a", "b", "c"},
		},
		{
			name: "ok 2",
			m: map[string]int{
				"1": 1,
				"3": 3,
				"2": 2,
			},
			want: []string{"1", "2", "3"},
		},
	}

	for _, tt := range tt {
		t.Run(tt.name, func(t *testing.T) {
			sm := New[string, int]()
			for k, v := range tt.m {
				sm.m[k] = v
			}

			ks := sm.GetKeys()
			require.Equal(t, tt.want, ks)

			sm2 := NewFromMap(tt.m)
			ks2 := sm2.GetKeys()
			require.Equal(t, tt.want, ks2)

			require.Equal(t, len(tt.want), sm.Len())
		})
	}
}

func TestSortedMapPut(t *testing.T) {
	sm := New[string, int]()
	sm.Put("a", 1)
	sm.Put("b", 2)

	v1, ok := sm.Get("a")
	require.True(t, ok)
	require.Equal(t, 1, v1)

	v2, ok := sm.Get("b")
	require.True(t, ok)
	require.Equal(t, 2, v2)

	sm.Put("a", 3)
	v1, ok = sm.Get("a")
	require.True(t, ok)
	require.Equal(t, 3, v1)
}

func TestGetValues(t *testing.T) {
	tt := []struct {
		name string
		m    map[string]int
		want []int
	}{
		{
			name: "ok",
			m: map[string]int{
				"a": 1,
				"b": 2,
				"c": 3,
			},
			want: []int{1, 2, 3},
		},
		{
			name: "ok 2",
			m: map[string]int{
				"1": 1,
				"3": 3,
				"2": 2,
			},
			want: []int{1, 2, 3},
		},
	}

	for _, tt := range tt {
		t.Run(tt.name, func(t *testing.T) {
			vs := GetValues(tt.m)
			require.Equal(t, tt.want, vs)
		})
	}
}
