// Copyright 2015 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package deque_test

import (
	"container/list"

	"github.com/juju/testing"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"

	"github.com/juju/collections/deque"
)

type suite struct {
	testing.IsolationSuite
	deque *deque.Deque
}

var _ = gc.Suite(&suite{})

const testLen = 1000

func (s *suite) SetUpTest(c *gc.C) {
	s.deque = deque.New()
}

func (s *suite) TestInit(c *gc.C) {
	s.checkEmpty(c)
}

func (s *suite) TestStackBack(c *gc.C) {
	// Push many values on to the back.
	for i := 0; i < testLen; i++ {
		c.Assert(s.deque.Len(), gc.Equals, i)
		s.deque.PushBack(i)
	}

	// Pop them all off from the back.
	for i := testLen; i > 0; i-- {
		c.Assert(s.deque.Len(), gc.Equals, i)
		v, ok := s.deque.PopBack()
		c.Assert(ok, jc.IsTrue)
		c.Assert(v.(int), gc.Equals, i-1)
	}

	s.checkEmpty(c)
}

func (s *suite) TestStackFront(c *gc.C) {
	// Push many values on to the front.
	for i := 0; i < testLen; i++ {
		c.Assert(s.deque.Len(), gc.Equals, i)
		s.deque.PushFront(i)
	}

	// Pop them all off from the front.
	for i := testLen; i > 0; i-- {
		c.Assert(s.deque.Len(), gc.Equals, i)
		v, ok := s.deque.PopFront()
		c.Assert(ok, jc.IsTrue)
		c.Assert(v.(int), gc.Equals, i-1)
	}

	s.checkEmpty(c)
}

func (s *suite) TestQueueFromFront(c *gc.C) {
	// Push many values on to the back.
	for i := 0; i < testLen; i++ {
		s.deque.PushBack(i)
	}

	// Pop them all off the front.
	for i := 0; i < testLen; i++ {
		v, ok := s.deque.PopFront()
		c.Assert(ok, jc.IsTrue)
		c.Assert(v.(int), gc.Equals, i)
	}

	s.checkEmpty(c)
}

func (s *suite) TestQueueFromBack(c *gc.C) {
	// Push many values on to the front.
	for i := 0; i < testLen; i++ {
		s.deque.PushFront(i)
	}

	// Pop them all off the back.
	for i := 0; i < testLen; i++ {
		v, ok := s.deque.PopBack()
		c.Assert(ok, jc.IsTrue)
		c.Assert(v.(int), gc.Equals, i)
	}

	s.checkEmpty(c)
}

func (s *suite) TestFrontBack(c *gc.C) {
	// Populate from the front and back.
	for i := 0; i < testLen; i++ {
		c.Assert(s.deque.Len(), gc.Equals, i*2)
		s.deque.PushFront(i)
		s.deque.PushBack(i)
	}

	//  Remove half the items from the front and back.
	for i := testLen; i > testLen/2; i-- {
		c.Assert(s.deque.Len(), gc.Equals, i*2)

		vb, ok := s.deque.PopBack()
		c.Assert(ok, jc.IsTrue)
		c.Assert(vb.(int), gc.Equals, i-1)

		vf, ok := s.deque.PopFront()
		c.Assert(ok, jc.IsTrue)
		c.Assert(vf.(int), gc.Equals, i-1)
	}

	// Expand out again.
	for i := testLen / 2; i < testLen; i++ {
		c.Assert(s.deque.Len(), gc.Equals, i*2)
		s.deque.PushFront(i)
		s.deque.PushBack(i)
	}

	// Consume all.
	for i := testLen; i > 0; i-- {
		c.Assert(s.deque.Len(), gc.Equals, i*2)

		vb, ok := s.deque.PopBack()
		c.Assert(ok, jc.IsTrue)
		c.Assert(vb.(int), gc.Equals, i-1)

		vf, ok := s.deque.PopFront()
		c.Assert(ok, jc.IsTrue)
		c.Assert(vf.(int), gc.Equals, i-1)
	}

	s.checkEmpty(c)
}

func (s *suite) TestMaxLenFront(c *gc.C) {
	const max = 5
	d := deque.NewWithMaxLen(max)

	// Exceed the maximum length by 2
	for i := 0; i < max+2; i++ {
		d.PushFront(i)
	}

	// Observe the the first 2 items on the back were dropped.
	v, ok := d.PopBack()
	c.Assert(ok, jc.IsTrue)
	c.Assert(v.(int), gc.Equals, 2)
}

func (s *suite) TestMaxLenBack(c *gc.C) {
	const max = 5
	d := deque.NewWithMaxLen(max)

	// Exceed the maximum length by 3
	for i := 0; i < max+3; i++ {
		d.PushBack(i)
	}

	// Observe the the first 3 items on the front were dropped.
	v, ok := d.PopFront()
	c.Assert(ok, jc.IsTrue)
	c.Assert(v.(int), gc.Equals, 3)
}

