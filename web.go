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
	"database/sql"
	"fmt"
	"html/template"
	"io"
	notrand "math/rand"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"humungus.tedunangst.com/r/webs/cache"
	"humungus.tedunangst.com/r/webs/httpsig"
	"humungus.tedunangst.com/r/webs/junk"
	"humungus.tedunangst.com/r/webs/login"
	"humungus.tedunangst.com/r/webs/templates"
)

var readviews *templates.Template

var userSep = "u"
var honkSep = "h"

var develMode = false

func getInfo(r *http.Request) map[string]interface{} {
	templinfo := make(map[string]interface{})
	templinfo["ManifestParam"] = getassetparam(viewDir + "/views/manifest.webmanifest")
	templinfo["StyleParam"] = getassetparam(viewDir + "/views/style.css")
	templinfo["LocalStyleParam"] = getassetparam(dataDir + "/views/local.css")
	templinfo["GuestStyleParam"] = getassetparam(dataDir + "/views/guest.css")
	templinfo["JSParam"] = getassetparam(viewDir + "/views/honkpage.js")
	templinfo["LocalJSParam"] = getassetparam(dataDir + "/views/local.js")
	templinfo["ServerName"] = serverName
	templinfo["IconName"] = iconName
	templinfo["UserSep"] = userSep
	if u := login.GetUserInfo(r); u != nil {
		templinfo["UserInfo"], _ = butwhatabout(u.Username)
	}
	tmpl_hasprefix := strings.HasPrefix
	templinfo["HasPrefix"] = tmpl_hasprefix
	return templinfo
}

func homepage(w http.ResponseWriter, r *http.Request) {
	templinfo := getInfo(r)
	u := login.GetUserInfo(r)
	var honks []*Honk
	var userid int64 = -1

	templinfo["ServerMessage"] = serverMsg

	if u == nil {
		honks = getpublichonks()
	} else {
		userid = u.UserID
		switch r.URL.Path {
		case "/atme":
			templinfo["ServerMessage"] = "at me!"
			templinfo["PageName"] = "atme"
			honks = gethonksforme(userid, 0)
			menewnone(userid)
			templinfo["UserInfo"], _ = butwhatabout(u.Username)
		default:
			templinfo["PageName"] = "home"
			honks = gethonksforuser(userid, 0)
		}
		templinfo["HonkCSRF"] = login.GetCSRF("honkhonk", r)
	}

	honkpage(w, u, honks, templinfo)
}

func crappola(j junk.Junk) bool {
	t, _ := j.GetString("type")
	a, _ := j.GetString("actor")
	o, _ := j.GetString("object")
	if t == "Delete" && a == o {
		dlog.Printf("crappola from %s", a)
		return true
	}
	return false
}

