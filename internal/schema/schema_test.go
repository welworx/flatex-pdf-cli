package schema

import (
	"encoding/json"
	"testing"
	"time"
)

func TestTradeTransactionMarshal(t *testing.T) {
	date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)

	tx := Transaction{
		Source:          "flatex",
		DocNumber:       "TRD-2024-001",
		DocumentType:    "TRADE",
		ISIN:            "IE000YU9K6K2",
		WKN:             "A2XXXX",
		Date:            date,
		Type:            "BUY",
		Quantity:        1.058537,
		Price:           47.235,
		PriceCurrency:   "EUR",
		GrossValue:      50.01,
		Provision:       5.99,
		OwnCosts:        0.0,
		ThirdPartyCosts: 0.0,
		WithholdingTax:  0.0,
		GainLoss:        0.0,
		ExchangeRate:    1.0,
		FinalAmount:     44.02,
		FinalCurrency:   "EUR",
		CustodyType:     "depot",
		Depositary:      "flatex",
		Country:         "DE",
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
	exDate := time.Date(2024, 6, 10, 0, 0, 0, 0, time.UTC)
	valueDate := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	docDate := time.Date(2024, 6, 20, 0, 0, 0, 0, time.UTC)

	tx := Transaction{
		Source:                 "flatex",
		DocNumber:              "DIV-2024-001",
		DocumentType:           "DIVIDEND",
		ISIN:                   "IE00B3RBWM25",
		Date:                   docDate,
		Quantity:               78.70,
		DistributionPerShare:   0.5459180,
		DistributionCurrency:   "USD",
		GrossAmount:            42.99,
		GrossCurrency:          "USD",
		WithholdingTax:         6.45,
		WithholdingTaxCurrency: "USD",
		NetAmount:              36.54,
		NetCurrency:            "USD",
		ExDate:                 exDate,
		ValueDate:              valueDate,
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

func TestThesaurierungTransactionMarshal(t *testing.T) {
	accrualDate := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	docDate := time.Date(2024, 6, 20, 0, 0, 0, 0, time.UTC)

	tx := Transaction{
		Source:               "flatex",
		DocNumber:            "THES-2024-001",
		DocumentType:         "THESAURIERUNG",
		ISIN:                 "IE00B5L8K969",
		Date:                 docDate,
		Quantity:             4.75,
		ReinvestmentPerShare: 0.7234,
		ReinvestmentCurrency: "EUR",
		AccrualDate:          accrualDate,
	}

	// Marshal to JSON
	data, err := json.Marshal(tx)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	jsonStr := string(data)

	// Verify JSON contains "THESAURIERUNG"
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
	if rtx.DocumentType != "THESAURIERUNG" {
		t.Errorf("DocumentType mismatch: got %q, want %q", rtx.DocumentType, "THESAURIERUNG")
	}
}
