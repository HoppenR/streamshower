package main

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
)

func validateMappingKey(input string) error {
	_, err := parseMappingKey(input)
	return err
}

func encodeMappingKey(input *tcell.EventKey) string {
	switch input.Key() {
	case tcell.KeyRune:
		switch input.Rune() {
		case ' ':
			return "<Space>"
		default:
			return string(input.Rune())
		}
	case tcell.KeyBackspace:
		return "<BS>"
	case tcell.KeyEnter:
		return "<CR>"
	case tcell.KeyDown:
		return "<Down>"
	case tcell.KeyEsc:
		return "<Esc>"
	case tcell.KeyTab:
		return "<Tab>"
	case tcell.KeyUp:
		return "<Up>"
	case tcell.KeyRight:
		return "<Right>"
	}
	if input.Key() >= tcell.KeyCtrlA && input.Key() <= tcell.KeyCtrlZ {
		c := 'a' + rune(input.Key()-tcell.KeyCtrlA)
		return fmt.Sprintf("<C-%c>", c)
	}
	if input.Key() >= tcell.KeyF1 && input.Key() <= tcell.KeyF9 {
		c := '1' + rune(input.Key()-tcell.KeyF1)
		return fmt.Sprintf("<F%c>", c)
	}
	return fmt.Sprintf("<Key-%d>", input.Key())
}

func parseMappingKey(input string) (*tcell.EventKey, error) {
	input = strings.TrimSpace(input)
	if strings.HasPrefix(input, "<") && strings.HasSuffix(input, ">") {
		name := strings.ToLower(input[1 : len(input)-1])
		switch name {
		case "bs", "backspace":
			return tcell.NewEventKey(tcell.KeyBackspace, 0, tcell.ModNone), nil
		case "cr", "enter":
			return tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone), nil
		case "down":
			return tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone), nil
		case "esc":
			return tcell.NewEventKey(tcell.KeyEsc, 0, tcell.ModNone), nil
		case "space":
			return tcell.NewEventKey(tcell.KeyRune, ' ', tcell.ModNone), nil
		case "tab":
			return tcell.NewEventKey(tcell.KeyTab, 0, tcell.ModNone), nil
		case "up":
			return tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone), nil
		case "right":
			return tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone), nil
		}
		// TODO: Handle these with regex instead
		if strings.HasPrefix(name, "c-") && len(name) == 3 {
			c := name[2]
			if c >= 'a' && c <= 'z' {
				ctrlKey := tcell.KeyCtrlA + tcell.Key(c-'a')
				return tcell.NewEventKey(ctrlKey, 0, tcell.ModNone), nil
			}
		}
		if strings.HasPrefix(name, "f") && len(name) == 2 {
			c := name[1]
			if c >= '1' && c <= '9' {
				ctrlKey := tcell.KeyF1 + tcell.Key(c-'1')
				return tcell.NewEventKey(ctrlKey, 0, tcell.ModNone), nil
			}
		}
		return nil, fmt.Errorf("unknown key: %s", input)
	}
	if len([]rune(input)) == 1 {
		return tcell.NewEventKey(tcell.KeyRune, []rune(input)[0], tcell.ModNone), nil
	}
	return nil, fmt.Errorf("invalid key format: %s", input)
}
