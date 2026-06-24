# Document Type Detection Fix — Real PDF Analysis & Pattern Matching

**Date:** 2026-06-24  
**Status:** Design (Approved)  
**Scope:** Fix document type detection failure on real flatex PDFs using gxpdf text extraction and improved pattern matching.

---

## Problem Statement

The CLI fails on real flatex PDFs with:
```
error parsing /path/to/document.pdf: unknown document type: UNKNOWN
```

**Root causes:**
1. Text extraction is not implemented (`extractTextFromPDF()` returns empty string)
2. Pattern matching in `detectDocumentType()` uses simplistic keyword matching that doesn't account for real flatex document formatting
3. No test fixtures from real PDFs — tests use synthetic text only

**Impact:** Production use is blocked. All transaction types (TRADE, DIVIDEND, THESAURIERUNG, INTEREST) affected.

---

## Solution Overview

**Three-part approach:**

1. **Implement text extraction** using gxpdf library to extract real text from flatex PDFs
2. **Improve pattern matching** with field-based detection + flexible keyword fallback, handling German document structure
3. **Create test fixtures** by extracting text from 4 real PDFs, redacting PII, and using them for verification and testing

---

## Design Details

### 1. Text Extraction Integration

**File:** `internal/extractor/extractor.go` — `extractTextFromPDF()` function

