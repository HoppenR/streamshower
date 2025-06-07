package main

import (
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"

	"github.com/rivo/tview"
)

func (ui *UI) refreshTwitchList() {
	if ui.mainPage.focusedList != ui.mainPage.twitchList {
		ui.mainPage.twitchList.SetChangedFunc(nil)
		defer ui.mainPage.twitchList.SetChangedFunc(ui.updateTwitchStreamInfo)
	}
	oldIdx := ui.mainPage.twitchList.GetCurrentItem()
	ui.updateTwitchList(ui.twitchFilter.input)
	ui.mainPage.twitchList.SetCurrentItem(oldIdx)
}

func (ui *UI) updateTwitchList(filter string) {
	ui.mainPage.twitchList.Clear()
	ixs := ui.matchTwitchListIndex(filter)
	if ixs == nil {
		ui.mainPage.twitchList.AddItem("", "", 0, nil)
		return
	}
	for _, v := range ixs {
		mainstr := ui.mainPage.streams.Twitch.Data[v].UserName
		secstr := fmt.Sprintf(
			" %-6d[green:-:u]%s[-:-:-]",
			ui.mainPage.streams.Twitch.Data[v].ViewerCount,
			tview.Escape(ui.mainPage.streams.Twitch.Data[v].GameName),
		)
		ui.mainPage.twitchList.AddItem(mainstr, secstr, 0, nil)
	}
}

func (ui *UI) updateTwitchStreamInfo(ix int, pri, sec string, _ rune) {
	var index int = -1
	for i, v := range ui.mainPage.streams.Twitch.Data {
		if pri == v.UserName {
			index = i
			break
		}
	}
	add := func(c string) {
		ui.mainPage.streamInfo.Write([]byte(c))
	}

	// {
	// 	// NOTE: Make sure that all multibyte characters are cleared, by
	// 	// adding text before clearing
	// 	ui.mainPage.streamInfo.SetText(strings.Repeat("#", 500))
	// 	ui.app.ForceDraw()
	// }

	ui.mainPage.streamInfo.Clear()
	if index == -1 {
		ui.mainPage.streamInfo.SetTitle("Stream Info")
		add("No results")
	} else {
		selStream := ui.mainPage.streams.Twitch.Data[index]
		startLocal := selStream.StartedAt.Local()
		if selStream.GameName == "" {
			selStream.GameName = "[::d]None[::-]"
		}
		selStream.Title = strings.ReplaceAll(selStream.Title, "\n", " ")
		ui.mainPage.streamInfo.SetTitle(selStream.UserName)
		add(fmt.Sprintf("[red]Title[-]: %s\n", tview.Escape(selStream.Title)))
		add(fmt.Sprintf("[red]Viewers[-]: %d\n", selStream.ViewerCount))
		add(fmt.Sprintf("[red]Game[-]: %s\n", selStream.GameName))
		add(fmt.Sprintf(
			"[red]Started At[-]: %2.2d:%2.2d [lightgray](%.0fd %.0fh %0.fm ago)[-]\n",
			startLocal.Hour(),
			startLocal.Minute(),
			math.Floor(time.Since(startLocal).Hours()/24),
			math.Mod(math.Floor(time.Since(startLocal).Hours()), 24),
			math.Mod(math.Floor(time.Since(startLocal).Minutes()), 60)),
		)
		add(fmt.Sprintf("[red]Language[-]: %s\n", selStream.Language))
		add(fmt.Sprintf("[red]Type[-]: %s\n", selStream.Type))
	}
}

func (ui *UI) matchTwitchListIndex(filter string) []int {
	var ixs []int
	re, err := regexp.Compile("(?i)" + filter)
	if err != nil {
		for i := range ui.mainPage.streams.Twitch.Data {
			ixs = append(ixs, i)
		}
		return ixs
	}
	for i, v := range ui.mainPage.streams.Twitch.Data {
		var match func(string) bool = re.MatchString
		matched := match(v.GameName) || match(v.Title) || match(v.UserName)
		if matched != ui.twitchFilter.inverted {
			ixs = append(ixs, i)
		}
	}
	return ixs
}
