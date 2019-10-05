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
	"net/http"
	"regexp"
)

type Filter struct {
	ID              int64
	Actor           string
	IncludeAudience bool
	Text            string
	IsAnnounce      bool
	Reject          bool
	SkipMedia       bool
	Hide            bool
	Collapse        bool
	Rewrite         string
	re_rewrite      *regexp.Regexp
	Replace         string
}

type filtType uint

const (
	filtNone filtType = iota
	filtAny
	filtReject
	filtSkipMedia
	filtHide
	filtCollapse
	filtRewrite
)

var filtNames = []string{"None", "Any", "Reject", "SkipMedia", "Hide", "Collapse", "Rewrite"}

func (ft filtType) String() string {
	return filtNames[ft]
}

type afiltermap map[filtType][]*Filter

var filtcache = cacheNew(func(userid int64) (afiltermap, bool) {
	rows, err := stmtGetFilters.Query(userid)
	if err != nil {
		log.Printf("error querying filters: %s", err)
		return nil, false
	}
	defer rows.Close()

	filtmap := make(afiltermap)
	for rows.Next() {
		filt := new(Filter)
		var j string
		var filterid int64
		err = rows.Scan(&filterid, &j)
		if err == nil {
			err = unjsonify(j, filt)
		}
		if err != nil {
			log.Printf("error scanning filter: %s", err)
			continue
		}
		filt.ID = filterid
		if filt.Reject {
			filtmap[filtReject] = append(filtmap[filtReject], filt)
		}
		if filt.SkipMedia {
			filtmap[filtSkipMedia] = append(filtmap[filtSkipMedia], filt)
		}
		if filt.Hide {
			filtmap[filtHide] = append(filtmap[filtHide], filt)
		}
		if filt.Collapse {
			filtmap[filtCollapse] = append(filtmap[filtCollapse], filt)
		}
		if filt.Rewrite != "" {
			filtmap[filtRewrite] = append(filtmap[filtRewrite], filt)
		}
	}
	return filtmap, true
})

func getfilters(userid int64, scope filtType) []*Filter {
	var filtmap afiltermap
	ok := filtcache.Get(userid, &filtmap)
	if ok {
		return filtmap[scope]
	}
	return nil
}

func rejectorigin(userid int64, origin string) bool {
	if o := originate(origin); o != "" {
		origin = o
	}
	filts := getfilters(userid, filtReject)
	for _, f := range filts {
		if f.Actor == origin {
			log.Printf("rejecting origin: %s", origin)
			return true
		}
	}
	return false
}

func rejectactor(userid int64, actor string) bool {
	origin := originate(actor)
	filts := getfilters(userid, filtReject)
	for _, f := range filts {
		if f.Actor == actor || (origin != "" && f.Actor == origin) {
			log.Printf("rejecting actor: %s", actor)
			return true
		}
	}
	return false
}

func stealthmode(userid int64, r *http.Request) bool {
	agent := r.UserAgent()
	agent = originate(agent)
	if agent != "" {
		fake := rejectorigin(userid, agent)
		if fake {
			log.Printf("faking 404 for %s", agent)
			return fake
		}
	}
	return false
}

// todo
func matchfilter(h *Honk, f *Filter) bool {
	match := true
	if match && f.Actor != "" {
		match = false
		if f.Actor == h.Honker || f.Actor == h.Oonker {
			match = true
		}
		if !match && (f.Actor == originate(h.Honker) ||
			f.Actor == originate(h.Oonker) ||
			f.Actor == originate(h.XID)) {
			match = true
		}
		if !match && f.IncludeAudience {
			for _, a := range h.Audience {
				if f.Actor == a || f.Actor == originate(a) {
					match = true
					break
				}
			}
		}
	}
	if match && f.Text != "" {
		match = false
		for _, d := range h.Donks {
			if d.Desc == f.Text {
				match = true
			}
		}
	}
	if match {
		return true
	}
	return false
}

func rejectxonk(xonk *Honk) bool {
	filts := getfilters(xonk.UserID, filtReject)
	for _, f := range filts {
		if matchfilter(xonk, f) {
			return true
		}
	}
	return false
}

func skipMedia(xonk *Honk) bool {
	filts := getfilters(xonk.UserID, filtSkipMedia)
	for _, f := range filts {
		if matchfilter(xonk, f) {
			return true
		}
	}
	return false
}

// todo
func unsee(filts []*Filter, h *Honk) string {
	return ""
}

func osmosis(honks []*Honk, userid int64) []*Honk {
	filts := getfilters(userid, filtHide)
	j := 0
outer:
	for _, h := range honks {
		for _, f := range filts {
			if matchfilter(h, f) {
				continue outer
			}
		}
		honks[j] = h
		j++
	}
	honks = honks[0:j]
	return honks
}
