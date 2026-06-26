# flatex-pdf-cli

A command-line tool for extracting transaction data from flatex (a German online broker) PDF documents. Parses trade confirmations, dividend notices, interest notices, accumulation (Ertragsmitteilung) notices, order confirmations, and crypto settlements into structured JSON format.

## Features

- **Multiple Document Types**: Supports trade confirmations, dividend statements, interest notices, accumulation notices, order confirmations, and crypto settlements
- **Batch Processing**: Process single PDF files or entire directories recursively
- **Structured Output**: Extract data into JSON format with comprehensive transaction details
- **Metadata Support**: Optionally include depot number and holder information in output
- **Source Tracking**: Optionally add source filename to each transaction for auditing

## Installation

See [skill/INSTALL.md](skill/INSTALL.md) for detailed installation instructions (go install, build from source, pre-built binaries).

## Use with AI Agents (Claude Code skill)

This repo ships a ready-made skill in [`skill/SKILL.md`](skill/SKILL.md) so AI
coding agents can call the CLI to process flatex PDFs. Install it once:

```bash
# install the CLI, then the skill
go install github.com/welworx/flatex-pdf-cli@latest
git clone https://github.com/welworx/flatex-pdf-cli.git /tmp/flatex-pdf-cli
mkdir -p ~/.claude/skills/flatex-pdf-cli
cp /tmp/flatex-pdf-cli/skill/SKILL.md ~/.claude/skills/flatex-pdf-cli/
```

The agent then runs `flatex-pdf-cli -quiet -include-metadata <path>` and
consumes the JSON. See the skill file for the full contract.

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

- `-o FILE` — Output file (stdout if not provided)
- `-include-source` — Add source filename to each transaction
- `-include-metadata` — Wrap output with depot metadata
- `-quiet` — Hide skipped/problematic files; emit only valid JSON
- `-version` — Show version and exit

When given a directory, the tool processes every `.pdf` it finds. A file it
cannot parse is reported on stderr and **skipped** — the rest still produce
output, so one bad document never aborts the batch. Use `-quiet` to suppress
the skip messages and get pure JSON on stdout.

### Examples

Save output to file:

```bash
./flatex-pdf-cli -o output.json path/to/documents/
```

Include depot metadata in output:

```bash
./flatex-pdf-cli -include-metadata path/to/trade-confirmation.pdf
```

Include source filename with transactions (for audit trail):

```bash
./flatex-pdf-cli -include-source -o transactions.json path/to/documents/
```

Combine flags:

```bash
./flatex-pdf-cli -include-source -include-metadata -o output.json path/to/documents/
```

## Supported Document Types

The tool automatically detects and parses the following flatex document types:

- **TRADE** — Buy/sell confirmations (Wertpapierabrechnung) with pricing, costs, and gain/loss information
- **DIVIDEND** — Dividend payment statements with distribution details and withholding tax
- **INTEREST** — Interest payment notices on cash accounts
- **ACCUMULATING** — Reinvestment/accumulation notices (Ertragsmitteilung, thesaurierende Fonds)
- **ORDER** — Order confirmations (Sammelauftragsbestätigung); one record per pending order
- **CRYPTO** — Crypto buy/sell settlements (Sammelabrechnung Kryptowerte)

## Language Support

**German PDFs only.** All document-type detection and field extraction is keyed
to German labels (`Wertpapierabrechnung`, `Ertragsmitteilung`, `Valuta`,
`Devisenkurs`, …) and the German number format (`1.234,56`).

- ✅ **German** flatexDEGIRO statements — fully supported.
- ❌ **English** (or any non-German) statements — **not implemented.** Such files
  are detected and rejected with an error rather than silently mis-parsed:
  `unsupported document language: only German flatex PDFs are implemented`.

Numbers are parsed format-agnostically (both `1.234,56` and `1,234.56` are
accepted), so the German requirement is purely about field labels and keywords.
Adding English support requires a real English sample to map the English labels.

## Implementation Status

