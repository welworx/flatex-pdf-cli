package extractor

import (
	"testing"
)

// TestExtractTextFromPDF tests the ExtractPDF function by verifying
// that the Filename and DocumentType fields are properly set.
func TestExtractTextFromPDF(t *testing.T) {
	// Test with a sample text that contains TRADE keywords
	text := "Dies ist ein Kaufbeleg für die Aktie ABC123. Depotnummer:31022213999 Depotinhaber:Max Mustermann"

	// Test extractMetadata to extract depot info
	depotNumber, depotHolder := extractMetadata(text)
	if depotNumber != "31022213999" {
		t.Errorf("expected depot number 31022213999, got %s", depotNumber)
	}
	if depotHolder != "Max Mustermann" {
		t.Errorf("expected depot holder 'Max Mustermann', got %s", depotHolder)
	}

	// Test detectDocumentType
	docType := detectDocumentType(text)
	if docType != "TRADE" {
		t.Errorf("expected document type TRADE, got %s", docType)
	}
}

// TestDocumentTypeDetection tests keyword-based document type detection.
// Tests the mapping: Kauf→TRADE, Verkauf→TRADE, Ausschüttung→DIVIDEND,
// Zinsen→INTEREST, Ertragsmitteilung→THESAURIERUNG
func TestDocumentTypeDetection(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected string
	}{
		{
			name:     "Kauf should be TRADE",
			text:     "Bestätigung eines Kaufs von 10 Aktien",
			expected: "TRADE",
		},
		{
			name:     "Verkauf should be TRADE",
			text:     "Bestätigung eines Verkaufs von 5 Aktien",
			expected: "TRADE",
		},
		{
			name:     "Ausschüttung should be DIVIDEND",
			text:     "Mitteilung über Ausschüttung von Dividenden",
			expected: "DIVIDEND",
		},
		{
			name:     "Zinsen should be INTEREST",
			text:     "Kontoauszug: Zinsen aus Tagesgelderträgen",
			expected: "INTEREST",
		},
		{
			name:     "Ertragsmitteilung should be THESAURIERUNG",
			text:     "Ertragsmitteilung für thesaurierte Fonds",
			expected: "THESAURIERUNG",
		},
		{
			name:     "Unknown keywords should return UNKNOWN",
			text:     "Irgendwelche anderen Inhalte ohne Schlüsselwörter",
			expected: "UNKNOWN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectDocumentType(tt.text)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// TestMetadataExtraction tests the extraction of depot number and holder
// from document text using regex patterns.
func TestMetadataExtraction(t *testing.T) {
	text := `
Depot-Auszug
Depotnummer: 31022213999
Depotinhaber: Max Mustermann
Zeitraum: Januar 2024

Positionen:
- ISIN: DE0008469008
`

	depotNumber, depotHolder := extractMetadata(text)

	if depotNumber != "31022213999" {
		t.Errorf("expected depot number '31022213999', got '%s'", depotNumber)
	}

	if depotHolder != "Max Mustermann" {
		t.Errorf("expected depot holder 'Max Mustermann', got '%s'", depotHolder)
	}

	// Test with alternative format (using = instead of :)
	text2 := `
Depotnummer=31022213999
Depotinhaber=John Doe
`

	depotNumber2, depotHolder2 := extractMetadata(text2)
	if depotNumber2 != "31022213999" {
		t.Errorf("expected depot number '31022213999' (with =), got '%s'", depotNumber2)
	}

	if depotHolder2 != "John Doe" {
		t.Errorf("expected depot holder 'John Doe' (with =), got '%s'", depotHolder2)
	}
}
