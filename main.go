package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
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

	basicAuthUser := strings.TrimSpace(os.Getenv("STREAMS_BASIC_AUTH_USER"))
	basicAuthPass := strings.TrimSpace(os.Getenv("STREAMS_BASIC_AUTH_PASS"))

	ui := NewUI()
	err := ui.SetAddress(*address)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid address: %s\n", err)
	}
	err = ui.SetBasicAuthCredentials(basicAuthUser, basicAuthPass)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot set basic auth: %s\n", err)
	}
	err = ui.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error running UI: %s", err)
	}
}
