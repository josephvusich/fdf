package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// TODO check console width to truncate and avoid autoscroll when non-quiet
	// TODO auto quiet on error, as it means probably not a terminal
	if prevANSI, err := terminalANSI(true); err == nil && !prevANSI {
		defer terminalANSI(prevANSI)
	}

	scanner := newScanner()
	dirs := scanner.options.ParseArgs(os.Args)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		fmt.Println("\nReceived", sig)
		fmt.Printf("\n%s\n", scanner.totals.PrettyFormat(scanner.options.Verb()))
		scanner.Exit(1)
	}()

	scanErr := scanner.Scan(dirs...)

	fmt.Printf("\033[2K\n%s\n", scanner.totals.PrettyFormat(scanner.options.Verb()))

	if err := writeReport(scanner.options.JsonReport, scanner.table.pairs); err != nil {
		fmt.Println("Unable to write JSON report:", err)
	}

	if scanErr != nil {
		fmt.Printf("Finished with error: %s\n", scanErr)
		os.Exit(1)
	}
}
