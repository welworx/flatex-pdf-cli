# Test Data Directory

Synthetic, **PII-free** flatex PDF fixtures. `TestAllFixturesParse` in
`internal/parser/parser_test.go` runs every one of them through the real
extract-and-parse path and asserts the identifiers each should yield.

## How these were made

Each fixture is generated from a real flatexDEGIRO document using the
**`redacting-flatex-pdfs`** skill (`.claude/skills/redacting-flatex-pdfs/`).
The skill replaces customer name, address, and Depot/Konto/Transaktion/
Auftragsnummer with synthetic values while keeping the file byte-for-byte
visually identical to the original. The real source documents live in the
git-ignored `sensitive_test_docs/` directory and are **never committed**.

To regenerate or add fixtures, point the skill at a document in
`sensitive_test_docs/` and drop the redacted output here.

**Do not skip the skill's reflow step.** PyMuPDF appends replacement text as a
new content stream, so a fixture can render perfectly while every replaced
identifier is out of position in content-stream order — which is the order
`gxpdf` reads. All fixtures were once broken this way: two failed to parse
at all and the rest silently returned empty order/transaction/depot numbers.
After adding a fixture, run `go test ./internal/parser/ -run TestAllFixturesParse`.

`trade_sample_3.pdf` is the **older** trade layout: it uses the base-14 fonts
directly (`Courier`, `Helvetica`) rather than the embedded `CursorBFO`/`HerosBFO`
subsets, and its mono body font is StandardEncoding, which `gxpdf` decodes as
WinAnsi. It is kept because that combination is what exposed the `ß` → `û`
mis-decode repaired by `extractor.standardEncodingFixes`; its `deposit_country: GB`
assertion fails if that repair is removed.

## Fixtures

| File | Type | Detected as |
|------|------|-------------|
| `trade_sample_1.pdf`, `trade_sample_2.pdf`, `trade_sample_3.pdf` | Wertpapierabrechnung Kauf | `TRADE` |
| `dividend_sample_1.pdf`, `dividend_sample_2.pdf` | Ertragsmitteilung / Ausschüttung | `DIVIDEND` |
| `orderbestaetigung_sample_1.pdf` | Sammelauftragsbestätigung (order confirmation) | `ORDER` |
| `krypto_sample_1.pdf` | Sammelabrechnung Kryptowerte (crypto settlement) | `CRYPTO` |
| `sparplan_sample_1.pdf` | Sammelabrechnung aus (annual savings-plan settlement) | `SAVINGSPLAN` |

These also exercise the skip-and-continue behaviour and serve as PII-free
samples of each layout for regression tests.

## Document type detection

The extractor identifies types by German keywords, checked in this order
(more specific layouts first, since several also contain "Kauf"):

- **CRYPTO**: "Sammelabrechnung" + "Kryptowerte"
- **ORDER**: "Sammelauftragsbestätigung"
- **SAVINGSPLAN**: "Sammelabrechnung" (without "Kryptowerte")
- **TRADE**: "Kauf" / "Verkauf"
- **DIVIDEND**: "Ausschüttung"
- **INTEREST**: "Zinsen"
- **ACCUMULATING**: "Ertragsmitteilung"
- **UNKNOWN**: no recognized keywords
