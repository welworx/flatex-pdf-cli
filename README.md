# flatex-pdf-cli

[![CI](https://github.com/welworx/flatex-pdf-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/welworx/flatex-pdf-cli/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/welworx/flatex-pdf-cli)](https://goreportcard.com/report/github.com/welworx/flatex-pdf-cli)
[![Release](https://img.shields.io/github/v/release/welworx/flatex-pdf-cli)](https://github.com/welworx/flatex-pdf-cli/releases/latest)
[![Go Version](https://img.shields.io/github/go-mod/go-version/welworx/flatex-pdf-cli)](go.mod)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

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

## Supported Documents

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

**German PDFs only** — non-German statements are rejected with an error (see
[Known Limitations](#known-limitations)).

## Installation

See [skill/INSTALL.md](skill/INSTALL.md) for detailed installation instructions (go install, build from source, pre-built binaries).

## Usage

Process a single PDF file (JSON to stdout) or a directory of PDFs:

```bash
flatex-pdf-cli path/to/statement.pdf
flatex-pdf-cli path/to/documents/
```

### Flags

- `-o FILE` — Output file (stdout if not provided)
- `-format FORMAT` — Output format: `json` (default), `csv`, or `pp` (see Export Formats)
- `-lang LANG` — Language for `pp` output: `en` (default) or `de`
- `-include-source` — Add source filename to each transaction
- `-include-metadata` — Wrap output with depot metadata
- `-quiet` — Hide skipped/problematic files; emit only valid JSON
- `-version` — Show version and exit

When given a directory, the tool processes every `.pdf` it finds. A file it
cannot parse is reported on stderr and **skipped** — the rest still produce
output, so one bad document never aborts the batch. Use `-quiet` to suppress
the skip messages and get pure JSON on stdout.

### Examples

```bash
# Save output to file
flatex-pdf-cli -o output.json path/to/documents/

# Include depot metadata in output
flatex-pdf-cli -include-metadata path/to/trade-confirmation.pdf

# Include source filename with transactions (for audit trail)
flatex-pdf-cli -include-source -o transactions.json path/to/documents/

# Combine flags
flatex-pdf-cli -include-source -include-metadata -o output.json path/to/documents/
```

## Export Formats

By default the CLI emits JSON. Use `-format` to emit CSV instead:

- `-format csv` — one row per transaction, every parsed field as a column. Good for spreadsheets or your own scripts.
- `-format pp` — two CSVs shaped for [Portfolio Performance](https://www.portfolio-performance.info/)'s CSV import. Requires `-o <base>` since two files are written.

```bash
flatex-pdf-cli -format csv -o transactions.csv ~/Downloads/flatex
flatex-pdf-cli -format pp -lang de -o portfolio ~/Downloads/flatex
# writes portfolio-portfolio.csv and portfolio-accounts.csv
```

Use `-lang de` if your Portfolio Performance runs in German — it emits German
headers, `Typ` values, and number format so PP's locale-sensitive column
auto-recognition works. Import walkthrough, `-lang de` details, and caveats:
**[docs/portfolio-performance.md](docs/portfolio-performance.md)**.

## Organize Downloads

Sort flatex PDFs from your Downloads folder into a structured archive — one
folder per depot, files renamed by date and document type — using the CLI's
JSON output and `jq`. Ready-to-paste shell recipes:
**[docs/organize-downloads.md](docs/organize-downloads.md)**.

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

## JSON Reference

Each transaction is a flat JSON object; `-include-metadata` wraps the list
with depot metadata:

```json
{
  "metadata": {
    "depot_number": "1234567890",
    "depot_holder": "Max Mustermann",
    "account_number": "9876543210"
  },
  "transactions": [
    {
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

Full field reference (common, trade, dividend, interest, accumulating, order,
and crypto fields): **[docs/output-format.md](docs/output-format.md)**.

## Known Limitations

- **German PDFs only.** Document-type detection and field extraction are keyed
  to German labels (`Wertpapierabrechnung`, `Valuta`, `Devisenkurs`, …);
  non-German statements are detected and rejected with an error rather than
  silently mis-parsed. Numbers are parsed format-agnostically (both `1.234,56`
  and `1,234.56` are accepted), so the restriction is purely about field
  labels — English support needs a real English sample to map the labels.
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

## Contributing & Development

Contributions are welcome — bug reports, real-world sample documents (PII
removed!), and code. Project layout, test/lint setup, how the PII-free test
fixtures were made, and the PR checklist: **[CONTRIBUTING.md](CONTRIBUTING.md)**.
For issues, feature requests, or questions, open an issue on GitHub.

## License

Licensed under the [MIT License](LICENSE). You're free to use, modify, and
redistribute it, including for commercial purposes, provided the copyright
notice is retained. The software is provided "as is", without warranty of any
kind and with no liability on the author's part — see the LICENSE file for the
full disclaimer.
