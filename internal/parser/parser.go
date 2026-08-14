package parser

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/welworx/flatex-pdf-cli/internal/extractor"
	"github.com/welworx/flatex-pdf-cli/internal/schema"
)

// Parse routes an ExtractedDocument to the appropriate parser based on
// DocumentType, then cross-checks the result. It returns a slice because some
// document types (e.g. order confirmations) contain multiple transactions.
func Parse(doc *extractor.ExtractedDocument) ([]*schema.Transaction, error) {
	txns, err := parseByType(doc)
	if err != nil {
		return nil, err
	}
	if err := validate(txns); err != nil {
		return nil, err
	}
	return txns, nil
}

func parseByType(doc *extractor.ExtractedDocument) ([]*schema.Transaction, error) {
	switch doc.DocumentType {
	case "TRADE":
		return one(parseTrade(doc))
	case "DIVIDEND":
		return one(parseDividend(doc))
	case "INTEREST":
		return one(parseInterest(doc))
	case "ACCUMULATING":
		return one(parseAccumulating(doc))
	case "CRYPTO":
		return one(parseCrypto(doc))
	case "ORDER":
		return parseOrderConfirmation(doc)
	case "SAVINGSPLAN":
		return parseSavingsPlan(doc)
	default:
		return nil, fmt.Errorf("unknown document type: %s", doc.DocumentType)
	}
}

// one wraps a single-transaction parser result into a slice.
func one(tx *schema.Transaction, err error) ([]*schema.Transaction, error) {
	if err != nil {
		return nil, err
	}
	return []*schema.Transaction{tx}, nil
}

// parseTrade parses a TRADE document.
func parseTrade(doc *extractor.ExtractedDocument) (*schema.Transaction, error) {
	text := doc.Text

	// Extract ISIN and WKN
	isin := extractISIN(text)
	if isin == "" {
		return nil, fmt.Errorf("ISIN not found in document")
	}

	// flatex prints four dates in a trade header: the letter date ("Graz,
	// 16.09.2025"), Auftragsdatum (order placed), Handelstag (executed) and
	// Valuta (settled). Handelstag is the one that dates the position change.
	// Scanning for the first date-shaped string in the document instead
	// picked up the letter date, which is usually a day or more late.
	//
	// The three labels below are the same field under different names across
	// layouts. Auftragsdatum used to close this chain as a last resort, which
	// meant a document without a Handelstag reported its order date as the
	// trade date — a wrong answer dressed as a right one, and one the
	// order_date field already carries. A trade confirmation states when it
	// executed; if none of these three is present, say so.
	tradeDate := firstNonEmpty(
		dateField(text, `Handelstag`),
		dateField(text, `Schlusstag`),
		dateField(text, `Ausführungsdatum`),
	)
	if tradeDate == "" {
		return nil, fmt.Errorf("trade date (Handelstag/Schlusstag/Ausführungsdatum) not found in document")
	}

	// Determine trade type: "Kauf" → "BUY", "Verkauf" → "SELL"
	tradeType := "BUY"
	if strings.Contains(strings.ToLower(text), "verkauf") {
		tradeType = "SELL"
	}

	// Extract quantity (executed shares)
	quantity, err := extractFloat(text, `Ausgeführt\s*:\s*([\d\s.,]+)\s*St\.`)
	if err != nil {
		return nil, fmt.Errorf("quantity not found: %w", err)
	}

	// Extract price per share
	price, err := extractFloat(text, `Kurs\s*:\s*([\d\s.,]+)\s*EUR`)
	if err != nil {
		return nil, fmt.Errorf("price not found: %w", err)
	}

	// Extract currency (extract after "Kurswert")
	currency := extractString(text, `Kurswert\s*:\s*[\d\s.,]+\s*([A-Z]{3})`)
	if currency == "" {
		currency = "EUR" // Default to EUR if not found
	}

	// Extract gross value (Kurswert)
	grossValue, err := extractFloat(text, `Kurswert\s*:\s*([\d\s.,]+)\s*[A-Z]{3}`)
	if err != nil {
		return nil, fmt.Errorf("gross value not found: %w", err)
	}

	// Devisenkurs, when the document prints one. A EUR settlement carries no
	// such line, and the field is then absent rather than filled in with 1:
	// an unstated rate is not an extracted value.
	exchangeRate := floatPtr(extractFloat(text, `Devisenkurs\s*:\s*([\d\s.,]+)`))

	// Extract WKN from ISIN/WKN pattern (e.g., "IE000YU9K6K2/A3DP9J")
	wkn := extractString(text, `/([A-Z0-9]{6})[)\]]`)
	if wkn == "" {
		// Fallback to general WKN extraction
		wkn = extractWKN(text)
	}

	// Security name sits on the order line, between the side and "(ISIN/WKN)":
	// "Nr.700000011/1  Kauf  L&G GOLD MINING ETF (IE00B3CNHG25/A0Q8HZ)".
	// Older layouts print "Nr. 800000022/1" with a space after the dot.
	securityName := extractString(text,
		`Nr\.\s*[\d/]+\s+(?:Kauf|Verkauf)\s+(.+?)\s*\([A-Z]{2}[A-Z0-9]{9}[0-9]/`)

	// Extract identifiers (all optional)
	orderNumber := extractString(text, `Auftragsnummer\s*:?\s*(\S+)`)
	transactionNumber := extractString(text, `Transaktion-Nr\.\s*:?\s*(\d+)`)
	executionVenue := extractString(text, `Ausf\.platz/-art\s*([^\n]+)`)

	// Trades carry the withheld capital-gains tax as "Einbeh. KESt"; the
	// "Einbeh. Steuer" label used on dividend and crypto documents does not
	// appear here.
	withholdingTax := floatPtr(eurField(text, `Einbeh\.[^\S\n]*KESt`))
	gainLoss := floatPtr(eurField(text, `Gewinn/Verlust`))
	finalAmount := floatPtr(eurField(text, `Endbetrag`))

	transaction := &schema.Transaction{
		OrderNumber:       orderNumber,
		TransactionNumber: transactionNumber,
		DocumentType:      "TRADE",
		ISIN:              isin,
		WKN:               wkn,
		SecurityName:      securityName,
		TradeDate:         tradeDate,
		OrderDate:         dateField(text, `Auftragsdatum`),
		ValueDate:         dateField(text, `Valuta`),
		ExecutionTime:     timeField(text, `Ausführungszeit`),
		Type:              tradeType,
		Quantity:          ptr(quantity),
		Price:             ptr(price),
		GrossCurrency:     currency,
		GrossAmount:       ptr(grossValue),
		WithholdingTax:    withholdingTax,
		GainLoss:          gainLoss,
		ExchangeRate:      exchangeRate,
		NetAmount:         finalAmount,
		NetCurrency:       "EUR",
		CustodyType:       extractString(text, `Verwahrart[^\S\n]*:[^\S\n]*([^\n*]+)`),
		Depositary:        extractString(text, `Lagerstelle[^\S\n]*:[^\S\n]*([^\n*]+)`),
		DepositCountry:    countryISO2(extractString(text, `Lagerland[^\S\n]*:[^\S\n]*([^\n*]+)`)),
		ExecutionVenue:    executionVenue,
		Costs:             extractCosts(text),
	}

	return transaction, nil
}

