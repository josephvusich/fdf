package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/josephvusich/fdf/report"
	"github.com/stretchr/testify/require"
)

func TestWriteReport_EmptyPath(t *testing.T) {
	assert := require.New(t)
	assert.NoError(writeReport("", nil, nil, newDB()))
}

func TestWriteReport_ValidJSON(t *testing.T) {
	assert := require.New(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "report.json")

	pairs := [][]string{{"a.txt", "b.txt"}}
	namePairs := [][]string{{"c.txt", "d.txt"}}

	assert.NoError(writeReport(path, pairs, namePairs, newDB()))

	data, err := os.ReadFile(path)
	assert.NoError(err)

	var r report.Report
	assert.NoError(json.Unmarshal(data, &r))
	assert.Equal(pairs, r.ContentMatches)
	assert.Equal(namePairs, r.NameMatches)
}

func TestWriteReport_EmptyData(t *testing.T) {
	assert := require.New(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "report.json")

	assert.NoError(writeReport(path, nil, nil, newDB()))

	data, err := os.ReadFile(path)
	assert.NoError(err)

	var r report.Report
	assert.NoError(json.Unmarshal(data, &r))
	assert.Nil(r.ContentMatches)
	assert.Nil(r.NameMatches)
}

func TestWriteReport_InvalidPath(t *testing.T) {
	assert := require.New(t)
	err := writeReport("/nonexistent/dir/report.json", nil, nil, newDB())
	assert.Error(err)
}
