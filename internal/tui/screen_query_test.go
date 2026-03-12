package tui

import (
	"testing"

	"oslo/internal/export"
)

func TestInferExportFormatFromPath(t *testing.T) {
	tests := []struct {
		path string
		want export.Format
	}{
		{path: "query-results.csv", want: export.FormatCSV},
		{path: "query-results.tsv", want: export.FormatTSV},
		{path: "query-results.json", want: export.FormatJSON},
		{path: "query-results.txt", want: export.FormatTable},
		{path: "query-results", want: export.FormatTable},
	}

	for _, tt := range tests {
		got := inferExportFormat(tt.path)
		if got != tt.want {
			t.Fatalf("inferExportFormat(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}
