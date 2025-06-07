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

type UI struct {
	app          *tview.Application
	pages        *tview.Pages
	mainPage     *MainPage
	twitchFilter *FilterInput
	strimsFilter *FilterInput
	addr         string
	cmdRegistry  *CommandRegistry

	updateStreamsCh     chan struct{}
	forceRemoteUpdateCh chan struct{}
	wg                  sync.WaitGroup
}

type FilterInput struct {
	input    string
	inverted bool
}

type DialogueBox struct {
	modal *tview.Modal
}

type MainPage struct {
	con             *tview.Flex
	focusedList     *tview.List
	streamInfo      *tview.TextView
	streams         *sc.Streams
	streamsCon      *tview.Flex
	infoCon         *tview.Flex
	strimsList      *tview.List
	twitchList      *tview.List
	keybindInfoText *tview.TextView
	appStatusText   *tview.TextView
	commandLine     *tview.InputField

	lastSearch    string
	strimsVisible bool
}

func (ui *UI) SetAddress(address string) *UI {
	ui.addr = address
	return ui
}

func (ui *UI) updateStreams(ctx context.Context) error {
	ctxTo, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	streams, err := sc.GetServerData(ctxTo, ui.addr)
	if err != nil {
		return err
	}
	sort.Sort(sort.Reverse(streams.Twitch))
	sort.Sort(sort.Reverse(streams.Strims))
	ui.mainPage.streams = streams
	return nil
}

func (ui *UI) forceRemoteUpdate(ctx context.Context) error {
	ctxTo, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctxTo, http.MethodPost, ui.addr, nil)
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
	ui := &UI{
		app:   tview.NewApplication(),
		pages: tview.NewPages(),
		mainPage: &MainPage{
			con:             tview.NewFlex(),
			streamsCon:      tview.NewFlex(),
			infoCon:         tview.NewFlex(),
			twitchList:      tview.NewList(),
			strimsList:      tview.NewList(),
			streamInfo:      tview.NewTextView(),
			keybindInfoText: tview.NewTextView(),
			appStatusText:   tview.NewTextView(),
			commandLine:     tview.NewInputField(),
			streams:         new(sc.Streams),
			strimsVisible:   true,
		},
		twitchFilter:        &FilterInput{},
		strimsFilter:        &FilterInput{},
		cmdRegistry:         NewCommandRegistry(),
		updateStreamsCh:     make(chan struct{}, 1),
		forceRemoteUpdateCh: make(chan struct{}, 1),
	}
	ui.mainPage.focusedList = ui.mainPage.twitchList
	return ui
}

func (ui *UI) Run() error {
	// Set title to "Streamshower"
	fmt.Print("\033]2;Streamshower\a")

	ui.pages.SetBackgroundColor(tcell.ColorDefault)
	ui.setupMainPage()
	ui.pages.AddPage("Main Window", ui.mainPage.con, true, true)
	ui.app.SetRoot(ui.pages, true)

	// NOTE: These are in-order (LIFO) deferred calls
	ctx, cancel := context.WithCancel(context.Background())
	defer ui.wg.Wait()
	defer cancel()

	// Set up remote update checking
	ui.wg.Add(1)
	go ui.streamUpdateLoop(ctx)

	if err := ui.app.Run(); err != nil {
		return err
	}

	return nil
}

func (ui *UI) streamUpdateLoop(ctx context.Context) {
	setStatus := func(color string, text string) {
		ui.app.QueueUpdateDraw(func() {
			ui.mainPage.appStatusText.SetText(fmt.Sprintf("[%s]%s[-]", color, text))
		})
	}
	defer ui.wg.Done()

	var err error
	timer := time.NewTimer(0)
	defer timer.Stop()

	for {
		if !ui.mainPage.streams.LastFetched.IsZero() {
			var nextUpdate time.Time
			nextUpdate = ui.mainPage.streams.LastFetched.Add(ui.mainPage.streams.RefreshInterval)
			timer.Reset(time.Until(nextUpdate))
		}
		select {
		case <-ctx.Done():
			return
		case <-ui.forceRemoteUpdateCh:
			setStatus("orange", "Sending update...")
			err = ui.forceRemoteUpdate(ctx)
			if errors.Is(err, context.Canceled) {
				return
			} else if err != nil {
				setStatus("red", fmt.Sprintf("Error updating: %s", err))
				continue
			}
			continue
		case <-ui.updateStreamsCh:
			// pass
		case <-timer.C:
			// pass
		}

		setStatus("orange", "Fetching streams...")
		err = ui.updateStreams(ctx)
		if errors.Is(err, context.Canceled) {
			return
		} else if errors.Is(err, sc.ErrAuthPending) {
			setStatus("yellow", "run `:sync` to refresh after authenticating")
			continue
		} else if err != nil {
			setStatus("red", fmt.Sprintf("Error fetching: %s", err))
			continue
		}

		ui.app.QueueUpdate(func() {
			if ui.mainPage.streams.Strims.Len() == 0 {
				ui.app.SetFocus(ui.mainPage.twitchList)
				ui.mainPage.focusedList = ui.mainPage.twitchList
				ui.disableStrimsList()
			} else {
				ui.enableStrimsList()
			}
			ui.refreshTwitchList()
			ui.refreshStrimsList()
		})
		setStatus("green", fmt.Sprintf(
			"Fetched %d Twitch streams and %d Strims streams",
			ui.mainPage.streams.Twitch.Len(),
			ui.mainPage.streams.Strims.Len(),
		))
	}
}

