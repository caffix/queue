// Copyright © by Jeff Foley 2017-2025. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.
// SPDX-License-Identifier: Apache-2.0

package queue

import (
	"testing"
	"time"

	"github.com/caffix/stringset"
)

func TestAppend(t *testing.T) {
	num := 100
	q := NewQueue()

	for i := 0; i < num; i++ {
		q.Append("placeholder")
	}
	if qlen := q.Len(); q.Empty() || qlen != num {
		t.Errorf("expected the queue to contain %d elements, got %d", num, qlen)
	}
	// At a fixed priority, the queue should maintain insertion order (FIFO)
	for i := 0; i < num; i++ {
		if _, ok := q.Next(); !ok {
			t.Errorf("the element at index %d was missing from the queue", i)
			break
		}
	}
	if !q.Empty() {
		t.Errorf("expected the queue to be empty after popping inserted elements, but it still has %d elements", q.Len())
	}
}

func TestAppendPriority(t *testing.T) {
	q := NewQueue()

	q.AppendPriority("value1", PriorityLow)
	q.AppendPriority("value2", PriorityNormal)
	q.AppendPriority("value3", PriorityHigh)
	q.AppendPriority("value4", PriorityCritical)
	q.AppendPriority("value5", PriorityLow)
	q.AppendPriority("value6", PriorityNormal)
	q.AppendPriority("value7", PriorityHigh)
	q.AppendPriority("value8", PriorityCritical)

	expected := []string{
		"value4", "value8",
		"value3", "value7",
		"value2", "value6",
		"value1", "value5",
	}
	for _, want := range expected {
		if have, _ := q.Next(); want != have {
			t.Errorf("element popped out of priority order, expected '%s' but got '%s'", want, have)
		}
	}

	if !q.Empty() {
		t.Errorf("expected the queue to be empty after popping inserted elements, but it still has %d elements", q.Len())
	}
}

func TestSignal(t *testing.T) {
	q := NewQueue()
	times := 1000

	for i := 0; i < times; i++ {
		q.Append("element")
	}
	go func() {
		for i := 0; i < times; i++ {
			q.Append("element")
		}
	}()

	timer := time.NewTimer(10 * time.Second)
	defer timer.Stop()
loop:
	for i := 0; i < times*2; i++ {
		select {
		case <-q.Signal():
			_, _ = q.Next()
		case <-timer.C:
			t.Errorf("use of the Append method did not populate the channel enough")
			break loop
		}
	}
}

func TestNext(t *testing.T) {
	q := NewQueue()
	values := []string{"test1", "test2", "test3", "test4"}
	priorities := []QueuePriority{PriorityNormal, PriorityLow, PriorityCritical, PriorityHigh}
	expected := []string{"test3", "test4", "test1", "test2"}

	for i, v := range values {
		q.AppendPriority(v, priorities[i])
	}
	for _, v := range expected {
		if e, b := q.Next(); b && e.(string) != v {
			t.Errorf("returned %s instead of %s", e.(string), v)
		}
	}
	if _, b := q.Next(); b != false {
		t.Errorf("an empty Queue claimed to return another element")
	}
}

func TestProcess(t *testing.T) {
	q := NewQueue()
	set := stringset.New("element1", "element2")
	defer set.Close()

	for _, e := range set.Slice() {
		q.Append(e)
	}

	ret := stringset.New()
	defer ret.Close()

	q.Process(func(e interface{}) {
		if s, ok := e.(string); ok {
			ret.Insert(s)
		}
	})

	set.Subtract(ret)
	if set.Len() > 0 {
		t.Errorf("not all elements of the queue were provided")
	}
	if q.Len() > 0 {
		t.Errorf("the queue was not empty after executing the Process method")
	}
}

func TestEmpty(t *testing.T) {
	q := NewQueue()

	if !q.Empty() {
		t.Errorf("a new Queue did not claim to be empty")
	}

	q.Append("testing")
	if q.Empty() {
		t.Errorf("a queue with elements claimed to be empty")
	}
}

func TestLen(t *testing.T) {
	q := NewQueue()

	if l := q.Len(); l != 0 {
		t.Errorf("a new Queue returned a length of %d instead of zero", l)
	}

	q.Append("testing")
	if l := q.Len(); l != 1 {
		t.Errorf("a Queue with elements returned a length of %d instead of one", l)
	}
}

func BenchmarkAppend(b *testing.B) {
	q := NewQueue()

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		q.Append("testing")
	}
	b.StopTimer()

	if e, _ := q.Next(); e != "testing" {
		b.Errorf("the element was appended as %s instead of 'testing'", e.(string))
	}
	if want, have := b.N-1, q.Len(); want != have {
		b.Errorf("expected %d elements left on the queue, got %d", want, have)
	}
}

func BenchmarkAppendPriority(b *testing.B) {
	q := NewQueue()

	values := []struct {
		token    string
		priority QueuePriority
	}{
		{"valueLow", PriorityLow},
		{"valueNormal", PriorityNormal},
		{"valueHigh", PriorityHigh},
		{"valueCritical", PriorityCritical},
	}

	topIdx := -1
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		idx := i % len(values)
		q.AppendPriority(values[idx].token, values[idx].priority)
		if topIdx < idx {
			topIdx = idx
		}
	}
	b.StopTimer()

	if e, _ := q.Next(); topIdx > -1 && e != values[topIdx].token {
		b.Errorf("the element was appended as %s instead of %s", e.(string), values[topIdx].token)
	}
	if want, have := b.N-1, q.Len(); want != have {
		b.Errorf("expected %d elements left on the queue, got %d", want, have)
	}
}

func BenchmarkNext(b *testing.B) {
	q := NewQueue()
	for i := 0; i < b.N; i++ {
		q.Append("testing")
	}

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		_, _ = q.Next()
	}
	b.StopTimer()

	if have := q.Len(); have != 0 {
		b.Errorf("expected 0 elements left on the queue, got %d", have)
	}
}

func BenchmarkProcess(b *testing.B) {
	q := NewQueue()
	for i := 0; i < b.N; i++ {
		q.Append("testing")
	}

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		q.Process(func(e interface{}) {})
	}
	b.StopTimer()

	if have := q.Len(); have != 0 {
		b.Errorf("expected 0 elements left on the queue, got %d", have)
	}
}
