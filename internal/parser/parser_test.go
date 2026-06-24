package parser

import (
	"strings"
	"testing"

	"github.com/welworx/flatex-pdf-cli/internal/extractor"
)

// TestParseRouting tests that the Parse function routes TRADE documents correctly.
func TestParseRouting(t *testing.T) {
	doc := &extractor.ExtractedDocument{
		Filename:     "trade.pdf",
		Text:         "Kauf von Aktien",
		DocumentType: "TRADE",
	}

	_, err := Parse(doc)
	if err == nil {
		t.Errorf("Parse should return error for stub implementation, got nil")
	}
	if !strings.Contains(err.Error(), "ParseTrade") {
		t.Errorf("expected ParseTrade error, got: %v", err)
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
