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
	"context"
	"crypto/rand"
	"crypto/sha512"
	"crypto/subtle"
	"database/sql"
	"fmt"
	"hash"
	"io"
	"log"
	"net/http"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type keytype struct{}

var thekey keytype

func LoginChecker(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userinfo, ok := checkauthcookie(r)
		if ok {
			ctx := context.WithValue(r.Context(), thekey, userinfo)
			r = r.WithContext(ctx)
		}
		handler.ServeHTTP(w, r)
	})
}

func LoginRequired(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ok := GetUserInfo(r) != nil
		if !ok {
			loginredirect(w, r)
			return
		}
		handler.ServeHTTP(w, r)
	})
}

func GetUserInfo(r *http.Request) *UserInfo {
	userinfo, ok := r.Context().Value(thekey).(*UserInfo)
	if !ok {
		return nil
	}
	return userinfo
}

func calculateCSRF(salt, action, auth string) string {
	hasher := sha512.New512_256()
	zero := []byte{0}
	hasher.Write(zero)
	hasher.Write([]byte(auth))
	hasher.Write(zero)
	hasher.Write([]byte(csrfkey))
	hasher.Write(zero)
	hasher.Write([]byte(salt))
	hasher.Write(zero)
	hasher.Write([]byte(action))
	hasher.Write(zero)
	hash := hexsum(hasher)

	return salt + hash
}

func GetCSRF(action string, r *http.Request) string {
	auth := getauthcookie(r)
	if auth == "" {
		return ""
	}
	hasher := sha512.New512_256()
	io.CopyN(hasher, rand.Reader, 32)
	salt := hexsum(hasher)

	return calculateCSRF(salt, action, auth)
}

func CheckCSRF(action string, r *http.Request) bool {
	auth := getauthcookie(r)
	if auth == "" {
		return false
	}
	csrf := r.FormValue("CSRF")
	if len(csrf) != authlen*2 {
		return false
	}
	salt := csrf[0:authlen]
	rv := calculateCSRF(salt, action, auth)
	ok := subtle.ConstantTimeCompare([]byte(rv), []byte(csrf)) == 1
	return ok
}

func CSRFWrap(action string, handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ok := CheckCSRF(action, r)
		if !ok {
			http.Error(w, "invalid csrf", 403)
			return
		}
		handler.ServeHTTP(w, r)
	})
}

func loginredirect(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "auth",
		Value:    "",
		MaxAge:   -1,
		Secure:   securecookies,
		HttpOnly: true,
	})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

var authregex = regexp.MustCompile("^[[:alnum:]]+$")
var authlen = 32

var stmtUserName, stmtUserAuth, stmtSaveAuth, stmtDeleteAuth *sql.Stmt
var csrfkey string
var securecookies bool

func LoginInit(db *sql.DB) {
	var err error
	stmtUserName, err = db.Prepare("select userid, hash from users where username = ?")
	if err != nil {
		log.Fatal(err)
	}
	var userinfo UserInfo
	t := reflect.TypeOf(userinfo)
	var fields []string
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		fields = append(fields, strings.ToLower(f.Name))
	}
	stmtUserAuth, err = db.Prepare(fmt.Sprintf("select %s from users where userid = (select userid from auth where hash = ?)", strings.Join(fields, ", ")))
	if err != nil {
		log.Fatal(err)
	}
	stmtSaveAuth, err = db.Prepare("insert into auth (userid, hash) values (?, ?)")
	if err != nil {
		log.Fatal(err)
	}
	stmtDeleteAuth, err = db.Prepare("delete from auth where userid = ?")
	if err != nil {
		log.Fatal(err)
	}
	debug := false
	getconfig("debug", &debug)
	securecookies = !debug
	getconfig("csrfkey", &csrfkey)
}

var authinprogress = make(map[string]bool)
var authprogressmtx sync.Mutex

func rateandwait(username string) bool {
	authprogressmtx.Lock()
	defer authprogressmtx.Unlock()
	if authinprogress[username] {
		return false
	}
	authinprogress[username] = true
	go func(name string) {
		time.Sleep(1 * time.Second / 2)
		authprogressmtx.Lock()
		authinprogress[name] = false
		authprogressmtx.Unlock()
	}(username)
	return true
}

func getauthcookie(r *http.Request) string {
	cookie, err := r.Cookie("auth")
	if err != nil {
		return ""
	}
	auth := cookie.Value
	if !(len(auth) == authlen && authregex.MatchString(auth)) {
		log.Printf("login: bad auth: %s", auth)
		return ""
	}
	return auth
}

func checkauthcookie(r *http.Request) (*UserInfo, bool) {
	auth := getauthcookie(r)
	if auth == "" {
		return nil, false
	}
	hasher := sha512.New512_256()
	hasher.Write([]byte(auth))
	authhash := hexsum(hasher)
	row := stmtUserAuth.QueryRow(authhash)
	var userinfo UserInfo
	v := reflect.ValueOf(&userinfo).Elem()
	var ptrs []interface{}
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		ptrs = append(ptrs, f.Addr().Interface())
	}
	err := row.Scan(ptrs...)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("login: no auth found")
		} else {
			log.Printf("login: error scanning auth row: %s", err)
		}
		return nil, false
	}
	return &userinfo, true
}

func loaduser(username string) (int64, string, bool) {
	row := stmtUserName.QueryRow(username)
	var userid int64
	var hash string
	err := row.Scan(&userid, &hash)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("login: no username found")
		} else {
			log.Printf("login: error loading username: %s", err)
		}
		return -1, "", false
	}
	return userid, hash, true
}

var userregex = regexp.MustCompile("^[[:alnum:]]+$")
var userlen = 32
var passlen = 128

func hexsum(h hash.Hash) string {
	return fmt.Sprintf("%x", h.Sum(nil))[0:authlen]
}

func dologin(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")

	if len(username) == 0 || len(username) > userlen ||
		!userregex.MatchString(username) || len(password) == 0 ||
		len(password) > passlen {
		log.Printf("login: invalid password attempt")
		loginredirect(w, r)
		return
	}
	userid, hash, ok := loaduser(username)
	if !ok {
		loginredirect(w, r)
		return
	}

	if !rateandwait(username) {
		loginredirect(w, r)
		return
	}

	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		log.Printf("login: incorrect password")
		loginredirect(w, r)
		return
	}
	hasher := sha512.New512_256()
	io.CopyN(hasher, rand.Reader, 32)
	hash = hexsum(hasher)

	http.SetCookie(w, &http.Cookie{
		Name:     "auth",
		Value:    hash,
		MaxAge:   3600 * 24 * 30,
		Secure:   securecookies,
		HttpOnly: true,
	})

	hasher.Reset()
	hasher.Write([]byte(hash))
	authhash := hexsum(hasher)

	_, err = stmtSaveAuth.Exec(userid, authhash)
	if err != nil {
		log.Printf("error saving auth: %s", err)
	}

	log.Printf("login: successful login")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func dologout(w http.ResponseWriter, r *http.Request) {
	userinfo, ok := checkauthcookie(r)
	if ok && CheckCSRF("logout", r) {
		_, err := stmtDeleteAuth.Exec(userinfo.UserID)
		if err != nil {
			log.Printf("login: error deleting old auth: %s", err)
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name:     "auth",
			Value:    "",
			MaxAge:   -1,
			Secure:   securecookies,
			HttpOnly: true,
		})
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
