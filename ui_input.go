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
		"[red]{o|^w}[-]ChList",
		"[red::u]i[-::-]Info",
		"[red::u]f[-::-]Filter",
		"[red::u]F[-::-]Rfilter",
		"[red::u]r[-::-]Sync",
		"[red::u]R[-::-]Update",
		"[red::u]t[-::-]ToggleWin",
		"[red::u]u[-::-]Undo",
		"[red::u]q[-::-]Quit",
		"[red::u]![-::-]InverseFilter",
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
	printError := func(err error) {
		if err != nil {
			ui.mainPage.appStatusText.SetText(fmt.Sprintf("[%s]%s[-]", "orange", err.Error()))
		}
	}
	var ok bool
	ui.mainPage.focusedList, ok = ui.app.GetFocus().(*tview.List)
	if !ok {
		return event
	}
	switch event.Key() {
	case tcell.KeyRune:
		switch event.Rune() {
		case 'c':
			printError(ui.execCommandChain(":open chat | quit"))
		case 'f':
			ui.editFilter(false)
		case 'F':
			ui.editFilter(true)
		case 'G':
			ui.moveBot()
		case 'g':
			ui.moveTop()
		case 'i':
			ui.mainPage.con.ResizeItem(ui.mainPage.streamsCon, 0, 0)
			ui.app.SetFocus(ui.mainPage.streamInfo)
		case 'j':
			ui.moveDown()
		case 'k':
			ui.moveUp()
		case 'l':
			printError(ui.execCommandChain(":open embed | quit"))
		case 'M':
			ui.moveMid()
		case 'm':
			printError(ui.execCommandChain(":open mpv | quit"))
		case 'N':
			printError(ui.execCommandSilent("?"))
		case 'n':
			printError(ui.execCommandSilent("/"))
		case 'o':
			printError(ui.execCommand(":focus toggle"))
		case 'q':
			printError(ui.execCommand(":quit"))
		case 'r':
			printError(ui.execCommand(":sync"))
		case 'R':
			printError(ui.execCommandChain(":update | sync"))
		case 's':
			printError(ui.execCommandChain(":open strims | quit"))
		case 't':
			printError(ui.execCommandChain(":set! strims | focus twitch"))
		case 'u':
			printError(ui.execCommand(":clear"))
		case 'U':
			printError(ui.execCommand(":clear!"))
		case 'w':
			printError(ui.execCommandChain(":open homepage | quit"))
		case 'z':
			ui.redrawMid()
		case '!':
			ui.invertFilter()
		case ' ':
			ui.mainPage.commandLine.SetText(":open ")
			ui.app.SetFocus(ui.mainPage.commandLine)
			ui.mainPage.commandLine.Autocomplete()
		case '/':
			ui.mainPage.commandLine.SetText("/")
			ui.app.SetFocus(ui.mainPage.commandLine)
		case '?':
			ui.mainPage.commandLine.SetText("?")
			ui.app.SetFocus(ui.mainPage.commandLine)
		case ':':
			ui.mainPage.commandLine.SetText(":")
			ui.app.SetFocus(ui.mainPage.commandLine)
			ui.mainPage.commandLine.Autocomplete()
		default:
			return event
		}
	case tcell.KeyLeft:
		return nil
	case tcell.KeyCtrlE:
		ui.redrawUp()
	case tcell.KeyCtrlY:
		ui.redrawDown()
	case tcell.KeyCtrlW:
		printError(ui.execCommand(":focus toggle"))
	case tcell.KeyCtrlU:
		ui.movePgUp()
	case tcell.KeyCtrlD:
		ui.movePgDown()
	case tcell.KeyEnter, tcell.KeyRight, tcell.KeyCtrlJ:
		printError(ui.execCommandChain(":open embed | quit"))
	case tcell.KeyDown, tcell.KeyCtrlN:
		ui.moveDown()
	case tcell.KeyUp, tcell.KeyCtrlP:
		ui.moveUp()
	case tcell.KeyCtrlH:
		ui.redrawLeft()
	case tcell.KeyCtrlL:
		ui.redrawRight()
	default:
		return event
	}
	return nil
}

func (ui *UI) moveUp() {
	listIdx := ui.mainPage.focusedList.GetCurrentItem()
	if listIdx != 0 {
		ui.mainPage.focusedList.SetCurrentItem(listIdx - 1)
	}
}

