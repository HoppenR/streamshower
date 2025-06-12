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
	if m.focusedList != m.twitchList {
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
		stream := m.streams.Twitch.Data[v]
		mainstr := highlightSearch(stream.UserName, m.lastSearch)
		secstr := fmt.Sprintf(
			" %-6d[green:-:u]%s[-:-:-]",
			stream.ViewerCount,
			tview.Escape(stream.GameName),
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
	stream := m.streams.Twitch.Data[ix]
	startLocal := stream.StartedAt.Local()
	if stream.GameName == "" {
		stream.GameName = "[::d]None[::-]"
	}
	title := strings.ReplaceAll(stream.Title, "\n", " ")
	title = removeVariationSelectors(title)
	title = tview.Escape(title)
	m.streamInfo.SetTitle(stream.UserName)
	add(fmt.Sprintf("[red]Title[-]: %s\n", title))
	add(fmt.Sprintf("[red]Viewers[-]: %d\n", stream.ViewerCount))
	add(fmt.Sprintf("[red]Game[-]: %s\n", stream.GameName))
	add(fmt.Sprintf(
		"[red]Started At[-]: %2.2d:%2.2d [lightgray](%.0fd %.0fh %0.fm ago)[-]\n",
		startLocal.Hour(),
		startLocal.Minute(),
		math.Floor(time.Since(startLocal).Hours()/24),
		math.Mod(math.Floor(time.Since(startLocal).Hours()), 24),
		math.Mod(math.Floor(time.Since(startLocal).Minutes()), 60)),
	)
	add(fmt.Sprintf("[red]Language[-]: %s\n", stream.Language))
	add(fmt.Sprintf("[red]Type[-]: %s\n", stream.Type))
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
