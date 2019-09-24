//
// Copyright (c) 2019 Ted Unangst <tedu@tedunangst.com>
//
// Permission to use, copy, modify, and distribute this software for any
// purpose with or without fee is hereby granted, provided that the above
// copyright notice and this permission notice appear in all copies.
//
// THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
// WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
// MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
// ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
// WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
// ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
// OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.

package main

import (
	"reflect"
	"sync"
)

type cacheFiller func(key interface{}) (interface{}, bool)

type Cache struct {
	cache  map[interface{}]interface{}
	filler cacheFiller
	lock   sync.Mutex
}

func cacheNew(filler cacheFiller) *Cache {
	c := new(Cache)
	c.cache = make(map[interface{}]interface{})
	c.filler = filler
	return c
}

func (cache *Cache) Get(key interface{}, value interface{}) bool {
	cache.lock.Lock()
	defer cache.lock.Unlock()
	v, ok := cache.cache[key]
	if !ok {
		v, ok = cache.filler(key)
		if ok {
			cache.cache[key] = v
		}
	}
	if ok {
		ptr := reflect.ValueOf(v)
		reflect.ValueOf(value).Elem().Set(ptr)
	}
	return ok
}

func (cache *Cache) Clear(key interface{}) {
	cache.lock.Lock()
	defer cache.lock.Unlock()
	delete(cache.cache, key)
}

func (cache *Cache) Flush() {
	cache.lock.Lock()
	defer cache.lock.Unlock()
	cache.cache = make(map[interface{}]interface{})
}
