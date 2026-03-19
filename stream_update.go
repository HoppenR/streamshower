package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os/exec"
	"sort"
	"time"

	ls "github.com/HoppenR/libstreams"
)

func (ui *UI) streamUpdateLoop(ctx context.Context) {
	setStatus := func(color string, text string) {
		ui.app.QueueUpdateDraw(func() {
			ui.mainPage.appStatusText.SetText(fmt.Sprintf("[%s]%s[-]", color, text))
		})
	}
	defer ui.wg.Done()

	var err error
	fetchTimer := time.NewTimer(0)
	defer fetchTimer.Stop()
	redrawTimer := time.NewTicker(time.Second)
	defer redrawTimer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ui.forceRemoteUpdateCh:
			setStatus("orange", "Sending update...")
			err = forceRemoteUpdate(ctx, ui.addr.String())
			if errors.Is(err, context.Canceled) {
				return
			} else if err != nil {
				setStatus("red", fmt.Sprintf("Error updating: %s", err))
			}
			continue
		case <-redrawTimer.C:
			ui.app.Draw()
			continue
		case <-ui.updateStreamsCh:
			fetchTimer.Stop()
			// pass
		case <-fetchTimer.C:
			// pass
		}

		setStatus("orange", "Fetching streams...")
		var streams *ls.Streams
		streams, err = updateStreams(ctx, ui.addr.String())

		var redirectErr *ls.RedirectError
		if errors.Is(err, context.Canceled) {
			return
		} else if errors.As(err, &redirectErr) {
			var absoluteURL *url.URL
			absoluteURL, err = url.Parse(redirectErr.Location)
			if err != nil {
				setStatus("red", "invalid redirect location")
				continue
			}
			if absoluteURL.Scheme != "http" && absoluteURL.Scheme != "https" {
				setStatus("red", "refusing to redirect to non-web scheme")
				continue
			}
			cmd := exec.Command("xdg-open", redirectErr.Location)
			err = cmd.Start()
			if err != nil {
				setStatus("red", "Could not launch xdg-open")
				continue
			}
			setStatus("yellow", "run `:sync` to refresh after authenticating")
			go func() {
				_ = cmd.Wait()
			}()
			continue
		} else if err != nil {
			setStatus("red", fmt.Sprintf("Error fetching: %s", err))
			fetchTimer.Reset(time.Minute)
			continue
		}
		ui.mainPage.streams = streams
		fetchTimer.Reset(ui.mainPage.streams.RefreshInterval)

		ui.app.QueueUpdate(func() {
			ui.mainPage.refreshTwitchList()
			ui.mainPage.refreshStrimsList()
		})
		setStatus("green", fmt.Sprintf(
			"Fetched %d Twitch streams and %d Strims streams",
			ui.mainPage.streams.Twitch.Len(),
			ui.mainPage.streams.Strims.Len(),
		))
	}
}

func updateStreams(ctx context.Context, addr string) (*ls.Streams, error) {
	ctxTo, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	streams, err := ls.GetServerData(ctxTo, addr)
	if err != nil {
		return nil, err
	}
	sort.Sort(sort.Reverse(streams.Twitch))
	sort.Sort(sort.Reverse(streams.Strims))
	return streams, nil
}

func forceRemoteUpdate(ctx context.Context, addr string) error {
	ctxTo, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctxTo, http.MethodPost, addr, nil)
	if err != nil {
		return err
	}

	var resp *http.Response
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	err = resp.Body.Close()

	return err
}
