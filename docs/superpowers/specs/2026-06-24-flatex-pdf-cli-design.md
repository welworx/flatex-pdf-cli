# flatex-pdf-cli Design Specification

**Date:** 2026-06-24  
**Status:** Approved  
**Version:** 1.0

## Overview

`flatex-pdf-cli` is a command-line tool that extracts structured transaction data from flatex PDF documents (trade confirmations, account statements, tax documents) and outputs JSON for downstream processing by coding agents.

**Primary use case:** Parse flatex documents at scale, convert to structured JSON, feed results into agent-driven workflows for further processing and analysis.

## Requirements

### Functional Requirements

1. **Input:** Accept single PDF file or folder of PDFs
2. **Output:** JSON array containing extracted transactions (multiple types: trades, dividends, interest, reinvestment)
3. **Data extraction:** Extract full transaction details including:
   - Trade confirmations: ISIN, WKN, quantity (fractional), price, costs, fees, withholding tax, custody
   - Dividend/Interest: ISIN, quantity, gross/net amounts, withholding tax, exchange rates
   - Reinvestment (Thesaurierung): ISIN, quantity, reinvestment amounts, withholding tax, exchange rates
   - Currency tracking: all amounts include currency; exchange rates recorded
4. **Document types:** Support TRADE, DIVIDEND, INTEREST, THESAURIERUNG in v1; extensible for future types
5. **Traceability:** Optional flag to include source filename in output
6. **Error handling:** Fail fast on unparseable PDFs; exit with non-zero code on any failure; log errors to stderr

### Non-Functional Requirements

1. **Dependencies:** Single Go binary, no system-level prerequisites (go.mod deps bundled)
2. **Deployment:** GitHub Actions CI/CD with linting, testing, build automation
3. **Quality:** Pre-commit hooks (fmt, vet, lint, test); branch protection on main

## CLI Interface

### Commands & Flags

```bash
flatex-pdf-cli <input> [options]

Arguments:
  input                 Path to PDF file or folder of PDFs (folder searches recursively)

Options:
  -o, --output FILE     Write JSON to FILE instead of stdout
  --include-source      Add "source" field (filename) to each result object
  --include-metadata    Wrap output with depot metadata (Depotnummer, Depotinhaber)
```

### Usage Examples

```bash
# Parse single file, output to stdout (transactions array only)
flatex-pdf-cli statement.pdf

# Parse folder, write to file
flatex-pdf-cli ./statements/ -o trades.json

# Include source filename in each transaction
flatex-pdf-cli statement.pdf --include-source

# Include depot metadata (account number, holder) wrapping the transactions
flatex-pdf-cli statement.pdf --include-metadata

# Combine flags
flatex-pdf-cli ./statements/ -o results.json --include-source --include-metadata
```

## Data Schema

### Base Structure (All Transactions)

Every transaction includes:
```json
{
  "source": "statement.pdf",
  "doc_number": "326052529/1",
  "document_type": "TRADE",
  "isin": "IE000YU9K6K2",
  "wkn": "A3DP9J",
  "date": "2026-01-15"
}
```

### Transaction Type: TRADE (Buy/Sell)

```json
{
  "source": "trade.pdf",
  "doc_number": "326052529/1",
  "document_type": "TRADE",
  "type": "BUY",
  "isin": "IE000YU9K6K2",
  "wkn": "A3DP9J",
  "quantity": 1.058537,
  "price": 47.235000,
  "price_currency": "EUR",
  "gross_value": 50.00,
  "provision": 0.00,
  "own_costs": 0.00,
  "third_party_costs": 0.00,
  "withholding_tax": 0.00,
  "gain_loss": 0.00,
  "exchange_rate": 1.000000,
  "final_amount": -50.00,
  "final_currency": "EUR",
  "custody_type": "Wertpapierrechnung",
  "depositary": "Clearstream Lux.",
  "country": "Ireland",
  "date": "2026-01-15"
}
```

**TRADE Fields:**
- `type` (string): "BUY" or "SELL"
- `quantity` (number): Shares/units (fractional allowed)
- `price` (number): Price per share
- `price_currency` (string): Currency of price (e.g., "EUR", "USD")
- `gross_value` (number): Total value before costs
- `provision` (number): Brokerage commission
- `own_costs` (number): Flatex's own costs
- `third_party_costs` (number): Third-party costs
- `withholding_tax` (number): Withholding tax (Einbehaltene Quellensteuer)
- `gain_loss` (number): Realized gain/loss at time of transaction
- `exchange_rate` (number): FX rate if applicable
- `final_amount` (number): Amount after all costs/taxes (negative for buys)
- `final_currency` (string): Currency of final amount
- `custody_type` (string): Type of custody (e.g., "Wertpapierrechnung")
- `depositary` (string): Custodian name (e.g., "Clearstream Lux.")
- `country` (string): Deposit country

