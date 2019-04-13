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
	"compress/gzip"
	"crypto/rsa"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

func NewJunk() map[string]interface{} {
	return make(map[string]interface{})
}

func WriteJunk(w io.Writer, j map[string]interface{}) error {
	e := json.NewEncoder(w)
	e.SetEscapeHTML(false)
	e.SetIndent("", "  ")
	err := e.Encode(j)
	return err
}

func ReadJunk(r io.Reader) (map[string]interface{}, error) {
	decoder := json.NewDecoder(r)
	var j map[string]interface{}
	err := decoder.Decode(&j)
	if err != nil {
		return nil, err
	}
	return j, nil
}

var theonetruename = `application/ld+json; profile="https://www.w3.org/ns/activitystreams"`
var falsenames = []string{
	`application/ld+json`,
	`application/activity+json`,
}
var itiswhatitis = "https://www.w3.org/ns/activitystreams"
var thewholeworld = "https://www.w3.org/ns/activitystreams#Public"

func friendorfoe(ct string) bool {
	ct = strings.ToLower(ct)
	for _, at := range falsenames {
		if strings.HasPrefix(ct, at) {
			return true
		}
	}
	return false
}

func PostJunk(keyname string, key *rsa.PrivateKey, url string, j map[string]interface{}) error {
	client := http.DefaultClient
	var buf bytes.Buffer
	WriteJunk(&buf, j)
	req, err := http.NewRequest("POST", url, &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", theonetruename)
	zig(keyname, key, req, buf.Bytes())
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 && resp.StatusCode != 202 {
		resp.Body.Close()
		return fmt.Errorf("http post status: %d", resp.StatusCode)
	}
	log.Printf("successful post: %s %d", url, resp.StatusCode)
	return nil
}

type gzCloser struct {
	r     *gzip.Reader
	under io.ReadCloser
}

func (gz *gzCloser) Read(p []byte) (int, error) {
	return gz.r.Read(p)
}

func (gz *gzCloser) Close() error {
	defer gz.under.Close()
	return gz.r.Close()
}

func GetJunk(url string) (map[string]interface{}, error) {
	client := http.DefaultClient
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", theonetruename)
	req.Header.Set("Accept-Encoding", "gzip")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		resp.Body.Close()
		return nil, fmt.Errorf("http get status: %d", resp.StatusCode)
	}
	if strings.EqualFold(resp.Header.Get("Content-Encoding"), "gzip") {
		gz, err := gzip.NewReader(resp.Body)
		if err != nil {
			resp.Body.Close()
			return nil, err
		}
		resp.Body = &gzCloser{r: gz, under: resp.Body}
	}
	defer resp.Body.Close()
	j, err := ReadJunk(resp.Body)
	return j, err
}

func jsonfindinterface(ii interface{}, keys []string) interface{} {
	for _, key := range keys {
		idx, err := strconv.Atoi(key)
		if err == nil {
			m := ii.([]interface{})
			if idx >= len(m) {
				return nil
			}
			ii = m[idx]
		} else {
			m := ii.(map[string]interface{})
			ii = m[key]
			if ii == nil {
				return nil
			}
		}
	}
	return ii
}
func jsonfindstring(j interface{}, keys []string) (string, bool) {
	s, ok := jsonfindinterface(j, keys).(string)
	return s, ok
}
func jsonfindarray(j interface{}, keys []string) ([]interface{}, bool) {
	a, ok := jsonfindinterface(j, keys).([]interface{})
	return a, ok
}
func jsonfindmap(j interface{}, keys []string) (map[string]interface{}, bool) {
	m, ok := jsonfindinterface(j, keys).(map[string]interface{})
	return m, ok
}
func jsongetstring(j interface{}, key string) (string, bool) {
	return jsonfindstring(j, []string{key})
}
func jsongetarray(j interface{}, key string) ([]interface{}, bool) {
	return jsonfindarray(j, []string{key})
}
func jsongetmap(j interface{}, key string) (map[string]interface{}, bool) {
	return jsonfindmap(j, []string{key})
}

func sha256string(s string) string {
	hasher := sha256.New()
	io.WriteString(hasher, s)
	sum := hasher.Sum(nil)
	return fmt.Sprintf("%x", sum)
}

