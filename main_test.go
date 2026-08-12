package main

import (
	"encoding/json"
	"io"
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

func TestDiscoverPDFsFindsAndSortsRecursively(t *testing.T) {
	dir := t.TempDir()
	for _, p := range []string{"b.pdf", "a.pdf", "sub/c.pdf", "notes.txt"} {
		full := filepath.Join(dir, p)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("MkdirAll failed: %v", err)
		}
		if err := os.WriteFile(full, []byte("x"), 0o644); err != nil {
			t.Fatalf("WriteFile failed: %v", err)
		}
	}

	got, err := discoverPDFs(dir)
	if err != nil {
		t.Fatalf("discoverPDFs failed: %v", err)
	}

	want := []string{
		filepath.Join(dir, "a.pdf"),
		filepath.Join(dir, "b.pdf"),
		filepath.Join(dir, "sub/c.pdf"),
	}
	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("expected %v, got %v", want, got)
			break
		}
	}
}

func TestWriteOutputCSVFormat(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "out.csv")
	txns := []*schema.Transaction{{DocumentType: "TRADE", ISIN: "IE000YU9K6K2", Date: "2024-06-15", Type: "BUY", Quantity: 1, GrossValue: 50}}

	if err := writeOutput("csv", out, "en", txns, nil, false); err != nil {
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
	if err := writeOutput("pp", "", "en", nil, nil, false); err == nil {
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

	if err := writeOutput("pp", out, "en", txns, nil, false); err != nil {
		t.Fatalf("writeOutput failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "out-portfolio.csv")); err != nil {
		t.Errorf("expected out-portfolio.csv: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "out-accounts.csv")); err != nil {
		t.Errorf("expected out-accounts.csv: %v", err)
	}
}

func TestWriteOutputPPFormatRejectsUnknownLangWithoutWritingFiles(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "out.csv")
	txns := []*schema.Transaction{{DocumentType: "TRADE", ISIN: "IE000YU9K6K2", Date: "2024-06-15", Type: "BUY", Quantity: 1, GrossValue: 50}}

	if err := writeOutput("pp", out, "fr", txns, nil, false); err == nil {
		t.Fatal("expected error for unknown lang")
	}

	if _, err := os.Stat(filepath.Join(dir, "out-portfolio.csv")); !os.IsNotExist(err) {
		t.Errorf("expected out-portfolio.csv to not be created, stat err: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "out-accounts.csv")); !os.IsNotExist(err) {
		t.Errorf("expected out-accounts.csv to not be created, stat err: %v", err)
	}
}

func TestWriteOutputUnknownFormat(t *testing.T) {
	if err := writeOutput("xlsx", "", "en", nil, nil, false); err == nil {
		t.Fatal("expected error for unknown format")
	}
}

// captureStdout redirects os.Stdout for the duration of fn and returns what
// was written to it.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "stdout")
	if err != nil {
		t.Fatalf("CreateTemp failed: %v", err)
	}
	orig := os.Stdout
	os.Stdout = f
	defer func() { os.Stdout = orig }()

	fn()

	if err := f.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
	data, err := os.ReadFile(f.Name())
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	return string(data)
}

// The json format is the default and, unlike csv/pp, was previously exercised
// only through schema-level marshal tests — never through writeOutput itself.

func TestWriteOutputJSONEmitsBareArrayWithoutMetadataFlag(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "out.json")
	txns := []*schema.Transaction{{DocumentType: "TRADE", ISIN: "IE000YU9K6K2", Date: "2024-06-15", Type: "BUY", Quantity: 1, GrossValue: 50}}
	meta := &schema.DocumentMetadata{DepotNumber: "123456789"}

	// Metadata is passed but includeMetadata is false: it must not leak out.
	if err := writeOutput("json", out, "en", txns, meta, false); err != nil {
		t.Fatalf("writeOutput failed: %v", err)
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	var got []*schema.Transaction
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("expected a bare JSON array, got %s (%v)", data, err)
	}
	if len(got) != 1 || got[0].ISIN != "IE000YU9K6K2" {
		t.Errorf("expected the one transaction, got %s", data)
	}
	if strings.Contains(string(data), "123456789") {
		t.Errorf("metadata leaked without -include-metadata: %s", data)
	}
}

func TestWriteOutputJSONWithMetadataWrapsTransactions(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "out.json")
	txns := []*schema.Transaction{{DocumentType: "TRADE", ISIN: "IE000YU9K6K2", Date: "2024-06-15", Type: "BUY", Quantity: 1, GrossValue: 50}}
	meta := &schema.DocumentMetadata{DepotNumber: "123456789", DepotHolder: "Max Mustermann"}

	if err := writeOutput("json", out, "en", txns, meta, true); err != nil {
		t.Fatalf("writeOutput failed: %v", err)
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	var got schema.Output
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("expected a JSON object, got %s (%v)", data, err)
	}
	if got.Metadata == nil || got.Metadata.DepotNumber != "123456789" {
		t.Errorf("expected depot metadata, got %s", data)
	}
	if len(got.Transactions) != 1 {
		t.Errorf("expected 1 wrapped transaction, got %s", data)
	}
}

