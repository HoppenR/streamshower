package main

import (
	"context"
	"fmt"
	"sync"

	ls "github.com/HoppenR/libstreams"
	"github.com/rivo/tview"
)

type UI struct {
	app         *tview.Application
	mainPage    *MainPage
	addr        string
	cmdRegistry *CommandRegistry
	mapRegistry *MappingRegistry
	mapDepth    int

	updateStreamsCh     chan struct{}
	forceRemoteUpdateCh chan struct{}
	wg                  sync.WaitGroup
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

	focusedList  *tview.List // Can either be strimsList or twitchList
	streams      *ls.Streams
	twitchFilter *FilterInput
	strimsFilter *FilterInput
	lastSearch   string

	// :set options
	strims  bool
	winopen bool
}

type FilterInput struct {
	input        string
	inverted     bool
	indexMapping []int
}

func (ui *UI) SetAddress(address string) *UI {
	ui.addr = address
	return ui
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
			twitchFilter:  &FilterInput{},
			strimsFilter:  &FilterInput{},
			streams: &ls.Streams{
				Twitch: new(ls.TwitchStreams),
				Strims: new(ls.StrimsStreams),
			},
			strims: true,
		},
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
