package main

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
)

func execNormCommand(ui *UI, args []string, bang bool) error {
	key, err := parseMappingKey(args[0])
	if err != nil {
		return err
	}
	switch key.Key() {
	case tcell.KeyRune:
		switch key.Rune() {
		case 'G':
			ui.moveBot()
		case 'g':
			ui.moveTop()
		case 'j':
			ui.moveDown()
		case 'k':
			ui.moveUp()
		case 'M':
			ui.moveMid()
		case 'z':
			ui.redrawMid()
		default:
			return nil
		}
	case tcell.KeyCtrlE:
		ui.redrawUp()
	case tcell.KeyCtrlY:
		ui.redrawDown()
	case tcell.KeyCtrlU:
		ui.movePgUp()
	case tcell.KeyCtrlD:
		ui.movePgDown()
	case tcell.KeyDown, tcell.KeyCtrlN:
		ui.moveDown()
	case tcell.KeyUp, tcell.KeyCtrlP:
		ui.moveUp()
	default:
		return nil
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

func (ui *UI) scrollInfo(amount int) {
	row, _ := ui.mainPage.streamInfo.GetScrollOffset()
	ui.mainPage.streamInfo.ScrollTo(row+amount, 0)
}

func (ui *UI) searchNext() {
	list := ui.mainPage.focusedList
	count := list.GetItemCount()
	if count == 0 || ui.mainPage.lastSearch == "" {
		return
	}
	current := list.GetCurrentItem()
	ui.mainPage.commandLine.SetText("/" + ui.mainPage.lastSearch)
	for i := 1; i <= count; i++ {
		index := (current + i) % count
		primaryText, _ := list.GetItemText(index)
		if strings.Contains(strings.ToLower(primaryText), strings.ToLower(ui.mainPage.lastSearch)) {
			list.SetCurrentItem(index)
			return
		}
	}
	ui.mainPage.appStatusText.SetText(fmt.Sprintf("[yellow]No match for %q[-]", ui.mainPage.lastSearch))
}

func (ui *UI) searchPrev() {
	list := ui.mainPage.focusedList
	count := list.GetItemCount()
	if count == 0 || ui.mainPage.lastSearch == "" {
		return
	}
	current := list.GetCurrentItem()
	ui.mainPage.commandLine.SetText("?" + ui.mainPage.lastSearch)
	for i := 1; i <= count; i++ {
		index := (current - i + count) % count
		primaryText, _ := list.GetItemText(index)
		if strings.Contains(strings.ToLower(primaryText), strings.ToLower(ui.mainPage.lastSearch)) {
			list.SetCurrentItem(index)
			return
		}
	}
	ui.mainPage.appStatusText.SetText(fmt.Sprintf("[yellow]No match for %q[-]", ui.mainPage.lastSearch))
}
