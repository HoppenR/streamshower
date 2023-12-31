package main

import (
	"errors"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"syscall"

	sc "github.com/HoppenR/streamchecker"
)

type OpenMethod int

const (
	lnkOpenEmbed OpenMethod = iota
	lnkOpenHomePage
	lnkOpenMpv
	lnkOpenStrims
)

func (ui *UI) openSelectedStream(method OpenMethod) error {
	listIdx := ui.pg1.focusedList.GetCurrentItem()
	var data sc.StreamData
	primaryText, _ := ui.pg1.focusedList.GetItemText(listIdx)
	switch ui.pg1.focusedList.GetTitle() {
	case "Twitch":
		for _, v := range ui.pg1.streams.Twitch.Data {
			if v.UserName == primaryText {
				data = &v
				break
			}
		}
	case "Strims":
		for _, v := range ui.pg1.streams.Strims.Data {
			if v.Channel == primaryText {
				data = &v
				break
			}
		}
	}
	if data == nil {
		return errors.New("cannot open empty result")
	}
	program := ""
	switch method {
	case lnkOpenStrims, lnkOpenEmbed, lnkOpenHomePage:
		program = os.Getenv("BROWSER")
	case lnkOpenMpv:
		program = "/usr/bin/mpv"
	}
	if program == "" {
		return errors.New("set $BROWSER before opening links")
	}
	rawURL, err := streamToUrlString(data, method)
	if err != nil {
		return err
	}
	p, err := exec.LookPath(program)
	if err != nil {
		return err
	}

	cmd := exec.Command(p, rawURL)
	// Set the new process process group-ID to its process ID
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Pgid:    0,
		Setpgid: true,
	}
	return cmd.Start()
}

func streamToUrlString(data sc.StreamData, method OpenMethod) (string, error) {
	var (
		q url.Values
		u *url.URL
	)
	switch method {
	case lnkOpenEmbed:
		switch data.GetService() {
		case "angelthump":
			q = url.Values{
				"channel": {strings.ToLower(data.GetName())},
			}
			u = &url.URL{
				Host: "player.angelthump.com",
			}
		case "m3u8":
			u = &url.URL{
				Host: "strims.gg",
				Path: "m3u8/" + data.GetName(),
			}
		case "twitch", "twitch-followed":
			q = url.Values{
				"channel": {strings.ToLower(data.GetName())},
				"parent":  {"strims.gg"},
			}
			u = &url.URL{
				Host: "player.twitch.tv",
			}
		case "twitch-vod":
			q = url.Values{
				"video":  {"v" + data.GetName()},
				"parent": {"strims.gg"},
			}
			u = &url.URL{
				Host: "player.twitch.tv",
			}
		case "youtube":
			q = url.Values{
				"autoplay": {"true"},
			}
			u = &url.URL{
				Host: "www.youtube.com",
				Path: "embed/" + data.GetName(),
			}
		default:
			return "", errors.New("Platform " + data.GetService() + " not implemented!")
		}
	// TODO: Split into 2 separate
	case lnkOpenHomePage, lnkOpenMpv:
		switch data.GetService() {
		case "angelthump":
			switch method {
			case lnkOpenHomePage:
				u = &url.URL{
					Host: "angelthump.com",
					Path: data.GetName(),
				}
			case lnkOpenMpv:
				u = &url.URL{
					Host: "ams-haproxy.angelthump.com",
					Path: "/hls/" + data.GetName() + "/index.m3u8",
				}
			}
		case "m3u8":
			switch method {
			case lnkOpenHomePage:
				u = &url.URL{
					Host: "strims.gg",
					Path: "m3u8/" + data.GetName(),
				}
			case lnkOpenMpv:
				var err error
				// NOTE: This never keeps its query string
				//       u.RawQuery gets replaced with `q`s values
				u, err = url.Parse(data.GetName())
				if err != nil {
					return "", err
				}
			}
		case "twitch", "twitch-followed":
			u = &url.URL{
				Host: "www.twitch.tv",
				Path: data.GetName(),
			}
		case "twitch-vod":
			u = &url.URL{
				Host: "www.twitch.tv",
				Path: "videos/" + data.GetName(),
			}
		case "youtube":
			u = &url.URL{
				Host: "www.youtube.com",
				Path: "watch",
			}
			q = url.Values{
				"v": {data.GetName()},
			}
		default:
			return "", errors.New("Platform " + data.GetService() + " not implemented!")
		}
	case lnkOpenStrims:
		u = &url.URL{
			Host: "strims.gg",
			Path: strings.Replace(
				data.GetService(),
				"-followed",
				"",
				1,
			) + "/" + strings.ToLower(data.GetName()),
		}
	}
	u.Scheme = "https"
	u.RawQuery = q.Encode()
	return u.String(), nil
}