func (s *suite) TestBlockAllocation(c *gc.C) {
	// This test confirms that the Deque allocates and deallocates
	// blocks as expected.

	for i := 0; i < testLen; i++ {
		s.deque.PushFront(i)
		s.deque.PushBack(i)
	}
	// 2000 items at a blockLen of 64:
	// 31 full blocks + 1 partial front + 1 partial back = 33
	c.Assert(deque.GetDequeBlocks(s.deque), gc.Equals, 33)

	for i := 0; i < testLen; i++ {
		s.deque.PopFront()
		s.deque.PopBack()
	}
	// At empty there should be just 1 block.
	c.Assert(deque.GetDequeBlocks(s.deque), gc.Equals, 1)
}

func (s *suite) checkEmpty(c *gc.C) {
	c.Assert(s.deque.Len(), gc.Equals, 0)

	_, ok := s.deque.PopFront()
	c.Assert(ok, jc.IsFalse)

	_, ok = s.deque.PopBack()
	c.Assert(ok, jc.IsFalse)
}

func (s *suite) BenchmarkPushBackList(c *gc.C) {
	l := list.New()
	for i := 0; i < c.N; i++ {
		l.PushBack(i)
	}
}

func (s *suite) BenchmarkPushBackDeque(c *gc.C) {
	d := deque.New()
	for i := 0; i < c.N; i++ {
		d.PushBack(i)
	}
}

func (s *suite) BenchmarkPushFrontList(c *gc.C) {
	l := list.New()
	for i := 0; i < c.N; i++ {
		l.PushFront(i)
	}
}

func (s *suite) BenchmarkPushFrontDeque(c *gc.C) {
	d := deque.New()
	for i := 0; i < c.N; i++ {
		d.PushFront(i)
	}
}

func (s *suite) BenchmarkPushPopFrontList(c *gc.C) {
	l := list.New()
	for i := 0; i < c.N; i++ {
		l.PushFront(i)
	}
	for i := 0; i < c.N; i++ {
		elem := l.Front()
		_ = elem.Value
		l.Remove(elem)
	}
}

func (s *suite) BenchmarkPushPopFrontDeque(c *gc.C) {
	d := deque.New()
	for i := 0; i < c.N; i++ {
		d.PushFront(i)
	}
	for i := 0; i < c.N; i++ {
		_, _ = d.PopFront()
	}
}

func (s *suite) BenchmarkPushPopBackList(c *gc.C) {
	l := list.New()
	for i := 0; i < c.N; i++ {
		l.PushBack(i)
	}
	for i := 0; i < c.N; i++ {
		elem := l.Back()
		_ = elem.Value
		l.Remove(elem)
	}
}

func (s *suite) BenchmarkPushPopBackDeque(c *gc.C) {
	d := deque.New()
	for i := 0; i < c.N; i++ {
		d.PushBack(i)
	}
	for i := 0; i < c.N; i++ {
		_, _ = d.PopBack()
	}
}

func iterToSlice(iter deque.Iterator) []string {
	var result []string
	var value string
	for iter.Next(&value) {
		result = append(result, value)
	}
	return result
}

func (s *suite) TestIterEmpty(c *gc.C) {
	c.Assert(iterToSlice(s.deque.Iterator()), gc.HasLen, 0)
}

func (s *suite) TestIter(c *gc.C) {
	s.deque.PushFront("second")
	s.deque.PushBack("third")
	s.deque.PushFront("first")
	c.Assert(iterToSlice(s.deque.Iterator()), jc.DeepEquals, []string{"first", "second", "third"})
}

func (s *suite) TestIterOverBlocksBack(c *gc.C) {
	for i := 0; i < testLen; i++ {
		s.deque.PushBack(i)
	}

	iter := s.deque.Iterator()
	expect := 0
	var obtained int
	for iter.Next(&obtained) {
		c.Assert(obtained, gc.Equals, expect)
		expect++
	}
	c.Assert(expect, gc.Equals, testLen)
}

func (s *suite) TestIterOverBlocksFront(c *gc.C) {
	for i := 0; i < testLen; i++ {
		s.deque.PushFront(i)
	}

	iter := s.deque.Iterator()
	expect := testLen - 1
	var obtained int
	for iter.Next(&obtained) {
		c.Assert(obtained, gc.Equals, expect)
		expect--
	}
	c.Assert(expect, gc.Equals, -1)
}

func (s *suite) TestWrongTypePanics(c *gc.C) {
	// Use an int in the deque, and try to get strings out using the iterToSlice method.
	s.deque.PushFront(14)

	c.Assert(func() {
		iterToSlice(s.deque.Iterator())
	}, gc.PanicMatches, "reflect.Set: value of type int is not assignable to type string")

	c.Assert(func() {
		iter := s.deque.Iterator()
		var i int
		iter.Next(i)
	}, gc.PanicMatches, "value is not a pointer")
}
