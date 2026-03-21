package main

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os/exec"
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
	fetchTimer := time.NewTimer(100 * time.Millisecond)
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
			ui.app.QueueUpdateDraw(func() {})
			continue
		case <-ui.updateStreamsCh:
			fetchTimer.Stop()
			// pass
		case <-fetchTimer.C:
			// pass
		}

		setStatus("orange", "Fetching streams...")
		var (
			streams *ls.Streams
			meta    *ResponseMetadata
		)
		streams, meta, err = updateStreams(ctx, ui.fetchMeta, ui.addr.String())

		var redirectErr *RedirectError
		if errors.Is(err, context.Canceled) {
			return
		} else if errors.Is(err, ErrStreamsNotModified) {
			ui.fetchMeta = meta
			setStatus("green", fmt.Sprintf(
				"No updates (%d Twitch streams and %d Strims streams)",
				ui.mainPage.streams.Twitch.Len(),
				ui.mainPage.streams.Strims.Len(),
			))
			nextUpdate := ui.fetchMeta.LastModified.Add(ui.fetchMeta.RefreshInterval)
			fetchTimer.Reset(time.Until(nextUpdate))
			continue
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
		ui.fetchMeta = meta
		nextUpdate := ui.fetchMeta.LastModified.Add(ui.fetchMeta.RefreshInterval)
		fetchTimer.Reset(time.Until(nextUpdate))

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