func savedonk(url string, name, media string) *Donk {
	log.Printf("saving donk: %s", url)
	var donk Donk
	row := stmtFindFile.QueryRow(url)
	err := row.Scan(&donk.FileID)
	if err == nil {
		return &donk
	}
	if err != nil && err != sql.ErrNoRows {
		log.Printf("err querying: %s", err)
	}
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("errer fetching %s: %s", url, err)
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil
	}
	var buf bytes.Buffer
	io.Copy(&buf, resp.Body)

	xid := xfiltrate()

	res, err := stmtSaveFile.Exec(xid, name, url, media, buf.Bytes())
	if err != nil {
		log.Printf("error saving file %s: %s", url, err)
		return nil
	}
	donk.FileID, _ = res.LastInsertId()
	return &donk
}

func needxonk(userid int64, x *Honk) bool {
	row := stmtFindXonk.QueryRow(userid, x.XID, x.What)
	err := row.Scan(&x.ID)
	if err == nil {
		return false
	}
	if err != sql.ErrNoRows {
		log.Printf("err querying xonk: %s", err)
	}
	return true
}

func savexonk(x *Honk) {
	if x.What == "eradicate" {
		log.Printf("eradicating %s by %s", x.RID, x.Honker)
		_, err := stmtDeleteHonk.Exec(x.RID, x.Honker)
		if err != nil {
			log.Printf("error eradicating: %s", err)
		}
		return
	}
	dt := x.Date.UTC().Format(dbtimeformat)
	aud := strings.Join(x.Audience, " ")
	res, err := stmtSaveHonk.Exec(x.UserID, x.What, x.Honker, x.XID, x.RID, dt, x.URL, aud, x.Noise)
	if err != nil {
		log.Printf("err saving xonk: %s", err)
		return
	}
	x.ID, _ = res.LastInsertId()
	for _, d := range x.Donks {
		_, err = stmtSaveDonk.Exec(x.ID, d.FileID)
		if err != nil {
			log.Printf("err saving donk: %s", err)
			return
		}
	}
}

var boxofboxes = make(map[string]string)
var boxlock sync.Mutex

func getboxes(ident string) (string, string, error) {
	boxlock.Lock()
	defer boxlock.Unlock()
	b, ok := boxofboxes[ident]
	if ok {
		if b == "" {
			return "", "", fmt.Errorf("error?")
		}
		m := strings.Split(b, "\n")
		return m[0], m[1], nil
	}
	j, err := GetJunk(ident)
	if err != nil {
		boxofboxes[ident] = ""
		return "", "", err
	}
	inbox, _ := jsongetstring(j, "inbox")
	outbox, _ := jsongetstring(j, "outbox")
	boxofboxes[ident] = inbox + "\n" + outbox
	return inbox, outbox, err
}

func peeppeep() {
	user, _ := butwhatabout("")
	honkers := gethonkers(user.ID)
	for _, f := range honkers {
		if f.Flavor != "peep" {
			continue
		}
		log.Printf("getting updates: %s", f.XID)
		_, outbox, err := getboxes(f.XID)
		if err != nil {
			log.Printf("error getting outbox: %s", err)
			continue
		}
		log.Printf("getting outbox")
		j, err := GetJunk(outbox)
		if err != nil {
			log.Printf("err: %s", err)
			continue
		}
		t, _ := jsongetstring(j, "type")
		if t == "OrderedCollection" {
			items, _ := jsongetarray(j, "orderedItems")
			if items == nil {
				page1, _ := jsongetstring(j, "first")
				j, err = GetJunk(page1)
				if err != nil {
					log.Printf("err: %s", err)
					continue
				}
				items, _ = jsongetarray(j, "orderedItems")
			}

			for _, item := range items {
				xonk := xonkxonk(item)
				if xonk != nil && needxonk(user.ID, xonk) {
					xonk.UserID = user.ID
					savexonk(xonk)
				}
			}
		}
	}
}

func whosthere(xid string) []string {
	obj, err := GetJunk(xid)
	if err != nil {
		log.Printf("error getting remote xonk: %s", err)
		return nil
	}
	return newphone(nil, obj)
}

func newphone(a []string, obj map[string]interface{}) []string {
	for _, addr := range []string{"to", "cc", "attributedTo"} {
		who, _ := jsongetstring(obj, addr)
		if who != "" {
			a = append(a, who)
		}
		whos, _ := jsongetarray(obj, addr)
		for _, w := range whos {
			who, _ := w.(string)
			if who != "" {
				a = append(a, who)
			}
		}
	}
	return a
}

