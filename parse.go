package main

import (
	"fmt"
	"regexp"
	"slices"
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
				return nil
			} else if len(args) > possible[0].MaxArgs {
				return nil
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

// Callback for when the command line has finished
func (ui *UI) execCommandChainCallback(key tcell.Key) {
	switch key {
	case tcell.KeyEnter:
		cmdLine := ui.mainPage.commandLine.GetText()
		ui.cmdRegistry.history = append(ui.cmdRegistry.history, cmdLine)
		ui.cmdRegistry.histIndex = len(ui.cmdRegistry.history)
		err := ui.execCommandChainSilent(cmdLine)
		if err != nil {
			ui.mainPage.appStatusText.SetText(err.Error())
		}
		fallthrough
	case tcell.KeyEsc:
		ui.app.SetFocus(ui.mainPage.focusedList)
	}
}

// Execute the command chain from a mapping
func (ui *UI) execCommandChainMapping(keyStrings []string) error {
	for _, mapping := range keyStrings {
		k, err := parseMappingKey(mapping)
		if err != nil {
			return err
		}
		ui.app.QueueEvent(k)
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
				return fmt.Errorf("Trailing characters: %s", strings.Join(args, " "))
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

func (m *MainPage) applyFilterFromArg(arg string, bang bool, invertMatching bool) {
	re := regexp.MustCompile(`^\/([^\/]*)\/([dp])$`)
	matches := re.FindStringSubmatch(arg)
	if len(matches) <= 2 {
		return
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
}

func (r *CommandRegistry) matchPossibleCommands(name string) []*ExCommand {
	var possible []*ExCommand
	for _, cmd := range r.commands {
		if strings.HasPrefix(cmd.Name, name) {
			possible = append(possible, cmd)
		}
	}
	return possible
}

func matchPossibleBuiltinHelpNames(name string) []string {
	var possible []string
	for _, bh := range builtinHelps {
		for _, bhName := range bh.Names {
			if strings.HasPrefix(bhName, name) {
				possible = append(possible, bhName)
			}
		}
	}
	return possible
}

func matchPossibleBuiltinHelps(name string) []BuiltinHelp {
	var possible []BuiltinHelp
	for _, bh := range builtinHelps {
		if slices.ContainsFunc(bh.Names, func(bhName string) bool {
			return strings.HasPrefix(bhName, name)
		}) {
			possible = append(possible, bh)
		}
	}
	return possible
}

func parseCommandParts(input string) (string, []string, bool) {
	cmdLine := strings.TrimSpace(strings.TrimPrefix(input, ":"))
	name, rest := extractCmdName(cmdLine)
	bang := false
	rest = strings.TrimSpace(rest)
	if strings.HasPrefix(rest, "!") {
		bang = true
		rest = strings.TrimPrefix(rest, "!")
	}
	if strings.HasSuffix(rest, "!") {
		bang = true
		rest = strings.TrimSuffix(rest, "!")
	}
	args := strings.Fields(rest)
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

// Variation selectors seem to cause issues with tview rendering, remove them
func removeVariationSelectors(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if unicode.Is(unicode.Variation_Selector, r) {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
