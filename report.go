package main

import (
	"encoding/json"
	"os"

	"github.com/josephvusich/fdf/report"
)

func writeReport(path string, pairs [][]string) error {
	if path == "" {
		return nil
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(&report.Report{ContentMatches: pairs})
}
