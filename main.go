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

// Streamchecker
// TODO(sc): Support hosting server on a separate machine
//           [x] Server redirects client to authentication when needed
//           [ ] Support https:// protocol
//           [ ] Requests/redirect-URI is set correctly at all request/responses

// TODO(sc): [ ] Support automatically re-getting tokens on http status codes
// TODO(sc): [ ] Support automatically re-getting tokens on expiry
// TODO(sc): [x] Save follows data between requests.

// Streamshower
// TODO(ss): [ ] '/': Search instead of filter.
// TODO(ss): [ ] On-demand get start time for angelthump streams
//               From https://api.angelthump.com/v2/streams
// TODO(ss): [ ] Don't display m3u8 streams of angelthump, instead make them
//               expandable/collapsable under the angelthump stream they're from
// TODO(ss): [ ] Display age of data in the UI?

func main() {
	background := flag.Bool(
		"b",
		false,
		"Check for streams in the background and serve data over local network",
	)
	address := flag.String(
		"a",
		":8181",
		"Address of the server. Unset to disable data hosting",
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
		// bg.SetLiveCallback(notifyLives)
		bg.SetOfflineCallback(nil)
		err = bg.Run()
		if err != nil {
			log.Fatalln(err)
		}
		err = ad.SaveCache()
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

/*
func notifyLives(stream sc.StreamData) {
	var args []string
	iconbase := "/usr/share/icons/Adwaita/48x48/categories/"
	switch stream.GetService() {
	case "angelthump":
		if stream.GetName() == "psrngafk" {
			break
		}
		args = []string{
			stream.GetName(),
			"Is being viewed on Strims!",
			"--icon=" + iconbase + "applications-multimedia-symbolic.symbolic.png",
		}
	case "m3u8":
		break
	case "twitch":
		break
	case "twitch-followed":
		args = []string{
			stream.GetName(),
			"Just went live!",
			"--icon=" + iconbase + "applications-games-symbolic.symbolic.png",
		}
	case "twitch-vod":
		break
	case "youtube":
		break
	default:
		break
	}
	if args != nil {
		var args_action = []string{
			"--action=open",
			"--wait=5000",
		}
		args = append(args, args_action...)
		exec.Command("notify-send", args...).Run()
	}
}
*/