func (ui *UI) setupMainPage() {
	ui.mainPage.con.AddItem(ui.mainPage.streamsCon, 0, 1, true)
	ui.mainPage.con.AddItem(ui.mainPage.infoCon, 0, 2, false)
	ui.mainPage.streamsCon.SetDirection(tview.FlexRow)
	ui.mainPage.infoCon.SetDirection(tview.FlexRow)
	// TwitchList
	ui.mainPage.streamsCon.AddItem(ui.mainPage.twitchList, 0, 3, true)
	ui.mainPage.twitchList.SetChangedFunc(ui.updateTwitchStreamInfo)
	ui.mainPage.twitchList.SetBackgroundColor(tcell.ColorDefault)
	ui.mainPage.twitchList.SetBorder(true)
	ui.mainPage.twitchList.SetBorderPadding(0, 0, 1, 1)
	ui.mainPage.twitchList.SetInputCapture(ui.listInputHandler)
	ui.mainPage.twitchList.SetSecondaryTextColor(tcell.ColorDefault)
	ui.mainPage.twitchList.SetTitle("Twitch")
	ui.mainPage.twitchList.SetSelectedFocusOnly(true)
	// StrimsList
	ui.mainPage.streamsCon.AddItem(ui.mainPage.strimsList, 0, 2, false)
	ui.mainPage.strimsList.SetChangedFunc(ui.updateStrimsStreamInfo)
	ui.mainPage.strimsList.SetBackgroundColor(tcell.ColorDefault)
	ui.mainPage.strimsList.SetBorder(true)
	ui.mainPage.strimsList.SetBorderPadding(0, 0, 1, 1)
	ui.mainPage.strimsList.SetInputCapture(ui.listInputHandler)
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
	ui.mainPage.infoCon.AddItem(ui.mainPage.streamInfo, 0, 1, true)
	ui.mainPage.streamInfo.SetBackgroundColor(tcell.ColorDefault)
	ui.mainPage.streamInfo.SetBorder(true)
	ui.mainPage.streamInfo.SetInputCapture(ui.streamInfoInputHandler)
	ui.mainPage.streamInfo.SetDynamicColors(true)
	ui.mainPage.streamInfo.SetTitle("Stream Info")
	// TextInfo
	ui.mainPage.infoCon.AddItem(ui.mainPage.keybindInfoText, 3, 0, false)
	ui.mainPage.keybindInfoText.SetBackgroundColor(tcell.ColorDefault)
	ui.mainPage.keybindInfoText.SetDynamicColors(true)
	ui.mainPage.keybindInfoText.SetDrawFunc(ui.updateMainKeybindInfo)
	ui.mainPage.streamInfo.SetFocusFunc(func() {
		ui.mainPage.keybindInfoText.SetDrawFunc(ui.updateInfowinKeybindInfo)
	})
	ui.mainPage.streamInfo.SetBlurFunc(func() {
		ui.mainPage.keybindInfoText.SetDrawFunc(ui.updateMainKeybindInfo)
	})
	// CommandLine
	ui.mainPage.infoCon.AddItem(ui.mainPage.commandLine, 1, 0, true)
	ui.mainPage.commandLine.SetFieldTextColor(tcell.ColorForestGreen)
	ui.mainPage.commandLine.SetFieldBackgroundColor(tcell.ColorBlack)
	ui.mainPage.commandLine.SetChangedFunc(ui.parseCommandChain)
	ui.mainPage.commandLine.SetFinishedFunc(ui.execCommandChainCallback)
	ui.mainPage.commandLine.SetAutocompletedFunc(func(text string, index int, source int) bool {
		if source == tview.AutocompletedEnter {
			ui.execCommand(text)
			return true
		} else if source == tview.AutocompletedTab {
			ui.mainPage.commandLine.SetText(text + " ")
			return false
		}
		return false
	})
	ui.mainPage.commandLine.SetAutocompleteFunc(func(currentText string) []string {
		if currentText == "" {
			return nil
		}
		fields := strings.Split(currentText, " ")
		switch len(fields) {
		case 1:
			possibleCmds := ui.cmdRegistry.matchPossibleCommands(strings.TrimLeft(fields[0], ":"))
			entries := make([]string, 0, len(possibleCmds))
			for _, cmd := range possibleCmds {
				entries = append(entries, ":"+cmd.Name)
			}
			return entries
		case 2:
			possibleCmds := ui.cmdRegistry.matchPossibleCommands(strings.TrimLeft(fields[0], ":"))
			if len(possibleCmds) == 1 {
				if possibleCmds[0].Complete != nil {
					return possibleCmds[0].Complete(ui, fields[1])
				}
			}
		}
		return nil
	})
}

func (ui *UI) updateMainKeybindInfo(s tcell.Screen, x, y, w, h int) (int, int, int, int) {
	ui.mainPage.keybindInfoText.Clear()
	if w >= 90 {
		ui.mainPage.keybindInfoText.Write([]byte(SHORTCUT_MAINWIN_HELP))
		if !ui.mainPage.streams.LastFetched.IsZero() {
			ui.mainPage.keybindInfoText.Write([]byte(strings.Repeat(" ", 25)))
			ui.mainPage.keybindInfoText.Write([]byte(ui.mainPage.streams.LastFetched.In(time.Local).Format(time.Stamp)))
		}
	}
	return x, y, w, h
}

func (ui *UI) updateInfowinKeybindInfo(s tcell.Screen, x, y, w, h int) (int, int, int, int) {
	ui.mainPage.keybindInfoText.Clear()
	if w >= 15 {
		ui.mainPage.keybindInfoText.Write([]byte(SHORTCUT_INFOWIN_HELP))
		if !ui.mainPage.streams.LastFetched.IsZero() {
			ui.mainPage.keybindInfoText.Write([]byte(ui.mainPage.streams.LastFetched.In(time.Local).Format(time.Stamp)))
		}
	}
	return x, y, w, h
}
