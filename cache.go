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
	"time"
)

type cacheFiller func(key interface{}) (interface{}, bool)

type cacheOptions struct {
	Filler   interface{}
	Duration time.Duration
}

type Cache struct {
	cache    map[interface{}]interface{}
	filler   cacheFiller
	lock     sync.Mutex
	stale    time.Time
	duration time.Duration
}

func cacheNew(options cacheOptions) *Cache {
	c := new(Cache)
	c.cache = make(map[interface{}]interface{})
	fillfn := options.Filler
	ftype := reflect.TypeOf(fillfn)
	if ftype.Kind() != reflect.Func {
		panic("cache filler is not function")
	}
	if ftype.NumIn() != 1 || ftype.NumOut() != 2 {
		panic("cache filler has wrong argument count")
	}
	c.filler = func(key interface{}) (interface{}, bool) {
		vfn := reflect.ValueOf(fillfn)
		args := []reflect.Value{reflect.ValueOf(key)}
		rv := vfn.Call(args)
		return rv[0].Interface(), rv[1].Bool()
	}
	if options.Duration != 0 {
		c.duration = options.Duration
		c.stale = time.Now().Add(c.duration)
	}
	return c
}

func (cache *Cache) Get(key interface{}, value interface{}) bool {
	cache.lock.Lock()
	defer cache.lock.Unlock()
	if !cache.stale.IsZero() && cache.stale.Before(time.Now()) {
		cache.stale = time.Now().Add(cache.duration)
		cache.cache = make(map[interface{}]interface{})
	}
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