**Change:**
- Add `gxpdf` as a dependency
- Remove `pdfcpu` dependency (no longer needed)
- Replace the current stub (returns empty string) with real gxpdf text extraction
- Keep function signature unchanged: `func extractTextFromPDF(file *os.File) (string, error)`
- If gxpdf fails to extract text, return a clear error (don't silently return empty string)

**Why gxpdf:**
- Reliable for German PDFs (flatex uses German)
- Handles complex PDF structures
- Pure Go library (no external binaries)
- No new system dependencies

**Error handling:**
- If PDF cannot be read: return `fmt.Errorf("failed to read PDF: %w", err)`
- If text extraction fails: return `fmt.Errorf("failed to extract text: %w", err)`
- Do not return empty string on failure — let the error bubble up so CLI can report it clearly

---

### 2. Pattern Matching Improvements

**File:** `internal/extractor/extractor.go` — `detectDocumentType()` function

**Current state:** Simple keyword matching on lowercase text
```go
if strings.Contains(lowerText, "kauf") || strings.Contains(lowerText, "verkauf") {
    return "TRADE"
}
```

**New approach:** Field-based + keyword-based detection

**Detection order (highest to lowest priority):**

1. **Field-based detection** — look for document type indicators in common locations:
   - TRADE: "kaufbestätigung", "verkaufsbestätigung", "trade confirmation" (headers/titles)
   - DIVIDEND: "ertragsausschüttung", "dividendenausschüttung", "dividend payment" (headers)
   - THESAURIERUNG: "ertragsmitteilung", "thesaurierung", "reinvestment notice" (headers)
   - INTEREST: "zinsen", "zinsabrechnung", "interest statement" (headers)

2. **Keyword fallback** — if field-based misses, search for keywords anywhere in text:
   - Use word boundaries where possible (e.g., `\bkauf\b` to avoid false matches)
   - Normalize whitespace and case

3. **Return UNKNOWN** — if no pattern matches

**Language support:**
- German keywords only (for now)
- Structure allows English keywords to be added later without refactoring
- ponytail: start German-only, extend if needed

**Example implementation sketch:**
```go
func detectDocumentType(text string) string {
    lowerText := strings.ToLower(text)
    
    // Field-based detection (check document headers/titles first)
    fieldPatterns := map[string][]string{
        "TRADE": {"kaufbestätigung", "verkaufsbestätigung"},
        "DIVIDEND": {"ertragsausschüttung", "dividendenausschüttung"},
        "THESAURIERUNG": {"ertragsmitteilung", "thesaurierung"},
        "INTEREST": {"zinsabrechnung", "zinsen"},
    }
    
    for docType, patterns := range fieldPatterns {
        for _, pattern := range patterns {
            if strings.Contains(lowerText, pattern) {
                return docType
            }
        }
    }
    
    return "UNKNOWN"
}
```

**Why this approach:**
- More reliable than simple substring matching
- Handles German compound words and formatting variations
- Easy to extend with English later
- Field-based detection catches document type from headers, not random text matches

---

### 3. PII Redaction & Test Fixtures

**Process:**

1. **Extract text** from 4 real PDFs in `sensitive_test_docs/`:
   - `20250916_KaufFondsZertifikate_31022213999_100000001.pdf` (TRADE)
   - `20260130_KaufFondsZertifikate_31022213999_100000004.pdf` (TRADE)
   - `20251002_Fondsertragsausschuettung_31022213999_100000002.pdf` (DIVIDEND)
   - `20251003_Fondsertragsausschuettung_31022213999_100000003.pdf` (DIVIDEND)

2. **Identify PII** to redact:
   - Names (Depotinhaber, Kontoinhaber, etc.)
   - Account/depot numbers (Depotnummer, Kontonummer)
   - ISINs, portfolio identifiers
   - Amounts/prices (optional but recommended)
   - Transaction dates (optional)
   - Email, phone, address

3. **Redact manually**:
   - Replace sensitive values with placeholders: `[NAME_REDACTED]`, `[ACCOUNT_123456]`, `[ISIN_XXXXXXXXXX]`
   - Keep document structure and type-detection keywords intact
   - Preserve formatting so patterns still match as they would in real documents

4. **Save redacted versions** to temporary location (not committed yet)

5. **User review** (approval gate):
   - Show redacted files to user before committing
   - Get explicit approval that PII is sufficiently redacted
   - This is a hard gate — files do not enter the repo without user sign-off

6. **Commit to repo**:
   - Move redacted `.txt` files to `internal/extractor/testdata/`
   - Add test cases referencing these fixtures

**Redacted files location (temporary, pre-commit):**
```
/private/tmp/claude-501/-Users-welworx-dev-private-flatex-pdf-cli/605d7ff0-103e-465c-8998-c6a622ca32fc/scratchpad/
  - trade_sample_1_redacted.txt
  - trade_sample_2_redacted.txt
  - dividend_sample_1_redacted.txt
  - dividend_sample_2_redacted.txt
```

**Final location (post-approval):**
```
internal/extractor/testdata/
  - trade_sample_1.txt
  - trade_sample_2.txt
  - dividend_sample_1.txt
  - dividend_sample_2.txt
```

---

### 4. Testing Strategy

**Update existing integration tests** in `internal/extractor/extractor_test.go`:

- `TestIntegrationTradeConfirmation` — load `trade_sample_1.txt`, verify detects as TRADE
- `TestIntegrationDividendStatement` — load `dividend_sample_1.txt`, verify detects as DIVIDEND
- `TestIntegrationThesaurierung` — (not in 4 PDFs, but can be extended later)

**New tests:**
- Add table-driven test that verifies all 4 redacted samples detect correctly
- Verify metadata extraction (Depotnummer, Depotinhaber) works on real text

**What we're validating:**
- Text extraction via gxpdf works
- Pattern matching detects correct types from real flatex documents
- Metadata extraction still works with real formatting
- No regressions in synthetic tests

---

## Implementation Order

1. **Add gxpdf dependency** → implement `extractTextFromPDF()` with real extraction
2. **Extract text** from the 4 real PDFs, save raw output
3. **Analyze keywords** — identify actual patterns in extracted text
4. **Redact PII** → create redacted versions
5. **Show user** redacted files → get approval
6. **Improve `detectDocumentType()`** → based on real keyword patterns found
7. **Add test fixtures** to `testdata/` → update tests to use them
8. **Verify** all 4 PDFs detect correctly
9. **Commit** redacted fixtures + improved code

---

## Success Criteria

- ✅ gxpdf text extraction is integrated and working
- ✅ All 4 real PDFs extract text successfully
- ✅ `detectDocumentType()` correctly identifies all 4 PDFs (2 TRADE, 2 DIVIDEND)
- ✅ PII-redacted fixtures exist in repo and are used by tests
- ✅ User has approved redacted content before commit
- ✅ No UNKNOWN document type errors on real flatex PDFs
- ✅ Tests pass with real extracted text (not just synthetic text)

---

## Files Modified

- `internal/extractor/extractor.go` — remove pdfcpu import, add gxpdf text extraction + improved pattern matching
- `internal/extractor/extractor_test.go` — update integration tests
- `go.mod` / `go.sum` — remove pdfcpu, add gxpdf dependency
- `internal/extractor/testdata/*.txt` — new redacted test fixtures (committed)

---

## Constraints & Assumptions

- **Scope:** Fix the 4 test PDFs only (TRADE + DIVIDEND). THESAURIERUNG and INTEREST not yet in real PDFs.
- **Language:** German only (English support deferred).
- **PII redaction:** Manual visual review, user approval required before commit.
- **Do not commit:** `sensitive_test_docs/` remains local and uncommitted.

---

## Future Extensions

- Add English keyword patterns to `detectDocumentType()` when needed
- Add real samples for THESAURIERUNG and INTEREST document types
- Extend field-based detection with more specific flatex document patterns
- Consider structured field extraction if more document types are added
