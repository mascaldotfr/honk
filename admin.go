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

/*
#include <termios.h>
*/
import "C"
import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
)

func adminscreen() {
	log.SetOutput(ioutil.Discard)
	stdout := os.Stdout
	esc := "\x1b"
	smcup := esc + "[?1049h"
	rmcup := esc + "[?1049l"

	hidecursor := func() {
	}
	showcursor := func() {
	}
	movecursor := func(x, y int) {
		stdout.WriteString(fmt.Sprintf(esc+"[%d;%dH", x, y))
	}
	clearscreen := func() {
		stdout.WriteString(esc + "[2J")
	}

	savedtio := new(C.struct_termios)
	C.tcgetattr(1, savedtio)
	restore := func() {
		stdout.WriteString(rmcup)
		showcursor()
		C.tcsetattr(1, C.TCSAFLUSH, savedtio)
	}
	defer restore()

	init := func() {
		tio := new(C.struct_termios)
		C.tcgetattr(1, tio)
		tio.c_lflag = tio.c_lflag & ^C.uint(C.ECHO|C.ICANON)
		C.tcsetattr(1, C.TCSADRAIN, tio)

		hidecursor()
		stdout.WriteString(smcup)
		clearscreen()
		movecursor(1, 1)
	}

	init()

	for {
		var buf [1]byte
		os.Stdin.Read(buf[:])
		c := buf[0]
		switch c {
		case 'q':
			return
		default:
			os.Stdout.Write(buf[:])
		}
	}
}
