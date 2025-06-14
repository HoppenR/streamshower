package main

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (ui *UI) listInputHandler(event *tcell.EventKey) *tcell.EventKey {
	var ok bool
	if ui.mainPage.focusedList, ok = ui.app.GetFocus().(*tview.List); !ok {
		panic("input handler called where it shouldn't have been")
	}

	var lhs, rhs string
	lhs = encodeMappingKey(event)
	rhs, ok = ui.mapRegistry.mappings[lhs]
	if ok {
		if ui.mapDepth > 0 {
			panic("bug: new mapping called from unfinished mapping")
		}
		keyStrings, err := ui.mapRegistry.resolveMappings(rhs)
		if err != nil {
			ui.mainPage.appStatusText.SetText(fmt.Sprintf("[%s]%s[-]", "orange", err.Error()))
			ui.mapDepth = 0
			return nil
		}
		ui.mapDepth += len(keyStrings)
		err = ui.execCommandChainMapping(keyStrings)
		if err != nil {
			ui.mainPage.appStatusText.SetText(fmt.Sprintf("[%s]%s[-]", "orange", err.Error()))
		}
		return nil
	}

	switch lhs {
	case "/", ":", "?":
		ui.mainPage.commandLine.SetText(lhs)
		ui.app.SetFocus(ui.mainPage.commandLine)
	case "<C-d>":
		ui.movePgDown()
	case "<C-e>":
		ui.redrawUp()
	case "<C-n>", "<Down>", "j":
		ui.moveDown()
	case "<C-p>", "<Up>", "k":
		ui.moveUp()
	case "<C-u>":
		ui.movePgUp()
	case "<C-y>":
		ui.redrawDown()
	case "G":
		ui.moveBot()
	case "M":
		ui.moveMid()
	case "N":
		ui.searchPrev()
	case "g":
		ui.moveTop()
	case "n":
		ui.searchNext()
	case "z":
		ui.redrawMid()
	}
	if ui.mapDepth > 0 {
		ui.mapDepth--
	}
	return nil
}

func (ui *UI) commandLineInputHandler(event *tcell.EventKey) *tcell.EventKey {
	if ui.mapDepth > 0 {
		ui.mapDepth--
		if event.Key() == tcell.KeyCtrlZ {
			// Temporarily pretend we are not in a keymapping
			// so that the autocomplete function can run normally
			oldMapDepth := ui.mapDepth
			ui.mapDepth = 0
			ui.mainPage.commandLine.Autocomplete()
			ui.mapDepth = oldMapDepth
			return nil
		}
		return event
	}

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
		} else if ui.cmdRegistry.histIndex == len(ui.cmdRegistry.history) {
			ui.cmdRegistry.histIndex -= 1
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
		// Common vim key for accepting completion window
		return tcell.NewEventKey(tcell.KeyTab, 0, tcell.ModNone)
	case tcell.KeyCtrlE:
		// Common vim key for dismissing completion window
		return tcell.NewEventKey(tcell.KeyEsc, 0, tcell.ModNone)
	}
	return event
}
