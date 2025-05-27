package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	sc "github.com/HoppenR/streamchecker"
)

const (
	ConfigFile  = "config.json"
	CacheFolder = "streamshower"
)

// TODO(sc): Support disabling/enabling twitch/strims fetching
//           [ ] Add flags
//           [ ] Send empty data to client requests for disabled platform

// Streamshower
// TODO(ss): [ ] '/': Search instead of filter.
// TODO(ss): [ ] On-demand get start time for angelthump streams
//               From https://api.angelthump.com/v2/streams
// TODO(ss): [ ] Don't display m3u8 streams of angelthump, instead make them
//               expandable/collapsable under the angelthump stream they're from

func main() {
	background := flag.Bool(
		"b",
		false,
		"Check for streams in the background and serve data over the network",
	)
	address := flag.String(
		"a",
		"http://0.0.0.0:8181",
		"Address of the server",
	)
	redirect := flag.String(
		"e",
		"http://localhost:8181/oauth-callback",
		"Callback address for authenticating",
	)
	refreshTime := flag.Duration(
		"r",
		5*time.Minute,
		"How often the daemon refreshes the data",
	)
	useCache := flag.Bool(
		"u",
		true,
		"Use cache, set to false to refresh cache (useful after making changes to config.json)",
	)
	flag.Usage = func() {
		fmt.Fprintf(
			flag.CommandLine.Output(),
			"Usage: %s [-b] [-a=ADDRESS] [-r=DURATION] [-u=false]\n",
			os.Args[0],
		)
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() > 0 {
		flag.Usage()
		os.Exit(2)
	}

	var err error

	if *background {
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

		cfg := new(Config)
		err = cfg.SetConfigFolder("streamshower")
		if err != nil {
			logger.Error("error making config folder", "err", err)
			return
		}
		configErr := cfg.Load(ConfigFile)
		if configErr != nil {
			logger.Warn("config read failed", "err", configErr)
			err = cfg.GetFromEnv()
			if err != nil {
				logger.Error("error reading env", "err", err)
				return
			}
			err = cfg.Save(ConfigFile)
			if err != nil {
				logger.Error("error saving config", "err", err)
				return
			}
		} else {
			logger.Info("read config data")
		}

		ad := new(sc.AuthData)
		ad.SetClientID(cfg.data.ClientID)
		ad.SetClientSecret(cfg.data.ClientSecret)
		ad.SetUserName(cfg.data.UserName)
		err = ad.SetCacheFolder(CacheFolder)
		if err != nil {
			logger.Error("error making cache folder", "err", err)
			return
		}
		if *useCache {
			err = ad.GetCachedData()
			if err != nil {
				logger.Warn("cache read failed", "err", configErr)
			} else {
				logger.Info("read cached data")
			}
		}

		srv := sc.NewServer()
		srv.SetAddress(*address)
		srv.SetAuthData(ad)
		srv.SetInterval(*refreshTime)
		srv.SetRedirect(*redirect)
		srv.EnableStrims(false)
		srv.SetLogger(logger)
		err = srv.Run()
		if err != nil {
			logger.Error("server exited abnormally", "err", err)
			return
		}
		err = ad.SaveCachedData()
		if err != nil {
			logger.Error("error saving cache", "err", err)
			return
		}
	}

	if !*background {
		ui := NewUI()
		ui.SetAddress(*address)
		err = ui.Run()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error running UI: %s", err)
		}
	}
}
