// Copyright 2017 Andreas Pannewitz. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package list

// This file was generated with dotgo
// DO NOT EDIT - Improve the pattern!

import (
	list "container/list"
)

// Note: originally inspired by parts of "cmd/doc/dirs.go"

// ElementSPile is a hybrid container for
// a lazily and concurrently populated growing-only slice
// of items (of type `[]*list.Element`)
// which may be traversed in parallel to it's growth.
//
// Usage for a pile `p`:
//  p := MakeElementSPile(128, 32)
//
// Have it grow concurrently using multiple:
//  var item []*list.Element = something
//  p.Pile(item)
// in as many go routines as You may seem fit.
//
// In parallel, You may either
// traverse `p` in parallel right away:
//  for item, ok := p.Iter(); ok; item, ok = p.Next() { ... do sth with item ... }
// Here p.Iter() starts a new transversal with the first item (if any), and
// p.Next() keeps traverses the ElementSPile.
//
// or traverse blocking / awaiting close first:
//  for item := range <-p.Done() { ... do sth with item ... }
//
// or use the result when available:
//  r, p := <-p.Done(), nil
// Hint: here we get the result in `r` and at the same time discard / deallocate / forget the pile `p` itself.
//
// Note: The traversal is *not* intended to be concurrency safe!
// Thus: You may call `Pile` concurrently to Your traversal, but use of
// either `Done` or `Iter` and `Next` *must* be confined to a single go routine (thread).
//
type ElementSPile struct {
	pile   chan []*list.Element // channel to receive further items
	list   [][]*list.Element    // list of known items
	offset int                  // index for Next()
}

// MakeElementSPile returns a (pointer to a) fresh pile
// of items (of type `[]*list.Element`)
// with size as initial capacity
// and
// with buff as initial leeway, allowing as many Pile's to execute non-blocking before respective Done or Next's.
func MakeElementSPile(size, buff int) *ElementSPile {
	pile := new(ElementSPile)
	pile.list = make([][]*list.Element, 0, size)
	pile.pile = make(chan []*list.Element, buff)
	return pile
}

// Pile appends an `[]*list.Element` item to the ElementSPile.
//
// Note: Pile will block iff buff is exceeded and no Done() or Next()'s are used.
func (d *ElementSPile) Pile(item []*list.Element) {
	d.pile <- item
}

// Close - call once when everything has been piled.
//
// Close intentionally implements io.Closer
//
// Note: After Close(),
// any Close(...) will panic
// and
// any Pile(...) will panic
// and
// any Done() or Next() will return immediately: no eventual blocking, that is.
func (d *ElementSPile) Close() (err error) {
	defer func() {
		if r := recover(); r != nil {
			var ok bool
			if err, ok = r.(error); !ok {
				panic(r)
			}
		}
	}()
	close(d.pile)
	return nil
}

// Iter puts the pile iterator back to the beginning
// and returns the first `Next()`, iff any.
// Usage for a pile `p`:
//  for item, ok := p.Iter(); ok; item, ok = p.Next() { ... do sth with item ... }
func (d *ElementSPile) Iter() (item []*list.Element, ok bool) {
	d.offset = 0
	return d.Next()
}

// Next returns the next item,
// or false iff the pile is exhausted.
//
// Note: Iff the pile is not closed yet,
// Next may block, awaiting some Pile().
func (d *ElementSPile) Next() (item []*list.Element, ok bool) {
	if d.offset < len(d.list) {
		ok = true
		item = d.list[d.offset]
		d.offset++
	} else if item, ok = <-d.pile; ok {
		d.list = append(d.list, item)
		d.offset++
	}
	return item, ok
}

// Done returns a channel which emits the result (as slice of ElementS) once the pile is closed.
//
// Users of Done() *must not* iterate (via Iter() Next()...) before the done-channel is closed!
//
// Done is a convenience - useful iff You do not like/need to start any traversal before the pile is fully populated.
// Once the pile is closed, Done() will signal in constant time.
//
// Note: Upon signalling, the pile is reset to it's tip,
// so You may traverse it (via Next) right away.
// Usage for a pile `p`:
// Traverse blocking / awaiting close first:
//  for item := range <-p.Done() { ... do sth with item ... }
// or use the result when available
//  r, p := <-p.Done(), nil
// while discaring the pile itself.
func (d *ElementSPile) Done() (done <-chan [][]*list.Element) {
	cha := make(chan [][]*list.Element)
	go func(cha chan<- [][]*list.Element, d *ElementSPile) {
		defer close(cha)
		d.offset = 0
		if len(d.list) > d.offset {
			// skip what's already known
			d.offset = len(d.list) - 1
		}
		for _, ok := d.Next(); ok; _, ok = d.Next() {
			// keep draining
		}
		d.offset = 0  // reset
		cha <- d.list // signal the result, and terminate
	}(cha, d)
	return cha
}
