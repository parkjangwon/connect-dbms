package export

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"oslo/internal/db"
)

type Format string

const (
	FormatTable Format = "table"
	FormatJSON  Format = "json"
	FormatCSV   Format = "csv"
	FormatTSV   Format = "tsv"
)

func ParseFormat(s string) (Format, error) {
	switch strings.ToLower(s) {
	case "table", "":
		return FormatTable, nil
	case "json":
		return FormatJSON, nil
	case "csv":
		return FormatCSV, nil
	case "tsv":
		return FormatTSV, nil
	default:
		return "", fmt.Errorf("unknown format: %s (use: table, json, csv, tsv)", s)
	}
}

func Write(w io.Writer, result *db.QueryResult, format Format, noHeader bool) error {
	switch format {
	case FormatJSON:
		return writeJSON(w, result)
	case FormatCSV:
		return writeDelimited(w, result, ',', noHeader)
	case FormatTSV:
		return writeDelimited(w, result, '\t', noHeader)
	case FormatTable:
		return writeTable(w, result)
	default:
		return fmt.Errorf("unknown format: %s", format)
	}
}

func writeJSON(w io.Writer, result *db.QueryResult) error {
	var rows []map[string]interface{}
	for _, row := range result.Rows {
		m := make(map[string]interface{})
		for i, col := range result.Columns {
			if i < len(row) {
				m[col] = row[i]
			}
		}
		rows = append(rows, m)
	}
	if rows == nil {
		rows = []map[string]interface{}{}
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(rows)
}

func writeDelimited(w io.Writer, result *db.QueryResult, delim rune, noHeader bool) error {
	cw := csv.NewWriter(w)
	cw.Comma = delim

	if !noHeader {
		if err := cw.Write(result.Columns); err != nil {
			return err
		}
	}

	for _, row := range result.Rows {
		record := make([]string, len(row))
		for i, v := range row {
			record[i] = fmt.Sprintf("%v", v)
		}
		if err := cw.Write(record); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

func writeTable(w io.Writer, result *db.QueryResult) error {
	if len(result.Columns) == 0 {
		return nil
	}

	// Calculate column widths
	widths := make([]int, len(result.Columns))
	for i, col := range result.Columns {
		widths[i] = len(col)
	}
	for _, row := range result.Rows {
		for i, v := range row {
			s := fmt.Sprintf("%v", v)
			if len(s) > widths[i] {
				widths[i] = len(s)
			}
		}
	}

	// Cap column width
	for i := range widths {
		if widths[i] > 50 {
			widths[i] = 50
		}
	}

	// Print header
	printRow(w, result.Columns, widths)
	printSep(w, widths)

	// Print rows
	for _, row := range result.Rows {
		strs := make([]string, len(row))
		for i, v := range row {
			strs[i] = fmt.Sprintf("%v", v)
		}
		printRow(w, strs, widths)
	}

	return nil
}

func printRow(w io.Writer, vals []string, widths []int) {
	for i, v := range vals {
		if len(v) > widths[i] {
			v = v[:widths[i]-2] + ".."
		}
		if i == 0 {
			fmt.Fprintf(w, " %-*s", widths[i], v)
		} else {
			fmt.Fprintf(w, " | %-*s", widths[i], v)
		}
	}
	fmt.Fprintln(w)
}

func printSep(w io.Writer, widths []int) {
	for i, wid := range widths {
		if i == 0 {
			fmt.Fprintf(w, " %s", strings.Repeat("-", wid))
		} else {
			fmt.Fprintf(w, "-+-%s", strings.Repeat("-", wid))
		}
	}
	fmt.Fprintln(w)
}
