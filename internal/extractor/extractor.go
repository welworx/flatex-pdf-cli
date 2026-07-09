package extractor

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/coregx/gxpdf"
)

// ExtractedDocument contains extracted text and metadata from a PDF file.
type ExtractedDocument struct {
	Filename      string
	Text          string
	DepotNumber   string
	DepotHolder   string
	AccountNumber string
	DocumentType  string
}

// ExtractPDF extracts text and metadata from a PDF file.
func ExtractPDF(filePath string) (*ExtractedDocument, error) {
	text, err := extractTextFromPDF(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to extract text: %w", err)
	}

	if !isGermanFlatex(text) {
		return nil, fmt.Errorf("unsupported document language: only German flatex PDFs are implemented (English/other languages: not supported)")
	}

	depotNumber, depotHolder := extractMetadata(text)
	documentType := detectDocumentType(text)

	return &ExtractedDocument{
		Filename:      filepath.Base(filePath),
		Text:          text,
		DepotNumber:   depotNumber,
		DepotHolder:   depotHolder,
		AccountNumber: extractAccountNumber(text),
		DocumentType:  documentType,
	}, nil
}

// extractAccountNumber extracts the settlement account (Konto Nr.) from the text.
// ponytail: bounded to 11 digits because PDF text extraction runs the next
// page's header straight onto the number with no separator; widen if flatex
// ever issues account numbers of a different length.
func extractAccountNumber(text string) string {
	regex := regexp.MustCompile(`Konto Nr\.\s*[:=]?\s*(\d{11})`)
	if matches := regex.FindStringSubmatch(text); len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// extractTextFromPDF extracts text content from a PDF file.
func extractTextFromPDF(filePath string) (string, error) {
	doc, err := gxpdf.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open PDF: %w", err)
	}
	defer doc.Close()

	var text strings.Builder
	pageCount := doc.PageCount()
	for i := 1; i <= pageCount; i++ {
		pageText, err := doc.ExtractTextFromPage(i)
		if err != nil {
			return "", fmt.Errorf("failed to extract text from page %d: %w", i, err)
		}
		text.WriteString(pageText)
	}

	return strings.ToValidUTF8(text.String(), ""), nil
}

// extractMetadata extracts depot number and depot holder from PDF text.
func extractMetadata(text string) (depotNumber, depotHolder string) {
	// Extract depot number using regex
	depotRegex := regexp.MustCompile(`Depotnummer\s*[:=]\s*(\d+)`)
	if matches := depotRegex.FindStringSubmatch(text); len(matches) > 1 {
		depotNumber = matches[1]
	}

	// Extract depot holder: try Depotinhaber label first, fall back to salutation
	holderRegex := regexp.MustCompile(`Depotinhaber\s*[:=]\s*([^\n]+)`)
	if matches := holderRegex.FindStringSubmatch(text); len(matches) > 1 {
		depotHolder = strings.TrimSpace(matches[1])
	} else {
		salutationRegex := regexp.MustCompile(`Sehr geehrte[rn]?\s+(?:Herr|Frau)\s+(.+?),`)
		if matches := salutationRegex.FindStringSubmatch(text); len(matches) > 1 {
			depotHolder = strings.TrimSpace(matches[1])
		}
	}

	return depotNumber, depotHolder
}

// isGermanFlatex reports whether the extracted text is a German flatex statement.
// Only German documents are implemented; English/other-language flatex PDFs are
// rejected here rather than silently mis-parsed (their field labels and keywords
// differ). Fail-closed: proceed only on a confident German match.
func isGermanFlatex(text string) bool {
	lower := strings.ToLower(text)
	// German-specific labels/keywords present in flatex German statements; none of
	// these occur in an English-language statement.
	anchors := []string{
		"wertpapierabrechnung", "ertragsmitteilung", "depotnummer", "depotinhaber",
		"auftragsdatum", "auftragsnummer", "ausführungszeit", "handelstag",
		"valuta", "devisenkurs", "ausschüttung", "zinsen", "kauf", "verkauf",
	}
	for _, a := range anchors {
		if strings.Contains(lower, a) {
			return true
		}
	}
	return false
}

// detectDocumentType detects the document type based on keywords in the text.
func detectDocumentType(text string) string {
	lowerText := strings.ToLower(text)

	// Crypto settlement and order confirmations also contain "Kauf", so they
	// must be checked before the generic TRADE keyword.
	if strings.Contains(lowerText, "sammelabrechnung") && strings.Contains(lowerText, "kryptowerte") {
		return "CRYPTO"
	}
	// SAVINGSPLAN must precede TRADE (rows contain "Kauf"/"Verkauf")
	if strings.Contains(lowerText, "sammelabrechnung") && !strings.Contains(lowerText, "kryptowerte") {
		return "SAVINGSPLAN"
	}
	if strings.Contains(lowerText, "sammelauftragsbestätigung") {
		return "ORDER"
	}

	// Check for TRADE keywords
	if strings.Contains(lowerText, "kauf") || strings.Contains(lowerText, "verkauf") {
		return "TRADE"
	}

	// Check for DIVIDEND keywords
	if strings.Contains(lowerText, "ausschüttung") {
		return "DIVIDEND"
	}

	// Check for INTEREST keywords
	if strings.Contains(lowerText, "zinsen") {
		return "INTEREST"
	}

	// Check for ACCUMULATING keywords
	if strings.Contains(lowerText, "ertragsmitteilung") {
		return "ACCUMULATING"
	}

	return "UNKNOWN"
}
