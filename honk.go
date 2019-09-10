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
	"fmt"
	"html/template"
	"log"
	"os"
	"time"
)

type WhatAbout struct {
	ID        int64
	Name      string
	Display   string
	About     string
	Key       string
	URL       string
	SkinnyCSS bool
}

type Honk struct {
	ID       int64
	UserID   int64
	Username string
	What     string
	Honker   string
	Handle   string
	Oonker   string
	Oondle   string
	XID      string
	RID      string
	Date     time.Time
	URL      string
	Noise    string
	Precis   string
	Convoy   string
	Audience []string
	Public   bool
	Whofore  int64
	Replies  []*Honk
	Flags    int64
	HTML     template.HTML
	Style    string
	Open     string
	Donks    []*Donk
	Onts     []string
}

const (
	flagIsAcked  = 1
	flagIsBonked = 2
)

func (honk *Honk) IsAcked() bool {
	return honk.Flags&flagIsAcked != 0
}

func (honk *Honk) IsBonked() bool {
	return honk.Flags&flagIsBonked != 0
}

type Donk struct {
	FileID  int64
	XID     string
	Name    string
	Desc    string
	URL     string
	Media   string
	Local   bool
	Content []byte
}

type Honker struct {
	ID     int64
	UserID int64
	Name   string
	XID    string
	Handle string
	Flavor string
	Combos []string
}

type Zonker struct {
	ID        int64
	Name      string
	Wherefore string
}

var serverName string
var iconName = "icon.png"
var serverMsg = "Things happen."

func ElaborateUnitTests() {
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
	getconfig("servermsg", &serverMsg)
	getconfig("servername", &serverName)
	getconfig("usersep", &userSep)
	getconfig("honksep", &honkSep)
	getconfig("dnf", &donotfedafterdark)
	prepareStatements(db)
	switch cmd {
	case "adduser":
		adduser()
	case "cleanup":
		arg := "30"
		if len(os.Args) > 2 {
			arg = os.Args[2]
		}
		cleanupdb(arg)
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