| Document type | Status | Notes |
|---|---|---|
| TRADE | ✅ Full | Wertpapierabrechnung Kauf/Verkauf |
| DIVIDEND | ✅ Full | Ausschüttung |
| INTEREST | ✅ Full | Zinsen |
| ACCUMULATING | ✅ Full | Ertragsmitteilung (thesaurierende Fonds) |
| CRYPTO | ✅ Full | Sammelabrechnung Kryptowerte |
| ORDER | 🟡 Partial | Sammelauftragsbestätigung — see limitations below |

## Known Limitations

- **ORDER `security_name` includes the execution venue.** gxpdf does not always
  put a space between the Bezeichnung and Ausf.platz/-art columns (e.g.
  `"GLOBAL X COPPER MINERS ETXETRA"`), so the venue is left attached to the name
  rather than split unreliably. Order confirmations therefore do **not** populate
  a separate `execution_venue`.
- **German only** — non-German PDFs are rejected (see Language Support).
- **Metadata extraction (`depot_holder`, `depot_number`)** can be empty or noisy
  on documents whose layout places the value far from its label.
- **Depot/account numbers** are matched at a fixed length (11 digits) to work
  around a page-break run-on in text extraction; non-standard lengths won't match.
- **Synthetic test fixtures** in `testdata/` are visually faithful and PII-free,
  but the redaction re-inserts text out of reading order, so the ORDER and CRYPTO
  fixtures only exercise *type detection*, not full field extraction (the parsers
  are verified against real documents instead).

## Roadmap / TODO

- [ ] Split ORDER `security_name` / `execution_venue` reliably (needs positional
      extraction, not gxpdf's flattened text).
- [ ] English-language document support (needs a real English sample).
- [ ] More robust `depot_holder` / `depot_number` extraction.
- [ ] Reading-order-preserving redaction so synthetic ORDER/CRYPTO fixtures parse
      end-to-end.
- [ ] Additional document types as samples become available (e.g. tax reports).

## Output Format

### Transaction Object

All extracted transactions are returned as JSON objects with the following structure:

```json
{
  "source": "filename.pdf",
  "order_number": "999888777/1",
  "transaction_number": "8887776665",
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
  "country": "DE",
  "execution_venue": "XETRA"
}
```

### Common Fields (All Transactions)

- `source` — Source filename (only if `-include-source` flag is used)
- `order_number` — Order number (Auftragsnummer), if present
- `transaction_number` — Tax-report transaction number (Transaktion-Nr.), if present
- `document_type` — Type of document (TRADE, DIVIDEND, INTEREST, ACCUMULATING, ORDER, CRYPTO)
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
- `execution_venue` — Execution venue/type (Ausf.platz/-art), e.g. XETRA

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

### Accumulating-Specific Fields

- `reinvestment_per_share` — Reinvestment amount per unit
- `reinvestment_currency` — Currency of reinvestment
- `accrual_date` — Date reinvestment was accrued

### Order-Specific Fields (Sammelauftragsbestätigung)

- `security_name` — Bezeichnung (may include the execution venue, which the PDF column layout does not always separate)
- `limit` — Limit price of the order
- `valid_until` — Order validity date (Gültig bis)

### Crypto-Specific Fields (Sammelabrechnung Kryptowerte)

- `security_name` — Crypto asset name (e.g. BITCOIN); crypto positions have no ISIN
- `custody_type` — Verwahrart (e.g. Kryptoverwahrung)
- `depositary` — Kryptoverwahrer (e.g. Tangany GmbH)

### Full Output Example (with metadata)

```json
{
  "metadata": {
    "depot_number": "1234567890",
    "depot_holder": "Max Mustermann",
    "account_number": "9876543210"
  },
  "transactions": [
    {
      "order_number": "999888777/1",
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

Licensed under the [MIT License](LICENSE). You're free to use, modify, and
redistribute it, including for commercial purposes, provided the copyright
notice is retained. The software is provided "as is", without warranty of any
kind and with no liability on the author's part — see the LICENSE file for the
full disclaimer.

## Contributing

Contributions are welcome! Please ensure:

1. All tests pass: `go test ./...`
2. Code is formatted: `go fmt ./...`
3. Linter passes: `golangci-lint run`
4. Commit messages follow conventional commits format

## Support

For issues, feature requests, or questions, please open an issue on GitHub.
