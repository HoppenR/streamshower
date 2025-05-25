package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

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

	updateStreamsCh     chan struct{}
	forceRemoteUpdateCh chan struct{}
	wg                  sync.WaitGroup
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

func (ui *UI) updateStreams(ctx context.Context) error {
	streams, err := sc.GetServerData(ctx, ui.addr)
	if err != nil {
		return err
	}
	sort.Sort(sort.Reverse(streams.Twitch))
	sort.Sort(sort.Reverse(streams.Strims))
	ui.pg1.streams = streams
	return nil
}

func (ui *UI) forceRemoteUpdate(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ui.addr, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/octet-stream")

	var resp *http.Response
	resp, err = http.DefaultClient.Do(req)
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
		updateStreamsCh:     make(chan struct{}),
		forceRemoteUpdateCh: make(chan struct{}),
	}
}

func (ui *UI) Run() error {
	// Set title to "Streamshower"
	fmt.Print("\033]2;Streamshower\a")

	// Run initial data fetch in main goroutine
	err := ui.updateStreams(context.Background())
	if err != nil {
		return fmt.Errorf("no server running (%s)", err)
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
	ui.app.SetRoot(ui.pages, true)

	// NOTE: These are in-order (LIFO) deferred calls
	ctx, cancel := context.WithCancel(context.Background())
	defer ui.wg.Wait()
	defer cancel()

	// Set up remote update checking
	ui.wg.Add(1)
	go ui.streamUpdateLoop(ctx)

	if err = ui.app.Run(); err != nil {
		return err
	}

	return nil
}

func (ui *UI) streamUpdateLoop(ctx context.Context) {
	// TODO: Can we have better error handling in this goroutine?
	defer ui.wg.Done()

	var err error
	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		var nextUpdate time.Time
		nextUpdate = ui.pg1.streams.LastFetched.Add(ui.pg1.streams.RefreshInterval)
		timer.Reset(time.Until(nextUpdate))
		select {
		case <-ctx.Done():
			return
		case <-ui.forceRemoteUpdateCh:
			err = ui.forceRemoteUpdate(ctx)
			if errors.Is(err, context.Canceled) {
				return
			} else if err != nil {
				panic(err)
			}
			continue
		case <-ui.updateStreamsCh:
			// pass
		case <-timer.C:
			// pass
		}

		err = ui.updateStreams(ctx)
		if errors.Is(err, context.Canceled) {
			return
		} else if err != nil {
			panic(err)
		}

		ui.app.QueueUpdateDraw(func() {
			switch ui.pg1.focusedList {
			case ui.pg1.twitchList:
				ui.refreshStrimsList()
				ui.refreshTwitchList()
			case ui.pg1.strimsList:
				ui.refreshTwitchList()
				ui.refreshStrimsList()
			}
		})
	}
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
	ui.pg1.streamInfo.SetTitle("Stream Info (" + ui.addr + ")")
	// TextInfo
	ui.pg1.infoCon.AddItem(ui.pg1.infoText, 3, 0, false)
	ui.pg1.infoText.SetBackgroundColor(tcell.ColorDefault)
	ui.pg1.infoText.SetDynamicColors(true)
	ui.pg1.infoText.SetDrawFunc(func(s tcell.Screen, x, y, w, h int) (int, int, int, int) {
		if w < 90 {
			ui.pg1.infoText.Clear()
		} else {
			ui.pg1.infoText.SetText(
				SHORTCUT_HELP +
					strings.Repeat(" ", 16) +
					ui.pg1.streams.LastFetched.Format(time.Stamp),
			)
		}
		return x, y, w, h
	})
}

func (ui *UI) setupRefreshDialoguePage() {
	ui.pg4.modal.SetBackgroundColor(tcell.ColorDefault)
	ui.pg4.modal.SetText("Force refresh of server's streams?")
	buttonLabels := []string{"Refresh", "Refresh Server Data", "Cancel"}
	ui.pg4.modal.AddButtons(buttonLabels)
	ui.pg4.modal.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
		if buttonIndex == 0 || buttonIndex == 1 {
			ui.pg4.modal.SetText("Loading...")
			ui.app.ForceDraw()
			if buttonIndex == 1 {
				ui.forceRemoteUpdateCh <- struct{}{}
			}
			ui.updateStreamsCh <- struct{}{}
			ui.pg4.modal.SetText("Force refresh of server's streams?")
		}
		ui.pages.HidePage("Refresh-Dialogue")
		ui.app.SetFocus(ui.pg1.focusedList)
	})
}
