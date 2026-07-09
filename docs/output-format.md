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

## Common Fields (All Transactions)

- `source` — Source filename (only if `-include-source` flag is used)
- `order_number` — Order number (Auftragsnummer), if present
- `transaction_number` — Tax-report transaction number (Transaktion-Nr.), if present
- `document_type` — Type of document (TRADE, DIVIDEND, INTEREST, ACCUMULATING, ORDER, CRYPTO, SAVINGSPLAN)
- `isin` — ISIN of the security
- `wkn` — German securities identification number (if available)
- `date` — Transaction date in YYYY-MM-DD format

## Trade-Specific Fields

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
