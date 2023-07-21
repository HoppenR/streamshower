package main

import (
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (ui *UI) refreshTwitchList() {
	oldIdx := ui.pg1.twitchList.GetCurrentItem()
	ui.filterTwitchList(ui.pg2.input.GetText())
	ui.pg1.twitchList.SetCurrentItem(oldIdx)
}

func (ui *UI) setupFilterTwitchPage() {
	ui.pg2.input.SetBackgroundColor(tcell.ColorDefault)
	ui.pg2.input.SetBorder(true)
	ui.pg2.input.SetTitle("Filter(Regex)")
	ui.pg2.input.SetFinishedFunc(func(key tcell.Key) {
		ui.pages.HidePage("Filter-Twitch")
		ui.app.SetFocus(ui.pg1.twitchList)
	})
	ui.pg2.input.SetChangedFunc(ui.filterTwitchList)
	const (
		FilterWidth  = 26
		FilterHeight = 3
	)
	ui.pg2.input.SetAcceptanceFunc(func(toCheck string, lastChar rune) bool {
		if lastChar == '!' {
			if ui.pg2.inverted {
				ui.pg2.input.SetTitle("Filter(Regex)")
				ui.pg2.inverted = false
			} else {
				ui.pg2.input.SetTitle("Filter(Regex(inverted))")
				ui.pg2.inverted = true
			}
			ui.refreshTwitchList()
			return false
		}
		return tview.InputFieldMaxLength(FilterWidth-3)(toCheck, lastChar)
	})
	//  tview.InputFieldMaxLength(FilterWidth - 3))
	ui.pg2.con.SetColumns(0, FilterWidth, 0)
	ui.pg2.con.SetRows(0, FilterHeight, 0)
	ui.pg2.con.AddItem(ui.pg2.input, 1, 1, 1, 1, 0, 0, true)
}

func (ui *UI) filterTwitchList(filter string) {
	ui.pg1.twitchList.Clear()
	ixs := ui.matchTwitchListIndex(filter, ui.pg2.inverted)
	if ixs == nil {
		ui.pg1.twitchList.AddItem("", "", 0, nil)
		return
	}
	for _, v := range ixs {
		mainstr := ui.pg1.streams.Twitch.Data[v].UserName
		secstr := fmt.Sprintf(
			" %-6d[green:-:u]%s[-:-:-]",
			ui.pg1.streams.Twitch.Data[v].ViewerCount,
			tview.Escape(ui.pg1.streams.Twitch.Data[v].GameName),
		)
		ui.pg1.twitchList.AddItem(mainstr, secstr, 0, nil)
	}
}

func (ui *UI) updateTwitchStreamInfo(ix int, pri, sec string, _ rune) {
	var index int = -1
	for i, v := range ui.pg1.streams.Twitch.Data {
		if pri == v.UserName {
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
		selStream := ui.pg1.streams.Twitch.Data[index]
		startLocal := selStream.StartedAt.Local()
		if selStream.GameName == "" {
			selStream.GameName = "[::d]None[::-]"
		}
		selStream.Title = strings.ReplaceAll(selStream.Title, "\n", " ")
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

func (ui *UI) matchTwitchListIndex(filter string, inverted bool) []int {
	var ixs []int
	re, err := regexp.Compile(filter)
	if err != nil {
		ui.pg2.input.SetBorderColor(tcell.ColorRed)
	} else {
		ui.pg2.input.SetBorderColor(tcell.ColorDefault)
	}
	for i, v := range ui.pg1.streams.Twitch.Data {
		if err != nil {
			ixs = append(ixs, i)
			continue
		}
		matches := []bool{
			re.MatchString(v.GameName),
			re.MatchString(v.Title),
			re.MatchString(v.UserName),
		}
		valid := inverted
		for _, v := range matches {
			if v {
				valid = !valid
				break
			}
		}
		if valid {
			ixs = append(ixs, i)
		}
	}
	return ixs
}
