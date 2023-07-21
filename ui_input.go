package main

import (
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var SHORTCUT_HELP = strings.Join(
	[]string{
		"",
		"[red]{↓|j|^n}[-]↓",
		"[red]{↑|k|^p}[-]↑",
		"[red::u]g[-::-]Top",
		"[red::u]G[-::-]Bot",
		"[red::u]^e[-::-]Scr↓",
		"[red::u]^y[-::-]Scr↑",
		"[red::u]^h[-::-]Scr←",
		"[red::u]^l[-::-]Scr→",
		"[red::u]M[-::-]Mid",
		"\n",
		"[red]{o|^w}[-]List",
		"[red::u]i[-::-]Info",
		"[red::u]f[-::-]Filter",
		"[red::u]F[-::-]ClearFilter",
		"[red::u]r[-::-]Refresh",
		"[red::u]q[-::-]Quit",
		"\n",
		"[red]{l|→|↵|^j}[-]Open Embed",
		"[red::u]w[-::-]Open Website",
		"[red::u]s[-::-]Open Strims",
		"[red::u]m[-::-]Open Mpv",
	},
	" ",
)

func (ui *UI) streamInfoInputHandler(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyRune:
		switch event.Rune() {
		case 'i', 'q':
			ui.pg1.con.ResizeItem(ui.pg1.streamsCon, 0, 1)
			ui.app.SetFocus(ui.pg1.focusedList)
			return nil
		}
	}
	return event
}

func (ui *UI) listInputHandler(event *tcell.EventKey) *tcell.EventKey {
	printerr := func(err error) {
		ui.pg1.streamInfo.SetText("[red]⚠ " + err.Error() + "[-]")
	}
	ui.pg1.focusedList = ui.app.GetFocus().(*tview.List)
	listCnt := ui.pg1.focusedList.GetItemCount()
	listIdx := ui.pg1.focusedList.GetCurrentItem()
	switch event.Key() {
	case tcell.KeyRune:
		switch event.Rune() {
		case 'g':
			ui.pg1.focusedList.SetCurrentItem(0)
			return nil
		case 'G':
			ui.pg1.focusedList.SetCurrentItem(listCnt - 1)
			return nil
		case 'M':
			offset, _ := ui.pg1.focusedList.GetOffset()
			_, _, _, height := ui.pg1.focusedList.GetRect()
			midView := offset + (height / 4) - 1
			midItem := offset + (listCnt-1)/2
			if midItem < midView {
				ui.pg1.focusedList.SetCurrentItem(midItem)
			} else {
				ui.pg1.focusedList.SetCurrentItem(midView)
			}
			return nil
		case 'i':
			ui.pg1.con.ResizeItem(ui.pg1.streamsCon, 0, 0)
			ui.app.SetFocus(ui.pg1.streamInfo)
			return nil
		case 'j':
			if listIdx != listCnt-1 {
				ui.pg1.focusedList.SetCurrentItem(listIdx + 1)
			}
			return nil
		case 'k':
			if listIdx != 0 {
				ui.pg1.focusedList.SetCurrentItem(listIdx - 1)
			}
			return nil
		case 'f':
			switch ui.pg1.focusedList.GetTitle() {
			case "Twitch":
				ui.pages.ShowPage("Filter-Twitch")
			case "Strims":
				ui.pages.ShowPage("Filter-Strims")
			}
			return nil
		case 'F':
			switch ui.pg1.focusedList.GetTitle() {
			case "Twitch":
				ui.pg2.input.SetText(DefaultTwitchFilter)
			case "Strims":
				ui.pg3.input.SetText(DefaultRustlerMin)
			}
			return nil
		case 'l':
			if err := ui.openSelectedStream(lnkOpenEmbed); err != nil {
				printerr(err)
			} else {
				ui.app.Stop()
			}
			return nil
		case 'm':
			if err := ui.openSelectedStream(lnkOpenMpv); err != nil {
				printerr(err)
			} else {
				ui.app.Stop()
			}
		case 'o':
			switch ui.pg1.focusedList.GetTitle() {
			case "Twitch":
				ui.app.SetFocus(ui.pg1.strimsList)
				ui.refreshStrimsList()
			case "Strims":
				ui.app.SetFocus(ui.pg1.twitchList)
				ui.refreshTwitchList()
			}
			return nil
		case 'q':
			ui.app.Stop()
			return nil
		case 'r':
			ui.pages.ShowPage("Refresh-Dialogue")
			return nil
		case 's':
			if err := ui.openSelectedStream(lnkOpenStrims); err != nil {
				printerr(err)
			} else {
				ui.app.Stop()
			}
			return nil
		case 'w':
			if err := ui.openSelectedStream(lnkOpenHomePage); err != nil {
				printerr(err)
			} else {
				ui.app.Stop()
			}
			return nil
		case 'z':
			rOff, cOff := ui.pg1.focusedList.GetOffset()
			_, _, _, height := ui.pg1.focusedList.GetRect()
			delta := (listIdx - rOff) - (height / 4)
			ui.pg1.focusedList.SetOffset(rOff+delta, cOff)
		case '!':
			if ui.pg2.inverted {
				ui.pg2.input.SetTitle("Filter(Regex)")
				ui.pg2.inverted = false
			} else {
				ui.pg2.input.SetTitle("Filter(Regex(inverted))")
				ui.pg2.inverted = true
			}
			ui.refreshTwitchList()
		}
	case tcell.KeyLeft:
		return nil
	case tcell.KeyCtrlE:
		rOff, cOff := ui.pg1.focusedList.GetOffset()
		if listIdx == rOff {
			ui.pg1.focusedList.SetCurrentItem(listIdx + 1)
		}
		ui.pg1.focusedList.SetOffset(rOff+1, cOff)
		return nil
	case tcell.KeyCtrlY:
		rOff, cOff := ui.pg1.focusedList.GetOffset()
		_, _, _, height := ui.pg1.focusedList.GetInnerRect()
		if rOff > 0 {
			if listIdx-rOff == (height/2)-1 {
				ui.pg1.focusedList.SetCurrentItem(listIdx - 1)
			}
			ui.pg1.focusedList.SetOffset(rOff-1, cOff)
		}
		return nil
	case tcell.KeyCtrlW:
		switch ui.pg1.focusedList.GetTitle() {
		case "Twitch":
			ui.app.SetFocus(ui.pg1.strimsList)
			ui.refreshStrimsList()
		case "Strims":
			ui.app.SetFocus(ui.pg1.twitchList)
			ui.refreshTwitchList()
		}
		return nil
	case tcell.KeyEnter, tcell.KeyRight, tcell.KeyCtrlJ:
		if err := ui.openSelectedStream(lnkOpenEmbed); err != nil {
			printerr(err)
		} else {
			ui.app.Stop()
		}
		return nil
	case tcell.KeyDown, tcell.KeyCtrlN:
		if listIdx != listCnt-1 {
			ui.pg1.focusedList.SetCurrentItem(listIdx + 1)
		}
		return nil
	case tcell.KeyUp, tcell.KeyCtrlP:
		if listIdx != 0 {
			ui.pg1.focusedList.SetCurrentItem(listIdx - 1)
		}
		return nil
	case tcell.KeyCtrlH:
		rOff, cOff := ui.pg1.focusedList.GetOffset()
		ui.pg1.focusedList.SetOffset(rOff, cOff-1)
		return nil
	case tcell.KeyCtrlL:
		rOff, cOff := ui.pg1.focusedList.GetOffset()
		ui.pg1.focusedList.SetOffset(rOff, cOff+1)
		return nil
	}
	// Let the default list primitive key event handler handle the rest
	return event
}
