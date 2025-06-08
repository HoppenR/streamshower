package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
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
		if !ui.mainPage.streams.LastFetched.IsZero() {
			var nextUpdate time.Time
			nextUpdate = ui.mainPage.streams.LastFetched.Add(ui.mainPage.streams.RefreshInterval)
			fetchTimer.Reset(time.Until(nextUpdate))
		}
		select {
		case <-ctx.Done():
			return
		case <-ui.forceRemoteUpdateCh:
			setStatus("orange", "Sending update...")
			err = forceRemoteUpdate(ctx, ui.addr)
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
			// pass
		case <-fetchTimer.C:
			// pass
		}

		setStatus("orange", "Fetching streams...")
		var streams *ls.Streams
		streams, err = updateStreams(ctx, ui.addr)
		if errors.Is(err, context.Canceled) {
			return
		} else if errors.Is(err, ls.ErrAuthPending) {
			setStatus("yellow", "run `:sync` to refresh after authenticating")
			continue
		} else if err != nil {
			setStatus("red", fmt.Sprintf("Error fetching: %s", err))
			continue
		}
		ui.mainPage.streams = streams

		ui.app.QueueUpdate(func() {
			if ui.mainPage.streams.Strims.Len() == 0 {
				ui.app.SetFocus(ui.mainPage.twitchList)
				ui.mainPage.focusedList = ui.mainPage.twitchList
				ui.disableStrimsList()
			} else {
				ui.enableStrimsList()
			}
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
	req.Header.Set("Content-Type", "application/octet-stream")

	var resp *http.Response
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}
