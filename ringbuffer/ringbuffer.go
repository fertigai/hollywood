package ringbuffer

import (
	"sync"
	"sync/atomic"
)

type buffer[T any] struct {
	items           []T
	head, tail, mod int64
}

type RingBuffer[T any] struct {
	len     int64
	content *buffer[T]
	mu      sync.Mutex
}

func New[T any](size int64) *RingBuffer[T] {
	return &RingBuffer[T]{
		content: &buffer[T]{
			items: make([]T, size),
			head:  0,
			tail:  0,
			mod:   size,
		},
		len: 0,
	}
}

func (rb *RingBuffer[T]) Push(item T) {
	rb.mu.Lock()
	rb.content.tail = (rb.content.tail + 1) % rb.content.mod
	if rb.content.tail == rb.content.head {
		size := rb.content.mod * 2
		newBuff := make([]T, size)
		for i := int64(0); i < rb.content.mod; i++ {
			idx := (rb.content.tail + i) % rb.content.mod
			newBuff[i] = rb.content.items[idx]
		}
		content := &buffer[T]{
			items: newBuff,
			head:  0,
			tail:  rb.content.mod,
			mod:   size,
		}
		rb.content = content
	}
	atomic.AddInt64(&rb.len, 1)
	rb.content.items[rb.content.tail] = item
	rb.mu.Unlock()
}

func (rb *RingBuffer[T]) Len() int64 {
	return atomic.LoadInt64(&rb.len)
}

func (rb *RingBuffer[T]) Pop() (T, bool) {
	rb.mu.Lock()
	if rb.len == 0 {
		rb.mu.Unlock()
		var t T
		return t, false
	}
	rb.content.head = (rb.content.head + 1) % rb.content.mod
	item := rb.content.items[rb.content.head]
	var t T
	rb.content.items[rb.content.head] = t
	atomic.AddInt64(&rb.len, -1)
	rb.mu.Unlock()
	return item, true
}

// PushFront inserts an item at the front of the buffer.
// The item will be the first one to be popped.
func (rb *RingBuffer[T]) PushFront(item T) {
	rb.mu.Lock()

	// Check if buffer is full (need to grow before inserting)
	if rb.len >= rb.content.mod-1 {
		size := rb.content.mod * 2
		newBuff := make([]T, size)
		// Copy existing items to positions 1, 2, ..., len
		for i := int64(0); i < rb.len; i++ {
			idx := (rb.content.head + 1 + i) % rb.content.mod
			newBuff[i+1] = rb.content.items[idx]
		}
		rb.content = &buffer[T]{
			items: newBuff,
			head:  0,
			tail:  rb.len,
			mod:   size,
		}
	}

	// Write at current head position (the empty slot)
	rb.content.items[rb.content.head] = item
	// Decrement head to create new empty slot
	rb.content.head = (rb.content.head - 1 + rb.content.mod) % rb.content.mod
	atomic.AddInt64(&rb.len, 1)

	rb.mu.Unlock()
}

func (rb *RingBuffer[T]) PopN(n int64) ([]T, bool) {
	rb.mu.Lock()
	if rb.len == 0 {
		rb.mu.Unlock()
		return nil, false
	}
	content := rb.content

	if n >= rb.len {
		n = rb.len
	}
	atomic.AddInt64(&rb.len, -n)

	items := make([]T, n)
	for i := int64(0); i < n; i++ {
		pos := (content.head + 1 + i) % content.mod
		items[i] = content.items[pos]
		var t T
		content.items[pos] = t
	}
	content.head = (content.head + n) % content.mod

	rb.mu.Unlock()
	return items, true
}