// parseCrypto parses a Sammelabrechnung Kryptowerte (crypto buy/sell settlement).
// Crypto positions have no ISIN; the security is identified by name (e.g. BITCOIN).
func parseCrypto(doc *extractor.ExtractedDocument) (*schema.Transaction, error) {
	text := doc.Text

	// "Nr.<order>/N    Kauf    <NAME>" — order number, side and security name.
	side := extractString(text, `Nr\.[\d/]+\s+(Kauf|Verkauf)`)
	if side == "" {
		return nil, fmt.Errorf("crypto order line not found")
	}
	tradeType := "BUY"
	if side == "Verkauf" {
		tradeType = "SELL"
	}

	name := extractString(text, `Nr\.[\d/]+\s+(?:Kauf|Verkauf)\s+([^\n]+)`)
	if name == "" {
		return nil, fmt.Errorf("crypto security name not found")
	}

	// Schlusstag is the trade date (may be followed by a time).
	tradeDate := convertGermanDate(extractString(text, `Schlusstag:\s*(\d{2}\.\d{2}\.\d{4})`))
	if tradeDate == "" {
		return nil, fmt.Errorf("trade date (Schlusstag) not found in document")
	}

	quantity, err := extractFloat(text, `davon ausgef\.:\s*([\d.,]+)\s*St\.`)
	if err != nil {
		return nil, fmt.Errorf("executed quantity not found: %w", err)
	}

	// Note: "Kurs:" is case-sensitive and does not match "Devisenkurs:".
	price, err := extractFloat(text, `Kurs:\s*([\d.,]+)\s*EUR`)
	if err != nil {
		return nil, fmt.Errorf("price not found: %w", err)
	}
	grossValue, err := extractFloat(text, `Kurswert:\s*([\d.,]+)\s*EUR`)
	if err != nil {
		return nil, fmt.Errorf("gross value not found: %w", err)
	}

	withholdingTax := floatPtr(extractFloat(text, `Einbeh\. Steuer:\s*(-?[\d.,]+)\s*EUR`))
	gainLoss := floatPtr(extractFloat(text, `Gewinn/Verlust:\s*(-?[\d.,]+)\s*EUR`))
	finalAmount := floatPtr(extractFloat(text, `Endbetrag:\s*(-?[\d.,]+)\s*EUR`))

	// Devisenkurs, when the document prints one. A EUR settlement carries no
	// such line, and the field is then absent rather than filled in with 1:
	// an unstated rate is not an extracted value.
	exchangeRate := floatPtr(extractFloat(text, `Devisenkurs:\s*([\d.,]+)`))

	return &schema.Transaction{
		OrderNumber:       extractString(text, `Nr\.([\d/]+)`),
		TransactionNumber: extractString(text, `Transaktion-Nr\.:\s*(\d+)`),
		DocumentType:      "CRYPTO",
		SecurityName:      name,
		TradeDate:         tradeDate,
		// Crypto has no Ausführungszeit line; the time rides on Schlusstag
		// ("Schlusstag: 29.01.2026, 16:00 Uhr").
		ExecutionTime:  extractString(text, `Schlusstag:\s*\d{2}\.\d{2}\.\d{4},\s*(\d{2}:\d{2})`),
		Type:           tradeType,
		Quantity:       ptr(quantity),
		Price:          ptr(price),
		GrossCurrency:  "EUR",
		GrossAmount:    ptr(grossValue),
		Costs:          extractCosts(text),
		WithholdingTax: withholdingTax,
		GainLoss:       gainLoss,
		ExchangeRate:   exchangeRate,
		NetAmount:      finalAmount,
		NetCurrency:    "EUR",
		CustodyType:    extractString(text, `Verwahrart:\s*([^\n*]+)`),
		Depositary:     extractString(text, `Kryptoverwahrer:\s*([^\n*]+)`),
		ValueDate:      convertGermanDate(extractString(text, `Valuta:\s*(\d{2}\.\d{2}\.\d{4})`)),
	}, nil
}

