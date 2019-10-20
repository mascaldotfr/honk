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
	"html/template"
	"io/ioutil"
	"log"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

func adminscreen() {
	log.SetOutput(ioutil.Discard)

	app := tview.NewApplication()
	var maindriver func(event *tcell.EventKey) *tcell.EventKey

	table := tview.NewTable().SetFixed(1, 0).SetSelectable(true, false).
		SetSelectedStyle(tcell.ColorBlack, tcell.ColorGreen, 0)

	mainframe := tview.NewFrame(table)
	mainframe.AddText(tview.Escape("honk admin - [q] quit"),
		true, 0, tcell.ColorGreen)
	mainframe.SetBorders(1, 0, 1, 0, 4, 0)

	dupecell := func(base *tview.TableCell) *tview.TableCell {
		rv := new(tview.TableCell)
		*rv = *base
		return rv
	}

	showtable := func() {
		table.Clear()

		row := 0
		{
			col := 0
			headcell := tview.TableCell{
				Color: tcell.ColorWhite,
			}
			cell := dupecell(&headcell)
			cell.Text = "Message"
			table.SetCell(row, col, cell)
			col++
			cell = dupecell(&headcell)
			cell.Text = ""
			table.SetCell(row, col, cell)

			row++
		}
		{
			col := 0
			headcell := tview.TableCell{
				Color: tcell.ColorWhite,
			}
			cell := dupecell(&headcell)
			cell.Text = "Server"
			table.SetCell(row, col, cell)
			col++
			cell = dupecell(&headcell)
			cell.Text = tview.Escape(string(serverMsg))
			table.SetCell(row, col, cell)

			row++
		}
		{
			col := 0
			headcell := tview.TableCell{
				Color: tcell.ColorWhite,
			}
			cell := dupecell(&headcell)
			cell.Text = "About"
			table.SetCell(row, col, cell)
			col++
			cell = dupecell(&headcell)
			cell.Text = tview.Escape(string(aboutMsg))
			table.SetCell(row, col, cell)

			row++
		}
		{
			col := 0
			headcell := tview.TableCell{
				Color: tcell.ColorWhite,
			}
			cell := dupecell(&headcell)
			cell.Text = "Login"
			table.SetCell(row, col, cell)
			col++
			cell = dupecell(&headcell)
			cell.Text = tview.Escape(string(loginMsg))
			table.SetCell(row, col, cell)

			row++
		}

		app.SetInputCapture(maindriver)
		app.SetRoot(mainframe, true)
	}

	arrowadapter := func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyDown:
			return tcell.NewEventKey(tcell.KeyTab, '\t', tcell.ModNone)
		case tcell.KeyUp:
			return tcell.NewEventKey(tcell.KeyBacktab, '\t', tcell.ModNone)
		}
		return event
	}

	editform := tview.NewForm()
	namebox := tview.NewInputField().SetLabel("name").SetFieldWidth(20)
	descbox := tview.NewInputField().SetLabel("description").SetFieldWidth(60)
	orderbox := tview.NewInputField().SetLabel("order").SetFieldWidth(10)
	editform.AddButton("save", nil)
	editform.AddButton("cancel", nil)
	savebutton := editform.GetButton(0)
	editform.SetFieldTextColor(tcell.ColorBlack)
	editform.SetFieldBackgroundColor(tcell.ColorGreen)
	editform.SetLabelColor(tcell.ColorWhite)
	editform.SetButtonTextColor(tcell.ColorGreen)
	editform.SetButtonBackgroundColor(tcell.ColorBlack)
	editform.GetButton(1).SetSelectedFunc(showtable)
	editform.SetCancelFunc(showtable)

	hadchanges := false

	showform := func() {
		editform.Clear(false)
		editform.AddFormItem(namebox)
		editform.AddFormItem(descbox)
		editform.AddFormItem(orderbox)
		app.SetInputCapture(arrowadapter)
		app.SetRoot(editform, true)
	}

	editrepo := func(which string) {
		namebox.SetText(which)
		descbox.SetText("message")
		savebutton.SetSelectedFunc(func() {
			serverMsg = template.HTML(descbox.GetText())
			hadchanges = true
			showtable()
		})
		showform()
	}

	maindriver = func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'd':
		case 'e':
			editrepo("servermsg")
		case 'q':
			app.Stop()
			return nil
		}
		return event
	}

	showtable()
	app.Run()
}