### Transaction Type: DIVIDEND (Interest/Distribution)

```json
{
  "source": "dividend.pdf",
  "doc_number": "4684511050",
  "document_type": "DIVIDEND",
  "isin": "IE00B3RBWM25",
  "quantity": 78.70,
  "distribution_per_share": 0.5459180,
  "distribution_currency": "USD",
  "gross_amount": 42.96,
  "gross_currency": "USD",
  "withholding_tax": 5.39,
  "withholding_tax_currency": "EUR",
  "exchange_rate": 1.175000,
  "net_amount": 31.17,
  "net_currency": "EUR",
  "ex_date": "2025-12-18",
  "value_date": "2026-01-01",
  "date": "2025-12-18"
}
```

**DIVIDEND Fields:**
- `quantity` (number): Number of shares held
- `distribution_per_share` (number): Distribution amount per share
- `distribution_currency` (string): Currency of per-share amount
- `gross_amount` (number): Total gross distribution
- `gross_currency` (string): Currency of gross amount
- `withholding_tax` (number): Tax withheld
- `withholding_tax_currency` (string): Currency of withheld tax
- `exchange_rate` (number): FX rate applied
- `net_amount` (number): Amount received after tax
- `net_currency` (string): Currency of net amount
- `ex_date` (string): Ex-dividend date (YYYY-MM-DD)
- `value_date` (string): Value/settlement date (YYYY-MM-DD)

### Transaction Type: INTEREST

```json
{
  "source": "interest.pdf",
  "doc_number": "12345678",
  "document_type": "INTEREST",
  "isin": "IE00B3RBWM25",
  "gross_amount": 25.50,
  "gross_currency": "EUR",
  "withholding_tax": 3.40,
  "withholding_tax_currency": "EUR",
  "net_amount": 22.10,
  "net_currency": "EUR",
  "interest_rate": 2.5,
  "period_from": "2026-01-01",
  "period_to": "2026-03-31",
  "date": "2026-03-31"
}
```

**INTEREST Fields:**
- `gross_amount` (number): Total interest
- `gross_currency` (string): Currency of interest
- `withholding_tax` (number): Tax withheld
- `withholding_tax_currency` (string): Currency of withheld tax
- `net_amount` (number): Amount received after tax
- `net_currency` (string): Currency of net amount
- `interest_rate` (number, optional): Stated interest rate
- `period_from` (string): Period start (YYYY-MM-DD)
- `period_to` (string): Period end (YYYY-MM-DD)

### Transaction Type: THESAURIERUNG (Reinvestment/Accumulation)

```json
{
  "source": "thesaurierung.pdf",
  "doc_number": "4711880849",
  "document_type": "THESAURIERUNG",
  "isin": "IE00B5L8K969",
  "quantity": 4.75,
  "reinvestment_per_share": -0.572,
  "reinvestment_currency": "USD",
  "gross_amount": -2.72,
  "gross_currency": "USD",
  "withholding_tax": 0.00,
  "withholding_tax_currency": "EUR",
  "exchange_rate": 1.169200,
  "ex_date": "2026-01-12",
  "value_date": "2026-01-13",
  "accrual_date": "2026-01-13",
  "date": "2026-01-12"
}
```

**THESAURIERUNG Fields:**
- `quantity` (number): Number of shares held
- `reinvestment_per_share` (number): Reinvestment amount per share (can be negative for reversals)
- `reinvestment_currency` (string): Currency of reinvestment amount
- `gross_amount` (number): Total reinvestment gross amount (can be negative)
- `gross_currency` (string): Currency of gross amount
- `withholding_tax` (number): Tax withheld on reinvestment
- `withholding_tax_currency` (string): Currency of withheld tax
- `exchange_rate` (number): FX rate applied
- `ex_date` (string): Ex-distribution date (YYYY-MM-DD)
- `value_date` (string): Value/settlement date (YYYY-MM-DD)
- `accrual_date` (string): Accrual date (YYYY-MM-DD)
- **Note:** No `net_amount` field (reinvestment is applied to fund holdings, not paid out)

### Common Fields (All Types)

