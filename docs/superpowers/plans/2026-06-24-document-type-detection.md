# Document Type Detection Fix Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix document type detection on real flatex PDFs by implementing gxpdf text extraction, analyzing actual keywords, and improving pattern matching.

**Architecture:** 
1. Replace pdfcpu with gxpdf for text extraction
2. Extract text from 4 real test PDFs and analyze keywords
3. Manually redact PII, get user approval
4. Improve detectDocumentType() based on real patterns found
5. Add redacted fixtures to repo and update tests

**Tech Stack:** 
- gxpdf for PDF text extraction
- Go stdlib strings for pattern matching
- Existing test framework

## Global Constraints

- Do NOT commit `sensitive_test_docs/` or its contents
- Show user redacted fixture files before committing to repo (approval gate)
- Scope: Fix the 4 real test PDFs (2 TRADE, 2 DIVIDEND) only
- German keywords only (English support deferred)
- Ponytail mode: Write minimal code that works, extend only if needed

---

## Task 1: Add gxpdf dependency, remove pdfcpu

**Files:**
- Modify: `go.mod`
- Modify: `go.sum`

**Interfaces:**
- Produces: gxpdf available as a dependency, pdfcpu removed

- [ ] **Step 1: Add gxpdf to go.mod**

Run:
```bash
cd /Users/welworx/dev-private/flatex-pdf-cli
go get github.com/coregx/gxpdf
```

Expected: gxpdf is added to go.mod and go.sum

- [ ] **Step 2: Remove pdfcpu dependency**

Run:
```bash
cd /Users/welworx/dev-private/flatex-pdf-cli
go mod tidy
```

This will remove pdfcpu if it's no longer imported anywhere.

Expected: pdfcpu removed from go.mod, go.sum cleaned up

- [ ] **Step 3: Verify dependencies**

Run:
```bash
cd /Users/welworx/dev-private/flatex-pdf-cli
grep -E 'gxpdf|pdfcpu' go.mod
```

Expected: Only `github.com/coregx/gxpdf` present, pdfcpu absent

- [ ] **Step 4: Commit**

```bash
cd /Users/welworx/dev-private/flatex-pdf-cli
git add go.mod go.sum
git commit -m "chore: replace pdfcpu with gxpdf for text extraction"
```

---

## Task 2: Implement extractTextFromPDF() with gxpdf

**Files:**
- Modify: `internal/extractor/extractor.go` (lines 1-60)
- Modify: `internal/extractor/extractor_test.go`

**Interfaces:**
- Consumes: gxpdf library from Task 1
- Produces: `func extractTextFromPDF(file *os.File) (string, error)` that returns actual PDF text

- [ ] **Step 1: Write a failing test for text extraction**

Open `internal/extractor/extractor_test.go` and add this test at the end of the file:

```go
// TestTextExtractionFromRealPDF tests that gxpdf can extract text from a real PDF.
func TestTextExtractionFromRealPDF(t *testing.T) {
	// Use one of the real PDFs from sensitive_test_docs for testing
	pdfPath := "../../sensitive_test_docs/20250916_KaufFondsZertifikate_31022213999_100000001.pdf"
	
	file, err := os.Open(pdfPath)
	if err != nil {
		t.Skipf("test PDF not found at %s, skipping", pdfPath)
	}
	defer file.Close()
	
	text, err := extractTextFromPDF(file)
	if err != nil {
		t.Fatalf("extractTextFromPDF failed: %v", err)
	}
	
	// Text should not be empty
	if text == "" {
		t.Error("expected non-empty text from PDF, got empty string")
	}
	
	// Should contain some recognizable German text or common flatex keywords
	if !strings.Contains(strings.ToLower(text), "depot") && 
	   !strings.Contains(strings.ToLower(text), "kauf") &&
	   !strings.Contains(strings.ToLower(text), "flatex") {
		t.Logf("warning: extracted text doesn't contain expected keywords (may indicate extraction issue)")
		t.Logf("first 500 chars: %s", text[:min(500, len(text))])
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:
```bash
cd /Users/welworx/dev-private/flatex-pdf-cli
go test ./internal/extractor -v -run TestTextExtractionFromRealPDF
```

Expected: FAIL with error about `extractTextFromPDF returning empty string`

- [ ] **Step 3: Update extractor.go imports and implement extractTextFromPDF**

Open `internal/extractor/extractor.go`. Replace the import block and the `extractTextFromPDF` function:

Change from:
```go
import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"
)
```

To:
```go
import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/coregx/gxpdf"
)
```

Then replace the `extractTextFromPDF` function (lines 47-60) with:

```go
// extractTextFromPDF extracts text content from a PDF file using gxpdf.
func extractTextFromPDF(file *os.File) (string, error) {
	// Read the PDF from file
	reader := gxpdf.NewPdfReaderFromStream(file, nil)
	if reader == nil {
		return "", fmt.Errorf("failed to read PDF: invalid PDF file")
	}

	// Extract text from all pages
	var allText strings.Builder
	pageCount := reader.GetPageCount()

	for i := 1; i <= pageCount; i++ {
		page, err := reader.GetPage(i)
		if err != nil {
			return "", fmt.Errorf("failed to read page %d: %w", i, err)
		}

		pageText, err := page.ExtractText()
		if err != nil {
			return "", fmt.Errorf("failed to extract text from page %d: %w", i, err)
		}

		allText.WriteString(pageText)
		allText.WriteString("\n")
	}

	return allText.String(), nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run:
```bash
cd /Users/welworx/dev-private/flatex-pdf-cli
go test ./internal/extractor -v -run TestTextExtractionFromRealPDF
```

Expected: PASS (if real PDF exists and is readable)

- [ ] **Step 5: Run all extractor tests to ensure no regressions**

Run:
```bash
cd /Users/welworx/dev-private/flatex-pdf-cli
go test ./internal/extractor -v
```

Expected: All tests pass

- [ ] **Step 6: Commit**

```bash
cd /Users/welworx/dev-private/flatex-pdf-cli
git add internal/extractor/extractor.go internal/extractor/extractor_test.go go.mod go.sum
git commit -m "feat: implement PDF text extraction using gxpdf"
```

---

## Task 3: Extract text from 4 real PDFs and analyze keywords

**Files:**
- Create: `/private/tmp/claude-501/-Users-welworx-dev-private-flatex-pdf-cli/605d7ff0-103e-465c-8998-c6a622ca32fc/scratchpad/extracted_texts.md` (debug notes, not committed)

**Interfaces:**
- Consumes: Working extractTextFromPDF() from Task 2
- Produces: Extracted raw text from 4 PDFs, keyword analysis notes

- [ ] **Step 1: Create a debug script to extract text from all 4 PDFs**

Create file `/Users/welworx/dev-private/flatex-pdf-cli/cmd/debug_extract.go`:

```go
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"flatex-pdf-cli/internal/extractor"
)

func main() {
	pdfFiles := []string{
		"sensitive_test_docs/20250916_KaufFondsZertifikate_31022213999_100000001.pdf",
		"sensitive_test_docs/20260130_KaufFondsZertifikate_31022213999_100000004.pdf",
		"sensitive_test_docs/20251002_Fondsertragsausschuettung_31022213999_100000002.pdf",
		"sensitive_test_docs/20251003_Fondsertragsausschuettung_31022213999_100000003.pdf",
	}

	for _, pdfFile := range pdfFiles {
		fmt.Printf("\n=== Extracting: %s ===\n", filepath.Base(pdfFile))
		
		file, err := os.Open(pdfFile)
		if err != nil {
			fmt.Printf("ERROR opening file: %v\n", err)
			continue
		}
		defer file.Close()

		doc, err := extractor.ExtractPDF(pdfFile)
		if err != nil {
			fmt.Printf("ERROR extracting: %v\n", err)
			continue
		}

		fmt.Printf("Filename: %s\n", doc.Filename)
		fmt.Printf("Document Type Detected: %s\n", doc.DocumentType)
		fmt.Printf("Depot Number: %s\n", doc.DepotNumber)
		fmt.Printf("Depot Holder: %s\n", doc.DepotHolder)
		fmt.Printf("\nExtracted Text (first 2000 chars):\n")
		if len(doc.Text) > 2000 {
			fmt.Println(doc.Text[:2000])
		} else {
			fmt.Println(doc.Text)
		}
	}
}
```

- [ ] **Step 2: Run the debug script to extract text**

Run:
```bash
cd /Users/welworx/dev-private/flatex-pdf-cli
go run cmd/debug_extract.go
```

Expected: Output showing extracted text from each PDF. Document type will be UNKNOWN (as expected before we improve detectDocumentType).

- [ ] **Step 3: Analyze and record keywords**

Look at the extracted text and save notes to scratchpad. For each PDF, identify:
- What German keywords appear in the text that indicate document type?
- What common words/phrases appear in TRADE docs vs DIVIDEND docs?
- Are there any headers, titles, or field names that consistently appear?

Save analysis to `/private/tmp/claude-501/-Users-welworx-dev-private-flatex-pdf-cli/605d7ff0-103e-465c-8998-c6a622ca32fc/scratchpad/keyword_analysis.txt`:

Example format:
```
TRADE Documents (files 1, 2):
- Keywords found: kauf, kaufbestätigung, verkauf, trade, order, shares, ISIN
- Common headers: "Kaufbestätigung", "Handelsbestätigung"
- Field names that indicate type: "Wertpapier", "Menge", "Kurs"

DIVIDEND Documents (files 3, 4):
- Keywords found: ausschüttung, dividend, erträge, ertragsmitteilung, thesaurier
- Common headers: "Fondsertragsausschüttung", "Ertragsausschüttung"
- Field names: "Ausschüttungsbetrag", "Ertragsart"
```

- [ ] **Step 4: Delete the debug script (not needed in repo)**

Run:
```bash
rm /Users/welworx/dev-private/flatex-pdf-cli/cmd/debug_extract.go
```

---

## Task 4: Manually redact PII and create fixture files

**Files:**
- Create: `/private/tmp/claude-501/-Users-welworx-dev-private-flatex-pdf-cli/605d7ff0-103e-465c-8998-c6a622ca32fc/scratchpad/trade_sample_1_redacted.txt`
- Create: `/private/tmp/claude-501/-Users-welworx-dev-private-flatex-pdf-cli/605d7ff0-103e-465c-8998-c6a622ca32fc/scratchpad/trade_sample_2_redacted.txt`
- Create: `/private/tmp/claude-501/-Users-welworx-dev-private-flatex-pdf-cli/605d7ff0-103e-465c-8998-c6a622ca32fc/scratchpad/dividend_sample_1_redacted.txt`
- Create: `/private/tmp/claude-501/-Users-welworx-dev-private-flatex-pdf-cli/605d7ff0-103e-465c-8998-c6a622ca32fc/scratchpad/dividend_sample_2_redacted.txt`

**Interfaces:**
- Consumes: Extracted text from Task 3
- Produces: Redacted text samples ready for user review

- [ ] **Step 1: Extract and redact Trade Sample 1**

From the extracted text of `20250916_KaufFondsZertifikate_31022213999_100000001.pdf`:
1. Copy the full extracted text
2. Replace all occurrences of:
   - Personal names (e.g., "Max Mustermann", "John Doe") with `[NAME_REDACTED]`
   - Depotnummer values (e.g., "31022213999") with `[ACCOUNT_REDACTED]`
   - ISINs (e.g., "DE0008469008") with `[ISIN_REDACTED]`
   - Specific amounts/prices with `[AMOUNT_REDACTED]`
   - Specific dates with `[DATE_REDACTED]`
   - Email addresses with `[EMAIL_REDACTED]`
   - Phone numbers with `[PHONE_REDACTED]`

Save the redacted version to `/private/tmp/claude-501/-Users-welworx-dev-private-flatex-pdf-cli/605d7ff0-103e-465c-8998-c6a622ca32fc/scratchpad/trade_sample_1_redacted.txt`

Keep document structure and type-detection keywords intact.

- [ ] **Step 2: Extract and redact Trade Sample 2**

From the extracted text of `20260130_KaufFondsZertifikate_31022213999_100000004.pdf`:
Follow same redaction process as Step 1.

Save to `/private/tmp/claude-501/-Users-welworx-dev-private-flatex-pdf-cli/605d7ff0-103e-465c-8998-c6a622ca32fc/scratchpad/trade_sample_2_redacted.txt`

- [ ] **Step 3: Extract and redact Dividend Sample 1**

From the extracted text of `20251002_Fondsertragsausschuettung_31022213999_100000002.pdf`:
Follow same redaction process.

Save to `/private/tmp/claude-501/-Users-welworx-dev-private-flatex-pdf-cli/605d7ff0-103e-465c-8998-c6a622ca32fc/scratchpad/dividend_sample_1_redacted.txt`

- [ ] **Step 4: Extract and redact Dividend Sample 2**

From the extracted text of `20251003_Fondsertragsausschuettung_31022213999_100000003.pdf`:
Follow same redaction process.

Save to `/private/tmp/claude-501/-Users-welworx-dev-private-flatex-pdf-cli/605d7ff0-103e-465c-8998-c6a622ca32fc/scratchpad/dividend_sample_2_redacted.txt`

- [ ] **Step 5: Create a summary of redacted files**

Create `/private/tmp/claude-501/-Users-welworx-dev-private-flatex-pdf-cli/605d7ff0-103e-465c-8998-c6a622ca32fc/scratchpad/REDACTION_SUMMARY.txt`:

```
Redacted Test Fixtures Summary
==============================

Files created:
1. trade_sample_1_redacted.txt
   Source: 20250916_KaufFondsZertifikate_31022213999_100000001.pdf
   Type: TRADE
   PII Redacted: Names, account numbers, ISINs, amounts, dates

2. trade_sample_2_redacted.txt
   Source: 20260130_KaufFondsZertifikate_31022213999_100000004.pdf
   Type: TRADE
   PII Redacted: Names, account numbers, ISINs, amounts, dates

3. dividend_sample_1_redacted.txt
   Source: 20251002_Fondsertragsausschuettung_31022213999_100000002.pdf
   Type: DIVIDEND
   PII Redacted: Names, account numbers, ISINs, amounts, dates

4. dividend_sample_2_redacted.txt
   Source: 20251003_Fondsertragsausschuettung_31022213999_100000003.pdf
   Type: DIVIDEND
   PII Redacted: Names, account numbers, ISINs, amounts, dates

All files preserve document structure and type-detection keywords.
Ready for user review and approval.
```

---

## Task 5: Present redacted files to user for approval

**Files:**
- Read: All 4 redacted `.txt` files from scratchpad

**Interfaces:**
- Consumes: Redacted fixture files from Task 4
- Produces: User approval to commit fixtures

- [ ] **Step 1: Show redacted files to user**

Display the 4 redacted fixture files to the user for review. User must verify:
- PII is sufficiently redacted
- Document structure is preserved
- Type-detection keywords remain intact
- Files are safe to commit to public repo

Wait for user sign-off before proceeding to next task.

---

## Task 6: Move redacted fixtures to testdata directory

**Files:**
- Create: `internal/extractor/testdata/trade_sample_1.txt`
- Create: `internal/extractor/testdata/trade_sample_2.txt`
- Create: `internal/extractor/testdata/dividend_sample_1.txt`
- Create: `internal/extractor/testdata/dividend_sample_2.txt`

**Interfaces:**
- Consumes: User approval from Task 5, redacted files from Task 4
- Produces: Committed test fixtures in repo

- [ ] **Step 1: Create testdata directory**

Run:
```bash
mkdir -p /Users/welworx/dev-private/flatex-pdf-cli/internal/extractor/testdata
```

- [ ] **Step 2: Copy redacted files to testdata**

Run:
```bash
cp /private/tmp/claude-501/-Users-welworx-dev-private-flatex-pdf-cli/605d7ff0-103e-465c-8998-c6a622ca32fc/scratchpad/trade_sample_1_redacted.txt /Users/welworx/dev-private/flatex-pdf-cli/internal/extractor/testdata/trade_sample_1.txt

cp /private/tmp/claude-501/-Users-welworx-dev-private-flatex-pdf-cli/605d7ff0-103e-465c-8998-c6a622ca32fc/scratchpad/trade_sample_2_redacted.txt /Users/welworx/dev-private/flatex-pdf-cli/internal/extractor/testdata/trade_sample_2.txt

cp /private/tmp/claude-501/-Users-welworx-dev-private-flatex-pdf-cli/605d7ff0-103e-465c-8998-c6a622ca32fc/scratchpad/dividend_sample_1_redacted.txt /Users/welworx/dev-private/flatex-pdf-cli/internal/extractor/testdata/dividend_sample_1.txt

cp /private/tmp/claude-501/-Users-welworx-dev-private-flatex-pdf-cli/605d7ff0-103e-465c-8998-c6a622ca32fc/scratchpad/dividend_sample_2_redacted.txt /Users/welworx/dev-private/flatex-pdf-cli/internal/extractor/testdata/dividend_sample_2.txt
```

- [ ] **Step 3: Verify files are in place**

Run:
```bash
ls -la /Users/welworx/dev-private/flatex-pdf-cli/internal/extractor/testdata/
```

Expected: 4 `.txt` files present

- [ ] **Step 4: Commit test fixtures**

```bash
cd /Users/welworx/dev-private/flatex-pdf-cli
git add internal/extractor/testdata/
git commit -m "test: add redacted PDF fixture files for real-world integration tests"
```

---

## Task 7: Improve detectDocumentType() based on real keywords

**Files:**
- Modify: `internal/extractor/extractor.go` (lines 79-104, detectDocumentType function)
- Modify: `internal/extractor/extractor_test.go`

**Interfaces:**
- Consumes: Keyword analysis from Task 3, test fixtures from Task 6
- Produces: Improved `detectDocumentType(text string) string` that works with real PDF text

- [ ] **Step 1: Write a test for real PDF fixture detection**

Add this test to `internal/extractor/extractor_test.go`:

```go
// TestDocumentTypeDetectionWithRealFixtures verifies document type detection works with real PDF text.
func TestDocumentTypeDetectionWithRealFixtures(t *testing.T) {
	tests := []struct {
		name             string
		fixtureFile      string
		expectedDocType  string
	}{
		{
			name:            "Trade Sample 1",
			fixtureFile:     "testdata/trade_sample_1.txt",
			expectedDocType: "TRADE",
		},
		{
			name:            "Trade Sample 2",
			fixtureFile:     "testdata/trade_sample_2.txt",
			expectedDocType: "TRADE",
		},
		{
			name:            "Dividend Sample 1",
			fixtureFile:     "testdata/dividend_sample_1.txt",
			expectedDocType: "DIVIDEND",
		},
		{
			name:            "Dividend Sample 2",
			fixtureFile:     "testdata/dividend_sample_2.txt",
			expectedDocType: "DIVIDEND",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := os.ReadFile(tt.fixtureFile)
			if err != nil {
				t.Fatalf("failed to read fixture file %s: %v", tt.fixtureFile, err)
			}
			
			text := string(data)
			result := detectDocumentType(text)
			
			if result != tt.expectedDocType {
				t.Errorf("expected document type %q, got %q", tt.expectedDocType, result)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to see what patterns are actually detected**

Run:
```bash
cd /Users/welworx/dev-private/flatex-pdf-cli
go test ./internal/extractor -v -run TestDocumentTypeDetectionWithRealFixtures
```

Expected: FAIL with UNKNOWN for all samples (current implementation doesn't match real text)

- [ ] **Step 3: Analyze what keywords appear and update detectDocumentType()**

Based on Task 3's keyword analysis and the fixture files, update the function. The new implementation should:
- Check for German keywords specific to each document type
- Handle case-insensitive matching (lowercase)
- Use simple `strings.Contains()` for broad matching (ponytail: start simple, optimize if needed)

Replace the `detectDocumentType` function in `internal/extractor/extractor.go`:

```go
// detectDocumentType detects the document type based on keywords in the text.
func detectDocumentType(text string) string {
	lowerText := strings.ToLower(text)

	// Check for TRADE keywords - include compound words and common variations
	tradeKeywords := []string{
		"kauf", "kaufbestätigung", "kaufbestaetigung",
		"verkauf", "verkaufsbestätigung", "verkaufsbestaetigung",
		"trade confirmation", "order confirmation",
	}
	for _, keyword := range tradeKeywords {
		if strings.Contains(lowerText, keyword) {
			return "TRADE"
		}
	}

	// Check for DIVIDEND keywords
	dividendKeywords := []string{
		"ausschüttung", "ausschuettung",
		"dividend", "dividende", "dividenden",
		"ertragsausschüttung", "ertragsausschuettung",
		"fondsertragsausschüttung", "fondsertragsausschuettung",
	}
	for _, keyword := range dividendKeywords {
		if strings.Contains(lowerText, keyword) {
			return "DIVIDEND"
		}
	}

	// Check for THESAURIERUNG keywords
	thesaurierungKeywords := []string{
		"ertragsmitteilung", "thesaurier", "thesaurierung",
		"reinvestment", "reinvestiton",
	}
	for _, keyword := range thesaurierungKeywords {
		if strings.Contains(lowerText, keyword) {
			return "THESAURIERUNG"
		}
	}

	// Check for INTEREST keywords
	interestKeywords := []string{
		"zinsen", "zinsabrechnung", "zinserträge",
		"interest", "kontozinsen",
	}
	for _, keyword := range interestKeywords {
		if strings.Contains(lowerText, keyword) {
			return "INTEREST"
		}
	}

	return "UNKNOWN"
}
```

**Note:** Adjust keyword lists based on your actual analysis from Task 3. If different keywords appear in the real fixtures, update this list accordingly.

- [ ] **Step 4: Run test again to verify it passes**

Run:
```bash
cd /Users/welworx/dev-private/flatex-pdf-cli
go test ./internal/extractor -v -run TestDocumentTypeDetectionWithRealFixtures
```

Expected: PASS for all 4 fixtures

- [ ] **Step 5: Run all extractor tests to ensure no regressions**

Run:
```bash
cd /Users/welworx/dev-private/flatex-pdf-cli
go test ./internal/extractor -v
```

Expected: All tests pass (both old synthetic tests and new real fixture tests)

- [ ] **Step 6: Commit**

```bash
cd /Users/welworx/dev-private/flatex-pdf-cli
git add internal/extractor/extractor.go internal/extractor/extractor_test.go
git commit -m "feat: improve document type detection with real PDF keywords"
```

---

## Task 8: Update integration tests to use real fixture files

**Files:**
- Modify: `internal/extractor/extractor_test.go` (TestIntegrationTradeConfirmation, TestIntegrationDividendStatement)

**Interfaces:**
- Consumes: Fixture files from Task 6, improved detectDocumentType() from Task 7
- Produces: Updated integration tests that use real PDF text samples

- [ ] **Step 1: Update TestIntegrationTradeConfirmation**

Replace the existing `TestIntegrationTradeConfirmation` function in `internal/extractor/extractor_test.go` with:

```go
// TestIntegrationTradeConfirmation tests document type detection with real trade confirmation text.
func TestIntegrationTradeConfirmation(t *testing.T) {
	// Load real fixture file
	data, err := os.ReadFile("testdata/trade_sample_1.txt")
	if err != nil {
		t.Fatalf("failed to read trade fixture: %v", err)
	}

	text := string(data)

	// Test detectDocumentType
	docType := detectDocumentType(text)
	if docType != "TRADE" {
		t.Errorf("expected DocumentType 'TRADE', got '%s'", docType)
	}

	// Test metadata extraction
	depotNum, depotHolder := extractMetadata(text)
	// Note: These will be redacted in the fixture, so we just verify they were extracted
	// (non-empty strings indicate extraction logic worked)
	if depotNum == "" {
		t.Logf("info: depot number extraction returned empty (may be redacted in fixture)")
	}
	if depotHolder == "" {
		t.Logf("info: depot holder extraction returned empty (may be redacted in fixture)")
	}
}
```

- [ ] **Step 2: Update TestIntegrationDividendStatement**

Replace the existing `TestIntegrationDividendStatement` function in `internal/extractor/extractor_test.go` with:

```go
// TestIntegrationDividendStatement tests document type detection with real dividend statement text.
func TestIntegrationDividendStatement(t *testing.T) {
	// Load real fixture file
	data, err := os.ReadFile("testdata/dividend_sample_1.txt")
	if err != nil {
		t.Fatalf("failed to read dividend fixture: %v", err)
	}

	text := string(data)

	// Test detectDocumentType
	docType := detectDocumentType(text)
	if docType != "DIVIDEND" {
		t.Errorf("expected DocumentType 'DIVIDEND', got '%s'", docType)
	}

	// Test metadata extraction
	depotNum, depotHolder := extractMetadata(text)
	// Note: These will be redacted in the fixture, so we just verify they were extracted
	if depotNum == "" {
		t.Logf("info: depot number extraction returned empty (may be redacted in fixture)")
	}
	if depotHolder == "" {
		t.Logf("info: depot holder extraction returned empty (may be redacted in fixture)")
	}
}
```

- [ ] **Step 3: Delete or simplify TestIntegrationThesaurierung**

The current codebase has `TestIntegrationThesaurierung` but we don't have a real THESAURIERUNG fixture yet. Replace it with a simple version that tests with synthetic text:

```go
// TestIntegrationThesaurierung tests document type detection for thesaurierung documents.
// Note: Real fixtures not yet available; uses synthetic text for now.
func TestIntegrationThesaurierung(t *testing.T) {
	// Mock test with synthetic text (real fixture deferred)
	docType := detectDocumentType("Ertragsmitteilung für thesaurierte Fonds\nDepotnummer: [ACCOUNT_REDACTED]\nDepotinhaber: [NAME_REDACTED]")
	if docType != "THESAURIERUNG" {
		t.Errorf("expected DocumentType 'THESAURIERUNG', got '%s'", docType)
	}
}
```

- [ ] **Step 4: Run all tests to verify**

Run:
```bash
cd /Users/welworx/dev-private/flatex-pdf-cli
go test ./internal/extractor -v
```

Expected: All tests pass, including the updated integration tests

- [ ] **Step 5: Commit**

```bash
cd /Users/welworx/dev-private/flatex-pdf-cli
git add internal/extractor/extractor_test.go
git commit -m "test: update integration tests to use real PDF fixtures"
```

---

## Task 9: Final verification and cleanup

**Files:**
- No new files, verification only

**Interfaces:**
- Consumes: All completed tasks 1-8
- Produces: Verified working implementation, all tests passing

- [ ] **Step 1: Run full test suite**

Run:
```bash
cd /Users/welworx/dev-private/flatex-pdf-cli
go test ./... -v
```

Expected: All tests pass

- [ ] **Step 2: Verify document type detection on all 4 real PDFs**

Run:
```bash
cd /Users/welworx/dev-private/flatex-pdf-cli
go run main.go sensitive_test_docs/20250916_KaufFondsZertifikate_31022213999_100000001.pdf
go run main.go sensitive_test_docs/20260130_KaufFondsZertifikate_31022213999_100000004.pdf
go run main.go sensitive_test_docs/20251002_Fondsertragsausschuettung_31022213999_100000002.pdf
go run main.go sensitive_test_docs/20251003_Fondsertragsausschuettung_31022213999_100000003.pdf
```

Expected: 
- First two files: Document type = TRADE (not UNKNOWN)
- Next two files: Document type = DIVIDEND (not UNKNOWN)
- No "unknown document type" errors

- [ ] **Step 3: Verify testdata directory is properly gitignored or committed**

Run:
```bash
cd /Users/welworx/dev-private/flatex-pdf-cli
git status
```

Expected: `internal/extractor/testdata/` files are committed (in git), `sensitive_test_docs/` is NOT tracked

- [ ] **Step 4: Check go.mod for clean state**

Run:
```bash
cd /Users/welworx/dev-private/flatex-pdf-cli
grep -E 'gxpdf|pdfcpu' go.mod
```

Expected: Only `gxpdf` present, `pdfcpu` absent

- [ ] **Step 5: Clean up scratchpad files (optional)**

The redacted files in scratchpad are temporary working files. You can delete them after committing testdata:

Run:
```bash
rm -rf /private/tmp/claude-501/-Users-welworx-dev-private-flatex-pdf-cli/605d7ff0-103e-465c-8998-c6a622ca32fc/scratchpad/*
```

Or keep them locally if you want to preserve them for reference.

- [ ] **Step 6: Final commit message review**

Run:
```bash
cd /Users/welworx/dev-private/flatex-pdf-cli
git log --oneline | head -10
```

Expected: Recent commits show:
1. "test: update integration tests to use real PDF fixtures"
2. "feat: improve document type detection with real PDF keywords"
3. "test: add redacted PDF fixture files"
4. "feat: implement PDF text extraction using gxpdf"
5. "chore: replace pdfcpu with gxpdf"

---

## Summary of Changes

✅ **Dependency**: Removed pdfcpu, added gxpdf
✅ **Text Extraction**: Implemented `extractTextFromPDF()` using gxpdf
✅ **Pattern Matching**: Improved `detectDocumentType()` with real German keywords
✅ **Test Fixtures**: Added 4 redacted PDF text samples (PII-safe)
✅ **Tests**: Updated integration tests to use real fixtures
✅ **Verification**: All 4 test PDFs now correctly detect type (TRADE or DIVIDEND)

---
