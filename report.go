package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/josephvusich/fdf/report"
)

func writeReport(path string, pairs, namePairs [][]string) error {
	if path == "" {
		return nil
	}

	fmt.Printf("Writing %s...\n", path)

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(&report.Report{
		ContentMatches: pairs,
		NameMatches:    namePairs,
	})
}
