package main

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sync"

	ls "github.com/HoppenR/libstreams"
	"github.com/rivo/tview"
)

type UI struct {
	app                 *tview.Application
	mainPage            *MainPage
	cmdRegistry         *CommandRegistry
	mapRegistry         *MappingRegistry
	updateStreamsCh     chan struct{}
	forceRemoteUpdateCh chan struct{}
	addr                *url.URL
	wg                  sync.WaitGroup
	mapDepth            int
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
	indexMapping []int
	inverted     bool
}

func (ui *UI) SetAddress(rawAddr string) error {
	u, err := url.Parse(rawAddr)
	if err != nil {
		return err
	}
	ui.addr = u
	return nil
}

func (ui *UI) SetBasicAuthCredentials(user, pass string) error {
	if ui.addr == nil {
		return errors.New("tried adding user and password to invalid address")
	}
	ui.addr.User = url.UserPassword(user, pass)
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
