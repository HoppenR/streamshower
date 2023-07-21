package main

import (
	"errors"
	"net/http"
	"sort"

	sc "github.com/HoppenR/streamchecker"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const DefaultTwitchFilter = "(?i)"
const DefaultRustlerMin = "3"

type UI struct {
	app   *tview.Application
	pages *tview.Pages
	pg1   *MainPage    // "Main Window"
	pg2   *FilterInput // "Filter-Twitch"
	pg3   *FilterInput // "Filter-Strims"
	pg4   *DialogueBox // "Refresh-Dialogue"
	addr  string
}

type FilterInput struct {
	con      *tview.Grid
	input    *tview.InputField
	inverted bool
}

type DialogueBox struct {
	modal *tview.Modal
}

type MainPage struct {
	con         *tview.Flex
	focusedList *tview.List
	streamInfo  *tview.TextView
	streams     *sc.Streams
	streamsCon  *tview.Flex
	infoCon     *tview.Flex
	strimsList  *tview.List
	twitchList  *tview.List
	infoText    *tview.TextView
}

func (ui *UI) SetAddress(address string) {
	ui.addr = address
}

func (ui *UI) updateStreams() error {
	streams, err := sc.GetLocalServerData(ui.addr)
	if err != nil {
		return err
	}
	sort.Sort(sort.Reverse(streams.Twitch))
	sort.Sort(sort.Reverse(streams.Strims))
	ui.pg1.streams = streams
	return nil
}

func (ui *UI) forceRemoteUpdate() error {
	resp, err := http.Post("http://" + ui.addr, "application/octet-stream", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func NewUI() *UI {
	return &UI{
		app:   tview.NewApplication(),
		pages: tview.NewPages(),
		pg1: &MainPage{
			con:        tview.NewFlex(),
			streamsCon: tview.NewFlex(),
			infoCon:    tview.NewFlex(),
			twitchList: tview.NewList(),
			strimsList: tview.NewList(),
			streamInfo: tview.NewTextView(),
			infoText:   tview.NewTextView(),
		},
		pg2: &FilterInput{
			con:   tview.NewGrid(),
			input: tview.NewInputField(),
		},
		pg3: &FilterInput{
			con:   tview.NewGrid(),
			input: tview.NewInputField(),
		},
		pg4: &DialogueBox{
			modal: tview.NewModal(),
		},
	}
}

// TODO: Keep track of how old the data is and request new data whenever a new
// one is available (streams.age <= main.refreshTime)
// TODO: Also display this in the UI
func (ui *UI) Run() error {
	err := ui.updateStreams()
	if err != nil {
		return errors.New("no local server running")
	}
	ui.pages.SetBackgroundColor(tcell.ColorDefault)
	ui.setupMainPage()
	ui.setupFilterTwitchPage()
	ui.setupFilterStrimsPage()
	ui.setupRefreshDialoguePage()
	ui.pages.AddPage("Main Window", ui.pg1.con, true, true)
	ui.pages.AddPage("Filter-Twitch", ui.pg2.con, true, false)
	ui.pages.AddPage("Filter-Strims", ui.pg3.con, true, false)
	ui.pages.AddPage("Refresh-Dialogue", ui.pg4.modal, true, false)
	// Force refresh windows by triggering OnChange
	ui.pg3.input.SetText(DefaultRustlerMin)
	ui.pg2.input.SetText(DefaultTwitchFilter)
	err = ui.app.SetRoot(ui.pages, true).Run()
	if err != nil {
		return err
	}
	return nil
}

// TODO: Use SetDrawFunc instead of OnChange + initializing?
//       No need to trigger OnChange initially

func (ui *UI) setupMainPage() {
	ui.pg1.con.AddItem(ui.pg1.streamsCon, 0, 1, true)
	ui.pg1.streamsCon.SetDirection(tview.FlexRow)
	ui.pg1.con.AddItem(ui.pg1.infoCon, 0, 2, false)
	ui.pg1.infoCon.SetDirection(tview.FlexRow)
	// TwitchList
	ui.pg1.streamsCon.AddItem(ui.pg1.twitchList, 0, 3, true)
	ui.pg1.twitchList.SetChangedFunc(ui.updateTwitchStreamInfo)
	ui.pg1.twitchList.SetBackgroundColor(tcell.ColorDefault)
	ui.pg1.twitchList.SetBorder(true)
	ui.pg1.twitchList.SetBorderPadding(0, 0, 1, 1)
	ui.pg1.twitchList.SetInputCapture(ui.listInputHandler)
	ui.pg1.twitchList.SetSecondaryTextColor(tcell.ColorDefault)
	ui.pg1.twitchList.SetTitle("Twitch")
	ui.pg1.twitchList.SetSelectedFocusOnly(true)
	// StrimsList
	ui.pg1.streamsCon.AddItem(ui.pg1.strimsList, 0, 2, false)
	ui.pg1.strimsList.SetChangedFunc(ui.updateStrimsStreamInfo)
	ui.pg1.strimsList.SetBackgroundColor(tcell.ColorDefault)
	ui.pg1.strimsList.SetBorder(true)
	ui.pg1.strimsList.SetBorderPadding(0, 0, 1, 1)
	ui.pg1.strimsList.SetInputCapture(ui.listInputHandler)
	ui.pg1.strimsList.SetSecondaryTextColor(tcell.ColorDefault)
	ui.pg1.strimsList.SetTitle("Strims")
	ui.pg1.strimsList.SetSelectedFocusOnly(true)
	// StreamInfo
	ui.pg1.infoCon.AddItem(ui.pg1.streamInfo, 0, 5, true)
	ui.pg1.streamInfo.SetBackgroundColor(tcell.ColorDefault)
	ui.pg1.streamInfo.SetBorder(true)
	ui.pg1.streamInfo.SetInputCapture(ui.streamInfoInputHandler)
	ui.pg1.streamInfo.SetDynamicColors(true)
	ui.pg1.streamInfo.SetTitle("Stream Info")
	// TextInfo
	ui.pg1.infoCon.AddItem(ui.pg1.infoText, 3, 0, false)
	ui.pg1.infoText.SetBackgroundColor(tcell.ColorDefault)
	ui.pg1.infoText.SetDynamicColors(true)
	ui.pg1.infoText.SetDrawFunc(func(s tcell.Screen, x, y, w, h int) (int, int, int, int) {
		if w < 63 {
			ui.pg1.infoText.Clear()
		} else {
			ui.pg1.infoText.SetText(SHORTCUT_HELP)
		}
		return x, y, w, h
	})
}

func (ui *UI) setupRefreshDialoguePage() {
	ui.pg4.modal.SetBackgroundColor(tcell.ColorDefault)
	ui.pg4.modal.SetText("Force refresh of server's streams?")
	ui.pg4.modal.AddButtons([]string{"Refresh", "Cancel"})
	ui.pg4.modal.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
		// TODO: Can we have better error handling in here?
		switch buttonLabel {
		case "Refresh":
			ui.pg4.modal.SetText("Loading...")
			ui.app.ForceDraw()
			_ = ui.forceRemoteUpdate()
			_ = ui.updateStreams()
			ui.pg4.modal.SetText("Force refresh of server's streams?")
		case "Cancel":
		}
		ui.pages.HidePage("Refresh-Dialogue")
		switch ui.pg1.focusedList {
		case ui.pg1.twitchList:
			ui.refreshStrimsList()
			ui.refreshTwitchList()
		case ui.pg1.strimsList:
			ui.refreshTwitchList()
			ui.refreshStrimsList()
		}
		ui.app.SetFocus(ui.pg1.focusedList)
	})
}
