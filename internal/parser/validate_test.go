package parser

import (
	"strings"
	"testing"

	"github.com/welworx/flatex-pdf-cli/internal/extractor"
	"github.com/welworx/flatex-pdf-cli/internal/schema"
)

// amt builds an optional-amount pointer; amt(0) is a stated 0,00, which is not
// the same as leaving the field nil.
func amt(v float64) *float64 { return &v }

func TestValidateTransaction(t *testing.T) {
	tests := []struct {
		name    string
		txn     *schema.Transaction
		wantErr bool
	}{
		{
			name:    "trade reconciles",
			txn:     &schema.Transaction{DocumentType: "TRADE", Quantity: amt(35), Price: amt(58.12), GrossCurrency: "EUR", GrossAmount: amt(2034.20)},
			wantErr: false,
		},
		{
			name:    "trade gross off by a column",
			txn:     &schema.Transaction{DocumentType: "TRADE", Quantity: amt(35), Price: amt(58.12), GrossCurrency: "EUR", GrossAmount: amt(3.00)},
			wantErr: true,
		},
		{
			name:    "trade within rounding tolerance",
			txn:     &schema.Transaction{DocumentType: "TRADE", Quantity: amt(0.685401), Price: amt(72.95), GrossCurrency: "EUR", GrossAmount: amt(50.00)},
			wantErr: false,
		},
		{
			name: "crypto is not reconstructed from its coarse quantity",
			// 0,014 St. printed to three decimals cannot reproduce the gross
			// value closely enough to check; only the settlement line applies.
			txn:     &schema.Transaction{DocumentType: "CRYPTO", Type: "BUY", Quantity: amt(0.014), Price: amt(72462.22), GrossCurrency: "EUR", GrossAmount: amt(1014.47), NetAmount: amt(-1019.54), Costs: &schema.Costs{Total: 5.07}},
			wantErr: false,
		},
		{
			name:    "crypto settlement does not add up",
			txn:     &schema.Transaction{DocumentType: "CRYPTO", Type: "BUY", Quantity: amt(0.014), Price: amt(72462.22), GrossCurrency: "EUR", GrossAmount: amt(1014.47), NetAmount: amt(-1019.54), Costs: &schema.Costs{Total: 99.00}},
			wantErr: true,
		},
		{
			name:    "trade settlement adds up",
			txn:     &schema.Transaction{DocumentType: "TRADE", Type: "BUY", Quantity: amt(35), Price: amt(58.12), GrossCurrency: "EUR", GrossAmount: amt(2034.20), NetAmount: amt(-2037.20), Costs: &schema.Costs{Total: 3.00}},
			wantErr: false,
		},
		{
			name: "sale settlement adds up",
			// verkauf_sample_1's real figures: the deductions come off the
			// proceeds rather than being added to them.
			txn:     &schema.Transaction{DocumentType: "TRADE", Type: "SELL", Quantity: amt(14), Price: amt(110), GrossCurrency: "EUR", GrossAmount: amt(1540.00), NetAmount: amt(1507.08), WithholdingTax: amt(24.51), Costs: &schema.Costs{Total: 8.41}},
			wantErr: false,
		},
		{
			name:    "sale settlement does not add up",
			txn:     &schema.Transaction{DocumentType: "TRADE", Type: "SELL", Quantity: amt(14), Price: amt(110), GrossCurrency: "EUR", GrossAmount: amt(1540.00), NetAmount: amt(1600.00), WithholdingTax: amt(24.51), Costs: &schema.Costs{Total: 8.41}},
			wantErr: true,
		},
		{
			name: "sale deductions must not be added like a purchase",
			// 1540.00 + 8.41 + 24.51 = 1572.92 is what a buy would settle at.
			// This case fails if the sale ever reverts to the purchase sign.
			txn:     &schema.Transaction{DocumentType: "TRADE", Type: "SELL", Quantity: amt(14), Price: amt(110), GrossCurrency: "EUR", GrossAmount: amt(1540.00), NetAmount: amt(1572.92), WithholdingTax: amt(24.51), Costs: &schema.Costs{Total: 8.41}},
			wantErr: true,
		},
		{
			name: "missing settlement total is skipped",
			// Absent means the document printed no Endbetrag: NetAmount nil.
			txn:     &schema.Transaction{DocumentType: "TRADE", Type: "BUY", Quantity: amt(35), Price: amt(58.12), GrossCurrency: "EUR", GrossAmount: amt(2034.20)},
			wantErr: false,
		},
		{
			name: "a stated 0,00 settlement is checked, not skipped",
			// The document printed "Endbetrag: 0,00 EUR", which cannot be right
			// for a 2034.20 purchase. A plain float64 made this indistinguishable
			// from "no Endbetrag" and the check was silently skipped.
			txn:     &schema.Transaction{DocumentType: "TRADE", Type: "BUY", Quantity: amt(35), Price: amt(58.12), GrossCurrency: "EUR", GrossAmount: amt(2034.20), NetAmount: amt(0), Costs: &schema.Costs{Total: 3.00}},
			wantErr: true,
		},
		{
			name: "trade in foreign currency is not cross-checked",
			// Kurs is extracted in EUR while Kurswert here is USD; the two are
			// related by Devisenkurs, so the identity must not be applied.
			txn:     &schema.Transaction{DocumentType: "TRADE", Quantity: amt(35), Price: amt(58.12), GrossCurrency: "USD", GrossAmount: amt(2200.00)},
			wantErr: false,
		},
		{
			name: "missing operand is left to the extraction errors",
			// Absent means the document printed no Kurs at all: Price nil.
			txn:     &schema.Transaction{DocumentType: "TRADE", Quantity: amt(35), GrossCurrency: "EUR", GrossAmount: amt(2034.20)},
			wantErr: false,
		},
		{
			name: "a stated price of 0,00 is checked, not skipped",
			// 35 x 0,00 cannot produce a gross value of 2034.20.
			txn:     &schema.Transaction{DocumentType: "TRADE", Quantity: amt(35), Price: amt(0), GrossCurrency: "EUR", GrossAmount: amt(2034.20)},
			wantErr: true,
		},
		{
			name: "KESt far below the rate is normal, not a fault",
			// verkauf_sample_1's real figures: 24.51 withheld on a gain of
			// 403.97 is 6.07%, because the gain is netted against the
			// Verluststeuertopf. A check expecting ~27.5% would reject the only
			// real sale in the corpus.
			txn:     &schema.Transaction{DocumentType: "TRADE", GainLoss: amt(403.97), WithholdingTax: amt(24.51)},
			wantErr: false,
		},
		{
			name:    "Altbestand withholds nothing on a real gain",
			txn:     &schema.Transaction{DocumentType: "TRADE", GainLoss: amt(500.00), WithholdingTax: amt(0)},
			wantErr: false,
		},
		{
			name:    "KESt at exactly the rate is allowed",
			txn:     &schema.Transaction{DocumentType: "TRADE", GainLoss: amt(400.00), WithholdingTax: amt(110.00)},
			wantErr: false,
		},
		{
			name: "KESt above the rate cannot come from this gain",
			// 50.00 withheld on a 100.00 gain is 50%, which no Austrian rate
			// produces: one of the two figures is in the wrong column.
			txn:     &schema.Transaction{DocumentType: "TRADE", GainLoss: amt(100.00), WithholdingTax: amt(50.00)},
			wantErr: true,
		},
		{
			name:    "a gain must not refund tax",
			txn:     &schema.Transaction{DocumentType: "TRADE", GainLoss: amt(100.00), WithholdingTax: amt(-10.00)},
			wantErr: true,
		},
		{
			name: "a loss may refund tax withheld earlier in the year",
			// Verluststeuertopf: the refund is bounded by prior withholdings,
			// which this document does not state, so nothing is checked.
			txn:     &schema.Transaction{DocumentType: "TRADE", GainLoss: amt(-200.00), WithholdingTax: amt(-55.00)},
			wantErr: false,
		},
		{
			name:    "a loss that withheld nothing is fine",
			txn:     &schema.Transaction{DocumentType: "TRADE", GainLoss: amt(-200.00), WithholdingTax: amt(0)},
			wantErr: false,
		},
		{
			name:    "dividend reconciles",
			txn:     &schema.Transaction{DocumentType: "DIVIDEND", Quantity: amt(74.45), DistributionPerShare: amt(0.4227), DistributionCurrency: "USD", GrossAmount: amt(31.47), GrossCurrency: "USD"},
			wantErr: false,
		},
		{
			name:    "dividend gross wrong",
			txn:     &schema.Transaction{DocumentType: "DIVIDEND", Quantity: amt(74.45), DistributionPerShare: amt(0.4227), DistributionCurrency: "USD", GrossAmount: amt(4.41), GrossCurrency: "USD"},
			wantErr: true,
		},
		{
			name: "savings plan is not cross-checked",
			// The plan's Kurswert is the order amount, not the share value.
			txn:     &schema.Transaction{DocumentType: "SAVINGSPLAN", Quantity: amt(1.4787), Price: amt(134.24), GrossCurrency: "EUR", GrossAmount: amt(200.00)},
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
		{"trade_sample_1.pdf", func(x *schema.Transaction) { *x.GrossAmount *= 10 }},
		{"trade_sample_2.pdf", func(x *schema.Transaction) { *x.GrossAmount *= 10 }},
		{"krypto_sample_1.pdf", func(x *schema.Transaction) { *x.GrossAmount *= 10 }},
		{"dividend_sample_1.pdf", func(x *schema.Transaction) { *x.GrossAmount *= 10 }},
		{"dividend_sample_2.pdf", func(x *schema.Transaction) { *x.GrossAmount *= 10 }},
		// The settlement identity is what covers crypto, so corrupt the
		// settlement side directly to prove that path runs on a real document
		// rather than riding on the gross-value check above.
		{"krypto_sample_1.pdf", func(x *schema.Transaction) { *x.NetAmount += 100 }},
		{"trade_sample_2.pdf", func(x *schema.Transaction) { *x.NetAmount += 100 }},
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
		DocumentType: "TRADE", Quantity: amt(35), Price: amt(58.12), GrossCurrency: "EUR", GrossAmount: amt(3.00),
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
