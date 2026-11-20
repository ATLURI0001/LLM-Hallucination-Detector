package report

import (
	"strings"
	"testing"
)

func TestMarkdownTable_Render(t *testing.T) {
	tbl := MarkdownTable{
		Headers: []string{"Claim ID", "Status", "Confidence"},
		Rows: [][]string{
			{"c-001", "VERIFIED", "0.95"},
			{"c-002", "HALLUCINATION", "0.12"},
		},
	}

	out := tbl.Render()
	if !strings.Contains(out, "| Claim ID | Status | Confidence |") {
		t.Fatalf("expected header line in output, got: %s", out)
	}
	if !strings.Contains(out, "| --- | --- | --- |") {
		t.Fatalf("expected separator line in output, got: %s", out)
	}
	if !strings.Contains(out, "| c-001 | VERIFIED | 0.95 |") {
		t.Fatalf("expected row 1 in output, got: %s", out)
	}
	if tbl.RowCount() != 2 {
		t.Fatalf("expected 2 rows, got %d", tbl.RowCount())
	}

	// Empty table
	empty := MarkdownTable{}
	if empty.Render() != "" {
		t.Fatalf("expected empty string for table without headers, got: %s", empty.Render())
	}
}