// orderBlockRe matches one pending-order block of a Sammelauftragsbestätigung as
// extracted by gxpdf (two columns are merged per line):
//
//	<Auftrags-Nr> <ISIN> <Bezeichnung [+ venue]>
//	<WKN> Kauf|Verkauf vom <date> <qty> St.
//	Gültig bis: <date>
//	Limit: <price> EUR
//
// The Bezeichnung and Ausf.platz/-art share a column boundary that gxpdf does not
// always separate with a space (e.g. "…MINERS ETXETRA"), so they are captured
// together as the security name rather than split unreliably.
var orderBlockRe = regexp.MustCompile(
	`(\d{9}) ([A-Z0-9]{12}) ([^\n]+)\n([A-Z0-9]{6}) (Kauf|Verkauf) vom (\d{2}\.\d{2}\.\d{4}) ([\d.,]+) St\.\nGültig bis: (\d{2}\.\d{2}\.\d{4})\nLimit: ([\d.,]+) EUR`)

// parseOrderConfirmation parses a Sammelauftragsbestätigung into one transaction
// per pending order listed in the document.
func parseOrderConfirmation(doc *extractor.ExtractedDocument) ([]*schema.Transaction, error) {
	matches := orderBlockRe.FindAllStringSubmatch(doc.Text, -1)
	if len(matches) == 0 {
		return nil, fmt.Errorf("no orders found in document")
	}

	var txs []*schema.Transaction
	for _, m := range matches {
		tradeType := "BUY"
		if m[5] == "Verkauf" {
			tradeType = "SELL"
		}
		txs = append(txs, &schema.Transaction{
			OrderNumber:  m[1],
			DocumentType: "ORDER",
			ISIN:         m[2],
			SecurityName: strings.TrimSpace(m[3]),
			WKN:          m[4],
			Type:         tradeType,
			// "Kauf vom 28.01.2026" — the Auftr.Datum column. A pending order
			// has not executed, so it has an order date and no trade date;
			// this used to be reported as the transaction's date.
			OrderDate:  convertGermanDate(m[6]),
			Quantity:   ptr(mustFloat(m[7])),
			ValidUntil: convertGermanDate(m[8]),
			Limit:      ptr(mustFloat(m[9])),
		})
	}
	return txs, nil
}