func inbox(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	user, err := butwhatabout(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	var buf bytes.Buffer
	limiter := io.LimitReader(r.Body, 1*1024*1024)
	io.Copy(&buf, limiter)
	payload := buf.Bytes()
	j, err := junk.FromBytes(payload)
	if err != nil {
		ilog.Printf("bad payload: %s", err)
		ilog.Writer().Write(payload)
		ilog.Writer().Write([]byte{'\n'})
		return
	}

	if crappola(j) {
		return
	}
	what, _ := j.GetString("type")
	obj, _ := j.GetString("object")
	if what == "Like" || what == "EmojiReact" {
		return
	}
	who, _ := j.GetString("actor")

	keyname, err := httpsig.VerifyRequest(r, payload, zaggy)
	if err != nil && keyname != "" {
		savingthrow(keyname)
		keyname, err = httpsig.VerifyRequest(r, payload, zaggy)
	}
	if err != nil {
		ilog.Printf("inbox message failed signature for %s from %s: %s", keyname, r.Header.Get("X-Forwarded-For"), err)
		if keyname != "" {
			ilog.Printf("bad signature from %s", keyname)
		}
		http.Error(w, "what did you call me?", http.StatusTeapot)
		return
	}
	origin := keymatch(keyname, who)
	if origin == "" {
		ilog.Printf("keyname actor mismatch: %s <> %s", keyname, who)
		return
	}

	switch what {
	case "Follow":
		if obj != user.URL {
			ilog.Printf("can't follow %s", obj)
			return
		}
		followme(user, who, who, j)
	case "Accept":
		followyou2(user, j)
	case "Reject":
		nofollowyou2(user, j)
	case "Update":
		obj, ok := j.GetMap("object")
		if ok {
			what, _ := obj.GetString("type")
			switch what {
			case "Service":
				fallthrough
			case "Person":
				return
			case "Question":
				return
			}
		}
		go xonksaver(user, j, origin)
	case "Undo":
		obj, ok := j.GetMap("object")
		if !ok {
			folxid, ok := j.GetString("object")
			if ok && originate(folxid) == origin {
				unfollowme(user, "", "", j)
			}
			return
		}
		what, _ := obj.GetString("type")
		switch what {
		case "Follow":
			unfollowme(user, who, who, j)
		case "Announce":
			xid, _ := obj.GetString("object")
			dlog.Printf("undo announce: %s", xid)
		case "Like":
		default:
			ilog.Printf("unknown undo: %s", what)
		}
	default:
		go xonksaver(user, j, origin)
	}
}

func serverinbox(w http.ResponseWriter, r *http.Request) {
	user := getserveruser()
	var buf bytes.Buffer
	io.Copy(&buf, r.Body)
	payload := buf.Bytes()
	j, err := junk.FromBytes(payload)
	if err != nil {
		ilog.Printf("bad payload: %s", err)
		ilog.Writer().Write(payload)
		ilog.Writer().Write([]byte{'\n'})
		return
	}
	if crappola(j) {
		return
	}
	keyname, err := httpsig.VerifyRequest(r, payload, zaggy)
	if err != nil && keyname != "" {
		savingthrow(keyname)
		keyname, err = httpsig.VerifyRequest(r, payload, zaggy)
	}
	if err != nil {
		ilog.Printf("inbox message failed signature for %s from %s: %s", keyname, r.Header.Get("X-Forwarded-For"), err)
		if keyname != "" {
			ilog.Printf("bad signature from %s", keyname)
		}
		http.Error(w, "what did you call me?", http.StatusTeapot)
		return
	}
	who, _ := j.GetString("actor")
	origin := keymatch(keyname, who)
	if origin == "" {
		ilog.Printf("keyname actor mismatch: %s <> %s", keyname, who)
		return
	}
	re_ont := regexp.MustCompile("https://" + serverName + "/o/([\\pL[:digit:]]+)")
	what, _ := j.GetString("type")
	dlog.Printf("server got a %s", what)
	switch what {
	case "Follow":
		obj, _ := j.GetString("object")
		if obj == user.URL {
			ilog.Printf("can't follow the server!")
			return
		}
		m := re_ont.FindStringSubmatch(obj)
		if len(m) != 2 {
			ilog.Printf("not sure how to handle this")
			return
		}
		ont := "#" + m[1]

		followme(user, who, ont, j)
	case "Undo":
		obj, ok := j.GetMap("object")
		if !ok {
			ilog.Printf("unknown undo no object")
			return
		}
		what, _ := obj.GetString("type")
		if what != "Follow" {
			ilog.Printf("unknown undo: %s", what)
			return
		}
		targ, _ := obj.GetString("object")
		m := re_ont.FindStringSubmatch(targ)
		if len(m) != 2 {
			ilog.Printf("not sure how to handle this")
			return
		}
		ont := "#" + m[1]
		unfollowme(user, who, ont, j)
	default:
		ilog.Printf("unhandled server activity: %s", what)
		dumpactivity(j)
	}
}

func serveractor(w http.ResponseWriter, r *http.Request) {
	user := getserveruser()
	j := junkuser(user)
	j.Write(w)
}

func ximport(w http.ResponseWriter, r *http.Request) {
	u := login.GetUserInfo(r)
	xid := strings.TrimSpace(r.FormValue("q"))
	xonk := getxonk(u.UserID, xid)
	if xonk == nil {
		p, _ := investigate(xid)
		if p != nil {
			xid = p.XID
		}
		j, err := GetJunk(u.UserID, xid)
		if err != nil {
			http.Error(w, "error getting external object", http.StatusInternalServerError)
			ilog.Printf("error getting external object: %s", err)
			return
		}
		allinjest(originate(xid), j)
		dlog.Printf("importing %s", xid)
		user, _ := butwhatabout(u.Username)

		info, _ := somethingabout(j)
		if info == nil {
			xonk = xonksaver(user, j, originate(xid))
		} else if info.What == SomeActor {
			outbox, _ := j.GetString("outbox")
			gimmexonks(user, outbox)
			http.Redirect(w, r, "/h?xid="+url.QueryEscape(xid), http.StatusSeeOther)
			return
		}
	}
	convoy := ""
	if xonk != nil {
		convoy = xonk.Convoy
	}
	http.Redirect(w, r, "/t?c="+url.QueryEscape(convoy), http.StatusSeeOther)
}

var oldoutbox = cache.New(cache.Options{Filler: func(name string) ([]byte, bool) {
	user, err := butwhatabout(name)
	if err != nil {
		return nil, false
	}
	honks := gethonksbyuser(name, false, 0)
	if len(honks) > 20 {
		honks = honks[0:20]
	}

	var jonks []junk.Junk
	for _, h := range honks {
		j, _ := jonkjonk(user, h)
		jonks = append(jonks, j)
	}

	j := junk.New()
	j["@context"] = itiswhatitis
	j["id"] = user.URL + "/outbox"
	j["attributedTo"] = user.URL
	j["type"] = "OrderedCollection"
	j["totalItems"] = len(jonks)
	j["orderedItems"] = jonks

	return j.ToBytes(), true
}, Duration: 1 * time.Minute})

func outbox(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	var j []byte
	ok := oldoutbox.Get(name, &j)
	if ok {
		w.Header().Set("Content-Type", theonetruename)
		w.Write(j)
	} else {
		http.NotFound(w, r)
	}
}

var oldempties = cache.New(cache.Options{Filler: func(url string) ([]byte, bool) {
	colname := "/followers"
	if strings.HasSuffix(url, "/following") {
		colname = "/following"
	}
	user := fmt.Sprintf("https://%s%s", serverName, url[:len(url)-10])
	j := junk.New()
	j["@context"] = itiswhatitis
	j["id"] = user + colname
	j["attributedTo"] = user
	j["type"] = "OrderedCollection"
	j["totalItems"] = 0
	j["orderedItems"] = []junk.Junk{}

	return j.ToBytes(), true
}})

func emptiness(w http.ResponseWriter, r *http.Request) {
	var j []byte
	ok := oldempties.Get(r.URL.Path, &j)
	if ok {
		w.Header().Set("Content-Type", theonetruename)
		w.Write(j)
	} else {
		http.NotFound(w, r)
	}
}

func showuser(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	user, err := butwhatabout(name)
	if err != nil {
		ilog.Printf("user not found %s: %s", name, err)
		http.NotFound(w, r)
		return
	}
	if friendorfoe(r.Header.Get("Accept")) {
		j, ok := asjonker(name)
		if ok {
			w.Header().Set("Content-Type", theonetruename)
			w.Write(j)
		} else {
			http.NotFound(w, r)
		}
		return
	}
	u := login.GetUserInfo(r)
	honks := gethonksbyuser(name, u != nil && u.Username == name, 0)
	templinfo := getInfo(r)
	templinfo["PageName"] = "user"
	templinfo["PageArg"] = name
	templinfo["Name"] = user.Name
	templinfo["WhatAbout"] = user.HTAbout
	templinfo["ServerMessage"] = ""
	templinfo["HonkCSRF"] = login.GetCSRF("honkhonk", r)
	honkpage(w, u, honks, templinfo)
}

func showhonker(w http.ResponseWriter, r *http.Request) {
	u := login.GetUserInfo(r)
	name := mux.Vars(r)["name"]
	var honks []*Honk
	var miniform template.HTML
	if name == "" {
		name = r.FormValue("xid")
		honks = gethonksbyxonker(u.UserID, name, 0)
	} else {
		honks = gethonksbyhonker(u.UserID, name, 0)
	}
	// Not known as a honker
	if shortname(u.UserID, name) == "" && fullname(name, u.UserID)  == "" {
		miniform = templates.Sprintf(`<form action="/submithonker" method="POST">
			<input type="hidden" name="CSRF" value="%s">
			<input type="hidden" name="url" value="%s">
			<button tabindex=1 name="add honker" value="add honker">add honker</button>
			</form>`, login.GetCSRF("submithonker", r), name)
	}

	msg := templates.Sprintf(`honks by honker: <a href="%s" ref="noreferrer">%s</a>%s`, name, name, miniform)
	templinfo := getInfo(r)
	templinfo["PageName"] = "honker"
	templinfo["PageArg"] = name
	templinfo["ServerMessage"] = msg
	templinfo["HonkCSRF"] = login.GetCSRF("honkhonk", r)
	honkpage(w, u, honks, templinfo)
}

func showconvoy(w http.ResponseWriter, r *http.Request) {
	c := r.FormValue("c")
	u := login.GetUserInfo(r)
	honks := gethonksbyconvoy(u.UserID, c, 0)
	templinfo := getInfo(r)
	if len(honks) > 0 {
		templinfo["TopHID"] = honks[0].ID
	}
	reversehonks(honks)
	templinfo["PageName"] = "convoy"
	templinfo["PageArg"] = c
	templinfo["ServerMessage"] = "honks in skein: " + c
	templinfo["HonkCSRF"] = login.GetCSRF("honkhonk", r)
	honkpage(w, u, honks, templinfo)
}
func showsearch(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.FormValue("q"))
	if strings.HasPrefix(q, "https://") {
		ximport(w, r)
		return
	}
	u := login.GetUserInfo(r)
	honks := gethonksbysearch(u.UserID, q, 0)
	templinfo := getInfo(r)
	templinfo["PageName"] = "search"
	templinfo["PageArg"] = q
	templinfo["ServerMessage"] = "honks for search: " + q
	templinfo["HonkCSRF"] = login.GetCSRF("honkhonk", r)
	honkpage(w, u, honks, templinfo)
}

