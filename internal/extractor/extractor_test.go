package extractor

import (
	"strings"
	"testing"
)

// TestLanguageGate verifies German flatex text is accepted and English text is
// rejected, so English PDFs fail fast instead of being silently mis-parsed.
func TestLanguageGate(t *testing.T) {
	german := "ﬂatexDEGIRO Bank AG\nAuftragsdatum 15.09.2025\nValuta 17.09.2025\nWertpapierabrechnung Kauf"
	if !isGermanFlatex(german) {
		t.Errorf("expected German flatex text to be recognized as German")
	}

	english := "flatexDEGIRO Bank AG\nSecurities Settlement - Purchase\nOrder date 2025-09-15\nValue date 2025-09-17\nQuantity 10 Total amount 50.00 EUR"
	if isGermanFlatex(english) {
		t.Errorf("expected English text to be rejected (English is not supported)")
	}
}

// TestExtractAccountNumber verifies the settlement account (Konto Nr.) is
// extracted. Real text extraction concatenates the next page's header directly
// onto the account number with no line break, so the match must be bounded.
func TestExtractAccountNumber(t *testing.T) {
	text := "Die Verrechnung der Endbeträge erfolgt über Ihr Konto Nr.: 55000000999035120227000"
	if got := extractAccountNumber(text); got != "55000000999" {
		t.Errorf("extractAccountNumber = %q, want 55000000999", got)
	}
}

// TestDocumentTypeDetection tests keyword-based document type detection,
// including the three headings that contain "Kauf" but must not be read as a
// plain trade.
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
			name:     "Ertragsmitteilung should be ACCUMULATING",
			text:     "Ertragsmitteilung für thesaurierte Fonds",
			expected: "ACCUMULATING",
		},
		{
			name:     "Sammelauftragsbestätigung should be ORDER (despite Kauf)",
			text:     "Sammelauftragsbestätigung\nKauf vom 28.01.2026",
			expected: "ORDER",
		},
		{
			name:     "Sammelabrechnung Kryptowerte should be CRYPTO (despite Kauf)",
			text:     "Sammelabrechnung (Kauf/-verkauf Kryptowerte)",
			expected: "CRYPTO",
		},
		{
			name:     "Sammelabrechnung aus should be SAVINGSPLAN (not CRYPTO)",
			text:     "Sammelabrechnung aus\nAuftrags-Nr:0005500055\nKauf 15.01.2025 17.01.2025 1,478695 134,2400 EUR 200,00 EUR",
			expected: "SAVINGSPLAN",
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
Depotnummer: 55000000999
Depotinhaber: Max Mustermann
Zeitraum: Januar 2024

Positionen:
- ISIN: DE0008469008
`

	depotNumber, depotHolder := extractMetadata(text)

	if depotNumber != "55000000999" {
		t.Errorf("expected depot number '55000000999', got '%s'", depotNumber)
	}

	if depotHolder != "Max Mustermann" {
		t.Errorf("expected depot holder 'Max Mustermann', got '%s'", depotHolder)
	}

	// Test with alternative format (using = instead of :)
	text2 := `
Depotnummer=55000000999
Depotinhaber=John Doe
`

	depotNumber2, depotHolder2 := extractMetadata(text2)
	if depotNumber2 != "55000000999" {
		t.Errorf("expected depot number '55000000999' (with =), got '%s'", depotNumber2)
	}

	if depotHolder2 != "John Doe" {
		t.Errorf("expected depot holder 'John Doe' (with =), got '%s'", depotHolder2)
	}
}

// TestMetadataExtractionSalutationFallback verifies that when the
// "Depotinhaber" label is absent, the depot holder is still recovered from
// the letter's salutation line.
func TestMetadataExtractionSalutationFallback(t *testing.T) {
	text := "Depotnummer: 55000000999\nSehr geehrte Frau Musterfrau,\nvielen Dank für Ihren Auftrag."

	_, depotHolder := extractMetadata(text)
	if depotHolder != "Musterfrau" {
		t.Errorf("expected depot holder %q from salutation fallback, got %q", "Musterfrau", depotHolder)
	}
}

// TestTextExtractionFromRealPDF runs the gxpdf text layer against a committed
// fixture (synthetic and PII-free, generated from a real document via the
// redacting-flatex-pdfs skill). The keywords are the ones the rest of the
// pipeline depends on: the flatex issuer marker gates the language check,
// "kauf" drives document-type detection and "depot" carries the metadata.
func TestTextExtractionFromRealPDF(t *testing.T) {
	text, err := extractTextFromPDF("../../testdata/trade_sample_1.pdf")
	if err != nil {
		t.Fatalf("extractTextFromPDF failed: %v", err)
	}
	if text == "" {
		t.Fatal("extracted text is empty")
	}

	lowerText := strings.ToLower(text)
	for _, kw := range []string{"flatex", "kauf", "depot"} {
		if !strings.Contains(lowerText, kw) {
			t.Errorf("extracted text is missing %q", kw)
		}
	}
}

// TestDetectSavingsPlanFromFixture verifies that the synthetic savings-plan
// fixture is detected as SAVINGSPLAN (not TRADE).
func TestDetectSavingsPlanFromFixture(t *testing.T) {
	doc, err := ExtractPDF("../../testdata/sparplan_sample_1.pdf")
	if err != nil {
		t.Fatalf("ExtractPDF failed: %v", err)
	}
	if doc.DocumentType != "SAVINGSPLAN" {
		t.Errorf("DocumentType = %q, want SAVINGSPLAN", doc.DocumentType)
	}
}
