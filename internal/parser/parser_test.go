package parser

import (
	"testing"

	"github.com/welworx/flatex-pdf-cli/internal/extractor"
)

// TestParseRouting tests that the Parse function routes TRADE documents correctly.
func TestParseRouting(t *testing.T) {
	text := "Kauf VANECK SPACE INNOVATORS E (IE000YU9K6K2/A3DP9J)\nAusgeführt : 1,058537 St. Kurswert : 50,00 EUR\nKurs : 47,235000 EUR Provision : 0,00 EUR\nDevisenkurs : 1,000000\nAusführungsdatum : 15.06.2026"
	doc := &extractor.ExtractedDocument{
		Filename:     "trade.pdf",
		Text:         text,
		DocumentType: "TRADE",
	}

	_, err := Parse(doc)
	if err != nil {
		t.Errorf("Parse should successfully route TRADE document, got error: %v", err)
	}
}

// TestParseDividendRouting tests that the Parse function routes DIVIDEND documents correctly.
func TestParseDividendRouting(t *testing.T) {
	text := "Nr.4684511050 VANGUARD FTSE ALL-WLD UCI (IE00B3RBWM25/A1JX52)\nSt. : 78,70 Bruttoausschüttung\npro Stück : 0,5459180 USD\nExtag : 18.12.2025 Bruttoausschüttung : 42,96 USD\nValuta : 01.01.2026\n*Einbeh. Steuer : 5,39 EUR\nDevisenkurs : 1,175000\nEndbetrag : 31,17 EUR"
	doc := &extractor.ExtractedDocument{
		Filename:     "dividend.pdf",
		Text:         text,
		DocumentType: "DIVIDEND",
	}

	_, err := Parse(doc)
	if err != nil {
		t.Errorf("Parse should successfully route DIVIDEND document, got error: %v", err)
	}
}

// TestParseThesaurierungRouting tests that the Parse function routes THESAURIERUNG documents correctly.
func TestParseThesaurierungRouting(t *testing.T) {
	text := "Nr.4684511050 XTRACKERS IE00 (IE00B5L8K969/A2H514)\nSt. : 4,75 pro Stück : -0,572 USD\nExtag : 15.06.2026 Bruttothesaurierung : -2,72 USD\nValuta : 30.06.2026\nEinbeh. Steuer : 0,00 EUR\nDevisenkurs : 1,080000"
	doc := &extractor.ExtractedDocument{
		Filename:     "thesaurierung.pdf",
		Text:         text,
		DocumentType: "THESAURIERUNG",
	}

	_, err := Parse(doc)
	if err != nil {
		t.Errorf("Parse should successfully route THESAURIERUNG document, got error: %v", err)
	}
}

// TestParseTradeBuy tests parsing a BUY trade confirmation.
func TestParseTradeBuy(t *testing.T) {
	text := "Kauf VANECK SPACE INNOVATORS E (IE000YU9K6K2/A3DP9J)\nAusgeführt : 1,058537 St. Kurswert : 50,00 EUR\nKurs : 47,235000 EUR Provision : 0,00 EUR\nDevisenkurs : 1,000000\nAusführungsdatum : 15.06.2026"
	doc := &extractor.ExtractedDocument{
		Filename:     "trade_buy.pdf",
		Text:         text,
		DocumentType: "TRADE",
	}

	tx, err := ParseTrade(doc)
	if err != nil {
		t.Fatalf("ParseTrade failed: %v", err)
	}

	// Verify core fields
	if tx.Type != "BUY" {
		t.Errorf("expected Type=BUY, got %s", tx.Type)
	}
	if tx.ISIN != "IE000YU9K6K2" {
		t.Errorf("expected ISIN=IE000YU9K6K2, got %s", tx.ISIN)
	}
	if tx.WKN != "A3DP9J" {
		t.Errorf("expected WKN=A3DP9J, got %s", tx.WKN)
	}
	if tx.Quantity != 1.058537 {
		t.Errorf("expected Quantity=1.058537, got %f", tx.Quantity)
	}
	if tx.Price != 47.235 {
		t.Errorf("expected Price=47.235, got %f", tx.Price)
	}
	if tx.PriceCurrency != "EUR" {
		t.Errorf("expected PriceCurrency=EUR, got %s", tx.PriceCurrency)
	}
	if tx.GrossValue != 50.00 {
		t.Errorf("expected GrossValue=50.00, got %f", tx.GrossValue)
	}
	if tx.Provision != 0.00 {
		t.Errorf("expected Provision=0.00, got %f", tx.Provision)
	}
}

