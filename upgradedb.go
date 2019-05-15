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
		doordie(db, "alter table honks add column convoy text")
		doordie(db, "update honks set convoy = ''")
		doordie(db, "create index idx_honksconvoy on honks(convoy)")
		doordie(db, "create table xonkers (xonkerid integer primary key, xid text, ibox text, obox text, sbox text, pubkey text)")
		doordie(db, "insert into xonkers (xid, ibox, obox, sbox, pubkey) select xid, '', '', '', pubkey from honkers where flavor = 'key'")
		doordie(db, "delete from honkers where flavor = 'key'")
		doordie(db, "create index idx_xonkerxid on xonkers(xid)")
		doordie(db, "create table zonkers (zonkerid integer primary key, userid integer, name text, wherefore text)")
		doordie(db, "create index idx_zonkersname on zonkers(name)")
		doordie(db, "update config set value = 3 where key = 'dbversion'")
		fallthrough
	case 3:
		doordie(db, "alter table honks add column whofore integer")
		doordie(db, "update honks set whofore = 0")
		doordie(db, "update honks set whofore = 1 where honkid in (select honkid from honks join users on honks.userid = users.userid where instr(audience, username) > 0)")
		doordie(db, "update config set value = 4 where key = 'dbversion'")
		fallthrough
	case 4:
		doordie(db, "alter table honkers add column combos text")
		doordie(db, "update honkers set combos = ''")
		doordie(db, "update config set value = 5 where key = 'dbversion'")
		fallthrough
	case 5:
		doordie(db, "delete from donks where honkid in (select honkid from honks where what = 'zonk')")
		doordie(db, "delete from honks where what = 'zonk'")
		doordie(db, "update config set value = 6 where key = 'dbversion'")
		fallthrough
	case 6:
		doordie(db, "alter table honks add column format")
		doordie(db, "update honks set format = 'html'")
		doordie(db, "alter table honks add column precis")
		doordie(db, "update honks set precis = ''")
		doordie(db, "alter table honks add column oonker")
		doordie(db, "update honks set oonker = ''")
		doordie(db, "update config set value = 7 where key = 'dbversion'")
		fallthrough
	case 7:
	default:
		log.Fatalf("can't upgrade unknown version %d", dbversion)
	}
	os.Exit(0)
}