func (ui *UI) moveDown() {
	listCnt := ui.mainPage.focusedList.GetItemCount()
	listIdx := ui.mainPage.focusedList.GetCurrentItem()
	if listIdx != listCnt-1 {
		ui.mainPage.focusedList.SetCurrentItem(listIdx + 1)
	}
}

func (ui *UI) moveTop() {
	ui.mainPage.focusedList.SetCurrentItem(0)
}

func (ui *UI) moveMid() {
	listCnt := ui.mainPage.focusedList.GetItemCount()
	offset, _ := ui.mainPage.focusedList.GetOffset()
	_, _, _, height := ui.mainPage.focusedList.GetRect()
	midView := offset + (height / 4) - 1
	midItem := offset + (listCnt-1)/2
	if midItem < midView {
		ui.mainPage.focusedList.SetCurrentItem(midItem)
	} else {
		ui.mainPage.focusedList.SetCurrentItem(midView)
	}
}

func (ui *UI) moveBot() {
	listCnt := ui.mainPage.focusedList.GetItemCount()
	ui.mainPage.focusedList.SetCurrentItem(listCnt - 1)
}

func (ui *UI) movePgUp() {
	listIdx := ui.mainPage.focusedList.GetCurrentItem()
	_, _, _, height := ui.mainPage.focusedList.GetRect()
	jumpoff := (height / 4) - 1
	if listIdx <= jumpoff {
		ui.mainPage.focusedList.SetCurrentItem(0)
	} else {
		ui.mainPage.focusedList.SetCurrentItem(listIdx - jumpoff)
	}
}

func (ui *UI) movePgDown() {
	listCnt := ui.mainPage.focusedList.GetItemCount()
	listIdx := ui.mainPage.focusedList.GetCurrentItem()
	_, _, _, height := ui.mainPage.focusedList.GetRect()
	jumpoff := (height / 4) - 1
	if listIdx >= listCnt-jumpoff {
		ui.mainPage.focusedList.SetCurrentItem(listCnt)
	} else {
		ui.mainPage.focusedList.SetCurrentItem(listIdx + jumpoff)
	}
}

func (ui *UI) redrawUp() {
	listIdx := ui.mainPage.focusedList.GetCurrentItem()
	rOff, cOff := ui.mainPage.focusedList.GetOffset()
	if listIdx == rOff {
		ui.mainPage.focusedList.SetCurrentItem(listIdx + 1)
	}
	ui.mainPage.focusedList.SetOffset(rOff+1, cOff)
}

func (ui *UI) redrawDown() {
	listIdx := ui.mainPage.focusedList.GetCurrentItem()
	rOff, cOff := ui.mainPage.focusedList.GetOffset()
	_, _, _, height := ui.mainPage.focusedList.GetInnerRect()
	if rOff > 0 {
		if listIdx-rOff == (height/2)-1 {
			ui.mainPage.focusedList.SetCurrentItem(listIdx - 1)
		}
		ui.mainPage.focusedList.SetOffset(rOff-1, cOff)
	}
}

func (ui *UI) redrawMid() {
	listIdx := ui.mainPage.focusedList.GetCurrentItem()
	rOff, cOff := ui.mainPage.focusedList.GetOffset()
	_, _, _, height := ui.mainPage.focusedList.GetRect()
	delta := (listIdx - rOff) - (height / 4)
	ui.mainPage.focusedList.SetOffset(rOff+delta, cOff)
}

func (ui *UI) redrawLeft() {
	rOff, cOff := ui.mainPage.focusedList.GetOffset()
	ui.mainPage.focusedList.SetOffset(rOff, cOff-1)
}

func (ui *UI) redrawRight() {
	rOff, cOff := ui.mainPage.focusedList.GetOffset()
	ui.mainPage.focusedList.SetOffset(rOff, cOff+1)
}

func (ui *UI) editFilter(invert bool) {
	var curFilter *FilterInput
	switch ui.mainPage.focusedList {
	case ui.mainPage.twitchList:
		curFilter = ui.twitchFilter
	case ui.mainPage.strimsList:
		curFilter = ui.strimsFilter
	}
	var repeatType rune
	if invert {
		repeatType = 'g'
	} else {
		repeatType = 'v'
	}
	ui.mainPage.commandLine.SetText(fmt.Sprintf(":%c/%s/d", repeatType, curFilter.input))
	ui.mainPage.commandLine.InputHandler()(tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone), nil)
	ui.mainPage.commandLine.InputHandler()(tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone), nil)
	ui.app.SetFocus(ui.mainPage.commandLine)
}

func (ui *UI) invertFilter() {
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
}
