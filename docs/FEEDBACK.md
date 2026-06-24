# Real-World Testing Feedback

## Issue: Document Type Detection Failing on Real flatex PDFs

**Date:** 2026-06-24  
**Status:** OPEN  
**Severity:** High (core functionality blocked)

### Problem

When testing flatex-pdf-cli with real flatex PDF documents, document type detection fails with:

```
error parsing /path/to/document.pdf: unknown document type: UNKNOWN
```

**Example failure:**
```
File: 20250916_KaufFondsZertifikate_31022213999_517614092.pdf
Expected type: TRADE
Detected type: UNKNOWN
```

### Root Cause

Document type detection in `internal/extractor/detectDocumentType()` uses keyword matching with patterns:
- "kauf" or "verkauf" → TRADE
- "ausschüttung" (not "thesaurierung") → DIVIDEND
- "ertragsmitteilung" or "thesaurierung" → THESAURIERUNG
- "zinsen" → INTEREST

These patterns are not matching the actual text in real flatex PDFs. Possible reasons:
1. Keywords have different case, spacing, or formatting
2. Keywords appear in different context or position
3. PDF text extraction (via pdfcpu) produces different layout than expected
4. Document structure differs from test assumptions

### Impact

- CLI cannot parse real flatex documents
- All transaction types (TRADE, DIVIDEND, INTEREST, THESAURIERUNG) affected
- Blocks production use

### Required Fix

1. **Debug extracted text** from real PDFs to identify actual keyword patterns
2. **Improve pattern matching** in `internal/extractor/detectDocumentType()`:
   - Make keyword search more flexible (trim whitespace, handle case variations)
   - Add context-sensitive detection (field names specific to each type)
   - Consider fallback detection methods
3. **Add test fixtures** with real PDF text samples (anonymized)
4. **Update detection logic** to handle real-world PDF formats

### Investigation Needed

To fix this, we need to see:
- Actual extracted text from failing PDFs (first 2000+ characters)
- Document type that PDF should be classified as
- Any patterns/keywords visible in the text

### Files Affected

- `internal/extractor/extractor.go` — `detectDocumentType()` function (lines ~70-95)

### Notes

- PDF text extraction is working (error occurs during type detection, not extraction)
- Metadata extraction (Depotnummer, Depotinhaber) is working
- This is a document type detection algorithm issue, not a PDF parsing issue
