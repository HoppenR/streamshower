package main

import (
	"fmt"
	"strings"
)

type BuiltinHelp struct {
	Names       []string
	Description string
}

var builtinHelps = []BuiltinHelp{
	{Names: []string{"/", "?"}, Description: "Enter search mode"},
	{Names: []string{":"}, Description: "Enter command mode"},
	{Names: []string{"<Bar>"}, Description: "Special character representing `|` for chaining commands inside the rhs in mappings"},
	{Names: []string{"<C-d>"}, Description: "Scroll downwards half of the list"},
	{Names: []string{"<C-e>"}, Description: "Scroll downwards one line"},
	{Names: []string{"<C-n>", "<Down>", "j"}, Description: "Go down one line"},
	{Names: []string{"<C-p>", "<Up>", "k"}, Description: "Go up one line"},
	{Names: []string{"<C-u>"}, Description: "Scroll upwards half of the list"},
	{Names: []string{"<C-y>"}, Description: "Scroll upwards one line"},
	{Names: []string{"<C-z>"}, Description: "When used in mappings, this triggers autocomplete (like `wildcharm` in vim)"},
	{Names: []string{"G"}, Description: "Go to last line of the list"},
	{Names: []string{"M"}, Description: "Go to middle of the list"},
	{Names: []string{"N"}, Description: "Go to previous search match"},
	{Names: []string{"g"}, Description: "Go to first line of the list"},
	{Names: []string{"special-keys"}, Description: "<Bar> <Down> <CR> <Esc> <Left> <Right> <Space> <Tab> <Up> <C-a>..<C-z> <F1>..<F12>"},
	{Names: []string{"n"}, Description: "Go to next search match"},
	{Names: []string{"option-list"}, Description: "strims: toggle strims window;  winopen: open links in new browser window"},
	{Names: []string{"z"}, Description: "Redraw line at center of window"},
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
