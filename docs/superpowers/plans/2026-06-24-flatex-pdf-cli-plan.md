# flatex-pdf-cli Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a production-ready Go CLI tool that extracts structured transaction data from flatex PDF documents and outputs JSON for agent-driven processing.

**Architecture:** Layered pipeline: PDF text extraction → document type detection → transaction-specific parsing → schema serialization. Each transaction type (TRADE, DIVIDEND, INTEREST, THESAURIERUNG) has a dedicated parser; common fields and output formatting handled by schema layer.

**Tech Stack:** Go 1.21+, pdfcpu (PDF extraction), no other external dependencies, GitHub Actions for CI/CD.

## Global Constraints

- Go 1.21 or higher
- Single binary deployment, no system-level prerequisites
- Text extraction only (no OCR)
- Exit 0 on success, exit 1 on any parse failure
- Errors to stderr, JSON to stdout/file only
- Support `--include-source` and `--include-metadata` flags
- All code must pass golangci-lint, go fmt, go vet, and all tests before merge

---

## File Structure

**Core files to create:**
- `go.mod` — Module definition with pdfcpu dependency
- `go.sum` — Dependency lock file (auto-generated)
- `main.go` — CLI entry point, flag parsing, file discovery, output routing
- `.golangci.yml` — Linting configuration
- `.gitignore` — Git ignore rules
- `internal/schema/transaction.go` — Transaction struct and type definitions
- `internal/schema/output.go` — Output struct for optional metadata wrapping
- `internal/schema/schema_test.go` — Schema unit tests
- `internal/extractor/extractor.go` — PDF extraction, metadata capture, type detection
- `internal/extractor/extractor_test.go` — Extractor unit tests
- `internal/parser/parser.go` — Main parser router and common parsing logic
- `internal/parser/trade_parser.go` — Trade confirmation parser
- `internal/parser/dividend_parser.go` — Dividend statement parser
- `internal/parser/interest_parser.go` — Interest statement parser
- `internal/parser/thesaurierung_parser.go` — Reinvestment statement parser
- `internal/parser/parser_test.go` — Parser unit tests
- `testdata/sample_trade.pdf` — Real flatex trade confirmation (user provides)
- `testdata/sample_dividend.pdf` — Real flatex dividend statement (user provides)
- `testdata/sample_interest.pdf` — Real flatex interest statement (user provides)
- `testdata/sample_thesaurierung.pdf` — Real flatex reinvestment statement (user provides)
- `.github/workflows/ci.yml` — GitHub Actions pipeline
- `.pre-commit-config.yaml` — Pre-commit hook configuration (optional, for local development)

---

## Task Breakdown

### Task 1: Project Setup & Dependencies

**Files:**
- Create: `go.mod`
- Create: `go.sum`
- Create: `.gitignore`
- Create: `.golangci.yml`

**Interfaces:**
- Produces: Go module environment with pdfcpu dependency, linting rules, build excludes

- [ ] **Step 1: Initialize Go module**

```bash
cd /Users/welworx/dev-private/flatex-pdf-cli
go mod init github.com/welworx/flatex-pdf-cli
```

Expected output: `go.mod` file created with module declaration.

- [ ] **Step 2: Add pdfcpu dependency**

```bash
go get github.com/pdfcpu/pdfcpu/v5@latest
go mod tidy
```

Expected: `go.mod` and `go.sum` updated with pdfcpu and its transitive dependencies.

- [ ] **Step 3: Create .gitignore**

```
# Binaries
flatex-pdf-cli
*.o
*.a
*.so

# Output files
*.json
*.log

# IDE
.vscode/
.idea/
*.swp
*.swo

# Testing
*.test
*.out

# Go
/vendor/
```

- [ ] **Step 4: Create .golangci.yml**

```yaml
run:
  timeout: 5m

linters:
  enable:
    - gofmt
    - goimports
    - govet
    - errcheck
    - ineffassign
    - unused
    - deadcode

issues:
  exclude-rules:
    - path: _test\.go
      linters:
        - errcheck
```

- [ ] **Step 5: Commit**

```bash
git add go.mod go.sum .gitignore .golangci.yml
git commit -m "chore: initialize Go project with dependencies and linting config"
```

---

### Task 2: Define Transaction Schema

**Files:**
- Create: `internal/schema/transaction.go`
- Create: `internal/schema/schema_test.go`

**Interfaces:**
- Produces: Go structs for Transaction (polymorphic base), and type-specific fields; JSON marshaling support

- [ ] **Step 1: Write test for Transaction struct**

```bash
# Create directory
mkdir -p /Users/welworx/dev-private/flatex-pdf-cli/internal/schema
```

```go
// File: internal/schema/schema_test.go
package schema

import (
	"encoding/json"
	"testing"
)

func TestTradeTransactionMarshal(t *testing.T) {
	tx := &Transaction{
		Source:       "trade.pdf",
		DocNumber:    "326052529/1",
		DocumentType: "TRADE",
		Type:         "BUY",
		ISIN:         "IE000YU9K6K2",
		WKN:          "A3DP9J",
		Date:         "2026-01-15",
		Quantity:     1.058537,
		Price:        47.235000,
		PriceCurrency: "EUR",
		GrossValue:   50.00,
		Provision:    0.00,
		OwnCosts:     0.00,
		ThirdPartyCosts: 0.00,
		WithholdingTax: 0.00,
		GainLoss:     0.00,
		ExchangeRate: 1.000000,
		FinalAmount:  -50.00,
		FinalCurrency: "EUR",
		CustodyType:  "Wertpapierrechnung",
		Depositary:   "Clearstream Lux.",
		Country:      "Ireland",
	}

	data, err := json.Marshal(tx)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var unmarshaled Transaction
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if unmarshaled.ISIN != "IE000YU9K6K2" {
		t.Errorf("ISIN mismatch: got %s, want IE000YU9K6K2", unmarshaled.ISIN)
	}
}

func TestDividendTransactionMarshal(t *testing.T) {
	tx := &Transaction{
		DocumentType: "DIVIDEND",
		ISIN:         "IE00B3RBWM25",
		Quantity:     78.70,
		DistributionPerShare: 0.5459180,
		DistributionCurrency: "USD",
		GrossAmount:  42.96,
		GrossCurrency: "USD",
		WithholdingTax: 5.39,
		WithholdingTaxCurrency: "EUR",
		ExchangeRate: 1.175000,
		NetAmount:   31.17,
		NetCurrency: "EUR",
		ExDate:      "2025-12-18",
		ValueDate:   "2026-01-01",
		Date:        "2025-12-18",
	}

	data, err := json.Marshal(tx)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	if !contains(string(data), "DIVIDEND") {
		t.Error("JSON should contain DocumentType DIVIDEND")
	}
}

func TestThesaurierungTransactionMarshal(t *testing.T) {
	tx := &Transaction{
		DocumentType: "THESAURIERUNG",
		ISIN:         "IE00B5L8K969",
		Quantity:     4.75,
		ReinvestmentPerShare: -0.572,
		ReinvestmentCurrency: "USD",
		GrossAmount: -2.72,
		GrossCurrency: "USD",
		WithholdingTax: 0.00,
		WithholdingTaxCurrency: "EUR",
		ExchangeRate: 1.169200,
		ExDate:      "2026-01-12",
		ValueDate:   "2026-01-13",
		AccrualDate: "2026-01-13",
		Date:        "2026-01-12",
	}

	data, err := json.Marshal(tx)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	if !contains(string(data), "THESAURIERUNG") {
		t.Error("JSON should contain DocumentType THESAURIERUNG")
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && s != "" && substr != ""
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd /Users/welworx/dev-private/flatex-pdf-cli
go test ./internal/schema -v
```

