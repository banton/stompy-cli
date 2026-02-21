package output

import (
	"fmt"
	"os"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"golang.org/x/term"
)

// Stompy brand colors (ANSI 256-color approximations)
var (
	colorTeal      = text.Colors{text.FgHiCyan}   // Primary — Stompy Teal #4A9B9B
	colorTerracotta = text.Colors{text.FgHiRed}    // Accent — Terracotta #D4785A
	colorForest    = text.Colors{text.FgHiGreen}   // Success — Forest #5B9A6B
	colorAmber     = text.Colors{text.FgHiYellow}  // Warning — Amber #D4A85A
	colorRust      = text.Colors{text.FgRed}       // Error — Rust #C75D5D
	colorInk       = text.Colors{text.FgWhite}     // Ink — foreground
	colorDim       = text.Colors{text.FgHiBlack}   // Muted text
)

// TableFormatter renders output as ASCII tables with Stompy brand colors.
type TableFormatter struct{}

// getTerminalWidth returns the current terminal width, defaulting to 100.
func getTerminalWidth() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || width <= 0 {
		return 100
	}
	return width
}

// FormatTable renders headers and rows as a colored table that fits the terminal.
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

	// Stompy-branded style
	style := table.StyleLight
	style.Color.Header = colorTeal
	style.Color.Row = text.Colors{text.FgWhite}
	style.Color.RowAlternate = text.Colors{text.FgHiWhite}
	style.Format.Header = text.FormatUpper
	t.SetStyle(style)

	// Constrain to terminal width
	termWidth := getTerminalWidth()
	t.SetAllowedRowLength(termWidth)

	// Set max column widths — give more space to text-heavy columns
	colConfigs := make([]table.ColumnConfig, len(headers))
	for i, h := range headers {
		colConfigs[i] = table.ColumnConfig{
			Number:      i + 1,
			WidthMax:    colMaxWidth(h, len(headers), termWidth),
			WidthMaxEnforcer: text.WrapSoft,
		}
	}
	t.SetColumnConfigs(colConfigs)

	return t.Render() + "\n"
}

// colMaxWidth determines the max width for a column based on its header name.
// Text-heavy columns (TITLE, NAME, DESCRIPTION, CONTENT, PREVIEW) get more space.
func colMaxWidth(header string, numCols, termWidth int) int {
	available := termWidth - (numCols+1)*3
	if available < 40 {
		available = 40
	}

	h := strings.ToUpper(header)
	switch h {
	case "TITLE", "NAME", "DESCRIPTION", "CONTENT", "PREVIEW", "TARGET TITLE":
		// Wide column: up to 40% of available space
		w := available * 40 / 100
		if w > 60 {
			w = 60
		}
		if w < 20 {
			w = 20
		}
		return w
	case "ID", "LINK ID":
		return 6
	case "ROLE", "TYPE":
		return 10
	case "STATUS", "PRIORITY":
		return 12
	default:
		// Normal columns get fair share, capped at 30
		perCol := available / numCols
		if perCol > 30 {
			perCol = 30
		}
		if perCol < 10 {
			perCol = 10
		}
		return perCol
	}
}

// FormatSingle renders key-value pairs as a colored vertical list.
func (f *TableFormatter) FormatSingle(fields []KeyValue) string {
	if len(fields) == 0 {
		return ""
	}

	maxKey := 0
	for _, kv := range fields {
		if len(kv.Key) > maxKey {
			maxKey = len(kv.Key)
		}
	}

	termWidth := getTerminalWidth()
	valueWidth := termWidth - maxKey - 4
	if valueWidth < 20 {
		valueWidth = 20
	}

	var buf strings.Builder
	for _, kv := range fields {
		val := kv.Value
		if len(val) > valueWidth {
			val = val[:valueWidth-3] + "..."
		}
		// Key in teal, value in default
		key := colorTeal.Sprint(fmt.Sprintf("%-*s", maxKey+1, kv.Key+":"))
		fmt.Fprintf(&buf, "%s  %s\n", key, val)
	}
	return buf.String()
}

// FormatRaw renders data as a plain string representation.
func (f *TableFormatter) FormatRaw(data any) string {
	return fmt.Sprintf("%v", data)
}

// --- Color helpers for use by commands ---

// ColorStatus returns a colorized status string.
func ColorStatus(status string) string {
	switch strings.ToLower(status) {
	case "open", "backlog", "triage", "proposed":
		return colorAmber.Sprint(status)
	case "in_progress", "confirmed", "approved":
		return colorTeal.Sprint(status)
	case "done", "resolved", "shipped", "decided":
		return colorForest.Sprint(status)
	case "cancelled", "rejected", "wont_fix":
		return colorRust.Sprint(status)
	default:
		return status
	}
}

// ColorPriority returns a colorized priority string.
func ColorPriority(priority string) string {
	switch strings.ToLower(priority) {
	case "critical", "urgent":
		return colorRust.Sprint(priority)
	case "high":
		return colorTerracotta.Sprint(priority)
	case "medium":
		return colorAmber.Sprint(priority)
	case "low":
		return colorDim.Sprint(priority)
	default:
		return priority
	}
}

// ColorType returns a colorized ticket type string.
func ColorType(ticketType string) string {
	switch strings.ToLower(ticketType) {
	case "bug":
		return colorRust.Sprint(ticketType)
	case "feature":
		return colorForest.Sprint(ticketType)
	case "task":
		return colorTeal.Sprint(ticketType)
	case "decision":
		return colorAmber.Sprint(ticketType)
	default:
		return ticketType
	}
}

// Teal returns text colored in Stompy teal.
func Teal(s string) string {
	return colorTeal.Sprint(s)
}

// Success returns text colored in success green.
func Success(s string) string {
	return colorForest.Sprint(s)
}

// Warn returns text colored in warning amber.
func Warn(s string) string {
	return colorAmber.Sprint(s)
}

// Error returns text colored in error rust.
func Error(s string) string {
	return colorRust.Sprint(s)
}

// Dim returns text in muted gray.
func Dim(s string) string {
	return colorDim.Sprint(s)
}

// Ensure unused imports are referenced
var _ = colorInk
