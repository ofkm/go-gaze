package queue

import "sync"

type Queue[T any] struct {
	mu       sync.Mutex
	notEmpty *sync.Cond
	notFull  *sync.Cond
	buf      []T
	head     int
	tail     int
	size     int
	closed   bool
}

func New[T any](capacity int) *Queue[T] {
	if capacity < 1 {
		capacity = 1
	}
	q := &Queue[T]{
		buf: make([]T, capacity),
	}
	q.notEmpty = sync.NewCond(&q.mu)
	q.notFull = sync.NewCond(&q.mu)
	return q
}

func (q *Queue[T]) Push(v T) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	for !q.closed && q.size == len(q.buf) {
		q.notFull.Wait()
	}
	if q.closed {
		return false
	}

	q.buf[q.tail] = v
	q.tail = (q.tail + 1) % len(q.buf)
	q.size++
	q.notEmpty.Signal()
	return true
}

func (q *Queue[T]) Pop() (T, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	var zero T
	for !q.closed && q.size == 0 {
		q.notEmpty.Wait()
	}
	if q.size == 0 {
		return zero, false
	}

	value := q.buf[q.head]
	q.buf[q.head] = zero
	q.head = (q.head + 1) % len(q.buf)
	q.size--
	q.notFull.Signal()
	return value, true
}

func (q *Queue[T]) Close() {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return
	}
	q.closed = true
	q.notEmpty.Broadcast()
	q.notFull.Broadcast()
}
