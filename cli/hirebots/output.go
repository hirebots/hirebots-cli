package main

// output.go — shared output helpers for CLI commands.
// Supports table format (default) and JSON format (-o json).

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// outputFormat is set by the --output-format / -o flag on the root command.
// Values: "table" (default) or "json".
var outputFormat string

// quietFlag is set by --quiet / -q. Suppresses notices and tips on stderr.
var quietFlag bool

// shouldPrintNotices returns true if notices/tips should be shown (stderr).
func shouldPrintNotices() bool {
	return !quietFlag && outputFormat != "json"
}

// printJSON pretty-prints raw JSON bytes to stdout.
func printJSON(data []byte) {
	var buf bytes.Buffer
	if json.Indent(&buf, data, "", "  ") == nil {
		fmt.Println(buf.String())
		return
	}
	// Not valid JSON — print raw
	fmt.Println(string(data))
}

// wantJSON returns true if the user requested JSON output via -o json.
func wantJSON() bool {
	return outputFormat == "json"
}

// ── Table helpers ───────────────────────────────────────────────

// tableRow holds one row of column values for table printing.
type tableRow []string

// printTable prints a header + rows as aligned columns.
// headers: column titles. rows: one slice of strings per row.
// widths: optional explicit column widths; if nil, auto-computed.
func printTable(headers []string, rows []tableRow, widths []int) {
	if len(rows) == 0 {
		fmt.Println("No items found.")
		return
	}

	// Auto-compute column widths from header + data
	if widths == nil {
		widths = make([]int, len(headers))
		for i, h := range headers {
			widths[i] = len(h)
		}
		for _, row := range rows {
			for i, val := range row {
				if i < len(widths) && len(val) > widths[i] {
					widths[i] = len(val)
				}
			}
		}
	}

	// Print header
	printTableRow(headers, widths)
	fmt.Println(strings.Repeat("-", sumWidths(widths)))

	// Print rows
	for _, row := range rows {
		printTableRow(row, widths)
	}
}

func printTableRow(cols []string, widths []int) {
	parts := make([]string, len(cols))
	for i, col := range cols {
		w := widths[i]
		if len(col) > w {
			col = col[:w]
		}
		parts[i] = fmt.Sprintf("%-*s", w, col)
	}
	fmt.Println(strings.Join(parts, "  "))
}

func sumWidths(widths []int) int {
	total := 0
	for _, w := range widths {
		total += w
	}
	return total + (len(widths)-1)*2 // separators
}

// ── Key-Value helpers (for show/detail commands) ────────────────

// printKeyValue prints a label: value pair with consistent indentation.
func printKeyValue(label, value string) {
	fmt.Printf("%-20s  %s\n", label, value)
}

// printKeyValueSection prints a section header followed by key-value pairs.
func printKeyValueSection(title string) {
	fmt.Printf("\n── %s ──\n", title)
}

// ── Format helpers ──────────────────────────────────────────────

// formatDateTime formats an ISO-8601 timestamp for display.
// Returns "N/A" if the timestamp is empty or null.
func formatDateTime(ts string) string {
	if ts == "" || ts == "null" {
		return "N/A"
	}
	t, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		// Try without nano
		t, err = time.Parse(time.RFC3339, ts)
		if err != nil {
			return ts // return raw if we can't parse
		}
	}
	return t.Format("2006-01-02 15:04 MST")
}

// joinStrings joins a slice of strings with ", ".
func joinStrings(ss []string) string {
	return strings.Join(ss, ", ")
}

// extractString safely extracts a string value from a map[string]interface{}.
func extractString(m map[string]interface{}, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case float64:
		return fmt.Sprintf("%.0f", val)
	default:
		return fmt.Sprintf("%v", val)
	}
}

// extractFloatString extracts a float value and formats it as a string.
func extractFloatString(m map[string]interface{}, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return "N/A"
	}
	switch val := v.(type) {
	case float64:
		if val == float64(int(val)) {
			return fmt.Sprintf("%.0f", val)
		}
		return fmt.Sprintf("%.2f", val)
	case string:
		return val
	default:
		return fmt.Sprintf("%v", val)
	}
}

// extractStringSlice extracts a []string from a map[string]interface{}.
func extractStringSlice(m map[string]interface{}, key string) []string {
	v, ok := m[key]
	if !ok || v == nil {
		return nil
	}
	switch val := v.(type) {
	case []interface{}:
		result := make([]string, 0, len(val))
		for _, item := range val {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	case []string:
		return val
	default:
		return nil
	}
}

// printNoticesPlaceholder is a no-op until cmd_notices.go is added in commit 3.
// This allows commands to call printNotices() without compilation errors
// before the notices module is implemented.
var printNotices = func(client *Client) {}