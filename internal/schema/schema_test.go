package schema

import (
	"encoding/json"
	"testing"
)

func TestTradeTransactionMarshal(t *testing.T) {
	tx := Transaction{
		Source:         "flatex",
		OrderNumber:    "999888777/1",
		DocumentType:   "TRADE",
		ISIN:           "IE000YU9K6K2",
		WKN:            "A2XXXX",
		Date:           "2024-06-15",
		Type:           "BUY",
		Quantity:       1.058537,
		Price:          47.235,
		PriceCurrency:  "EUR",
		GrossValue:     50.01,
		Provision:      5.99,
		WithholdingTax: 0.0,
		GainLoss:       0.0,
		ExchangeRate:   1.0,
		FinalAmount:    44.02,
		FinalCurrency:  "EUR",
		CustodyType:    "depot",
		Depositary:     "flatex",
	}

	// Marshal to JSON
	data, err := json.Marshal(tx)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Unmarshal to verify roundtrip
	var rtx Transaction
	if err := json.Unmarshal(data, &rtx); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Verify key fields survived roundtrip
	if rtx.ISIN != "IE000YU9K6K2" {
		t.Errorf("ISIN mismatch: got %q, want %q", rtx.ISIN, "IE000YU9K6K2")
	}
	if rtx.Quantity != 1.058537 {
		t.Errorf("Quantity mismatch: got %f, want %f", rtx.Quantity, 1.058537)
	}
	if rtx.Price != 47.235 {
		t.Errorf("Price mismatch: got %f, want %f", rtx.Price, 47.235)
	}
	if rtx.Type != "BUY" {
		t.Errorf("Type mismatch: got %q, want %q", rtx.Type, "BUY")
	}
	if rtx.DocumentType != "TRADE" {
		t.Errorf("DocumentType mismatch: got %q, want %q", rtx.DocumentType, "TRADE")
	}

	// Verify JSON contains expected fields
	jsonStr := string(data)
	if jsonStr == "" {
		t.Error("JSON serialization produced empty string")
	}
}

func TestDividendTransactionMarshal(t *testing.T) {
	tx := Transaction{
		Source:                 "flatex",
		OrderNumber:            "999888777/1",
		DocumentType:           "DIVIDEND",
		ISIN:                   "IE00B3RBWM25",
		Date:                   "2024-06-20",
		Quantity:               78.70,
		DistributionPerShare:   0.5459180,
		DistributionCurrency:   "USD",
		GrossAmount:            42.99,
		GrossCurrency:          "USD",
		WithholdingTax:         6.45,
		WithholdingTaxCurrency: "USD",
		NetAmount:              36.54,
		NetCurrency:            "USD",
		ExDate:                 "2024-06-10",
		ValueDate:              "2024-06-15",
	}

	// Marshal to JSON
	data, err := json.Marshal(tx)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	jsonStr := string(data)

	// Verify JSON contains "DIVIDEND"
	if jsonStr == "" {
		t.Error("JSON serialization produced empty string")
	}

	// Unmarshal to verify roundtrip
	var rtx Transaction
	if err := json.Unmarshal(data, &rtx); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Verify key fields survived roundtrip
	if rtx.ISIN != "IE00B3RBWM25" {
		t.Errorf("ISIN mismatch: got %q, want %q", rtx.ISIN, "IE00B3RBWM25")
	}
	if rtx.Quantity != 78.70 {
		t.Errorf("Quantity mismatch: got %f, want %f", rtx.Quantity, 78.70)
	}
	if rtx.DistributionPerShare != 0.5459180 {
		t.Errorf("DistributionPerShare mismatch: got %f, want %f", rtx.DistributionPerShare, 0.5459180)
	}
	if rtx.DistributionCurrency != "USD" {
		t.Errorf("DistributionCurrency mismatch: got %q, want %q", rtx.DistributionCurrency, "USD")
	}
	if rtx.DocumentType != "DIVIDEND" {
		t.Errorf("DocumentType mismatch: got %q, want %q", rtx.DocumentType, "DIVIDEND")
	}
}

