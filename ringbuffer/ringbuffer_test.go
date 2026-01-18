package ringbuffer

import (
	"sync"
	"sync/atomic"
	"testing"
)

type Item struct {
	i int
}

func TestPushPop(t *testing.T) {
	rb := New[Item](1024)
	for i := 0; i < 5000; i++ {
		rb.Push(Item{i})
		item, ok := rb.Pop()
		if ok {
			if item.i != i {
				t.Fatal("invalid item popped")
			}
		}
	}
}

func TestPushPopN(t *testing.T) {
	rb := New[Item](1024)
	n := 5000
	for i := 0; i < n; i++ {
		rb.Push(Item{i})
	}
	items, ok := rb.PopN(int64(n))
	if !ok {
		t.Fatal("expected to pop many items")
	}
	for i := 0; i < n; i++ {
		if items[i].i != i {
			t.Fatal("invalid item popped")
		}
	}
}

func TestPushFront(t *testing.T) {
	rb := New[Item](1024)
	rb.Push(Item{1})
	rb.Push(Item{2})
	rb.PushFront(Item{0}) // Should be first out

	item, ok := rb.Pop()
	if !ok {
		t.Fatal("expected to pop an item")
	}
	if item.i != 0 {
		t.Fatalf("expected 0, got %d", item.i)
	}

	item, ok = rb.Pop()
	if !ok {
		t.Fatal("expected to pop an item")
	}
	if item.i != 1 {
		t.Fatalf("expected 1, got %d", item.i)
	}

	item, ok = rb.Pop()
	if !ok {
		t.Fatal("expected to pop an item")
	}
	if item.i != 2 {
		t.Fatalf("expected 2, got %d", item.i)
	}
}

func TestPushFrontWithGrow(t *testing.T) {
	// Start with a small buffer to force growth
	rb := New[Item](4)
	// Fill the buffer
	rb.Push(Item{1})
	rb.Push(Item{2})
	rb.Push(Item{3})
	// This should trigger growth
	rb.PushFront(Item{0})

	// Pop and verify order
	item, ok := rb.Pop()
	if !ok {
		t.Fatal("expected to pop an item")
	}
	if item.i != 0 {
		t.Fatalf("expected 0, got %d", item.i)
	}

	item, _ = rb.Pop()
	if item.i != 1 {
		t.Fatalf("expected 1, got %d", item.i)
	}

	item, _ = rb.Pop()
	if item.i != 2 {
		t.Fatalf("expected 2, got %d", item.i)
	}

	item, _ = rb.Pop()
	if item.i != 3 {
		t.Fatalf("expected 3, got %d", item.i)
	}
}

func TestClear(t *testing.T) {
	rb := New[Item](1024)
	rb.Push(Item{1})
	rb.Push(Item{2})
	rb.Push(Item{3})

	if rb.Len() != 3 {
		t.Fatalf("expected len 3, got %d", rb.Len())
	}

	rb.Clear()

	if rb.Len() != 0 {
		t.Fatalf("expected len 0 after clear, got %d", rb.Len())
	}

	// Verify we can still push/pop after clear
	rb.Push(Item{4})
	item, ok := rb.Pop()
	if !ok || item.i != 4 {
		t.Fatal("push/pop after clear failed")
	}
}

func TestPopThreadSafety(t *testing.T) {
	t.Run("Pop should be thread-safe", func(t *testing.T) {
		testCase := func() {
			rb := New[int](4)
			rb.Push(1)
			wg := sync.WaitGroup{}
			for i := 0; i < 2; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					rb.Pop()
				}()
			}
			wg.Wait()
			if rb.Len() == -1 {
				t.Fatal("item popped twice")
			}
		}

		// Increase the number of iterations to raise the likelihood of reproducing the race condition
		for i := 0; i < 100_000; i++ {
			testCase()
		}
	})

	t.Run("PopN should be thread-safe", func(t *testing.T) {
		testCase := func() {
			rb := New[int](4)
			rb.Push(1)
			counter := atomic.Int32{}
			wg := sync.WaitGroup{}
			for i := 0; i < 2; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					_, ok := rb.PopN(1)
					if ok {
						counter.Add(1)
					}
				}()
			}
			wg.Wait()
			if counter.Load() > 1 {
				t.Fatal("false positive item removal")
			}
		}

		// Increase the number of iterations to raise the likelihood of reproducing the race condition
		for i := 0; i < 100_000; i++ {
			testCase()
		}
	})
}
