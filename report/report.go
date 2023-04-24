package report

type Report struct {
	ContentMatches [][]string `json:"content_matches"`
	NameMatches    [][]string `json:"name_matches"`
}
