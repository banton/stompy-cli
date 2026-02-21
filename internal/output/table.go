package output

import (
	"fmt"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
)

// TableFormatter renders output as ASCII tables.
type TableFormatter struct{}

// FormatTable renders headers and rows as an ASCII table.
func (f *TableFormatter) FormatTable(headers []string, rows [][]string) string {
	t := table.NewWriter()

	headerRow := make(table.Row, len(headers))
	for i, h := range headers {
		headerRow[i] = h
	}
	t.AppendHeader(headerRow)

	for _, row := range rows {
		tableRow := make(table.Row, len(row))
		for i, cell := range row {
			tableRow[i] = cell
		}
		t.AppendRow(tableRow)
	}

	t.SetStyle(table.StyleLight)
	return t.Render() + "\n"
}

// FormatSingle renders key-value pairs as a vertical list.
func (f *TableFormatter) FormatSingle(fields []KeyValue) string {
	if len(fields) == 0 {
		return ""
	}

	// Find max key length for alignment
	maxKey := 0
	for _, kv := range fields {
		if len(kv.Key) > maxKey {
			maxKey = len(kv.Key)
		}
	}

	var buf strings.Builder
	for _, kv := range fields {
		fmt.Fprintf(&buf, "%-*s  %s\n", maxKey+1, kv.Key+":", kv.Value)
	}
	return buf.String()
}

// FormatRaw renders data as a plain string representation.
func (f *TableFormatter) FormatRaw(data any) string {
	return fmt.Sprintf("%v", data)
}
