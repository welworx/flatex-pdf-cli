# Test Data Directory

This directory contains sample PDF documents used for integration testing of the flatex PDF CLI extractor.

## Sample Files

The following test files are expected by the integration tests in `internal/extractor/extractor_test.go`:

### sample_trade.pdf
- **Description**: Trade confirmation document (Handelsbestätigung)
- **Document Type**: `TRADE`
- **Expected Content**: Purchase or sale of securities
- **Required Keywords**: "Kauf" (purchase) or "Verkauf" (sale)
- **Metadata**: Depot number and depot holder information

### sample_dividend.pdf
- **Description**: Dividend statement (Ausschüttungsmitteilung)
- **Document Type**: `DIVIDEND`
- **Expected Content**: Dividend payment notification
- **Required Keywords**: "Ausschüttung" (distribution/dividend)
- **Metadata**: Depot number and depot holder information

### sample_thesaurierung.pdf
- **Description**: Earnings statement for thesaurified funds (Ertragsmitteilung)
- **Document Type**: `THESAURIERUNG`
- **Expected Content**: Reinvested earnings from funds
- **Required Keywords**: "Ertragsmitteilung" (earnings statement)
- **Metadata**: Depot number and depot holder information

## Adding Test PDFs

To enable full integration testing with real flatex documents:

1. Obtain sample PDF documents from your flatex account (account statements, transaction confirmations, dividend notifications, etc.)
2. Place them in this directory with the appropriate names matching the test expectations
3. Run the integration tests: `go test ./internal/extractor -v -run TestIntegration`

The tests will:
- Skip gracefully if PDF files are not found
- Run with mock text extraction if PDF files are present but invalid
- Run full integration tests with actual PDF parsing if valid PDFs are provided

## Current Test Behavior

Currently, the integration tests run with mock text extraction. This allows the tests to validate:
- Document type detection (TRADE, DIVIDEND, THESAURIERUNG, etc.)
- Metadata extraction (depot number, depot holder)
- End-to-end extraction workflow

Once real flatex PDFs are added, the tests will validate actual PDF parsing and text extraction capabilities.

## Document Type Detection

The extractor identifies document types based on German language keywords:

- **TRADE**: "Kauf" (purchase) or "Verkauf" (sale)
- **DIVIDEND**: "Ausschüttung" (distribution)
- **INTEREST**: "Zinsen" (interest)
- **THESAURIERUNG**: "Ertragsmitteilung" (earnings statement)
- **UNKNOWN**: No recognized keywords

## Metadata Extraction

All documents should contain:
- **Depotnummer**: Portfolio/depot number (e.g., "31022213999")
- **Depotinhaber**: Depot holder/account owner name (e.g., "Max Mustermann")

The regex patterns support both colon (`:`) and equals (`=`) as separators.
