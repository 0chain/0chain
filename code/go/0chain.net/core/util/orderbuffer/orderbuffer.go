package orderbuffer

import (
	"sync"
)

// Item represents a data item with a round number
type Item struct {
	Round int64
	Data  interface{}
}

// OrderBuffer
type OrderBuffer struct {
	mu     sync.Mutex
	max    int
	Buffer []Item
}

func New(max int) *OrderBuffer {
	return &OrderBuffer{
		max:    max,
		Buffer: make([]Item, 0, 100),
	}
}

// Add adds an item to the buffer
func (rb *OrderBuffer) Add(round int64, data interface{}) bool {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	item := Item{Round: round, Data: data}

	index := rb.search(round)
	if len(rb.Buffer) > 0 {
		if index > 0 {
			if rb.Buffer[index-1].Data == item.Data {
				// for item has exact the same round number and data
				return true
			}
		}
	}

	rb.Buffer = append(rb.Buffer, Item{})
	copy(rb.Buffer[index+1:], rb.Buffer[index:])
	rb.Buffer[index] = item

	// Check if the buffer exceeds the maximum limit
	if len(rb.Buffer) > rb.max {
		rb.Buffer = rb.Buffer[:rb.max]
	}

	return true
}

func (rb *OrderBuffer) search(roundNumber int64) int {
	left, right := 0, len(rb.Buffer)
	for left < right {
		middle := (left + right) / 2
		if rb.Buffer[middle].Round <= roundNumber {
			left = middle + 1
		} else {
			right = middle
		}
	}
	return left
}

func (rb *OrderBuffer) First() (Item, bool) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if len(rb.Buffer) == 0 {
		return Item{}, false
	}
	return rb.Buffer[0], true
}

func (rb *OrderBuffer) Pop() (Item, bool) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if len(rb.Buffer) == 0 {
		return Item{}, false
	}
	first := rb.Buffer[0]
	rb.Buffer = rb.Buffer[1:]
	return first, true
}

func (rb *OrderBuffer) Size() int {
	var size int
	rb.mu.Lock()
	size = len(rb.Buffer)
	rb.mu.Unlock()
	return size
}

// func (rb *OrderBuffer) Ch() <-chan Item {
// 	ch := make(chan Item, rb.max)
// 	go func() {
// 		for _, item := range rb.Buffer {
// 			ch <- item
// 		}
// 		close(ch)
// 	}()
// 	return ch
// }
