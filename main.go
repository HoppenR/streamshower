package main

import (
	"flag"
	"fmt"
	"log"
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
	// TODO: Not sure? :
	//       Default for -b flag should be http://0.0.0.0 or https://
	//       but default without should be 0.0.0.0:8181
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

	log.SetFlags(log.Ltime | log.Lshortfile)

	if *background {
		cfg := new(Config)
		cfg.SetConfigFolder("streamshower")
		configErr := cfg.Load(ConfigFile)
		if configErr != nil {
			// TODO: If ad.configFolder is unset, it will prompt for
			//        input and then crash anyway when saving data
			log.Println(
				"warn: config read failed:",
				configErr.Error(),
			)
			err = cfg.GetFromUserInput()
			if err != nil {
				log.Fatalln(err)
			}
			err = cfg.Save(ConfigFile)
			if err != nil {
				log.Fatalln(err)
			}
		} else {
			log.Println("Read config data")
		}

		ad := new(sc.AuthData)
		ad.SetClientID(cfg.data.ClientID)
		ad.SetClientSecret(cfg.data.ClientSecret)
		ad.SetUserName(cfg.data.UserName)
		err = ad.SetCacheFolder(CacheFolder)
		if err != nil {
			log.Fatalln(err)
		}
		if *useCache {
			err = ad.GetCachedData()
			if err != nil {
				log.Println(
					"warn: could not read cached data:",
					err.Error(),
				)
			} else {
				log.Println("Read cached data")
			}
		}

		bg := sc.NewBG()
		bg.SetAddress(*address)
		bg.SetAuthData(ad)
		bg.SetInterval(*refreshTime)
		bg.SetRedirect(*redirect)
		bg.SetLiveCallback(nil)
		bg.SetOfflineCallback(nil)
		err = bg.Run()
		if err != nil {
			log.Fatalln(err)
		}
		err = ad.SaveCachedData()
		if err != nil {
			log.Fatalln(err)
		}
	}

	if !*background {
		ui := NewUI()
		ui.SetAddress(*address)
		err = ui.Run()
		if err != nil {
			log.Fatalln(err)
		}
	}
}
