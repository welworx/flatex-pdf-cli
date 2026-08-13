package parser

import (
	"strings"
	"testing"

	"github.com/welworx/flatex-pdf-cli/internal/extractor"
	"github.com/welworx/flatex-pdf-cli/internal/schema"
)

func TestValidateTransaction(t *testing.T) {
	tests := []struct {
		name    string
		txn     *schema.Transaction
		wantErr bool
	}{
		{
			name:    "trade reconciles",
			txn:     &schema.Transaction{DocumentType: "TRADE", Quantity: 35, Price: 58.12, PriceCurrency: "EUR", GrossValue: 2034.20},
			wantErr: false,
		},
		{
			name:    "trade gross off by a column",
			txn:     &schema.Transaction{DocumentType: "TRADE", Quantity: 35, Price: 58.12, PriceCurrency: "EUR", GrossValue: 3.00},
			wantErr: true,
		},
		{
			name:    "trade within rounding tolerance",
			txn:     &schema.Transaction{DocumentType: "TRADE", Quantity: 0.685401, Price: 72.95, PriceCurrency: "EUR", GrossValue: 50.00},
			wantErr: false,
		},
		{
			name: "crypto is not reconstructed from its coarse quantity",
			// 0,014 St. printed to three decimals cannot reproduce the gross
			// value closely enough to check; only the settlement line applies.
			txn:     &schema.Transaction{DocumentType: "CRYPTO", Type: "BUY", Quantity: 0.014, Price: 72462.22, PriceCurrency: "EUR", GrossValue: 1014.47, FinalAmount: -1019.54, Costs: &schema.Costs{Total: 5.07}},
			wantErr: false,
		},
		{
			name:    "crypto settlement does not add up",
			txn:     &schema.Transaction{DocumentType: "CRYPTO", Type: "BUY", Quantity: 0.014, Price: 72462.22, PriceCurrency: "EUR", GrossValue: 1014.47, FinalAmount: -1019.54, Costs: &schema.Costs{Total: 99.00}},
			wantErr: true,
		},
		{
			name:    "trade settlement adds up",
			txn:     &schema.Transaction{DocumentType: "TRADE", Type: "BUY", Quantity: 35, Price: 58.12, PriceCurrency: "EUR", GrossValue: 2034.20, FinalAmount: -2037.20, Costs: &schema.Costs{Total: 3.00}},
			wantErr: false,
		},
		{
			name: "sale is not settlement-checked",
			// Deductions reverse sign on a sale and no sell fixture exists.
			txn:     &schema.Transaction{DocumentType: "TRADE", Type: "SELL", Quantity: 35, Price: 58.12, PriceCurrency: "EUR", GrossValue: 2034.20, FinalAmount: -2031.20, Costs: &schema.Costs{Total: 3.00}},
			wantErr: false,
		},
		{
			name:    "missing settlement total is skipped",
			txn:     &schema.Transaction{DocumentType: "TRADE", Type: "BUY", Quantity: 35, Price: 58.12, PriceCurrency: "EUR", GrossValue: 2034.20, FinalAmount: 0},
			wantErr: false,
		},
		{
			name: "trade in foreign currency is not cross-checked",
			// Kurs is extracted in EUR while Kurswert here is USD; the two are
			// related by Devisenkurs, so the identity must not be applied.
			txn:     &schema.Transaction{DocumentType: "TRADE", Quantity: 35, Price: 58.12, PriceCurrency: "USD", GrossValue: 2200.00},
			wantErr: false,
		},
		{
			name:    "missing operand is left to the extraction errors",
			txn:     &schema.Transaction{DocumentType: "TRADE", Quantity: 35, Price: 0, PriceCurrency: "EUR", GrossValue: 2034.20},
			wantErr: false,
		},
		{
			name:    "dividend reconciles",
			txn:     &schema.Transaction{DocumentType: "DIVIDEND", Quantity: 74.45, DistributionPerShare: 0.4227, DistributionCurrency: "USD", GrossAmount: 31.47, GrossCurrency: "USD"},
			wantErr: false,
		},
		{
			name:    "dividend gross wrong",
			txn:     &schema.Transaction{DocumentType: "DIVIDEND", Quantity: 74.45, DistributionPerShare: 0.4227, DistributionCurrency: "USD", GrossAmount: 4.41, GrossCurrency: "USD"},
			wantErr: true,
		},
		{
			name: "savings plan is not cross-checked",
			// The plan's Kurswert is the order amount, not the share value.
			txn:     &schema.Transaction{DocumentType: "SAVINGSPLAN", Quantity: 1.4787, Price: 134.24, PriceCurrency: "EUR", GrossValue: 200.00},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateTransaction(tc.txn)
			if tc.wantErr && err == nil {
				t.Fatalf("expected a validation error, got none")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected no validation error, got %v", err)
			}
		})
	}
}

// TestValidateFiresOnRealDocuments guards against the failure mode that matters
// most for a check like this: passing because it never actually runs. For each
// fixture whose type is cross-checked, it parses the real PDF and then corrupts
// the stated total the way a shifted column would. If validation does not
// reject the corrupted transaction, the guard conditions (currency, non-zero
// operands) are not satisfied by real documents and the check is decorative.
func TestValidateFiresOnRealDocuments(t *testing.T) {
	tests := []struct {
		file    string
		corrupt func(*schema.Transaction)
	}{
		{"trade_sample_1.pdf", func(x *schema.Transaction) { x.GrossValue *= 10 }},
		{"trade_sample_2.pdf", func(x *schema.Transaction) { x.GrossValue *= 10 }},
		{"krypto_sample_1.pdf", func(x *schema.Transaction) { x.GrossValue *= 10 }},
		{"dividend_sample_1.pdf", func(x *schema.Transaction) { x.GrossAmount *= 10 }},
		{"dividend_sample_2.pdf", func(x *schema.Transaction) { x.GrossAmount *= 10 }},
		// The settlement identity is what covers crypto, so corrupt the
		// settlement side directly to prove that path runs on a real document
		// rather than riding on the gross-value check above.
		{"krypto_sample_1.pdf", func(x *schema.Transaction) { x.FinalAmount += 100 }},
		{"trade_sample_2.pdf", func(x *schema.Transaction) { x.FinalAmount += 100 }},
	}

	for _, tc := range tests {
		t.Run(tc.file, func(t *testing.T) {
			doc, err := extractor.ExtractPDF("../../testdata/" + tc.file)
			if err != nil {
				t.Fatalf("extract: %v", err)
			}
			txns, err := Parse(doc)
			if err != nil {
				t.Fatalf("the unmodified fixture must parse clean: %v", err)
			}
			if len(txns) == 0 {
				t.Fatal("no transactions parsed")
			}

			tc.corrupt(txns[0])
			if err := validateTransaction(txns[0]); err == nil {
				t.Fatal("validation passed a corrupted total, so the check does not run on this document")
			}
		})
	}
}

func TestValidateErrorNamesTheProblem(t *testing.T) {
	err := validateTransaction(&schema.Transaction{
		DocumentType: "TRADE", Quantity: 35, Price: 58.12, PriceCurrency: "EUR", GrossValue: 3.00,
	})
	if err == nil {
		t.Fatal("expected an error")
	}
	// The message is what lands on stderr when a run fails, so it has to say
	// which figure disagreed and what was expected instead.
	for _, want := range []string{"gross value", "3.00", "2034.20", "layout"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %q", err.Error(), want)
		}
	}
}
