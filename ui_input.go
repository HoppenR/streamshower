package main

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var SHORTCUT_MAINWIN_HELP = strings.Join(
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
		"[red::u]^d[-::-]PgDwn",
		"[red::u]^u[-::-]PgUp",
		"[red::u]z[-::-]CenterWin",
		"\n",
		"[red]{o|^w}[-]List",
		"[red::u]i[-::-]Info",
		"[red::u]f[-::-]OpenFilter",
		"[red::u]F[-::-]ClearFilter",
		"[red::u]r[-::-]Sync",
		"[red::u]R[-::-]Update",
		"[red::u]t[-::-]ToggleStrims",
		"[red::u]q[-::-]Quit",
		"\n",
		"Open [red]{l|→|↵|^j}[-]Embed",
		"[red::u]w[-::-]Website",
		"[red::u]s[-::-]Strims",
		"[red::u]c[-::-]Chat",
		"[red::u]m[-::-]Mpv",
	},
	" ",
)

var SHORTCUT_INFOWIN_HELP = strings.Join(
	[]string{
		"",
		"[red]{i|q}[-]Back",
		"\n",
		"\n",
	},
	" ",
)

func (ui *UI) streamInfoInputHandler(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyRune:
		switch event.Rune() {
		case 'i', 'q':
			ui.mainPage.con.ResizeItem(ui.mainPage.streamsCon, 0, 1)
			ui.app.SetFocus(ui.mainPage.focusedList)
			return nil
			// case 'u':
			// 	data, err := ui.getSelectedStreamData()
			// 	if err != nil {
			// 		return nil
			// 	}
			// 	if data.IsFollowed() {
			// 		sc.UnfollowStream(data)
			// 	}
		}
	}
	return event
}

