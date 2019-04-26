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
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"
)

func sb64(data []byte) string {
	var sb strings.Builder
	b64 := base64.NewEncoder(base64.StdEncoding, &sb)
	b64.Write(data)
	b64.Close()
	return sb.String()

}
func b64s(s string) []byte {
	var buf bytes.Buffer
	b64 := base64.NewDecoder(base64.StdEncoding, strings.NewReader(s))
	io.Copy(&buf, b64)
	return buf.Bytes()
}
func sb64sha256(content []byte) string {
	h := sha256.New()
	h.Write(content)
	return sb64(h.Sum(nil))
}

func zig(keyname string, key *rsa.PrivateKey, req *http.Request, content []byte) {
	headers := []string{"(request-target)", "date", "host", "content-type", "digest"}
	var stuff []string
	for _, h := range headers {
		var s string
		switch h {
		case "(request-target)":
			s = strings.ToLower(req.Method) + " " + req.URL.RequestURI()
		case "date":
			s = req.Header.Get(h)
			if s == "" {
				s = time.Now().UTC().Format(http.TimeFormat)
				req.Header.Set(h, s)
			}
		case "host":
			s = req.Header.Get(h)
			if s == "" {
				s = req.URL.Hostname()
				req.Header.Set(h, s)
			}
		case "content-type":
			s = req.Header.Get(h)
		case "digest":
			s = req.Header.Get(h)
			if s == "" {
				s = "SHA-256=" + sb64sha256(content)
				req.Header.Set(h, s)
			}
		}
		stuff = append(stuff, h+": "+s)
	}

	h := sha256.New()
	h.Write([]byte(strings.Join(stuff, "\n")))
	sig, _ := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, h.Sum(nil))
	bsig := sb64(sig)

	sighdr := fmt.Sprintf(`keyId="%s",algorithm="%s",headers="%s",signature="%s"`,
		keyname, "rsa-sha256", strings.Join(headers, " "), bsig)
	req.Header.Set("Signature", sighdr)
}

var re_sighdrval = regexp.MustCompile(`(.*)="(.*)"`)

func zag(req *http.Request, content []byte) (string, error) {
	sighdr := req.Header.Get("Signature")

	var keyname, algo, heads, bsig string
	for _, v := range strings.Split(sighdr, ",") {
		m := re_sighdrval.FindStringSubmatch(v)
		if len(m) != 3 {
			return "", fmt.Errorf("bad scan: %s from %s\n", v, sighdr)
		}
		switch m[1] {
		case "keyId":
			keyname = m[2]
		case "algorithm":
			algo = m[2]
		case "headers":
			heads = m[2]
		case "signature":
			bsig = m[2]
		default:
			return "", fmt.Errorf("bad sig val: %s", m[1])
		}
	}
	if keyname == "" || algo == "" || heads == "" || bsig == "" {
		return "", fmt.Errorf("missing a sig value")
	}

	key := zaggy(keyname)
	if key == nil {
		return keyname, fmt.Errorf("no key for %s", keyname)
	}
	headers := strings.Split(heads, " ")
	var stuff []string
	for _, h := range headers {
		var s string
		switch h {
		case "(request-target)":
			s = strings.ToLower(req.Method) + " " + req.URL.RequestURI()
		case "host":
			s = req.Host
			if s != serverName {
				log.Printf("caution: servername host header mismatch")
			}
		default:
			s = req.Header.Get(h)
		}
		stuff = append(stuff, h+": "+s)
	}

	h := sha256.New()
	h.Write([]byte(strings.Join(stuff, "\n")))
	sig := b64s(bsig)
	err := rsa.VerifyPKCS1v15(key, crypto.SHA256, h.Sum(nil), sig)
	if err != nil {
		return keyname, err
	}
	return keyname, nil
}

func pez(s string) (pri *rsa.PrivateKey, pub *rsa.PublicKey, err error) {
	block, _ := pem.Decode([]byte(s))
	if block == nil {
		err = fmt.Errorf("no pem data")
		return
	}
	switch block.Type {
	case "PUBLIC KEY":
		var k interface{}
		k, err = x509.ParsePKIXPublicKey(block.Bytes)
		if k != nil {
			pub, _ = k.(*rsa.PublicKey)
		}
	case "RSA PUBLIC KEY":
		pub, err = x509.ParsePKCS1PublicKey(block.Bytes)
	case "RSA PRIVATE KEY":
		pri, err = x509.ParsePKCS1PrivateKey(block.Bytes)
		if err == nil {
			pub = &pri.PublicKey
		}
	default:
		err = fmt.Errorf("unknown key type")
	}
	return
}

func zem(i interface{}) (string, error) {
	var b pem.Block
	var err error
	switch k := i.(type) {
	case *rsa.PrivateKey:
		b.Type = "RSA PRIVATE KEY"
		b.Bytes = x509.MarshalPKCS1PrivateKey(k)
	case *rsa.PublicKey:
		b.Type = "PUBLIC KEY"
		b.Bytes, err = x509.MarshalPKIXPublicKey(k)
	default:
		err = fmt.Errorf("unknown key type: %s", k)
	}
	if err != nil {
		return "", err
	}
	return string(pem.EncodeToMemory(&b)), nil
}
