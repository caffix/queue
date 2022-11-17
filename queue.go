// Copyright © by Jeff Foley 2017-2022. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.
// SPDX-License-Identifier: Apache-2.0

package queue

import (
	"container/heap"
	"sync"
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
	Data     interface{}
	priority int
	index    int
}

type priorityQueue []*queueElement

// Len returns the number of elements remaining in the queue.
func (pq priorityQueue) Len() int { return len(pq) }

// Less returns true when i has a higher priority than j.
func (pq priorityQueue) Less(i, j int) bool {
	return pq[i].priority > pq[j].priority
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
	pq     priorityQueue
}

// NewQueue returns an initialized Queue.
func NewQueue() Queue {
	q := &queue{signal: make(chan struct{}, 1)}
	heap.Init(&q.pq)
	return q
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

	heap.Push(&q.pq, &queueElement{
		Data:     data,
		priority: priority,
	})

	select {
	case q.signal <- struct{}{}:
	default:
	}
}

// Signal implements the Queue interface.
func (q *queue) Signal() <-chan struct{} {
	q.prepSignal()
	return q.signal
}

func (q *queue) prepSignal() {
	q.Lock()
	defer q.Unlock()

	var send bool
	select {
	case _, send = <-q.signal:
	default:
	}

	if !send && q.pq.Len() > 0 {
		send = true
	}
	if send {
		select {
		case q.signal <- struct{}{}:
		default:
		}
	}
}

func (q *queue) drain() {
loop:
	for {
		select {
		case <-q.signal:
		default:
			break loop
		}
	}
}

// Next implements the Queue interface.
func (q *queue) Next() (interface{}, bool) {
	q.Lock()
	defer q.Unlock()

	if q.pq.Len() == 0 {
		q.drain()
		return nil, false
	}

	element := heap.Pop(&q.pq).(*queueElement)
	return element.Data, true
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

	return q.pq.Len()
}
