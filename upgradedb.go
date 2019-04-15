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
	"database/sql"
	"log"
	"os"
)

func doordie(db *sql.DB, s string) {
	_, err := db.Exec(s)
	if err != nil {
		log.Fatal(err)
	}
}

func upgradedb() {
	db := opendatabase()
	dbversion := 0
	getconfig("dbversion", &dbversion)

	switch dbversion {
	case 0:
		doordie(db, "insert into config (key, value) values ('dbversion', 1)")
		fallthrough
	case 1:
		doordie(db, "create table doovers(dooverid integer primary key, dt text, tries integer, username text, rcpt text, msg blob)")
		doordie(db, "update config set value = 2 where key = 'dbversion'")
		fallthrough
	case 2:
	default:
		log.Fatalf("can't upgrade unknown version %d", dbversion)
	}
	os.Exit(0)
}
