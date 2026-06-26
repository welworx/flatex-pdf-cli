---
name: flatex-pdf-cli
description: Use to extract structured transaction data as JSON from German flatexDEGIRO broker PDFs — trade confirmations, dividends, interest, accumulation (Ertragsmitteilung) notices, order confirmations, and crypto settlements. Wraps the `flatex-pdf-cli` command-line tool. Use whenever the user points at a flatex/flatexDEGIRO PDF or a folder of them and wants the data parsed.
---

# flatex-pdf-cli

Turn German flatexDEGIRO broker PDFs into structured JSON. The CLI does the
PDF text extraction, document-type detection, and field parsing; you just invoke
it and consume the JSON on stdout.

## Check setup

**First time?** Verify the tool is installed:

```bash
flatex-pdf-cli --help
```

If "command not found", see [Install](#install-one-time) below.

## Install (one time)

See [INSTALL.md](INSTALL.md) for detailed installation instructions.

Quick: download the binary for your platform from the [releases page](https://github.com/welworx/flatex-pdf-cli/releases), or `go install github.com/welworx/flatex-pdf-cli@latest` (requires Go 1.26+).

## Usage

```bash
flatex-pdf-cli [flags] <file.pdf | directory>
```

Recommended invocation for agents (pure JSON, account context, source tracking):

```bash
flatex-pdf-cli -quiet -include-metadata -include-source /path/to/pdfs/
```

Flags:
- `-quiet` — hide skipped/problematic files; emit only valid JSON
- `-include-metadata` — wrap output with depot metadata
- `-include-source` — add source filename to each transaction
- `-o FILE` — output file (stdout if not provided)
- `-version` — show version and exit

Behavior:
- A directory is scanned recursively for `*.pdf`.
- A file that cannot be parsed is **skipped**, not fatal — the rest still produce output. Exit code is non-zero only if *nothing* parsed.
- **German PDFs only.** Non-German documents are rejected with a clear error.

## Output

Without `-include-metadata`, stdout is a JSON array of transaction objects.
Key fields (most are `omitempty`):

| Field | Meaning |
|---|---|
| `document_type` | `TRADE`, `DIVIDEND`, `INTEREST`, `ACCUMULATING`, `ORDER`, or `CRYPTO` |
| `source` | source filename (with `-include-source`) |
| `isin`, `wkn` | security identifiers (crypto has none) |
| `security_name` | name when there is no ISIN (crypto) or for orders |
| `order_number`, `transaction_number` | Auftragsnummer / Transaktion-Nr. |
| `type` | `BUY` / `SELL` |
| `date`, `value_date` | ISO `YYYY-MM-DD` |
| `quantity`, `price`, `gross_value`, `provision`, `final_amount` | numbers (decimals normalized) |
| `limit`, `valid_until` | ORDER only |
| `custody_type`, `depositary` | e.g. CRYPTO `Kryptoverwahrung` / `Tangany GmbH` |

`metadata` (with `-include-metadata`): `depot_number`, `depot_holder`, `account_number`.

## Agent tips

- Always pass `-quiet` when machine-reading the output, then `json.loads` stdout.
- A folder of mixed flatex documents parses in one call; group/aggregate the
  returned array by `document_type` as needed.
- `ORDER` documents yield **one record per pending order**, so a single PDF can
  produce multiple array entries.

## Known limitations

See the project README "Known Limitations" — notably: ORDER `security_name` may
include the execution venue, German-only support, and `depot_holder` can be
noisy on some layouts. The tool never fails the whole batch for one bad file.
