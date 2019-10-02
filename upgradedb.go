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
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
)

func doordie(db *sql.DB, s string, args ...interface{}) {
	_, err := db.Exec(s, args...)
	if err != nil {
		log.Fatalf("can't run %s: %s", s, err)
	}
}

func upgradedb() {
	db := opendatabase()
	dbversion := 0
	getconfig("dbversion", &dbversion)
	getconfig("servername", &serverName)

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
		users := allusers()
		for _, u := range users {
			h := fmt.Sprintf("https://%s/u/%s", serverName, u.Username)
			doordie(db, fmt.Sprintf("update honks set xid = '%s/h/' || xid, honker = ?, whofore = 2 where userid = ? and honker = '' and (what = 'honk' or what = 'tonk')", h), h, u.UserID)
			doordie(db, "update honks set honker = ?, whofore = 2 where userid = ? and honker = '' and what = 'bonk'", h, u.UserID)
		}
		doordie(db, "update config set value = 8 where key = 'dbversion'")
		fallthrough
	case 8:
		doordie(db, "alter table files add column local integer")
		doordie(db, "update files set local = 1")
		doordie(db, "update config set value = 9 where key = 'dbversion'")
		fallthrough
	case 9:
		doordie(db, "drop table xonkers")
		doordie(db, "create table xonkers (xonkerid integer primary key, name text, info text, flavor text)")
		doordie(db, "create index idx_xonkername on xonkers(name)")
		doordie(db, "update config set value = 10 where key = 'dbversion'")
		fallthrough
	case 10:
		doordie(db, "update zonkers set wherefore = 'zomain' where wherefore = 'zurl'")
		doordie(db, "update zonkers set wherefore = 'zord' where wherefore = 'zword'")
		doordie(db, "update config set value = 11 where key = 'dbversion'")
		fallthrough
	case 11:
		doordie(db, "alter table users add column options text")
		doordie(db, "update users set options = ''")
		doordie(db, "update config set value = 12 where key = 'dbversion'")
		fallthrough
	case 12:
		doordie(db, "create index idx_honksoonker on honks(oonker)")
		doordie(db, "update config set value = 13 where key = 'dbversion'")
		fallthrough
	case 13:
		doordie(db, "alter table honks add column flags integer")
		doordie(db, "update honks set flags = 0")
		doordie(db, "update config set value = 14 where key = 'dbversion'")
		fallthrough
	case 14:
		doordie(db, "create table onts (ontology text, honkid integer)")
		doordie(db, "create index idx_ontology on onts(ontology)")
		doordie(db, "update config set value = 15 where key = 'dbversion'")
		fallthrough
	case 15:
		doordie(db, "delete from onts")
		ontmap := make(map[int64][]string)
		rows, err := db.Query("select honkid, noise from honks")
		if err != nil {
			log.Fatalf("can't query honks: %s", err)
		}
		re_more := regexp.MustCompile(`#<span>[[:alpha:]][[:alnum:]-]*`)
		for rows.Next() {
			var honkid int64
			var noise string
			err := rows.Scan(&honkid, &noise)
			if err != nil {
				log.Fatalf("can't scan honks: %s", err)
			}
			onts := ontologies(noise)
			mo := re_more.FindAllString(noise, -1)
			for _, o := range mo {
				onts = append(onts, "#"+o[7:])
			}
			if len(onts) > 0 {
				ontmap[honkid] = oneofakind(onts)
			}
		}
		rows.Close()
		tx, err := db.Begin()
		if err != nil {
			log.Fatalf("can't begin: %s", err)
		}
		stmtOnts, err := tx.Prepare("insert into onts (ontology, honkid) values (?, ?)")
		if err != nil {
			log.Fatal(err)
		}
		for honkid, onts := range ontmap {
			for _, o := range onts {
				_, err = stmtOnts.Exec(strings.ToLower(o), honkid)
				if err != nil {
					log.Fatal(err)
				}
			}
		}
		err = tx.Commit()
		if err != nil {
			log.Fatalf("can't commit: %s", err)
		}
		doordie(db, "update config set value = 16 where key = 'dbversion'")
		fallthrough
	case 16:
		doordie(db, "alter table files add column description text")
		doordie(db, "update files set description = name")
		doordie(db, "update config set value = 17 where key = 'dbversion'")
		fallthrough
	case 17:
		doordie(db, "create table forsaken (honkid integer, precis text, noise text)")
		doordie(db, "update config set value = 18 where key = 'dbversion'")
		fallthrough
	case 18:
		doordie(db, "create index idx_onthonkid on onts(honkid)")
		doordie(db, "update config set value = 19 where key = 'dbversion'")
		fallthrough
	case 19:
		doordie(db, "create table places (honkid integer, name text, latitude real, longitude real)")
		doordie(db, "create index idx_placehonkid on places(honkid)")
		fallthrough
	case 20:
		doordie(db, "alter table places add column url text")
		doordie(db, "update places set url = ''")
		doordie(db, "update config set value = 21 where key = 'dbversion'")
		fallthrough
	case 21:
		// here we go...
		initblobdb()
		blobdb, err := sql.Open("sqlite3", blobdbname)
		if err != nil {
			log.Fatal(err)
		}
		tx, err := blobdb.Begin()
		if err != nil {
			log.Fatalf("can't begin: %s", err)
		}
		doordie(db, "drop index idx_filesxid")
		doordie(db, "drop index idx_filesurl")
		doordie(db, "create table filemeta (fileid integer primary key, xid text, name text, description text, url text, media text, local integer)")
		doordie(db, "insert into filemeta select fileid, xid, name, description, url, media, local from files")
		doordie(db, "create index idx_filesxid on filemeta(xid)")
		doordie(db, "create index idx_filesurl on filemeta(url)")

		rows, err := db.Query("select xid, media, content from files where local = 1")
		if err != nil {
			log.Fatal(err)
		}
		for rows.Next() {
			var xid, media string
			var data []byte
			err = rows.Scan(&xid, &media, &data)
			if err != nil {
				log.Fatal(err)
			}
			_, err = tx.Exec("insert into filedata (xid, media, content) values (?, ?, ?)", xid, media, data)
			if err != nil {
				log.Fatal(err)
			}
		}
		rows.Close()
		err = tx.Commit()
		if err != nil {
			log.Fatalf("can't commit: %s", err)
		}
		doordie(db, "drop table files")
		doordie(db, "vacuum")
		doordie(db, "update config set value = 22 where key = 'dbversion'")
		fallthrough
	case 22:
		doordie(db, "create table honkmeta (honkid integer, genus text, json text)")
		doordie(db, "create index idx_honkmetaid on honkmeta(honkid)")
		doordie(db, "drop table forsaken") // don't bother saving this one
		rows, err := db.Query("select honkid, name, latitude, longitude, url from places")
		if err != nil {
			log.Fatal(err)
		}
		places := make(map[int64]*Place)
		for rows.Next() {
			var honkid int64
			p := new(Place)
			err = rows.Scan(&honkid, &p.Name, &p.Latitude, &p.Longitude, &p.Url)
			if err != nil {
				log.Fatal(err)
			}
			places[honkid] = p
		}
		rows.Close()
		tx, err := db.Begin()
		if err != nil {
			log.Fatalf("can't begin: %s", err)
		}
		for honkid, p := range places {
			j, err := jsonify(p)
			_, err = tx.Exec("insert into honkmeta (honkid, genus, json) values (?, ?, ?)",
				honkid, "place", j)
			if err != nil {
				log.Fatal(err)
			}
		}
		err = tx.Commit()
		if err != nil {
			log.Fatalf("can't commit: %s", err)
		}
		doordie(db, "update config set value = 23 where key = 'dbversion'")
		fallthrough
	case 23:

	default:
		log.Fatalf("can't upgrade unknown version %d", dbversion)
	}
	os.Exit(0)
}
