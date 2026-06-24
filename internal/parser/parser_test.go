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
	doc := &extractor.ExtractedDocument{
		Filename:     "dividend.pdf",
		Text:         "Ausschüttung",
		DocumentType: "DIVIDEND",
	}

	_, err := Parse(doc)
	if err == nil {
		t.Errorf("Parse should return error for stub implementation, got nil")
	}
	if !strings.Contains(err.Error(), "ParseDividend") {
		t.Errorf("expected ParseDividend error, got: %v", err)
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