func (ui *UI) listInputHandler(event *tcell.EventKey) *tcell.EventKey {
	handleFinish := func(err error) {
		if err != nil {
			ui.mainPage.appStatusText.SetText(fmt.Sprintf("[%s]%s[-]", "orange", err.Error()))
			return
		}
		ui.app.Stop()
	}
	var ok bool
	ui.mainPage.focusedList, ok = ui.app.GetFocus().(*tview.List)
	if !ok {
		return event
	}
	listCnt := ui.mainPage.focusedList.GetItemCount()
	listIdx := ui.mainPage.focusedList.GetCurrentItem()
	switch event.Key() {
	case tcell.KeyRune:
		switch event.Rune() {
		case 'c':
			handleFinish(ui.openSelectedStream(lnkOpenChat))
			return nil
		case 'f':
			var curFilter *FilterInput
			switch ui.mainPage.focusedList {
			case ui.mainPage.twitchList:
				curFilter = ui.twitchFilter
			case ui.mainPage.strimsList:
				curFilter = ui.strimsFilter
			}
			var repeatType rune
			if curFilter.inverted {
				repeatType = 'g'
			} else {
				repeatType = 'v'
			}
			ui.mainPage.commandLine.SetText(fmt.Sprintf(":%c/%s/d", repeatType, curFilter.input))
			ui.mainPage.commandLine.InputHandler()(tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone), nil)
			ui.mainPage.commandLine.InputHandler()(tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone), nil)
			ui.app.SetFocus(ui.mainPage.commandLine)
			return nil
		case 'F':
			switch ui.mainPage.focusedList {
			case ui.mainPage.twitchList:
				ui.twitchFilter.inverted = false
				ui.twitchFilter.input = ""
				ui.updateTwitchList(ui.twitchFilter.input)
			case ui.mainPage.strimsList:
				ui.strimsFilter.inverted = false
				ui.strimsFilter.input = ""
				ui.updateStrimsList(ui.strimsFilter.input)
			}
			return nil
		case 'G':
			ui.mainPage.focusedList.SetCurrentItem(listCnt - 1)
			return nil
		case 'g':
			ui.mainPage.focusedList.SetCurrentItem(0)
			return nil
		case 'i':
			ui.mainPage.con.ResizeItem(ui.mainPage.streamsCon, 0, 0)
			ui.app.SetFocus(ui.mainPage.streamInfo)
			return nil
		case 'j':
			if listIdx != listCnt-1 {
				ui.mainPage.focusedList.SetCurrentItem(listIdx + 1)
			}
			return nil
		case 'k':
			if listIdx != 0 {
				ui.mainPage.focusedList.SetCurrentItem(listIdx - 1)
			}
			return nil
		case 'l':
			handleFinish(ui.openSelectedStream(lnkOpenEmbed))
			return nil
		case 'M':
			offset, _ := ui.mainPage.focusedList.GetOffset()
			_, _, _, height := ui.mainPage.focusedList.GetRect()
			midView := offset + (height / 4) - 1
			midItem := offset + (listCnt-1)/2
			if midItem < midView {
				ui.mainPage.focusedList.SetCurrentItem(midItem)
			} else {
				ui.mainPage.focusedList.SetCurrentItem(midView)
			}
			return nil
		case 'm':
			handleFinish(ui.openSelectedStream(lnkOpenMpv))
			return nil
		case 'N':
			ui.searchPrev()
		case 'n':
			ui.searchNext()
		case 'o':
			ui.enableStrimsList()
			switch ui.mainPage.focusedList {
			case ui.mainPage.twitchList:
				ui.app.SetFocus(ui.mainPage.strimsList)
				ui.mainPage.focusedList = ui.mainPage.strimsList
				ui.refreshStrimsList()
			case ui.mainPage.strimsList:
				ui.app.SetFocus(ui.mainPage.twitchList)
				ui.mainPage.focusedList = ui.mainPage.twitchList
				ui.refreshTwitchList()
			}
			return nil
		case 'q':
			ui.app.Stop()
			return nil
		case 'r':
			ui.mainPage.commandLine.SetText(":sync")
			ui.app.SetFocus(ui.mainPage.commandLine)
			return nil
		case 'R':
			ui.mainPage.commandLine.SetText(":update!")
			ui.app.SetFocus(ui.mainPage.commandLine)
			return nil
		case 's':
			handleFinish(ui.openSelectedStream(lnkOpenStrims))
			return nil
		case 't':
			ui.toggleStrimsList()
			ui.app.SetFocus(ui.mainPage.twitchList)
			ui.refreshTwitchList()
			return nil
		case 'w':
			handleFinish(ui.openSelectedStream(lnkOpenHomePage))
			return nil
		case 'z':
			rOff, cOff := ui.mainPage.focusedList.GetOffset()
			_, _, _, height := ui.mainPage.focusedList.GetRect()
			delta := (listIdx - rOff) - (height / 4)
			ui.mainPage.focusedList.SetOffset(rOff+delta, cOff)
		case '!':
			switch ui.mainPage.focusedList {
			case ui.mainPage.twitchList:
				if ui.twitchFilter.inverted {
					ui.mainPage.commandLine.SetText(":v/" + ui.twitchFilter.input + "/d")
					ui.twitchFilter.inverted = false
				} else {
					ui.mainPage.commandLine.SetText(":g/" + ui.twitchFilter.input + "/d")
					ui.twitchFilter.inverted = true
				}
			case ui.mainPage.strimsList:
				if ui.strimsFilter.inverted {
					ui.mainPage.commandLine.SetText(":v/" + ui.strimsFilter.input + "/d")
					ui.strimsFilter.inverted = false
				} else {
					ui.mainPage.commandLine.SetText(":g/" + ui.strimsFilter.input + "/d")
					ui.strimsFilter.inverted = true
				}
			}
		case '/':
			ui.mainPage.commandLine.SetText("/")
			ui.app.SetFocus(ui.mainPage.commandLine)
		case '?':
			ui.mainPage.commandLine.SetText("?")
			ui.app.SetFocus(ui.mainPage.commandLine)
		case ':':
			ui.mainPage.commandLine.SetText(":")
			ui.app.SetFocus(ui.mainPage.commandLine)
		}
	case tcell.KeyLeft:
		return nil
	case tcell.KeyCtrlE:
		rOff, cOff := ui.mainPage.focusedList.GetOffset()
		if listIdx == rOff {
			ui.mainPage.focusedList.SetCurrentItem(listIdx + 1)
		}
		ui.mainPage.focusedList.SetOffset(rOff+1, cOff)
		return nil
	case tcell.KeyCtrlY:
		rOff, cOff := ui.mainPage.focusedList.GetOffset()
		_, _, _, height := ui.mainPage.focusedList.GetInnerRect()
		if rOff > 0 {
			if listIdx-rOff == (height/2)-1 {
				ui.mainPage.focusedList.SetCurrentItem(listIdx - 1)
			}
			ui.mainPage.focusedList.SetOffset(rOff-1, cOff)
		}
		return nil
	case tcell.KeyCtrlW:
		ui.enableStrimsList()
		switch ui.mainPage.focusedList {
		case ui.mainPage.twitchList:
			ui.app.SetFocus(ui.mainPage.strimsList)
			ui.mainPage.focusedList = ui.mainPage.strimsList
			ui.refreshStrimsList()
		case ui.mainPage.strimsList:
			ui.app.SetFocus(ui.mainPage.twitchList)
			ui.mainPage.focusedList = ui.mainPage.twitchList
			ui.refreshTwitchList()
		}
		return nil
	case tcell.KeyCtrlU:
		_, _, _, height := ui.mainPage.focusedList.GetRect()
		jumpoff := (height / 4) - 1
		if listIdx <= jumpoff {
			ui.mainPage.focusedList.SetCurrentItem(0)
		} else {
			ui.mainPage.focusedList.SetCurrentItem(listIdx - jumpoff)
		}
	case tcell.KeyCtrlD:
		_, _, _, height := ui.mainPage.focusedList.GetRect()
		jumpoff := (height / 4) - 1
		if listIdx >= listCnt-jumpoff {
			ui.mainPage.focusedList.SetCurrentItem(listCnt)
		} else {
			ui.mainPage.focusedList.SetCurrentItem(listIdx + jumpoff)
		}
	case tcell.KeyEnter, tcell.KeyRight, tcell.KeyCtrlJ:
		handleFinish(ui.openSelectedStream(lnkOpenEmbed))
		return nil
	case tcell.KeyDown, tcell.KeyCtrlN:
		if listIdx != listCnt-1 {
			ui.mainPage.focusedList.SetCurrentItem(listIdx + 1)
		}
		return nil
	case tcell.KeyUp, tcell.KeyCtrlP:
		if listIdx != 0 {
			ui.mainPage.focusedList.SetCurrentItem(listIdx - 1)
		}
		return nil
	case tcell.KeyCtrlH:
		rOff, cOff := ui.mainPage.focusedList.GetOffset()
		ui.mainPage.focusedList.SetOffset(rOff, cOff-1)
		return nil
	case tcell.KeyCtrlL:
		rOff, cOff := ui.mainPage.focusedList.GetOffset()
		ui.mainPage.focusedList.SetOffset(rOff, cOff+1)
		return nil
	}
	// Let the default list primitive key event handler handle the rest
	return event
}
