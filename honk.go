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
	"crypto/rand"
	"crypto/rsa"
	"database/sql"
	"fmt"
	"html/template"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

type UserInfo struct {
	UserID   int64
	Username string
}

type WhatAbout struct {
	ID      int64
	Name    string
	Display string
	About   string
	Key     string
	URL     string
}

type Honk struct {
	ID       int64
	UserID   int64
	Username string
	What     string
	Honker   string
	XID      string
	RID      string
	Date     time.Time
	URL      string
	Noise    string
	Audience []string
	HTML     template.HTML
	Donks    []*Donk
}

type Donk struct {
	FileID  int64
	XID     string
	Name    string
	URL     string
	Media   string
	Content []byte
}

type Honker struct {
	ID     int64
	UserID int64
	Name   string
	XID    string
	Flavor string
}

var serverName string
var iconName = "icon.png"

var readviews *Template

func getInfo(r *http.Request) map[string]interface{} {
	templinfo := make(map[string]interface{})
	templinfo["StyleParam"] = getstyleparam("views/style.css")
	templinfo["LocalStyleParam"] = getstyleparam("views/local.css")
	templinfo["ServerName"] = serverName
	templinfo["IconName"] = iconName
	templinfo["UserInfo"] = GetUserInfo(r)
	templinfo["LogoutCSRF"] = GetCSRF("logout", r)
	return templinfo
}

func homepage(w http.ResponseWriter, r *http.Request) {
	templinfo := getInfo(r)
	u := GetUserInfo(r)
	var honks []*Honk
	if u != nil {
		honks = gethonksforuser(u.UserID)
		templinfo["HonkCSRF"] = GetCSRF("honkhonk", r)
	} else {
		honks = gethonks()
	}
	sort.Slice(honks, func(i, j int) bool {
		return honks[i].Date.After(honks[j].Date)
	})
	reverbolate(honks)

	var modtime time.Time
	if len(honks) > 0 {
		modtime = honks[0].Date
	}
	debug := false
	getconfig("debug", &debug)
	imh := r.Header.Get("If-Modified-Since")
	if !debug && imh != "" && !modtime.IsZero() {
		ifmod, err := time.Parse(http.TimeFormat, imh)
		if err == nil && !modtime.After(ifmod) {
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}

	msg := "Things happen."
	getconfig("servermsg", &msg)
	templinfo["Honks"] = honks
	templinfo["ShowRSS"] = true
	templinfo["ServerMessage"] = msg
	w.Header().Set("Cache-Control", "max-age=0")
	w.Header().Set("Last-Modified", modtime.Format(http.TimeFormat))
	err := readviews.ExecuteTemplate(w, "homepage.html", templinfo)
	if err != nil {
		log.Print(err)
	}
}

func showrss(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]

	var honks []*Honk
	if name != "" {
		honks = gethonksbyuser(name)
	} else {
		honks = gethonks()
	}
	sort.Slice(honks, func(i, j int) bool {
		return honks[i].Date.After(honks[j].Date)
	})
	reverbolate(honks)

	home := fmt.Sprintf("https://%s/", serverName)
	base := home
	if name != "" {
		home += "u/" + name
		name += " "
	}
	feed := RssFeed{
		Title:       name + "honk",
		Link:        home,
		Description: name + "honk rss",
		FeedImage: &RssFeedImage{
			URL:   base + "icon.png",
			Title: name + "honk rss",
			Link:  home,
		},
	}
	var modtime time.Time
	past := time.Now().UTC().Add(-3 * 24 * time.Hour)
	for _, honk := range honks {
		if honk.Date.Before(past) {
			break
		}
		if honk.URL[0] == '/' {
			honk.URL = "https://" + serverName + honk.URL
		}
		feed.Items = append(feed.Items, &RssItem{
			Title:       fmt.Sprintf("%s %s %s", honk.Username, honk.What, honk.XID),
			Description: RssCData{string(honk.HTML)},
			Link:        honk.URL,
			PubDate:     honk.Date.Format(time.RFC1123),
		})
		if honk.Date.After(modtime) {
			modtime = honk.Date
		}
	}
	w.Header().Set("Cache-Control", "max-age=300")
	w.Header().Set("Last-Modified", modtime.Format(http.TimeFormat))

	err := feed.Write(w)
	if err != nil {
		log.Printf("error writing rss: %s", err)
	}
}

