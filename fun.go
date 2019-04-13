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
	"crypto/rand"
	"crypto/rsa"
	"database/sql"
	"fmt"
	"html"
	"html/template"
	"log"
	"regexp"
	"strings"
)

func reverbolate(honks []*Honk) {
	for _, h := range honks {
		h.What += "ed"
		if h.Honker == "" {
			h.Honker = "https://" + serverName + "/u/" + h.Username
			if strings.IndexByte(h.XID, '/') == -1 {
				h.URL = h.Honker + "/h/" + h.XID
			} else {
				h.URL = h.XID
			}
		} else {
			idx := strings.LastIndexByte(h.Honker, '/')
			if idx != -1 {
				h.Username = honkerhandle(h.Honker)
			} else {
				h.Username = h.Honker
			}
			if h.URL == "" {
				h.URL = h.XID
			}
		}
		zap := make(map[*Donk]bool)
		h.HTML = cleanstring(h.Noise)
		emuxifier := func(e string) string {
			for _, d := range h.Donks {
				if d.Name == e {
					zap[d] = true
					return fmt.Sprintf(`<img class="emu" title="%s" src="/d/%s">`, d.Name, d.XID)
				}
			}
			return e
		}
		h.HTML = template.HTML(re_emus.ReplaceAllStringFunc(string(h.HTML), emuxifier))
		for i := 0; i < len(h.Donks); i++ {
			if zap[h.Donks[i]] {
				copy(h.Donks[i:], h.Donks[i+1:])
				h.Donks = h.Donks[:len(h.Donks)-1]
				i--
			}
		}
	}
}

func xfiltrate() string {
	letters := "BCDFGHJKLMNPQRSTVWXYZbcdfghjklmnpqrstvwxyz1234567891234567891234"
	db := opendatabase()
	for {
		var x int64
		var b [16]byte
		rand.Read(b[:])
		for i, c := range b {
			b[i] = letters[c&63]
		}
		s := string(b[:])
		r := db.QueryRow("select honkid from honks where xid = ?", s)
		err := r.Scan(&x)
		if err == nil {
			continue
		}
		if err != sql.ErrNoRows {
			log.Printf("err picking xid: %s", err)
			return ""
		}
		r = db.QueryRow("select fileid from files where name = ?", s)
		err = r.Scan(&x)
		if err == nil {
			continue
		}
		if err != sql.ErrNoRows {
			log.Printf("err picking xid: %s", err)
			return ""
		}
		return s
	}
}

type Mention struct {
	who   string
	where string
}

var re_mentions = regexp.MustCompile(`@[[:alnum:]]+@[[:alnum:].]+`)

func grapevine(s string) []string {
	m := re_mentions.FindAllString(s, -1)
	var mentions []string
	for i := range m {
		where := gofish(m[i])
		if where != "" {
			mentions = append(mentions, where)
		}
	}
	return mentions
}

func bunchofgrapes(s string) []Mention {
	m := re_mentions.FindAllString(s, -1)
	var mentions []Mention
	for i := range m {
		where := gofish(m[i])
		if where != "" {
			mentions = append(mentions, Mention{who: m[i], where: where})
		}
	}
	return mentions
}

type Emu struct {
	ID   string
	Name string
}

var re_link = regexp.MustCompile(`https?://[^\s"]+[\w/)]`)
var re_emus = regexp.MustCompile(`:[[:alnum:]_]+:`)

func herdofemus(noise string) []Emu {
	m := re_emus.FindAllString(noise, -1)
	m = oneofakind(m)
	var emus []Emu
	for _, e := range m {
		fname := e[1 : len(e)-1]
		url := fmt.Sprintf("https://%s/emu/%s.png", serverName, fname)
		emus = append(emus, Emu{ID: url, Name: e})
	}
	return emus
}

func obfusbreak(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Replace(s, "\r", "", -1)
	s = html.EscapeString(s)
	linkfn := func(url string) string {
		addparen := false
		adddot := false
		if strings.HasSuffix(url, ")") && strings.IndexByte(url, '(') == -1 {
			url = url[:len(url)-1]
			addparen = true
		}
		if strings.HasSuffix(url, ".") {
			url = url[:len(url)-1]
			adddot = true
		}
		url = fmt.Sprintf(`<a href="%s">%s</a>`, url, url)
		if adddot {
			url += "."
		}
		if addparen {
			url += ")"
		}
		return url
	}
	s = re_link.ReplaceAllStringFunc(s, linkfn)

	s = strings.Replace(s, "\n", "<br>", -1)
	s = re_mentions.ReplaceAllStringFunc(s, func(m string) string {
		return fmt.Sprintf(`<a href="%s">%s</a>`, html.EscapeString(gofish(m)),
			html.EscapeString(m))
	})
	return s
}

var re_unurl = regexp.MustCompile("https://([^/]+).*/([^/]+)")

func honkerhandle(h string) string {
	m := re_unurl.FindStringSubmatch(h)
	if len(m) > 2 {
		return fmt.Sprintf("%s@%s", m[2], m[1])
	}
	return h
}

func prepend(s string, x []string) []string {
	return append([]string{s}, x...)
}

func oneofakind(a []string) []string {
	var x []string
	for n, s := range a {
		if s != "" {
			x = append(x, s)
			for i := n + 1; i < len(a); i++ {
				if a[i] == s {
					a[i] = ""
				}
			}
		}
	}
	return x
}

func ziggy(user *WhatAbout) (keyname string, key *rsa.PrivateKey) {
	db := opendatabase()
	row := db.QueryRow("select seckey from users where userid = ?", user.ID)
	var data string
	row.Scan(&data)
	var err error
	key, _, err = pez(data)
	if err != nil {
		log.Printf("error loading %s seckey: %s", user.Name, err)
	}
	keyname = user.URL + "#key"
	return
}

func zaggy(keyname string) (key *rsa.PublicKey) {
	db := opendatabase()
	row := db.QueryRow("select pubkey from honkers where flavor = 'key' and xid = ?", keyname)
	var data string
	err := row.Scan(&data)
	savekey := false
	if err != nil {
		savekey = true
		j, err := GetJunk(keyname)
		if err != nil {
			log.Printf("error getting %s pubkey: %s", keyname, err)
			return
		}
		var ok bool
		data, ok = jsonfindstring(j, []string{"publicKey", "publicKeyPem"})
		if !ok {
			log.Printf("error getting %s pubkey", keyname)
			return
		}
		_, ok = jsonfindstring(j, []string{"publicKey", "owner"})
		if !ok {
			log.Printf("error getting %s pubkey owner", keyname)
			return
		}
	}
	_, key, err = pez(data)
	if err != nil {
		log.Printf("error getting %s pubkey: %s", keyname, err)
		return
	}
	if savekey {
		db.Exec("insert into honkers (name, xid, flavor, pubkey) values (?, ?, ?, ?)",
			"", keyname, "key", data)
	}
	return
}

func keymatch(keyname string, actor string) bool {
	return strings.HasPrefix(keyname, actor)
}
