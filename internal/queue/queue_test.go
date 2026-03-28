package queue

import (
	"sync"
	"testing"
	"time"
)

func TestQueueBlocksUntilConsumer(t *testing.T) {
	q := New[int](1)

	if !q.Push(1) {
		t.Fatal("first Push() = false, want true")
	}

	released := make(chan struct{})
	go func() {
		defer close(released)
		if !q.Push(2) {
			t.Error("second Push() = false, want true")
		}
	}()

	select {
	case <-released:
		t.Fatal("second push completed before capacity was released")
	case <-time.After(100 * time.Millisecond):
	}

	if got, ok := q.Pop(); !ok || got != 1 {
		t.Fatalf("Pop() = (%v, %v), want (1, true)", got, ok)
	}

	select {
	case <-released:
	case <-time.After(time.Second):
		t.Fatal("second push did not unblock")
	}

	if got, ok := q.Pop(); !ok || got != 2 {
		t.Fatalf("Pop() = (%v, %v), want (2, true)", got, ok)
	}
}

func TestQueueCloseUnblocksWaiters(t *testing.T) {
	q := New[int](1)
	var wg sync.WaitGroup
	wg.Go(func() {
		_, _ = q.Pop()
	})

	time.Sleep(50 * time.Millisecond)
	q.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)
		wg.Wait()
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Pop waiter did not unblock on Close")
	}
}

func TestQueueMinimumCapacityAndPushAfterClose(t *testing.T) {
	q := New[int](0)

	if !q.Push(1) {
		t.Fatal("Push() = false, want true before Close")
	}
	if got, ok := q.Pop(); !ok || got != 1 {
		t.Fatalf("Pop() = (%d, %v), want (1, true)", got, ok)
	}

	q.Close()
	if q.Push(2) {
		t.Fatal("Push() after Close = true, want false")
	}
}
