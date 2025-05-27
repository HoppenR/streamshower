package main

import (
	"bytes"
	"errors"
	"net/url"
	"os"
	"os/exec"
	"slices"
	"strings"
	"syscall"
	"text/template"

	sc "github.com/HoppenR/streamchecker"
)

type OpenMethod int

type UrlTemplateSource struct {
	MethodTemplates map[string]UrlTemplates
	DefaultTemplate *UrlTemplates
}

type UrlTemplates struct {
	Host   string
	Path   string
	Values V
}

type V map[string]string // simpler, pseudo url.Values helper struct

const (
	lnkOpenEmbed OpenMethod = iota
	lnkOpenHomePage
	lnkOpenMpv
	lnkOpenStrims
	lnkOpenChat
)

var urlBuilders = map[OpenMethod]UrlTemplateSource{
	lnkOpenEmbed: {MethodTemplates: map[string]UrlTemplates{
		"angelthump": {Host: "player.angelthump.com", Values: V{"channel": "{{.Name}}"}},
		"m3u8":       {Host: "strims.gg", Path: "m3u8/{{.Name}}"},
		"twitch":     {Host: "player.twitch.tv", Values: V{"channel": "{{.Name}}", "parent": "strims.gg"}},
		"twitch-vod": {Host: "player.twitch.tv", Values: V{"video": "v{{.Name}}", "parent": "strims.gg"}},
		"youtube":    {Host: "www.youtube.com", Path: "embed/{{.Name}}", Values: V{"autoplay": "true"}},
	}},
	lnkOpenHomePage: {MethodTemplates: map[string]UrlTemplates{
		"angelthump": {Host: "angelthump.com", Path: "{{.Name}}"},
		"m3u8":       {Host: "strims.gg", Path: "m3u8/{{.Name}}"},
		"twitch":     {Host: "www.twitch.tv", Path: "{{.Name}}"},
		"twitch-vod": {Host: "www.twitch.tv", Path: "videos/{{.Name}}"},
		"youtube":    {Host: "www.youtube.com", Path: "watch", Values: V{"v": "{{.Name}}"}},
	}},
	lnkOpenMpv: {MethodTemplates: map[string]UrlTemplates{
		"angelthump": {Host: "ams-haproxy.angelthump.com", Path: "hls/{{.Name}}/index.m3u8"},
		"m3u8":       {Host: "{{.Name}}"},
		"twitch":     {Host: "www.twitch.tv", Path: "{{.Name}}"},
		"twitch-vod": {Host: "www.twitch.tv", Path: "videos/{{.Name}}"},
		"youtube":    {Host: "www.youtube.com", Path: "watch", Values: V{"v": "{{.Name}}"}},
	}},
	lnkOpenStrims: {
		DefaultTemplate: &UrlTemplates{Host: "strims.gg", Path: "{{.Service}}/{{.Name}}"},
	},
	lnkOpenChat: {
		MethodTemplates: map[string]UrlTemplates{
			"twitch": {Host: "www.twitch.tv", Path: "popout/{{.Name}}/chat"},
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
	program, err := ui.getProgram(method)
	if err != nil {
		return err
	}
	executable, err := exec.LookPath(program)
	if err != nil {
		return err
	}
	cmd := exec.Command(executable, url.String())
	// Set the new process process group-ID to its process ID
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Pgid:    0,
		Setpgid: true,
	}
	return cmd.Start()
}

func (ui *UI) getSelectedStreamData() (sc.StreamData, error) {
	listIdx := ui.pg1.focusedList.GetCurrentItem()
	primaryText, _ := ui.pg1.focusedList.GetItemText(listIdx)
	switch ui.pg1.focusedList.GetTitle() {
	case "Twitch":
		ix := slices.IndexFunc(ui.pg1.streams.Twitch.Data, func(sd sc.TwitchStreamData) bool {
			return sd.UserName == primaryText
		})
		if ix != -1 {
			return &ui.pg1.streams.Twitch.Data[ix], nil
		}
	case "Strims":
		ix := slices.IndexFunc(ui.pg1.streams.Strims.Data, func(sd sc.StrimsStreamData) bool {
			return sd.Channel == primaryText
		})
		if ix != -1 {
			return &ui.pg1.streams.Strims.Data[ix], nil
		}
	}
	return nil, errors.New("cannot open empty result")
}

func streamToUrl(data sc.StreamData, method OpenMethod) (*url.URL, error) {
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

func (ut *UrlTemplates) apply(data sc.StreamData) (*url.URL, error) {
	newHost, err := executeTemplateString(ut.Host, data)
	if err != nil {
		return nil, err
	}
	newPath, err := executeTemplateString(ut.Path, data)
	if err != nil {
		return nil, err
	}
	newValues := make(url.Values)
	for key, value := range ut.Values {
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

func executeTemplateString(templateString string, data sc.StreamData) (string, error) {
	tmpl, err := template.New("t").Parse(templateString)
	if err != nil {
		return "", err
	}
	var buffer bytes.Buffer
	err = tmpl.Execute(&buffer, map[string]string{
		"Name":    strings.ToLower(data.GetName()),
		"Service": data.GetService(),
	})
	return buffer.String(), err
}

func (ui *UI) getProgram(method OpenMethod) (string, error) {
	switch method {
	case lnkOpenMpv:
		return "/usr/bin/mpv", nil
	default:
		browser := os.Getenv("BROWSER")
		if browser == "" {
			return "", errors.New("set $BROWSER before opening links")
		}
		return browser, nil
	}
}
