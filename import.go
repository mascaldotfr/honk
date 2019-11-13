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
	"encoding/json"
	"fmt"
	"html"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strings"
	"time"
)

func importMain(username, flavor, source string) {
	switch flavor {
	case "twitter":
		importTwitter(username, source)
	default:
		log.Fatal("unknown source flavor")
	}
}

func importTwitter(username, source string) {
	user, err := butwhatabout(username)
	if err != nil {
		log.Fatal(err)
	}

	type Tweet struct {
		ID_str                  string
		Created_at              string
		Full_text               string
		In_reply_to_screen_name string
		Entities                struct {
			Hashtags []struct {
				Text string
			}
			Media []struct {
				Url       string
				Media_url string
			}
			Urls []struct {
				Url          string
				Expanded_url string
			}
		}
		date time.Time
		text string
	}

	var tweets []*Tweet
	fd, err := os.Open(source + "/tweet.js")
	if err != nil {
		log.Fatal(err)
	}
	fd.Seek(25, 0)
	dec := json.NewDecoder(fd)
	err = dec.Decode(&tweets)
	if err != nil {
		log.Fatalf("error parsing json: %s", err)
	}
	fd.Close()
	for _, t := range tweets {
		t.date, _ = time.Parse("Mon Jan 02 15:04:05 -0700 2006", t.Created_at)
	}
	sort.Slice(tweets, func(i, j int) bool {
		return tweets[i].date.Before(tweets[j].date)
	})
	for _, t := range tweets {
		what := "honk"
		if t.In_reply_to_screen_name != "" {
			what = "tonk"
		}
		audience := []string{thewholeworld}
		xid := fmt.Sprintf("%s/%s/%s", user.URL, honkSep, xfiltrate())
		honk := Honk{
			UserID:   user.ID,
			Username: user.Name,
			What:     what,
			Honker:   user.URL,
			XID:      xid,
			Date:     t.date,
			Format:   "markdown",
			Audience: audience,
			Convoy:   "data:,acoustichonkytonk-" + xfiltrate(),
			Public:   true,
			Whofore:  2,
		}
		noise := t.Full_text
		// unbelievable
		noise = html.UnescapeString(noise)
		for _, r := range t.Entities.Urls {
			noise = strings.Replace(noise, r.Url, r.Expanded_url, -1)
		}
		for _, m := range t.Entities.Media {
			u := m.Media_url
			idx := strings.LastIndexByte(u, '/')
			u = u[idx+1:]
			fname := fmt.Sprintf("%s/tweet_media/%s-%s", source, t.ID_str, u)
			data, err := ioutil.ReadFile(fname)
			if err != nil {
				log.Printf("error reading media: %s", fname)
				continue
			}
			newurl := fmt.Sprintf("https://%s/d/%s", serverName, u)

			fileid, err := savefile(u, u, u, newurl, "image/jpg", true, data)
			if err != nil {
				log.Printf("error saving media: %s", fname)
				continue
			}
			donk := &Donk{
				FileID: fileid,
			}
			honk.Donks = append(honk.Donks, donk)
			noise = strings.Replace(noise, m.Url, "", -1)
		}
		for _, ht := range t.Entities.Hashtags {
			honk.Onts = append(honk.Onts, "#"+ht.Text)
		}
		honk.Noise = noise
		savehonk(&honk)
	}
}
