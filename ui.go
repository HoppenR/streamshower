package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	ls "github.com/HoppenR/libstreams"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type UI struct {
	app          *tview.Application
	mainPage     *MainPage
	twitchFilter *FilterInput
	strimsFilter *FilterInput
	addr         string
	cmdRegistry  *CommandRegistry
	mapRegistry  *MappingRegistry

	updateStreamsCh     chan struct{}
	forceRemoteUpdateCh chan struct{}
	wg                  sync.WaitGroup
}

type FilterInput struct {
	input    string
	inverted bool
}

type MainPage struct {
	con           *tview.Flex
	streamsCon    *tview.Flex
	infoCon       *tview.Flex
	commandRow    *tview.Flex
	commandLine   *tview.InputField
	appStatusText *tview.TextView
	fetchTimeView *tview.TextView
	streamInfo    *tview.TextView
	strimsList    *tview.List
	twitchList    *tview.List

	focusedList   *tview.List // Can either be strimsList or twitchList
	streams       *ls.Streams
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

	streams, err := ls.GetServerData(ctxTo, ui.addr)
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
		app: tview.NewApplication(),
		mainPage: &MainPage{
			appStatusText: tview.NewTextView(),
			commandLine:   tview.NewInputField(),
			commandRow:    tview.NewFlex(),
			con:           tview.NewFlex(),
			fetchTimeView: tview.NewTextView(),
			infoCon:       tview.NewFlex(),
			streamInfo:    tview.NewTextView(),
			streamsCon:    tview.NewFlex(),
			strimsList:    tview.NewList(),
			twitchList:    tview.NewList(),
			streams:       new(ls.Streams),
			strimsVisible: true,
		},
		twitchFilter:        &FilterInput{},
		strimsFilter:        &FilterInput{},
		cmdRegistry:         NewCommandRegistry(),
		mapRegistry:         NewMappingRegistry(),
		updateStreamsCh:     make(chan struct{}, 1),
		forceRemoteUpdateCh: make(chan struct{}, 1),
	}
	ui.mainPage.focusedList = ui.mainPage.twitchList
	return ui
}

func (ui *UI) Run() error {
	// Set title to "Streamshower"
	fmt.Print("\033]2;Streamshower\a")

	ui.setupMainPage()
	ui.app.SetRoot(ui.mainPage.con, true)

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
	fetchTimer := time.NewTimer(0)
	defer fetchTimer.Stop()
	redrawTimer := time.NewTicker(time.Second)
	defer redrawTimer.Stop()
	for {
		if !ui.mainPage.streams.LastFetched.IsZero() {
			var nextUpdate time.Time
			nextUpdate = ui.mainPage.streams.LastFetched.Add(ui.mainPage.streams.RefreshInterval)
			fetchTimer.Reset(time.Until(nextUpdate))
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
			}
			continue
		case <-redrawTimer.C:
			ui.app.Draw()
			continue
		case <-ui.updateStreamsCh:
			// pass
		case <-fetchTimer.C:
			// pass
		}

		setStatus("orange", "Fetching streams...")
		err = ui.updateStreams(ctx)
		if errors.Is(err, context.Canceled) {
			return
		} else if errors.Is(err, ls.ErrAuthPending) {
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
	ui.mainPage.con.SetDirection(tview.FlexColumn)
	ui.mainPage.streamsCon.SetDirection(tview.FlexRow)
	ui.mainPage.infoCon.SetDirection(tview.FlexRow)
	ui.mainPage.commandRow.SetDirection(tview.FlexColumn)
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
	ui.mainPage.infoCon.AddItem(ui.mainPage.streamInfo, 0, 1, false)
	ui.mainPage.streamInfo.SetBackgroundColor(tcell.ColorDefault)
	ui.mainPage.streamInfo.SetBorder(true)
	ui.mainPage.streamInfo.SetInputCapture(ui.streamInfoInputHandler)
	ui.mainPage.streamInfo.SetDynamicColors(true)
	ui.mainPage.streamInfo.SetTitle("Stream Info")
	// CommandRow
	ui.mainPage.infoCon.AddItem(ui.mainPage.commandRow, 1, 0, false)
	ui.mainPage.commandRow.AddItem(ui.mainPage.commandLine, 0, 1, true)
	ui.mainPage.commandRow.AddItem(ui.mainPage.fetchTimeView, 26, 0, false)
	// CommandLine
	ui.mainPage.commandLine.SetText(" Please see `:help` or `:map`!")
	ui.mainPage.commandLine.SetFieldBackgroundColor(tcell.ColorBlack)
	ui.mainPage.commandLine.SetChangedFunc(ui.parseCommandChain)
	ui.mainPage.commandLine.SetFinishedFunc(ui.execCommandChainCallback)
	ui.mainPage.commandLine.SetInputCapture(ui.commandLineCapture)
	ui.mainPage.commandLine.SetAutocompletedFunc(ui.commandLineCompleteDone)
	ui.mainPage.commandLine.SetAutocompleteFunc(ui.commandLineComplete)
	// Fetch time view
	ui.mainPage.fetchTimeView.SetBackgroundColor(tcell.ColorOrange)
	ui.mainPage.fetchTimeView.SetTextColor(tcell.ColorBlack)
	ui.mainPage.fetchTimeView.SetText("No timing data ")
	ui.mainPage.fetchTimeView.SetTextAlign(tview.AlignRight)
	ui.mainPage.fetchTimeView.SetDrawFunc(func(screen tcell.Screen, x, y, width, height int) (int, int, int, int) {
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
	})
}

func (ui *UI) commandLineCapture(event *tcell.EventKey) *tcell.EventKey {
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
