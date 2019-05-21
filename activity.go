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
	"database/sql"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"humungus.tedunangst.com/r/webs/image"
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
var thefakename = `application/activity+json`
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
	var buf bytes.Buffer
	WriteJunk(&buf, j)
	return PostMsg(keyname, key, url, buf.Bytes())
}

func PostMsg(keyname string, key *rsa.PrivateKey, url string, msg []byte) error {
	client := http.DefaultClient
	req, err := http.NewRequest("POST", url, bytes.NewReader(msg))
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "honksnonk/5.0")
	req.Header.Set("Content-Type", theonetruename)
	zig(keyname, key, req, msg)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	switch resp.StatusCode {
	case 200:
	case 201:
	case 202:
	default:
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
	at := thefakename
	if strings.Contains(url, ".well-known/webfinger?resource") {
		at = "application/jrd+json"
	}
	req.Header.Set("Accept", at)
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("User-Agent", "honksnonk/5.0")
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

func savedonk(url string, name, media string) *Donk {
	var donk Donk
	row := stmtFindFile.QueryRow(url)
	err := row.Scan(&donk.FileID)
	if err == nil {
		return &donk
	}
	log.Printf("saving donk: %s", url)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("error querying: %s", err)
	}
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("error fetching %s: %s", url, err)
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil
	}
	var buf bytes.Buffer
	io.Copy(&buf, resp.Body)

	xid := xfiltrate()

	data := buf.Bytes()
	if strings.HasPrefix(media, "image") {
		img, err := image.Vacuum(&buf)
		if err != nil {
			log.Printf("unable to decode image: %s", err)
			return nil
		}
		data = img.Data
		media = "image/" + img.Format
	}
	res, err := stmtSaveFile.Exec(xid, name, url, media, data)
	if err != nil {
		log.Printf("error saving file %s: %s", url, err)
		return nil
	}
	donk.FileID, _ = res.LastInsertId()
	return &donk
}

func needxonk(user *WhatAbout, x *Honk) bool {
	if x.What == "eradicate" {
		return true
	}
	if thoudostbitethythumb(user.ID, x.Audience, x.XID) {
		log.Printf("not saving thumb biter? %s via %s", x.XID, x.Honker)
		return false
	}
	return needxonkid(user, x.XID)
}
func needxonkid(user *WhatAbout, xid string) bool {
	if strings.HasPrefix(xid, user.URL+"/h/") {
		return false
	}
	row := stmtFindXonk.QueryRow(user.ID, xid)
	var id int64
	err := row.Scan(&id)
	if err == nil {
		return false
	}
	if err != sql.ErrNoRows {
		log.Printf("err querying xonk: %s", err)
	}
	return true
}

func savexonk(user *WhatAbout, x *Honk) {
	if x.What == "eradicate" {
		log.Printf("eradicating %s by %s", x.RID, x.Honker)
		xonk := getxonk(user.ID, x.RID)
		if xonk != nil {
			stmtZonkDonks.Exec(xonk.ID)
			_, err := stmtZonkIt.Exec(user.ID, x.RID)
			if err != nil {
				log.Printf("error eradicating: %s", err)
			}
		}
		return
	}
	log.Printf("saving xonk: %s", x.XID)
	dt := x.Date.UTC().Format(dbtimeformat)
	aud := strings.Join(x.Audience, " ")
	whofore := 0
	if strings.Contains(aud, user.URL) {
		whofore = 1
	}
	res, err := stmtSaveHonk.Exec(x.UserID, x.What, x.Honker, x.XID, x.RID, dt, x.URL, aud,
		x.Noise, x.Convoy, whofore, "html", x.Precis, x.Oonker)
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

type Box struct {
	In     string
	Out    string
	Shared string
}

var boxofboxes = make(map[string]*Box)
var boxlock sync.Mutex
var boxinglock sync.Mutex

func getboxes(ident string) (*Box, error) {
	boxlock.Lock()
	b, ok := boxofboxes[ident]
	boxlock.Unlock()
	if ok {
		return b, nil
	}

	boxinglock.Lock()
	defer boxinglock.Unlock()

	boxlock.Lock()
	b, ok = boxofboxes[ident]
	boxlock.Unlock()
	if ok {
		return b, nil
	}

	row := stmtGetBoxes.QueryRow(ident)
	b = &Box{}
	err := row.Scan(&b.In, &b.Out, &b.Shared)
	if err != nil {
		j, err := GetJunk(ident)
		if err != nil {
			return nil, err
		}
		inbox, _ := jsongetstring(j, "inbox")
		outbox, _ := jsongetstring(j, "outbox")
		sbox, _ := jsonfindstring(j, []string{"endpoints", "sharedInbox"})
		b = &Box{In: inbox, Out: outbox, Shared: sbox}
		if inbox != "" {
			_, err = stmtSaveBoxes.Exec(ident, inbox, outbox, sbox, "")
			if err != nil {
				log.Printf("error saving boxes: %s", err)
			}
		}
	}
	boxlock.Lock()
	boxofboxes[ident] = b
	boxlock.Unlock()
	return b, nil
}

func peeppeep() {
	user, _ := butwhatabout("htest")
	honkers := gethonkers(user.ID)
	for _, f := range honkers {
		if f.Flavor != "peep" {
			continue
		}
		log.Printf("getting updates: %s", f.XID)
		box, err := getboxes(f.XID)
		if err != nil {
			log.Printf("error getting outbox: %s", err)
			continue
		}
		log.Printf("getting outbox")
		j, err := GetJunk(box.Out)
		if err != nil {
			log.Printf("err: %s", err)
			continue
		}
		t, _ := jsongetstring(j, "type")
		origin := originate(f.XID)
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
				xonk := xonkxonk(user, item, origin)
				if xonk != nil {
					savexonk(user, xonk)
				}
			}
		}
	}
}

