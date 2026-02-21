package output

import (
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v3"
)

// JSONFormatter renders output as JSON.
type JSONFormatter struct{}

// FormatTable renders headers and rows as a JSON array of objects.
func (f *JSONFormatter) FormatTable(headers []string, rows [][]string) string {
	items := make([]map[string]string, 0, len(rows))
	for _, row := range rows {
		item := make(map[string]string, len(headers))
		for i, h := range headers {
			if i < len(row) {
				item[h] = row[i]
			}
		}
		items = append(items, item)
	}
	return marshalJSON(items)
}

// FormatSingle renders key-value fields as a JSON object.
func (f *JSONFormatter) FormatSingle(fields []KeyValue) string {
	obj := make(map[string]string, len(fields))
	for _, kv := range fields {
		obj[kv.Key] = kv.Value
	}
	return marshalJSON(obj)
}

// FormatRaw renders data as indented JSON.
func (f *JSONFormatter) FormatRaw(data any) string {
	return marshalJSON(data)
}

func marshalJSON(v any) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("{\"error\": %q}", err.Error())
	}
	return string(b)
}

// YAMLFormatter renders output as YAML.
type YAMLFormatter struct{}

// FormatTable renders headers and rows as a YAML array of objects.
func (f *YAMLFormatter) FormatTable(headers []string, rows [][]string) string {
	items := make([]map[string]string, 0, len(rows))
	for _, row := range rows {
		item := make(map[string]string, len(headers))
		for i, h := range headers {
			if i < len(row) {
				item[h] = row[i]
			}
		}
		items = append(items, item)
	}
	return marshalYAML(items)
}

// FormatSingle renders key-value fields as a YAML object.
func (f *YAMLFormatter) FormatSingle(fields []KeyValue) string {
	obj := make(map[string]string, len(fields))
	for _, kv := range fields {
		obj[kv.Key] = kv.Value
	}
	return marshalYAML(obj)
}

// FormatRaw renders data as YAML.
func (f *YAMLFormatter) FormatRaw(data any) string {
	return marshalYAML(data)
}

func marshalYAML(v any) string {
	b, err := yaml.Marshal(v)
	if err != nil {
		return fmt.Sprintf("error: %q", err.Error())
	}
	return string(b)
}
