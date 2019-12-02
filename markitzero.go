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
	"fmt"
	"regexp"
	"strings"

	"humungus.tedunangst.com/r/webs/synlight"
)

var re_bolder = regexp.MustCompile(`(^|\W)\*\*((?s:.*?))\*\*($|\W)`)
var re_italicer = regexp.MustCompile(`(^|\W)\*((?s:.*?))\*($|\W)`)
var re_bigcoder = regexp.MustCompile("```(.*)\n?((?s:.*?))\n?```\n?")
var re_coder = regexp.MustCompile("`([^`]*)`")
var re_quoter = regexp.MustCompile(`(?m:^&gt; (.*)(\n- ?(.*))?\n?)`)
var re_reciter = regexp.MustCompile(`(<cite><a href=".*?">)https://twitter.com/([^/]+)/.*?(</a></cite>)`)
var re_link = regexp.MustCompile(`.?.?https?://[^\s"]+[\w/)!]`)
var re_zerolink = regexp.MustCompile(`\[([^]]*)\]\(([^)]*\)?)\)`)
var re_imgfix = regexp.MustCompile(`<img ([^>]*)>`)
var re_lister = regexp.MustCompile(`((^|\n)(\+|-).*)+\n?`)

var lighter = synlight.New(synlight.Options{Format: synlight.HTML})

// fewer side effects than html.EscapeString
func fasterescaper(s []byte) []byte {
	buf := make([]byte, 0, len(s))
	for _, c := range []byte(s) {
		switch c {
		case '&':
			buf = append(buf, []byte("&amp;")...)
		case '<':
			buf = append(buf, []byte("&lt;")...)
		case '>':
			buf = append(buf, []byte("&gt;")...)
		default:
			buf = append(buf, c)
		}
	}
	return buf
}

func replaceifmatch(re *regexp.Regexp, input []byte, repl []byte) []byte {
	if !re.Match(input) {
		return input
	}
	return re.ReplaceAll(input, repl)
}

func replaceifmatchfn(re *regexp.Regexp, input []byte, repl func([]byte) []byte) []byte {
	if !re.Match(input) {
		return input
	}
	return re.ReplaceAllFunc(input, repl)
}

func replacenocopy(input []byte, pat []byte, repl []byte) []byte {
	if !bytes.Contains(input, pat) {
		return input
	}
	return bytes.Replace(input, pat, repl, -1)
}

func markitzero(ss string) string {
	s := []byte(ss)
	// prepare the string
	s = bytes.TrimSpace(s)
	s = replacenocopy(s, []byte("\r"), []byte(""))

	hascode := bytes.Contains(s, []byte("`"))

	// save away the code blocks so we don't mess them up further
	var bigcodes, lilcodes, images [][]byte
	if hascode {
		s = replaceifmatchfn(re_bigcoder, s, func(code []byte) []byte {
			bigcodes = append(bigcodes, code)
			return []byte("``````")
		})
		s = replaceifmatchfn(re_coder, s, func(code []byte) []byte {
			lilcodes = append(lilcodes, code)
			return []byte("`x`")
		})
	}
	s = replaceifmatchfn(re_imgfix, s, func(img []byte) []byte {
		images = append(images, img)
		return []byte("<img x>")
	})

	s = fasterescaper(s)

	// mark it zero
	if bytes.Contains(s, []byte("http")) {
		s = replaceifmatchfn(re_link, s, linkreplacer)
	}
	s = replaceifmatch(re_zerolink, s, []byte(`<a href="$2">$1</a>`))
	if bytes.Contains(s, []byte("**")) {
		s = replaceifmatch(re_bolder, s, []byte("$1<b>$2</b>$3"))
	}
	if bytes.Contains(s, []byte("*")) {
		s = replaceifmatch(re_italicer, s, []byte("$1<i>$2</i>$3"))
	}
	if bytes.Contains(s, []byte("&gt; ")) {
		s = replaceifmatch(re_quoter, s, []byte("<blockquote>$1<br><cite>$3</cite></blockquote><p>"))
		s = replaceifmatch(re_reciter, s, []byte("$1$2$3"))
	}
	s = replacenocopy(s, []byte("\n---\n"), []byte("<hr><p>"))

	if bytes.Contains(s, []byte("\n+")) || bytes.Contains(s, []byte("\n-")) {
		s = replaceifmatchfn(re_lister, s, func(m []byte) []byte {
			m = bytes.Trim(m, "\n")
			items := bytes.Split(m, []byte("\n"))
			r := []byte("<ul>")
			for _, item := range items {
				r = append(r, []byte("<li>")...)
				r = append(r, bytes.Trim(item[1:], " ")...)
			}
			r = append(r, []byte("</ul><p>")...)
			return r
		})
	}

	// restore images
	s = replacenocopy(s, []byte("&lt;img x&gt;"), []byte("<img x>"))
	s = replaceifmatchfn(re_imgfix, s, func([]byte) []byte {
		img := images[0]
		images = images[1:]
		return img
	})

	// now restore the code blocks
	if hascode {
		s = replaceifmatchfn(re_coder, s, func([]byte) []byte {
			code := lilcodes[0]
			lilcodes = lilcodes[1:]
			return fasterescaper(code)
		})
		s = replaceifmatchfn(re_bigcoder, s, func([]byte) []byte {
			code := bigcodes[0]
			bigcodes = bigcodes[1:]
			m := re_bigcoder.FindSubmatch(code)
			var buf bytes.Buffer
			buf.WriteString("<pre><code>")
			lighter.Highlight(m[2], string(m[1]), &buf)
			buf.WriteString("</code></pre><p>")
			return buf.Bytes()
		})
		s = replaceifmatch(re_coder, s, []byte("<code>$1</code>"))
	}

	// some final fixups
	s = replacenocopy(s, []byte("\n"), []byte("<br>"))
	s = replacenocopy(s, []byte("<br><blockquote>"), []byte("<blockquote>"))
	s = replacenocopy(s, []byte("<br><cite></cite>"), []byte(""))
	s = replacenocopy(s, []byte("<br><pre>"), []byte("<pre>"))
	s = replacenocopy(s, []byte("<br><ul>"), []byte("<ul>"))
	s = replacenocopy(s, []byte("<p><br>"), []byte("<p>"))
	return string(s)
}

func linkreplacer(burl []byte) []byte {
	url := string(burl)
	if url[0:2] == "](" {
		return burl
	}
	prefix := ""
	for !strings.HasPrefix(url, "http") {
		prefix += url[0:1]
		url = url[1:]
	}
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
	return []byte(prefix + url)
}