// parseDividend parses a DIVIDEND document.
func parseDividend(doc *extractor.ExtractedDocument) (*schema.Transaction, error) {
	text := doc.Text

	// Extract ISIN
	isin := extractISIN(text)
	if isin == "" {
		return nil, fmt.Errorf("ISIN not found in document")
	}

	// Extract value date (Valuta field)
	valueDateStr := extractString(text, `Valuta\s*:\s*(\d{2}\.\d{2}\.\d{4})`)
	if valueDateStr == "" {
		return nil, fmt.Errorf("value date not found in document")
	}
	valueDate := convertGermanDate(valueDateStr)

	// Extract ex-date (Extag field - may contain different date)
	exDateStr := extractString(text, `Extag\s*:\s*(\d{2}\.\d{2}\.\d{4})`)
	exDate := convertGermanDate(exDateStr)

	// Extract quantity (shares held)
	quantity, err := extractFloat(text, `St\.\s*:\s*([\d\s.,]+)\s*Brutto`)
	if err != nil {
		return nil, fmt.Errorf("quantity not found: %w", err)
	}

	// Extract distribution per share
	distributionPerShare, err := extractFloat(text, `pro Stück\s*:\s*([\d\s.,]+)\s*[A-Z]{3}`)
	if err != nil {
		return nil, fmt.Errorf("distribution per share not found: %w", err)
	}

	// Extract distribution currency
	distributionCurrency := extractString(text, `pro Stück\s*:\s*[\d\s.,]+\s*([A-Z]{3})`)
	if distributionCurrency == "" {
		distributionCurrency = "EUR"
	}

	// Extract gross amount
	grossAmount, err := extractFloat(text, `Bruttoausschüttung\s*:\s*([\d\s.,]+)\s*[A-Z]{3}`)
	if err != nil {
		return nil, fmt.Errorf("gross amount not found: %w", err)
	}

	// Extract gross currency
	grossCurrency := extractString(text, `Bruttoausschüttung\s*:\s*[\d\s.,]+\s*([A-Z]{3})`)
	if grossCurrency == "" {
		grossCurrency = "EUR"
	}

	// Extract withholding tax
	withholdingTaxVal, err := extractFloat(text, `Einbeh\.\s*Steuer\s*:\s*(-?[\d\s.,]+)\s*[A-Z]{3}`)
	if err != nil {
		return nil, fmt.Errorf("withholding tax not found: %w", err)
	}
	withholdingTax := &withholdingTaxVal

	// Extract withholding tax currency
	withholdingTaxCurrency := extractString(text, `Einbeh\.\s*Steuer\s*:\s*-?[\d\s.,]+\s*([A-Z]{3})`)
	if withholdingTaxCurrency == "" {
		withholdingTaxCurrency = "EUR"
	}

	// Extract net amount (Endbetrag)
	netAmount, err := extractFloat(text, `Endbetrag\s*:\s*([\d\s.,]+)\s*[A-Z]{3}`)
	if err != nil {
		return nil, fmt.Errorf("net amount not found: %w", err)
	}

	// Extract net currency
	netCurrency := extractString(text, `Endbetrag\s*:\s*[\d\s.,]+\s*([A-Z]{3})`)
	if netCurrency == "" {
		netCurrency = "EUR"
	}

	// Devisenkurs, when the document prints one. A EUR settlement carries no
	// such line, and the field is then absent rather than filled in with 1:
	// an unstated rate is not an extracted value.
	exchangeRate := floatPtr(extractFloat(text, `Devisenkurs\s*:\s*([\d.,]+)`))

	// Extract WKN from ISIN/WKN pattern
	wkn := extractString(text, `/([A-Z0-9]{6})[)\]]`)
	if wkn == "" {
		wkn = extractWKN(text)
	}

	transaction := &schema.Transaction{
		DocumentType:           "DIVIDEND",
		ISIN:                   isin,
		WKN:                    wkn,
		Quantity:               ptr(quantity),
		DistributionPerShare:   ptr(distributionPerShare),
		DistributionCurrency:   distributionCurrency,
		GrossAmount:            ptr(grossAmount),
		GrossCurrency:          grossCurrency,
		WithholdingTax:         withholdingTax,
		WithholdingTaxCurrency: withholdingTaxCurrency,
		NetAmount:              ptr(netAmount),
		NetCurrency:            netCurrency,
		ExchangeRate:           exchangeRate,
		ExDate:                 exDate,
		ValueDate:              valueDate,
	}

	return transaction, nil
}