type Track struct {
	xid string
	who string
}

func getbacktracks(xid string) []string {
	c := make(chan bool)
	dumptracks <- c
	<-c
	row := stmtGetTracks.QueryRow(xid)
	var rawtracks string
	err := row.Scan(&rawtracks)
	if err != nil {
		if err != sql.ErrNoRows {
			elog.Printf("error scanning tracks: %s", err)
		}
		return nil
	}
	var rcpts []string
	for _, f := range strings.Split(rawtracks, " ") {
		idx := strings.LastIndexByte(f, '#')
		if idx != -1 {
			f = f[:idx]
		}
		if !strings.HasPrefix(f, "https://") {
			f = fmt.Sprintf("%%https://%s/inbox", f)
		}
		rcpts = append(rcpts, f)
	}
	return rcpts
}

func savetracks(tracks map[string][]string) {
	db := opendatabase()
	tx, err := db.Begin()
	if err != nil {
		elog.Printf("savetracks begin error: %s", err)
		return
	}
	defer func() {
		err := tx.Commit()
		if err != nil {
			elog.Printf("savetracks commit error: %s", err)
		}

	}()
	stmtGetTracks, err := tx.Prepare("select fetches from tracks where xid = ?")
	if err != nil {
		elog.Printf("savetracks error: %s", err)
		return
	}
	stmtNewTracks, err := tx.Prepare("insert into tracks (xid, fetches) values (?, ?)")
	if err != nil {
		elog.Printf("savetracks error: %s", err)
		return
	}
	stmtUpdateTracks, err := tx.Prepare("update tracks set fetches = ? where xid = ?")
	if err != nil {
		elog.Printf("savetracks error: %s", err)
		return
	}
	count := 0
	for xid, f := range tracks {
		count += len(f)
		var prev string
		row := stmtGetTracks.QueryRow(xid)
		err := row.Scan(&prev)
		if err == sql.ErrNoRows {
			f = oneofakind(f)
			stmtNewTracks.Exec(xid, strings.Join(f, " "))
		} else if err == nil {
			all := append(strings.Split(prev, " "), f...)
			all = oneofakind(all)
			stmtUpdateTracks.Exec(strings.Join(all, " "))
		} else {
			elog.Printf("savetracks error: %s", err)
		}
	}
	dlog.Printf("saved %d new fetches", count)
}

var trackchan = make(chan Track)
var dumptracks = make(chan chan bool)

func tracker() {
	timeout := 4 * time.Minute
	sleeper := time.NewTimer(timeout)
	tracks := make(map[string][]string)
	workinprogress++
	for {
		select {
		case track := <-trackchan:
			tracks[track.xid] = append(tracks[track.xid], track.who)
		case <-sleeper.C:
			if len(tracks) > 0 {
				go savetracks(tracks)
				tracks = make(map[string][]string)
			}
			sleeper.Reset(timeout)
		case c := <-dumptracks:
			if len(tracks) > 0 {
				savetracks(tracks)
			}
			c <- true
		case <-endoftheworld:
			if len(tracks) > 0 {
				savetracks(tracks)
			}
			readyalready <- true
			return
		}
	}
}

var re_keyholder = regexp.MustCompile(`keyId="([^"]+)"`)

func trackback(xid string, r *http.Request) {
	agent := r.UserAgent()
	who := originate(agent)
	sig := r.Header.Get("Signature")
	if sig != "" {
		m := re_keyholder.FindStringSubmatch(sig)
		if len(m) == 2 {
			who = m[1]
		}
	}
	if who != "" {
		trackchan <- Track{xid: xid, who: who}
	}
}

