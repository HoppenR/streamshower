package main

import (
	"flag"
	"fmt"
	"os"
)

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
