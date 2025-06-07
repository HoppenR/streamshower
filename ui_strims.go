package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/rivo/tview"
)

func (ui *UI) refreshStrimsList() {
	if ui.mainPage.focusedList != ui.mainPage.strimsList {
		ui.mainPage.strimsList.SetChangedFunc(nil)
		defer ui.mainPage.strimsList.SetChangedFunc(ui.updateStrimsStreamInfo)
	}
	oldIdx := ui.mainPage.strimsList.GetCurrentItem()
	ui.updateStrimsList(ui.strimsFilter.input)
	ui.mainPage.strimsList.SetCurrentItem(oldIdx)
}

func (ui *UI) updateStrimsList(filter string) {
	ui.mainPage.strimsList.Clear()
	ixs := ui.matchStrimsListIndex(filter)
	if ixs == nil {
		ui.mainPage.strimsList.AddItem("", "", 0, nil)
		return
	}
	for ix := range ixs {
		mainstr := ui.mainPage.streams.Strims.Data[ix].Channel
		color := "green"
		if ui.mainPage.streams.Strims.Data[ix].Nsfw {
			color = "red"
		}
		secstr := fmt.Sprintf(
			" %-6d[%s:-:u]%s[-:-:-]",
			ui.mainPage.streams.Strims.Data[ix].Rustlers,
			color,
			tview.Escape(ui.mainPage.streams.Strims.Data[ix].Title),
		)
		ui.mainPage.strimsList.AddItem(mainstr, secstr, 0, nil)
	}
}

func (ui *UI) updateStrimsStreamInfo(ix int, pri, sec string, _ rune) {
	var index int = -1
	for i, v := range ui.mainPage.streams.Strims.Data {
		if pri == v.Channel {
			index = i
			break
		}
	}
	add := func(c string) {
		ui.mainPage.streamInfo.Write([]byte(c))
	}
	ui.mainPage.streamInfo.Clear()
	if index == -1 {
		ui.mainPage.streamInfo.SetTitle("Stream Info")
		add("No results")
	} else {
		selStream := ui.mainPage.streams.Strims.Data[index]
		if selStream.Service == "m3u8" {
			selStream.Title = selStream.URL
		} else {
			selStream.Title = strings.ReplaceAll(selStream.Title, "\n", " ")
		}
		ui.mainPage.streamInfo.SetTitle(selStream.Channel)
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

func (ui *UI) matchStrimsListIndex(filter string) []int {
	var ixs []int
	re, err := regexp.Compile("(?i)" + filter)
	if err != nil {
		for i := range ui.mainPage.streams.Strims.Data {
			ixs = append(ixs, i)
		}
		return ixs
	}
	for i, v := range ui.mainPage.streams.Strims.Data {
		var match func(string) bool = re.MatchString
		matched := match(v.Service) || match(v.Title) || match(v.Channel)
		if matched != ui.twitchFilter.inverted {
			ixs = append(ixs, i)
		}
	}
	return ixs
}
