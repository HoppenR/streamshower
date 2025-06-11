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
	Complete: func(u *UI, s string, bang bool) []string {
		methods := []string{"embed", "homepage", "mpv", "strims", "chat"}
		matches := make([]string, 0, len(methods))
		for _, method := range methods {
			if strings.HasPrefix(method, s) {
				matches = append(matches, ":copyurl "+method)
			}
		}
		return matches
	},
	Execute: func(ui *UI, args []string, bang bool) error {
		switch args[0] {
		case "embed":
			return ui.copySelectedStreamToClipboard(lnkOpenEmbed)
		case "homepage":
			return ui.copySelectedStreamToClipboard(lnkOpenHomePage)
		case "mpv":
			return ui.copySelectedStreamToClipboard(lnkOpenMpv)
		case "strims":
			return ui.copySelectedStreamToClipboard(lnkOpenStrims)
		case "chat":
			return ui.copySelectedStreamToClipboard(lnkOpenChat)
		default:
			return fmt.Errorf("unsupported method")
		}
	},
}, {
	Name:        "focus",
	Description: "Focus the window for {list}",
	Usage:       "fo[cus[] {list=twitch|strims|toggle}",
	MinArgs:     1,
	MaxArgs:     1,
	Complete: func(ui *UI, s string, bang bool) []string {
		lists := []string{"twitch", "strims", "toggle"}
		entries := make([]string, 0, len(lists))
		for _, list := range lists {
			if strings.HasPrefix(list, s) {
				entries = append(entries, ":focus "+list)
			}
		}
		return entries
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
		return []string{":global//d"}
	},
	OnType: func(ui *UI, args []string, bang bool) error {
		return applyFilterFromArg(ui.mainPage, strings.Join(args, " "), bang, false)
	},
}, {
	Name:        "help",
	Description: "Show help for all commands, or those matching [subject[] if provided",
	Usage:       "h[elp[] [subject[]",
	MinArgs:     0,
	MaxArgs:     1,
	Complete: func(ui *UI, s string, bang bool) []string {
		possibleCmds := ui.cmdRegistry.matchPossibleCommands(s)
		entries := make([]string, 0, len(possibleCmds))
		for _, cmd := range possibleCmds {
			entries = append(entries, ":help :"+cmd.Name)
		}
		return entries
	},
	Execute: func(ui *UI, args []string, bang bool) error {
		ui.mainPage.streamInfo.Clear()
		ui.mainPage.streamInfo.Write([]byte("--- [orange::b]<C-f>/<C-b> to scroll up/down the window[-::-] ---\n\n"))
		switch len(args) {
		case 0:
			for _, cmd := range ui.cmdRegistry.commands {
				ui.mainPage.streamInfo.Write(fmt.Appendf(nil, ":[red]%s[-]\n  %s\n", cmd.Usage, cmd.Description))
			}
		case 1:
			query := strings.TrimPrefix(args[0], ":")
			for _, cmd := range ui.cmdRegistry.commands {
				if !strings.HasPrefix(cmd.Name, query) {
					continue
				}
				ui.mainPage.streamInfo.Write(fmt.Appendf(nil, ":[red]%s[-] - %s\n", cmd.Usage, cmd.Description))
			}
		}
		ui.mainPage.streamInfo.SetTitle("HELP")
		return nil
	},
}, {
	Name:        "map",
	Description: "map keypress [lhs[] into command [rhs[]. Use <Bar> over | for chaining commands in [rhs[]",
	Usage:       "m[ap[] [lhs rhs[]",
	MinArgs:     0,
	MaxArgs:     math.MaxInt,
	Complete: func(u *UI, s string, bang bool) []string {
		methods := maps.Keys(u.mapRegistry.mappings)
		matches := make([]string, 0, len(u.mapRegistry.mappings))
		for method := range methods {
			if strings.HasPrefix(method, s) {
				matches = append(matches, ":map "+method)
			}
		}
		return matches
	},
	Execute: func(ui *UI, args []string, bang bool) error {
		var mappings []string
		switch len(args) {
		case 0:
			for lhs, rhs := range ui.mapRegistry.mappings {
				mappings = append(mappings, fmt.Sprintf("[red]%s[-] - %s\n", lhs, rhs))
			}
		case 1:
			for lhs, rhs := range ui.mapRegistry.mappings {
				if !strings.HasPrefix(lhs, args[0]) {
					continue
				}
				mappings = append(mappings, fmt.Sprintf("[red]%s[-] - %s\n", lhs, rhs))
			}
		default:
			if err := validateMappingKey(args[0]); err != nil {
				return err
			}
			rhs := strings.Join(args[1:], " ")
			rhs = strings.ReplaceAll(rhs, "<Bar>", "|")
			ui.mapRegistry.mappings[args[0]] = rhs
			return nil
		}
		sort.Strings(mappings)
		ui.mainPage.streamInfo.Clear()
		ui.mainPage.streamInfo.Write([]byte("--- [orange::b]<C-f>/<C-b> to scroll up/down the window[-::-] ---\n"))
		ui.mainPage.streamInfo.Write([]byte("  `:normal {key}` commands are equivalent to the respective {key} in vim\n\n"))
		for _, line := range mappings {
			ui.mainPage.streamInfo.Write([]byte(line))
		}
		ui.mainPage.streamInfo.SetTitle("MAPPINGS")
		return nil
	},
}, {
	Name:        "nohlsearch",
	Description: "Stop higlighting search",
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
	Complete: func(u *UI, s string, bang bool) []string {
		methods := []string{"embed", "homepage", "mpv", "strims", "chat"}
		matches := make([]string, 0, len(methods))
		for _, method := range methods {
			if strings.HasPrefix(method, s) {
				matches = append(matches, ":open "+method)
			}
		}
		return matches
	},
	Execute: func(ui *UI, args []string, bang bool) error {
		switch args[0] {
		case "embed":
			return ui.openSelectedStream(lnkOpenEmbed)
		case "homepage":
			return ui.openSelectedStream(lnkOpenHomePage)
		case "mpv":
			return ui.openSelectedStream(lnkOpenMpv)
		case "strims":
			return ui.openSelectedStream(lnkOpenStrims)
		case "chat":
			return ui.openSelectedStream(lnkOpenChat)
		default:
			return fmt.Errorf("unsupported method")
		}
	},
}, {
	Name:        "scrollinfo",
	Description: "Scroll the stream info window by {direction=up|down}",
	Usage:       "sc[rollinfo[] {direction}",
	MinArgs:     1,
	MaxArgs:     1,
	Complete: func(u *UI, s string, bang bool) []string {
		methods := []string{"up", "down"}
		matches := make([]string, 0, len(methods))
		for _, method := range methods {
			if strings.HasPrefix(method, s) {
				matches = append(matches, ":scrollinfo "+method)
			}
		}
		return matches
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
	Name:        "resize",
	Description: "Resize current window to {size} (default: 1)",
	Usage:       "r[esize[] {size}",
	MinArgs:     1,
	MaxArgs:     1,
	Execute: func(ui *UI, args []string, bang bool) error {
		n, err := strconv.Atoi(args[0])
		if err != nil {
			return err
		}
		ui.mainPage.streamsCon.ResizeItem(ui.mainPage.focusedList, 0, n)
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
	Description: "Set [option[] or [no{option}[], ! toggles the value. Available options: strims",
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
		matches := make([]string, 0, len(options))
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
	Usage:       "un[do[]",
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
		return []string{":vglobal//d"}
	},
	OnType: func(ui *UI, args []string, bang bool) error {
		return applyFilterFromArg(ui.mainPage, strings.Join(args, " "), bang, true)
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
