package main

import (
	"errors"
	"fmt"
	"maps"
	"math"
	"sort"
	"strconv"
	"strings"
)

type CommandRegistry struct {
	commands  []*ExCommand
	history   []string
	histIndex int
}

type ExCommand struct {
	Name        string
	Description string
	Usage       string
	Execute     func(*UI, []string, bool) error
	OnType      func(*UI, []string, bool) error
	Complete    func(*UI, string, bool) []string
	MinArgs     int
	MaxArgs     int
}

func NewCommandRegistry() *CommandRegistry {
	return &CommandRegistry{commands: defaultCommands}
}

var defaultCommands = []*ExCommand{{
	Name:        "copyurl",
	Description: "Copy url of stream by the chosen method",
	Usage:       "c[opyurl[] {method}",
	MinArgs:     1,
	MaxArgs:     1,
	Complete: func(ui *UI, s string, bang bool) []string {
		return matchCompletion(s, ":copyurl ", []string{"chat", "embed", "homepage", "mpv", "strims"})
	},
	Execute: func(ui *UI, args []string, bang bool) error {
		switch args[0] {
		case "chat":
			return ui.copySelectedStreamToClipboard(lnkOpenChat)
		case "embed":
			return ui.copySelectedStreamToClipboard(lnkOpenEmbed)
		case "homepage":
			return ui.copySelectedStreamToClipboard(lnkOpenHomePage)
		case "mpv":
			return ui.copySelectedStreamToClipboard(lnkOpenMpv)
		case "strims":
			return ui.copySelectedStreamToClipboard(lnkOpenStrims)
		default:
			return fmt.Errorf("unsupported method")
		}
	},
}, {
	Name:        "echo",
	Description: "Echo a string to the commandline",
	Usage:       "e[cho[] {string}",
	MinArgs:     1,
	MaxArgs:     math.MaxInt,
	Execute: func(ui *UI, args []string, bang bool) error {
		ui.mainPage.commandLine.SetText(strings.Join(args, " "))
		return nil
	},
}, {
	Name:        "focus",
	Description: "Focus the window for {list}",
	Usage:       "fo[cus[] {list=twitch|strims|toggle}",
	MinArgs:     1,
	MaxArgs:     1,
	Complete: func(ui *UI, s string, bang bool) []string {
		return matchCompletion(s, ":focus ", []string{"strims", "toggle", "twitch"})
	},
	Execute: func(ui *UI, args []string, bang bool) error {
		if args[0] == "twitch" || (args[0] == "toggle" && ui.mainPage.focusedList == ui.mainPage.strimsList) {
			ui.app.SetFocus(ui.mainPage.twitchList)
			ui.mainPage.focusedList = ui.mainPage.twitchList
			ui.mainPage.refreshTwitchList()
		} else if args[0] == "strims" || (args[0] == "toggle" && ui.mainPage.focusedList == ui.mainPage.twitchList) {
			ui.enableStrimsList()
			ui.app.SetFocus(ui.mainPage.strimsList)
			ui.mainPage.focusedList = ui.mainPage.strimsList
			ui.mainPage.refreshStrimsList()
		} else {
			return fmt.Errorf("unknown list %s", args[0])
		}
		return nil
	},
}, {
	Name:        "global",
	Description: "Filter {cmd=d|p} lines matching {pattern}, ! filters all lists",
	Usage:       "g[lobal[][![]/{pattern}/{cmd}",
	MinArgs:     1,
	MaxArgs:     math.MaxInt,
	Complete: func(ui *UI, s string, bang bool) []string {
		var filter *FilterInput
		switch ui.mainPage.focusedList {
		case ui.mainPage.twitchList:
			filter = ui.mainPage.twitchFilter
		case ui.mainPage.strimsList:
			filter = ui.mainPage.strimsFilter
		}
		var ret []string
		if filter.input != "" {
			ret = append(ret, ":global/"+filter.input+"/d")
		}
		return append(ret, ":global//d")
	},
	OnType: func(ui *UI, args []string, bang bool) error {
		ui.mainPage.applyFilterFromArg(strings.Join(args, " "), bang, false)
		return nil
	},
}, {
	Name:        "help",
	Description: "Show help for all commands, or those matching [subject[] if provided",
	Usage:       "h[elp[] [subject[]",
	MinArgs:     0,
	MaxArgs:     1,
	Complete: func(ui *UI, s string, bang bool) []string {
		possibleBuiltinHelps := matchPossibleBuiltinHelpNames(s)
		entries := make([]string, 0, len(ui.cmdRegistry.commands)+len(builtinHelps))
		for _, bh := range possibleBuiltinHelps {
			entries = append(entries, ":help "+bh)
		}
		if strings.HasPrefix(s, ":") {
			cmdName := strings.TrimPrefix(s, ":")
			for _, cmd := range ui.cmdRegistry.matchPossibleCommands(cmdName) {
				entries = append(entries, ":help :"+cmd.Name)
			}
		}
		return entries
	},
	Execute: func(ui *UI, args []string, bang bool) error {
		var mappings []byte
		switch len(args) {
		case 0:
			for _, bh := range builtinHelps {
				mappings = fmt.Appendf(mappings, "([::b]builtin[::-]) [red]%s[-]\n  %s\n", strings.Join(bh.Names, "[-] or [red]"), bh.Description)
			}
			for _, cmd := range ui.cmdRegistry.commands {
				mappings = fmt.Appendf(mappings, ":[red]%s[-]\n  %s\n", cmd.Usage, cmd.Description)
			}
		case 1:
			for _, bh := range matchPossibleBuiltinHelps(args[0]) {
				mappings = fmt.Appendf(mappings, "([::b]builtin[::-]) [red]%s[-]\n  %s\n", strings.Join(bh.Names, "[-] or [red]"), bh.Description)
			}
			if strings.HasPrefix(args[0], ":") {
				cmdName := strings.TrimPrefix(args[0], ":")
				for _, cmd := range ui.cmdRegistry.matchPossibleCommands(cmdName) {
					mappings = fmt.Appendf(mappings, ":[red]%s[-]\n  %s\n", cmd.Usage, cmd.Description)
				}
			}
			if len(mappings) == 0 {
				return fmt.Errorf("no help found for %s", args[0])
			}
		}
		ui.mainPage.streamInfo.Clear()
		ui.mainPage.streamInfo.ScrollTo(0, 0)
		ui.mainPage.streamInfo.Write([]byte("--- [orange::b]<C-f>/<C-b> to scroll up/down in the info window[-::-] ---\n"))
		ui.mainPage.streamInfo.Write(mappings)
		ui.mainPage.streamInfo.SetTitle("HELP")
		return nil
	},
}, {
	Name:        "map",
	Description: "Print mappings or map keypress [lhs[] into command [rhs[]. <Bar> replaces | in mappings",
	Usage:       "m[ap[] [lhs rhs[]",
	MinArgs:     0,
	MaxArgs:     math.MaxInt,
	Complete: func(ui *UI, s string, bang bool) []string {
		matches := make([]string, 0, len(ui.mapRegistry.mappings))
		for method := range maps.Keys(ui.mapRegistry.mappings) {
			if strings.HasPrefix(method, s) {
				matches = append(matches, ":map "+method)
			}
		}
		sort.Strings(matches)
		return matches
	},
	Execute: func(ui *UI, args []string, bang bool) error {
		keys := make([]string, 0, len(ui.mapRegistry.mappings))
		switch len(args) {
		case 0:
			for lhs := range maps.Keys(ui.mapRegistry.mappings) {
				keys = append(keys, lhs)
			}
		case 1:
			for lhs := range maps.Keys(ui.mapRegistry.mappings) {
				if !strings.HasPrefix(lhs, args[0]) {
					continue
				}
				keys = append(keys, lhs)
			}
			if len(keys) == 0 {
				return fmt.Errorf("no mapping found for %s", args[0])
			}
		default:
			if err := validateMappingKey(args[0]); err != nil {
				return err
			}
			if args[0] == ":" {
				return errors.New("unsupported mapping using ':'")
			}
			rhs := strings.Join(args[1:], " ")
			rhs = strings.ReplaceAll(rhs, "<Bar>", "|")
			ui.mapRegistry.mappings[args[0]] = rhs
			return nil
		}
		sort.Strings(keys)
		var mappings []byte
		for _, lhs := range keys {
			rhs := ui.mapRegistry.mappings[lhs]
			mappings = fmt.Appendf(mappings, "[red]%-7s[-] %s\n", lhs, rhs)
		}
		ui.mainPage.streamInfo.Clear()
		ui.mainPage.streamInfo.ScrollTo(0, 0)
		ui.mainPage.streamInfo.Write([]byte("--- [orange::b]<C-f>/<C-b> to scroll up/down in the info window[-::-] ---\n"))
		ui.mainPage.streamInfo.Write(mappings)
		ui.mainPage.streamInfo.SetTitle("MAPPINGS")
		return nil
	},
}, {
	Name:        "nohlsearch",
	Description: "Stop highlighting search",
	Usage:       "n[ohlsearch[]",
	MinArgs:     0,
	MaxArgs:     0,
	Execute: func(ui *UI, s []string, b bool) error {
		ui.mainPage.lastSearch = ""
		ui.mainPage.refreshTwitchList()
		ui.mainPage.refreshStrimsList()
		return nil
	},
}, {
	Name:        "open",
	Description: "Open stream with the chosen method",
	Usage:       "o[pen[] {method}",
	MinArgs:     1,
	MaxArgs:     1,
	Complete: func(ui *UI, s string, bang bool) []string {
		return matchCompletion(s, ":open ", []string{"chat", "embed", "homepage", "mpv", "strims"})
	},
	Execute: func(ui *UI, args []string, bang bool) error {
		switch args[0] {
		case "chat":
			return ui.openSelectedStream(lnkOpenChat)
		case "embed":
			return ui.openSelectedStream(lnkOpenEmbed)
		case "homepage":
			return ui.openSelectedStream(lnkOpenHomePage)
		case "mpv":
			return ui.openSelectedStream(lnkOpenMpv)
		case "strims":
			return ui.openSelectedStream(lnkOpenStrims)
		default:
			return fmt.Errorf("unsupported method")
		}
	},
}, {
	Name:        "resize",
	Description: "Resize current window to {size} (startup value: 1)",
	Usage:       "r[esize[] {size}",
	MinArgs:     1,
	MaxArgs:     1,
	Execute: func(ui *UI, args []string, bang bool) error {
		n, err := strconv.Atoi(args[0])
		if err != nil {
			return err
		}
		if n <= 0 {
			return fmt.Errorf("cannot resize to zero or negative size %d", n)
		}
		ui.mainPage.streamsCon.ResizeItem(ui.mainPage.focusedList, 0, n)
		return nil
	},
}, {
	Name:        "scrollinfo",
	Description: "Scroll the stream info window by {direction=up|down}",
	Usage:       "sc[rollinfo[] {direction}",
	MinArgs:     1,
	MaxArgs:     1,
	Complete: func(ui *UI, s string, bang bool) []string {
		return matchCompletion(s, ":scrollinfo ", []string{"down", "up"})
	},
	Execute: func(ui *UI, args []string, bang bool) error {
		switch args[0] {
		case "down":
			ui.scrollInfo(1)
		case "up":
			ui.scrollInfo(-1)
		default:
			return fmt.Errorf("unknown direction: %s", args[0])
		}
		return nil
	},
}, {
	Name:        "sync",
	Description: "Syncronize all streams on the client side",
	Usage:       "sy[nc[]",
	MinArgs:     0,
	MaxArgs:     0,
	Execute: func(ui *UI, args []string, bang bool) error {
		select {
		case ui.updateStreamsCh <- struct{}{}:
		default:
			return errors.New("[red]Skipped fetching streams, try again later...[-]")
		}
		return nil
	},
}, {
	Name:        "set",
	Description: "Set [option[] or [no{option}[], ! toggles the value. see `:h option-list`",
	Usage:       "se[t[][![] [option[]",
	MinArgs:     1,
	MaxArgs:     1,
	Complete: func(ui *UI, s string, bang bool) []string {
		options := []string{"strims", "winopen"}
		var cmdPfx strings.Builder
		cmdPfx.WriteString(":set")
		if bang {
			cmdPfx.WriteString("!")
		}
		cmdPfx.WriteString(" ")
		if strings.HasPrefix(s, "no") {
			cmdPfx.WriteString("no")
			s = strings.TrimPrefix(s, "no")
		}
		var matches []string
		for _, opt := range options {
			if strings.HasPrefix(opt, s) {
				matches = append(matches, cmdPfx.String()+opt)
			}
		}
		return matches
	},
	Execute: func(ui *UI, args []string, bang bool) error {
		switch len(args) {
		case 1:
			var prefixno bool
			arg := args[0]
			if strings.HasPrefix(arg, "no") {
				prefixno = true
				arg = strings.TrimPrefix(arg, "no")
			}
			switch arg {
			case "strims":
				if bang {
					ui.toggleStrimsList()
				} else if prefixno {
					ui.disableStrimsList()
				} else {
					ui.enableStrimsList()
				}
				ui.mainPage.refreshTwitchList()
				ui.mainPage.refreshStrimsList()
			case "winopen":
				if bang {
					ui.mainPage.winopen = !ui.mainPage.winopen
				} else if prefixno {
					ui.mainPage.winopen = false
				} else {
					ui.mainPage.winopen = true
				}
			default:
				return fmt.Errorf("unknown option %s", arg)
			}
		}
		return nil
	},
}, {
	Name:        "quit",
	Description: "Quit the app",
	Usage:       "q[uit[]",
	MinArgs:     0,
	MaxArgs:     0,
	Execute: func(ui *UI, args []string, bang bool) error {
		ui.app.Stop()
		return nil
	},
}, {
	Name:        "undo",
	Description: "Undo filters",
	Usage:       "und[o[]",
	MinArgs:     0,
	MaxArgs:     0,
	Execute: func(ui *UI, args []string, bang bool) error {
		if ui.mainPage.focusedList == ui.mainPage.twitchList {
			ui.mainPage.twitchFilter.inverted = false
			ui.mainPage.twitchFilter.input = ""
			ui.mainPage.refreshTwitchList()
		} else if ui.mainPage.focusedList == ui.mainPage.strimsList {
			ui.mainPage.strimsFilter.inverted = false
			ui.mainPage.strimsFilter.input = ""
			ui.mainPage.refreshStrimsList()
		}
		return nil
	},
}, {
	Name:        "unmap",
	Description: "Unmap the mapping tied to the keypress {lhs}",
	Usage:       "unm[ap[] {lhs}",
	MinArgs:     1,
	MaxArgs:     1,
	Complete: func(ui *UI, s string, bang bool) []string {
		matches := make([]string, 0, len(ui.mapRegistry.mappings))
		for method := range maps.Keys(ui.mapRegistry.mappings) {
			if strings.HasPrefix(method, s) {
				matches = append(matches, ":unmap "+method)
			}
		}
		sort.Strings(matches)
		return matches
	},
	Execute: func(ui *UI, args []string, bang bool) error {
		if _, ok := ui.mapRegistry.mappings[args[0]]; ok {
			delete(ui.mapRegistry.mappings, args[0])
			return nil
		}
		return fmt.Errorf("no mapping found for %s", args[0])
	},
}, {
	Name:        "update",
	Description: "Update all streams on the connected server",
	Usage:       "up[date[]",
	MinArgs:     0,
	MaxArgs:     0,
	Execute: func(ui *UI, args []string, bang bool) error {
		select {
		case ui.forceRemoteUpdateCh <- struct{}{}:
		default:
			return errors.New("[red]Skipped remote update, try again later...[-]")
		}
		return nil
	},
}, {
	Name:        "vglobal",
	Description: "Filter {cmd=d|p} lines NOT matching {pattern}, ! filters all lists",
	Usage:       "v[global[][![]/{pattern}/{cmd}",
	MinArgs:     1,
	MaxArgs:     math.MaxInt,
	Complete: func(ui *UI, s string, bang bool) []string {
		var filter *FilterInput
		switch ui.mainPage.focusedList {
		case ui.mainPage.twitchList:
			filter = ui.mainPage.twitchFilter
		case ui.mainPage.strimsList:
			filter = ui.mainPage.strimsFilter
		}
		var ret []string
		if filter.input != "" {
			ret = append(ret, ":vglobal/"+filter.input+"/d")
		}
		return append(ret, ":vglobal//d")
	},
	OnType: func(ui *UI, args []string, bang bool) error {
		ui.mainPage.applyFilterFromArg(strings.Join(args, " "), bang, true)
		return nil
	},
}, {
	Name:        "windo",
	Description: "execute {command} once for each list",
	Usage:       "w[indo[] {cmd}",
	MinArgs:     1,
	MaxArgs:     1,
	Execute: func(ui *UI, args []string, bang bool) error {
		ui.mainPage.focusedList = ui.mainPage.twitchList
		err := ui.execCommand(":" + strings.Join(args, ""))
		if err != nil {
			return err
		}
		ui.mainPage.focusedList = ui.mainPage.strimsList
		err = ui.execCommand(":" + strings.Join(args, ""))
		if err != nil {
			return nil
		}
		return nil
	},
}}

func matchCompletion(s string, prefix string, options []string) []string {
	var matches []string
	for _, method := range options {
		if strings.HasPrefix(method, s) {
			matches = append(matches, prefix+method)
		}
	}
	return matches
}
