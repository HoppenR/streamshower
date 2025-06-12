package main

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
)

type Mode int

const (
	ModeNormal Mode = iota
	ModeCommand
)

type MappingRegistry struct {
	mappings map[string]string
}

var defaultMappingLiterals = map[string]string{
	"<C-b>":   ":scrollinfo up<CR>",
	"<C-f>":   ":scrollinfo down<CR>",
	"<C-l>":   ":nohlsearch<CR>",
	"<C-w>":   ":focus toggle<CR>",
	"<CR>":    "lq",
	"<F1>":    ":echo Please see `:help` or `:map`!<CR>",
	"<Right>": "lq",
	"<Space>": ":open<Space>",
	"R":       ":update<CR>r",
	"U":       ":windo undo<CR>",
	"W":       ":set! winopen<CR>",
	"b":       "lc",
	"c":       ":set winopen | open chat<CR>q",
	"f":       ":global <C-z><Tab>",
	"h":       "<F1>",
	"l":       ":open embed<CR>",
	"m":       ":open mpv<CR>q",
	"o":       "<C-w>",
	"q":       ":quit<CR>",
	"r":       ":sync<CR>",
	"s":       ":open strims<CR>q",
	"t":       ":set! strims | focus twitch<CR>",
	"u":       ":undo<CR>",
	"v":       ":vglobal <C-z><Tab>",
	"w":       ":open homepage<CR>q",
	"y":       ":copyurl<Space>",
}

func NewMappingRegistry() *MappingRegistry {
	return &MappingRegistry{mappings: defaultMappingLiterals}
}

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

func parseMappingKey(key string) (*tcell.EventKey, error) {
	if key != " " {
		key = strings.TrimSpace(key)
	}
	if strings.HasPrefix(key, "<") && strings.HasSuffix(key, ">") {
		name := strings.ToLower(key[1 : len(key)-1])
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
		return nil, fmt.Errorf("unknown key: %s", key)
	}
	if len([]rune(key)) == 1 {
		return tcell.NewEventKey(tcell.KeyRune, []rune(key)[0], tcell.ModNone), nil
	}
	return nil, fmt.Errorf("invalid key format: %s", key)
}

func (r *MappingRegistry) resolveMappings(input string) ([]string, error) {
	mode := ModeNormal

	var keys []string
	for i := 0; i < len(input); {
		if input[i] == ':' {
			mode = ModeCommand
			keys = append(keys, ":")
			i++
			continue
		}

		if input[i] == '<' {
			end := strings.IndexRune(input[i:], '>')
			if end != -1 {
				key := input[i : i+end+1]
				err := validateMappingKey(key)
				if err != nil {
					return nil, err
				}
				switch mode {
				case ModeCommand:
					switch key {
					case "<CR>", "<Esc>":
						mode = ModeNormal
						fallthrough
					default:
						keys = append(keys, key)
					}
				case ModeNormal:
					rhs, ok := r.mappings[key]
					if ok {
						rKeys, err := r.resolveMappings(rhs)
						if err != nil {
							return nil, err
						}
						keys = append(keys, rKeys...)
					} else {
						keys = append(keys, key)
					}
				}
				i += end + 1
				continue
			} else {
				return nil, fmt.Errorf("unmatched < in mapping at position %d", i)
			}
		}

		switch mode {
		case ModeCommand:
			keys = append(keys, input[i:i+1])
		case ModeNormal:
			rhs, ok := r.mappings[input[i:i+1]]
			if ok {
				rKeys, err := r.resolveMappings(rhs)
				if err != nil {
					return nil, err
				}
				keys = append(keys, rKeys...)
			} else {
				keys = append(keys, input[i:i+1])
			}
		}
		i++
	}
	return keys, nil
}
