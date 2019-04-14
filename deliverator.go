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
	"log"
	"math/rand"
	"time"
)

func sayitagain(goarounds int, username string, rcpt string, msg []byte) {
	var drift time.Duration
	switch goarounds {
	case 1:
		drift = 5 * time.Minute
	case 2:
		drift = 1 * time.Hour
	case 3:
		drift = 12 * time.Hour
	case 4:
		drift = 24 * time.Hour
	default:
		log.Printf("he's dead jim: %s", rcpt)
		return
	}
	drift += time.Duration(rand.Int63n(int64(drift / 16)))
	when := time.Now().UTC().Add(drift)
	log.Print(when.Format(dbtimeformat), goarounds, username, rcpt, msg)
}

func deliverate(goarounds int, username string, rcpt string, msg []byte) {
	keyname, key := ziggy(username)
	inbox, _, err := getboxes(rcpt)
	if err != nil {
		log.Printf("error getting inbox %s: %s", rcpt, err)
		sayitagain(goarounds+1, username, rcpt, msg)
		return
	}
	err = PostMsg(keyname, key, inbox, msg)
	if err != nil {
		log.Printf("failed to post json to %s: %s", inbox, err)
		sayitagain(goarounds+1, username, rcpt, msg)
	}
}