func showonehonk(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	user, err := butwhatabout(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	xid := fmt.Sprintf("https://%s%s", serverName, r.URL.Path)

	if friendorfoe(r.Header.Get("Accept")) {
		j, ok := gimmejonk(xid)
		if ok {
			trackback(xid, r)
			w.Header().Set("Content-Type", theonetruename)
			w.Write(j)
		} else {
			http.NotFound(w, r)
		}
		return
	}
	honk := getxonk(user.ID, xid)
	if honk == nil {
		http.NotFound(w, r)
		return
	}
	u := login.GetUserInfo(r)
	if u != nil && u.UserID != user.ID {
		u = nil
	}
	if !honk.Public {
		if u == nil {
			http.NotFound(w, r)
			return

		}
		honks := []*Honk{honk}
		donksforhonks(honks)
		templinfo := getInfo(r)
		templinfo["ServerMessage"] = "one honk maybe more"
		templinfo["HonkCSRF"] = login.GetCSRF("honkhonk", r)
		honkpage(w, u, honks, templinfo)
		return
	}
	rawhonks := gethonksbyconvoy(honk.UserID, honk.Convoy, 0)
	reversehonks(rawhonks)
	var honks []*Honk
	for _, h := range rawhonks {
		if h.XID == xid && len(honks) != 0 {
			h.Style += " glow"
		}
		if h.Public && h.Whofore == 2 {
			honks = append(honks, h)
		}
	}

	templinfo := getInfo(r)
	templinfo["ServerMessage"] = "one honk maybe more"
	templinfo["HonkCSRF"] = login.GetCSRF("honkhonk", r)
	honkpage(w, u, honks, templinfo)
}

func honkpage(w http.ResponseWriter, u *login.UserInfo, honks []*Honk, templinfo map[string]interface{}) {
	var userid int64 = -1
	if u != nil {
		userid = u.UserID
		templinfo["User"], _ = butwhatabout(u.Username)
	}
	reverbolate(userid, honks)
	templinfo["Honks"] = honks
	if templinfo["TopHID"] == nil {
		if len(honks) > 0 {
			templinfo["TopHID"] = honks[0].ID
		} else {
			templinfo["TopHID"] = 0
		}
	}
	if u == nil && !develMode {
		w.Header().Set("Cache-Control", "max-age=60")
	}
	err := readviews.Execute(w, "honkpage.html", templinfo)
	if err != nil {
		elog.Print(err)
	}
}

func saveuser(w http.ResponseWriter, r *http.Request) {
	whatabout := r.FormValue("whatabout")
	whatabout = strings.Replace(whatabout, "\r", "", -1)
	u := login.GetUserInfo(r)
	user, _ := butwhatabout(u.Username)
	db := opendatabase()

	options := user.Options
	if r.FormValue("mentionall") == "mentionall" {
		options.MentionAll = true
	} else {
		options.MentionAll = false
	}

	sendupdate := false
	whatabout = strings.TrimSpace(whatabout)
	if whatabout != user.About {
		sendupdate = true
	}
	j, err := jsonify(options)
	if err == nil {
		_, err = db.Exec("update users set about = ?, options = ? where username = ?", whatabout, j, u.Username)
	}
	if err != nil {
		elog.Printf("error bouting what: %s", err)
	}
	somenamedusers.Clear(u.Username)
	somenumberedusers.Clear(u.UserID)
	oldjonkers.Clear(u.Username)

	if sendupdate {
		updateMe(u.Username)
	}

	http.Redirect(w, r, "/account", http.StatusSeeOther)
}

func bonkit(xid string, user *WhatAbout) {
	dlog.Printf("bonking %s", xid)

	xonk := getxonk(user.ID, xid)
	if xonk == nil {
		return
	}
	if !xonk.Public {
		return
	}
	if xonk.IsBonked() {
		return
	}
	donksforhonks([]*Honk{xonk})

	_, err := stmtUpdateFlags.Exec(flagIsBonked, xonk.ID)
	if err != nil {
		elog.Printf("error acking bonk: %s", err)
	}

	oonker := xonk.Oonker
	if oonker == "" {
		oonker = xonk.Honker
	}
	dt := time.Now().UTC()
	bonk := &Honk{
		UserID:   user.ID,
		Username: user.Name,
		What:     "bonk",
		Honker:   user.URL,
		Oonker:   oonker,
		XID:      xonk.XID,
		RID:      xonk.RID,
		Noise:    xonk.Noise,
		Precis:   xonk.Precis,
		URL:      xonk.URL,
		Date:     dt,
		Donks:    xonk.Donks,
		Whofore:  2,
		Convoy:   xonk.Convoy,
		Audience: []string{thewholeworld, oonker},
		Public:   true,
		Format:   xonk.Format,
	}

	err = savehonk(bonk)
	if err != nil {
		elog.Printf("uh oh")
		return
	}

	go honkworldwide(user, bonk)
}

func submitbonk(w http.ResponseWriter, r *http.Request) {
	xid := r.FormValue("xid")
	userinfo := login.GetUserInfo(r)
	user, _ := butwhatabout(userinfo.Username)

	bonkit(xid, user)

	if r.FormValue("js") != "1" {
		templinfo := getInfo(r)
		templinfo["ServerMessage"] = "Bonked!"
		err := readviews.Execute(w, "msg.html", templinfo)
		if err != nil {
			elog.Print(err)
		}
	}
}

func sendzonkofsorts(xonk *Honk, user *WhatAbout, what string, aux string) {
	zonk := &Honk{
		What:     what,
		XID:      xonk.XID,
		Date:     time.Now().UTC(),
		Audience: oneofakind(xonk.Audience),
		Noise:    aux,
	}
	zonk.Public = loudandproud(zonk.Audience)

	dlog.Printf("announcing %sed honk: %s", what, xonk.XID)
	go honkworldwide(user, zonk)
}

func zonkit(w http.ResponseWriter, r *http.Request) {
	wherefore := r.FormValue("wherefore")
	what := r.FormValue("what")
	userinfo := login.GetUserInfo(r)
	user, _ := butwhatabout(userinfo.Username)

	// my hammer is too big, oh well
	defer oldjonks.Flush()

	if wherefore == "bonk" {
		user, _ := butwhatabout(userinfo.Username)
		bonkit(what, user)
		return
	}

	if wherefore == "unbonk" {
		xonk := getbonk(userinfo.UserID, what)
		if xonk != nil {
			deletehonk(xonk.ID)
			xonk = getxonk(userinfo.UserID, what)
			_, err := stmtClearFlags.Exec(flagIsBonked, xonk.ID)
			if err != nil {
				elog.Printf("error unbonking: %s", err)
			}
			sendzonkofsorts(xonk, user, "unbonk", "")
		}
		return
	}

	ilog.Printf("zonking %s %s", wherefore, what)
	if wherefore == "zonk" {
		xonk := getxonk(userinfo.UserID, what)
		if xonk != nil {
			deletehonk(xonk.ID)
			if xonk.Whofore == 2 || xonk.Whofore == 3 {
				sendzonkofsorts(xonk, user, "zonk", "")
			}
		}
	}
	_, err := stmtSaveZonker.Exec(userinfo.UserID, what, wherefore)
	if err != nil {
		elog.Printf("error saving zonker: %s", err)
		return
	}
}

func edithonkpage(w http.ResponseWriter, r *http.Request) {
	u := login.GetUserInfo(r)
	user, _ := butwhatabout(u.Username)
	xid := r.FormValue("xid")
	honk := getxonk(u.UserID, xid)
	if !canedithonk(user, honk) {
		http.Error(w, "no editing that please", http.StatusInternalServerError)
		return
	}

	noise := honk.Noise

	honks := []*Honk{honk}
	donksforhonks(honks)
	reverbolate(u.UserID, honks)
	templinfo := getInfo(r)
	templinfo["HonkCSRF"] = login.GetCSRF("honkhonk", r)
	templinfo["Honks"] = honks
	templinfo["Noise"] = noise
	templinfo["ServerMessage"] = "honk edit 2"
	templinfo["IsPreview"] = true
	templinfo["UpdateXID"] = honk.XID
	if len(honk.Donks) > 0 {
		templinfo["SavedFile"] = honk.Donks[0].XID
	}
	err := readviews.Execute(w, "honkpage.html", templinfo)
	if err != nil {
		elog.Print(err)
	}
}

func newhonkpage(w http.ResponseWriter, r *http.Request) {
	u := login.GetUserInfo(r)
	rid := r.FormValue("rid")
	noise := ""

	xonk := getxonk(u.UserID, rid)
	if xonk != nil {
		_, replto := handles(xonk.Honker)
		if replto != "" {
			noise = "@" + replto + " "
		}
	}

	templinfo := getInfo(r)
	templinfo["HonkCSRF"] = login.GetCSRF("honkhonk", r)
	templinfo["InReplyTo"] = rid
	templinfo["Noise"] = noise
	templinfo["ServerMessage"] = "compose honk"
	templinfo["IsPreview"] = true
	err := readviews.Execute(w, "honkpage.html", templinfo)
	if err != nil {
		elog.Print(err)
	}
}

func canedithonk(user *WhatAbout, honk *Honk) bool {
	if honk == nil || honk.Honker != user.URL || honk.What == "bonk" {
		return false
	}
	return true
}

func websubmithonk(w http.ResponseWriter, r *http.Request) {
	h := submithonk(w, r)
	if h == nil {
		return
	}
	http.Redirect(w, r, h.XID[len(serverName)+8:], http.StatusSeeOther)
}

// what a hot mess this function is
func submithonk(w http.ResponseWriter, r *http.Request) *Honk {
	rid := r.FormValue("rid")
	noise := r.FormValue("noise")
	format := r.FormValue("format")
	if format == "" {
		format = "markdown"
	}
	if !(format == "markdown" || format == "html") {
		http.Error(w, "unknown format", 500)
		return nil
	}

	userinfo := login.GetUserInfo(r)
	user, _ := butwhatabout(userinfo.Username)

	dt := time.Now().UTC()
	updatexid := r.FormValue("updatexid")
	var honk *Honk
	if updatexid != "" {
		honk = getxonk(userinfo.UserID, updatexid)
		if !canedithonk(user, honk) {
			http.Error(w, "no editing that please", http.StatusInternalServerError)
			return nil
		}
		honk.Date = dt
		honk.What = "update"
		honk.Format = format
	} else {
		xid := fmt.Sprintf("%s/%s/%s", user.URL, honkSep, xfiltrate())
		what := "honk"
		if rid != "" {
			what = "tonk"
		}
		honk = &Honk{
			UserID:   userinfo.UserID,
			Username: userinfo.Username,
			What:     what,
			Honker:   user.URL,
			XID:      xid,
			Date:     dt,
			Format:   format,
		}
	}

	var convoy string
	noise = strings.Replace(noise, "\r", "", -1)
	if updatexid == "" && rid == "" {
		noise = re_convoy.ReplaceAllStringFunc(noise, func(m string) string {
			convoy = m[7:]
			convoy = strings.TrimSpace(convoy)
			if !re_convalidate.MatchString(convoy) {
				convoy = ""
			}
			return ""
		})
	}
	noise = quickrename(noise, userinfo.UserID)
	honk.Noise = noise
	precipitate(honk)
	noise = honk.Noise
	translate(honk)

	if rid != "" {
		xonk := getxonk(userinfo.UserID, rid)
		if xonk == nil {
			http.Error(w, "replyto disappeared", http.StatusNotFound)
			return nil
		}
		if xonk.Public {
			honk.Audience = append(honk.Audience, xonk.Audience...)
		}
		convoy = xonk.Convoy
		for i, a := range honk.Audience {
			if a == thewholeworld {
				honk.Audience[0], honk.Audience[i] = honk.Audience[i], honk.Audience[0]
				break
			}
		}
		honk.RID = rid
		if xonk.Precis != "" && honk.Precis == "" {
			honk.Precis = xonk.Precis
			if !re_dangerous.MatchString(honk.Precis) {
				honk.Precis = "re: " + honk.Precis
			}
		}
	} else {
		honk.Audience = []string{thewholeworld}
	}
	if honk.Noise != "" && honk.Noise[0] == '@' {
		honk.Audience = append(grapevine(honk.Mentions), honk.Audience...)
	} else {
		honk.Audience = append(honk.Audience, grapevine(honk.Mentions)...)
	}

	if convoy == "" {
		convoy = "data:,electrichonkytonk-" + xfiltrate()
	}
	butnottooloud(honk.Audience)
	honk.Audience = oneofakind(honk.Audience)
	if len(honk.Audience) == 0 {
		ilog.Printf("honk to nowhere")
		http.Error(w, "honk to nowhere...", http.StatusNotFound)
		return nil
	}
	honk.Public = loudandproud(honk.Audience)
	honk.Convoy = convoy

	donkxid := r.FormValue("donkxid")
	if donkxid != "" {
		xid := donkxid
		url := fmt.Sprintf("https://%s/d/%s", serverName, xid)
		donk := finddonk(url)
		if donk != nil {
			honk.Donks = append(honk.Donks, donk)
		} else {
			ilog.Printf("can't find file: %s", xid)
		}
	}

	if honk.Public {
		honk.Whofore = 2
	} else {
		honk.Whofore = 3
	}

	// back to markdown
	honk.Noise = noise

	if r.FormValue("preview") == "preview" {
		honks := []*Honk{honk}
		reverbolate(userinfo.UserID, honks)
		templinfo := getInfo(r)
		templinfo["HonkCSRF"] = login.GetCSRF("honkhonk", r)
		templinfo["Honks"] = honks
		templinfo["InReplyTo"] = r.FormValue("rid")
		templinfo["Noise"] = r.FormValue("noise")
		templinfo["SavedFile"] = donkxid
		templinfo["IsPreview"] = true
		templinfo["UpdateXID"] = updatexid
		templinfo["ServerMessage"] = "honk preview"
		err := readviews.Execute(w, "honkpage.html", templinfo)
		if err != nil {
			elog.Print(err)
		}
		return nil
	}

	if updatexid != "" {
		updatehonk(honk)
		oldjonks.Clear(honk.XID)
	} else {
		err := savehonk(honk)
		if err != nil {
			elog.Printf("uh oh")
			return nil
		}
	}

	// reload for consistency
	honk.Donks = nil
	donksforhonks([]*Honk{honk})

	go honkworldwide(user, honk)

	return honk
}

func showhonkers(w http.ResponseWriter, r *http.Request) {
	userinfo := login.GetUserInfo(r)
	templinfo := getInfo(r)
	templinfo["Honkers"] = gethonkers(userinfo.UserID)
	templinfo["HonkerCSRF"] = login.GetCSRF("submithonker", r)
	err := readviews.Execute(w, "honkers.html", templinfo)
	if err != nil {
		elog.Print(err)
	}
}

func websubmithonker(w http.ResponseWriter, r *http.Request) {
	h := submithonker(w, r)
	if h == nil {
		return
	}
	http.Redirect(w, r, "/honkers", http.StatusSeeOther)
}

func submithonker(w http.ResponseWriter, r *http.Request) *Honker {
	u := login.GetUserInfo(r)
	user, _ := butwhatabout(u.Username)
	name := strings.TrimSpace(r.FormValue("name"))
	url := strings.TrimSpace(r.FormValue("url"))
	peep := r.FormValue("peep")
	combos := ""
	meta := ""
	honkerid, _ := strconv.ParseInt(r.FormValue("honkerid"), 10, 0)

	re_namecheck := regexp.MustCompile("^[\\pL[:digit:]_.-]+$")
	if name != "" && !re_namecheck.MatchString(name) {
		http.Error(w, "please use a plainer name", http.StatusInternalServerError)
		return nil
	}

	defer honkerinvalidator.Clear(u.UserID)

	// mostly dummy, fill in later...
	h := &Honker{
		ID: honkerid,
	}

	if honkerid > 0 {
		if r.FormValue("delete") == "delete" {
			unfollowyou(user, honkerid, false)
			stmtDeleteHonker.Exec(honkerid)
			return h
		}
		if r.FormValue("unsub") == "unsub" {
			unfollowyou(user, honkerid, false)
		}
		if r.FormValue("sub") == "sub" {
			followyou(user, honkerid, false)
		}
		_, err := stmtUpdateHonker.Exec(name, combos, meta, honkerid, u.UserID)
		if err != nil {
			elog.Printf("update honker err: %s", err)
			return nil
		}
		return h
	}

	if url == "" {
		http.Error(w, "subscribing to nothing?", http.StatusInternalServerError)
		return nil
	}

	flavor := "presub"
	if peep == "peep" {
		flavor = "peep"
	}

	var err error
	honkerid, err = savehonker(user, url, name, flavor, combos, meta)
	if err != nil {
		http.Error(w, "had some trouble with that: "+err.Error(), http.StatusInternalServerError)
		return nil
	}

	if flavor == "presub" {
		followyou(user, honkerid, false)
	}
	h.ID = honkerid
	return h
}

func searchxonkers(w http.ResponseWriter, r *http.Request) {
	query := r.FormValue("q")
	results:= getmanyxonkers(query)
	if len(results) == 0 {
		w.WriteHeader(http.StatusNoContent)
	} else {
		w.Write([]byte(results))
	}
}

func hfcspage(w http.ResponseWriter, r *http.Request) {
	userinfo := login.GetUserInfo(r)

	filters := getfilters(userinfo.UserID, filtAny)

	templinfo := getInfo(r)
	templinfo["Filters"] = filters
	templinfo["FilterCSRF"] = login.GetCSRF("filter", r)
	err := readviews.Execute(w, "hfcs.html", templinfo)
	if err != nil {
		elog.Print(err)
	}
}

func savehfcs(w http.ResponseWriter, r *http.Request) {
	userinfo := login.GetUserInfo(r)
	itsok := r.FormValue("itsok")
	if itsok == "iforgiveyou" {
		hfcsid, _ := strconv.ParseInt(r.FormValue("hfcsid"), 10, 0)
		_, err := stmtDeleteFilter.Exec(userinfo.UserID, hfcsid)
		if err != nil {
			elog.Printf("error deleting filter: %s", err)
		}
		filtInvalidator.Clear(userinfo.UserID)
		http.Redirect(w, r, "/hfcs", http.StatusSeeOther)
		return
	}

	filt := new(Filter)
	filt.Actor = strings.TrimSpace(r.FormValue("actor"))
	filt.Text = strings.TrimSpace(r.FormValue("filttext"))
	filt.Reject = true

	if filt.Actor == "" && filt.Text == "" {
		ilog.Printf("blank filter")
		http.Error(w, "can't save a blank filter", http.StatusInternalServerError)
		return
	}

	j, err := jsonify(filt)
	if err == nil {
		_, err = stmtSaveFilter.Exec(userinfo.UserID, j)
	}
	if err != nil {
		elog.Printf("error saving filter: %s", err)
	}

	filtInvalidator.Clear(userinfo.UserID)
	http.Redirect(w, r, "/hfcs", http.StatusSeeOther)
}

func accountpage(w http.ResponseWriter, r *http.Request) {
	u := login.GetUserInfo(r)
	user, _ := butwhatabout(u.Username)
	templinfo := getInfo(r)
	templinfo["UserCSRF"] = login.GetCSRF("saveuser", r)
	templinfo["LogoutCSRF"] = login.GetCSRF("logout", r)
	templinfo["User"] = user
	about := user.About
	templinfo["WhatAbout"] = about
	err := readviews.Execute(w, "account.html", templinfo)
	if err != nil {
		elog.Print(err)
	}
}

func dochpass(w http.ResponseWriter, r *http.Request) {
	err := login.ChangePassword(w, r)
	if err != nil {
		elog.Printf("error changing password: %s", err)
	}
	http.Redirect(w, r, "/account", http.StatusSeeOther)
}

func fingerlicker(w http.ResponseWriter, r *http.Request) {
	orig := r.FormValue("resource")

	dlog.Printf("finger lick: %s", orig)

	if strings.HasPrefix(orig, "acct:") {
		orig = orig[5:]
	}

	name := orig
	idx := strings.LastIndexByte(name, '/')
	if idx != -1 {
		name = name[idx+1:]
		if fmt.Sprintf("https://%s/%s/%s", serverName, userSep, name) != orig {
			ilog.Printf("foreign request rejected")
			name = ""
		}
	} else {
		idx = strings.IndexByte(name, '@')
		if idx != -1 {
			name = name[:idx]
			if !(name+"@"+serverName == orig || name+"@"+masqName == orig) {
				ilog.Printf("foreign request rejected")
				name = ""
			}
		}
	}
	user, err := butwhatabout(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	j := junk.New()
	j["subject"] = fmt.Sprintf("acct:%s@%s", user.Name, masqName)
	j["aliases"] = []string{user.URL}
	l := junk.New()
	l["rel"] = "self"
	l["type"] = `application/activity+json`
	l["href"] = user.URL
	j["links"] = []junk.Junk{l}

	w.Header().Set("Content-Type", "application/jrd+json")
	j.Write(w)
}

func knowninformation(w http.ResponseWriter, r *http.Request) {
	j := junk.New()
	l := junk.New()

	l["rel"] = `http://nodeinfo.diaspora.software/ns/schema/2.0`
	l["href"] = fmt.Sprintf("https://%s/nodeinfo/2.0", serverName)
	j["links"] = []junk.Junk{l}

	w.Header().Set("Content-Type", "application/json")
	j.Write(w)
}

func actualinformation(w http.ResponseWriter, r *http.Request) {
	j := junk.New()

	soft := junk.New()
	soft["name"] = "honk"
	soft["version"] = softwareVersion

	services := junk.New()
	services["inbound"] = []string{}
	services["outbound"] = []string{}

	users := junk.New()
	users["total"] = getusercount()
	users["activeHalfyear"] = getactiveusercount(6)
	users["activeMonth"] = getactiveusercount(1)

	usage := junk.New()
	usage["users"] = users
	usage["localPosts"] = getlocalhonkcount()

	j["version"] = "2.0"
	j["protocols"] = []string{"activitypub"}
	j["software"] = soft
	j["services"] = services
	j["openRegistrations"] = false
	j["usage"] = usage

	w.Header().Set("Content-Type", "application/json")
	j.Write(w)
}

func somedays() string {
	secs := 432000 + notrand.Int63n(432000)
	return fmt.Sprintf("%d", secs)
}

func avatate(w http.ResponseWriter, r *http.Request) {
	if develMode {
		loadAvatarColors()
	}
	n := r.FormValue("a")
	a := genAvatar(n)
	if !develMode {
		w.Header().Set("Cache-Control", "max-age="+somedays())
	}
	w.Write(a)
}

func serveviewasset(w http.ResponseWriter, r *http.Request) {
	serveasset(w, r, viewDir)
}
func servedataasset(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/favicon.ico" {
		r.URL.Path = "/icon.png"
	}
	serveasset(w, r, dataDir)
}

func serveasset(w http.ResponseWriter, r *http.Request, basedir string) {
	if !develMode {
		w.Header().Set("Cache-Control", "max-age=7776000")
	}
	http.ServeFile(w, r, basedir+"/views"+r.URL.Path)
}
func servehtml(w http.ResponseWriter, r *http.Request) {
	u := login.GetUserInfo(r)
	templinfo := getInfo(r)
	templinfo["AboutMsg"] = aboutMsg
	templinfo["LoginMsg"] = loginMsg
	templinfo["HonkVersion"] = softwareVersion
	if r.URL.Path == "/about" {
		templinfo["Sensors"] = getSensors()
	}
	if u == nil && !develMode {
		w.Header().Set("Cache-Control", "max-age=60")
	}
	err := readviews.Execute(w, r.URL.Path[1:]+".html", templinfo)
	if err != nil {
		elog.Print(err)
	}
}

func servefile(w http.ResponseWriter, r *http.Request) {
	xid := mux.Vars(r)["xid"]
	var media string
	var data []byte
	row := stmtGetFileData.QueryRow(xid)
	err := row.Scan(&media, &data)
	if err != nil {
		elog.Printf("error loading file: %s", err)
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", media)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "max-age="+somedays())
	w.Write(data)
}

func nomoroboto(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "User-agent: *\n")
	io.WriteString(w, "Disallow: /a\n")
	io.WriteString(w, "Disallow: /d/\n")
	io.WriteString(w, "Disallow: /meme/\n")
	io.WriteString(w, "Disallow: /o\n")
	io.WriteString(w, "Disallow: /o/\n")
	io.WriteString(w, "Disallow: /help/\n")
	for _, u := range allusers() {
		fmt.Fprintf(w, "Disallow: /%s/%s/%s/\n", userSep, u.Username, honkSep)
	}
}

type Hydration struct {
	Tophid    int64
	Srvmsg    template.HTML
	Honks     string
	MeCount   int64
	ChatCount int64
}

func webhydra(w http.ResponseWriter, r *http.Request) {
	u := login.GetUserInfo(r)
	userid := u.UserID
	templinfo := getInfo(r)
	templinfo["HonkCSRF"] = login.GetCSRF("honkhonk", r)
	page := r.FormValue("page")

	wanted, _ := strconv.ParseInt(r.FormValue("tophid"), 10, 0)

	var hydra Hydration

	var honks []*Honk
	switch page {
	case "atme":
		honks = gethonksforme(userid, wanted)
		menewnone(userid)
		hydra.Srvmsg = "at me!"
	case "home":
		honks = gethonksforuser(userid, wanted)
		hydra.Srvmsg = serverMsg
	case "convoy":
		c := r.FormValue("c")
		honks = gethonksbyconvoy(userid, c, wanted)
		hydra.Srvmsg = templates.Sprintf("honks in skein: %s", c)
	case "honker":
		xid := r.FormValue("xid")
		honks = gethonksbyxonker(userid, xid, wanted)
		miniform := templates.Sprintf(`<form action="/submithonker" method="POST">
			<input type="hidden" name="CSRF" value="%s">
			<input type="hidden" name="url" value="%s">
			<button tabindex=1 name="add honker" value="add honker">add honker</button>
			</form>`, login.GetCSRF("submithonker", r), xid)
		msg := templates.Sprintf(`honks by honker: <a href="%s" ref="noreferrer">%s</a>%s`, xid, xid, miniform)
		hydra.Srvmsg = msg
	case "user":
		uname := r.FormValue("uname")
		honks = gethonksbyuser(uname, u != nil && u.Username == uname, wanted)
		hydra.Srvmsg = templates.Sprintf("honks by user: %s", uname)
	default:
		http.NotFound(w, r)
	}

	if len(honks) > 0 {
		hydra.Tophid = honks[0].ID
	} else {
		hydra.Tophid = wanted
	}
	reverbolate(userid, honks)

	user, _ := butwhatabout(u.Username)

	var buf strings.Builder
	templinfo["Honks"] = honks
	templinfo["User"], _ = butwhatabout(u.Username)
	err := readviews.Execute(&buf, "honkfrags.html", templinfo)
	if err != nil {
		elog.Printf("frag error: %s", err)
		return
	}
	hydra.Honks = buf.String()
	hydra.MeCount = user.Options.MeCount
	w.Header().Set("Content-Type", "application/json")
	j, _ := jsonify(&hydra)
	io.WriteString(w, j)
}

var honkline = make(chan bool)

func honkhonkline() {
	for {
		select {
		case honkline <- true:
		default:
			return
		}
	}
}

func apihandler(w http.ResponseWriter, r *http.Request) {
	u := login.GetUserInfo(r)
	userid := u.UserID
	action := r.FormValue("action")
	wait, _ := strconv.ParseInt(r.FormValue("wait"), 10, 0)
	dlog.Printf("api request '%s' on behalf of %s", action, u.Username)
	switch action {
	case "honk":
		h := submithonk(w, r)
		if h == nil {
			return
		}
		fmt.Fprintf(w, "%s", h.XID)
	case "donk":
		http.Error(w, "donks are not implemented on this server", http.StatusBadRequest)
	case "zonkit":
		zonkit(w, r)
	case "gethonks":
		var honks []*Honk
		wanted, _ := strconv.ParseInt(r.FormValue("after"), 10, 0)
		page := r.FormValue("page")
		var waitchan <-chan time.Time
	requery:
		switch page {
		case "atme":
			honks = gethonksforme(userid, wanted)
			menewnone(userid)
		case "longago":
			honks = gethonksfromlongago(userid, wanted)
		case "home":
			honks = gethonksforuser(userid, wanted)
		case "myhonks":
			honks = gethonksbyuser(u.Username, true, wanted)
		default:
			http.Error(w, "unknown page", http.StatusNotFound)
			return
		}
		if len(honks) == 0 && wait > 0 {
			if waitchan == nil {
				waitchan = time.After(time.Duration(wait) * time.Second)
			}
			select {
			case <-honkline:
				goto requery
			case <-waitchan:
			}
		}
		reverbolate(userid, honks)
		j := junk.New()
		j["honks"] = honks
		j.Write(w)
	case "sendactivity":
		user, _ := butwhatabout(u.Username)
		public := r.FormValue("public") == "1"
		rcpts := boxuprcpts(user, r.Form["rcpt"], public)
		msg := []byte(r.FormValue("msg"))
		for rcpt := range rcpts {
			go deliverate(0, userid, rcpt, msg, true)
		}
	case "gethonkers":
		j := junk.New()
		j["honkers"] = gethonkers(u.UserID)
		j.Write(w)
	case "savehonker":
		h := submithonker(w, r)
		if h == nil {
			return
		}
		fmt.Fprintf(w, "%d", h.ID)
	default:
		http.Error(w, "unknown action", http.StatusNotFound)
		return
	}
}


var endoftheworld = make(chan bool)
var readyalready = make(chan bool)
var workinprogress = 0

func enditall() {
	sig := make(chan os.Signal)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	<-sig
	ilog.Printf("stopping...")
	for i := 0; i < workinprogress; i++ {
		endoftheworld <- true
	}
	ilog.Printf("waiting...")
	for i := 0; i < workinprogress; i++ {
		<-readyalready
	}
	ilog.Printf("apocalypse")
	os.Exit(0)
}

var preservehooks []func()

func bgmonitor() {
	for {
		when := time.Now().Add(-3 * 24 * time.Hour).UTC().Format(dbtimeformat)
		_, err := stmtDeleteOldXonkers.Exec("pubkey", when)
		if err != nil {
			elog.Printf("error deleting old xonkers: %s", err)
		}
		zaggies.Flush()
		time.Sleep(50 * time.Minute)
	}
}


func addcspheaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'self'; connect-src 'self'; style-src 'self'; img-src 'self'; manifest-src 'self'; report-uri /csp-violation")
		next.ServeHTTP(w, r)
	})
}

