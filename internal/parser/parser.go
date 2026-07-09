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
// DocumentType. It returns a slice because some document types (e.g. order
// confirmations) contain multiple transactions.
func Parse(doc *extractor.ExtractedDocument) ([]*schema.Transaction, error) {
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

	// Extract date
	date := extractDate(text)
	if date == "" {
		return nil, fmt.Errorf("date not found in document")
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

	// Extract provision (fees)
	provision, err := extractFloat(text, `Provision\s*:\s*([\d\s.,]+)\s*EUR`)
	if err != nil {
		// Default to 0 if not found (some trades may have no provision)
		provision = 0
	}

	// Extract exchange rate (optional, default to 1.0)
	exchangeRate, err := extractFloat(text, `Devisenkurs\s*:\s*([\d\s.,]+)`)
	if err != nil {
		exchangeRate = 1.0
	}

	// Extract WKN from ISIN/WKN pattern (e.g., "IE000YU9K6K2/A3DP9J")
	wkn := extractString(text, `/([A-Z0-9]{6})[)\]]`)
	if wkn == "" {
		// Fallback to general WKN extraction
		wkn = extractWKN(text)
	}

	// Extract identifiers (all optional)
	orderNumber := extractString(text, `Auftragsnummer\s*:?\s*(\S+)`)
	transactionNumber := extractString(text, `Transaktion-Nr\.\s*:?\s*(\d+)`)
	executionVenue := extractString(text, `Ausf\.platz/-art\s*([^\n]+)`)

	transaction := &schema.Transaction{
		OrderNumber:       orderNumber,
		TransactionNumber: transactionNumber,
		DocumentType:      "TRADE",
		ISIN:              isin,
		WKN:               wkn,
		Date:              date,
		Type:              tradeType,
		Quantity:          quantity,
		Price:             price,
		PriceCurrency:     currency,
		GrossValue:        grossValue,
		Provision:         provision,
		ExchangeRate:      exchangeRate,
		ExecutionVenue:    executionVenue,
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
	date := convertGermanDate(extractString(text, `Schlusstag:\s*(\d{2}\.\d{2}\.\d{4})`))
	if date == "" {
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

	provision, _ := extractFloat(text, `Provision:\s*([\d.,]+)\s*EUR`)
	withholdingTax, _ := extractFloat(text, `Einbeh\. Steuer:\s*([\d.,]+)\s*EUR`)
	gainLoss, _ := extractFloat(text, `Gewinn/Verlust:\s*(-?[\d.,]+)\s*EUR`)
	finalAmount, _ := extractFloat(text, `Endbetrag:\s*(-?[\d.,]+)\s*EUR`)

	exchangeRate, err := extractFloat(text, `Devisenkurs:\s*([\d.,]+)`)
	if err != nil {
		exchangeRate = 1.0
	}

	return &schema.Transaction{
		OrderNumber:       extractString(text, `Nr\.([\d/]+)`),
		TransactionNumber: extractString(text, `Transaktion-Nr\.:\s*(\d+)`),
		DocumentType:      "CRYPTO",
		SecurityName:      name,
		Date:              date,
		Type:              tradeType,
		Quantity:          quantity,
		Price:             price,
		PriceCurrency:     "EUR",
		GrossValue:        grossValue,
		Provision:         provision,
		WithholdingTax:    withholdingTax,
		GainLoss:          gainLoss,
		ExchangeRate:      exchangeRate,
		FinalAmount:       finalAmount,
		FinalCurrency:     "EUR",
		CustodyType:       extractString(text, `Verwahrart:\s*([^\n*]+)`),
		Depositary:        extractString(text, `Kryptoverwahrer:\s*([^\n*]+)`),
		ValueDate:         convertGermanDate(extractString(text, `Valuta:\s*(\d{2}\.\d{2}\.\d{4})`)),
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
			Date:         convertGermanDate(m[6]),
			Quantity:     mustFloat(m[7]),
			ValidUntil:   convertGermanDate(m[8]),
			Limit:        mustFloat(m[9]),
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
	withholdingTax, err := extractFloat(text, `Einbeh\.\s*Steuer\s*:\s*([\d\s.,]+)\s*[A-Z]{3}`)
	if err != nil {
		return nil, fmt.Errorf("withholding tax not found: %w", err)
	}

	// Extract withholding tax currency
	withholdingTaxCurrency := extractString(text, `Einbeh\.\s*Steuer\s*:\s*[\d\s.,]+\s*([A-Z]{3})`)
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

	// Extract exchange rate (optional, default to 1.0)
	exchangeRate, err := extractFloat(text, `Devisenkurs\s*:\s*([\d.,]+)`)
	if err != nil {
		exchangeRate = 1.0
	}

	// Extract WKN from ISIN/WKN pattern
	wkn := extractString(text, `/([A-Z0-9]{6})[)\]]`)
	if wkn == "" {
		wkn = extractWKN(text)
	}

	transaction := &schema.Transaction{
		DocumentType:           "DIVIDEND",
		ISIN:                   isin,
		WKN:                    wkn,
		Date:                   valueDate,
		Quantity:               quantity,
		DistributionPerShare:   distributionPerShare,
		DistributionCurrency:   distributionCurrency,
		GrossAmount:            grossAmount,
		GrossCurrency:          grossCurrency,
		WithholdingTax:         withholdingTax,
		WithholdingTaxCurrency: withholdingTaxCurrency,
		NetAmount:              netAmount,
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
	withholdingTax, err := extractFloat(text, `Einbeh\.\s*KESt\s*:\s*([\d\s.,]+)\s*[A-Z]{3}`)
	if err != nil {
		return nil, fmt.Errorf("withholding tax not found: %w", err)
	}

	// Extract withholding tax currency
	withholdingTaxCurrency := extractString(text, `Einbeh\.\s*KESt\s*:\s*[\d\s.,]+\s*([A-Z]{3})`)
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
		Date:                   valueDate,
		GrossAmount:            grossAmount,
		GrossCurrency:          grossCurrency,
		WithholdingTax:         withholdingTax,
		WithholdingTaxCurrency: withholdingTaxCurrency,
		NetAmount:              netAmount,
		NetCurrency:            netCurrency,
		InterestRate:           interestRate,
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

	// Extract withholding tax (Einbeh. Steuer)
	withholdingTax, err := extractFloat(text, `Einbeh\.\s*Steuer\s*:\s*([-\d\s.,]+)\s*[A-Z]{3}`)
	if err != nil {
		// Default to 0 if not found
		withholdingTax = 0
	}

	// Extract withholding tax currency
	withholdingTaxCurrency := extractString(text, `Einbeh\.\s*Steuer\s*:\s*[-\d\s.,]+\s*([A-Z]{3})`)
	if withholdingTaxCurrency == "" {
		withholdingTaxCurrency = "EUR"
	}

	// Extract exchange rate (optional, default to 1.0)
	exchangeRate, err := extractFloat(text, `Devisenkurs\s*:\s*([\d\s.,]+)`)
	if err != nil {
		exchangeRate = 1.0
	}

	// Extract WKN from ISIN/WKN pattern
	wkn := extractString(text, `/([A-Z0-9]{6})[)\]]`)
	if wkn == "" {
		wkn = extractWKN(text)
	}

	transaction := &schema.Transaction{
		DocumentType:           "ACCUMULATING",
		ISIN:                   isin,
		WKN:                    wkn,
		Date:                   valueDate,
		Quantity:               quantity,
		ReinvestmentPerShare:   reinvestmentPerShare,
		ReinvestmentCurrency:   reinvestmentCurrency,
		GrossAmount:            grossAmount,
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
		`(Kauf|Verkauf)\s+(\d{2}\.\d{2}\.\d{4})\s+\d{2}\.\d{2}\.\d{4}\s+([\d,]+)\s+([\d.,]+)\s+EUR\s+([\d.,]+)\s+EUR`,
	)

	var txns []*schema.Transaction
	for _, m := range rowRe.FindAllStringSubmatch(text, -1) {
		tradeType := "BUY"
		if strings.ToLower(m[1]) == "verkauf" {
			tradeType = "SELL"
		}
		txns = append(txns, &schema.Transaction{
			DocumentType:  "SAVINGSPLAN",
			ISIN:          isin,
			WKN:           wkn,
			OrderNumber:   orderNumber,
			SecurityName:  securityName,
			Date:          convertGermanDate(m[2]),
			Type:          tradeType,
			Quantity:      mustFloat(m[3]),
			Price:         mustFloat(m[4]),
			PriceCurrency: "EUR",
			GrossValue:    mustFloat(m[5]),
		})
	}

	if len(txns) == 0 {
		return nil, fmt.Errorf("no rows found in Sammelabrechnung table")
	}
	return txns, nil
}

// extractFloat extracts a float from text using a regex pattern.
// Handles European decimal format (comma as decimal separator).
func extractFloat(text, pattern string) (float64, error) {
	regex := regexp.MustCompile(pattern)
	matches := regex.FindStringSubmatch(text)
	if len(matches) < 2 {
		return 0, fmt.Errorf("pattern not found: %s", pattern)
	}

	f, err := strconv.ParseFloat(normalizeDecimal(matches[1]), 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse float from '%s': %w", matches[1], err)
	}

	return f, nil
}

// mustFloat parses a German/English-formatted number, returning 0 on failure.
func mustFloat(s string) float64 {
	f, _ := strconv.ParseFloat(normalizeDecimal(s), 64)
	return f
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

// extractWKN extracts a WKN (Wertpapierkennnummer) from text.
// WKN format: [A-Z0-9]{6}
func extractWKN(text string) string {
	pattern := `\b([A-Z0-9]{6})\b`
	return extractString(text, pattern)
}

// extractDate extracts a date in DD.MM.YYYY format and converts to YYYY-MM-DD.
func extractDate(text string) string {
	pattern := `(\d{2})\.(\d{2})\.(\d{4})`
	regex := regexp.MustCompile(pattern)
	matches := regex.FindStringSubmatch(text)
	if len(matches) < 4 {
		return ""
	}

	day := matches[1]
	month := matches[2]
	year := matches[3]

	return fmt.Sprintf("%s-%s-%s", year, month, day)
}