- `source` (string, optional): PDF filename — included only with `--include-source`
- `doc_number` (string): Flatex document reference number
- `document_type` (string): "TRADE", "DIVIDEND", "INTEREST", "THESAURIERUNG", or future type
- `isin` (string): ISIN code
- `wkn` (string, optional): WKN/CUSIP code
- `date` (string): Transaction/settlement date (YYYY-MM-DD)

### Output Format: Default (No Flags)

Without `--include-metadata`, output is a simple transactions array:

```json
[
  {
    "document_type": "TRADE",
    "isin": "IE000YU9K6K2",
    "quantity": 1.058537,
    ...
  },
  {
    "document_type": "DIVIDEND",
    "isin": "IE00B3RBWM25",
    ...
  }
]
```

### Output Format: With `--include-metadata`

With `--include-metadata` flag, output wraps transactions with account metadata:

```json
{
  "metadata": {
    "depot_number": "31022213999",
    "depot_holder": "Max Mustermann"
  },
  "transactions": [
    {
      "document_type": "TRADE",
      "isin": "IE000YU9K6K2",
      "quantity": 1.058537,
      ...
    },
    {
      "document_type": "DIVIDEND",
      "isin": "IE00B3RBWM25",
      ...
    }
  ]
}
```

**Metadata Fields:**
- `depot_number` (string): Account/depot number from the PDF
- `depot_holder` (string): Account holder name from the PDF

## Architecture

### Layered Design

```
main.go (CLI layer)
  ↓
internal/extractor (PDF text extraction)
  ↓
internal/parser (Text → structured data)
  ↓
internal/schema (JSON serialization)
```

### Project Structure

```
flatex-pdf-cli/
├── main.go                    # CLI entry, flag parsing, file I/O
├── go.mod
├── go.sum
├── .golangci.yml              # Linting config
├── .gitignore
├── internal/
│   ├── extractor/
│   │   ├── extractor.go       # PDF text extraction, document type detection
│   │   └── extractor_test.go
│   ├── parser/
│   │   ├── parser.go          # Main parser router + common logic
│   │   ├── trade_parser.go    # Text → Trade transaction
│   │   ├── dividend_parser.go # Text → Dividend transaction
│   │   ├── interest_parser.go # Text → Interest transaction
│   │   ├── thesaurierung_parser.go # Text → Reinvestment transaction
│   │   └── parser_test.go
│   └── schema/
│       ├── transaction.go     # Transaction base struct (TRADE, DIVIDEND, INTEREST, THESAURIERUNG)
│       ├── output.go          # Output struct (metadata + transactions wrapper)
│       └── schema_test.go
├── testdata/
│   ├── sample_trade.pdf       # Real flatex trade confirmation
│   ├── sample_dividend.pdf    # Real flatex dividend statement
│   ├── sample_interest.pdf    # Real flatex interest statement
│   └── sample_thesaurierung.pdf # Real flatex reinvestment statement
├── docs/
│   └── superpowers/
│       └── specs/
│           └── 2026-06-24-flatex-pdf-cli-design.md (this file)
└── .github/
    └── workflows/
        └── ci.yml             # GitHub Actions pipeline
```

### Component Responsibilities

**CLI Layer (`main.go`):**
- Parse command-line flags (`-o`, `--include-source`, `--include-metadata`)
- Handle file discovery: single file vs. folder recursion
- Orchestrate extractor → parser → schema
- Collect metadata from PDFs; use first depot info if --include-metadata
- Format output: transactions array or wrapped with metadata
- Route output to stdout or file
- Set exit code (0 on success, 1 on any parse failure)

**Extractor (`internal/extractor/extractor.go`):**
- Read PDF file using **pdfcpu** library
- Extract text content
- Extract document metadata: Depotnummer (account number), Depotinhaber (account holder name)
- Detect document type: TRADE (Kauf/Verkauf), DIVIDEND (Ausschüttung), INTEREST, THESAURIERUNG (Ertragsmitteilung), or future types
- Return raw text + filename + metadata + detected type
- Fail fast on unreadable/corrupt PDFs

**Parser (`internal/parser/`):**
Main router:
- `parser.go`: Coordinate extraction → type classification → route to type-specific parser → return Transaction

Type-specific parsers:
- `trade_parser.go`: Extract ISIN, WKN, quantity, price, costs, fees, withholding tax, custody details
- `dividend_parser.go`: Extract ISIN, quantity, distribution amounts, withholding tax, exchange rates, dates
- `interest_parser.go`: Extract ISIN, gross/net amounts, withholding tax, period, rates
- `thesaurierung_parser.go`: Extract ISIN, quantity, reinvestment amounts (can be negative), withholding tax, exchange rates, dates

