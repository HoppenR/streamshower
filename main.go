package main

import (
	"flag"
	"fmt"
	"os"
)

// TODO:
//    - Support ~/.config/streamshower/rc for running mappings/commands on
//      startup?
// TODO:
//    - Fix wrongfully detecting bangs inside regex patterns in `:g/re!gex/p`

// BONUS:
//    - Command `:silent {mapping}`
//      - calls execCommandSilent `{mapping}`
//    - Show last fetch info somewhere in bottom right?
//      - Can split commandLine into a flexbox with two windows

func main() {
	address := flag.String(
		"a",
		"http://0.0.0.0:8181",
		"Address of the server",
	)

	flag.Parse()

	if flag.NArg() > 0 {
		flag.Usage()
		os.Exit(2)
	}

	ui := NewUI()
	ui.SetAddress(*address)
	err := ui.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error running UI: %s", err)
	}
}