// parseInterest parses an INTEREST document.
func parseInterest(doc *extractor.ExtractedDocument) (*schema.Transaction, error) {
	text := doc.Text

	// Extract ISIN
	isin := extractISIN(text)
	if isin == "" {
		return nil, fmt.Errorf("ISIN not found in document")
	}

	// Extract value date (Valuta field)
	valueDateStr := extractString(text, `Valuta\s*:\s*(\d{2}\.\d{2}\.\d{4})`)
	if valueDateStr == "" {
		return nil, fmt.Errorf("value date not found in document")
	}
	valueDate := convertGermanDate(valueDateStr)

	// Extract gross amount
	grossAmount, err := extractFloat(text, `Bruttobetrag\s*:\s*([\d\s.,]+)\s*[A-Z]{3}`)
	if err != nil {
		return nil, fmt.Errorf("gross amount not found: %w", err)
	}

	// Extract gross currency
	grossCurrency := extractString(text, `Bruttobetrag\s*:\s*[\d\s.,]+\s*([A-Z]{3})`)
	if grossCurrency == "" {
		grossCurrency = "EUR"
	}

	// Extract withholding tax
	withholdingTaxVal, err := extractFloat(text, `Einbeh\.\s*KESt\s*:\s*(-?[\d\s.,]+)\s*[A-Z]{3}`)
	if err != nil {
		return nil, fmt.Errorf("withholding tax not found: %w", err)
	}
	withholdingTax := &withholdingTaxVal

	// Extract withholding tax currency
	withholdingTaxCurrency := extractString(text, `Einbeh\.\s*KESt\s*:\s*-?[\d\s.,]+\s*([A-Z]{3})`)
	if withholdingTaxCurrency == "" {
		withholdingTaxCurrency = "EUR"
	}

	// Extract net amount (Endbetrag)
	netAmount, err := extractFloat(text, `Endbetrag\s*:\s*([\d\s.,]+)\s*[A-Z]{3}`)
	if err != nil {
		return nil, fmt.Errorf("net amount not found: %w", err)
	}

	// Extract net currency
	netCurrency := extractString(text, `Endbetrag\s*:\s*[\d\s.,]+\s*([A-Z]{3})`)
	if netCurrency == "" {
		netCurrency = "EUR"
	}

	// Extract interest rate (Zinssatz)
	interestRate, err := extractFloat(text, `Zinssatz\s*:\s*([\d\s.,]+)\s*%`)
	if err != nil {
		return nil, fmt.Errorf("interest rate not found: %w", err)
	}

	// Extract period (e.g., "01.01.2026 bis 31.03.2026")
	periodFromStr := extractString(text, `(\d{2}\.\d{2}\.\d{4})\s*bis\s*\d{2}\.\d{2}\.\d{4}`)
	periodToStr := extractString(text, `\d{2}\.\d{2}\.\d{4}\s*bis\s*(\d{2}\.\d{2}\.\d{4})`)

	periodFrom := convertGermanDate(periodFromStr)
	periodTo := convertGermanDate(periodToStr)

	// Extract WKN from ISIN/WKN pattern
	wkn := extractString(text, `/([A-Z0-9]{6})[)\]]`)
	if wkn == "" {
		wkn = extractWKN(text)
	}

	transaction := &schema.Transaction{
		DocumentType:           "INTEREST",
		ISIN:                   isin,
		WKN:                    wkn,
		ValueDate:              valueDate,
		GrossAmount:            ptr(grossAmount),
		GrossCurrency:          grossCurrency,
		WithholdingTax:         withholdingTax,
		WithholdingTaxCurrency: withholdingTaxCurrency,
		NetAmount:              ptr(netAmount),
		NetCurrency:            netCurrency,
		InterestRate:           ptr(interestRate),
		PeriodFrom:             periodFrom,
		PeriodTo:               periodTo,
	}

	return transaction, nil
}

