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
	"bytes"
	"fmt"
	"log"
	"strings"
	"sync"
)

var bloat_mtx sync.Mutex

func bloat_counterplusone(s string) string {
	bloat_mtx.Lock()
	defer bloat_mtx.Unlock()

	var bloat_counter int
	getconfig("bloat_counter", &bloat_counter)

	if bloat_counter < 9001 {
		bloat_counter++
		saveconfig("bloat_counter", bloat_counter)
	}
	// 1st 2nd 3rd 4th 5th 6th 7th 8th 9th 10th 11th 12th 13th
	suf := "th"
	switch bloat_counter % 10 {
	case 1:
		suf = "st"
	case 2:
		suf = "nd"
	case 3:
		suf = "rd"
	}
	if bloat_counter == 11 || bloat_counter == 12 || bloat_counter == 13 {
		suf = "th"
	}
	val := fmt.Sprintf("%d%s", bloat_counter, suf)
	log.Printf("now producing %s counter", val)
	s = strings.Replace(s, "&lt;bloat_counter&gt;", val, -1)
	return s
}

func bloat_counterfixhonk(honk *Honk) {
	honk.Noise = bloat_counterplusone(honk.Noise)
}

func bloat_counterhtml(honk *Honk) {
	honk.Noise = strings.Replace(honk.Noise, "&lt;bloat_counter&gt;", "1st", -1)
}

func bloat_counterannounce(user *WhatAbout, honk *Honk) {
	rcpts := make(map[string]bool)
	for _, a := range honk.Audience {
		if a != thewholeworld && a != user.URL && !strings.HasSuffix(a, "/followers") {
			box, _ := getboxes(a)
			if box != nil && honk.Public && box.Shared != "" {
				rcpts["%"+box.Shared] = true
			} else {
				rcpts[a] = true
			}
		}
	}
	if honk.Public {
		for _, f := range getdubs(user.ID) {
			box, _ := getboxes(f.XID)
			if box != nil && box.Shared != "" {
				rcpts["%"+box.Shared] = true
			} else {
				rcpts[f.XID] = true
			}
		}
	}
	orignoise := honk.Noise
	for a := range rcpts {
		honk.Noise = orignoise
		bloat_counterfixhonk(honk)
		jonk, _ := jonkjonk(user, honk)
		jonk["@context"] = itiswhatitis
		var buf bytes.Buffer
		jonk.Write(&buf)
		msg := buf.Bytes()
		go deliverate(0, user.Name, a, msg)
	}
}

func bloat_iscounter(honk *Honk) bool {
	return strings.Contains(honk.Noise, "&lt;bloat_counter&gt;")
}

func bloat_undocounter() {
	db := opendatabase()
	db.Exec("update honks set noise = 'This post has expired' where noise like '%&lt;bloat_counter&gt;%' and whofore = 2 and what = 'honk'")
}