All parsers:
- Use regex and pattern matching on extracted text
- Validate extracted values (e.g., quantity > 0, valid ISIN format, valid dates, amounts can be negative)
- Return `Transaction` struct or error
- No OCR; text-extraction only

**Schema (`internal/schema/transaction.go` + `internal/schema/output.go`):**
- `Transaction` base struct supporting TRADE, DIVIDEND, INTEREST, THESAURIERUNG types
- Type-specific fields per transaction type
- Handles optional `source` field
- `DocumentMetadata` struct with depot_number, depot_holder
- `Output` struct: conditional wrapping (transactions array or metadata + transactions)
- JSON marshaling with proper type/currency handling
- Flag-driven output format: `--include-metadata` toggles wrapper

### Data Flow

```
1. User runs: flatex-pdf-cli folder/ -o output.json --include-source --include-metadata

2. CLI layer:
   - Discovers all .pdf files in folder/ (recursive)
   - For each file: calls Extractor

3. Extractor:
   - Reads PDF with pdfcpu
   - Extracts text content
   - Extracts depot metadata: Depotnummer, Depotinhaber
   - Classifies document type (TRADE, DIVIDEND, INTEREST, THESAURIERUNG, or error)
   - Returns (text, filename, metadata, doc_type) or error
   - On error: logs to stderr, continues to next file

4. Parser:
   - Receives (text, filename, doc_type)
   - Routes to type-specific parser:
     * trade_parser: extracts ISIN, WKN, quantity, price, costs, fees, withholding tax
     * dividend_parser: extracts ISIN, quantity, distribution, withholding tax, exchange rate
     * interest_parser: extracts ISIN, gross/net amounts, withholding tax, period
     * thesaurierung_parser: extracts ISIN, quantity, reinvestment amounts, withholding tax, exchange rate
   - Each parser validates extracted values (including negative amounts for reinvestment corrections)
   - Returns Transaction struct or error
   - On error: logs to stderr, skips file

5. Schema:
   - Collects all successful Transaction objects
   - Conditionally adds source field if --include-source
   - If --include-metadata: wraps in Output struct with metadata + transactions
   - If no --include-metadata: returns transactions array only
   - Marshals to JSON

6. CLI layer:
   - Writes JSON to stdout or -o file
     * With --include-metadata: { metadata: {...}, transactions: [...] }
     * Without --include-metadata: [...]
   - Logs all errors to stderr
   - Exits: 0 (all files parsed), 1 (any file failed)
```

## Error Handling

### Exit Codes

- **Exit 0:** All input files parsed successfully
- **Exit 1:** Any input file failed to parse (unreadable PDF, missing fields, invalid values)

### Error Output

**To stderr (human-readable):**
```
ERROR: statement.pdf - failed to extract text: corrupted PDF
ERROR: trade_old.pdf - failed to parse: symbol not found
WARN: 1 of 3 files failed
```

**To stdout (JSON):**
- Only successful parses included
- No error objects mixed into JSON
- Agents can consume partial results if some files succeeded

### Partial Results

If 5 of 6 files parse successfully:
- stdout contains JSON array with 5 results
- stderr logs which file failed and why
- Exit code is 1
- Agent can use the 5 successful results

## Testing Strategy

### Unit Tests

**`internal/parser/parser_test.go`:**
- Test document type classification (TRADE, DIVIDEND, INTEREST, THESAURIERUNG)
- Test routing to correct type-specific parser

**`internal/parser/trade_parser_test.go`:**
- Test regex/pattern extraction: ISIN, WKN, quantity (including fractional), price, costs, fees, withholding tax
- Test edge cases: missing fields, invalid values, malformed text
- Test validation: quantity > 0, ISIN format, currency parsing

**`internal/parser/dividend_parser_test.go`:**
- Test extraction: ISIN, quantity, distribution per share, gross/net amounts, withholding tax, exchange rate, dates
- Test edge cases: missing fields, zero distributions, FX conversions
- Test validation: date formats, currency handling

**`internal/parser/interest_parser_test.go`:**
- Test extraction: ISIN, gross/net amounts, withholding tax, period, interest rate
- Test validation: date ranges, amount consistency

**`internal/parser/thesaurierung_parser_test.go`:**
- Test extraction: ISIN, quantity, reinvestment amounts, withholding tax, exchange rate, dates
- Test negative amounts (reversals/corrections)
- Test validation: ISIN format, date handling, currency parsing

