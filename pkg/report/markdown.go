package report

import (
	"strings"
)

// MarkdownTable represents a formatted GitHub-flavored markdown table.
type MarkdownTable struct {
	Headers []string
	Rows    [][]string
}

// Render formats headers and rows into clean markdown table syntax.
func (t MarkdownTable) Render() string {
	if len(t.Headers) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("| " + strings.Join(t.Headers, " | ") + " |\n")
	var sep []string
	for range t.Headers {
		sep = append(sep, "---")
	}
	sb.WriteString("| " + strings.Join(sep, " | ") + " |\n")

	for _, row := range t.Rows {
		sb.WriteString("| " + strings.Join(row, " | ") + " |\n")
	}
	return sb.String()
}

// RowCount returns total data rows.
func (t MarkdownTable) RowCount() int {
	return len(t.Rows)
}