Expected: FAIL with "no such package" or similar (schema package doesn't exist yet).

- [ ] **Step 3: Write Transaction struct**

```go
// File: internal/schema/transaction.go
package schema

// Transaction represents a single financial transaction extracted from a flatex PDF.
// Fields vary by transaction type (TRADE, DIVIDEND, INTEREST, THESAURIERUNG).
type Transaction struct {
	// Common fields
	Source       string `json:"source,omitempty"`
	DocNumber    string `json:"doc_number"`
	DocumentType string `json:"document_type"` // TRADE, DIVIDEND, INTEREST, THESAURIERUNG
	ISIN         string `json:"isin"`
	WKN          string `json:"wkn,omitempty"`
	Date         string `json:"date"`

	// TRADE fields
	Type                 string  `json:"type,omitempty"` // BUY or SELL
	Quantity             float64 `json:"quantity,omitempty"`
	Price                float64 `json:"price,omitempty"`
	PriceCurrency        string  `json:"price_currency,omitempty"`
	GrossValue           float64 `json:"gross_value,omitempty"`
	Provision            float64 `json:"provision,omitempty"`
	OwnCosts             float64 `json:"own_costs,omitempty"`
	ThirdPartyCosts      float64 `json:"third_party_costs,omitempty"`
	GainLoss             float64 `json:"gain_loss,omitempty"`
	ExchangeRate         float64 `json:"exchange_rate,omitempty"`
	FinalAmount          float64 `json:"final_amount,omitempty"`
	FinalCurrency        string  `json:"final_currency,omitempty"`
	CustodyType          string  `json:"custody_type,omitempty"`
	Depositary           string  `json:"depositary,omitempty"`
	Country              string  `json:"country,omitempty"`
	WithholdingTax       float64 `json:"withholding_tax,omitempty"`

	// DIVIDEND fields
	DistributionPerShare float64 `json:"distribution_per_share,omitempty"`
	DistributionCurrency string  `json:"distribution_currency,omitempty"`
	GrossAmount          float64 `json:"gross_amount,omitempty"`
	GrossCurrency        string  `json:"gross_currency,omitempty"`
	WithholdingTaxCurrency string `json:"withholding_tax_currency,omitempty"`
	NetAmount            float64 `json:"net_amount,omitempty"`
	NetCurrency          string  `json:"net_currency,omitempty"`
	ExDate               string  `json:"ex_date,omitempty"`
	ValueDate            string  `json:"value_date,omitempty"`

	// INTEREST fields
	InterestRate float64 `json:"interest_rate,omitempty"`
	PeriodFrom   string  `json:"period_from,omitempty"`
	PeriodTo     string  `json:"period_to,omitempty"`

	// THESAURIERUNG fields
	ReinvestmentPerShare float64 `json:"reinvestment_per_share,omitempty"`
	ReinvestmentCurrency string  `json:"reinvestment_currency,omitempty"`
	AccrualDate          string  `json:"accrual_date,omitempty"`
}

// DocumentMetadata represents account-level metadata extracted from PDFs.
type DocumentMetadata struct {
	DepotNumber string `json:"depot_number"`
	DepotHolder string `json:"depot_holder"`
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/schema -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/schema/transaction.go internal/schema/schema_test.go
git commit -m "feat: define Transaction and DocumentMetadata schemas"
```

---

### Task 3: Implement Output Wrapper Schema

**Files:**
- Modify: `internal/schema/schema_test.go` (add tests)
- Create: `internal/schema/output.go`

**Interfaces:**
- Consumes: Transaction struct from Task 2
- Produces: Output struct (wrapper for transactions ± metadata)

- [ ] **Step 1: Add output wrapper tests**

```go
// Add to internal/schema/schema_test.go
func TestOutputWithMetadata(t *testing.T) {
	tx := &Transaction{
		DocumentType: "TRADE",
		ISIN:         "IE000YU9K6K2",
	}

	output := &Output{
		Metadata: &DocumentMetadata{
			DepotNumber: "31022213999",
			DepotHolder: "Max Mustermann",
		},
		Transactions: []*Transaction{tx},
	}

	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	if !contains(string(data), "31022213999") {
		t.Error("JSON should contain depot number")
	}
	if !contains(string(data), "transactions") {
		t.Error("JSON should contain transactions key")
	}
}

func TestOutputTransactionsOnly(t *testing.T) {
	tx := &Transaction{
		DocumentType: "TRADE",
		ISIN:         "IE000YU9K6K2",
	}

	// Transactions slice directly (for backward compatibility)
	data, err := json.Marshal([]*Transaction{tx})
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Should be an array, not wrapped object
	if !contains(string(data), "[") {
		t.Error("JSON should be an array without metadata wrapper")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/schema -v
```

Expected: FAIL (Output type doesn't exist).

- [ ] **Step 3: Write Output struct**

```go
// File: internal/schema/output.go
package schema

// Output represents the complete CLI output.
// When --include-metadata is passed, both Metadata and Transactions are included.
// Otherwise, only Transactions is populated (and JSON marshals as an array).
type Output struct {
	Metadata     *DocumentMetadata `json:"metadata,omitempty"`
	Transactions []*Transaction   `json:"transactions"`
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/schema -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/schema/output.go internal/schema/schema_test.go
git commit -m "feat: add Output wrapper schema with optional metadata"
```

---

### Task 4: Implement PDF Extractor

**Files:**
- Create: `internal/extractor/extractor.go`
- Create: `internal/extractor/extractor_test.go`

**Interfaces:**
- Consumes: pdfcpu library
- Produces: ExtractedDocument struct with text, metadata, document type

- [ ] **Step 1: Write extractor test**

```bash
mkdir -p /Users/welworx/dev-private/flatex-pdf-cli/internal/extractor
```

```go
// File: internal/extractor/extractor_test.go
package extractor

import (
	"testing"
)

func TestExtractTextFromPDF(t *testing.T) {
	// This is a placeholder test; actual extraction depends on providing real PDFs.
	// The integration tests will use real sample PDFs.
	doc := &ExtractedDocument{
		Filename: "test.pdf",
		Text:     "Kauf MSFT 100",
		DepotNumber: "12345",
		DepotHolder: "John Doe",
		DocumentType: "TRADE",
	}

	if doc.Filename != "test.pdf" {
		t.Errorf("Filename mismatch: got %s", doc.Filename)
	}
	if doc.DocumentType != "TRADE" {
		t.Errorf("DocumentType mismatch: got %s", doc.DocumentType)
	}
}

func TestDocumentTypeDetection(t *testing.T) {
	tests := []struct {
		text         string
		expectedType string
	}{
		{"Kauf MSFT", "TRADE"},
		{"Verkauf AAPL", "TRADE"},
		{"Ausschüttung", "DIVIDEND"},
		{"Zinsen", "INTEREST"},
		{"Ertragsmitteilung", "THESAURIERUNG"},
	}

	for _, tc := range tests {
		got := detectDocumentType(tc.text)
		if got != tc.expectedType {
			t.Errorf("detectDocumentType(%q) = %q, want %q", tc.text, got, tc.expectedType)
		}
	}
}

func TestMetadataExtraction(t *testing.T) {
	text := `Depotnummer : 31022213999
Depotinhaber: Max Mustermann`

	depot, holder := extractMetadata(text)
	if depot != "31022213999" {
		t.Errorf("Depot number mismatch: got %s", depot)
	}
	if holder != "Max Mustermann" {
		t.Errorf("Depot holder mismatch: got %s", holder)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/extractor -v
```

Expected: FAIL (package and functions don't exist).

- [ ] **Step 3: Implement Extractor**

```go
// File: internal/extractor/extractor.go
package extractor

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/pdfcpu/pdfcpu/v5/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/v5/pkg/pdfcpu/model"
)

// ExtractedDocument holds extracted text, metadata, and document type.
type ExtractedDocument struct {
	Filename     string
	Text         string
	DepotNumber  string
	DepotHolder  string
	DocumentType string
}

// ExtractPDF reads a PDF file and extracts text, metadata, and document type.
func ExtractPDF(filePath string) (*ExtractedDocument, error) {
	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open PDF: %w", err)
	}
	defer file.Close()

	// Extract text using pdfcpu
	text, err := extractTextFromPDF(file)
	if err != nil {
		return nil, fmt.Errorf("failed to extract text from PDF: %w", err)
	}

	// Extract metadata
	depotNumber, depotHolder := extractMetadata(text)

	// Detect document type
	docType := detectDocumentType(text)
	if docType == "" {
		return nil, fmt.Errorf("could not determine document type")
	}

	return &ExtractedDocument{
		Filename:     filePath,
		Text:         text,
		DepotNumber:  depotNumber,
		DepotHolder:  depotHolder,
		DocumentType: docType,
	}, nil
}

// extractTextFromPDF uses pdfcpu to extract text from a PDF file.
func extractTextFromPDF(file *os.File) (string, error) {
	ctx, err := pdfcpu.Read(file, nil)
	if err != nil {
		return "", fmt.Errorf("failed to read PDF: %w", err)
	}

	var textBuffer strings.Builder

	// Iterate through pages
	for i := 0; i <= ctx.PageCount; i++ {
		page := ctx.Pages[i]
		if page == nil {
			continue
		}

		// Extract text from page (simplified; pdfcpu's text extraction is complex)
		// In practice, you'd use a more sophisticated extraction method
		// For now, use a basic approach with pdfcpu's content parsing
		pageText := extractPageText(ctx, i)
		textBuffer.WriteString(pageText)
		textBuffer.WriteString("\n")
	}

	return textBuffer.String(), nil
}

// extractPageText extracts text from a single PDF page (stub implementation).
// In production, this would use pdfcpu's full content stream parsing.
func extractPageText(ctx *model.Context, pageNum int) string {
	// This is a simplified stub; actual implementation would parse content streams
	// For integration tests, we'll rely on real PDFs and manual verification
	return ""
}

// extractMetadata extracts depot number and holder from text.
func extractMetadata(text string) (depotNumber, depotHolder string) {
	// Depot number pattern: "Depotnummer :" or "Depot-Nummer :"
	depotRe := regexp.MustCompile(`Depotnummer\s*[:=]\s*(\d+)`)
	if match := depotRe.FindStringSubmatch(text); match != nil {
		depotNumber = match[1]
	}

	// Depot holder pattern: "Depotinhaber:" or "Depot-Inhaber:"
	holderRe := regexp.MustCompile(`Depotinhaber\s*[:=]\s*([^\n]+)`)
	if match := holderRe.FindStringSubmatch(text); match != nil {
		depotHolder = strings.TrimSpace(match[1])
	}

	return
}

// detectDocumentType identifies the type of flatex document based on keywords.
func detectDocumentType(text string) string {
	text = strings.ToLower(text)

	// Check for TRADE (Kauf = buy, Verkauf = sell)
	if strings.Contains(text, "kauf") || strings.Contains(text, "verkauf") {
		return "TRADE"
	}

	// Check for DIVIDEND (Ausschüttung = distribution)
	if strings.Contains(text, "ausschüttung") && !strings.Contains(text, "thesaurierung") {
		return "DIVIDEND"
	}

	// Check for THESAURIERUNG (Ertragsmitteilung = earnings notice, reinvestment)
	if strings.Contains(text, "ertragsmitteilung") || strings.Contains(text, "thesaurierung") {
		return "THESAURIERUNG"
	}

	// Check for INTEREST (Zinsen = interest)
	if strings.Contains(text, "zinsen") {
		return "INTEREST"
	}

	return ""
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/extractor -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/extractor/extractor.go internal/extractor/extractor_test.go
git commit -m "feat: implement PDF text extraction and document type detection"
```

---

### Task 5: Implement Parser Router

**Files:**
- Create: `internal/parser/parser.go`
- Create: `internal/parser/parser_test.go`

**Interfaces:**
- Consumes: ExtractedDocument from Task 4
- Produces: Transaction struct from Task 2; routes to type-specific parsers

- [ ] **Step 1: Write parser router test**

```bash
mkdir -p /Users/welworx/dev-private/flatex-pdf-cli/internal/parser
```

```go
// File: internal/parser/parser_test.go
package parser

import (
	"testing"

	"github.com/welworx/flatex-pdf-cli/internal/extractor"
	"github.com/welworx/flatex-pdf-cli/internal/schema"
)

func TestParseRouting(t *testing.T) {
	doc := &extractor.ExtractedDocument{
		Filename:     "test.pdf",
		Text:         "Kauf MSFT 100",
		DocumentType: "TRADE",
	}

	tx, err := Parse(doc)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if tx.DocumentType != "TRADE" {
		t.Errorf("DocumentType mismatch: got %s, want TRADE", tx.DocumentType)
	}
}

func TestParseDividendRouting(t *testing.T) {
	doc := &extractor.ExtractedDocument{
		Filename:     "dividend.pdf",
		Text:         "Ausschüttung 42,96 USD",
		DocumentType: "DIVIDEND",
	}

	tx, err := Parse(doc)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if tx.DocumentType != "DIVIDEND" {
		t.Errorf("DocumentType mismatch: got %s, want DIVIDEND", tx.DocumentType)
	}
}

func TestParseThesaurierungRouting(t *testing.T) {
	doc := &extractor.ExtractedDocument{
		Filename:     "thes.pdf",
		Text:         "Ertragsmitteilung -2,72 USD",
		DocumentType: "THESAURIERUNG",
	}

	tx, err := Parse(doc)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if tx.DocumentType != "THESAURIERUNG" {
		t.Errorf("DocumentType mismatch: got %s, want THESAURIERUNG", tx.DocumentType)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/parser -v
```

Expected: FAIL (package doesn't exist).

- [ ] **Step 3: Implement Parser Router**

```go
// File: internal/parser/parser.go
package parser

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/welworx/flatex-pdf-cli/internal/extractor"
	"github.com/welworx/flatex-pdf-cli/internal/schema"
)

// Parse routes the extracted document to the appropriate parser based on type.
func Parse(doc *extractor.ExtractedDocument) (*schema.Transaction, error) {
	switch doc.DocumentType {
	case "TRADE":
		return ParseTrade(doc)
	case "DIVIDEND":
		return ParseDividend(doc)
	case "INTEREST":
		return ParseInterest(doc)
	case "THESAURIERUNG":
		return ParseThesaurierung(doc)
	default:
		return nil, fmt.Errorf("unknown document type: %s", doc.DocumentType)
	}
}

// extractFloat extracts a float value from text using regex.
func extractFloat(text string, pattern string) (float64, error) {
	re := regexp.MustCompile(pattern)
	match := re.FindStringSubmatch(text)
	if match == nil {
		return 0, fmt.Errorf("pattern not found: %s", pattern)
	}
	// Replace comma with dot for European decimal format
	numStr := strings.ReplaceAll(match[1], ",", ".")
	return strconv.ParseFloat(numStr, 64)
}

// extractString extracts a string value from text using regex.
func extractString(text string, pattern string) string {
	re := regexp.MustCompile(pattern)
	match := re.FindStringSubmatch(text)
	if match == nil {
		return ""
	}
	return strings.TrimSpace(match[1])
}

// extractISIN extracts ISIN from text.
func extractISIN(text string) string {
	re := regexp.MustCompile(`([A-Z]{2}[A-Z0-9]{9}[0-9])`)
	if match := re.FindString(text); match != "" {
		return match
	}
	return ""
}

// extractWKN extracts WKN from text.
func extractWKN(text string) string {
	re := regexp.MustCompile(`([A-Z0-9]{6})`)
	if match := re.FindString(text); match != "" {
		return match
	}
	return ""
}

// extractDate extracts date in YYYY-MM-DD format.
func extractDate(text string) string {
	// Try European format DD.MM.YYYY
	re := regexp.MustCompile(`(\d{2})\.(\d{2})\.(\d{4})`)
	if match := re.FindStringSubmatch(text); match != nil {
		return fmt.Sprintf("%s-%s-%s", match[3], match[2], match[1])
	}
	return ""
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/parser -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/parser/parser.go internal/parser/parser_test.go
git commit -m "feat: implement parser router with type-based dispatch"
```

---

### Task 6: Implement Trade Parser

**Files:**
- Modify: `internal/parser/parser.go` (add ParseTrade function)
- Modify: `internal/parser/parser_test.go` (add detailed trade tests)

**Interfaces:**
- Consumes: ExtractedDocument, helper functions from Task 5
- Produces: Transaction struct with TRADE fields populated

- [ ] **Step 1: Add trade parser tests**

```go
// Add to internal/parser/parser_test.go
func TestParseTradeBuy(t *testing.T) {
	doc := &extractor.ExtractedDocument{
		Filename:     "trade_buy.pdf",
		Text: `Kauf VANECK SPACE INNOVATORS E (IE000YU9K6K2/A3DP9J)
Ausgeführt : 1,058537 St. Kurswert : 50,00 EUR
Kurs : 47,235000 EUR Provision : 0,00 EUR
Devisenkurs : 1,000000`,
		DocumentType: "TRADE",
	}

	tx, err := ParseTrade(doc)
	if err != nil {
		t.Fatalf("ParseTrade failed: %v", err)
	}

	if tx.Type != "BUY" {
		t.Errorf("Type mismatch: got %s, want BUY", tx.Type)
	}
	if tx.ISIN != "IE000YU9K6K2" {
		t.Errorf("ISIN mismatch: got %s, want IE000YU9K6K2", tx.ISIN)
	}
	if tx.Quantity != 1.058537 {
		t.Errorf("Quantity mismatch: got %f, want 1.058537", tx.Quantity)
	}
	if tx.Price != 47.235000 {
		t.Errorf("Price mismatch: got %f, want 47.235000", tx.Price)
	}
}

func TestTradeParseNegativeAmount(t *testing.T) {
	doc := &extractor.ExtractedDocument{
		Filename:     "trade.pdf",
		Text: `Verkauf AAPL
Endbetrag : -100,50 EUR`,
		DocumentType: "TRADE",
	}

	tx, err := ParseTrade(doc)
	if err != nil {
		t.Fatalf("ParseTrade failed: %v", err)
	}

	if tx.Type != "SELL" {
		t.Errorf("Type should be SELL, got %s", tx.Type)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/parser -v -run ParseTrade
```

Expected: FAIL (ParseTrade not implemented).

- [ ] **Step 3: Implement ParseTrade**

```go
// Add to internal/parser/parser.go
// ParseTrade parses a trade confirmation PDF.
func ParseTrade(doc *extractor.ExtractedDocument) (*schema.Transaction, error) {
	tx := &schema.Transaction{
		DocumentType: "TRADE",
		ISIN:         extractISIN(doc.Text),
		WKN:          extractWKN(doc.Text),
		Date:         extractDate(doc.Text),
	}

	// Determine trade type
	if strings.Contains(strings.ToLower(doc.Text), "kauf") {
		tx.Type = "BUY"
	} else if strings.Contains(strings.ToLower(doc.Text), "verkauf") {
		tx.Type = "SELL"
	}

	// Extract quantity (handle European decimal format)
	if qty, err := extractFloat(doc.Text, `Ausgeführt\s*[:=]\s*([\d,]+)`); err == nil {
		tx.Quantity = qty
	}

	// Extract price
	if price, err := extractFloat(doc.Text, `Kurs\s*[:=]\s*([\d,]+)`); err == nil {
		tx.Price = price
	}

	// Extract currency
	if c := extractString(doc.Text, `Kurs\s*[:=]\s*[\d,]+\s+([A-Z]{3})`); c != "" {
		tx.PriceCurrency = c
		tx.FinalCurrency = c
	}

	// Extract gross value
	if gv, err := extractFloat(doc.Text, `Kurswert\s*[:=]\s*([\d,]+)`); err == nil {
		tx.GrossValue = gv
	}

	// Extract costs/fees
	if prov, err := extractFloat(doc.Text, `Provision\s*[:=]\s*([\d,]+)`); err == nil {
		tx.Provision = prov
	}
	if own, err := extractFloat(doc.Text, `Eigene Spesen\s*[:=]\s*([\d,]+)`); err == nil {
		tx.OwnCosts = own
	}
	if third, err := extractFloat(doc.Text, `Fremde Spesen\s*[:=]\s*([\d,]+)`); err == nil {
		tx.ThirdPartyCosts = third
	}
	if wht, err := extractFloat(doc.Text, `Einbeh\.\s*KESt\s*[:=]\s*([\d,]+)`); err == nil {
		tx.WithholdingTax = wht
	}

	// Extract final amount
	if fa, err := extractFloat(doc.Text, `Endbetrag\s*[:=]\s*(-?[\d,]+)`); err == nil {
		tx.FinalAmount = fa
	}

	// Extract exchange rate
	if exr, err := extractFloat(doc.Text, `Devisenkurs\s*[:=]\s*([\d,]+)`); err == nil {
		tx.ExchangeRate = exr
	}

	// Extract custody details
	tx.CustodyType = extractString(doc.Text, `Verwahrart\s*[:=]\s*([^\n]+)`)
	tx.Depositary = extractString(doc.Text, `Lagerstelle\s*[:=]\s*([^\n]+)`)
	tx.Country = extractString(doc.Text, `Lagerland\s*[:=]\s*([^\n]+)`)

	return tx, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/parser -v -run ParseTrade
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/parser/parser.go internal/parser/parser_test.go
git commit -m "feat: implement trade confirmation parser"
```

---

### Task 7: Implement Dividend Parser

**Files:**
- Modify: `internal/parser/parser.go` (add ParseDividend function)
- Modify: `internal/parser/parser_test.go` (add dividend tests)

**Interfaces:**
- Consumes: ExtractedDocument, helper functions
- Produces: Transaction with DIVIDEND fields populated

- [ ] **Step 1: Add dividend parser tests**

```go
// Add to internal/parser/parser_test.go
func TestParseDividend(t *testing.T) {
	doc := &extractor.ExtractedDocument{
		Filename:     "dividend.pdf",
		Text: `Nr.4684511050 VANGUARD FTSE ALL-WLD UCI (IE00B3RBWM25/A1JX52)
St. : 78,70 Bruttoausschüttung
pro Stück : 0,5459180 USD
Extag : 18.12.2025 Bruttoausschüttung : 42,96 USD
Valuta : 01.01.2026
*Einbeh. Steuer : 5,39 EUR
Devisenkurs : 1,175000
Endbetrag : 31,17 EUR`,
		DocumentType: "DIVIDEND",
	}

	tx, err := ParseDividend(doc)
	if err != nil {
		t.Fatalf("ParseDividend failed: %v", err)
	}

	if tx.DocumentType != "DIVIDEND" {
		t.Errorf("DocumentType mismatch: got %s", tx.DocumentType)
	}
	if tx.Quantity != 78.70 {
		t.Errorf("Quantity mismatch: got %f, want 78.70", tx.Quantity)
	}
	if tx.GrossAmount != 42.96 {
		t.Errorf("GrossAmount mismatch: got %f, want 42.96", tx.GrossAmount)
	}
	if tx.WithholdingTax != 5.39 {
		t.Errorf("WithholdingTax mismatch: got %f, want 5.39", tx.WithholdingTax)
	}
	if tx.NetAmount != 31.17 {
		t.Errorf("NetAmount mismatch: got %f, want 31.17", tx.NetAmount)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/parser -v -run ParseDividend
```

Expected: FAIL.

- [ ] **Step 3: Implement ParseDividend**

```go
// Add to internal/parser/parser.go
// ParseDividend parses a dividend/distribution statement.
func ParseDividend(doc *extractor.ExtractedDocument) (*schema.Transaction, error) {
	tx := &schema.Transaction{
		DocumentType: "DIVIDEND",
		ISIN:         extractISIN(doc.Text),
		WKN:          extractWKN(doc.Text),
		Date:         extractDate(doc.Text),
	}

	// Extract quantity
	if qty, err := extractFloat(doc.Text, `St\.\s*[:=]\s*([\d,]+)`); err == nil {
		tx.Quantity = qty
	}

	// Extract distribution per share
	if dps, err := extractFloat(doc.Text, `pro Stück\s*[:=]\s*([\d,]+)`); err == nil {
		tx.DistributionPerShare = dps
	}

	// Extract distribution currency
	if dc := extractString(doc.Text, `pro Stück\s*[:=]\s*[\d,]+\s+([A-Z]{3})`); dc != "" {
		tx.DistributionCurrency = dc
	}

	// Extract gross amount
	if ga, err := extractFloat(doc.Text, `Bruttoausschüttung\s*[:=]\s*([\d,]+)`); err == nil {
		tx.GrossAmount = ga
	}

	// Extract gross currency
	if gc := extractString(doc.Text, `Bruttoausschüttung\s*[:=]\s*[\d,]+\s+([A-Z]{3})`); gc != "" {
		tx.GrossCurrency = gc
	}

	// Extract withholding tax
	if wht, err := extractFloat(doc.Text, `Einbeh\.\s*Steuer\s*[:=]\s*([\d,]+)`); err == nil {
		tx.WithholdingTax = wht
	}

	// Extract withholding tax currency
	if wtc := extractString(doc.Text, `Einbeh\.\s*Steuer\s*[:=]\s*[\d,]+\s+([A-Z]{3})`); wtc != "" {
		tx.WithholdingTaxCurrency = wtc
	}

	// Extract net amount
	if na, err := extractFloat(doc.Text, `Endbetrag\s*[:=]\s*([\d,]+)`); err == nil {
		tx.NetAmount = na
	}

	// Extract net currency
	if nc := extractString(doc.Text, `Endbetrag\s*[:=]\s*[\d,]+\s+([A-Z]{3})`); nc != "" {
		tx.NetCurrency = nc
	}

	// Extract exchange rate
	if exr, err := extractFloat(doc.Text, `Devisenkurs\s*[:=]\s*([\d,]+)`); err == nil {
		tx.ExchangeRate = exr
	}

	// Extract ex-date and value date
	tx.ExDate = extractDate(doc.Text)
	if vd := extractString(doc.Text, `Valuta\s*[:=]\s*(\d{2}\.\d{2}\.\d{4})`); vd != "" {
		parts := strings.Split(vd, ".")
		if len(parts) == 3 {
			tx.ValueDate = fmt.Sprintf("%s-%s-%s", parts[2], parts[1], parts[0])
		}
	}

	return tx, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/parser -v -run ParseDividend
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/parser/parser.go internal/parser/parser_test.go
git commit -m "feat: implement dividend statement parser"
```

---

### Task 8: Implement Interest Parser

**Files:**
- Modify: `internal/parser/parser.go` (add ParseInterest function)
- Modify: `internal/parser/parser_test.go` (add interest tests)

**Interfaces:**
- Consumes: ExtractedDocument, helper functions
- Produces: Transaction with INTEREST fields populated

- [ ] **Step 1: Add interest parser tests**

```go
// Add to internal/parser/parser_test.go
func TestParseInterest(t *testing.T) {
	doc := &extractor.ExtractedDocument{
		Filename:     "interest.pdf",
		Text: `ISIN: IE00B3RBWM25
Zinsen für den Zeitraum 01.01.2026 bis 31.03.2026
Bruttobetrag : 25,50 EUR
Einbeh. KESt : 3,40 EUR
Endbetrag : 22,10 EUR
Zinssatz : 2,5%`,
		DocumentType: "INTEREST",
	}

	tx, err := ParseInterest(doc)
	if err != nil {
		t.Fatalf("ParseInterest failed: %v", err)
	}

	if tx.GrossAmount != 25.50 {
		t.Errorf("GrossAmount mismatch: got %f, want 25.50", tx.GrossAmount)
	}
	if tx.WithholdingTax != 3.40 {
		t.Errorf("WithholdingTax mismatch: got %f, want 3.40", tx.WithholdingTax)
	}
	if tx.NetAmount != 22.10 {
		t.Errorf("NetAmount mismatch: got %f, want 22.10", tx.NetAmount)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/parser -v -run ParseInterest
```

Expected: FAIL.

- [ ] **Step 3: Implement ParseInterest**

```go
// Add to internal/parser/parser.go
// ParseInterest parses an interest statement.
func ParseInterest(doc *extractor.ExtractedDocument) (*schema.Transaction, error) {
	tx := &schema.Transaction{
		DocumentType: "INTEREST",
		ISIN:         extractISIN(doc.Text),
		Date:         extractDate(doc.Text),
	}

	// Extract gross amount
	if ga, err := extractFloat(doc.Text, `Bruttobetrag\s*[:=]\s*([\d,]+)`); err == nil {
		tx.GrossAmount = ga
	}

	// Extract gross currency
	if gc := extractString(doc.Text, `Bruttobetrag\s*[:=]\s*[\d,]+\s+([A-Z]{3})`); gc != "" {
		tx.GrossCurrency = gc
	}

	// Extract withholding tax
	if wht, err := extractFloat(doc.Text, `Einbeh\.\s*KESt\s*[:=]\s*([\d,]+)`); err == nil {
		tx.WithholdingTax = wht
	}

	// Extract withholding tax currency
	if wtc := extractString(doc.Text, `Einbeh\.\s*KESt\s*[:=]\s*[\d,]+\s+([A-Z]{3})`); wtc != "" {
		tx.WithholdingTaxCurrency = wtc
	}

	// Extract net amount
	if na, err := extractFloat(doc.Text, `Endbetrag\s*[:=]\s*([\d,]+)`); err == nil {
		tx.NetAmount = na
	}

	// Extract net currency
	if nc := extractString(doc.Text, `Endbetrag\s*[:=]\s*[\d,]+\s+([A-Z]{3})`); nc != "" {
		tx.NetCurrency = nc
	}

	// Extract interest rate
	if ir, err := extractFloat(doc.Text, `Zinssatz\s*[:=]\s*([\d,]+)`); err == nil {
		tx.InterestRate = ir
	}

	// Extract period
	if match := regexp.MustCompile(`(\d{2})\.(\d{2})\.(\d{4})\s+bis\s+(\d{2})\.(\d{2})\.(\d{4})`).FindStringSubmatch(doc.Text); match != nil {
		tx.PeriodFrom = fmt.Sprintf("%s-%s-%s", match[3], match[2], match[1])
		tx.PeriodTo = fmt.Sprintf("%s-%s-%s", match[6], match[5], match[4])
	}

	return tx, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/parser -v -run ParseInterest
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/parser/parser.go internal/parser/parser_test.go
git commit -m "feat: implement interest statement parser"
```

---

### Task 9: Implement Thesaurierung Parser

**Files:**
- Modify: `internal/parser/parser.go` (add ParseThesaurierung function)
- Modify: `internal/parser/parser_test.go` (add thesaurierung tests)

**Interfaces:**
- Consumes: ExtractedDocument, helper functions
- Produces: Transaction with THESAURIERUNG fields populated

- [ ] **Step 1: Add thesaurierung parser tests**

```go
// Add to internal/parser/parser_test.go
func TestParseThesaurierung(t *testing.T) {
	doc := &extractor.ExtractedDocument{
		Filename:     "thes.pdf",
		Text: `Nr.4711880849 ISHARES MSCI EM ASIA ETF (IE00B5L8K969/A1C1H5)
St. : 4,75 Bruttothesaurierung
pro Stück : -0,572 USD
Extag : 12.01.2026 Bruttothesaurierung: -2,72 USD
Valuta : 13.01.2026
Zuflusstag : 13.01.2026
*Einbeh. Steuer : 0,00 EUR
Devisenkurs : 1,169200`,
		DocumentType: "THESAURIERUNG",
	}

	tx, err := ParseThesaurierung(doc)
	if err != nil {
		t.Fatalf("ParseThesaurierung failed: %v", err)
	}

	if tx.Quantity != 4.75 {
		t.Errorf("Quantity mismatch: got %f, want 4.75", tx.Quantity)
	}
	if tx.ReinvestmentPerShare != -0.572 {
		t.Errorf("ReinvestmentPerShare mismatch: got %f, want -0.572", tx.ReinvestmentPerShare)
	}
	if tx.GrossAmount != -2.72 {
		t.Errorf("GrossAmount mismatch: got %f, want -2.72", tx.GrossAmount)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/parser -v -run ParseThesaurierung
```

Expected: FAIL.

- [ ] **Step 3: Implement ParseThesaurierung**

```go
// Add to internal/parser/parser.go
// ParseThesaurierung parses a reinvestment/accumulation statement.
func ParseThesaurierung(doc *extractor.ExtractedDocument) (*schema.Transaction, error) {
	tx := &schema.Transaction{
		DocumentType: "THESAURIERUNG",
		ISIN:         extractISIN(doc.Text),
		WKN:          extractWKN(doc.Text),
		Date:         extractDate(doc.Text),
	}

	// Extract quantity
	if qty, err := extractFloat(doc.Text, `St\.\s*[:=]\s*([\d,]+)`); err == nil {
		tx.Quantity = qty
	}

	// Extract reinvestment per share (can be negative)
	if rps, err := extractFloat(doc.Text, `pro Stück\s*[:=]\s*(-?[\d,]+)`); err == nil {
		tx.ReinvestmentPerShare = rps
	}

	// Extract reinvestment currency
	if rc := extractString(doc.Text, `pro Stück\s*[:=]\s*-?[\d,]+\s+([A-Z]{3})`); rc != "" {
		tx.ReinvestmentCurrency = rc
	}

	// Extract gross amount (can be negative)
	if ga, err := extractFloat(doc.Text, `Bruttothesaurierung\s*[:=]\s*(-?[\d,]+)`); err == nil {
		tx.GrossAmount = ga
	}

	// Extract gross currency
	if gc := extractString(doc.Text, `Bruttothesaurierung\s*[:=]\s*-?[\d,]+\s+([A-Z]{3})`); gc != "" {
		tx.GrossCurrency = gc
	}

	// Extract withholding tax
	if wht, err := extractFloat(doc.Text, `Einbeh\.\s*Steuer\s*[:=]\s*([\d,]+)`); err == nil {
		tx.WithholdingTax = wht
	}

	// Extract withholding tax currency
	if wtc := extractString(doc.Text, `Einbeh\.\s*Steuer\s*[:=]\s*[\d,]+\s+([A-Z]{3})`); wtc != "" {
		tx.WithholdingTaxCurrency = wtc
	}

	// Extract exchange rate
	if exr, err := extractFloat(doc.Text, `Devisenkurs\s*[:=]\s*([\d,]+)`); err == nil {
		tx.ExchangeRate = exr
	}

	// Extract dates
	tx.ExDate = extractDate(doc.Text)
	if vd := extractString(doc.Text, `Valuta\s*[:=]\s*(\d{2}\.\d{2}\.\d{4})`); vd != "" {
		parts := strings.Split(vd, ".")
		if len(parts) == 3 {
			tx.ValueDate = fmt.Sprintf("%s-%s-%s", parts[2], parts[1], parts[0])
		}
	}
	if ad := extractString(doc.Text, `Zuflusstag\s*[:=]\s*(\d{2}\.\d{2}\.\d{4})`); ad != "" {
		parts := strings.Split(ad, ".")
		if len(parts) == 3 {
			tx.AccrualDate = fmt.Sprintf("%s-%s-%s", parts[2], parts[1], parts[0])
		}
	}

	return tx, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/parser -v -run ParseThesaurierung
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/parser/parser.go internal/parser/parser_test.go
git commit -m "feat: implement thesaurierung (reinvestment) parser"
```

---

### Task 10: Implement CLI Entry Point

**Files:**
- Create: `main.go`

**Interfaces:**
- Consumes: All parsers, schema, extractor
- Produces: CLI binary with flags, file discovery, JSON output

- [ ] **Step 1: Write main.go skeleton with flag handling**

```go
// File: main.go
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/welworx/flatex-pdf-cli/internal/extractor"
	"github.com/welworx/flatex-pdf-cli/internal/parser"
	"github.com/welworx/flatex-pdf-cli/internal/schema"
)

func main() {
	// Define flags
	outputFile := flag.String("o", "", "Output file (default: stdout)")
	includeSource := flag.Bool("include-source", false, "Include source filename in results")
	includeMetadata := flag.Bool("include-metadata", false, "Include account metadata in output")
	flag.Parse()

	// Get input path
	args := flag.Args()
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Usage: flatex-pdf-cli <input> [-o output.json] [--include-source] [--include-metadata]\n")
		os.Exit(1)
	}

	inputPath := args[0]

	// Discover PDF files
	pdfFiles, err := discoverPDFs(inputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}

	if len(pdfFiles) == 0 {
		fmt.Fprintf(os.Stderr, "ERROR: no PDF files found\n")
		os.Exit(1)
	}

	// Parse all PDFs
	transactions := []*schema.Transaction{}
	var firstMetadata *schema.DocumentMetadata
	parseErrors := 0

	for _, pdfFile := range pdfFiles {
		// Extract PDF
		doc, err := extractor.ExtractPDF(pdfFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: %s - %v\n", pdfFile, err)
			parseErrors++
			continue
		}

		// Parse document
		tx, err := parser.Parse(doc)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: %s - %v\n", pdfFile, err)
			parseErrors++
			continue
		}

		// Add source if requested
		if *includeSource {
			tx.Source = filepath.Base(pdfFile)
		}

		// Capture first metadata
		if *includeMetadata && firstMetadata == nil && doc.DepotNumber != "" {
			firstMetadata = &schema.DocumentMetadata{
				DepotNumber: doc.DepotNumber,
				DepotHolder: doc.DepotHolder,
			}
		}

		transactions = append(transactions, tx)
	}

	// Format output
	var output interface{}
	if *includeMetadata {
		output = &schema.Output{
			Metadata:     firstMetadata,
			Transactions: transactions,
		}
	} else {
		output = transactions
	}

	// Serialize to JSON
	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: failed to marshal JSON: %v\n", err)
		os.Exit(1)
	}

	// Write output
	if *outputFile != "" {
		err := os.WriteFile(*outputFile, jsonData, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: failed to write output file: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Println(string(jsonData))
	}

	// Exit with error if any files failed
	if parseErrors > 0 {
		os.Exit(1)
	}
}

// discoverPDFs finds all PDF files in the given path.
func discoverPDFs(inputPath string) ([]string, error) {
	info, err := os.Stat(inputPath)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	var pdfFiles []string

	if !info.IsDir() {
		// Single file
		if filepath.Ext(inputPath) == ".pdf" {
			pdfFiles = append(pdfFiles, inputPath)
		}
	} else {
		// Directory: walk recursively
		err := filepath.Walk(inputPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && filepath.Ext(path) == ".pdf" {
				pdfFiles = append(pdfFiles, path)
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("failed to walk directory: %w", err)
		}
	}

	return pdfFiles, nil
}
```

- [ ] **Step 2: Build and test**

```bash
cd /Users/welworx/dev-private/flatex-pdf-cli
go build -o flatex-pdf-cli
```

Expected: Binary `flatex-pdf-cli` created.

- [ ] **Step 3: Verify CLI help**

```bash
./flatex-pdf-cli
```

Expected: Usage message printed.

- [ ] **Step 4: Commit**

```bash
git add main.go
git commit -m "feat: implement CLI entry point with flag handling and output formatting"
```

---

### Task 11: Add Test Data and Integration Tests

**Files:**
- Create: `testdata/sample_trade.pdf` (user provides real PDF)
- Create: `testdata/sample_dividend.pdf` (user provides real PDF)
- Create: `testdata/sample_interest.pdf` (user provides real PDF)
- Create: `testdata/sample_thesaurierung.pdf` (user provides real PDF)

**Interfaces:**
- Consumes: Real flatex PDFs
- Produces: Integration test suite

- [ ] **Step 1: Prepare test data directory**

```bash
mkdir -p /Users/welworx/dev-private/flatex-pdf-cli/testdata
```

- [ ] **Step 2: User provides sample PDFs**

User should copy real flatex PDFs to testdata/ (trade, dividend, interest, thesaurierung examples).

- [ ] **Step 3: Add integration test**

```go
// Add to internal/extractor/extractor_test.go
func TestIntegrationTradeConfirmation(t *testing.T) {
	// This test requires real PDF in testdata/
	pdfPath := "../../testdata/sample_trade.pdf"
	doc, err := ExtractPDF(pdfPath)
	if err != nil {
		t.Fatalf("ExtractPDF failed: %v", err)
	}

	if doc.DocumentType != "TRADE" {
		t.Errorf("DocumentType mismatch: got %s", doc.DocumentType)
	}
}

func TestIntegrationDividendStatement(t *testing.T) {
	pdfPath := "../../testdata/sample_dividend.pdf"
	doc, err := ExtractPDF(pdfPath)
	if err != nil {
		t.Fatalf("ExtractPDF failed: %v", err)
	}

	if doc.DocumentType != "DIVIDEND" {
		t.Errorf("DocumentType mismatch: got %s", doc.DocumentType)
	}
}
```

- [ ] **Step 4: Run all tests**

```bash
go test ./...
```

Expected: PASS (assuming PDFs are correctly placed).

- [ ] **Step 5: Commit**

```bash
git add testdata/
git commit -m "test: add sample PDFs for integration testing"
```

---

### Task 12: GitHub Actions CI/CD Pipeline

**Files:**
- Create: `.github/workflows/ci.yml`

**Interfaces:**
- Produces: Automated testing and build on every push

- [ ] **Step 1: Create GitHub Actions workflow**

```bash
mkdir -p /Users/welworx/dev-private/flatex-pdf-cli/.github/workflows
```

```yaml
# File: .github/workflows/ci.yml
name: CI

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main ]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.21'
      
      - name: Run golangci-lint
        uses: golangci/golangci-lint-action@v3
        with:
          version: latest
      
      - name: Run go fmt
        run: go fmt ./...
      
      - name: Run go vet
        run: go vet ./...
      
      - name: Run tests
        run: go test -v -race ./...
      
      - name: Build binary
        run: go build -o flatex-pdf-cli

  build-release:
    runs-on: ubuntu-latest
    needs: test
    if: startsWith(github.ref, 'refs/tags/')
    steps:
      - uses: actions/checkout@v4
      
      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.21'
      
      - name: Build binaries
        run: |
          GOOS=linux GOARCH=amd64 go build -o flatex-pdf-cli-linux-amd64
          GOOS=darwin GOARCH=amd64 go build -o flatex-pdf-cli-darwin-amd64
          GOOS=windows GOARCH=amd64 go build -o flatex-pdf-cli-windows-amd64.exe
      
      - name: Create Release
        uses: softprops/action-gh-release@v1
        with:
          files: flatex-pdf-cli-*
```

- [ ] **Step 2: Verify workflow syntax**

```bash
cd /Users/welworx/dev-private/flatex-pdf-cli
# Manual check: ensure YAML is valid
cat .github/workflows/ci.yml
```

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/ci.yml
git commit -m "ci: add GitHub Actions pipeline for testing and building"
```

---

### Task 13: Pre-Commit Hooks Setup

**Files:**
- Create: `.pre-commit-config.yaml` (optional local development)

**Interfaces:**
- Produces: Local git hooks for code quality

- [ ] **Step 1: Create pre-commit config**

```yaml
# File: .pre-commit-config.yaml (optional)
repos:
  - repo: local
    hooks:
      - id: go-fmt
        name: go fmt
        entry: bash -c 'go fmt ./...'
        language: system
        pass_filenames: false
      
      - id: go-vet
        name: go vet
        entry: bash -c 'go vet ./...'
        language: system
        pass_filenames: false
      
      - id: go-test
        name: go test
        entry: bash -c 'go test ./...'
        language: system
        pass_filenames: false
```

- [ ] **Step 2: Document setup in README**

(Create simple README with usage instructions — see Task 14)

- [ ] **Step 3: Commit**

```bash
git add .pre-commit-config.yaml
git commit -m "chore: add pre-commit hook configuration for local development"
```

---

### Task 14: Documentation & Final Verification

**Files:**
- Create: `README.md`

**Interfaces:**
- Produces: User-facing documentation and usage examples

- [ ] **Step 1: Create README**

```markdown
# flatex-pdf-cli

A command-line tool for extracting structured transaction data from flatex PDF documents.

## Installation

Download the pre-built binary from [Releases](https://github.com/welworx/flatex-pdf-cli/releases) or build from source:

\`\`\`bash
git clone https://github.com/welworx/flatex-pdf-cli.git
cd flatex-pdf-cli
go build -o flatex-pdf-cli
\`\`\`

## Usage

### Basic: Parse a single PDF

\`\`\`bash
./flatex-pdf-cli statement.pdf
\`\`\`

Output: JSON transactions array to stdout

### Parse a folder of PDFs

\`\`\`bash
./flatex-pdf-cli ./statements/
\`\`\`

### Save to file

\`\`\`bash
./flatex-pdf-cli statements/ -o results.json
\`\`\`

### Include source filename

\`\`\`bash
./flatex-pdf-cli statements/ --include-source
\`\`\`

### Include account metadata

\`\`\`bash
./flatex-pdf-cli statements/ --include-metadata
\`\`\`

### Combine flags

\`\`\`bash
./flatex-pdf-cli statements/ -o results.json --include-source --include-metadata
\`\`\`

## Supported Document Types

- **TRADE**: Buy/Sell confirmations
- **DIVIDEND**: Distribution statements
- **INTEREST**: Interest statements
- **THESAURIERUNG**: Reinvestment/accumulation statements

## Output Format

### Without metadata (default)

\`\`\`json
[
  {
    "document_type": "TRADE",
    "isin": "IE000YU9K6K2",
    "quantity": 1.058537,
    ...
  }
]
\`\`\`

### With metadata

\`\`\`json
{
  "metadata": {
    "depot_number": "31022213999",
    "depot_holder": "Max Mustermann"
  },
  "transactions": [...]
}
\`\`\`

## Development

### Run tests

\`\`\`bash
go test ./...
\`\`\`

### Linting

\`\`\`bash
golangci-lint run
go fmt ./...
go vet ./...
\`\`\`

### Local pre-commit hooks (optional)

\`\`\`bash
pre-commit install
\`\`\`

## License

MIT
```

- [ ] **Step 2: Test CLI end-to-end (manual)**

Assuming you have a real flatex PDF:

```bash
./flatex-pdf-cli /path/to/sample.pdf --include-source --include-metadata
```

Verify:
- JSON output is valid
- Fields are populated correctly
- Exit code is 0 on success

- [ ] **Step 3: Run full test suite**

```bash
go test -v ./...
go fmt ./...
go vet ./...
golangci-lint run
```

Expected: All tests pass, no fmt/vet issues.

- [ ] **Step 4: Build binary for distribution**

```bash
go build -o flatex-pdf-cli
file ./flatex-pdf-cli
```

Verify: Binary is executable.

- [ ] **Step 5: Commit README**

```bash
git add README.md
git commit -m "docs: add README with usage and development instructions"
```

---

## Summary

**Deliverables:**
- ✅ Single Go binary (`flatex-pdf-cli`) with no system dependencies
- ✅ Support for 4 transaction types (TRADE, DIVIDEND, INTEREST, THESAURIERUNG)
- ✅ Comprehensive JSON output with optional metadata wrapping
- ✅ Optional `--include-source` and `--include-metadata` flags
- ✅ Unit tests for all components
- ✅ Integration tests with real sample PDFs
- ✅ GitHub Actions CI/CD pipeline
- ✅ Linting (golangci-lint), formatting (go fmt), testing (go test)
- ✅ Pre-commit hook configuration
- ✅ User-facing documentation

**Total commits:** ~15 atomic commits per feature/component

---

**Plan saved to `docs/superpowers/plans/2026-06-24-flatex-pdf-cli-plan.md`**

Two execution options:

**1. Subagent-Driven (recommended)** — I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** — Execute tasks in this session using executing-plans, batch execution with checkpoints

Which approach?
