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
| `date` | ISO `YYYY-MM-DD`. Trade date (Handelstag/Schlusstag/Buchtag) for trades, Valuta for dividends and interest — **not** the letter date printed at the top of the page |
| `order_date`, `value_date` | Auftragsdatum / Valuta, ISO `YYYY-MM-DD` |
| `execution_time` | Ausführungszeit as `"HH:MM"` — a bare local time, no date and no zone; the day is `date` |
| `quantity`, `price` | numbers (decimals normalized); `price` is per unit, in EUR |
| `gross_amount`, `gross_currency`, `net_amount`, `net_currency` | settlement amounts, the **same fields on every document type**: `gross_amount` is Kurswert on a trade and Bruttoausschüttung on a dividend, `net_amount` is always Endbetrag. `net_amount` is the only signed field — negative for a buy; `gross_amount`, `withholding_tax` and `costs` are unsigned, with the direction given by `type` |
| `costs` | charge block — `provision`, `own_expenses`, `foreign_expenses`, `total`, and a `foreign_expenses_breakdown` itemisation of `foreign_expenses` (the document's `* Fremde Spesen` footnote). Absent when the document has no charge block; zeros inside it are real. Use `costs.total` as the transaction's cost — the breakdown entries are already part of it |
| `limit`, `valid_until` | ORDER only |
| `custody_type`, `depositary` | e.g. CRYPTO `Kryptoverwahrung` / `Tangany GmbH` |
| `deposit_country` | Lagerland as an ISO 3166-1 alpha-2 code (`GB`); absent if the document names a country the translation table misses |

`metadata` (with `-include-metadata`): `depot_number`, `depot_holder`, `account_number`.

## Agent tips

- Always pass `-quiet` when machine-reading the output, then `json.loads` stdout.
- A folder of mixed flatex documents parses in one call; group/aggregate the
  returned array by `document_type` as needed.
- `ORDER` documents yield **one record per pending order**, so a single PDF can
  produce multiple array entries.
- Amounts are emitted at the document's own precision (`110.000000`, `1540.00`,
  `14`), so the raw JSON text is not byte-identical to a re-serialised
  `json.loads`/`json.dumps` round trip. They are ordinary JSON numbers and
  parse normally; only diff the parsed values, never the text.
- Nothing is rounded and nothing is invented: every number is printed on the
  page or an exact sum of printed numbers (`costs.total` is the only sum). A
  field the document does not state is **absent** — no Devisenkurs means no
  `exchange_rate` key, not a default of 1. Use `.get()`, do not assume a key.
- A `SAVINGSPLAN` row yields only `quantity`, `price` and `net_amount`; it
  prints no Kurswert and no fee line, so it has no `gross_amount` and no
  `costs`. `-format pp` still derives the withheld fee for Portfolio
  Performance.
- `-include-metadata` **fails** when a batch spans more than one depot, since
  one metadata block cannot describe two. Parse each depot separately, or drop
  the flag if you only need the transactions.

## Known limitations

See the project README "Known Limitations" — notably: ORDER `security_name` may
include the execution venue, German-only support, and `depot_holder` can be
noisy on some layouts. The tool never fails the whole batch for one bad file.
