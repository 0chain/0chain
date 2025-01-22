package ringbuffer

import "container/ring"

type RingBuffer struct {
	buffer *ring.Ring
	size   int
}

func New(capacity int) *RingBuffer {
	return &RingBuffer{
		buffer: ring.New(capacity),
		size:   0,
	}
}

func (cb *RingBuffer) Add(value any) {
	cb.buffer.Value = value
	cb.buffer = cb.buffer.Next()
	if cb.size < cb.buffer.Len() {
		cb.size++
	}
}

func (cb *RingBuffer) Prev() *ring.Ring {
	return cb.buffer.Prev()
}

func (cb *RingBuffer) Next() *ring.Ring {
	return cb.buffer.Next()
}

func (cb *RingBuffer) Len() int {
	return cb.size
}
