// Copyright 2018 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package deque

import (
	"container/list"
	"reflect"
)

// Iterator allows in order iteration of the deque elements.
// The iterator is valid as long as no elements are added or removed from the deque.
type Iterator struct {
	deque *Deque
	elem  *list.Element
	block blockT
	index int
	pos   int
}

// Iterator returns an `Iterator` for in order iteration of the elements.
func (d *Deque) Iterator() Iterator {
	front := d.blocks.Front()
	return Iterator{
		deque: d,
		elem:  front,
		block: front.Value.(blockT),
		pos:   d.frontIdx - 1,
	}
}

// Next returns true if there is a value, and the value is populated with the
// next element. Next returns false if it is at the end and the value is
// unchanged. The `value` must be a pointer to the type stored in the deque.
func (i *Iterator) Next(value interface{}) bool {
	if i.index >= i.deque.Len() {
		return false
	}
	i.index++
	i.pos++
	if i.pos == blockLen {
		i.pos = 0
		i.elem = i.elem.Next()
		i.block = i.elem.Value.(blockT)
	}
	v := reflect.ValueOf(value)
	if v.Kind() != reflect.Ptr {
		panic("value is not a pointer")
	}
	v = v.Elem() // dereference the pointer
	v.Set(reflect.ValueOf(i.block[i.pos]))
	return true
}
