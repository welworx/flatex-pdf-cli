# Importing into Portfolio Performance

Part of [flatex-pdf-cli](../README.md). `-format pp` writes two CSVs shaped for
[Portfolio Performance](https://www.portfolio-performance.info/)'s CSV import:

```bash
flatex-pdf-cli -format pp -o portfolio ~/Downloads/flatex
# writes portfolio-portfolio.csv and portfolio-accounts.csv
```

- `<base>-portfolio.csv` — buy/sell trades ("Portfolio Transactions" import)
- `<base>-accounts.csv` — dividends, interest, withheld tax on accumulating funds ("Account Transactions" import)

`-o <base>` is required since two files are written.

## Import walkthrough

In Portfolio Performance: **File > Import > CSV Files**, pick the "Portfolio
Transactions" or "Account Transactions" import, and use the matching CSV. PP's
CSV import lets you re-map any column, so if a column isn't auto-recognized,
map it by hand — after the first import, save the mapping as a template so
later imports are one click.

## Running PP in German? Use `-lang de`

```bash
flatex-pdf-cli -format pp -lang de -o portfolio ~/Downloads/flatex
```

`-lang de` produces German column headers (`Datum`, `Wert`, `Stück`, …), German
`Typ` values (`Kauf`, `Verkauf`, `Dividende`, `Zinsen`, `Steuern`), a semicolon
(`;`) field separator, and comma (`,`) as the decimal separator (e.g.
`1,478695`, not `1.478695`) — all German-locale conventions, and all defaults
PP's own import wizard already assumes on a German-locale install.

PP's CSV column auto-recognition is locale-sensitive with no English fallback,
so a German-locale PP install won't auto-map English headers at all — `-lang de`
is what makes auto-recognition work without manually mapping every column or
number format.

## Before bulk-importing

**Test-import a handful of rows first** and check the resulting positions/cash
balance against a statement you trust. The column mapping is our best-effort
read of PP's documented CSV fields; it hasn't been validated against every edge
case (e.g. multi-currency trades, partial fills).