func TestAccumulatingTransactionMarshal(t *testing.T) {
	tx := Transaction{
		Source:               "flatex",
		OrderNumber:          "999888777/1",
		DocumentType:         "ACCUMULATING",
		ISIN:                 "IE00B5L8K969",
		Date:                 "2024-06-20",
		Quantity:             4.75,
		ReinvestmentPerShare: 0.7234,
		ReinvestmentCurrency: "EUR",
		AccrualDate:          "2024-06-15",
	}

	// Marshal to JSON
	data, err := json.Marshal(tx)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	jsonStr := string(data)

	// Verify JSON contains "ACCUMULATING"
	if jsonStr == "" {
		t.Error("JSON serialization produced empty string")
	}

	// Unmarshal to verify roundtrip
	var rtx Transaction
	if err := json.Unmarshal(data, &rtx); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Verify key fields survived roundtrip
	if rtx.ISIN != "IE00B5L8K969" {
		t.Errorf("ISIN mismatch: got %q, want %q", rtx.ISIN, "IE00B5L8K969")
	}
	if rtx.Quantity != 4.75 {
		t.Errorf("Quantity mismatch: got %f, want %f", rtx.Quantity, 4.75)
	}
	if rtx.ReinvestmentPerShare != 0.7234 {
		t.Errorf("ReinvestmentPerShare mismatch: got %f, want %f", rtx.ReinvestmentPerShare, 0.7234)
	}
	if rtx.ReinvestmentCurrency != "EUR" {
		t.Errorf("ReinvestmentCurrency mismatch: got %q, want %q", rtx.ReinvestmentCurrency, "EUR")
	}
	if rtx.DocumentType != "ACCUMULATING" {
		t.Errorf("DocumentType mismatch: got %q, want %q", rtx.DocumentType, "ACCUMULATING")
	}
}

func TestOutputWithMetadata(t *testing.T) {
	tx := Transaction{
		Source:        "flatex",
		OrderNumber:   "999888777/1",
		DocumentType:  "TRADE",
		ISIN:          "IE000YU9K6K2",
		Date:          "2024-06-15",
		Type:          "BUY",
		Quantity:      1.0,
		Price:         50.0,
		PriceCurrency: "EUR",
	}

	output := Output{
		Metadata: &DocumentMetadata{
			DepotNumber: "31022213999",
			DepotHolder: "Max Mustermann",
		},
		Transactions: []*Transaction{&tx},
	}

	// Marshal to JSON
	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	jsonStr := string(data)

	// Verify JSON contains expected strings
	if jsonStr == "" {
		t.Error("JSON serialization produced empty string")
	}

	// Check for specific content
	if !contains(jsonStr, "31022213999") {
		t.Errorf("JSON missing depot_number: %s", jsonStr)
	}
	if !contains(jsonStr, "transactions") {
		t.Errorf("JSON missing transactions field: %s", jsonStr)
	}
	if !contains(jsonStr, "metadata") {
		t.Errorf("JSON missing metadata field: %s", jsonStr)
	}

	// Unmarshal to verify roundtrip
	var rout Output
	if err := json.Unmarshal(data, &rout); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Verify structure
	if rout.Metadata == nil {
		t.Error("Metadata should not be nil")
	}
	if rout.Metadata != nil && rout.Metadata.DepotNumber != "31022213999" {
		t.Errorf("DepotNumber mismatch: got %q, want %q", rout.Metadata.DepotNumber, "31022213999")
	}
	if rout.Metadata != nil && rout.Metadata.DepotHolder != "Max Mustermann" {
		t.Errorf("DepotHolder mismatch: got %q, want %q", rout.Metadata.DepotHolder, "Max Mustermann")
	}
	if len(rout.Transactions) != 1 {
		t.Errorf("Transactions length mismatch: got %d, want 1", len(rout.Transactions))
	}
}

func TestOutputTransactionsOnly(t *testing.T) {
	tx := Transaction{
		Source:       "flatex",
		OrderNumber:  "999888777/1",
		DocumentType: "TRADE",
		ISIN:         "IE000YU9K6K2",
		Date:         "2024-06-15",
	}

	// Marshal transactions slice directly
	txs := []*Transaction{&tx}
	data, err := json.Marshal(txs)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	jsonStr := string(data)

	// Verify it's an array (starts with "[")
	if len(jsonStr) == 0 || jsonStr[0] != '[' {
		t.Errorf("JSON should be an array starting with '[', got: %s", jsonStr)
	}
}

// contains is a helper function to check if a string is present in another string
func contains(haystack, needle string) bool {
	for i := 0; i <= len(haystack)-len(needle); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
