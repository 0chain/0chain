// orderbuffer package contains a buffer that stores items in order of their round number.
// The buffer has a maximum limit, and when the buffer exceeds the maximum limit,
// the oldest items are removed from the buffer.
//
// The buffer is implemented using a slice of Item type,
// and the Add method adds an item to the buffer.
// The First method returns the first item in the buffer,
// and the Pop method removes and returns the first item in the buffer.
package orderbuffer

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOrderBuffer(t *testing.T) {
	ob := New(10)

	// Test Add method
	items := []Item{
		{Round: 2, Data: "item2"},
		{Round: 1, Data: "item1"},
		{Round: 3, Data: "item3"},
	}
	for _, item := range items {
		ob.Add(item.Round, item.Data)
	}

	// Test First method
	first, ok := ob.First()
	if !ok || first.Round != 1 || first.Data != "item1" {
		t.Errorf("Expected {Round: 1, Data: item1}, got {%d, %s}", first.Round, first.Data)
	}

	// Test Pop method
	for i := 1; i <= 3; i++ {
		item, ok := ob.Pop()
		if !ok || item.Round != int64(i) || item.Data != "item"+strconv.Itoa(i) {
			t.Errorf("Expected {Round: %d, Data: item%d}, got {%d, %s}", i, i, item.Round, item.Data)
		}
	}

	// Test Pop method when buffer is empty
	_, ok = ob.Pop()
	if ok {
		t.Error("Expected false, got ", ok)
	}

}

func TestMaxLimit(t *testing.T) {
	// Test max limit
	maxItems := []Item{
		{Round: 11, Data: "item11"},
		{Round: 13, Data: "item13"},
		{Round: 17, Data: "item17"},
		{Round: 12, Data: "item12"},
		{Round: 19, Data: "item19"},
		{Round: 14, Data: "item14"},
		{Round: 15, Data: "item15"},
		{Round: 20, Data: "item20"},
		{Round: 18, Data: "item18"},
		{Round: 21, Data: "item21"},
		{Round: 16, Data: "item16"},
	}

	ob := New(10)
	for _, item := range maxItems {
		ob.Add(item.Round, item.Data)
	}

	// Test Pop method after reaching max limit
	for i := 11; i <= 20; i++ {
		item, ok := ob.Pop()
		if !ok || item.Round != int64(i) || item.Data != "item"+strconv.Itoa(i) {
			t.Errorf("Expected {Round: %d, Data: item%d}, got {%d, %s}", i, i, item.Round, item.Data)
		}
	}

	// Test Pop method when buffer is empty after reaching max limit
	_, ok := ob.Pop()
	if ok {
		t.Error("Expected false, got ", ok)
	}
}

func TestAdd(t *testing.T) {
	tests := []struct {
		name   string
		items  []Item
		expect []Item
	}{
		{
			name: "Add 1 item",
			items: []Item{
				{Round: 1, Data: "item1"},
			},
			expect: []Item{
				{Round: 1, Data: "item1"},
			},
		},
		{
			name: "Add 2 items",
			items: []Item{
				{Round: 1, Data: "item1"},
				{Round: 2, Data: "item2"},
			},
			expect: []Item{
				{Round: 1, Data: "item1"},
				{Round: 2, Data: "item2"},
			},
		},
		{
			name: "Add 3 items",
			items: []Item{
				{Round: 1, Data: "item1"},
				{Round: 2, Data: "item2"},
				{Round: 3, Data: "item3"},
			},
			expect: []Item{
				{Round: 1, Data: "item1"},
				{Round: 2, Data: "item2"},
				{Round: 3, Data: "item3"},
			},
		},
		{
			name: "Add random items",
			items: []Item{
				{Round: 5, Data: "item5"},
				{Round: 2, Data: "item2"},
				{Round: 7, Data: "item7"},
				{Round: 3, Data: "item3"},
			},
			expect: []Item{
				{Round: 2, Data: "item2"},
				{Round: 3, Data: "item3"},
				{Round: 5, Data: "item5"},
				{Round: 7, Data: "item7"},
			},
		},
		{
			name: "Add sequence items",
			items: []Item{
				{Round: 1, Data: "item1"},
				{Round: 2, Data: "item2"},
				{Round: 3, Data: "item3"},
				{Round: 4, Data: "item4"},
				{Round: 5, Data: "item5"},
			},
			expect: []Item{
				{Round: 1, Data: "item1"},
				{Round: 2, Data: "item2"},
				{Round: 3, Data: "item3"},
				{Round: 4, Data: "item4"},
				{Round: 5, Data: "item5"},
			},
		},
		{
			name: "Add item with same round",
			items: []Item{
				{Round: 1, Data: "item1"},
				{Round: 2, Data: "item2"},
				{Round: 1, Data: "item12"},
			},
			expect: []Item{
				{Round: 1, Data: "item1"},
				{Round: 1, Data: "item12"},
				{Round: 2, Data: "item2"},
			},
		},
		{
			name: "Add item with same round and same data",
			items: []Item{
				{Round: 1, Data: "item1"},
				{Round: 2, Data: "item2"},
				{Round: 1, Data: "item1"},
			},
			expect: []Item{
				{Round: 1, Data: "item1"},
				{Round: 2, Data: "item2"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ob := New(10)

			for _, item := range tt.items {
				ob.Add(item.Round, item.Data)
			}

			require.Equal(t, tt.expect, ob.Buffer)
		})
	}
}