func xonkxonk(item interface{}) *Honk {
	// id, _ := jsongetstring(item, "id")
	what, _ := jsongetstring(item, "type")
	dt, _ := jsongetstring(item, "published")

	var audience []string
	var err error
	var xid, rid, url, content string
	var obj map[string]interface{}
	switch what {
	case "Announce":
		xid, _ = jsongetstring(item, "object")
		log.Printf("getting bonk: %s", xid)
		obj, err = GetJunk(xid)
		if err != nil {
			log.Printf("error regetting: %s", err)
		}
		what = "bonk"
	case "Create":
		obj, _ = jsongetmap(item, "object")
		what = "honk"
	case "Delete":
		obj, _ = jsongetmap(item, "object")
		what = "eradicate"
	default:
		log.Printf("unknown activity: %s", what)
		return nil
	}
	who, _ := jsongetstring(item, "actor")

	var xonk Honk
	if obj != nil {
		ot, _ := jsongetstring(obj, "type")
		url, _ = jsongetstring(obj, "url")
		if ot == "Note" || ot == "Article" {
			audience = newphone(audience, obj)
			xid, _ = jsongetstring(obj, "id")
			content, _ = jsongetstring(obj, "content")
			summary, _ := jsongetstring(obj, "content")
			if summary != "" {
				content = "<p>summary: " + summary + content
			}
			rid, _ = jsongetstring(obj, "inReplyTo")
			if what == "honk" && rid != "" {
				what = "tonk"
			}
		}
		if ot == "Tombstone" {
			rid, _ = jsongetstring(obj, "id")
		}
		atts, _ := jsongetarray(obj, "attachment")
		for _, att := range atts {
			at, _ := jsongetstring(att, "type")
			mt, _ := jsongetstring(att, "mediaType")
			u, _ := jsongetstring(att, "url")
			name, _ := jsongetstring(att, "name")
			if at == "Document" {
				mt = strings.ToLower(mt)
				log.Printf("attachment: %s %s", mt, u)
				if mt == "image/jpeg" || mt == "image/png" ||
					mt == "image/gif" {
					donk := savedonk(u, name, mt)
					if donk != nil {
						xonk.Donks = append(xonk.Donks, donk)
					}
				}
			}
		}
		tags, _ := jsongetarray(obj, "tag")
		for _, tag := range tags {
			tt, _ := jsongetstring(tag, "type")
			name, _ := jsongetstring(tag, "name")
			if tt == "Emoji" {
				icon, _ := jsongetmap(tag, "icon")
				mt, _ := jsongetstring(icon, "mediaType")
				u, _ := jsongetstring(icon, "url")
				donk := savedonk(u, name, mt)
				if donk != nil {
					xonk.Donks = append(xonk.Donks, donk)
				}
			}
		}
	}
	audience = append(audience, who)

	audience = oneofakind(audience)

	xonk.What = what
	xonk.Honker = who
	xonk.XID = xid
	xonk.RID = rid
	xonk.Date, _ = time.Parse(time.RFC3339, dt)
	xonk.URL = url
	xonk.Noise = content
	xonk.Audience = audience

	return &xonk
}

func rubadubdub(user *WhatAbout, req map[string]interface{}) {
	xid, _ := jsongetstring(req, "id")
	reqactor, _ := jsongetstring(req, "actor")
	j := NewJunk()
	j["@context"] = itiswhatitis
	j["id"] = user.URL + "/dub/" + xid
	j["type"] = "Accept"
	j["actor"] = user.URL
	j["to"] = reqactor
	j["published"] = time.Now().UTC().Format(time.RFC3339)
	j["object"] = req

	WriteJunk(os.Stdout, j)

	actor, _ := jsongetstring(req, "actor")
	inbox, _, err := getboxes(actor)
	if err != nil {
		log.Printf("can't get dub box: %s", err)
		return
	}
	keyname, key := ziggy(user)
	err = PostJunk(keyname, key, inbox, j)
	if err != nil {
		log.Printf("can't rub a dub: %s", err)
		return
	}
	stmtSaveDub.Exec(user.ID, actor, actor, "dub")
}

func subsub(user *WhatAbout, xid string) {
	j := NewJunk()
	j["@context"] = itiswhatitis
	j["id"] = user.URL + "/sub/" + xid
	j["type"] = "Follow"
	j["actor"] = user.URL
	j["to"] = xid
	j["object"] = xid
	j["published"] = time.Now().UTC().Format(time.RFC3339)

	inbox, _, err := getboxes(xid)
	if err != nil {
		log.Printf("can't send follow: %s", err)
		return
	}
	WriteJunk(os.Stdout, j)
	keyname, key := ziggy(user)
	err = PostJunk(keyname, key, inbox, j)
	if err != nil {
		log.Printf("failed to subsub: %s", err)
	}
}

