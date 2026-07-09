# flatex-pdf-cli

A command-line tool for extracting transaction data from flatex (a German online broker) PDF documents. Parses trade confirmations, dividend notices, interest notices, accumulation (Ertragsmitteilung) notices, order confirmations, and crypto settlements into structured JSON format.

> **Disclaimer:** This is an independent, unofficial open-source project. It is
> **not** affiliated with, endorsed by, sponsored by, or in any way associated
> with flatexDEGIRO AG, flatex, DEGIRO, or any of their subsidiaries. "flatex"
> and "flatexDEGIRO" are trademarks of their respective owners and are used
> here only to describe the document format this tool parses. Use at your own
> risk; always verify extracted data against the original documents.

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

## Export Formats

By default the CLI emits JSON. Use `-format` to emit CSV instead:

- `-format csv` — one row per transaction, every parsed field as a column. Good for spreadsheets or your own scripts.
- `-format pp` — two CSVs shaped for [Portfolio Performance](https://www.portfolio-performance.info/)'s CSV import: `<base>-portfolio.csv` (buy/sell trades) and `<base>-accounts.csv` (dividends, interest, withheld tax on accumulating funds). Requires `-o <base>` since two files are written.

```bash
flatex-pdf-cli -format csv -o transactions.csv ~/Downloads/flatex
flatex-pdf-cli -format pp -o portfolio ~/Downloads/flatex
# writes portfolio-portfolio.csv and portfolio-accounts.csv
```

Then in Portfolio Performance: **File > Import > CSV Files**, pick the "Portfolio Transactions" or "Account Transactions" import, and use the matching CSV. PP's CSV import lets you re-map any column, so if a column isn't auto-recognized, map it by hand — after the first import, save the mapping as a template so later imports are one click.

**Running PP in German?** Add `-lang de` to get German column headers (`Datum`, `Wert`, `Stück`, …), German `Typ` values (`Kauf`, `Verkauf`, `Dividende`, `Zinsen`, `Steuern`), a semicolon (`;`) field separator, and comma (`,`) as the decimal separator (e.g. `1,478695`, not `1.478695`) — all German-locale conventions, and all defaults PP's own import wizard already assumes on a German-locale install. PP's CSV column auto-recognition is locale-sensitive with no English fallback, so a German-locale PP install won't auto-map English headers at all — `-lang de` is what makes auto-recognition work without manually mapping every column or number format.

```bash
flatex-pdf-cli -format pp -lang de -o portfolio ~/Downloads/flatex
```

**Before bulk-importing, test-import a handful of rows first** and check the resulting positions/cash balance against a statement you trust. The column mapping above is our best-effort read of PP's documented CSV fields; it hasn't been validated against every edge case (e.g. multi-currency trades, partial fills).

## Organize Downloads

Automatically sort flatex PDFs from your Downloads folder into a structured archive.
Requires `jq` (`brew install jq` on macOS).

### One-time paste

Edit the `TARGET` line, then paste the whole block into your terminal:

```bash
TARGET=~/Documents/flatex-organized
find ~/Downloads -name '*.pdf' | while IFS= read -r pdf; do
  json=$(flatex-pdf-cli -include-metadata -quiet "$pdf" 2>/dev/null) || continue
  account=$(jq -r '.metadata.depot_number // "unknown"' <<<"$json")
  date=$(jq -r '.transactions[0].date' <<<"$json")
  type=$(jq -r '.transactions[0].document_type' <<<"$json")
  dest="$TARGET/$account"
  mkdir -p "$dest"
  cp "$pdf" "$dest/${date}_${type}_$(basename "$pdf")"
  echo "  -> $dest/${date}_${type}_$(basename "$pdf")"
done
```

### Reusable shell function

Add this to your `~/.zshrc` or `~/.bashrc` to call it by name:

```bash
flatex-organize() {
  local src="${1:-$HOME/Downloads}"
  local target="${2:?Usage: flatex-organize [source_dir] <target_dir>}"

  find "$src" -name '*.pdf' | while IFS= read -r pdf; do
    json=$(flatex-pdf-cli -include-metadata -quiet "$pdf" 2>/dev/null) || continue
    account=$(jq -r '.metadata.depot_number // "unknown"' <<<"$json")
    date=$(jq -r '.transactions[0].date' <<<"$json")
    type=$(jq -r '.transactions[0].document_type' <<<"$json")
    dest="$target/$account"
    mkdir -p "$dest"
    cp "$pdf" "$dest/${date}_${type}_$(basename "$pdf")"
    echo "  -> $dest/${date}_${type}_$(basename "$pdf")"
  done
}
```

```bash
flatex-organize ~/Documents/flatex-organized            # source defaults to ~/Downloads
flatex-organize ~/Downloads ~/Documents/flatex-organized
```

### Result layout

```
flatex-organized/
  31022213792/
    2025-09-16_TRADE_20250916_KaufFondsZertifikate_31022213792_517614092.pdf
    2025-10-02_DIVIDEND_20251002_Fondsertragsausschuettung_31022213792_528846930.pdf
```

Non-flatex PDFs in the source directory are silently skipped.

## Supported Document Types

The tool automatically detects and parses the following flatex document types:

| Type | Status | Description |
|---|---|---|
| TRADE | ✅ Full | Buy/sell confirmations (Wertpapierabrechnung Kauf/Verkauf) with pricing, costs, and gain/loss |
| DIVIDEND | ✅ Full | Dividend payment statements (Ausschüttung) with distribution details and withholding tax |
| INTEREST | ✅ Full | Interest payment notices (Zinsen) on cash accounts |
| ACCUMULATING | ✅ Full | Reinvestment/accumulation notices (Ertragsmitteilung, thesaurierende Fonds) |
| ORDER | 🟡 Partial | Order confirmations (Sammelauftragsbestätigung); one record per pending order — see limitations |
| CRYPTO | ✅ Full | Crypto buy/sell settlements (Sammelabrechnung Kryptowerte) |
| SAVINGSPLAN | ✅ Full | Annual savings-plan settlement (Sammelabrechnung aus); one transaction per executed order row |

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

## Known Limitations

- **ORDER `security_name` includes the execution venue.** gxpdf does not always
  put a space between the Bezeichnung and Ausf.platz/-art columns (e.g.
  `"GLOBAL X COPPER MINERS ETXETRA"`), so the venue is left attached to the name
  rather than split unreliably. Order confirmations therefore do **not** populate
  a separate `execution_venue`.
- **Metadata extraction (`depot_holder`, `depot_number`)** can be empty or noisy
  on documents whose layout places the value far from its label.
- **Account number (`Konto Nr.`)** is matched at a fixed length (11 digits) to
  work around a page-break run-on in text extraction; non-standard lengths won't
  match. (The depot number is matched at any length.)
- **Synthetic test fixtures** in `testdata/` are visually faithful and PII-free,
  but the redaction re-inserts text out of reading order, so the ORDER and CRYPTO
  fixtures only exercise *type detection*, not full field extraction (the parsers
  are verified against real documents instead).
- **SAVINGSPLAN WKN** is not present in Sammelabrechnung documents; the `wkn` field will be empty for these transactions.

Additional document types (e.g. tax reports) will be added as samples become available.

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

### Metadata Wrapper (`-include-metadata`)

With `-include-metadata`, the transaction list is wrapped in an object with depot metadata:

```json
{
  "metadata": {
    "depot_number": "1234567890",
    "depot_holder": "Max Mustermann",
    "account_number": "9876543210"
  },
  "transactions": [
    { "document_type": "TRADE", "isin": "DE0005140008", "date": "2024-06-15" }
  ]
}
```

Transaction objects are as shown above (abbreviated here).

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

### Test Fixtures

The fixtures in `testdata/` are real flatex PDFs with the PII redacted and
replaced in place with synthetic values, so they behave exactly like production
documents. How they were made — and why naive synthetic PDFs don't work — is
covered in [Your AI's Test Fixtures Are Lying to You. Make real-world synthetic PDF files, PII safe!](https://pub.automatetherest.com/your-ais-test-fixtures-are-lying-to-you-0bc4f4ec7604).

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

Optional: `pip install pre-commit && pre-commit install` — runs `go fmt`,
`go vet`, and `go test` on every commit (config in `.pre-commit-config.yaml`).

## Project Structure

CLI entry point in `main.go`; PDF text extraction in `internal/extractor`,
document detection and parsing in `internal/parser`, output types in
`internal/schema`. Agent skill in `skill/`, PII-free sample PDFs in `testdata/`.

## Dependencies

- [gxpdf](https://github.com/coregx/gxpdf) v0.8.2 — PDF text extraction

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