// parseAccumulating parses a ACCUMULATING (reinvestment/accumulation) document.
func parseAccumulating(doc *extractor.ExtractedDocument) (*schema.Transaction, error) {
	text := doc.Text

	// Extract ISIN
	isin := extractISIN(text)
	if isin == "" {
		return nil, fmt.Errorf("ISIN not found in document")
	}

	// Extract value date (Valuta field) - serves as main date for thesaurierung
	valueDateStr := extractString(text, `Valuta\s*:\s*(\d{2}\.\d{2}\.\d{4})`)
	if valueDateStr == "" {
		return nil, fmt.Errorf("value date not found in document")
	}
	valueDate := convertGermanDate(valueDateStr)

	// Extract ex-date (Extag field - optional)
	exDateStr := extractString(text, `Extag\s*:\s*(\d{2}\.\d{2}\.\d{4})`)
	exDate := convertGermanDate(exDateStr)

	// Extract accrual date (Fälligkeitstag field - optional)
	accrualDateStr := extractString(text, `Fälligkeitstag\s*:\s*(\d{2}\.\d{2}\.\d{4})`)
	accrualDate := convertGermanDate(accrualDateStr)

	// Extract quantity (shares held - St. field)
	quantity, err := extractFloat(text, `St\.\s*:\s*([\d\s.,]+)\s*(?:Brutto|pro)`)
	if err != nil {
		return nil, fmt.Errorf("quantity not found: %w", err)
	}

	// Extract reinvestment per share (pro Stück field)
	// Handles negative amounts (e.g., "-0,572 USD")
	reinvestmentPerShare, err := extractFloat(text, `pro Stück\s*:\s*([-\d\s.,]+)\s*[A-Z]{3}`)
	if err != nil {
		return nil, fmt.Errorf("reinvestment per share not found: %w", err)
	}

	// Extract reinvestment currency
	reinvestmentCurrency := extractString(text, `pro Stück\s*:\s*[-\d\s.,]+\s*([A-Z]{3})`)
	if reinvestmentCurrency == "" {
		reinvestmentCurrency = "EUR"
	}

	// Extract gross amount (Bruttothesaurierung - can be negative)
	grossAmount, err := extractFloat(text, `Bruttothesaurierung\s*:\s*([-\d\s.,]+)\s*[A-Z]{3}`)
	if err != nil {
		return nil, fmt.Errorf("gross amount not found: %w", err)
	}

	// Extract gross currency
	grossCurrency := extractString(text, `Bruttothesaurierung\s*:\s*[-\d\s.,]+\s*([A-Z]{3})`)
	if grossCurrency == "" {
		grossCurrency = "EUR"
	}

	// Extract withholding tax (Einbeh. Steuer). Absent stays nil rather than
	// collapsing to 0, so "no tax line" is distinguishable from "0,00 EUR".
	withholdingTax := floatPtr(extractFloat(text, `Einbeh\.\s*Steuer\s*:\s*([-\d\s.,]+)\s*[A-Z]{3}`))

	// Extract withholding tax currency
	withholdingTaxCurrency := extractString(text, `Einbeh\.\s*Steuer\s*:\s*[-\d\s.,]+\s*([A-Z]{3})`)
	if withholdingTaxCurrency == "" {
		withholdingTaxCurrency = "EUR"
	}

	// Devisenkurs, when the document prints one. A EUR settlement carries no
	// such line, and the field is then absent rather than filled in with 1:
	// an unstated rate is not an extracted value.
	exchangeRate := floatPtr(extractFloat(text, `Devisenkurs\s*:\s*([\d\s.,]+)`))

	// Extract WKN from ISIN/WKN pattern
	wkn := extractString(text, `/([A-Z0-9]{6})[)\]]`)
	if wkn == "" {
		wkn = extractWKN(text)
	}

	transaction := &schema.Transaction{
		DocumentType:           "ACCUMULATING",
		ISIN:                   isin,
		WKN:                    wkn,
		Quantity:               ptr(quantity),
		ReinvestmentPerShare:   ptr(reinvestmentPerShare),
		ReinvestmentCurrency:   reinvestmentCurrency,
		GrossAmount:            ptr(grossAmount),
		GrossCurrency:          grossCurrency,
		WithholdingTax:         withholdingTax,
		WithholdingTaxCurrency: withholdingTaxCurrency,
		ExchangeRate:           exchangeRate,
		ExDate:                 exDate,
		ValueDate:              valueDate,
		AccrualDate:            accrualDate,
	}

	return transaction, nil
}

// parseSavingsPlan parses a "Sammelabrechnung aus" — an annual savings-plan
// (Sparplan) settlement that lists each executed order as a table row.
// Returns one Transaction per row; ISIN and order number are shared across
// all rows.
func parseSavingsPlan(doc *extractor.ExtractedDocument) ([]*schema.Transaction, error) {
	text := doc.Text

	isin := extractISIN(text)
	if isin == "" {
		return nil, fmt.Errorf("ISIN not found in document")
	}

	wkn := extractString(text, `/([A-Z0-9]{6})[)\]]`)
	orderNumber := extractString(text, `Auftrags-Nr\s*:?\s*(\d+)`)
	securityName := strings.TrimSpace(extractString(text, `Bezeichnung\s*:([^\n]+)`))

	// Each row: K/V  Buchtag  Valuta  Stücke/Nom.  Ausf.-Kurs  EUR  Betrag  EUR
	rowRe := regexp.MustCompile(
		`(Kauf|Verkauf)\s+(\d{2}\.\d{2}\.\d{4})\s+(\d{2}\.\d{2}\.\d{4})\s+([\d,]+)\s+([\d.,]+)\s+EUR\s+([\d.,]+)\s+EUR`,
	)

	var txns []*schema.Transaction
	for _, m := range rowRe.FindAllStringSubmatch(text, -1) {
		tradeType := "BUY"
		if strings.ToLower(m[1]) == "verkauf" {
			tradeType = "SELL"
		}

		quantity := mustFloat(m[4])
		price := mustFloat(m[5])
		settled := mustFloat(m[6])

		// A row prints Stücke, Ausf.-Kurs and Betrag and nothing else — no
		// Kurswert line, no fee line. Those three are therefore all this row
		// yields. The gap between the cash that moved and the value of the
		// shares is the fee the document withholds, and it is checked for
		// plausibility here, but it is not reported: a recovered figure is not
		// an extracted one, and rebuilding it from a six-decimal quantity
		// gives it digits it has not earned.
		if err := checkSavingsPlanRow(tradeType, settled, schema.Mul(quantity, price)); err != nil {
			return nil, fmt.Errorf("savings-plan row %s: %w", m[2], err)
		}

		// Buys move cash out, sales move it in, matching NetAmount's sign
		// convention on trade confirmations.
		finalAmount := schema.Num(-settled.Float(), settled.Scale())
		if tradeType == "SELL" {
			finalAmount = settled
		}

		txns = append(txns, &schema.Transaction{
			DocumentType: "SAVINGSPLAN",
			ISIN:         isin,
			WKN:          wkn,
			OrderNumber:  orderNumber,
			SecurityName: securityName,
			BookingDate:  convertGermanDate(m[2]), // Buchtag — when the row was booked
			ValueDate:    convertGermanDate(m[3]), // Valuta — settlement
			Type:         tradeType,
			Quantity:     ptr(quantity),
			Price:        ptr(price),
			NetAmount:    ptr(finalAmount), // Betrag — the cash the row moved
			NetCurrency:  "EUR",
			// No GrossAmount, no Costs and no ExchangeRate: the row prints no
			// Kurswert, no charge line and no Devisenkurs, so there is nothing
			// to report for any of them.
		})
	}

	if len(txns) == 0 {
		return nil, fmt.Errorf("no rows found in Sammelabrechnung table")
	}
	return txns, nil
}

