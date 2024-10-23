package ringbuffer

import (
	"fmt"
	"testing"
)

func TestRingBuffer(t *testing.T) {
	rb := New(10)
	for i := 0; i < 10; i++ {
		rb.Add(i)
	}

	r := rb.Prev()
	for i := 0; i < 11; i++ {
		fmt.Println(r.Value)
		r = r.Prev()
	}
}
