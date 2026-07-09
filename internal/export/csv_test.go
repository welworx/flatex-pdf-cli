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
	fields := strings.Split(lines[1], ",")
	if fields[9] != "0" { // "quantity" column
		t.Errorf("expected zero quantity to render as \"0\", got %q", fields[9])
	}
}
