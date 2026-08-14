# JSON Output Format

Part of [flatex-pdf-cli](../README.md). Full field reference for the JSON the
CLI emits.

## Transaction Object

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
  "order_date": "2024-06-13",
  "value_date": "2024-06-17",
  "type": "BUY",
  "quantity": 10.0,
  "price": 25.50,
  "price_currency": "EUR",
  "gross_value": 255.00,
  "costs": {
    "provision": 5.50,
    "own_expenses": 0.00,
    "foreign_expenses": 3.00,
    "total": 8.50,
    "fees": {
      "courtage": 0.00,
      "trading_fee": 0.50,
      "settlement": 2.50,
      "closing_notes": 0.00,
      "ls_allocation": 0.00,
      "financial_transaction_tax": 0.00,
      "other": 0.00
    }
  },
  "withholding_tax": 0.00,
  "gain_loss": 0.00,
  "exchange_rate": 1.0,
  "final_amount": -263.50,
  "final_currency": "EUR",
  "custody_type": "Wertpapierrechnung",
  "depositary": "Clearstream Lux.",
  "deposit_country": "GB",
  "execution_venue": "XETRA"
}
```

## Common Fields (All Transactions)

- `source` — Source filename (only if `-include-source` flag is used)
- `order_number` — Order number (Auftragsnummer), if present
- `transaction_number` — Tax-report transaction number (Transaktion-Nr.), if present
- `document_type` — Type of document (TRADE, DIVIDEND, INTEREST, ACCUMULATING, ORDER, CRYPTO, SAVINGSPLAN)
- `isin` — ISIN of the security
- `wkn` — German securities identification number (if available)
- `security_name` — Bezeichnung of the security, if the document prints one
- `date` — Transaction date in YYYY-MM-DD format. See [Which date is `date`?](#which-date-is-date).

## Which date is `date`?

A flatex trade confirmation prints four dates, and they are frequently
different days:

| Label | Meaning | Field |
|---|---|---|
| `Graz, …` | Letter/print date — when the PDF was generated | *not extracted* |
| `Auftragsdatum` | When the order was placed | `order_date` |
| `Handelstag` (`Schlusstag` on crypto, `Buchtag` on savings plans) | When the trade executed | `date` |
| `Valuta` | When cash and securities settle | `value_date` |

`date` is the **trade date**, because that is the date that fixes the price
and the holding period, and it is the date Portfolio Performance expects in
the `Datum` column of a portfolio-transaction import. `order_date` and
`value_date` are emitted alongside it so a different convention needs no
re-parsing.

For pure cash events with no trade — `DIVIDEND`, `INTEREST`,
`ACCUMULATING` — `date` is the `Valuta`, since that is when the money moves.

## Trade-Specific Fields

- `type` — BUY or SELL
- `quantity` — Number of shares/units
- `price` — Price per unit
- `price_currency` — Currency of price
- `gross_value` — Kurswert: quantity × price, before costs
- `order_date` — Auftragsdatum, when the order was placed
- `value_date` — Valuta, when the trade settles
- `costs` — Charge block; see [Costs](#costs)
- `withholding_tax` — Tax withheld on the transaction (Einbeh. KESt on trades,
  Einbeh. Steuer on dividends and crypto)
- `gain_loss` — Capital gain or loss (sell transactions)

  Both are **omitted entirely when the document states no such line**, and
  emitted as `0` when it states `0,00 EUR`. Those are different facts: a sale
  closed exactly at cost has a real gain of zero. In `-format csv` the same
  distinction appears as an empty cell versus `0`.
- `exchange_rate` — Currency exchange rate (1.0 when the document has no Devisenkurs)
- `final_amount` — Endbetrag, signed by cash direction: **negative for a buy**
- `final_currency` — Currency of final amount
- `custody_type` — Verwahrart, e.g. `Wertpapierrechnung`
- `depositary` — Lagerstelle, e.g. `Clearstream Lux.`
- `deposit_country` — Lagerland, translated to an ISO 3166-1 alpha-2 code
  (`Großbritannien` → `GB`). Omitted when the document has no Lagerland or
  names a country outside the translation table — the German name is never
  passed through untranslated, because gxpdf runs the Lagerland column
  straight into the one beside it (`GroßbritannienBemessungsgrundlage: 0,00
  EUR`) and the country list is what separates them. Open an issue if a real
  statement names a country the table misses.
- `execution_venue` — Execution venue/type (Ausf.platz/-art), e.g. XETRA

## Costs

`costs` is present whenever the document has a charge block, and **absent when
it has none** — so a dividend notice has no `costs` key at all, while a trade
that cost nothing reports explicit zeros. Unlike the rest of the schema, the
fields inside `costs` are emitted even when zero, because "flatex charged
nothing" and "this was never parsed" are different answers.

- `provision` — Provision, flatex's own commission
- `own_expenses` — Eigene Spesen
- `foreign_expenses` — Fremde Spesen, charges passed through from third parties
- `total` — `provision + own_expenses + foreign_expenses`; the number to use as
  the transaction's total cost
- `fees` — itemisation of `foreign_expenses` from the document's
  "Enthalten sind folgende Gebühren" block, present only when that block is:
  `courtage`, `trading_fee` (Tradinggebühr), `settlement` (Regulierung),
  `closing_notes` (Schlussnoten), `ls_allocation` (LS-Umlegung),
  `financial_transaction_tax` (Finanztransaktionssteuer), `other` (Sonstige).

The `fees` components sum to `foreign_expenses` and are therefore already
counted in `total`. **Do not add them on top of it.**

## Dividend-Specific Fields

- `distribution_per_share` — Dividend per unit held
- `distribution_currency` — Currency of dividend
- `gross_amount` — Total dividend before withholding
- `gross_currency` — Currency of gross amount
- `withholding_tax_currency` — Currency of withholding tax amount
- `net_amount` — Dividend after withholding tax
- `net_currency` — Currency of net amount
- `ex_date` — Ex-dividend date
- `value_date` — Value date for the payment

## Interest-Specific Fields

- `interest_rate` — Interest rate percentage
- `period_from` — Start of interest period
- `period_to` — End of interest period

## Accumulating-Specific Fields

- `reinvestment_per_share` — Reinvestment amount per unit
- `reinvestment_currency` — Currency of reinvestment
- `accrual_date` — Date reinvestment was accrued

## Order-Specific Fields (Sammelauftragsbestätigung)

- `security_name` — Bezeichnung (may include the execution venue, which the PDF column layout does not always separate)
- `limit` — Limit price of the order
- `valid_until` — Order validity date (Gültig bis)

## Crypto-Specific Fields (Sammelabrechnung Kryptowerte)

- `security_name` — Crypto asset name (e.g. BITCOIN); crypto positions have no ISIN
- `custody_type` — Verwahrart (e.g. Kryptoverwahrung)
- `depositary` — Kryptoverwahrer (e.g. Tangany GmbH)

## Metadata Wrapper (`-include-metadata`)

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
