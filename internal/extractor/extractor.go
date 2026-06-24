package extractor

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"
)

// ExtractedDocument contains extracted text and metadata from a PDF file.
type ExtractedDocument struct {
	Filename     string
	Text         string
	DepotNumber  string
	DepotHolder  string
	DocumentType string
}

// ExtractPDF extracts text and metadata from a PDF file.
func ExtractPDF(filePath string) (*ExtractedDocument, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open PDF: %w", err)
	}
	defer file.Close()

	text, err := extractTextFromPDF(file)
	if err != nil {
		return nil, fmt.Errorf("failed to extract text: %w", err)
	}

	depotNumber, depotHolder := extractMetadata(text)
	documentType := detectDocumentType(text)

	return &ExtractedDocument{
		Filename:     filepath.Base(filePath),
		Text:         text,
		DepotNumber:  depotNumber,
		DepotHolder:  depotHolder,
		DocumentType: documentType,
	}, nil
}

// extractTextFromPDF extracts text content from a PDF file.
// ponytail: pdfcpu v0.13.0 doesn't expose simple text extraction.
// This validates the PDF is readable; full text extraction via content
// stream parsing or specialized library (unidoc, etc) is a future step.
func extractTextFromPDF(file *os.File) (string, error) {
	_, err := api.ReadContext(file, api.LoadConfiguration())
	if err != nil {
		return "", fmt.Errorf("failed to read PDF context: %w", err)
	}

	// ponytail: return empty string for now.
	// The PDF is validated as readable; text extraction deferred.
	return "", nil
}

// extractMetadata extracts depot number and depot holder from PDF text.
func extractMetadata(text string) (depotNumber, depotHolder string) {
	// Extract depot number using regex
	depotRegex := regexp.MustCompile(`Depotnummer\s*[:=]\s*(\d+)`)
	if matches := depotRegex.FindStringSubmatch(text); len(matches) > 1 {
		depotNumber = matches[1]
	}

	// Extract depot holder using regex
	holderRegex := regexp.MustCompile(`Depotinhaber\s*[:=]\s*([^\n]+)`)
	if matches := holderRegex.FindStringSubmatch(text); len(matches) > 1 {
		depotHolder = strings.TrimSpace(matches[1])
	}

	return depotNumber, depotHolder
}

// detectDocumentType detects the document type based on keywords in the text.
func detectDocumentType(text string) string {
	lowerText := strings.ToLower(text)

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

	// Check for THESAURIERUNG keywords
	if strings.Contains(lowerText, "ertragsmitteilung") {
		return "THESAURIERUNG"
	}

	return "UNKNOWN"
}
