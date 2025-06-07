package main

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"strings"
	"unicode"

	"github.com/gdamore/tcell/v2"
)

type Command struct {
	Name        string
	Description string
	Usage       string
	Execute     func(*UI, []string, bool) error
	OnType      func(*UI, []string, bool) error
	Complete    func(*UI, string) []string
	MinArgs     int
	MaxArgs     int
}

type CommandRegistry struct {
	commands []*Command
}

var defaultCommands = []*Command{{
	Name:        "clear",
	Description: "Clear filter, ! clears filter for all lists",
	Usage:       "c[lear[][![]",
	MinArgs:     0,
	MaxArgs:     0,
	Execute: func(ui *UI, args []string, bang bool) error {
		var filters []*FilterInput
		if ui.mainPage.focusedList == ui.mainPage.twitchList || bang {
			filters = append(filters, ui.twitchFilter)
		}
		if ui.mainPage.focusedList == ui.mainPage.strimsList || bang {
			filters = append(filters, ui.strimsFilter)
		}
		for _, f := range filters {
			f.inverted = false
			f.input = ""
		}
		ui.refreshTwitchList()
		ui.refreshStrimsList()
		return nil
	},
}, {
	Name:        "focus",
	Description: "Focus the window for {list}",
	Usage:       "f[ocus[] {list=twitch|strims|toggle}",
	MinArgs:     0,
	MaxArgs:     1,
	Complete: func(ui *UI, s string) []string {
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
			ui.refreshTwitchList()
		} else if args[0] == "strims" || (args[0] == "toggle" && ui.mainPage.focusedList == ui.mainPage.twitchList) {
			ui.enableStrimsList()
			ui.app.SetFocus(ui.mainPage.strimsList)
			ui.mainPage.focusedList = ui.mainPage.strimsList
			ui.refreshStrimsList()
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
	OnType: func(ui *UI, args []string, bang bool) error {
		return applyFilterFromArg(ui, strings.Join(args, " "), bang, false)
	},
}, {
	Name:        "help",
	Description: "Show help for all commands, or those matching [subject[] if provided",
	Usage:       "h[elp[] [subject[]",
	MinArgs:     0,
	MaxArgs:     1,
	Complete: func(ui *UI, s string) []string {
		possibleCmds := ui.cmdRegistry.matchPossibleCommands(s)
		entries := make([]string, 0, len(possibleCmds))
		for _, cmd := range possibleCmds {
			entries = append(entries, ":help "+cmd.Name)
		}
		return entries
	},
	Execute: func(ui *UI, args []string, bang bool) error {
		var help string
		switch len(args) {
		case 0:
			help = ui.cmdRegistry.Help()
		case 1:
			query := strings.TrimPrefix(args[0], ":")
			for _, cmd := range ui.cmdRegistry.commands {
				if !strings.HasPrefix(cmd.Name, query) {
					continue
				}
				help += fmt.Sprintf(":[red]%s[-] - %s", cmd.Usage, cmd.Description)
			}
		}
		ui.mainPage.streamInfo.SetText(help)
		ui.mainPage.streamInfo.SetTitle("HELP")
		ui.mainPage.con.ResizeItem(ui.mainPage.streamsCon, 0, 0)
		ui.app.SetFocus(ui.mainPage.streamInfo)
		return nil
	},
}, {
	Name:        "open",
	Description: "Open stream with the chosen method",
	Usage:       "o[pen[] {method}",
	MinArgs:     1,
	MaxArgs:     1,
	Complete: func(u *UI, s string) []string {
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
	Name:        "sync",
	Description: "Syncronize streams: client side",
	Usage:       "s[ync[]",
	MinArgs:     0,
	MaxArgs:     0,
	Execute: func(ui *UI, args []string, bang bool) error {
		select {
		case ui.updateStreamsCh <- struct{}{}:
		default:
			ui.mainPage.appStatusText.SetText("[red]Skipped fetching streams, try again later...[-]")
		}
		return nil
	},
}, {
	Name:        "set",
	Description: "Set [option[] or [no{option}[], no argument shows all options, ! toggles the value",
	Usage:       "s[et[] [option[![][]",
	MinArgs:     0,
	MaxArgs:     1,
	Complete: func(ui *UI, s string) []string {
		options := []string{"strims"}
		cmdPfx := ":set "
		if strings.HasPrefix(s, "no") {
			cmdPfx += "no"
			s = strings.TrimPrefix(s, "no")
		}
		matches := make([]string, 0, len(options))
		for _, opt := range options {
			if strings.HasPrefix(opt, s) {
				matches = append(matches, cmdPfx+opt)
			}
		}
		return matches
	},
	Execute: func(ui *UI, args []string, bang bool) error {
		switch len(args) {
		case 0:
			ui.mainPage.appStatusText.Write([]byte("[red]strims[-] - whether to keep the strims window open"))
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
				// ui.app.SetFocus(ui.mainPage.twitchList)
				ui.refreshTwitchList()
				ui.refreshStrimsList()
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
	Name:        "update",
	Description: "Update streams: server side",
	Usage:       "u[pdate[][![]",
	MinArgs:     0,
	MaxArgs:     0,
	Execute: func(ui *UI, args []string, bang bool) error {
		select {
		case ui.forceRemoteUpdateCh <- struct{}{}:
		default:
			ui.mainPage.appStatusText.SetText("[red]Skipped remote update, try again later...[-]")
		}
		return nil
	},
}, {
	Name:        "vglobal",
	Description: "Filter {cmd=d|p} lines NOT matching {pattern}, ! filters all lists",
	Usage:       "v[global[][![]/{pattern}/{cmd}",
	MinArgs:     1,
	MaxArgs:     math.MaxInt,
	OnType: func(ui *UI, args []string, bang bool) error {
		return applyFilterFromArg(ui, strings.Join(args, " "), bang, true)
	},
}}

func NewCommandRegistry() *CommandRegistry {
	return &CommandRegistry{commands: defaultCommands}
}

func (r *CommandRegistry) Help() string {
	var lines []string
	for _, cmd := range r.commands {
		lines = append(lines, fmt.Sprintf(":[red]%s[-] - %s", cmd.Usage, cmd.Description))
	}
	return strings.Join(lines, "\n")
}

func (ui *UI) parseCommandChain(cmdLine string) {
	commands := strings.Split(cmdLine, "|")
	for i, cmd := range commands {
		cmd = strings.TrimSpace(cmd)
		if cmd == "" {
			continue
		}
		// NOTE: bar implies command
		if i > 0 {
			cmd = ":" + cmd
		}
		_ = ui.parseCommand(cmd)
	}
}

func (ui *UI) parseCommand(cmd string) error {
	if cmd == "" {
		ui.app.SetFocus(ui.mainPage.focusedList)
		return nil
	}
	if strings.HasPrefix(cmd, ":") {
		cmd = strings.TrimPrefix(cmd, ":")
		namepart, args, bang := parseCommandParts(cmd)
		if namepart == "" {
			return nil
		}
		possible := ui.cmdRegistry.matchPossibleCommands(namepart)
		switch len(possible) {
		case 1:
			if len(args) < possible[0].MinArgs {
				return fmt.Errorf("Argument required for command: %s", possible[0].Name)
			} else if len(args) > possible[0].MaxArgs {
				return fmt.Errorf("Trailing characters: %s", strings.Join(args, ""))
			}
			if possible[0].OnType != nil {
				err := possible[0].OnType(ui, args, bang)
				if err != nil {
					ui.mainPage.appStatusText.SetText(err.Error())
				}
			}
		}
	} else if strings.HasPrefix(cmd, "/") {
		ui.mainPage.lastSearch = strings.TrimPrefix(cmd, "/")
	} else if strings.HasPrefix(cmd, "?") {
		ui.mainPage.lastSearch = strings.TrimPrefix(cmd, "?")
	}
	return nil
}

func (ui *UI) execCommandChainCallback(key tcell.Key) {
	if key == tcell.KeyEnter {
		cmdLine := ui.mainPage.commandLine.GetText()
		err := ui.execCommandChainSilent(cmdLine)
		if err != nil {
			ui.mainPage.appStatusText.SetText(err.Error())
		}
	}
	ui.app.SetFocus(ui.mainPage.focusedList)
}

func (ui *UI) execCommandChain(cmdLine string) error {
	ui.mainPage.commandLine.SetText(cmdLine)
	return ui.execCommandChainSilent(cmdLine)
}

func (ui *UI) execCommandChainSilent(cmdLine string) error {
	commands := strings.Split(cmdLine, "|")
	for i, cmd := range commands {
		cmd = strings.TrimSpace(cmd)
		if cmd == "" {
			continue
		}
		// NOTE: bar implies command
		if i > 0 {
			cmd = ":" + cmd
		}
		err := ui.execCommandSilent(cmd)
		if err != nil {
			return err
		}
	}
	return nil
}

func (ui *UI) execCommand(cmdLine string) error {
	ui.mainPage.commandLine.SetText(cmdLine)
	return ui.execCommandSilent(cmdLine)
}

func (ui *UI) execCommandSilent(cmdLine string) error {
	cmdLine = strings.TrimSpace(cmdLine)
	if cmdLine == "" {
		return nil
	}
	if strings.HasPrefix(cmdLine, ":") {
		cmdLine = strings.TrimPrefix(cmdLine, ":")
		namepart, args, bang := parseCommandParts(cmdLine)
		if namepart == "" {
			return nil
		}
		possible := ui.cmdRegistry.matchPossibleCommands(namepart)
		switch len(possible) {
		case 0:
			return fmt.Errorf("[red]Unknown command: %s[-]", namepart)
		case 1:
			if len(args) < possible[0].MinArgs {
				return fmt.Errorf("Argument required for command: %s", possible[0].Name)
			} else if len(args) > possible[0].MaxArgs {
				return fmt.Errorf(fmt.Sprintf("Trailing characters: %s", strings.Join(args, "")))
			}
			if possible[0].Execute != nil {
				err := possible[0].Execute(ui, args, bang)
				if err != nil {
					return err
				}
			}
		default:
			var names []string
			for _, m := range possible {
				names = append(names, m.Name)
			}
			return fmt.Errorf("[red]Ambiguous: %s (could be %s)[-]", namepart, strings.Join(names, ", "))
		}
	} else if strings.HasPrefix(cmdLine, "/") {
		ui.searchNext()
	} else if strings.HasPrefix(cmdLine, "?") {
		ui.searchPrev()
	}
	return nil
}

func extractCmdName(text string) (name string, rest string) {
	for i, r := range text {
		if !unicode.IsLetter(r) {
			return text[:i], text[i:]
		}
	}
	return text, ""
}

func applyFilterFromArg(ui *UI, arg string, bang bool, invertMatching bool) error {
	re, err := regexp.Compile(`^\/([^\/]*)\/([dp])$`)
	if err != nil {
		return err
	}
	matches := re.FindStringSubmatch(arg)
	if len(matches) <= 2 {
		return errors.New("No matches")
	}
	cmdArgument := matches[1]
	exCmd := rune(matches[2][0])
	var filters []*FilterInput
	if ui.mainPage.focusedList == ui.mainPage.twitchList || bang {
		filters = append(filters, ui.twitchFilter)
	}
	if ui.mainPage.focusedList == ui.mainPage.strimsList || bang {
		filters = append(filters, ui.strimsFilter)
	}
	for _, f := range filters {
		if invertMatching {
			f.inverted = (exCmd == 'p')
		} else {
			f.inverted = (exCmd == 'd')
		}
		f.input = cmdArgument
	}
	ui.refreshTwitchList()
	ui.refreshStrimsList()
	return nil
}

func (r *CommandRegistry) matchPossibleCommands(name string) []*Command {
	possible := []*Command{}
	for _, cmd := range r.commands {
		if strings.HasPrefix(cmd.Name, name) {
			possible = append(possible, cmd)
		}
	}
	return possible
}

func parseCommandParts(input string) (string, []string, bool) {
	cmdLine := strings.TrimSpace(strings.TrimPrefix(input, ":"))
	name, rest := extractCmdName(cmdLine)
	bang := false
	rest = strings.TrimSpace(rest)
	parts := strings.Fields(rest)
	var args []string
	for _, v := range parts {
		if strings.Contains(v, "!") {
			for _, x := range strings.Split(v, "!") {
				if len(x) > 0 {
					args = append(args, x)
				}
			}
			bang = true
		} else if len(v) != 0 {
			args = append(args, v)
		}
	}
	return name, args, bang
}

func (ui *UI) searchNext() {
	list := ui.mainPage.focusedList
	count := list.GetItemCount()
	if count == 0 || ui.mainPage.lastSearch == "" {
		return
	}
	current := list.GetCurrentItem()
	ui.mainPage.commandLine.SetText("/" + ui.mainPage.lastSearch)
	for i := 1; i <= count; i++ {
		index := (current + i) % count
		mainText, _ := list.GetItemText(index)
		if strings.Contains(strings.ToLower(mainText), strings.ToLower(ui.mainPage.lastSearch)) {
			list.SetCurrentItem(index)
			return
		}
	}
	ui.mainPage.appStatusText.SetText(fmt.Sprintf("[yellow]No match for %q[-]", ui.mainPage.lastSearch))
}

func (ui *UI) searchPrev() {
	list := ui.mainPage.focusedList
	count := list.GetItemCount()
	if count == 0 || ui.mainPage.lastSearch == "" {
		return
	}
	current := list.GetCurrentItem()
	ui.mainPage.commandLine.SetText("?" + ui.mainPage.lastSearch)
	for i := 1; i <= count; i++ {
		index := (current - i + count) % count
		mainText, _ := list.GetItemText(index)
		if strings.Contains(strings.ToLower(mainText), strings.ToLower(ui.mainPage.lastSearch)) {
			list.SetCurrentItem(index)
			return
		}
	}
	ui.mainPage.appStatusText.SetText(fmt.Sprintf("[yellow]No match for %q[-]", ui.mainPage.lastSearch))
}
