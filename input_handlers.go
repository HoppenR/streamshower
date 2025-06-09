package main

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (ui *UI) listInputHandler(event *tcell.EventKey) *tcell.EventKey {
	var ok bool
	ui.mainPage.focusedList, ok = ui.app.GetFocus().(*tview.List)
	if !ok {
		return event
	}
	lhs := encodeMappingKey(event)
	rhs, ok := ui.mapRegistry.mappings[lhs]
	if ok {
		err := ui.execCommandChainMapping(rhs)
		if err != nil {
			ui.mainPage.appStatusText.SetText(fmt.Sprintf("[%s]%s[-]", "orange", err.Error()))
		}
	}
	return nil
}

func (ui *UI) commandLineInputHandler(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyUp:
		if len(ui.cmdRegistry.history) == 0 {
			return nil
		}
		if ui.cmdRegistry.histIndex > 0 {
			ui.cmdRegistry.histIndex -= 1
		}
		cmdLine := ui.cmdRegistry.history[ui.cmdRegistry.histIndex]
		ui.mainPage.commandLine.SetText(cmdLine)
		ui.mainPage.commandLine.Autocomplete()
		return nil
	case tcell.KeyDown:
		if len(ui.cmdRegistry.history) == 0 {
			return nil
		}
		if ui.cmdRegistry.histIndex < len(ui.cmdRegistry.history)-1 {
			ui.cmdRegistry.histIndex += 1
		}
		cmdLine := ui.cmdRegistry.history[ui.cmdRegistry.histIndex]
		ui.mainPage.commandLine.SetText(cmdLine)
		ui.mainPage.commandLine.Autocomplete()
		return nil
	case tcell.KeyCtrlP:
		return tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
	case tcell.KeyCtrlN:
		return tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
	case tcell.KeyCtrlY:
		return tcell.NewEventKey(tcell.KeyTab, 0, tcell.ModNone)
	}
	return event
}