// writeOutput disables Go's default HTML escaping, so security names keep their
// literal characters instead of turning into & for downstream importers.
func TestWriteOutputJSONDoesNotEscapeAmpersandInSecurityName(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "out.json")
	txns := []*schema.Transaction{{DocumentType: "TRADE", SecurityName: "Procter & Gamble", Date: "2024-06-15", Type: "BUY"}}

	if err := writeOutput("json", out, "en", txns, nil, false); err != nil {
		t.Fatalf("writeOutput failed: %v", err)
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if !strings.Contains(string(data), "Procter & Gamble") {
		t.Errorf("expected unescaped ampersand, got %s", data)
	}
}

func TestWriteOutputJSONWritesToStdoutWhenNoOutputFile(t *testing.T) {
	txns := []*schema.Transaction{{DocumentType: "TRADE", ISIN: "IE000YU9K6K2", Date: "2024-06-15", Type: "BUY"}}

	var err error
	got := captureStdout(t, func() {
		err = writeOutput("json", "", "en", txns, nil, false)
	})
	if err != nil {
		t.Fatalf("writeOutput failed: %v", err)
	}
	if !strings.Contains(got, "IE000YU9K6K2") {
		t.Errorf("expected transaction on stdout, got %q", got)
	}
}

func TestWriteToReportsErrorForUncreatablePath(t *testing.T) {
	// A path whose parent directory does not exist cannot be created.
	bad := filepath.Join(t.TempDir(), "no-such-dir", "out.json")

	if err := writeTo(bad, func(w io.Writer) error { return nil }); err == nil {
		t.Fatal("expected an error creating a file under a missing directory")
	}
}

// -include-source stamps each transaction with the PDF it came from; without
// the flag the field must stay empty.
func TestProcessPDFsIncludeSourceStampsFilename(t *testing.T) {
	files := []string{"testdata/trade_sample_1.pdf"}

	with, _, errs := processPDFs(files, true)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(with) == 0 {
		t.Fatal("expected at least one transaction")
	}
	if with[0].Source != "trade_sample_1.pdf" {
		t.Errorf("expected source trade_sample_1.pdf, got %q", with[0].Source)
	}

	without, _, _ := processPDFs(files, false)
	if without[0].Source != "" {
		t.Errorf("expected empty source without the flag, got %q", without[0].Source)
	}
}

func TestProcessPDFsCapturesDepotMetadata(t *testing.T) {
	_, meta, errs := processPDFs([]string{"testdata/trade_sample_1.pdf"}, false)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if meta == nil {
		t.Fatal("expected depot metadata from the fixture")
	}
	if meta.DepotNumber == "" && meta.DepotHolder == "" && meta.AccountNumber == "" {
		t.Error("expected at least one metadata field to be populated")
	}
}

// The documented exit-code contract: -help and -version succeed, a bare
// invocation is a usage error.
func TestCommandExitCodes(t *testing.T) {
	if got := captureStdoutCode(t, help); got != 0 {
		t.Errorf("help() = %d, want 0", got)
	}
	if got := captureStdoutCode(t, printVersion); got != 0 {
		t.Errorf("printVersion() = %d, want 0", got)
	}
	if got := usage(); got != 2 {
		t.Errorf("usage() = %d, want 2", got)
	}
}

func captureStdoutCode(t *testing.T, fn func() int) int {
	t.Helper()
	var code int
	captureStdout(t, func() { code = fn() })
	return code
}

func TestPrintVersionFallsBackToDev(t *testing.T) {
	orig := version
	version = ""
	defer func() { version = orig }()

	got := captureStdout(t, func() { printVersion() })
	if !strings.Contains(got, "dev") {
		t.Errorf("expected dev fallback when version is unset, got %q", got)
	}
}

func TestDiscoverPDFsMissingPathIsAnError(t *testing.T) {
	if _, err := discoverPDFs(filepath.Join(t.TempDir(), "no-such-dir")); err == nil {
		t.Fatal("expected an error for a nonexistent path")
	}
}

// -format pp writes two derived files; a failure on the first must be
// reported rather than swallowed on the way to writing the second.
func TestWriteOutputPPFormatReportsUnwritablePath(t *testing.T) {
	out := filepath.Join(t.TempDir(), "no-such-dir", "out.csv")
	txns := []*schema.Transaction{{DocumentType: "TRADE", ISIN: "IE000YU9K6K2", Date: "2024-06-15", Type: "BUY", Quantity: 1, GrossValue: 50}}

	if err := writeOutput("pp", out, "en", txns, nil, false); err == nil {
		t.Fatal("expected an error writing into a nonexistent directory")
	}
}

func TestWriteOutputPPFormatGermanLang(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "out.csv")
	txns := []*schema.Transaction{{DocumentType: "TRADE", ISIN: "IE000YU9K6K2", Date: "2024-06-15", Type: "BUY", Quantity: 1, GrossValue: 50}}

	if err := writeOutput("pp", out, "de", txns, nil, false); err != nil {
		t.Fatalf("writeOutput failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "out-portfolio.csv"))
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if !strings.Contains(string(data), "Kauf") {
		t.Errorf("expected German Kauf label, got: %s", data)
	}
}
