package main

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (ui *UI) setupMainPage() {
	ui.app.EnableMouse(true)
	ui.mainPage.con.AddItem(ui.mainPage.streamsCon, 0, 1, true)
	ui.mainPage.con.AddItem(ui.mainPage.infoCon, 0, 2, false)
	ui.mainPage.con.SetDirection(tview.FlexColumn)
	ui.mainPage.streamsCon.SetDirection(tview.FlexRow)
	ui.mainPage.infoCon.SetDirection(tview.FlexRow)
	ui.mainPage.commandRow.SetDirection(tview.FlexColumn)
	// TwitchList
	ui.mainPage.streamsCon.AddItem(ui.mainPage.twitchList, 0, 1, true)
	ui.mainPage.streamsCon.SetInputCapture(ui.listInputHandler)
	ui.mainPage.twitchList.SetChangedFunc(ui.mainPage.updateTwitchStreamInfo)
	ui.mainPage.twitchList.SetBackgroundColor(tcell.ColorDefault)
	ui.mainPage.twitchList.SetBorder(true)
	ui.mainPage.twitchList.SetBorderPadding(0, 0, 1, 1)
	ui.mainPage.twitchList.SetSecondaryTextColor(tcell.ColorDefault)
	ui.mainPage.twitchList.SetTitle("Twitch")
	ui.mainPage.twitchList.SetSelectedFocusOnly(true)
	// StrimsList
	ui.mainPage.streamsCon.AddItem(ui.mainPage.strimsList, 0, 1, false)
	ui.mainPage.strimsList.SetChangedFunc(ui.mainPage.updateStrimsStreamInfo)
	ui.mainPage.strimsList.SetBackgroundColor(tcell.ColorDefault)
	ui.mainPage.strimsList.SetBorder(true)
	ui.mainPage.strimsList.SetBorderPadding(0, 0, 1, 1)
	ui.mainPage.strimsList.SetSecondaryTextColor(tcell.ColorDefault)
	ui.mainPage.strimsList.SetTitle("Strims")
	ui.mainPage.strimsList.SetSelectedFocusOnly(true)
	// ErrorInfo
	ui.mainPage.infoCon.AddItem(ui.mainPage.appStatusText, 3, 0, false)
	ui.mainPage.appStatusText.SetBackgroundColor(tcell.ColorDefault)
	ui.mainPage.appStatusText.SetTitle("Status (" + ui.addr + ")")
	ui.mainPage.appStatusText.SetBorder(true)
	ui.mainPage.appStatusText.SetDynamicColors(true)
	ui.mainPage.appStatusText.SetTextAlign(tview.AlignCenter)
	// StreamInfo
	ui.mainPage.infoCon.AddItem(ui.mainPage.streamInfo, 0, 1, false)
	ui.mainPage.streamInfo.SetBackgroundColor(tcell.ColorDefault)
	ui.mainPage.streamInfo.SetBorder(true)
	ui.mainPage.streamInfo.SetDynamicColors(true)
	ui.mainPage.streamInfo.SetTitle("Stream Info")
	// CommandRow
	ui.mainPage.infoCon.AddItem(ui.mainPage.commandRow, 1, 0, false)
	ui.mainPage.commandRow.AddItem(ui.mainPage.commandLine, 0, 1, true)
	ui.mainPage.commandRow.AddItem(ui.mainPage.fetchTimeView, 26, 0, false)
	// CommandLine
	ui.mainPage.commandLine.SetText("Please see `:help` or `:map`!")
	ui.mainPage.commandLine.SetFieldBackgroundColor(tcell.ColorBlack)
	ui.mainPage.commandLine.SetChangedFunc(ui.onTypeCommandChain)
	ui.mainPage.commandLine.SetFinishedFunc(ui.execCommandChainCallback)
	ui.mainPage.commandLine.SetInputCapture(ui.commandLineInputHandler)
	ui.mainPage.commandLine.SetAutocompletedFunc(ui.commandLineCompleteDone)
	ui.mainPage.commandLine.SetAutocompleteFunc(ui.commandLineComplete)
	// Fetch time view
	ui.mainPage.fetchTimeView.SetBackgroundColor(tcell.ColorOrange)
	ui.mainPage.fetchTimeView.SetTextColor(tcell.ColorBlack)
	ui.mainPage.fetchTimeView.SetText("No timing data ")
	ui.mainPage.fetchTimeView.SetTextAlign(tview.AlignRight)
	ui.mainPage.fetchTimeView.SetDrawFunc(ui.updateLastFetched)
}

func (ui *UI) updateLastFetched(screen tcell.Screen, x, y, width, height int) (int, int, int, int) {
	if !ui.mainPage.streams.LastFetched.IsZero() {
		ui.mainPage.fetchTimeView.Clear()
		ui.mainPage.fetchTimeView.Write(fmt.Appendf(
			nil,
			"%s (update in %.0fs) ",
			ui.mainPage.streams.LastFetched.In(time.Local).Format(time.TimeOnly),
			time.Until(ui.mainPage.streams.LastFetched.Add(ui.mainPage.streams.RefreshInterval)).Seconds(),
		))
	}
	return x, y, width, height
}
