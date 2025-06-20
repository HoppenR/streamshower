package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/rivo/tview"
)

func (m *MainPage) refreshStrimsList() {
	if m.focusedList != m.strimsList {
		m.strimsList.SetChangedFunc(nil)
		defer m.strimsList.SetChangedFunc(m.updateStrimsStreamInfo)
	}
	oldIdx := m.strimsList.GetCurrentItem()
	m.updateStrimsList(m.strimsFilter.input)
	m.strimsList.SetCurrentItem(oldIdx)
}

func (m *MainPage) updateStrimsList(filter string) {
	m.strimsList.Clear()
	m.strimsFilter.indexMapping = m.matchStrimsListIndex(filter)
	if m.strimsFilter.indexMapping == nil {
		m.strimsList.AddItem("", "", 0, nil)
		return
	}
	for _, v := range m.strimsFilter.indexMapping {
		stream := m.streams.Strims.Data[v]
		mainstr := highlightSearch(stream.Channel, m.lastSearch)
		secColor := "green"
		if stream.Nsfw {
			secColor = "red"
		}
		secstr := fmt.Sprintf(
			" %-6d[%s:-:u]%s[-:-:-]",
			stream.Rustlers,
			secColor,
			tview.Escape(stream.Title),
		)
		m.strimsList.AddItem(mainstr, secstr, 0, nil)
	}
}

func (m *MainPage) updateStrimsStreamInfo(tviewIx int, pri, sec string, _ rune) {
	add := func(c string) {
		m.streamInfo.Write([]byte(c))
	}
	m.streamInfo.Clear()
	if m.strimsFilter.indexMapping == nil {
		m.streamInfo.SetTitle("Stream Info")
		add("No results")
		return
	}
	ix := m.strimsFilter.indexMapping[tviewIx]
	stream := m.streams.Strims.Data[ix]
	var title string
	if stream.Service == "m3u8" {
		title = stream.URL
	} else {
		title = strings.ReplaceAll(stream.Title, "\n", " ")
	}
	title = removeVariationSelectors(title)
	title = tview.Escape(title)
	m.streamInfo.SetTitle(stream.Channel)
	add(fmt.Sprintf("[red]Title[-]: %s\n", title))
	add(fmt.Sprintf("[red]Rustlers[-]: %d [lightgray](%d afk)[-]\n", stream.Rustlers, stream.AfkRustlers))
	add(fmt.Sprintf("[red]Service[-]: %s\n", stream.Service))
	add(fmt.Sprintf("[red]Viewers[-]: %v\n", stream.Viewers))
	add(fmt.Sprintf("[red]Live[-]: %v\n", stream.Live))
	add(fmt.Sprintf("[red]AFK[-]: %v\n", stream.Afk))
}

func (ui *UI) toggleStrimsList() {
	if ui.mainPage.strims {
		ui.disableStrimsList()
	} else {
		ui.enableStrimsList()
	}
}

func (ui *UI) enableStrimsList() {
	if !ui.mainPage.strims {
		ui.mainPage.streamsCon.AddItem(ui.mainPage.strimsList, 0, 2, false)
		ui.mainPage.strims = true
	}
}

func (ui *UI) disableStrimsList() {
	if ui.mainPage.strims {
		ui.mainPage.streamsCon.RemoveItem(ui.mainPage.strimsList)
		ui.mainPage.strims = false
	}
}

func (m *MainPage) matchStrimsListIndex(filter string) []int {
	var ixs []int
	re, err := regexp.Compile(`(?i)` + filter)
	if err != nil {
		for i := range m.streams.Strims.Data {
			ixs = append(ixs, i)
		}
		return ixs
	}
	for i, v := range m.streams.Strims.Data {
		var match func(string) bool = re.MatchString
		matched := match(v.Service) || match(v.Title) || match(v.Channel)
		if matched != m.strimsFilter.inverted {
			ixs = append(ixs, i)
		}
	}
	return ixs
}
