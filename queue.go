// Copyright © by Jeff Foley 2017-2025. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.
// SPDX-License-Identifier: Apache-2.0

package queue

import (
	"sync"
)

type QueuePriority int

// The priority levels for the priority Queue.
const (
	PriorityLow      QueuePriority = 0
	PriorityNormal   QueuePriority = 1
	PriorityHigh     QueuePriority = 2
	PriorityCritical QueuePriority = 3
)

// Queue implements a FIFO data structure that can support a few priorities.
type Queue interface {
	// Append adds the data to the Queue at priority level PriorityNormal.
	Append(data interface{})

	// AppendPriority adds the data to the Queue with respect to priority.
	AppendPriority(data interface{}, priority QueuePriority)

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

type queue struct {
	sync.Mutex
	signal chan struct{}
	low    []interface{}
	norm   []interface{}
	high   []interface{}
	crit   []interface{}
}

// NewQueue returns an initialized Queue.
func NewQueue() Queue {
	return &queue{signal: make(chan struct{}, 1)}
}

// Append implements the Queue interface.
func (q *queue) Append(data interface{}) {
	q.append(data, PriorityNormal)
}

// AppendPriority implements the Queue interface.
func (q *queue) AppendPriority(data interface{}, priority QueuePriority) {
	q.append(data, priority)
}

func (q *queue) append(data interface{}, priority QueuePriority) {
	q.Lock()
	defer q.Unlock()

	switch priority {
	case PriorityLow:
		q.low = append(q.low, data)
	case PriorityNormal:
		q.norm = append(q.norm, data)
	case PriorityHigh:
		q.high = append(q.high, data)
	case PriorityCritical:
		q.crit = append(q.crit, data)
	}

	select {
	case q.signal <- struct{}{}:
	default:
	}
}

// Signal implements the Queue interface.
func (q *queue) Signal() <-chan struct{} {
	q.Lock()
	defer q.Unlock()

	q.prepSignal()
	return q.signal
}

func (q *queue) prepSignal() {
	var send bool

	select {
	case _, send = <-q.signal:
	default:
	}

	if !send && q.lenWithoutLock() > 0 {
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
	for {
		select {
		case <-q.signal:
		default:
			return
		}
	}
}

// Next implements the Queue interface.
func (q *queue) Next() (interface{}, bool) {
	q.Lock()
	defer q.Unlock()

	var data interface{}
	if len(q.crit) > 0 {
		data = q.crit[0]
		q.crit = q.crit[1:]
	} else if len(q.high) > 0 {
		data = q.high[0]
		q.high = q.high[1:]
	} else if len(q.norm) > 0 {
		data = q.norm[0]
		q.norm = q.norm[1:]
	} else if len(q.low) > 0 {
		data = q.low[0]
		q.low = q.low[1:]
	} else {
		q.drain()
		return nil, false
	}

	q.prepSignal()
	return data, true
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

	return q.lenWithoutLock()
}

func (q *queue) lenWithoutLock() int {
	qlen := len(q.low)
	qlen += len(q.norm)
	qlen += len(q.high)
	qlen += len(q.crit)
	return qlen
}