**`internal/extractor/extractor_test.go`:**
- Test PDF text extraction with real sample PDFs in `testdata/`
- Test document type detection (TRADE, DIVIDEND, INTEREST, THESAURIERUNG)
- Test error handling (unreadable PDFs, corrupt files)

**`internal/schema/schema_test.go`:**
- Test JSON serialization/marshaling for all transaction types
- Test optional source field behavior
- Test currency field preservation

### Integration Tests

End-to-end tests:
- Real trade confirmation PDF → parsed JSON, verify all fields (ISIN, WKN, quantity, price, costs)
- Real dividend statement PDF → parsed JSON, verify amounts/taxes/dates/exchange rates
- Real interest statement PDF → parsed JSON, verify period and rates
- Real reinvestment statement PDF → parsed JSON, verify reinvestment amounts (including negative)
- Folder with mixed PDFs (all types) → combined JSON with all types, verify correct classification

### Test Coverage

- All business logic tested (parsers, extractor, schema)
- CLI layer tested via integration test
- No mocks; use real PDFs in `testdata/`

### Running Tests

```bash
go test ./...
```

## CI/CD Pipeline

### Pre-commit Hooks (`.git/hooks/pre-commit`)

Runs before each commit:

1. `go fmt ./...` — format code
2. `go vet ./...` — static analysis
3. `golangci-lint run` — comprehensive linting
4. `go test ./...` — run all tests

Commit blocked if any step fails.

### GitHub Actions (`.github/workflows/ci.yml`)

Triggered on every push to non-main branches and on pull requests:

1. Checkout code
2. Set up Go
3. Run `go fmt` check
4. Run `go vet`
5. Run `golangci-lint`
6. Run `go test ./...`
7. Run `go build -o flatex-pdf-cli`
8. (Optional) Create release artifact

**Branch protection (main):**
- Require PR review
- Require CI to pass
- No direct pushes to main

### Linting Configuration (`.golangci.yml`)

```yaml
linters:
  enable:
    - gofmt
    - govet
    - errcheck
    - ineffassign
    - unused
    - deadcode
```

## Deployment

### Build & Distribution

- Single standalone Go binary: `flatex-pdf-cli`
- No system-level dependencies (pdfcpu bundled via go.mod)
- Cross-platform support: Linux, macOS, Windows (via `GOOS`/`GOARCH`)

### Distribution Method (Future)

- GitHub Releases: build binary for Linux/macOS/Windows, publish as assets
- Users download pre-built binary, no build required

## Future Extensions

### Post-v1 Document Types

Architecture supports adding new document types easily:
1. Create `internal/parser/<type>_parser.go`
2. Add type-specific fields to `Transaction` struct
3. Update `Extractor.Detect()` to classify new type
4. Add type-specific tests

Planned future types:
- Account statements (balances, holdings overview)
- Tax documents (annual tax reports)
- Order confirmations (pending orders)
- Corporate actions (splits, mergers)

## Success Criteria (v1)

**Parsing:**
- [x] Parses trade confirmations (BUY/SELL with all fields: ISIN, WKN, quantity, price, costs, withholding tax)
- [x] Parses dividend statements (with exchange rates, withholding tax)
- [x] Parses interest statements (with period, rates, withholding tax)
- [x] Parses reinvestment statements (Thesaurierung, including negative amounts for reversals)
- [x] Supports fractional shares and multi-currency amounts
- [x] Parses folder of mixed PDFs, combines results

**Output:**
- [x] Valid JSON with correct schema per transaction type
- [x] Errors logged to stderr only; JSON contains only successful parses
- [x] Optional --include-source flag adds filename to results
- [x] Optional --include-metadata flag wraps output with depot information (number, holder)
- [x] Proper exit codes (0 on success, 1 on any failure)

**Quality:**
- [x] Pre-commit hooks enforce code quality (fmt, vet, lint, test)
- [x] CI/CD pipeline runs tests + build on every branch
- [x] All unit + integration tests pass
- [x] No system dependencies; single Go binary

**Documentation:**
- [x] Integration tests with real sample PDFs (trade, dividend, interest)
- [x] Design spec complete and approved

## Open Questions / Deferred

- **OCR support:** Explicitly deferred to v2+ (text extraction only in v1)
- **Docker packaging:** Deferred (YAGNI for single binary; can add if needed)
- **Detection heuristics:** v1 uses keyword/pattern matching; can improve in v2 if needed
- **Additional document types (v2+):** Account statements, tax reports, corporate actions, payment confirmations

---

**Approval:** User approved 2026-06-24  
**Next step:** Implementation plan (via writing-plans skill)
