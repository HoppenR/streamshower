package main

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type MappingRegistry struct {
	mappings map[string]string
}

var defaultMappings = map[string]string{
	"/":       "/",
	":":       ":",
	"<C-b>":   ":scrollinfo up<CR>",
	"<C-d>":   ":normal <C-d><CR>",
	"<C-e>":   ":normal <C-e><CR>",
	"<C-f>":   ":scrollinfo down<CR>",
	"<C-j>":   ":open embed<CR>",
	"<C-n>":   ":normal <C-n><CR>",
	"<C-p>":   ":normal <C-p><CR>",
	"<C-u>":   ":normal <C-u><CR>",
	"<C-w>":   ":focus toggle<CR>",
	"<C-y>":   ":normal <C-y><CR>",
	"<CR>":    ":open embed | quit<CR>",
	"<Down>":  ":normal <Down><CR>",
	"<Enter>": ":open embed | quit<CR>",
	"<F1>":    "please see `:help` or `:map`!<CR>",
	"<Right>": ":open embed | quit<CR>",
	"<Space>": ":open ",
	"<Up>":    ":normal <Up><CR>",
	"?":       "?",
	"G":       ":normal G<CR>",
	"M":       ":normal M<CR>",
	"N":       "?<CR>",
	"R":       ":update | sync<CR>",
	"U":       ":clear!<CR>",
	"c":       ":open chat | quit<CR>",
	"g":       ":normal g<CR>",
	"j":       ":normal j<CR>",
	"k":       ":normal k<CR>",
	"l":       ":open embed | quit<CR>",
	"m":       ":open mpv | quit<CR>",
	"n":       "/<CR>",
	"o":       ":focus toggle<CR>",
	"q":       ":quit<CR>",
	"r":       ":sync<CR>",
	"s":       ":open strims | quit<CR>",
	"t":       ":set! strims | focus twitch<CR>",
	"u":       ":clear<CR>",
	"w":       ":open homepage | quit<CR>",
	"z":       ":normal z<CR>",
}

func NewMappingRegistry() *MappingRegistry {
	return &MappingRegistry{mappings: defaultMappings}
}

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
	var ok bool
	ui.mainPage.focusedList, ok = ui.app.GetFocus().(*tview.List)
	if !ok {
		return event
	}
	lhs := encodeMappingKey(event)
	rhs, ok := ui.mapRegistry.mappings[lhs]
	if ok {
		err := ui.execCommandChain(rhs)
		if err != nil {
			ui.mainPage.appStatusText.SetText(fmt.Sprintf("[%s]%s[-]", "orange", err.Error()))
		}
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
	// ui.mainPage.streamInfo.GetInputCapture()(tcell.NewEventKey(tcell.KeyRune, 'k', tcell.ModNone))
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
		primaryText, secondaryText := list.GetItemText(index)
		if strings.Contains(strings.ToLower(primaryText), strings.ToLower(ui.mainPage.lastSearch)) {
			list.SetCurrentItem(index)
			return
		} else if strings.Contains(strings.ToLower(secondaryText), strings.ToLower(ui.mainPage.lastSearch)) {
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

func validateMappingKey(input string) error {
	_, err := parseMappingKey(input)
	return err
}

func encodeMappingKey(input *tcell.EventKey) string {
	switch input.Key() {
	case tcell.KeyRune:
		switch input.Rune() {
		case ' ':
			return "<Space>"
		default:
			return string(input.Rune())
		}
	case tcell.KeyBackspace:
		return "<BS>"
	case tcell.KeyEnter:
		return "<CR>"
	case tcell.KeyDown:
		return "<Down>"
	case tcell.KeyEsc:
		return "<Esc>"
	case tcell.KeyTab:
		return "<Tab>"
	case tcell.KeyUp:
		return "<Up>"
	}
	if input.Key() >= tcell.KeyCtrlA && input.Key() <= tcell.KeyCtrlZ {
		c := 'a' + rune(input.Key()-tcell.KeyCtrlA)
		return fmt.Sprintf("<C-%c>", c)
	}
	if input.Key() >= tcell.KeyF1 && input.Key() <= tcell.KeyF9 {
		c := '1' + rune(input.Key()-tcell.KeyF1)
		return fmt.Sprintf("<F%c>", c)
	}
	return fmt.Sprintf("<Key-%d>", input.Key())
}

func parseMappingKey(input string) (*tcell.EventKey, error) {
	input = strings.TrimSpace(input)
	if strings.HasPrefix(input, "<") && strings.HasSuffix(input, ">") {
		name := strings.ToLower(input[1 : len(input)-1])
		switch name {
		case "bs", "backspace":
			return tcell.NewEventKey(tcell.KeyBackspace, 0, tcell.ModNone), nil
		case "cr", "enter":
			return tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone), nil
		case "down":
			return tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone), nil
		case "esc":
			return tcell.NewEventKey(tcell.KeyEsc, 0, tcell.ModNone), nil
		case "space":
			return tcell.NewEventKey(tcell.KeyRune, ' ', tcell.ModNone), nil
		case "tab":
			return tcell.NewEventKey(tcell.KeyTab, 0, tcell.ModNone), nil
		case "up":
			return tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone), nil
		}
		if strings.HasPrefix(name, "c-") && len(name) == 3 {
			c := name[2]
			if c >= 'a' && c <= 'z' {
				ctrlKey := tcell.KeyCtrlA + tcell.Key(c-'a')
				return tcell.NewEventKey(ctrlKey, 0, tcell.ModNone), nil
			}
		}
		if strings.HasPrefix(name, "f") && len(name) == 2 {
			c := name[1]
			if c >= '1' && c <= '9' {
				ctrlKey := tcell.KeyF1 + tcell.Key(c-'1')
				return tcell.NewEventKey(ctrlKey, 0, tcell.ModNone), nil
			}
		}
		return nil, fmt.Errorf("unknown key: %s", input)
	}
	if len([]rune(input)) == 1 {
		return tcell.NewEventKey(tcell.KeyRune, []rune(input)[0], tcell.ModNone), nil
	}
	return nil, fmt.Errorf("invalid key format: %s", input)
}
