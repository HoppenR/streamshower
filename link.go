package main

import (
	"bytes"
	"errors"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"syscall"
	"text/template"

	ls "github.com/HoppenR/libstreams"
)

type OpenMethod int

type UrlTemplateSource struct {
	MethodTemplates map[string]UrlTemplates
	DefaultTemplate *UrlTemplates
}

type UrlTemplates struct {
	Host   string
	Path   string
	Query  Values
	RawURL string
}

type Values map[string]string // simpler, pseudo url.Values helper struct

const (
	lnkOpenEmbed OpenMethod = iota
	lnkOpenHomePage
	lnkOpenMpv
	lnkOpenStrims
	lnkOpenChat
)

var urlBuilders = map[OpenMethod]UrlTemplateSource{
	lnkOpenEmbed: {MethodTemplates: map[string]UrlTemplates{
		"angelthump": {Host: "player.angelthump.com", Query: Values{"channel": "{{.NameI}}"}},
		"m3u8":       {Host: "strims.gg", Path: "m3u8/{{.Name}}"},
		"twitch":     {Host: "player.twitch.tv", Query: Values{"channel": "{{.NameI}}", "parent": "strims.gg"}},
		"twitch-vod": {Host: "player.twitch.tv", Query: Values{"video": "v{{.Name}}", "parent": "strims.gg"}},
		"youtube":    {Host: "www.youtube.com", Path: "embed/{{.Name}}", Query: Values{"autoplay": "true"}},
	}},
	lnkOpenHomePage: {MethodTemplates: map[string]UrlTemplates{
		"angelthump": {Host: "angelthump.com", Path: "{{.NameI}}"},
		"m3u8":       {Host: "strims.gg", Path: "m3u8/{{.Name}}"},
		"twitch":     {Host: "www.twitch.tv", Path: "{{.NameI}}"},
		"twitch-vod": {Host: "www.twitch.tv", Path: "videos/{{.Name}}"},
		"youtube":    {Host: "www.youtube.com", Path: "watch", Query: Values{"v": "{{.Name}}"}},
	}},
	lnkOpenMpv: {MethodTemplates: map[string]UrlTemplates{
		"angelthump": {Host: "ams-haproxy.angelthump.com", Path: "hls/{{.Name}}/index.m3u8"},
		"m3u8":       {RawURL: "{{.Name}}"},
		"twitch":     {Host: "www.twitch.tv", Path: "{{.NameI}}"},
		"twitch-vod": {Host: "www.twitch.tv", Path: "videos/{{.Name}}"},
		"youtube":    {Host: "www.youtube.com", Path: "watch", Query: Values{"v": "{{.Name}}"}},
	}},
	lnkOpenStrims: {
		DefaultTemplate: &UrlTemplates{Host: "strims.gg", Path: "{{.Service}}/{{.NameI}}"},
	},
	lnkOpenChat: {
		MethodTemplates: map[string]UrlTemplates{
			"twitch": {Host: "www.twitch.tv", Path: "popout/{{.NameI}}/chat"},
		},
		DefaultTemplate: &UrlTemplates{Host: "chat.strims.gg"},
	},
}

func (ui *UI) openSelectedStream(method OpenMethod) error {
	data, err := ui.getSelectedStreamData()
	if err != nil {
		return err
	}
	url, err := streamToUrl(data, method)
	if err != nil {
		return err
	}
	program := ui.getProgram(method)
	var args []string
	if ui.mainPage.winopen {
		switch program {
		case "brave", "chromium", "firefox", "google-chrome", "opera", "vivaldi":
			args = append(args, "--new-window")
		}
	}
	args = append(args, url.String())
	cmd := exec.Command(program, args...)
	// Set the new process process group-ID to its process ID
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Pgid:    0,
		Setpgid: true,
	}
	return cmd.Start()
}

func (ui *UI) copySelectedStreamToClipboard(method OpenMethod) error {
	data, err := ui.getSelectedStreamData()
	if err != nil {
		return err
	}
	url, err := streamToUrl(data, method)
	if err != nil {
		return err
	}
	return exec.Command("wl-copy", url.String()).Run()

}

func (ui *UI) getSelectedStreamData() (ls.StreamData, error) {
	listIdx := ui.mainPage.focusedList.GetCurrentItem()
	if listIdx >= ui.mainPage.focusedList.GetItemCount() {
		return nil, errors.New("current selection out of bounds")
	}
	primaryText, _ := ui.mainPage.focusedList.GetItemText(listIdx)
	switch ui.mainPage.focusedList {
	case ui.mainPage.twitchList:
		if ui.mainPage.twitchFilter.indexMapping == nil {
			break
		}
		tviewIx := ui.mainPage.twitchList.GetCurrentItem()
		ix := ui.mainPage.twitchFilter.indexMapping[tviewIx]
		return &ui.mainPage.streams.Twitch.Data[ix], nil
	case ui.mainPage.strimsList:
		ix := slices.IndexFunc(ui.mainPage.streams.Strims.Data, func(sd ls.StrimsStreamData) bool {
			return sd.Channel == primaryText
		})
		if ix != -1 {
			return &ui.mainPage.streams.Strims.Data[ix], nil
		}
	}
	return nil, errors.New("cannot open empty result")
}

func streamToUrl(data ls.StreamData, method OpenMethod) (*url.URL, error) {
	tmplSrc, ok := urlBuilders[method]
	if !ok {
		return nil, errors.New("unsupported method")
	}
	if tmplSrc.MethodTemplates != nil {
		if template, ok := tmplSrc.MethodTemplates[data.GetService()]; ok {
			return template.apply(data)
		}
	}
	if tmplSrc.DefaultTemplate != nil {
		return tmplSrc.DefaultTemplate.apply(data)
	}
	return nil, errors.New("template source incomplete")
}

func (ut *UrlTemplates) apply(data ls.StreamData) (*url.URL, error) {
	if ut.RawURL != "" {
		raw, err := executeTemplateString(ut.RawURL, data)
		if err != nil {
			return nil, err
		}
		return url.Parse(raw)
	}

	newHost, err := executeTemplateString(ut.Host, data)
	if err != nil {
		return nil, err
	}
	newPath, err := executeTemplateString(ut.Path, data)
	if err != nil {
		return nil, err
	}
	newValues := make(url.Values, len(ut.Query))
	for key, value := range ut.Query {
		newParam, err := executeTemplateString(value, data)
		if err != nil {
			return nil, err
		}
		newValues[key] = []string{newParam}
	}
	url := &url.URL{
		Scheme:   "https",
		Host:     newHost,
		Path:     newPath,
		RawQuery: newValues.Encode(),
	}
	return url, nil
}

func executeTemplateString(templateString string, data ls.StreamData) (string, error) {
	tmpl, err := template.New("t").Parse(templateString)
	if err != nil {
		return "", err
	}
	var buffer bytes.Buffer
	err = tmpl.Execute(&buffer, map[string]string{
		"NameI":   strings.ToLower(data.GetName()),
		"Name":    data.GetName(),
		"Service": data.GetService(),
	})
	return buffer.String(), err
}

func (ui *UI) getProgram(method OpenMethod) string {
	switch method {
	case lnkOpenMpv:
		return "mpv"
	default:
		browser := os.Getenv("BROWSER")
		if browser == "" {
			switch runtime.GOOS {
			case "darwin":
				return "open"
			case "windows":
				return "explorer"
			default:
				return "xdg-open"
			}
		}
		return filepath.Base(browser)
	}
}
