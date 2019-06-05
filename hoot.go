package main

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/andybalholm/cascadia"
	"golang.org/x/net/html"
	"humungus.tedunangst.com/r/webs/htfilter"
)

var tweetsel = cascadia.MustCompile("p.tweet-text")
var linksel = cascadia.MustCompile(".time a.tweet-timestamp")
var authorregex = regexp.MustCompile("twitter.com/([^/]+)")

func hootfixer(hoot string) string {
	url := hoot[5:]
	if url[0] == ' ' {
		url = url[1:]
	}
	url = strings.ReplaceAll(url, "mobile.twitter.com", "twitter.com")
	log.Printf("hooterizing %s", url)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("error: %s", err)
		return hoot
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; CrOS x86_64 11021.56.0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/70.0.3538.76 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("error: %s", err)
		return hoot
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Printf("error getting %s: %d", url, resp.StatusCode)
		return hoot
	}

	root, _ := html.Parse(resp.Body)
	divs := tweetsel.MatchAll(root)

	authormatch := authorregex.FindStringSubmatch(url)
	if len(authormatch) < 2 {
		log.Printf("no author")
		return hoot
	}
	wanted := authormatch[1]
	var buf strings.Builder

	fmt.Fprintf(&buf, "hoot: %s\n", url)
	for _, div := range divs {
		twp := div.Parent.Parent.Parent
		alink := linksel.MatchFirst(twp)
		if alink == nil {
			log.Printf("missing link")
			continue
		}
		link := "https://twitter.com" + htfilter.GetAttr(alink, "href")
		authormatch = authorregex.FindStringSubmatch(link)
		if len(authormatch) < 2 {
			log.Printf("no author?")
			continue
		}
		author := authormatch[1]
		if author != wanted {
			continue
		}
		text := htfilter.TextOnly(div)
		text = strings.ReplaceAll(text, "\n", " ")
		text = strings.ReplaceAll(text, "pic.twitter.com", "https://pic.twitter.com")

		fmt.Fprintf(&buf, "> @%s: %s\n", author, text)
	}
	return buf.String()
}

var re_hoots = regexp.MustCompile(`hoot: ?https://\S+`)

func hooterize(noise string) string {
	return re_hoots.ReplaceAllStringFunc(noise, hootfixer)
}
