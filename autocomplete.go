package main

import (
	"regexp"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (ui *UI) commandLineCompleteDone(cmdLine string, index int, source int) bool {
	if source == tview.AutocompletedEnter {
		ui.cmdRegistry.histIndex = len(ui.cmdRegistry.history)
		ui.cmdRegistry.history = append(ui.cmdRegistry.history, cmdLine)
		ui.mainPage.commandLine.SetText(cmdLine)
		err := ui.execCommand(cmdLine)
		if err != nil {
			ui.mainPage.appStatusText.SetText(err.Error())
			return false
		}
		ui.app.SetFocus(ui.mainPage.focusedList)
		return true
	} else if source == tview.AutocompletedTab {
		// Move cursor into the regex pattern if completion is :v or :g
		if cmdLine == ":global//d" || cmdLine == ":global//p" {
			ui.app.QueueEvent(tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone))
			ui.app.QueueEvent(tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone))
		} else if cmdLine == ":vglobal//d" || cmdLine == ":vglobal//p" {
			ui.app.QueueEvent(tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone))
			ui.app.QueueEvent(tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone))
		}
		ui.mainPage.commandLine.SetText(cmdLine)
		return false
	}
	return false
}

func (ui *UI) commandLineComplete(currentText string) []string {
	if currentText == "" {
		return nil
	}
	if strings.Contains(currentText, "|") {
		return nil
	}
	re := regexp.MustCompile(`[ /]`)
	fields := re.Split(currentText, -1)
	switch len(fields) {
	case 1:
		possibleCmds := ui.cmdRegistry.matchPossibleCommands(strings.TrimLeft(fields[0], ":"))
		entries := make([]string, 0, len(possibleCmds))
		for _, cmd := range possibleCmds {
			entries = append(entries, ":"+cmd.Name)
		}
		return entries
	case 2:
		cmd := strings.TrimPrefix(fields[0], ":")
		namepart, _, _ := parseCommandParts(cmd)
		if namepart == "" {
			return nil
		}
		possibleCmds := ui.cmdRegistry.matchPossibleCommands(namepart)
		if len(possibleCmds) == 1 {
			if possibleCmds[0].Complete != nil {
				return possibleCmds[0].Complete(ui, fields[1])
			}
		}
	}
	return nil
}