func jonkjonk(user *WhatAbout, h *Honk) (map[string]interface{}, map[string]interface{}) {
	dt := h.Date.Format(time.RFC3339)
	var jo map[string]interface{}
	j := NewJunk()
	j["id"] = user.URL + "/" + h.What + "/" + h.XID
	j["actor"] = user.URL
	j["published"] = dt
	j["to"] = h.Audience[0]
	if len(h.Audience) > 1 {
		j["cc"] = h.Audience[1:]
	}

	switch h.What {
	case "tonk":
		fallthrough
	case "honk":
		j["type"] = "Create"
		jo = NewJunk()
		jo["id"] = user.URL + "/h/" + h.XID
		jo["type"] = "Note"
		jo["published"] = dt
		jo["url"] = user.URL + "/h/" + h.XID
		jo["attributedTo"] = user.URL
		if h.RID != "" {
			jo["inReplyTo"] = h.RID
		}
		jo["to"] = h.Audience[0]
		if len(h.Audience) > 1 {
			jo["cc"] = h.Audience[1:]
		}
		jo["content"] = h.Noise
		var tags []interface{}
		g := bunchofgrapes(h.Noise)
		for _, m := range g {
			t := NewJunk()
			t["type"] = "Mention"
			t["name"] = m.who
			t["href"] = m.where
			tags = append(tags, t)
		}
		herd := herdofemus(h.Noise)
		for _, e := range herd {
			t := NewJunk()
			t["id"] = e.ID
			t["type"] = "Emoji"
			t["name"] = e.Name
			i := NewJunk()
			i["type"] = "Image"
			i["mediaType"] = "image/png"
			i["url"] = e.ID
			t["icon"] = i
			tags = append(tags, t)
		}
		if len(tags) > 0 {
			jo["tag"] = tags
		}
		var atts []interface{}
		for _, d := range h.Donks {
			if re_emus.MatchString(d.Name) {
				continue
			}
			jd := NewJunk()
			jd["mediaType"] = d.Media
			jd["name"] = d.Name
			jd["type"] = "Document"
			jd["url"] = d.URL
			atts = append(atts, jd)
		}
		if len(atts) > 0 {
			jo["attachment"] = atts
		}
		j["object"] = jo
	case "bonk":
		j["type"] = "Announce"
		j["object"] = h.XID
	}

	return j, jo
}

func honkworldwide(user *WhatAbout, honk *Honk) {
	aud := append([]string{}, honk.Audience...)
	for i, a := range aud {
		if a == thewholeworld || a == user.URL {
			aud[i] = ""
		}
	}
	keyname, key := ziggy(user)
	jonk, _ := jonkjonk(user, honk)
	jonk["@context"] = itiswhatitis
	for _, f := range getdubs(user.ID) {
		inbox, _, err := getboxes(f.XID)
		if err != nil {
			log.Printf("error getting inbox %s: %s", f.XID, err)
			continue
		}
		err = PostJunk(keyname, key, inbox, jonk)
		if err != nil {
			log.Printf("failed to post json to %s: %s", inbox, err)
		}
		for i, a := range aud {
			if a == f.XID {
				aud[i] = ""
			}
		}
	}
	for _, a := range aud {
		if a != "" && !strings.HasSuffix(a, "/followers") {
			inbox, _, err := getboxes(a)
			if err != nil {
				log.Printf("error getting inbox %s: %s", a, err)
				continue
			}
			err = PostJunk(keyname, key, inbox, jonk)
			if err != nil {
				log.Printf("failed to post json to %s: %s", inbox, err)
			}
		}
	}
}

func asjonker(user *WhatAbout) map[string]interface{} {
	about := obfusbreak(user.About)

	j := NewJunk()
	j["@context"] = itiswhatitis
	j["id"] = user.URL
	j["type"] = "Person"
	j["inbox"] = user.URL + "/inbox"
	j["outbox"] = user.URL + "/outbox"
	j["name"] = user.Display
	j["preferredUsername"] = user.Name
	j["summary"] = about
	j["url"] = user.URL
	a := NewJunk()
	a["type"] = "icon"
	a["mediaType"] = "image/png"
	a["url"] = fmt.Sprintf("https://%s/a?a=%s", serverName, url.QueryEscape(user.URL))
	j["icon"] = a
	k := NewJunk()
	k["id"] = user.URL + "#key"
	k["owner"] = user.URL
	k["publicKeyPem"] = user.Key
	j["publicKey"] = k

	return j
}
