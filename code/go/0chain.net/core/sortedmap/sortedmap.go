package sortedmap

import (
	"sort"

	"golang.org/x/exp/constraints"
)

// Map is a map that provides methods to return sorted values or keys
type Map[K constraints.Ordered, V any] struct {
	m map[K]V
}

func New[K constraints.Ordered, V any]() *Map[K, V] {
	return &Map[K, V]{
		m: map[K]V{},
	}
}

func NewFromMap[K constraints.Ordered, V any](m map[K]V) *Map[K, V] {
	sm := &Map[K, V]{
		m: make(map[K]V, len(m)),
	}

	for k, v := range m {
		sm.m[k] = v
	}

	return sm
}

func (sm *Map[K, V]) Put(key K, value V) {
	sm.m[key] = value
}

func (sm *Map[K, V]) Get(key K) (V, bool) {
	v, ok := sm.m[key]
	return v, ok
}

func (sm *Map[K, V]) Len() int {
	return len(sm.m)
}

func (sm *Map[K, V]) GetKeys() []K {
	keys := make([]K, 0, len(sm.m))

	for k := range sm.m {
		keys = append(keys, k)
	}

	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	return keys
}

func (sm *Map[K, V]) GetValues() []V {
	keys := make([]K, 0, len(sm.m))
	for k := range sm.m {
		keys = append(keys, k)
	}

	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	vs := make([]V, len(keys))
	for i, k := range keys {
		vs[i] = sm.m[k]
	}
	return vs
}

func GetValues[K constraints.Ordered, V any](m map[K]V) []V {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	vs := make([]V, len(keys))
	for i, k := range keys {
		vs[i] = m[k]
	}
	return vs
}
