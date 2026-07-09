# Test Data Directory

Synthetic, **PII-free** flatex PDF fixtures used by the integration tests in
`internal/extractor/extractor_test.go`.

## How these were made

Each fixture is generated from a real flatexDEGIRO document using the
**`redacting-flatex-pdfs`** skill (`.claude/skills/redacting-flatex-pdfs/`).
The skill replaces customer name, address, and Depot/Konto/Transaktion/
Auftragsnummer with synthetic values while keeping the file byte-for-byte
visually identical to the original. The real source documents live in the
git-ignored `sensitive_test_docs/` directory and are **never committed**.

To regenerate or add fixtures, point the skill at a document in
`sensitive_test_docs/` and drop the redacted output here.

## Fixtures

| File | Type | Detected as |
|------|------|-------------|
| `trade_sample_1.pdf`, `trade_sample_2.pdf` | Wertpapierabrechnung Kauf | `TRADE` |
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
