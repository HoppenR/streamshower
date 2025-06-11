package main

import (
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"

	"github.com/rivo/tview"
)

func (m *MainPage) refreshTwitchList() {
	if m.twitchList.HasFocus() {
		m.twitchList.SetChangedFunc(nil)
		defer m.twitchList.SetChangedFunc(m.updateTwitchStreamInfo)
	}
	oldIdx := m.twitchList.GetCurrentItem()
	m.updateTwitchList(m.twitchFilter.input)
	m.twitchList.SetCurrentItem(oldIdx)
}

func (m *MainPage) updateTwitchList(filter string) {
	m.twitchList.Clear()
	m.twitchFilter.indexMapping = m.matchTwitchListIndex(filter)
	if m.twitchFilter.indexMapping == nil {
		m.twitchList.AddItem("", "", 0, nil)
		return
	}
	for _, v := range m.twitchFilter.indexMapping {
		mainstr := highlightSearch(m.streams.Twitch.Data[v].UserName, m.lastSearch)
		secstr := fmt.Sprintf(
			" %-6d[green:-:u]%s[-:-:-]",
			m.streams.Twitch.Data[v].ViewerCount,
			tview.Escape(m.streams.Twitch.Data[v].GameName),
		)
		m.twitchList.AddItem(mainstr, secstr, 0, nil)
	}
}

func (m *MainPage) updateTwitchStreamInfo(tviewIx int, pri, sec string, _ rune) {
	add := func(c string) {
		m.streamInfo.Write([]byte(c))
	}
	m.streamInfo.Clear()
	if m.twitchFilter.indexMapping == nil {
		m.streamInfo.SetTitle("Stream Info")
		add("No results")
		return
	}
	ix := m.twitchFilter.indexMapping[tviewIx]
	selStream := m.streams.Twitch.Data[ix]
	startLocal := selStream.StartedAt.Local()
	if selStream.GameName == "" {
		selStream.GameName = "[::d]None[::-]"
	}
	title := strings.ReplaceAll(selStream.Title, "\n", " ")
	title = tview.Escape(title)
	m.streamInfo.SetTitle(selStream.UserName)
	add(fmt.Sprintf("[red]Title[-]: %s\n", title))
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

func (m *MainPage) matchTwitchListIndex(filter string) []int {
	var ixs []int
	re, err := regexp.Compile(`(?i)` + filter)
	if err != nil {
		for i := range m.streams.Twitch.Data {
			ixs = append(ixs, i)
		}
		return ixs
	}
	for i, v := range m.streams.Twitch.Data {
		var match func(string) bool = re.MatchString
		matched := match(v.GameName) || match(v.Title) || match(v.UserName)
		if matched != m.twitchFilter.inverted {
			ixs = append(ixs, i)
		}
	}
	return ixs
}
