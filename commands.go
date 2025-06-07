package main

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
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

	MinArgs int
	MaxArgs int
}

type CommandRegistry struct {
	commands map[string]*Command
}

func NewCommandRegistry() *CommandRegistry {
	return &CommandRegistry{
		commands: make(map[string]*Command),
	}
}

func (r *CommandRegistry) Register(cmd *Command) {
	r.commands[cmd.Name] = cmd
}

func (r *CommandRegistry) Help() string {
	var lines []string
	for _, cmd := range r.commands {
		lines = append(lines, fmt.Sprintf(":[red]%s[-] - %s", cmd.Usage, cmd.Description))
	}
	sort.Strings(lines)
	return strings.Join(lines, "\n")
}

func (ui *UI) parseCommand(cmdLine string) {
	if cmdLine == "" {
		ui.app.SetFocus(ui.mainPage.focusedList)
		return
	}

	if strings.HasPrefix(cmdLine, ":") {
		cmdLine = strings.TrimPrefix(cmdLine, ":")
		name, args, bang := parseCommandParts(cmdLine)
		if name == "" {
			return
		}
		possible := ui.cmdRegistry.matchPossibleCommands(name)
		if len(possible) == 1 && possible[0].OnType != nil {
			err := possible[0].OnType(ui, args, bang)
			if err != nil {
				ui.mainPage.appStatusText.SetText(err.Error())
			}
		}
	} else if strings.HasPrefix(cmdLine, "/") {
		ui.mainPage.lastSearch = strings.TrimPrefix(cmdLine, "/")
	} else if strings.HasPrefix(cmdLine, "?") {
		ui.mainPage.lastSearch = strings.TrimPrefix(cmdLine, "?")
	}
}

func (ui *UI) execCommand(key tcell.Key) {
	ui.app.SetFocus(ui.mainPage.focusedList)
	cmdLine := strings.TrimSpace(ui.mainPage.commandLine.GetText())
	if key != tcell.KeyEnter {
		return
	}
	if cmdLine == "" {
		return
	}

	if strings.HasPrefix(cmdLine, ":") {
		cmdLine = strings.TrimPrefix(cmdLine, ":")
		name, args, bang := parseCommandParts(cmdLine)
		if name == "" {
			return
		}
		possible := ui.cmdRegistry.matchPossibleCommands(name)
		switch len(possible) {
		case 0:
			ui.mainPage.appStatusText.SetText(fmt.Sprintf("[red]Unknown command: %s[-]", name))
		case 1:
			if len(args) < possible[0].MinArgs {
				ui.mainPage.appStatusText.SetText(fmt.Sprintf("Argument required for command: %s", name))
				return
			} else if len(args) > possible[0].MaxArgs {
				ui.mainPage.appStatusText.SetText(fmt.Sprintf("Trailing characters: %s", strings.Join(args, "")))
				return
			}
			err := possible[0].Execute(ui, args, bang)
			if err != nil {
				ui.mainPage.appStatusText.SetText(err.Error())
			}
		default:
			var names []string
			for _, m := range possible {
				names = append(names, m.Name)
			}
			ui.mainPage.appStatusText.SetText(fmt.Sprintf("[red]Ambiguous: %s (could be %s)[-]", name, strings.Join(names, ", ")))
		}
	} else if strings.HasPrefix(cmdLine, "/") {
		ui.searchNext()
	} else if strings.HasPrefix(cmdLine, "?") {
		ui.searchPrev()
	}
}

func extractCmdName(text string) (name string, rest string) {
	for i, r := range text {
		if !unicode.IsLetter(r) {
			return text[:i], text[i:]
		}
	}
	return text, ""
}

func applyFilterFromArg(u *UI, arg string, bang bool, invertMatching bool) error {
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
	if bang {
		filters = []*FilterInput{u.twitchFilter, u.strimsFilter}
	} else {
		switch u.mainPage.focusedList {
		case u.mainPage.twitchList:
			filters = []*FilterInput{u.twitchFilter}
		case u.mainPage.strimsList:
			filters = []*FilterInput{u.strimsFilter}
		}
	}
	for _, f := range filters {
		if invertMatching {
			f.inverted = (exCmd == 'p')
		} else {
			f.inverted = (exCmd == 'd')
		}
		f.input = cmdArgument
	}

	u.refreshTwitchList()
	u.refreshStrimsList()
	return nil
}

func (r *CommandRegistry) matchPossibleCommands(name string) []*Command {
	possible := []*Command{}
	for cmdName, cmd := range r.commands {
		if strings.HasPrefix(cmdName, name) {
			possible = append(possible, cmd)
		}
	}
	return possible
}

func parseCommandParts(input string) (string, []string, bool) {
	trimmed := strings.TrimSpace(strings.TrimPrefix(input, ":"))
	name, rest := extractCmdName(trimmed)

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