func butwhatabout(name string) (*WhatAbout, error) {
	row := stmtWhatAbout.QueryRow(name)
	var user WhatAbout
	err := row.Scan(&user.ID, &user.Name, &user.Display, &user.About, &user.Key)
	user.URL = fmt.Sprintf("https://%s/u/%s", serverName, user.Name)
	return &user, err
}

func crappola(j map[string]interface{}) bool {
	t, _ := jsongetstring(j, "type")
	a, _ := jsongetstring(j, "actor")
	o, _ := jsongetstring(j, "object")
	if t == "Delete" && a == o {
		log.Printf("crappola from %s", a)
		return true
	}
	return false
}

func ping(user *WhatAbout, who string) {
	inbox, _, err := getboxes(who)
	if err != nil {
		log.Printf("no inbox for ping: %s", err)
		return
	}
	j := NewJunk()
	j["@context"] = itiswhatitis
	j["type"] = "Ping"
	j["id"] = user.URL + "/ping/" + xfiltrate()
	j["actor"] = user.URL
	j["to"] = who
	keyname, key := ziggy(user.Name)
	err = PostJunk(keyname, key, inbox, j)
	if err != nil {
		log.Printf("can't send ping: %s", err)
		return
	}
	log.Printf("sent ping to %s: %s", who, j["id"])
}

func pong(user *WhatAbout, who string, obj string) {
	inbox, _, err := getboxes(who)
	if err != nil {
		log.Printf("no inbox for pong %s : %s", who, err)
		return
	}
	j := NewJunk()
	j["@context"] = itiswhatitis
	j["type"] = "Pong"
	j["id"] = user.URL + "/pong/" + xfiltrate()
	j["actor"] = user.URL
	j["to"] = who
	j["object"] = obj
	keyname, key := ziggy(user.Name)
	err = PostJunk(keyname, key, inbox, j)
	if err != nil {
		log.Printf("can't send pong: %s", err)
		return
	}
}

