// Copyright 2017-2021 Jeff Foley. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.

package queue

import (
	"container/heap"
	"sync"
	"time"
)

// The priority levels for the priority Queue.
const (
	PriorityLow int = iota
	PriorityNormal
	PriorityHigh
	PriorityCritical
)

// Queue implements a FIFO data structure that can support priorities.
type Queue interface {
	// Append adds the data to the Queue at priority level PriorityNormal.
	Append(data interface{})

	// AppendPriority adds the data to the Queue with respect to priority.
	AppendPriority(data interface{}, priority int)

	// Signal returns the Queue signal channel.
	Signal() <-chan struct{}

	// Next returns the data at the front of the Queue.
	Next() (interface{}, bool)

	// Process will execute the callback parameter for each element on the Queue.
	Process(callback func(interface{}))

	// Empty returns true if the Queue is empty.
	Empty() bool

	// Len returns the current length of the Queue.
	Len() int
}

type queueElement struct {
	Data      interface{}
	priority  int
	timestamp time.Time
	index     int
}

type priorityQueue []*queueElement

// Len returns the number of elements remaining in the queue.
func (pq priorityQueue) Len() int { return len(pq) }

// Less returns true when i has a higher priority than j.
func (pq priorityQueue) Less(i, j int) bool {
	if pq[i].priority > pq[j].priority {
		return true
	}
	if pq[i].priority == pq[j].priority && pq[i].timestamp.Before(pq[j].timestamp) {
		return true
	}
	return false
}

// Swap exchanges the ith and jth element of the priority queue.
func (pq priorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

// Push adds a new element to the priority queue.
func (pq *priorityQueue) Push(x interface{}) {
	n := len(*pq)
	element := x.(*queueElement)
	element.timestamp = time.Now()
	element.index = n
	*pq = append(*pq, element)
}

// Pop removes the next element from the queue in priority order.
func (pq *priorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	element := old[n-1]
	old[n-1] = nil     // avoid memory leak
	element.index = -1 // for safety
	*pq = old[:n-1]
	return element
}

type queue struct {
	sync.Mutex
	signal chan struct{}
	queue  priorityQueue
}

// NewQueue returns an initialized Queue.
func NewQueue() Queue {
	return &queue{signal: make(chan struct{}, 2)}
}

// Append implements the Queue interface.
func (q *queue) Append(data interface{}) {
	q.append(data, PriorityNormal)
}

// AppendPriority implements the Queue interface.
func (q *queue) AppendPriority(data interface{}, priority int) {
	q.append(data, priority)
}

func (q *queue) append(data interface{}, priority int) {
	q.Lock()
	defer q.Unlock()

	element := &queueElement{
		Data:     data,
		priority: priority,
	}

	heap.Push(&q.queue, element)
	q.sendSignal()
}

// Signal implements the Queue interface.
func (q *queue) Signal() <-chan struct{} {
	return q.signal
}

func (q *queue) sendSignal() {
	// Send the signal up to two times to avoid a race
	// allowing data to remain on the queue without a signal
	for i := 0; i < 2; i++ {
		if len(q.signal) > 1 {
			break
		}

		q.signal <- struct{}{}
	}
}

// Next implements the Queue interface.
func (q *queue) Next() (interface{}, bool) {
	q.Lock()
	defer q.Unlock()

	var ok bool
	var data interface{}
	if q.queue.Len() > 0 {
		element := heap.Pop(&q.queue).(*queueElement)
		ok = true
		data = element.Data
	}

	if q.queue.Len() > 0 {
		q.sendSignal()
	}

	return data, ok
}

// Process implements the Queue interface.
func (q *queue) Process(callback func(interface{})) {
	element, ok := q.Next()

	for ok {
		callback(element)
		element, ok = q.Next()
	}
}

// Empty implements the Queue interface.
func (q *queue) Empty() bool {
	return q.Len() == 0
}

// Len implements the Queue interface.
func (q *queue) Len() int {
	q.Lock()
	defer q.Unlock()

	return q.queue.Len()
}
