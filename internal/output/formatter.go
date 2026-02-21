package output

// KeyValue represents a labeled value for single-item display.
type KeyValue struct {
	Key   string
	Value string
}

// Formatter defines the interface for rendering CLI output in different formats.
type Formatter interface {
	FormatTable(headers []string, rows [][]string) string
	FormatSingle(fields []KeyValue) string
	FormatRaw(data any) string
}

// NewFormatter returns a Formatter for the given format string.
// Supported formats: "json", "yaml", "table" (default).
func NewFormatter(format string) Formatter {
	switch format {
	case "json":
		return &JSONFormatter{}
	case "yaml":
		return &YAMLFormatter{}
	default:
		return &TableFormatter{}
	}
}
