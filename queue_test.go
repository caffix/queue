// Copyright 2017 Jeff Foley. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.

package queue

import (
	"testing"

	"github.com/caffix/stringset"
)

func TestAppend(t *testing.T) {
	q := NewQueue()

	q.Append("testing")
	if q.Empty() || q.Len() == 0 {
		t.Errorf("The element was not appended to the queue")
	}

	if e, _ := q.Next(); e != "testing" {
		t.Errorf("The element was appended as %s instead of 'testing'", e.(string))
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
			t.Errorf("Element popped out of priority order, expected '%s' but got '%s'", want, have)
		}
	}
}

func TestSignal(t *testing.T) {
	q := NewQueue()

	q.Append("element")
	select {
	case <-q.Signal():
	default:
		t.Errorf("Use of the Append method did not populate the channel")
	}
}

func TestNext(t *testing.T) {
	q := NewQueue()
	values := []string{"test1", "test2", "test3", "test4"}
	priorities := []int{90, 75, 30, 5}

	for i, v := range values {
		q.AppendPriority(v, priorities[i])
	}

	for _, v := range values {
		if e, b := q.Next(); b && e.(string) != v {
			t.Errorf("Returned %s instead of %s", e.(string), v)
		}
	}

	if _, b := q.Next(); b != false {
		t.Errorf("An empty Queue claimed to return another element")
	}
}

func TestProcess(t *testing.T) {
	q := NewQueue()
	set := stringset.New("element1", "element2")

	for e := range set {
		q.Append(e)
	}

	ret := stringset.New()
	q.Process(func(e interface{}) {
		if s, ok := e.(string); ok {
			ret.Insert(s)
		}
	})

	set.Subtract(ret)
	if set.Len() > 0 {
		t.Errorf("Not all elements of the queue were provided")
	}

	if q.Len() > 0 {
		t.Errorf("The queue was not empty after executing the Process method")
	}
}

func TestEmpty(t *testing.T) {
	q := NewQueue()

	if !q.Empty() {
		t.Errorf("A new Queue did not claim to be empty")
	}

	q.Append("testing")
	if q.Empty() {
		t.Errorf("A queue with elements claimed to be empty")
	}
}

func TestLen(t *testing.T) {
	q := NewQueue()

	if l := q.Len(); l != 0 {
		t.Errorf("A new Queue returned a length of %d instead of zero", l)
	}

	q.Append("testing")
	if l := q.Len(); l != 1 {
		t.Errorf("A Queue with elements returned a length of %d instead of one", l)
	}
}
