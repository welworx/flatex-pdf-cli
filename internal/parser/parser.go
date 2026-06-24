package parser

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/welworx/flatex-pdf-cli/internal/extractor"
	"github.com/welworx/flatex-pdf-cli/internal/schema"
)

// Parse routes an ExtractedDocument to the appropriate parser based on DocumentType.
func Parse(doc *extractor.ExtractedDocument) (*schema.Transaction, error) {
	switch doc.DocumentType {
	case "TRADE":
		return ParseTrade(doc)
	case "DIVIDEND":
		return ParseDividend(doc)
	case "INTEREST":
		return ParseInterest(doc)
	case "THESAURIERUNG":
		return ParseThesaurierung(doc)
	default:
		return nil, fmt.Errorf("unknown document type: %s", doc.DocumentType)
	}
}

// ParseTrade parses a TRADE document.
func ParseTrade(doc *extractor.ExtractedDocument) (*schema.Transaction, error) {
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

	transaction := &schema.Transaction{
		Source:        doc.Filename,
		DocumentType:  "TRADE",
		ISIN:          isin,
		WKN:           wkn,
		Date:          date,
		Type:          tradeType,
		Quantity:      quantity,
		Price:         price,
		PriceCurrency: currency,
		GrossValue:    grossValue,
		Provision:     provision,
		ExchangeRate:  exchangeRate,
	}

	return transaction, nil
}

// ParseDividend parses a DIVIDEND document.
func ParseDividend(doc *extractor.ExtractedDocument) (*schema.Transaction, error) {
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
	// Convert DD.MM.YYYY to YYYY-MM-DD
	parts := strings.Split(valueDateStr, ".")
	var valueDate string
	if len(parts) == 3 {
		valueDate = fmt.Sprintf("%s-%s-%s", parts[2], parts[1], parts[0])
	}

	// Extract ex-date (Extag field - may contain different date)
	exDateStr := extractString(text, `Extag\s*:\s*(\d{2}\.\d{2}\.\d{4})`)
	var exDate string
	if exDateStr != "" {
		// Convert DD.MM.YYYY to YYYY-MM-DD
		parts := strings.Split(exDateStr, ".")
		if len(parts) == 3 {
			exDate = fmt.Sprintf("%s-%s-%s", parts[2], parts[1], parts[0])
		}
	}

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
		Source:                 doc.Filename,
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

// ParseInterest parses an INTEREST document.
func ParseInterest(doc *extractor.ExtractedDocument) (*schema.Transaction, error) {
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
	// Convert DD.MM.YYYY to YYYY-MM-DD
	parts := strings.Split(valueDateStr, ".")
	var valueDate string
	if len(parts) == 3 {
		valueDate = fmt.Sprintf("%s-%s-%s", parts[2], parts[1], parts[0])
	}

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

	var periodFrom, periodTo string
	if periodFromStr != "" {
		parts := strings.Split(periodFromStr, ".")
		if len(parts) == 3 {
			periodFrom = fmt.Sprintf("%s-%s-%s", parts[2], parts[1], parts[0])
		}
	}
	if periodToStr != "" {
		parts := strings.Split(periodToStr, ".")
		if len(parts) == 3 {
			periodTo = fmt.Sprintf("%s-%s-%s", parts[2], parts[1], parts[0])
		}
	}

	// Extract WKN from ISIN/WKN pattern
	wkn := extractString(text, `/([A-Z0-9]{6})[)\]]`)
	if wkn == "" {
		wkn = extractWKN(text)
	}

	transaction := &schema.Transaction{
		Source:                 doc.Filename,
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

// ParseThesaurierung parses a THESAURIERUNG (reinvestment/accumulation) document.
func ParseThesaurierung(doc *extractor.ExtractedDocument) (*schema.Transaction, error) {
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
	// Convert DD.MM.YYYY to YYYY-MM-DD
	parts := strings.Split(valueDateStr, ".")
	var valueDate string
	if len(parts) == 3 {
		valueDate = fmt.Sprintf("%s-%s-%s", parts[2], parts[1], parts[0])
	}

	// Extract ex-date (Extag field - optional)
	exDateStr := extractString(text, `Extag\s*:\s*(\d{2}\.\d{2}\.\d{4})`)
	var exDate string
	if exDateStr != "" {
		// Convert DD.MM.YYYY to YYYY-MM-DD
		parts := strings.Split(exDateStr, ".")
		if len(parts) == 3 {
			exDate = fmt.Sprintf("%s-%s-%s", parts[2], parts[1], parts[0])
		}
	}

	// Extract accrual date (Fälligkeitstag field - optional)
	accrualDateStr := extractString(text, `Fälligkeitstag\s*:\s*(\d{2}\.\d{2}\.\d{4})`)
	var accrualDate string
	if accrualDateStr != "" {
		// Convert DD.MM.YYYY to YYYY-MM-DD
		parts := strings.Split(accrualDateStr, ".")
		if len(parts) == 3 {
			accrualDate = fmt.Sprintf("%s-%s-%s", parts[2], parts[1], parts[0])
		}
	}

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
		Source:                 doc.Filename,
		DocumentType:           "THESAURIERUNG",
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

// extractFloat extracts a float from text using a regex pattern.
// Handles European decimal format (comma as decimal separator).
func extractFloat(text, pattern string) (float64, error) {
	regex := regexp.MustCompile(pattern)
	matches := regex.FindStringSubmatch(text)
	if len(matches) < 2 {
		return 0, fmt.Errorf("pattern not found: %s", pattern)
	}

	// Replace European decimal separator (comma) with dot
	value := strings.ReplaceAll(matches[1], ",", ".")
	// Remove any thousand separators (spaces or dots that precede comma)
	value = strings.ReplaceAll(value, " ", "")

	f, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse float from '%s': %w", matches[1], err)
	}

	return f, nil
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
