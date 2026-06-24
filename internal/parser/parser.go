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
	// Stub implementation
	return nil, fmt.Errorf("ParseTrade not implemented yet")
}

// ParseDividend parses a DIVIDEND document.
func ParseDividend(doc *extractor.ExtractedDocument) (*schema.Transaction, error) {
	// Stub implementation
	return nil, fmt.Errorf("ParseDividend not implemented yet")
}

// ParseInterest parses an INTEREST document.
func ParseInterest(doc *extractor.ExtractedDocument) (*schema.Transaction, error) {
	// Stub implementation
	return nil, fmt.Errorf("ParseInterest not implemented yet")
}

// ParseThesaurierung parses a THESAURIERUNG document.
func ParseThesaurierung(doc *extractor.ExtractedDocument) (*schema.Transaction, error) {
	// Stub implementation
	return nil, fmt.Errorf("ParseThesaurierung not implemented yet")
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