func serve() {
	db := opendatabase()
	login.Init(login.InitArgs{Db: db, Logger: ilog, Insecure: develMode, SameSiteStrict: !develMode})

	listener, err := openListener()
	if err != nil {
		elog.Fatal(err)
	}
	go enditall()
	go redeliverator()
	go tracker()
	go bgmonitor()
	loadLingo()

	readviews = templates.Load(develMode,
		viewDir+"/views/hfcs.html",
		viewDir+"/views/honkpage.html",
		viewDir+"/views/honkfrags.html",
		viewDir+"/views/honkers.html",
		viewDir+"/views/honkform.html",
		viewDir+"/views/honk.html",
		viewDir+"/views/account.html",
		viewDir+"/views/about.html",
		viewDir+"/views/login.html",
		viewDir+"/views/msg.html",
		viewDir+"/views/header.html",
		viewDir+"/views/honkpage.js",
	)
	if !develMode {
		assets := []string{
			viewDir + "/views/style.css",
			dataDir + "/views/local.css",
			dataDir + "/views/guest.css",
			viewDir + "/views/honkpage.js",
			dataDir + "/views/local.js",
			viewDir + "/views/manifest.webmanifest",
		}
		for _, s := range assets {
			savedassetparams[s] = getassetparam(s)
		}
		loadAvatarColors()
	}

	for _, h := range preservehooks {
		h()
	}

	mux := mux.NewRouter()
	mux.Use(addcspheaders)
	mux.Use(login.Checker)

	mux.Handle("/api", login.TokenRequired(http.HandlerFunc(apihandler)))

	posters := mux.Methods("POST").Subrouter()
	getters := mux.Methods("GET").Subrouter()

	getters.HandleFunc("/", homepage)
	getters.HandleFunc("/home", homepage)
	getters.HandleFunc("/robots.txt", nomoroboto)
	getters.HandleFunc("/"+userSep+"/{name:[\\pL[:digit:]]+}", showuser)
	getters.HandleFunc("/"+userSep+"/{name:[\\pL[:digit:]]+}/"+honkSep+"/{xid:[\\pL[:digit:]]+}", showonehonk)
	posters.HandleFunc("/"+userSep+"/{name:[\\pL[:digit:]]+}/inbox", inbox)
	getters.HandleFunc("/"+userSep+"/{name:[\\pL[:digit:]]+}/outbox", outbox)
	getters.HandleFunc("/"+userSep+"/{name:[\\pL[:digit:]]+}/followers", emptiness)
	getters.HandleFunc("/"+userSep+"/{name:[\\pL[:digit:]]+}/following", emptiness)
	getters.HandleFunc("/a", avatate)
	getters.HandleFunc("/d/{xid:[\\pL[:digit:].]+}", servefile)
	getters.HandleFunc("/.well-known/webfinger", fingerlicker)
	getters.HandleFunc("/.well-known/nodeinfo", knowninformation)
	getters.HandleFunc("/nodeinfo/2.0", actualinformation)

	getters.HandleFunc("/server", serveractor)
	posters.HandleFunc("/server/inbox", serverinbox)
	posters.HandleFunc("/inbox", serverinbox)

	getters.HandleFunc("/style.css", serveviewasset)
	getters.HandleFunc("/sw.js", serveviewasset)
	getters.HandleFunc("/honkpage.js", serveviewasset)
	getters.HandleFunc("/local.css", servedataasset)
	getters.HandleFunc("/local.js", servedataasset)
	getters.HandleFunc("/icon.png", servedataasset)
	getters.HandleFunc("/background.webp", servedataasset)
	getters.HandleFunc("/guest.css", servedataasset)
	getters.HandleFunc("/favicon.ico", servedataasset)
	getters.HandleFunc("/manifest.webmanifest", serveviewasset)

	getters.HandleFunc("/about", servehtml)
	getters.HandleFunc("/login", servehtml)
	posters.HandleFunc("/dologin", login.LoginFunc)
	getters.HandleFunc("/logout", login.LogoutFunc)

	loggedin := mux.NewRoute().Subrouter()
	loggedin.Use(login.Required)
	loggedin.HandleFunc("/account", accountpage)
	loggedin.HandleFunc("/chpass", dochpass)
	loggedin.HandleFunc("/atme", homepage)
	loggedin.HandleFunc("/hfcs", hfcspage)
	loggedin.HandleFunc("/newhonk", newhonkpage)
	loggedin.HandleFunc("/edit", edithonkpage)
	loggedin.Handle("/honk", login.CSRFWrap("honkhonk", http.HandlerFunc(websubmithonk)))
	loggedin.Handle("/bonk", login.CSRFWrap("honkhonk", http.HandlerFunc(submitbonk)))
	loggedin.Handle("/zonkit", login.CSRFWrap("honkhonk", http.HandlerFunc(zonkit)))
	loggedin.Handle("/saveuser", login.CSRFWrap("saveuser", http.HandlerFunc(saveuser)))
	loggedin.Handle("/savehfcs", login.CSRFWrap("filter", http.HandlerFunc(savehfcs)))
	loggedin.HandleFunc("/honkers", showhonkers)
	loggedin.HandleFunc("/searchxonkers", searchxonkers)
	loggedin.HandleFunc("/h/{name:[\\pL[:digit:]_.-]+}", showhonker)
	loggedin.HandleFunc("/h", showhonker)
	loggedin.HandleFunc("/t", showconvoy)
	loggedin.HandleFunc("/q", showsearch)
	loggedin.HandleFunc("/hydra", webhydra)
	loggedin.Handle("/submithonker", login.CSRFWrap("submithonker", http.HandlerFunc(websubmithonker)))

	err = http.Serve(listener, mux)
	if err != nil {
		elog.Fatal(err)
	}
}
