package main

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/gdamore/tcell/v2"
)

func (ui *UI) onTypeCommandChain(cmdLine string) {
	if cmdLine == "" {
		ui.app.SetFocus(ui.mainPage.focusedList)
		return
	}
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
		err := ui.onTypeCommand(cmd)
		if err != nil {
			ui.mainPage.appStatusText.SetText(err.Error())
		}
	}
}

func (ui *UI) onTypeCommand(cmd string) error {
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
					return err
				}
			}
		}
		return nil
	} else if strings.HasPrefix(cmd, "/") {
		ui.mainPage.lastSearch = strings.TrimPrefix(cmd, "/")
	} else if strings.HasPrefix(cmd, "?") {
		ui.mainPage.lastSearch = strings.TrimPrefix(cmd, "?")
	}
	ui.mainPage.refreshTwitchList()
	ui.mainPage.refreshStrimsList()
	return nil
}

// Callback for when a user has manually typed in a command
func (ui *UI) execCommandChainCallback(key tcell.Key) {
	if key == tcell.KeyEnter {
		cmdLine := ui.mainPage.commandLine.GetText()
		ui.cmdRegistry.histIndex = len(ui.cmdRegistry.history)
		ui.cmdRegistry.history = append(ui.cmdRegistry.history, cmdLine)
		err := ui.execCommandChainSilent(cmdLine)
		if err != nil {
			ui.mainPage.appStatusText.SetText(err.Error())
		}
	}
	ui.app.SetFocus(ui.mainPage.focusedList)
}

// Execute the command chain from a mapping
func (ui *UI) execCommandChainMapping(cmdLine string) error {
	// Execute directly if has suffix <CR>
	if strings.HasSuffix(cmdLine, "<CR>") {
		cmdLine = strings.TrimSuffix(cmdLine, "<CR>")
		ui.mainPage.commandLine.SetChangedFunc(nil)
		defer ui.mainPage.commandLine.SetChangedFunc(ui.onTypeCommandChain)
		ui.mainPage.commandLine.SetText(cmdLine)
		return ui.execCommandChainSilent(cmdLine)
	}
	// Input partial command, accept autocomplete if has suffix <Tab>
	var complete bool
	if strings.HasSuffix(cmdLine, "<Tab>") {
		cmdLine = strings.TrimSuffix(cmdLine, "<Tab>")
		complete = true
	}
	ui.mainPage.commandLine.SetText(cmdLine)
	ui.app.SetFocus(ui.mainPage.commandLine)
	ui.mainPage.commandLine.Autocomplete()
	if complete {
		ui.app.QueueEvent(tcell.NewEventKey(tcell.KeyTab, 0, tcell.ModNone))
	}
	return nil
}

// Execute a complete chain (without trailing special characters) as is,
// without printing
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
		err := ui.execCommand(cmd)
		if err != nil {
			return err
		}
	}
	return nil
}

// Execute a command
func (ui *UI) execCommand(cmdLine string) error {
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
				return fmt.Errorf("Trailing characters: %s", strings.Join(args, ""))
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

func applyFilterFromArg(m *MainPage, arg string, bang bool, invertMatching bool) error {
	re := regexp.MustCompile(`^\/([^\/]*)\/([dp])$`)
	matches := re.FindStringSubmatch(arg)
	if len(matches) <= 2 {
		return errors.New("No matches")
	}
	cmdArgument := matches[1]
	exCmd := rune(matches[2][0])
	var filters []*FilterInput
	if m.focusedList == m.twitchList || bang {
		filters = append(filters, m.twitchFilter)
	}
	if m.focusedList == m.strimsList || bang {
		filters = append(filters, m.strimsFilter)
	}
	for _, f := range filters {
		if invertMatching {
			f.inverted = (exCmd == 'p')
		} else {
			f.inverted = (exCmd == 'd')
		}
		f.input = cmdArgument
	}
	m.refreshTwitchList()
	m.refreshStrimsList()
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
	// TODO: Parse ! more intelligently,
	//       i.e don't detect in the middle of a regex - `:g/re!gex/p`
	for _, v := range parts {
		if strings.Contains(v, "!") {
			for x := range strings.SplitSeq(v, "!") {
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

func highlightSearch(text string, search string) string {
	if search == "" {
		return text
	}
	lowerText := strings.ToLower(text)
	lowerSearch := strings.ToLower(search)
	idx := strings.Index(lowerText, lowerSearch)
	if idx == -1 {
		return text
	}
	return text[:idx] + "[red]" + text[idx:idx+len(search)] + "[-]" + text[idx+len(search):]
}
