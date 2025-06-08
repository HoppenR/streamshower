package main

type MappingRegistry struct {
	mappings map[string]string
}

var defaultMappings = map[string]string{
	"/":       "/",
	":":       ":",
	"<C-b>":   ":scrollinfo up<CR>",
	"<C-d>":   ":normal <C-d><CR>",
	"<C-e>":   ":normal <C-e><CR>",
	"<C-f>":   ":scrollinfo down<CR>",
	"<C-j>":   ":open embed<CR>",
	"<C-n>":   ":normal <C-n><CR>",
	"<C-p>":   ":normal <C-p><CR>",
	"<C-u>":   ":normal <C-u><CR>",
	"<C-w>":   ":focus toggle<CR>",
	"<C-y>":   ":normal <C-y><CR>",
	"<CR>":    ":open embed | quit<CR>",
	"<Down>":  ":normal <Down><CR>",
	"<Enter>": ":open embed | quit<CR>",
	"<F1>":    " Please see `:help` or `:map`!<CR>",
	"<Right>": ":open embed | quit<CR>",
	"<Space>": ":open ",
	"<Up>":    ":normal <Up><CR>",
	"?":       "?",
	"G":       ":normal G<CR>",
	"M":       ":normal M<CR>",
	"N":       "?<CR>",
	"R":       ":update | sync<CR>",
	"U":       ":clear!<CR>",
	"c":       ":open chat | quit<CR>",
	"f":       ":global <Tab>",
	"g":       ":normal g<CR>",
	"j":       ":normal j<CR>",
	"k":       ":normal k<CR>",
	"l":       ":open embed | quit<CR>",
	"m":       ":open mpv | quit<CR>",
	"n":       "/<CR>",
	"o":       ":focus toggle<CR>",
	"q":       ":quit<CR>",
	"r":       ":sync<CR>",
	"s":       ":open strims | quit<CR>",
	"t":       ":set! strims | focus twitch<CR>",
	"u":       ":clear<CR>",
	"v":       ":vglobal <Tab>",
	"w":       ":open homepage | quit<CR>",
	"y":       ":copyurl ",
	"z":       ":normal z<CR>",
}

func NewMappingRegistry() *MappingRegistry {
	return &MappingRegistry{mappings: defaultMappings}
}
