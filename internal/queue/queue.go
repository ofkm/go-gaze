package queue

import "sync"

type Queue[T any] struct {
	ch        chan T
	done      chan struct{}
	closeOnce sync.Once
}

func New[T any](capacity int) *Queue[T] {
	if capacity < 1 {
		capacity = 1
	}
	return &Queue[T]{
		ch:   make(chan T, capacity),
		done: make(chan struct{}),
	}
}

func (q *Queue[T]) Push(v T) bool {
	select {
	case <-q.done:
		return false
	default:
	}

	select {
	case q.ch <- v:
		return true
	case <-q.done:
		return false
	}
}

func (q *Queue[T]) Pop() (T, bool) {
	var zero T

	select {
	case value := <-q.ch:
		return value, true
	case <-q.done:
		select {
		case value := <-q.ch:
			return value, true
		default:
			return zero, false
		}
	}
}

func (q *Queue[T]) Close() {
	q.closeOnce.Do(func() {
		close(q.done)
	})
}
