package export

import (
	"bytes"
	"strings"
	"testing"

	"github.com/welworx/flatex-pdf-cli/internal/schema"
)

func TestWriteCSVHeaderAndRow(t *testing.T) {
	txns := []*schema.Transaction{
		{
			DocumentType: "TRADE",
			ISIN:         "IE000YU9K6K2",
			Date:         "2024-06-15",
			Type:         "BUY",
			Quantity:     1.5,
			GrossValue:   50.01,
		},
	}

	var buf bytes.Buffer
	if err := WriteCSV(&buf, txns); err != nil {
		t.Fatalf("WriteCSV failed: %v", err)
	}

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected header + 1 row, got %d lines: %v", len(lines), lines)
	}
	if !strings.HasPrefix(lines[0], "source,order_number,transaction_number,document_type,isin") {
		t.Errorf("unexpected header: %s", lines[0])
	}
	if !strings.Contains(lines[1], "TRADE") || !strings.Contains(lines[1], "IE000YU9K6K2") {
		t.Errorf("unexpected row: %s", lines[1])
	}
}

func TestWriteCSVZeroFloatIsLiteralZero(t *testing.T) {
	txns := []*schema.Transaction{{DocumentType: "TRADE", ISIN: "X", Date: "2024-06-15"}}

	var buf bytes.Buffer
	if err := WriteCSV(&buf, txns); err != nil {
		t.Fatalf("WriteCSV failed: %v", err)
	}

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if got := csvField(t, lines, "quantity"); got != "0" {
		t.Errorf("expected zero quantity to render as \"0\", got %q", got)
	}
}

// TestWriteCSVCostColumnsBlankWithoutCostBlock pins the one exception to the
// literal-zero rule: a document with no charge block must leave the cost
// columns empty, so that a real "Provision: 0,00 EUR" stays distinguishable
// from a document that never had a Provision line.
func TestWriteCSVCostColumnsBlankWithoutCostBlock(t *testing.T) {
	txns := []*schema.Transaction{
		{DocumentType: "DIVIDEND", ISIN: "X", Date: "2024-06-15"},
		{DocumentType: "TRADE", ISIN: "Y", Date: "2024-06-15",
			Costs: &schema.Costs{Provision: 0, ForeignExpenses: 3, Total: 3}},
	}

	var buf bytes.Buffer
	if err := WriteCSV(&buf, txns); err != nil {
		t.Fatalf("WriteCSV failed: %v", err)
	}
	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")

	for _, col := range []string{"provision", "total_costs", "fee_courtage"} {
		if got := csvField(t, lines[:2], col); got != "" {
			t.Errorf("no cost block: expected %s blank, got %q", col, got)
		}
	}
	row := []string{lines[0], lines[2]}
	if got := csvField(t, row, "provision"); got != "0" {
		t.Errorf("charged nothing: expected provision \"0\", got %q", got)
	}
	if got := csvField(t, row, "total_costs"); got != "3" {
		t.Errorf("expected total_costs \"3\", got %q", got)
	}
}

// csvField returns the value of the named column in lines[1], resolving the
// index through the header row so that adding columns cannot silently
// misalign these assertions.
func csvField(t *testing.T, lines []string, name string) string {
	t.Helper()
	header := strings.Split(lines[0], ",")
	fields := strings.Split(lines[1], ",")
	for i, h := range header {
		if h == name {
			if i >= len(fields) {
				t.Fatalf("row has %d fields, header column %q is at %d", len(fields), name, i)
			}
			return fields[i]
		}
	}
	t.Fatalf("no column named %q in header %q", name, lines[0])
	return ""
}