func whosthere(xid string) ([]string, string) {
	obj, err := GetJunk(xid)
	if err != nil {
		log.Printf("error getting remote xonk: %s", err)
		return nil, ""
	}
	convoy, _ := jsongetstring(obj, "context")
	if convoy == "" {
		convoy, _ = jsongetstring(obj, "conversation")
	}
	return newphone(nil, obj), convoy
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

func consumeactivity(user *WhatAbout, j interface{}, origin string) {
	xonk := xonkxonk(user, j, origin)
	if xonk != nil {
		savexonk(user, xonk)
	}
}

func xonkxonk(user *WhatAbout, item interface{}, origin string) *Honk {
	depth := 0
	maxdepth := 4
	var xonkxonkfn func(item interface{}, origin string) *Honk

	saveoneup := func(xid string) {
		log.Printf("getting oneup: %s", xid)
		if depth >= maxdepth {
			log.Printf("in too deep")
			return
		}
		obj, err := GetJunk(xid)
		if err != nil {
			log.Printf("error getting oneup: %s", err)
			return
		}
		depth++
		xonk := xonkxonkfn(obj, originate(xid))
		if xonk != nil {
			savexonk(user, xonk)
		}
		depth--
	}

	xonkxonkfn = func(item interface{}, origin string) *Honk {
		// id, _ := jsongetstring(item, "id")
		what, _ := jsongetstring(item, "type")
		dt, _ := jsongetstring(item, "published")

		var audience []string
		var err error
		var xid, rid, url, content, precis, convoy, oonker string
		var obj map[string]interface{}
		var ok bool
		switch what {
		case "Announce":
			obj, ok = jsongetmap(item, "object")
			if ok {
				xid, _ = jsongetstring(obj, "id")
			} else {
				xid, _ = jsongetstring(item, "object")
			}
			if !needxonkid(user, xid) {
				return nil
			}
			log.Printf("getting bonk: %s", xid)
			obj, err = GetJunk(xid)
			if err != nil {
				log.Printf("error regetting: %s", err)
			}
			origin = originate(xid)
			what = "bonk"
		case "Create":
			obj, _ = jsongetmap(item, "object")
			what = "honk"
		case "Delete":
			obj, _ = jsongetmap(item, "object")
			rid, _ = jsongetstring(item, "object")
			what = "eradicate"
		case "Note":
			fallthrough
		case "Article":
			fallthrough
		case "Page":
			obj = item.(map[string]interface{})
			what = "honk"
		default:
			log.Printf("unknown activity: %s", what)
			fd, _ := os.OpenFile("savedinbox.json", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
			WriteJunk(fd, item.(map[string]interface{}))
			io.WriteString(fd, "\n")
			fd.Close()
			return nil
		}

		var xonk Honk
		who, _ := jsongetstring(item, "actor")
		if obj != nil {
			if who == "" {
				who, _ = jsongetstring(obj, "attributedTo")
			}
			oonker, _ = jsongetstring(obj, "attributedTo")
			ot, _ := jsongetstring(obj, "type")
			url, _ = jsongetstring(obj, "url")
			if ot == "Note" || ot == "Article" || ot == "Page" {
				audience = newphone(audience, obj)
				xid, _ = jsongetstring(obj, "id")
				precis, _ = jsongetstring(obj, "summary")
				content, _ = jsongetstring(obj, "content")
				if !strings.HasPrefix(content, "<p>") {
					content = "<p>" + content
				}
				rid, _ = jsongetstring(obj, "inReplyTo")
				convoy, _ = jsongetstring(obj, "context")
				if convoy == "" {
					convoy, _ = jsongetstring(obj, "conversation")
				}
				if what == "honk" && rid != "" {
					what = "tonk"
				}
			}
			if ot == "Tombstone" {
				rid, _ = jsongetstring(obj, "id")
			}
			atts, _ := jsongetarray(obj, "attachment")
			for i, att := range atts {
				at, _ := jsongetstring(att, "type")
				mt, _ := jsongetstring(att, "mediaType")
				u, _ := jsongetstring(att, "url")
				name, _ := jsongetstring(att, "name")
				if i < 4 && (at == "Document" || at == "Image") {
					mt = strings.ToLower(mt)
					log.Printf("attachment: %s %s", mt, u)
					if mt == "image/jpeg" || mt == "image/png" ||
						mt == "text/plain" {
						donk := savedonk(u, name, mt)
						if donk != nil {
							xonk.Donks = append(xonk.Donks, donk)
						}
					} else {
						u = html.EscapeString(u)
						content += fmt.Sprintf(
							`<p>External attachment: <a href="%s" rel=noreferrer>%s</a>`, u, u)
					}
				} else {
					log.Printf("unknown attachment: %s", at)
				}
			}
			tags, _ := jsongetarray(obj, "tag")
			for _, tag := range tags {
				tt, _ := jsongetstring(tag, "type")
				name, _ := jsongetstring(tag, "name")
				if tt == "Emoji" {
					icon, _ := jsongetmap(tag, "icon")
					mt, _ := jsongetstring(icon, "mediaType")
					if mt == "" {
						mt = "image/png"
					}
					u, _ := jsongetstring(icon, "url")
					donk := savedonk(u, name, mt)
					if donk != nil {
						xonk.Donks = append(xonk.Donks, donk)
					}
				}
			}
		}
		if originate(xid) != origin {
			log.Printf("original sin: %s <> %s", xid, origin)
			return nil
		}
		audience = append(audience, who)

		audience = oneofakind(audience)

		if oonker == who {
			oonker = ""
		}
		xonk.UserID = user.ID
		xonk.What = what
		xonk.Honker = who
		xonk.XID = xid
		xonk.RID = rid
		xonk.Date, _ = time.Parse(time.RFC3339, dt)
		xonk.URL = url
		xonk.Noise = content
		xonk.Precis = precis
		xonk.Audience = audience
		xonk.Convoy = convoy
		xonk.Oonker = oonker

		if needxonk(user, &xonk) {
			if what == "tonk" {
				if needxonkid(user, rid) {
					saveoneup(rid)
				}
			}
			return &xonk
		}
		return nil
	}

	return xonkxonkfn(item, origin)
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
	box, err := getboxes(actor)
	if err != nil {
		log.Printf("can't get dub box: %s", err)
		return
	}
	keyname, key := ziggy(user.Name)
	err = PostJunk(keyname, key, box.In, j)
	if err != nil {
		log.Printf("can't rub a dub: %s", err)
		return
	}
	stmtSaveDub.Exec(user.ID, actor, actor, "dub")
}

func itakeitallback(user *WhatAbout, xid string) error {
	j := NewJunk()
	j["@context"] = itiswhatitis
	j["id"] = user.URL + "/unsub/" + xid
	j["type"] = "Undo"
	j["actor"] = user.URL
	j["to"] = xid
	f := NewJunk()
	f["id"] = user.URL + "/sub/" + xid
	f["type"] = "Follow"
	f["actor"] = user.URL
	f["to"] = xid
	j["object"] = f
	j["published"] = time.Now().UTC().Format(time.RFC3339)

	box, err := getboxes(xid)
	if err != nil {
		return err
	}
	keyname, key := ziggy(user.Name)
	err = PostJunk(keyname, key, box.In, j)
	if err != nil {
		return err
	}
	return nil
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

	box, err := getboxes(xid)
	if err != nil {
		log.Printf("can't send follow: %s", err)
		return
	}
	WriteJunk(os.Stdout, j)
	keyname, key := ziggy(user.Name)
	err = PostJunk(keyname, key, box.In, j)
	if err != nil {
		log.Printf("failed to subsub: %s", err)
	}
}

func jonkjonk(user *WhatAbout, h *Honk) (map[string]interface{}, map[string]interface{}) {
	dt := h.Date.Format(time.RFC3339)
	var jo map[string]interface{}
	j := NewJunk()
	j["id"] = user.URL + "/" + h.What + "/" + shortxid(h.XID)
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
		jo["id"] = h.XID
		jo["type"] = "Note"
		jo["published"] = dt
		jo["url"] = h.XID
		jo["attributedTo"] = user.URL
		if h.RID != "" {
			jo["inReplyTo"] = h.RID
		}
		if h.Convoy != "" {
			jo["context"] = h.Convoy
			jo["conversation"] = h.Convoy
		}
		jo["to"] = h.Audience[0]
		if len(h.Audience) > 1 {
			jo["cc"] = h.Audience[1:]
		}
		jo["summary"] = h.Precis
		jo["content"] = mentionize(h.Noise)
		if strings.HasPrefix(h.Precis, "DZ:") {
			jo["sensitive"] = true
		}
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
	case "zonk":
		j["type"] = "Delete"
		j["object"] = h.XID
	}

	return j, jo
}

func honkworldwide(user *WhatAbout, honk *Honk) {
	jonk, _ := jonkjonk(user, honk)
	jonk["@context"] = itiswhatitis
	var buf bytes.Buffer
	WriteJunk(&buf, jonk)
	msg := buf.Bytes()

	rcpts := make(map[string]bool)
	for _, a := range honk.Audience {
		if a != thewholeworld && a != user.URL && !strings.HasSuffix(a, "/followers") {
			box, _ := getboxes(a)
			if box != nil && box.Shared != "" {
				rcpts["%"+box.Shared] = true
			} else {
				rcpts[a] = true
			}
		}
	}
	for _, f := range getdubs(user.ID) {
		box, _ := getboxes(f.XID)
		if box != nil && box.Shared != "" {
			rcpts["%"+box.Shared] = true
		} else {
			rcpts[f.XID] = true
		}
	}
	for a := range rcpts {
		go deliverate(0, user.Name, a, msg)
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
	j["followers"] = user.URL + "/followers"
	j["following"] = user.URL + "/following"
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

var handfull = make(map[string]string)
var handlock sync.Mutex

func gofish(name string) string {
	if name[0] == '@' {
		name = name[1:]
	}
	m := strings.Split(name, "@")
	if len(m) != 2 {
		log.Printf("bad fish name: %s", name)
		return ""
	}
	handlock.Lock()
	ref, ok := handfull[name]
	handlock.Unlock()
	if ok {
		return ref
	}
	db := opendatabase()
	row := db.QueryRow("select ibox from xonkers where xid = ?", name)
	var href string
	err := row.Scan(&href)
	if err == nil {
		handlock.Lock()
		handfull[name] = href
		handlock.Unlock()
		return href
	}
	log.Printf("fishing for %s", name)
	j, err := GetJunk(fmt.Sprintf("https://%s/.well-known/webfinger?resource=acct:%s", m[1], name))
	if err != nil {
		log.Printf("failed to go fish %s: %s", name, err)
		handlock.Lock()
		handfull[name] = ""
		handlock.Unlock()
		return ""
	}
	links, _ := jsongetarray(j, "links")
	for _, l := range links {
		href, _ := jsongetstring(l, "href")
		rel, _ := jsongetstring(l, "rel")
		t, _ := jsongetstring(l, "type")
		if rel == "self" && friendorfoe(t) {
			db.Exec("insert into xonkers (xid, ibox, obox, sbox, pubkey) values (?, ?, ?, ?, ?)",
				name, href, "", "", "")
			handlock.Lock()
			handfull[name] = href
			handlock.Unlock()
			return href
		}
	}
	handlock.Lock()
	handfull[name] = ""
	handlock.Unlock()
	return ""
}
