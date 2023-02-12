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
	"flag"
	"fmt"
	"html/template"
	golog "log"
	"log/syslog"
	notrand "math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"humungus.tedunangst.com/r/webs/httpsig"
	"humungus.tedunangst.com/r/webs/log"
)

var softwareVersion = "develop-muffled"

func init() {
	notrand.Seed(time.Now().Unix())
}

type WhatAbout struct {
	ID      int64
	Name    string
	Display string
	About   string
	HTAbout template.HTML
	Key     string
	URL     string
	Options UserOptions
	SecKey  httpsig.PrivateKey
}

type UserOptions struct {
	MentionAll bool   `json:",omitempty"`
	MeCount    int64
}

type KeyInfo struct {
	keyname string
	seckey  httpsig.PrivateKey
}

const serverUID int64 = -2
const readyLuserOne int64 = 1

type Honk struct {
	ID       int64
	UserID   int64
	Username string
	What     string
	Honker   string
	Handle   string
	Handles  string
	Oonker   string
	Oondle   string
	XID      string
	RID      string
	Date     time.Time
	URL      string
	Noise    string
	Precis   string
	Format   string
	Convoy   string
	Audience []string
	Public   bool
	Whofore  int64
	Replies  []*Honk
	Flags    int64
	HTPrecis template.HTML
	HTML     template.HTML
	Style    string
	Open     string
	Donks    []*Donk
	Mentions []Mention
}

type Mention struct {
	Who   string
	Where string
}

func (mention *Mention) IsPresent(noise string) bool {
	nick := strings.TrimLeft(mention.Who, "@")
	idx := strings.IndexByte(nick, '@')
	if idx != -1 {
		nick = nick[:idx]
	}
	return strings.Contains(noise, ">@"+nick) || strings.Contains(noise, "@<span>"+nick)
}

type OldRevision struct {
	Precis string
	Noise  string
}

const (
	flagIsAcked    = 1
	flagIsBonked   = 2
	flagIsSaved    = 4
	flagIsUntagged = 8
	flagIsReacted  = 16
)

func (honk *Honk) IsAcked() bool {
	return honk.Flags&flagIsAcked != 0
}

func (honk *Honk) IsBonked() bool {
	return honk.Flags&flagIsBonked != 0
}

func (honk *Honk) IsSaved() bool {
	return honk.Flags&flagIsSaved != 0
}

func (honk *Honk) IsUntagged() bool {
	return honk.Flags&flagIsUntagged != 0
}

func (honk *Honk) IsReacted() bool {
	return honk.Flags&flagIsReacted != 0
}

type Donk struct {
	FileID   int64
	XID      string
	Name     string
	Desc     string
	URL      string
	Media    string
	Local    bool
	External bool
}

type Honker struct {
	ID     int64
	UserID int64
	Name   string
	XID    string
	Handle string
	Flavor string
}

type SomeThing struct {
	What  int
	XID   string
	Owner string
	Name  string
}

const (
	SomeNothing int = iota
	SomeActor
	SomeCollection
)

var serverName string
var serverPrefix string
var masqName string
var dataDir = "."
var viewDir = "."
var iconName = "icon.png"
var serverMsg template.HTML
var aboutMsg template.HTML
var loginMsg template.HTML

func ElaborateUnitTests() {
}

func unplugserver(hostname string) {
	db := opendatabase()
	xid := fmt.Sprintf("%%https://%s/%%", hostname)
	db.Exec("delete from honkers where xid like ? and flavor = 'dub'", xid)
	db.Exec("delete from doovers where rcpt like ?", xid)
}

func reexecArgs(cmd string) []string {
	args := []string{"-datadir", dataDir}
	args = append(args, log.Args()...)
	args = append(args, cmd)
	return args
}

var elog, ilog, dlog *golog.Logger

func main() {
	flag.StringVar(&dataDir, "datadir", dataDir, "data directory")
	flag.StringVar(&viewDir, "viewdir", viewDir, "view directory")
	flag.Parse()

	log.Init(log.Options{Progname: "honk", Facility: syslog.LOG_UUCP})
	elog = log.E
	ilog = log.I
	dlog = log.D

	args := flag.Args()
	cmd := "run"
	if len(args) > 0 {
		cmd = args[0]
	}
	switch cmd {
	case "init":
		initdb()
	case "upgrade":
		upgradedb()
	case "version":
		fmt.Println(softwareVersion)
		os.Exit(0)
	}
	db := opendatabase()
	dbversion := 0
	getconfig("dbversion", &dbversion)
	if dbversion != myVersion {
		elog.Fatal("incorrect database version. run upgrade.")
	}
	getconfig("servermsg", &serverMsg)
	getconfig("aboutmsg", &aboutMsg)
	getconfig("loginmsg", &loginMsg)
	getconfig("servername", &serverName)
	getconfig("masqname", &masqName)
	if masqName == "" {
		masqName = serverName
	}
	serverPrefix = fmt.Sprintf("https://%s/", serverName)
	getconfig("usersep", &userSep)
	getconfig("honksep", &honkSep)
	getconfig("devel", &develMode)
	getconfig("fasttimeout", &fastTimeout)
	getconfig("slowtimeout", &slowTimeout)
	getconfig("signgets", &signGets)
	prepareStatements(db)
	switch cmd {
	case "admin":
		adminscreen()
	case "devel":
		if len(args) != 2 {
			elog.Fatal("need an argument: devel (on|off)")
		}
		switch args[1] {
		case "on":
			setconfig("devel", 1)
		case "off":
			setconfig("devel", 0)
		default:
			elog.Fatal("argument must be on or off")
		}
	case "setconfig":
		if len(args) != 3 {
			elog.Fatal("need an argument: setconfig key val")
		}
		var val interface{}
		var err error
		if val, err = strconv.Atoi(args[2]); err != nil {
			val = args[2]
		}
		setconfig(args[1], val)
	case "adduser":
		adduser()
	case "deluser":
		if len(args) < 2 {
			fmt.Printf("usage: honk deluser username\n")
			return
		}
		deluser(args[1])
	case "chpass":
		if len(args) < 2 {
			fmt.Printf("usage: honk chpass username\n")
			return
		}
		chpass(args[1])
	case "cleanup":
		arg := "30"
		if len(args) > 1 {
			arg = args[1]
		}
		cleanupdb(arg)
	case "unplug":
		if len(args) < 2 {
			fmt.Printf("usage: honk unplug servername\n")
			return
		}
		name := args[1]
		unplugserver(name)
	case "backup":
		if len(args) < 2 {
			fmt.Printf("usage: honk backup dirname\n")
			return
		}
		name := args[1]
		svalbard(name)
	case "run":
		serve()
	case "test":
		ElaborateUnitTests()
	default:
		elog.Fatal("unknown command")
	}
}
