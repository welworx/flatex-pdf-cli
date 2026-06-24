package parser

import (
	"strings"
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
	doc := &extractor.ExtractedDocument{
		Filename:     "thesaurierung.pdf",
		Text:         "Ertragsmitteilung",
		DocumentType: "THESAURIERUNG",
	}

	_, err := Parse(doc)
	if err == nil {
		t.Errorf("Parse should return error for stub implementation, got nil")
	}
	if !strings.Contains(err.Error(), "ParseThesaurierung") {
		t.Errorf("expected ParseThesaurierung error, got: %v", err)
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