func inbox(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	user, err := butwhatabout(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	var buf bytes.Buffer
	io.Copy(&buf, r.Body)
	payload := buf.Bytes()
	j, err := ReadJunk(bytes.NewReader(payload))
	if err != nil {
		log.Printf("bad payload: %s", err)
		io.WriteString(os.Stdout, "bad payload\n")
		os.Stdout.Write(payload)
		io.WriteString(os.Stdout, "\n")
		return
	}
	if crappola(j) {
		return
	}
	keyname, err := zag(r, payload)
	if err != nil {
		log.Printf("inbox message failed signature: %s", err)
		fd, _ := os.OpenFile("savedinbox.json", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		io.WriteString(fd, "bad signature:\n")
		WriteJunk(fd, j)
		io.WriteString(fd, "\n")
		fd.Close()
		return
	}
	fd, _ := os.OpenFile("savedinbox.json", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	WriteJunk(fd, j)
	io.WriteString(fd, "\n")
	fd.Close()
	who, _ := jsongetstring(j, "actor")
	if !keymatch(keyname, who) {
		log.Printf("keyname actor mismatch: %s <> %s", keyname, who)
		return
	}
	what, _ := jsongetstring(j, "type")
	switch what {
	case "Ping":
		obj, _ := jsongetstring(j, "id")
		log.Printf("ping from %s: %s", who, obj)
		pong(user, who, obj)
	case "Pong":
		obj, _ := jsongetstring(j, "object")
		log.Printf("pong from %s: %s", who, obj)
	case "Follow":
		log.Printf("updating honker follow: %s", who)
		rubadubdub(user, j)
	case "Accept":
		db := opendatabase()
		log.Printf("updating honker accept: %s", who)
		db.Exec("update honkers set flavor = 'sub' where xid = ? and flavor = 'presub'", who)
	case "Undo":
		obj, ok := jsongetmap(j, "object")
		if !ok {
			log.Printf("unknown undo no object")
		} else {
			what, _ := jsongetstring(obj, "type")
			if what != "Follow" {
				log.Printf("unknown undo: %s", what)
			} else {
				log.Printf("updating honker undo: %s", who)
				db := opendatabase()
				db.Exec("update honkers set flavor = 'undub' where xid = ? and flavor = 'dub'", who)
			}
		}
	default:
		xonk := xonkxonk(j)
		if xonk != nil && needxonk(user, xonk) {
			xonk.UserID = user.ID
			savexonk(xonk)
		}
	}
}

func outbox(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	user, err := butwhatabout(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	honks := gethonksbyuser(name)

	var jonks []map[string]interface{}
	for _, h := range honks {
		j, _ := jonkjonk(user, h)
		jonks = append(jonks, j)
	}

	j := NewJunk()
	j["@context"] = itiswhatitis
	j["id"] = user.URL + "/outbox"
	j["type"] = "OrderedCollection"
	j["totalItems"] = len(jonks)
	j["orderedItems"] = jonks

	w.Header().Set("Content-Type", theonetruename)
	WriteJunk(w, j)
}

func viewuser(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	user, err := butwhatabout(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if friendorfoe(r.Header.Get("Accept")) {
		j := asjonker(user)
		w.Header().Set("Content-Type", theonetruename)
		WriteJunk(w, j)
		return
	}
	honks := gethonksbyuser(name)
	u := GetUserInfo(r)
	honkpage(w, r, u, user, honks)
}

func viewhonker(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	u := GetUserInfo(r)
	honks := gethonksbyhonker(u.UserID, name)
	honkpage(w, r, nil, nil, honks)
}

func fingerlicker(w http.ResponseWriter, r *http.Request) {
	orig := r.FormValue("resource")

	log.Printf("finger lick: %s", orig)

	if strings.HasPrefix(orig, "acct:") {
		orig = orig[5:]
	}

	name := orig
	idx := strings.LastIndexByte(name, '/')
	if idx != -1 {
		name = name[idx+1:]
		if "https://"+serverName+"/u/"+name != orig {
			log.Printf("foreign request rejected")
			name = ""
		}
	} else {
		idx = strings.IndexByte(name, '@')
		if idx != -1 {
			name = name[:idx]
			if name+"@"+serverName != orig {
				log.Printf("foreign request rejected")
				name = ""
			}
		}
	}
	user, err := butwhatabout(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	j := NewJunk()
	j["subject"] = fmt.Sprintf("acct:%s@%s", user.Name, serverName)
	j["aliases"] = []string{user.URL}
	var links []map[string]interface{}
	l := NewJunk()
	l["rel"] = "self"
	l["type"] = `application/activity+json`
	l["href"] = user.URL
	links = append(links, l)
	j["links"] = links

	w.Header().Set("Content-Type", "application/jrd+json")
	WriteJunk(w, j)
}

func viewhonk(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	xid := mux.Vars(r)["xid"]
	user, err := butwhatabout(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	h := getxonk(name, xid)
	if h == nil {
		http.NotFound(w, r)
		return
	}
	if friendorfoe(r.Header.Get("Accept")) {
		_, j := jonkjonk(user, h)
		j["@context"] = itiswhatitis
		w.Header().Set("Content-Type", theonetruename)
		WriteJunk(w, j)
		return
	}
	honkpage(w, r, nil, nil, []*Honk{h})
}

func honkpage(w http.ResponseWriter, r *http.Request, u *UserInfo, user *WhatAbout, honks []*Honk) {
	reverbolate(honks)
	templinfo := getInfo(r)
	if u != nil && u.Username == user.Name {
		templinfo["UserCSRF"] = GetCSRF("saveuser", r)
		templinfo["HonkCSRF"] = GetCSRF("honkhonk", r)
	}
	if user != nil {
		templinfo["Name"] = user.Name
		whatabout := user.About
		templinfo["RawWhatAbout"] = whatabout
		whatabout = obfusbreak(whatabout)
		templinfo["WhatAbout"] = cleanstring(whatabout)
	}
	templinfo["Honks"] = honks
	err := readviews.ExecuteTemplate(w, "honkpage.html", templinfo)
	if err != nil {
		log.Print(err)
	}
}

func saveuser(w http.ResponseWriter, r *http.Request) {
	whatabout := r.FormValue("whatabout")
	u := GetUserInfo(r)
	db := opendatabase()
	_, err := db.Exec("update users set about = ? where username = ?", whatabout, u.Username)
	if err != nil {
		log.Printf("error bouting what: %s", err)
	}

	http.Redirect(w, r, "/u/"+u.Username, http.StatusSeeOther)
}

func gethonkers(userid int64) []*Honker {
	rows, err := stmtHonkers.Query(userid)
	if err != nil {
		log.Printf("error querying honkers: %s", err)
		return nil
	}
	defer rows.Close()
	var honkers []*Honker
	for rows.Next() {
		var f Honker
		err = rows.Scan(&f.ID, &f.UserID, &f.Name, &f.XID, &f.Flavor)
		if err != nil {
			log.Printf("error scanning honker: %s", err)
			return nil
		}
		honkers = append(honkers, &f)
	}
	return honkers
}

func getdubs(userid int64) []*Honker {
	rows, err := stmtDubbers.Query(userid)
	if err != nil {
		log.Printf("error querying dubs: %s", err)
		return nil
	}
	defer rows.Close()
	var honkers []*Honker
	for rows.Next() {
		var f Honker
		err = rows.Scan(&f.ID, &f.UserID, &f.Name, &f.XID, &f.Flavor)
		if err != nil {
			log.Printf("error scanning honker: %s", err)
			return nil
		}
		honkers = append(honkers, &f)
	}
	return honkers
}

func getxonk(name, xid string) *Honk {
	var h Honk
	var dt, aud string
	row := stmtOneXonk.QueryRow(xid)
	err := row.Scan(&h.ID, &h.UserID, &h.Username, &h.What, &h.Honker, &h.XID, &h.RID,
		&dt, &h.URL, &aud, &h.Noise)
	if err != nil {
		log.Printf("error scanning xonk: %s", err)
		return nil
	}
	if name != "" && h.Username != name {
		log.Printf("user xonk mismatch")
		return nil
	}
	h.Date, _ = time.Parse(dbtimeformat, dt)
	h.Audience = strings.Split(aud, " ")
	donksforhonks([]*Honk{&h})
	return &h
}

func gethonks() []*Honk {
	rows, err := stmtHonks.Query()
	return getsomehonks(rows, err)
}
func gethonksbyuser(name string) []*Honk {
	rows, err := stmtUserHonks.Query(name)
	return getsomehonks(rows, err)
}
func gethonksforuser(userid int64) []*Honk {
	dt := time.Now().UTC().Add(-2 * 24 * time.Hour)
	rows, err := stmtHonksForUser.Query(userid, dt.Format(dbtimeformat))
	return getsomehonks(rows, err)
}
func gethonksbyhonker(userid int64, honker string) []*Honk {
	rows, err := stmtHonksByHonker.Query(userid, honker)
	return getsomehonks(rows, err)
}

func getsomehonks(rows *sql.Rows, err error) []*Honk {
	if err != nil {
		log.Printf("error querying honks: %s", err)
		return nil
	}
	defer rows.Close()
	var honks []*Honk
	for rows.Next() {
		var h Honk
		var dt, aud string
		err = rows.Scan(&h.ID, &h.UserID, &h.Username, &h.What, &h.Honker, &h.XID, &h.RID,
			&dt, &h.URL, &aud, &h.Noise)
		if err != nil {
			log.Printf("error scanning honks: %s", err)
			return nil
		}
		h.Date, _ = time.Parse(dbtimeformat, dt)
		h.Audience = strings.Split(aud, " ")
		honks = append(honks, &h)
	}
	rows.Close()
	donksforhonks(honks)
	return honks
}

func donksforhonks(honks []*Honk) {
	db := opendatabase()
	var ids []string
	hmap := make(map[int64]*Honk)
	for _, h := range honks {
		ids = append(ids, fmt.Sprintf("%d", h.ID))
		hmap[h.ID] = h
	}
	q := fmt.Sprintf("select honkid, donks.fileid, xid, name, url, media from donks join files on donks.fileid = files.fileid where honkid in (%s)", strings.Join(ids, ","))
	rows, err := db.Query(q)
	if err != nil {
		log.Printf("error querying donks: %s", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var hid int64
		var d Donk
		err = rows.Scan(&hid, &d.FileID, &d.XID, &d.Name, &d.URL, &d.Media)
		if err != nil {
			log.Printf("error scanning donk: %s", err)
			continue
		}
		h := hmap[hid]
		h.Donks = append(h.Donks, &d)
	}
}

func savebonk(w http.ResponseWriter, r *http.Request) {
	xid := r.FormValue("xid")

	log.Printf("bonking %s", xid)

	xonk := getxonk("", xid)
	if xonk == nil {
		return
	}
	if xonk.Honker == "" {
		xonk.XID = fmt.Sprintf("https://%s/u/%s/h/%s", serverName, xonk.Username, xonk.XID)
	}

	userinfo := GetUserInfo(r)

	dt := time.Now().UTC()
	bonk := Honk{
		UserID:   userinfo.UserID,
		Username: userinfo.Username,
		Honker:   xonk.Honker,
		What:     "bonk",
		XID:      xonk.XID,
		Date:     dt,
		Noise:    xonk.Noise,
		Donks:    xonk.Donks,
		Audience: oneofakind(prepend(thewholeworld, xonk.Audience)),
	}

	aud := strings.Join(bonk.Audience, " ")
	res, err := stmtSaveHonk.Exec(userinfo.UserID, "bonk", "", xid, "",
		dt.Format(dbtimeformat), "", aud, bonk.Noise)
	if err != nil {
		log.Printf("error saving bonk: %s", err)
		return
	}
	bonk.ID, _ = res.LastInsertId()
	for _, d := range bonk.Donks {
		_, err = stmtSaveDonk.Exec(bonk.ID, d.FileID)
		if err != nil {
			log.Printf("err saving donk: %s", err)
			return
		}
	}

	user, _ := butwhatabout(userinfo.Username)

	go honkworldwide(user, &bonk)

}

func savehonk(w http.ResponseWriter, r *http.Request) {
	rid := r.FormValue("rid")
	noise := r.FormValue("noise")

	userinfo := GetUserInfo(r)

	dt := time.Now().UTC()
	xid := xfiltrate()
	if xid == "" {
		return
	}
	what := "honk"
	if rid != "" {
		what = "tonk"
	}
	honk := Honk{
		UserID:   userinfo.UserID,
		Username: userinfo.Username,
		What:     "honk",
		XID:      xid,
		RID:      rid,
		Date:     dt,
	}
	if noise[0] == '@' {
		honk.Audience = append(grapevine(noise), thewholeworld)
	} else {
		honk.Audience = append([]string{thewholeworld}, grapevine(noise)...)
	}
	if rid != "" {
		xonk := getxonk("", rid)
		if xonk != nil {
			honk.Audience = append(honk.Audience, xonk.Audience...)
		} else {
			xonkaud := whosthere(rid)
			honk.Audience = append(honk.Audience, xonkaud...)
		}
	}
	honk.Audience = oneofakind(honk.Audience)
	noise = obfusbreak(noise)
	honk.Noise = noise

	file, _, err := r.FormFile("donk")
	if err == nil {
		var buf bytes.Buffer
		io.Copy(&buf, file)
		file.Close()
		data := buf.Bytes()
		img, format, err := image.Decode(&buf)
		if err != nil {
			log.Printf("bad image: %s", err)
			return
		}
		data, format, err = vacuumwrap(img, format)
		if err != nil {
			log.Printf("can't vacuum image: %s", err)
			return
		}
		name := xfiltrate()
		media := "image/" + format
		if format == "jpeg" {
			format = "jpg"
		}
		name = name + "." + format
		url := fmt.Sprintf("https://%s/d/%s", serverName, name)
		res, err := stmtSaveFile.Exec(name, name, url, media, data)
		if err != nil {
			log.Printf("unable to save image: %s", err)
			return
		}
		var d Donk
		d.FileID, _ = res.LastInsertId()
		d.XID = name
		d.Name = name
		d.Media = media
		d.URL = url
		honk.Donks = append(honk.Donks, &d)
	}
	herd := herdofemus(honk.Noise)
	for _, e := range herd {
		donk := savedonk(e.ID, e.Name, "image/png")
		if donk != nil {
			honk.Donks = append(honk.Donks, donk)
		}
	}

	aud := strings.Join(honk.Audience, " ")
	res, err := stmtSaveHonk.Exec(userinfo.UserID, what, "", xid, rid,
		dt.Format(dbtimeformat), "", aud, noise)
	if err != nil {
		log.Printf("error saving honk: %s", err)
		return
	}
	honk.ID, _ = res.LastInsertId()
	for _, d := range honk.Donks {
		_, err = stmtSaveDonk.Exec(honk.ID, d.FileID)
		if err != nil {
			log.Printf("err saving donk: %s", err)
			return
		}
	}

	user, _ := butwhatabout(userinfo.Username)

	go honkworldwide(user, &honk)

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func showhonkers(w http.ResponseWriter, r *http.Request) {
	userinfo := GetUserInfo(r)
	templinfo := getInfo(r)
	templinfo["Honkers"] = gethonkers(userinfo.UserID)
	templinfo["HonkerCSRF"] = GetCSRF("savehonker", r)
	err := readviews.ExecuteTemplate(w, "honkers.html", templinfo)
	if err != nil {
		log.Print(err)
	}
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
	defer handlock.Unlock()
	ref, ok := handfull[name]
	if ok {
		return ref
	}
	j, err := GetJunk(fmt.Sprintf("https://%s/.well-known/webfinger?resource=acct:%s", m[1], name))
	if err != nil {
		log.Printf("failed to go fish %s: %s", name, err)
		handfull[name] = ""
		return ""
	}
	links, _ := jsongetarray(j, "links")
	for _, l := range links {
		href, _ := jsongetstring(l, "href")
		rel, _ := jsongetstring(l, "rel")
		t, _ := jsongetstring(l, "type")
		if rel == "self" && friendorfoe(t) {
			handfull[name] = href
			return href
		}
	}
	handfull[name] = ""
	return ""
}

func savehonker(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	url := r.FormValue("url")
	peep := r.FormValue("peep")
	flavor := "presub"
	if peep == "peep" {
		flavor = "peep"
	}

	if url == "" {
		return
	}
	if url[0] == '@' {
		url = gofish(url)
	}
	if url == "" {
		return
	}

	u := GetUserInfo(r)
	db := opendatabase()
	_, err := db.Exec("insert into honkers (userid, name, xid, flavor) values (?, ?, ?, ?)",
		u.UserID, name, url, flavor)
	if err != nil {
		log.Print(err)
	}
	if flavor == "presub" {
		user, _ := butwhatabout(u.Username)
		go subsub(user, url)
	}
	http.Redirect(w, r, "/honkers", http.StatusSeeOther)
}

func avatate(w http.ResponseWriter, r *http.Request) {
	n := r.FormValue("a")
	a := avatar(n)
	w.Header().Set("Cache-Control", "max-age=432000")
	w.Write(a)
}

func servecss(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "max-age=7776000")
	http.ServeFile(w, r, "views"+r.URL.Path)
}
func servehtml(w http.ResponseWriter, r *http.Request) {
	templinfo := getInfo(r)
	err := readviews.ExecuteTemplate(w, r.URL.Path[1:]+".html", templinfo)
	if err != nil {
		log.Print(err)
	}
}
func serveemu(w http.ResponseWriter, r *http.Request) {
	xid := mux.Vars(r)["xid"]
	w.Header().Set("Cache-Control", "max-age=432000")
	http.ServeFile(w, r, "emus/"+xid)
}

func servefile(w http.ResponseWriter, r *http.Request) {
	xid := mux.Vars(r)["xid"]
	row := stmtFileData.QueryRow(xid)
	var data []byte
	err := row.Scan(&data)
	if err != nil {
		log.Printf("error loading file: %s", err)
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "max-age=432000")
	w.Write(data)
}

func serve() {
	db := opendatabase()
	LoginInit(db)

	listener, err := openListener()
	if err != nil {
		log.Fatal(err)
	}
	debug := false
	getconfig("debug", &debug)
	readviews = ParseTemplates(debug,
		"views/homepage.html",
		"views/honkpage.html",
		"views/honkers.html",
		"views/honkform.html",
		"views/honk.html",
		"views/login.html",
		"views/header.html",
	)
	if !debug {
		s := "views/style.css"
		savedstyleparams[s] = getstyleparam(s)
		s = "views/local.css"
		savedstyleparams[s] = getstyleparam(s)
	}

	mux := mux.NewRouter()
	mux.Use(LoginChecker)

	posters := mux.Methods("POST").Subrouter()
	getters := mux.Methods("GET").Subrouter()

	getters.HandleFunc("/", homepage)
	getters.HandleFunc("/rss", showrss)
	getters.HandleFunc("/u/{name:[[:alnum:]]+}", viewuser)
	getters.HandleFunc("/u/{name:[[:alnum:]]+}/h/{xid:[[:alnum:]]+}", viewhonk)
	getters.HandleFunc("/u/{name:[[:alnum:]]+}/rss", showrss)
	posters.HandleFunc("/u/{name:[[:alnum:]]+}/inbox", inbox)
	getters.HandleFunc("/u/{name:[[:alnum:]]+}/outbox", outbox)
	getters.HandleFunc("/a", avatate)
	getters.HandleFunc("/d/{xid:[[:alnum:].]+}", servefile)
	getters.HandleFunc("/emu/{xid:[[:alnum:]_.]+}", serveemu)
	getters.HandleFunc("/h/{name:[[:alnum:]]+}", viewhonker)
	getters.HandleFunc("/.well-known/webfinger", fingerlicker)

	getters.HandleFunc("/style.css", servecss)
	getters.HandleFunc("/local.css", servecss)
	getters.HandleFunc("/login", servehtml)
	posters.HandleFunc("/dologin", dologin)
	getters.HandleFunc("/logout", dologout)

	loggedin := mux.NewRoute().Subrouter()
	loggedin.Use(LoginRequired)
	loggedin.Handle("/honk", CSRFWrap("honkhonk", http.HandlerFunc(savehonk)))
	loggedin.Handle("/bonk", CSRFWrap("honkhonk", http.HandlerFunc(savebonk)))
	loggedin.Handle("/saveuser", CSRFWrap("saveuser", http.HandlerFunc(saveuser)))
	loggedin.HandleFunc("/honkers", showhonkers)
	loggedin.Handle("/savehonker", CSRFWrap("savehonker", http.HandlerFunc(savehonker)))

	err = http.Serve(listener, mux)
	if err != nil {
		log.Fatal(err)
	}
}

var stmtHonkers, stmtDubbers, stmtOneXonk, stmtHonks, stmtUserHonks *sql.Stmt
var stmtHonksForUser, stmtDeleteHonk, stmtSaveDub *sql.Stmt
var stmtHonksByHonker, stmtSaveHonk, stmtFileData, stmtWhatAbout *sql.Stmt
var stmtFindXonk, stmtSaveDonk, stmtFindFile, stmtSaveFile *sql.Stmt

func prepareStatements(db *sql.DB) {
	var err error
	stmtHonkers, err = db.Prepare("select honkerid, userid, name, xid, flavor from honkers where userid = ? and flavor = 'sub' or flavor = 'peep'")
	if err != nil {
		log.Fatal(err)
	}
	stmtDubbers, err = db.Prepare("select honkerid, userid, name, xid, flavor from honkers where userid = ? and flavor = 'dub'")
	if err != nil {
		log.Fatal(err)
	}
	stmtOneXonk, err = db.Prepare("select honkid, honks.userid, users.username, what, honker, xid, rid, dt, url, audience, noise from honks join users on honks.userid = users.userid where xid = ?")
	if err != nil {
		log.Fatal(err)
	}
	stmtHonks, err = db.Prepare("select honkid, honks.userid, users.username, what, honker, xid, rid, dt, url, audience, noise from honks join users on honks.userid = users.userid where honker = '' order by honkid desc limit 50")
	if err != nil {
		log.Fatal(err)
	}
	stmtUserHonks, err = db.Prepare("select honkid, honks.userid, username, what, honker, xid, rid, dt, url, audience, noise from honks join users on honks.userid = users.userid where honker = '' and username = ? order by honkid desc limit 50")
	if err != nil {
		log.Fatal(err)
	}
	stmtHonksForUser, err = db.Prepare("select honkid, honks.userid, users.username, what, honker, xid, rid, dt, url, audience, noise from honks join users on honks.userid = users.userid where honks.userid = ? and dt > ? order by honkid desc limit 250")
	if err != nil {
		log.Fatal(err)
	}
	stmtHonksByHonker, err = db.Prepare("select honkid, honks.userid, users.username, what, honker, honks.xid, rid, dt, url, audience, noise from honks join users on honks.userid = users.userid join honkers on honkers.xid = honks.honker where honks.userid = ? and honkers.name = ? order by honkid desc limit 50")
	if err != nil {
		log.Fatal(err)
	}
	stmtSaveHonk, err = db.Prepare("insert into honks (userid, what, honker, xid, rid, dt, url, audience, noise) values (?, ?, ?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		log.Fatal(err)
	}
	stmtFileData, err = db.Prepare("select content from files where xid = ?")
	if err != nil {
		log.Fatal(err)
	}
	stmtFindXonk, err = db.Prepare("select honkid from honks where userid = ? and xid = ?")
	if err != nil {
		log.Fatal(err)
	}
	stmtSaveDonk, err = db.Prepare("insert into donks (honkid, fileid) values (?, ?)")
	if err != nil {
		log.Fatal(err)
	}
	stmtDeleteHonk, err = db.Prepare("update honks set what = 'zonk' where xid = ? and honker = ?")
	if err != nil {
		log.Fatal(err)
	}
	stmtFindFile, err = db.Prepare("select fileid from files where url = ?")
	if err != nil {
		log.Fatal(err)
	}
	stmtSaveFile, err = db.Prepare("insert into files (xid, name, url, media, content) values (?, ?, ?, ?, ?)")
	if err != nil {
		log.Fatal(err)
	}
	stmtWhatAbout, err = db.Prepare("select userid, username, displayname, about, pubkey from users where username = ?")
	if err != nil {
		log.Fatal(err)
	}
	stmtSaveDub, err = db.Prepare("insert into honkers (userid, name, xid, flavor) values (?, ?, ?, ?)")
	if err != nil {
		log.Fatal(err)
	}
}

func ElaborateUnitTests() {
}

func finishusersetup() error {
	db := opendatabase()
	k, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}
	pubkey, err := zem(&k.PublicKey)
	if err != nil {
		return err
	}
	seckey, err := zem(k)
	if err != nil {
		return err
	}
	_, err = db.Exec("update users set displayname = username, about = ?, pubkey = ?, seckey = ? where userid = 1", "what about me?", pubkey, seckey)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	cmd := "run"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}
	switch cmd {
	case "init":
		initdb()
	case "upgrade":
		upgradedb()
	}
	db := opendatabase()
	dbversion := 0
	getconfig("dbversion", &dbversion)
	if dbversion != myVersion {
		log.Fatal("incorrect database version. run upgrade.")
	}
	getconfig("servername", &serverName)
	prepareStatements(db)
	switch cmd {
	case "ping":
		if len(os.Args) < 4 {
			fmt.Printf("usage: honk ping from to\n")
			return
		}
		name := os.Args[2]
		targ := os.Args[3]
		user, err := butwhatabout(name)
		if err != nil {
			log.Printf("unknown user")
			return
		}
		ping(user, targ)
	case "peep":
		peeppeep()
	case "run":
		serve()
	case "test":
		ElaborateUnitTests()
	default:
		log.Fatal("unknown command")
	}
}