// TestParseDividend tests parsing a DIVIDEND statement.
func TestParseDividend(t *testing.T) {
	text := "Nr.4684511050 VANGUARD FTSE ALL-WLD UCI (IE00B3RBWM25/A1JX52)\nSt. : 78,70 Bruttoausschüttung\npro Stück : 0,5459180 USD\nExtag : 18.12.2025 Bruttoausschüttung : 42,96 USD\nValuta : 01.01.2026\n*Einbeh. Steuer : 5,39 EUR\nDevisenkurs : 1,175000\nEndbetrag : 31,17 EUR"
	doc := &extractor.ExtractedDocument{
		Filename:     "dividend.pdf",
		Text:         text,
		DocumentType: "DIVIDEND",
	}

	tx, err := ParseDividend(doc)
	if err != nil {
		t.Fatalf("ParseDividend failed: %v", err)
	}

	// Verify core fields
	if tx.DocumentType != "DIVIDEND" {
		t.Errorf("expected DocumentType=DIVIDEND, got %s", tx.DocumentType)
	}
	if tx.ISIN != "IE00B3RBWM25" {
		t.Errorf("expected ISIN=IE00B3RBWM25, got %s", tx.ISIN)
	}
	if tx.WKN != "A1JX52" {
		t.Errorf("expected WKN=A1JX52, got %s", tx.WKN)
	}
	if tx.Quantity != 78.70 {
		t.Errorf("expected Quantity=78.70, got %f", tx.Quantity)
	}
	if tx.DistributionPerShare != 0.5459180 {
		t.Errorf("expected DistributionPerShare=0.5459180, got %f", tx.DistributionPerShare)
	}
	if tx.DistributionCurrency != "USD" {
		t.Errorf("expected DistributionCurrency=USD, got %s", tx.DistributionCurrency)
	}
	if tx.GrossAmount != 42.96 {
		t.Errorf("expected GrossAmount=42.96, got %f", tx.GrossAmount)
	}
	if tx.GrossCurrency != "USD" {
		t.Errorf("expected GrossCurrency=USD, got %s", tx.GrossCurrency)
	}
	if tx.WithholdingTax != 5.39 {
		t.Errorf("expected WithholdingTax=5.39, got %f", tx.WithholdingTax)
	}
	if tx.WithholdingTaxCurrency != "EUR" {
		t.Errorf("expected WithholdingTaxCurrency=EUR, got %s", tx.WithholdingTaxCurrency)
	}
	if tx.NetAmount != 31.17 {
		t.Errorf("expected NetAmount=31.17, got %f", tx.NetAmount)
	}
	if tx.NetCurrency != "EUR" {
		t.Errorf("expected NetCurrency=EUR, got %s", tx.NetCurrency)
	}
	if tx.ExchangeRate != 1.175 {
		t.Errorf("expected ExchangeRate=1.175, got %f", tx.ExchangeRate)
	}
	if tx.ExDate != "2025-12-18" {
		t.Errorf("expected ExDate=2025-12-18, got %s", tx.ExDate)
	}
	if tx.ValueDate != "2026-01-01" {
		t.Errorf("expected ValueDate=2026-01-01, got %s", tx.ValueDate)
	}
}

// TestParseInterestRouting tests that the Parse function routes INTEREST documents correctly.
func TestParseInterestRouting(t *testing.T) {
	text := "ISIN: IE00B3RBWM25\nBruttobetrag : 25,50 EUR\nEinbeh. KESt : 3,40 EUR\nEndbetrag : 22,10 EUR\nZinssatz : 2,5%\nZinsperiode : 01.01.2026 bis 31.03.2026\nValuta : 15.04.2026"
	doc := &extractor.ExtractedDocument{
		Filename:     "interest.pdf",
		Text:         text,
		DocumentType: "INTEREST",
	}

	_, err := Parse(doc)
	if err != nil {
		t.Errorf("Parse should successfully route INTEREST document, got error: %v", err)
	}
}

