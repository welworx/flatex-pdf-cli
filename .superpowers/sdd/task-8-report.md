# Task 8: Interest Parser

## Summary
Successfully implemented `ParseInterest` function for parsing INTEREST documents with comprehensive test coverage.

## Implementation Details

### ParseInterest Function
Located in `/Users/welworx/dev-private/flatex-pdf-cli/internal/parser/parser.go` (lines 235-341)

**Extracted Fields:**
- **ISIN**: Investment identifier (required)
- **GrossAmount**: Pre-tax interest amount
- **GrossCurrency**: Currency of gross amount (defaults to EUR)
- **WithholdingTax**: Tax withheld (Einbeh. KESt)
- **WithholdingTaxCurrency**: Currency of withholding tax (defaults to EUR)
- **NetAmount**: Post-tax interest amount (Endbetrag)
- **NetCurrency**: Currency of net amount (defaults to EUR)
- **InterestRate**: Annual interest rate percentage (Zinssatz)
- **PeriodFrom**: Start date of interest period (converted from DD.MM.YYYY to YYYY-MM-DD)
- **PeriodTo**: End date of interest period (converted from DD.MM.YYYY to YYYY-MM-DD)
- **Date**: Value date (Valuta) used as transaction date
- **WKN**: Identifier (optional)
- **Source**: Document filename

### Key Features
- Regex-based extraction using patterns consistent with existing parsers
- European decimal format support (comma as decimal separator)
- Automatic currency defaulting to EUR when not found
- Date conversion from German format (DD.MM.YYYY) to ISO format (YYYY-MM-DD)
- Period range parsing for interest period dates

## Test Coverage

### Test Functions Added
1. **TestParseInterestRouting**: Verifies Parse() function correctly routes INTEREST documents
2. **TestParseInterest**: Comprehensive validation of all extracted fields

### Test Data
```
ISIN: IE00B3RBWM25
Bruttobetrag : 25,50 EUR
Einbeh. KESt : 3,40 EUR
Endbetrag : 22,10 EUR
Zinssatz : 2,5%
Zinsperiode : 01.01.2026 bis 31.03.2026
Valuta : 15.04.2026
```

### Assertions
- DocumentType: INTEREST
- ISIN: IE00B3RBWM25
- GrossAmount: 25.50
- GrossCurrency: EUR
- WithholdingTax: 3.40
- WithholdingTaxCurrency: EUR
- NetAmount: 22.10
- NetCurrency: EUR
- InterestRate: 2.5
- PeriodFrom: 2026-01-01
- PeriodTo: 2026-03-31
- Date: 2026-04-15

## Test Results
```
go test ./internal/parser -v

=== RUN   TestParseRouting
--- PASS: TestParseRouting (0.00s)
=== RUN   TestParseDividendRouting
--- PASS: TestParseDividendRouting (0.00s)
=== RUN   TestParseThesaurierungRouting
--- PASS: TestParseThesaurierungRouting (0.00s)
=== RUN   TestParseTradeBuy
--- PASS: TestParseTradeBuy (0.00s)
=== RUN   TestParseDividend
--- PASS: TestParseDividend (0.00s)
=== RUN   TestParseInterestRouting
--- PASS: TestParseInterestRouting (0.00s)
=== RUN   TestParseInterest
--- PASS: TestParseInterest (0.00s)
=== RUN   TestParseThesaurierung
--- PASS: TestParseThesaurierung (0.00s)

PASS
ok	github.com/welworx/flatex-pdf-cli/internal/parser	0.453s
```

**Result:** All 8 tests passing (including 2 new ParseInterest tests)

## Files Modified
1. `/Users/welworx/dev-private/flatex-pdf-cli/internal/parser/parser.go` - Implemented ParseInterest function
2. `/Users/welworx/dev-private/flatex-pdf-cli/internal/parser/parser_test.go` - Added TestParseInterestRouting and TestParseInterest
