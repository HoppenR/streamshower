package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	ls "github.com/HoppenR/libstreams"
)

type RedirectError struct {
	Location string
}

func (re *RedirectError) Error() string {
	return fmt.Sprintf("redirect to %s", re.Location)
}

type ResponseMetadata struct {
	RawLastModified string
	RefreshInterval time.Duration
	LastModified    time.Time
}

var (
	ErrStreamsNotModified  = errors.New("streams not modified")
	ErrStreamsUnauthorized = errors.New("not authorized")
)

func updateStreams(ctx context.Context, lastMeta *ResponseMetadata, addr string) (*ls.Streams, *ResponseMetadata, error) {
	ctxTo, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	streams, meta, err := GetServerData(ctxTo, addr, lastMeta)
	if err != nil {
		return nil, meta, err
	}
	sort.Sort(sort.Reverse(streams.Twitch))
	sort.Sort(sort.Reverse(streams.Strims))
	return streams, meta, nil
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

func GetServerData(ctx context.Context, address string, prevMeta *ResponseMetadata) (*ls.Streams, *ResponseMetadata, error) {
	noRedirectClient := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	req, err := http.NewRequestWithContext(ctx, "GET", address, nil)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Add("Accept", "application/octet-stream")

	if prevMeta != nil && prevMeta.RawLastModified != "" {
		req.Header.Set("If-Modified-Since", prevMeta.RawLastModified)
	}

	var resp *http.Response
	resp, err = noRedirectClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("fetching Streams failed: %w", err)
	}
	defer resp.Body.Close()

	return handleServerResponse(resp)
}

func handleServerResponse(resp *http.Response) (*ls.Streams, *ResponseMetadata, error) {
	var (
		err           error
		nextRefreshIn time.Duration
		lastModified  time.Time
	)
	meta := new(ResponseMetadata)
	meta.RawLastModified = resp.Header.Get("Last-Modified")
	if nextRefreshIn, err = getDurationUntilCacheInvalidation(resp); err == nil {
		meta.RefreshInterval = nextRefreshIn
	}
	if lastModified, err = time.Parse(http.TimeFormat, meta.RawLastModified); err == nil {
		meta.LastModified = lastModified
	}

	switch resp.StatusCode {
	case http.StatusOK:
		contentType := resp.Header.Get("Content-Type")
		if !strings.Contains(contentType, "application/octet-stream") {
			return nil, nil, fmt.Errorf("unexpected content type: %s", contentType)
		}

		var streams *ls.Streams
		streams, err = ls.DecodeStreams(resp.Body)
		if err != nil {
			return nil, nil, err
		}
		return streams, meta, nil
	case http.StatusFound:
		location := resp.Header.Get("Location")
		var relURL *url.URL
		relURL, err = url.Parse(location)
		if err != nil {
			return nil, nil, fmt.Errorf("could not parse redirect location: %w", err)
		}
		var absoluteURL *url.URL
		absoluteURL = resp.Request.URL.ResolveReference(relURL)
		return nil, meta, &RedirectError{Location: absoluteURL.String()}
	case http.StatusNotModified:
		return nil, meta, ErrStreamsNotModified
	case http.StatusUnauthorized:
		return nil, nil, ErrStreamsUnauthorized
	default:
		return nil, nil, fmt.Errorf("status getting streams: %d", resp.StatusCode)
	}
}

func getDurationUntilCacheInvalidation(resp *http.Response) (time.Duration, error) {
	cacheControl := resp.Header.Get("Cache-Control")
	if cacheControl == "" {
		return 0, errors.New("header not found: cache-control")
	}

	for part := range strings.SplitSeq(cacheControl, ",") {
		part = strings.TrimSpace(part)
		if secondsStr, ok := strings.CutPrefix(part, "max-age="); ok {
			seconds, err := strconv.Atoi(secondsStr)
			if err != nil {
				return 0, errors.New("could not parse cache-control header")
			}
			return time.Duration(seconds) * time.Second, nil
		}
	}
	return 0, errors.New("max-age not found in header")
}