// hSpace matches horizontal whitespace only. Field patterns use it instead of
// \s so that a label whose value column is blank cannot reach across the line
// break and capture the next line's number or date.
const hSpace = `[^\S\n]*`

// dateField reads a "<label> [:] DD.MM.YYYY" header field and returns it as
// YYYY-MM-DD, or "" if the label has no date on its own line.
func dateField(text, label string) string {
	return convertGermanDate(extractString(text, label+hSpace+`:?`+hSpace+`(\d{2}\.\d{2}\.\d{4})`))
}

// timeField reads a "<label> [:] HH:MM Uhr" header field. The separating
// whitespace is optional because gxpdf runs the label into its value column on
// some layouts ("Ausführungszeit09:15 Uhr").
func timeField(text, label string) string {
	return extractString(text, label+hSpace+`:?`+hSpace+`(\d{2}:\d{2})`+hSpace+`Uhr`)
}

// eurField reads a "<label> [:] <amount> EUR" money line.
func eurField(text, label string) (schema.Decimal, error) {
	return extractFloat(text, label+hSpace+`:?`+hSpace+`(-?[\d.,]+)`+hSpace+`EUR`)
}

// ptr boxes an amount the parser has already established is present, for the
// fields whose extraction fails the whole parse when the label is missing.
func ptr(v schema.Decimal) *schema.Decimal { return &v }

// floatPtr adapts an (value, error) extraction pair to the nil-means-absent
// convention: nil when the label was not found, otherwise a pointer to the
// value — including a genuine 0,00.
func floatPtr(v schema.Decimal, err error) *schema.Decimal {
	if err != nil {
		return nil
	}
	return &v
}

// firstNonEmpty returns the first non-empty argument, or "".
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// extractCosts parses a settlement's charge block: Provision plus the two
// Spesen lines, and the itemised Gebühren breakdown when present. It returns
// nil when the document has no Provision line at all, so that a document
// without a cost section stays distinguishable from one that charged 0,00.
func extractCosts(text string) *schema.Costs {
	provision, err := eurField(text, `Provision`)
	if err != nil {
		return nil
	}
	own := feeLine(text, `Eigene[^\S\n]*Spesen`)
	foreign := feeLine(text, `Fremde[^\S\n]*Spesen`)

	// Total is the one figure here the document does not print. Summed
	// exactly, so it is the true total rather than a float sum snapped back
	// to the cent to hide its own error.
	c := &schema.Costs{
		Provision:       provision,
		OwnExpenses:     own,
		ForeignExpenses: foreign,
		Total:           schema.Sum(provision, own, foreign),
	}

	// The itemised lines are only meaningful under their "Enthalten sind
	// folgende Gebühren" heading, which marks them as the breakdown of
	// Fremde Spesen rather than additional charges.
	if strings.Contains(text, "Enthalten sind folgende Gebühren") {
		c.ForeignExpensesBreakdown = &schema.FeeBreakdown{
			Courtage:                feeLine(text, `Courtage`),
			TradingFee:              feeLine(text, `Tradinggebühr`),
			Settlement:              feeLine(text, `Regulierung`),
			ClosingNotes:            feeLine(text, `Schlussnoten`),
			LSAllocation:            feeLine(text, `LS-Umlegung`),
			FinancialTransactionTax: feeLine(text, `Finanztransaktionssteuer`),
			Other:                   feeLine(text, `Sonstige`),
		}
	}
	return c
}

