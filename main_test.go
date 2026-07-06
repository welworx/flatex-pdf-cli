package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/welworx/flatex-pdf-cli/internal/schema"
)

// TestProcessPDFsContinuesPastFailures verifies that a single unparseable file
// does not abort the whole batch: good files are still processed and each
// failure is reported, not fatal.
func TestProcessPDFsContinuesPastFailures(t *testing.T) {
	files := []string{
		"testdata/trade_sample_1.pdf",    // good
		"testdata/does-not-exist.pdf",    // fails extraction
		"testdata/dividend_sample_1.pdf", // good
	}

	txns, _, errs := processPDFs(files, false)

	if len(txns) != 2 {
		t.Errorf("expected 2 transactions from the good files, got %d", len(txns))
	}
	if len(errs) != 1 {
		t.Errorf("expected 1 reported error, got %d", len(errs))
	}
}

func TestWriteOutputCSVFormat(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "out.csv")
	txns := []*schema.Transaction{{DocumentType: "TRADE", ISIN: "IE000YU9K6K2", Date: "2024-06-15", Type: "BUY", Quantity: 1, GrossValue: 50}}

	if err := writeOutput("csv", out, txns, nil, false); err != nil {
		t.Fatalf("writeOutput failed: %v", err)
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if !strings.Contains(string(data), "IE000YU9K6K2") {
		t.Errorf("expected CSV to contain ISIN, got: %s", data)
	}
}

func TestWriteOutputPPFormatRequiresOutputFile(t *testing.T) {
	if err := writeOutput("pp", "", nil, nil, false); err == nil {
		t.Fatal("expected error when -format pp used without -o")
	}
}

func TestWriteOutputPPFormatWritesTwoFiles(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "out.csv")
	txns := []*schema.Transaction{
		{DocumentType: "TRADE", ISIN: "IE000YU9K6K2", Date: "2024-06-15", Type: "BUY", Quantity: 1, GrossValue: 50},
		{DocumentType: "DIVIDEND", ISIN: "IE000YU9K6K2", Date: "2024-06-15", NetAmount: 10},
	}

	if err := writeOutput("pp", out, txns, nil, false); err != nil {
		t.Fatalf("writeOutput failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "out-portfolio.csv")); err != nil {
		t.Errorf("expected out-portfolio.csv: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "out-accounts.csv")); err != nil {
		t.Errorf("expected out-accounts.csv: %v", err)
	}
}

func TestWriteOutputUnknownFormat(t *testing.T) {
	if err := writeOutput("xlsx", "", nil, nil, false); err == nil {
		t.Fatal("expected error for unknown format")
	}
}
