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
	ixs := m.matchStrimsListIndex(filter)
	if ixs == nil {
		m.strimsList.AddItem("", "", 0, nil)
		return
	}
	for _, v := range ixs {
		mainstr := m.streams.Strims.Data[v].Channel
		color := "green"
		if m.streams.Strims.Data[v].Nsfw {
			color = "red"
		}
		secstr := fmt.Sprintf(
			" %-6d[%s:-:u]%s[-:-:-]",
			m.streams.Strims.Data[v].Rustlers,
			color,
			tview.Escape(m.streams.Strims.Data[v].Title),
		)
		m.strimsList.AddItem(mainstr, secstr, 0, nil)
	}
}

func (m *MainPage) updateStrimsStreamInfo(ix int, pri, sec string, _ rune) {
	var index int = -1
	for i, v := range m.streams.Strims.Data {
		if pri == v.Channel {
			index = i
			break
		}
	}
	add := func(c string) {
		m.streamInfo.Write([]byte(c))
	}
	m.streamInfo.Clear()
	if index == -1 {
		m.streamInfo.SetTitle("Stream Info")
		add("No results")
	} else {
		selStream := m.streams.Strims.Data[index]
		if selStream.Service == "m3u8" {
			selStream.Title = selStream.URL
		} else {
			selStream.Title = strings.ReplaceAll(selStream.Title, "\n", " ")
		}
		m.streamInfo.SetTitle(selStream.Channel)
		add(fmt.Sprintf("[red]Title[-]: %s\n", tview.Escape(selStream.Title)))
		add(fmt.Sprintf("[red]Rustlers[-]: %d [lightgray](%d afk)[-]\n", selStream.Rustlers, selStream.AfkRustlers))
		add(fmt.Sprintf("[red]Service[-]: %s\n", selStream.Service))
		add(fmt.Sprintf("[red]Viewers[-]: %v\n", selStream.Viewers))
		add(fmt.Sprintf("[red]Live[-]: %v\n", selStream.Live))
		add(fmt.Sprintf("[red]AFK[-]: %v\n", selStream.Afk))
	}
}

func (ui *UI) toggleStrimsList() {
	if ui.mainPage.strimsVisible {
		ui.disableStrimsList()
	} else {
		ui.enableStrimsList()
	}
}

func (ui *UI) enableStrimsList() {
	if !ui.mainPage.strimsVisible {
		ui.mainPage.streamsCon.AddItem(ui.mainPage.strimsList, 0, 2, false)
		ui.mainPage.strimsVisible = true
	}
}

func (ui *UI) disableStrimsList() {
	if ui.mainPage.strimsVisible {
		ui.mainPage.streamsCon.RemoveItem(ui.mainPage.strimsList)
		ui.mainPage.strimsVisible = false
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
