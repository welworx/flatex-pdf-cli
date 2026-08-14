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
  "order_date": "2024-06-13",
  "trade_date": "2024-06-15",
  "value_date": "2024-06-17",
  "execution_time": "13:56",
  "type": "BUY",
  "quantity": 10,
  "price": 25.500000,
  "gross_amount": 255.00,
  "gross_currency": "EUR",
  "withholding_tax": 0.00,
  "gain_loss": 0.00,
  "exchange_rate": 1.000000,
  "net_amount": -263.50,
  "net_currency": "EUR",
  "costs": {
    "provision": 5.50,
    "own_expenses": 0.00,
    "foreign_expenses": 3.00,
    "total": 8.50,
    "foreign_expenses_breakdown": {
      "courtage": 0.00,
      "trading_fee": 0.50,
      "settlement": 2.50,
      "closing_notes": 0.00,
      "ls_allocation": 0.00,
      "financial_transaction_tax": 0.00,
      "other": 0.00
    }
  },
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
- `isin` — ISIN of the security; its check digit is validated, see [ISIN check digit](#isin-check-digit)
- `wkn` — German securities identification number (if available)
- `security_name` — Bezeichnung of the security, if the document prints one
- Dates — see [Dates](#dates). Every date is ISO YYYY-MM-DD.

## A stated `0,00` is not a missing value

Every amount and quantity field — `quantity`, `price`, `gross_amount`,
`net_amount`, `limit`, `withholding_tax`, `gain_loss`,
`distribution_per_share`, `interest_rate`,
`reinvestment_per_share` — follows one rule:

| The document… | JSON | CSV |
|---|---|---|
| prints `0,00` | `0.00` | `0.00` |
| prints no such line | field omitted | empty cell |

These are different facts. A sale closed exactly at cost has a real gain of
zero; a 0% interest period has a real rate of zero; a fee-free settlement
really charged nothing. Collapsing them made a genuine zero read back as a
parse failure, and it also disabled the internal cross-checks: a stated
`Endbetrag: 0,00` used to be treated as "no Endbetrag" and silently skipped
settlement validation. A stated zero is now validated like any other value.

## Numbers keep the document's own precision

Every amount is emitted with the number of decimal places the statement
printed it with, not Go's shortest round-trip form:

| The document prints | JSON |
|---|---|
| `Kurs : 110,000000 EUR` | `"price": 110.000000` |
| `Kurswert : 1.540,00 EUR` | `"gross_amount": 1540.00` |
| `Ausgeführt : 14 St.` | `"quantity": 14` |
| `Provision : 5,90 EUR` | `"provision": 5.90` |

The precision is information the statement is giving you. A price quoted to
six places is not the same claim as one quoted to two, and a whole-share
execution is not a quantity that merely happens to be round — marshalled as
plain floats, `110,000000` and `14` both collapse and the distinction is gone.
These are still ordinary JSON numbers (`1540.00` parses as `1540`), so a
consumer that does not care about the digits does not have to do anything.

**Nothing is rounded, and nothing is invented.** This is an extractor: every
number in the output is either printed on the page or an exact sum of numbers
printed on the page.

`costs.total` is the only figure of the second kind — `provision` plus
`own_expenses` plus `foreign_expenses`, summed in exact decimal arithmetic so
it is the true total rather than a float sum snapped back to the cent. It
asserts nothing the itemised lines do not already say.

A field the document does not state is **absent**, not filled in:

| The document… | Result |
|---|---|
| prints no Devisenkurs | `exchange_rate` omitted — not defaulted to `1` |
| prints no charge line (dividends, savings plans) | `costs` omitted entirely |
| prints no Kurswert (savings plans) | `gross_amount` and `gross_currency` omitted |

A savings-plan row is the clearest case: it prints Stücke, Ausf.-Kurs and
Betrag, so `quantity`, `price` and `net_amount` are all it yields. Earlier
versions reconstructed a `gross_amount` of `198.5000168` and a
`costs.unitemised` fee of `1.4999832` from those three. Both have been removed
— neither appears anywhere on the statement, and their trailing digits came
from the quantity being printed to six decimals rather than from anything the
document knows. The gap between the settled amount and the value of the shares
is still computed as a sanity check on the column layout; it is simply not
reported.

The Portfolio Performance export (`-format pp`) plays by different rules,
because it is not an extract — it is a file shaped for another program:

- It **derives the savings-plan fee** the JSON omits, since PP is accounting
  for what a purchase cost and leaving it out would understate the position.
  Rounded to the cent there, and it never enters the JSON or CSV.
- It **supplies an exchange rate of 1** where no Devisenkurs was printed, since
  PP requires the column and a zero breaks its valuation.
- It keeps **shortest-form numbers** rather than the document's precision: PP
  re-parses these columns, and padding to the cent would round a fractional
  share count — `1.478695` shares is not `1.48`.

## ISIN check digit

Every extracted `isin` is checked against its own check digit, per ISO 6166's
Luhn algorithm. A misread character or a value that landed in the wrong field
is far more likely to fail this check than to pass it by coincidence, so a
failure is treated the same as any other cross-check: the parse fails loudly
rather than emitting a wrong ISIN. flatex's synthetic crypto ISINs (e.g.
`XFC000A2YY6Q`) are valid Luhn numbers and pass like any other.

## Withholding tax (Austrian KESt)

`withholding_tax` is **extracted, never computed** — flatex works the figure
out itself, including every offset this tool cannot see, so it is read off the
document as stated.

Austrian Kapitalertragsteuer is levied on the `gain_loss`, and the value is
signed:

- **A gain withholds tax** — a positive amount, at most 27.5% of the gain.
  It is very often much less: the gain is netted against the
  Verluststeuertopf first (`verkauf_sample_1` withholds 24.51 on a gain of
  403.97, i.e. 6.07%), and Altbestand acquired before the regime took effect
  is exempt, so `0` on a real gain is valid.
- **A loss can refund tax** — a negative amount, paid back out of the
  Verluststeuertopf, and only up to what was already withheld earlier that
  year. That ceiling is not stated on the document, so a refund's size is not
  cross-checked here.

The parser rejects a withheld tax that exceeds 27.5% of a stated gain, or a
refund on a gain, since either means a figure landed in the wrong column. It
never rejects a tax that is merely lower than the rate would suggest.

## Dates

One field per date the document prints, each present only when it does:

| Label | Meaning | Field |
|---|---|---|
| `Graz, …` | Letter/print date — when the PDF was generated | *not extracted* |
| `Auftragsdatum` | When the order was placed | `order_date` |
| `Handelstag` / `Schlusstag` / `Ausführungsdatum` | When the trade executed | `trade_date` |
| `Ausführungszeit` | Time of day the trade executed | `execution_time` |
| `Buchtag` | When a savings-plan row was booked | `booking_date` |
| `Valuta` | When cash and securities settle | `value_date` |

**`trade_date` is the standard term** for the day a deal is struck — the
counterpart to the value date, and what T+2 counts from. flatex writes it three
different ways depending on the layout (`Handelstag` on a trade confirmation,
`Schlusstag` on crypto, `Ausführungsdatum` on older forms); all three denote
the same thing and land in the same field.

`Buchtag` is **not** folded into it. A `Sammelabrechnung` row states a booking
day and a Valuta and no `Handelstag` at all, so those rows carry a
`booking_date` and no `trade_date` — they have no confirmed execution day to
report. Pure cash events (`DIVIDEND`, `INTEREST`, `ACCUMULATING`) have neither;
they carry `value_date`. A pending `ORDER` has not executed, so it carries only
`order_date`.

> **Removed in v0.7.0.** There used to be a single `date` field holding
> whichever of these mattered for the document type — the trade date on a
> trade, the value date on a dividend. It read as an extracted field and was
> not one: its meaning changed underneath a consumer depending on what it was
> looking at. Picking a primary date is a decision for whoever needs one, so
> the Portfolio Performance export makes it and the extracted data no longer
> pretends to. If you relied on `date`, the equivalent is
> `trade_date ?? booking_date ?? value_date`.

`execution_time` is `"HH:MM"`. The document prints a bare local time with no
date and no zone (`Ausführungszeit    13:56 Uhr`), so it stays a time rather
than being folded into `date`: the day is `date`, and picking a zone to build
a timestamp would assert more than the statement does. Crypto settlements
carry the same figure on their `Schlusstag` line (`29.01.2026, 16:00 Uhr`) and
populate the field from there. `00:00` is passed through as printed — some
confirmations really do state it — so treat it as the document's answer, not
as a missing value.

`date` is the **trade date**, because that is the date that fixes the price
and the holding period, and it is the date Portfolio Performance expects in
the `Datum` column of a portfolio-transaction import. `order_date` and
`value_date` are emitted alongside it so a different convention needs no
re-parsing.

For pure cash events with no trade — `DIVIDEND`, `INTEREST`,
`ACCUMULATING` — `date` is the `Valuta`, since that is when the money moves.

## Settlement Amounts (All Settled Documents)

Every document that settles states the amount that actually moved, and most
also state a gross figure and its deductions. Those are **one set of fields
shared by all document types**, not a separate set per type — a trade's
`Kurswert` and a dividend's `Bruttoausschüttung` both land in `gross_amount`,
and `Endbetrag` always lands in `net_amount`.

Each is present only when its document prints it. A savings-plan row states a
`Betrag` and no `Kurswert`, so it carries `net_amount` alone.

- `gross_amount` — Kurswert on a trade (quantity × price, before costs),
  Bruttoausschüttung on a dividend; in either case the value before deductions
- `gross_currency` — Currency of `gross_amount`
- `withholding_tax` — Tax withheld on the transaction (Einbeh. KESt on trades,
  Einbeh. Steuer on dividends and crypto); see
  [Withholding tax](#withholding-tax-austrian-kest)
- `withholding_tax_currency` — Currency of `withholding_tax`
- `gain_loss` — Capital gain or loss (sell transactions)
- `exchange_rate` — Devisenkurs; omitted when the document prints none
- `net_amount` — Endbetrag, signed by cash direction: **negative for a buy**
- `net_currency` — Currency of `net_amount`
- `costs` — Charge block; see [Costs](#costs)

The three are tied together by
`net_amount = gross_amount ∓ costs.total ∓ withholding_tax`: deductions add to
what a purchase costs and subtract from what a sale pays. The parser checks
this identity on every document and fails loudly when it does not hold, which
is how a value that landed in the wrong column is caught.

**`net_amount` is the only signed field.** `gross_amount`, `withholding_tax`
and everything under `costs` are unsigned magnitudes; the direction they apply
in follows from `type`. Summing `net_amount` across a mixed batch is
meaningful, summing `gross_amount` across one is not.

> **Renamed in v0.7.0.** These fields were previously spelled `gross_value`,
> `final_amount`, `final_currency` and `price_currency` on trades, and
> `gross_amount`, `net_amount`, `net_currency`, `gross_currency` on dividends —
> two names for each of the same quantities. `price_currency` was doubly
> misleading: `price` is only ever parsed in EUR, and the field actually held
> the currency of `Kurswert`. The trade spellings are gone; use the names above.

## Trade-Specific Fields

- `type` — BUY or SELL
- `quantity` — Number of shares/units
- `price` — Price per unit (Kurs), always in EUR
- `order_date` — Auftragsdatum, when the order was placed
- `value_date` — Valuta, when the trade settles
- `execution_time` — Ausführungszeit as `"HH:MM"`; see
  [Dates](#dates)
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
- `foreign_expenses_breakdown` — itemisation of `foreign_expenses`, present
  only when the document prints it: `courtage`, `trading_fee` (Tradinggebühr),
  `settlement` (Regulierung), `closing_notes` (Schlussnoten), `ls_allocation`
  (LS-Umlegung), `financial_transaction_tax` (Finanztransaktionssteuer),
  `other` (Sonstige).

The document itself marks what this breakdown belongs to with a footnote: the
charge line reads `* Fremde Spesen`, and the starred note below it —
"`* Enthalten sind folgende Gebühren`" — lists the components. They therefore
sum to `foreign_expenses` and are already counted in `total`. **Do not add
them on top of it.**

> **Renamed in v0.7.0.** This block was called `fees`, a name that said nothing
> about which charge it itemised and so invited exactly that double-count.

## Dividend-Specific Fields

The gross, tax and net amounts are the shared
[settlement amounts](#settlement-amounts-all-settled-documents).

- `distribution_per_share` — Dividend per unit held
- `distribution_currency` — Currency of dividend
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
    { "document_type": "TRADE", "isin": "DE0005140008", "trade_date": "2024-06-15" }
  ]
}
```

The metadata describes the **whole batch**, and `transactions` carries no
per-transaction depot. So when a batch spans more than one depot — two
accounts in the same folder, or a depot transfer — there is no single truthful
value to put here, and `-include-metadata` fails with the conflicting depot
numbers rather than writing one of them. Parse each depot separately, or drop
the flag if you only want the transactions.

> **Fixed in v0.7.0.** Previously the first file's depot was captured and
> stamped over the entire batch, silently reattributing a second account's
> transactions to the first holder.