// feeLine reads one line of the Gebühren itemisation. A line the block omits
// is a charge of nothing, reported as the 0.00 every other currency figure
// here carries rather than a bare 0 — the zero Decimal has no scale.
func feeLine(text, label string) schema.Decimal {
	f, err := eurField(text, label)
	if err != nil {
		return schema.Computed(0)
	}
	return f
}

// extractFloat extracts a number from text using a regex pattern, keeping the
// number of decimal places the document printed it with. Handles European
// decimal format (comma as decimal separator).
func extractFloat(text, pattern string) (schema.Decimal, error) {
	regex := regexp.MustCompile(pattern)
	matches := regex.FindStringSubmatch(text)
	if len(matches) < 2 {
		return schema.Decimal{}, fmt.Errorf("pattern not found: %s", pattern)
	}

	normalized := normalizeDecimal(matches[1])
	f, err := strconv.ParseFloat(normalized, 64)
	if err != nil {
		return schema.Decimal{}, fmt.Errorf("failed to parse float from '%s': %w", matches[1], err)
	}

	return schema.Num(f, schema.ScaleOf(normalized)), nil
}

// mustFloat parses a German/English-formatted number, returning 0 on failure.
// It too keeps the printed precision.
func mustFloat(s string) schema.Decimal {
	normalized := normalizeDecimal(s)
	f, _ := strconv.ParseFloat(normalized, 64)
	return schema.Num(f, schema.ScaleOf(normalized))
}

// convertGermanDate converts "DD.MM.YYYY" to "YYYY-MM-DD" (empty if not 3 parts).
func convertGermanDate(s string) string {
	p := strings.Split(s, ".")
	if len(p) != 3 {
		return ""
	}
	return fmt.Sprintf("%s-%s-%s", p[2], p[1], p[0])
}

// normalizeDecimal converts a German (1.234,56) or English (1,234.56) formatted
// number into a Go-parseable decimal. The rightmost of '.' or ',' is treated as
// the decimal separator; every other '.'/',' is a thousands separator and dropped.
// ponytail: a lone "1.234" is read as English 1.234, not German 1234 — that case
// is genuinely ambiguous without the document's locale; switch to locale-driven
// parsing if a real flatex field ever depends on it.
func normalizeDecimal(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, " ", "")

	lastDot := strings.LastIndex(s, ".")
	lastComma := strings.LastIndex(s, ",")

	var dec int // index of the decimal separator, -1 if none
	if lastDot > lastComma {
		dec = lastDot
	} else {
		dec = lastComma
	}

	var b strings.Builder
	for i, r := range s {
		switch r {
		case '.', ',':
			if i == dec {
				b.WriteByte('.')
			} // else: thousands separator, drop it
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// extractString extracts a string from text using a regex pattern and trims whitespace.
func extractString(text, pattern string) string {
	regex := regexp.MustCompile(pattern)
	matches := regex.FindStringSubmatch(text)
	if len(matches) < 2 {
		return ""
	}
	return strings.TrimSpace(matches[1])
}

// extractISIN extracts an ISIN code from text.
// ISIN format: [A-Z]{2}[A-Z0-9]{9}[0-9]
func extractISIN(text string) string {
	pattern := `([A-Z]{2}[A-Z0-9]{9}[0-9])`
	return extractString(text, pattern)
}

// isinChecksumValid reports whether s is a well-formed, 12-character ISIN
// whose trailing check digit is correct under the Luhn algorithm, as ISO 6166
// specifies.
func isinChecksumValid(s string) bool {
	if len(s) != 12 {
		return false
	}
	digits := make([]byte, 0, 24)
	for i := 0; i < len(s); i++ {
		switch c := s[i]; {
		case c >= '0' && c <= '9':
			digits = append(digits, c-'0')
		case c >= 'A' && c <= 'Z':
			n := int(c-'A') + 10 // letter -> two-digit value (A=10 ... Z=35)
			digits = append(digits, byte(n/10), byte(n%10))
		default:
			return false
		}
	}
	sum := 0
	for i, d := range digits {
		v := int(d)
		if (len(digits)-1-i)%2 == 1 { // double every second digit, from the check digit outward
			v *= 2
			if v > 9 {
				v -= 9
			}
		}
		sum += v
	}
	return sum%10 == 0
}

// extractWKN extracts a WKN (Wertpapierkennnummer) from text.
// WKN format: [A-Z0-9]{6}
func extractWKN(text string) string {
	pattern := `\b([A-Z0-9]{6})\b`
	return extractString(text, pattern)
}