// TestParseInterest tests parsing an INTEREST statement.
func TestParseInterest(t *testing.T) {
	text := "ISIN: IE00B3RBWM25\nBruttobetrag : 25,50 EUR\nEinbeh. KESt : 3,40 EUR\nEndbetrag : 22,10 EUR\nZinssatz : 2,5%\nZinsperiode : 01.01.2026 bis 31.03.2026\nValuta : 15.04.2026"
	doc := &extractor.ExtractedDocument{
		Filename:     "interest.pdf",
		Text:         text,
		DocumentType: "INTEREST",
	}

	tx, err := ParseInterest(doc)
	if err != nil {
		t.Fatalf("ParseInterest failed: %v", err)
	}

	// Verify core fields
	if tx.DocumentType != "INTEREST" {
		t.Errorf("expected DocumentType=INTEREST, got %s", tx.DocumentType)
	}
	if tx.ISIN != "IE00B3RBWM25" {
		t.Errorf("expected ISIN=IE00B3RBWM25, got %s", tx.ISIN)
	}
	if tx.GrossAmount != 25.50 {
		t.Errorf("expected GrossAmount=25.50, got %f", tx.GrossAmount)
	}
	if tx.GrossCurrency != "EUR" {
		t.Errorf("expected GrossCurrency=EUR, got %s", tx.GrossCurrency)
	}
	if tx.WithholdingTax != 3.40 {
		t.Errorf("expected WithholdingTax=3.40, got %f", tx.WithholdingTax)
	}
	if tx.WithholdingTaxCurrency != "EUR" {
		t.Errorf("expected WithholdingTaxCurrency=EUR, got %s", tx.WithholdingTaxCurrency)
	}
	if tx.NetAmount != 22.10 {
		t.Errorf("expected NetAmount=22.10, got %f", tx.NetAmount)
	}
	if tx.NetCurrency != "EUR" {
		t.Errorf("expected NetCurrency=EUR, got %s", tx.NetCurrency)
	}
	if tx.InterestRate != 2.5 {
		t.Errorf("expected InterestRate=2.5, got %f", tx.InterestRate)
	}
	if tx.PeriodFrom != "2026-01-01" {
		t.Errorf("expected PeriodFrom=2026-01-01, got %s", tx.PeriodFrom)
	}
	if tx.PeriodTo != "2026-03-31" {
		t.Errorf("expected PeriodTo=2026-03-31, got %s", tx.PeriodTo)
	}
	if tx.Date != "2026-04-15" {
		t.Errorf("expected Date=2026-04-15, got %s", tx.Date)
	}
}

// TestParseThesaurierung tests parsing a THESAURIERUNG (reinvestment) statement.
func TestParseThesaurierung(t *testing.T) {
	text := "Nr.4684511050 XTRACKERS IE00 (IE00B5L8K969/A2H514)\nSt. : 4,75 pro Stück : -0,572 USD\nExtag : 15.06.2026 Bruttothesaurierung : -2,72 USD\nValuta : 30.06.2026\nEinbeh. Steuer : 0,00 EUR\nDevisenkurs : 1,080000"
	doc := &extractor.ExtractedDocument{
		Filename:     "thesaurierung.pdf",
		Text:         text,
		DocumentType: "THESAURIERUNG",
	}

	tx, err := ParseThesaurierung(doc)
	if err != nil {
		t.Fatalf("ParseThesaurierung failed: %v", err)
	}

	// Verify core fields
	if tx.DocumentType != "THESAURIERUNG" {
		t.Errorf("expected DocumentType=THESAURIERUNG, got %s", tx.DocumentType)
	}
	if tx.ISIN != "IE00B5L8K969" {
		t.Errorf("expected ISIN=IE00B5L8K969, got %s", tx.ISIN)
	}
	if tx.WKN != "A2H514" {
		t.Errorf("expected WKN=A2H514, got %s", tx.WKN)
	}
	if tx.Quantity != 4.75 {
		t.Errorf("expected Quantity=4.75, got %f", tx.Quantity)
	}
	if tx.ReinvestmentPerShare != -0.572 {
		t.Errorf("expected ReinvestmentPerShare=-0.572, got %f", tx.ReinvestmentPerShare)
	}
	if tx.ReinvestmentCurrency != "USD" {
		t.Errorf("expected ReinvestmentCurrency=USD, got %s", tx.ReinvestmentCurrency)
	}
	if tx.GrossAmount != -2.72 {
		t.Errorf("expected GrossAmount=-2.72, got %f", tx.GrossAmount)
	}
	if tx.GrossCurrency != "USD" {
		t.Errorf("expected GrossCurrency=USD, got %s", tx.GrossCurrency)
	}
	if tx.WithholdingTax != 0.0 {
		t.Errorf("expected WithholdingTax=0.0, got %f", tx.WithholdingTax)
	}
	if tx.WithholdingTaxCurrency != "EUR" {
		t.Errorf("expected WithholdingTaxCurrency=EUR, got %s", tx.WithholdingTaxCurrency)
	}
	if tx.ExchangeRate != 1.08 {
		t.Errorf("expected ExchangeRate=1.08, got %f", tx.ExchangeRate)
	}
	if tx.ExDate != "2026-06-15" {
		t.Errorf("expected ExDate=2026-06-15, got %s", tx.ExDate)
	}
	if tx.ValueDate != "2026-06-30" {
		t.Errorf("expected ValueDate=2026-06-30, got %s", tx.ValueDate)
	}
}
