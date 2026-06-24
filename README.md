# flatex-pdf-cli

A command-line tool for extracting transaction data from flatex (a German online broker) PDF documents. Parses account statements, trade confirmations, dividend notices, interest notices, and thesaurierung (reinvestment) documents into structured JSON format.

## Features

- **Multiple Document Types**: Supports trade confirmations, dividend statements, interest notices, and thesaurierung documents
- **Batch Processing**: Process single PDF files or entire directories recursively
- **Structured Output**: Extract data into JSON format with comprehensive transaction details
- **Metadata Support**: Optionally include depot number and holder information in output
- **Source Tracking**: Optionally add source filename to each transaction for auditing

## Installation

### Download Pre-built Binary

Pre-built binaries are available on the [releases page](https://github.com/welworx/flatex-pdf-cli/releases).

```bash
# macOS
curl -L https://github.com/welworx/flatex-pdf-cli/releases/download/v0.1.0/flatex-pdf-cli-darwin-amd64 -o flatex-pdf-cli
chmod +x flatex-pdf-cli
```

### Build from Source

Requirements: Go 1.26.4 or later

```bash
git clone https://github.com/welworx/flatex-pdf-cli.git
cd flatex-pdf-cli
go build -o flatex-pdf-cli
```

## Usage

### Basic Usage

Process a single PDF file and output JSON to stdout:

```bash
./flatex-pdf-cli path/to/statement.pdf
```

Process a directory containing multiple PDFs:

```bash
./flatex-pdf-cli path/to/documents/
```

### Flags

- `-o FILE` — Write output to a file instead of stdout
- `--include-source` — Add source filename to each transaction (useful when processing multiple files)
- `--include-metadata` — Wrap output with depot metadata (depot number and holder)
- `-h`, `--help` — Show help message

### Examples

Save output to file:

```bash
./flatex-pdf-cli -o output.json path/to/documents/
```

Include depot metadata in output:

```bash
./flatex-pdf-cli --include-metadata path/to/trade-confirmation.pdf
```

Include source filename with transactions (for audit trail):

```bash
./flatex-pdf-cli --include-source -o transactions.json path/to/documents/
```

Combine flags:

```bash
./flatex-pdf-cli --include-source --include-metadata -o output.json path/to/documents/
```

## Supported Document Types

The tool automatically detects and parses the following flatex document types:

- **TRADE** — Buy/sell confirmations with pricing, costs, and gain/loss information
- **DIVIDEND** — Dividend payment statements with distribution details and withholding tax
- **INTEREST** — Interest payment notices on cash accounts
- **THESAURIERUNG** — Reinvestment notices for dividend reinvestment

## Output Format

### Transaction Object

All extracted transactions are returned as JSON objects with the following structure:

```json
{
  "source": "filename.pdf",
  "doc_number": "12345678",
  "document_type": "TRADE",
  "isin": "DE0005140008",
  "wkn": "514000",
  "date": "2024-06-15",
  "type": "BUY",
  "quantity": 10.0,
  "price": 25.50,
  "price_currency": "EUR",
  "gross_value": 255.00,
  "provision": 5.50,
  "own_costs": 1.00,
  "third_party_costs": 0.00,
  "withholding_tax": 0.00,
  "gain_loss": 0.00,
  "exchange_rate": 1.0,
  "final_amount": 248.50,
  "final_currency": "EUR",
  "custody_type": "DEPOT",
  "depositary": "flatex",
  "country": "DE"
}
```

### Common Fields (All Transactions)

- `source` — Source filename (only if `--include-source` flag is used)
- `doc_number` — Document reference number
- `document_type` — Type of document (TRADE, DIVIDEND, INTEREST, THESAURIERUNG)
- `isin` — ISIN of the security
- `wkn` — German securities identification number (if available)
- `date` — Transaction date in YYYY-MM-DD format

### Trade-Specific Fields

- `type` — BUY or SELL
- `quantity` — Number of shares/units
- `price` — Price per unit
- `price_currency` — Currency of price
- `gross_value` — Total transaction value before costs
- `provision` — Broker commission/fee
- `own_costs` — Costs charged by the investor's bank
- `third_party_costs` — Costs charged by third parties
- `withholding_tax` — Tax withheld on transaction
- `gain_loss` — Capital gain or loss (sell transactions)
- `exchange_rate` — Currency exchange rate (if applicable)
- `final_amount` — Net amount after all costs and taxes
- `final_currency` — Currency of final amount
- `custody_type` — Type of custody (DEPOT, etc.)
- `depositary` — Depositary institution name
- `country` — Country code of security

### Dividend-Specific Fields

- `distribution_per_share` — Dividend per unit held
- `distribution_currency` — Currency of dividend
- `gross_amount` — Total dividend before withholding
- `gross_currency` — Currency of gross amount
- `withholding_tax_currency` — Currency of withholding tax amount
- `net_amount` — Dividend after withholding tax
- `net_currency` — Currency of net amount
- `ex_date` — Ex-dividend date
- `value_date` — Value date for the payment

### Interest-Specific Fields

- `interest_rate` — Interest rate percentage
- `period_from` — Start of interest period
- `period_to` — End of interest period

### Thesaurierung-Specific Fields

- `reinvestment_per_share` — Reinvestment amount per unit
- `reinvestment_currency` — Currency of reinvestment
- `accrual_date` — Date reinvestment was accrued

### Full Output Example (with metadata)

```json
{
  "metadata": {
    "depot_number": "1234567890",
    "depot_holder": "Max Mustermann"
  },
  "transactions": [
    {
      "doc_number": "12345678",
      "document_type": "TRADE",
      "isin": "DE0005140008",
      "wkn": "514000",
      "date": "2024-06-15",
      "type": "BUY",
      "quantity": 10.0,
      "price": 25.50,
      "price_currency": "EUR",
      "gross_value": 255.00,
      "provision": 5.50,
      "final_amount": 248.50,
      "final_currency": "EUR"
    }
  ]
}
```

## Development

### Running Tests

Run all tests:

```bash
go test ./...
```

Run tests for a specific package with verbose output:

```bash
go test -v ./internal/parser
```

### Code Quality

The project uses `golangci-lint` for linting. Configuration is in `.golangci.yml`.

Format code:

```bash
go fmt ./...
```

Run linter checks:

```bash
golangci-lint run
```

### Pre-commit Hooks

Optional: Set up pre-commit hooks to automatically format, lint, and test before commits:

```bash
# Install pre-commit framework
pip install pre-commit

# Install the git hooks
pre-commit install

# Run hooks on all files
pre-commit run --all-files
```

The `.pre-commit-config.yaml` file runs `go fmt`, `go vet`, and `go test` automatically on commits.

## Project Structure

```
flatex-pdf-cli/
├── main.go                 # CLI entry point and PDF discovery
├── go.mod                  # Go module definition
├── go.sum                  # Dependency checksums
├── .golangci.yml          # Linter configuration
├── .pre-commit-config.yaml # Pre-commit hooks configuration
├── README.md              # This file
├── .gitignore             # Git ignore rules
├── internal/
│   ├── extractor/         # PDF text extraction
│   │   ├── extractor.go
│   │   └── extractor_test.go
│   ├── parser/            # Document type detection and parsing
│   │   ├── parser.go
│   │   └── parser_test.go
│   └── schema/            # Data structures and validation
│       ├── transaction.go
│       ├── output.go
│       └── schema_test.go
└── docs/                  # Additional documentation
```

## Dependencies

- [pdfcpu](https://github.com/pdfcpu/pdfcpu) v0.13.0 — PDF text extraction

## License

MIT — See LICENSE file for details

## Contributing

Contributions are welcome! Please ensure:

1. All tests pass: `go test ./...`
2. Code is formatted: `go fmt ./...`
3. Linter passes: `golangci-lint run`
4. Commit messages follow conventional commits format

## Support

For issues, feature requests, or questions, please open an issue on GitHub.
