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
	"fmt"
	"time"

	"humungus.tedunangst.com/r/webs/junk"
)

func updateMe() {
	var user *WhatAbout
	somenumberedusers.Get(1, &user)
	dt := time.Now().UTC().Format(time.RFC3339)
	j := junk.New()
	j["@context"] = itiswhatitis
	j["id"] = fmt.Sprintf("%s/upme/%s/%d", user.URL, user.Name, time.Now().Unix())
	j["actor"] = user.URL
	j["published"] = dt
	j["to"] = []string{thewholeworld, user.URL + "/followers"}
	j["type"] = "Update"
	jo := junkuser(user)
	j["object"] = jo

	msg := j.ToBytes()

	rcpts := make(map[string]bool)
	for _, f := range getdubs(user.ID) {
		if f.XID == user.URL {
			continue
		}
		var box *Box
		boxofboxes.Get(f.XID, &box)
		if box != nil && box.Shared != "" {
			rcpts["%"+box.Shared] = true
		} else {
			rcpts[f.XID] = true
		}
	}
	for a := range rcpts {
		deliverate(0, user.ID, a, msg)
	}
}
