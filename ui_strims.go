package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (ui *UI) refreshStrimsList() {
	oldIdx := ui.pg1.strimsList.GetCurrentItem()
	ui.filterStrimsList(ui.pg3.input.GetText())
	ui.pg1.strimsList.SetCurrentItem(oldIdx)
}

func (ui *UI) setupFilterStrimsPage() {
	ui.pg3.input.SetBackgroundColor(tcell.ColorDefault)
	ui.pg3.input.SetBorder(true)
	ui.pg3.input.SetTitle("Filter(Numeric)")
	ui.pg3.input.SetFinishedFunc(func(key tcell.Key) {
		ui.pages.HidePage("Filter-Strims")
		ui.app.SetFocus(ui.pg1.strimsList)
	})
	ui.pg3.input.SetChangedFunc(ui.filterStrimsList)
	const (
		FilterWidth  = 26
		FilterHeight = 3
	)
	ui.pg3.input.SetAcceptanceFunc(tview.InputFieldInteger)
	ui.pg3.con.SetColumns(0, FilterWidth, 0)
	ui.pg3.con.SetRows(0, FilterHeight, 0)
	ui.pg3.con.AddItem(ui.pg3.input, 1, 1, 1, 1, 0, 0, true)
}

func (ui *UI) filterStrimsList(filter string) {
	ui.pg1.strimsList.Clear()
	threshold, err := strconv.Atoi(filter)
	if err != nil {
		ui.pg3.input.SetBorderColor(tcell.ColorRed)
	} else {
		ui.pg3.input.SetBorderColor(tcell.ColorDefault)
	}
	var ixs []int
	for _, v := range ui.pg1.streams.Strims.Data {
		if v.Rustlers >= threshold {
			ixs = append(ixs, v.Rustlers)
		}
	}
	if ixs == nil {
		ui.pg1.strimsList.AddItem("", "", 0, nil)
		return
	}
	for v := range ixs {
		mainstr := ui.pg1.streams.Strims.Data[v].Channel
		color := "green"
		if ui.pg1.streams.Strims.Data[v].Nsfw {
			color = "red"
		}
		secstr := fmt.Sprintf(
			" %-6d[%s:-:u]%s[-:-:-]",
			ui.pg1.streams.Strims.Data[v].Rustlers,
			color,
			tview.Escape(ui.pg1.streams.Strims.Data[v].Title),
		)
		ui.pg1.strimsList.AddItem(mainstr, secstr, 0, nil)
	}
}

func (ui *UI) updateStrimsStreamInfo(ix int, pri, sec string, _ rune) {
	var index int = -1
	for i, v := range ui.pg1.streams.Strims.Data {
		if pri == v.Channel {
			index = i
			break
		}
	}
	add := func(c string) {
		ui.pg1.streamInfo.Write([]byte(c))
	}
	ui.pg1.streamInfo.Clear()
	if index == -1 {
		add("No results")
	} else {
		selStream := ui.pg1.streams.Strims.Data[index]
		selStream.Title = strings.ReplaceAll(selStream.Title, "\n", " ")
		add(fmt.Sprintf("[red]Title[-]: %s\n", tview.Escape(selStream.Title)))
		add(fmt.Sprintf("[red]Rustlers[-]: %d [lightgray](%d afk)[-]\n", selStream.Rustlers, selStream.AfkRustlers))
		add(fmt.Sprintf("[red]Service[-]: %s\n", selStream.Service))
		add(fmt.Sprintf("[red]Viewers[-]: %v\n", selStream.Viewers))
		add(fmt.Sprintf("[red]Live[-]: %v\n", selStream.Live))
		add(fmt.Sprintf("[red]AFK[-]: %v\n", selStream.Afk))
	}
}
