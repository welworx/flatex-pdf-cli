package export

import (
	"bytes"
	"strings"
	"testing"

	"github.com/welworx/flatex-pdf-cli/internal/schema"
)

// amt builds an optional-amount pointer; amt(0) is a stated 0,00, which is not
// the same as leaving the field nil.
func amt(v float64) *schema.Decimal { d := schema.Num(v, 2); return &d }

func TestWritePortfolioTransactionsBuyAndSell(t *testing.T) {
	txns := []*schema.Transaction{
		{DocumentType: "TRADE", ISIN: "IE000YU9K6K2", Date: "2024-06-15", Type: "BUY", Quantity: amt(1), GrossAmount: amt(50), Costs: &schema.Costs{Provision: schema.Num(5, 2), Total: schema.Num(5, 2)}},
		{DocumentType: "TRADE", ISIN: "IE000YU9K6K2", Date: "2024-06-16", Type: "SELL", Quantity: amt(1), GrossAmount: amt(60), Costs: &schema.Costs{Provision: schema.Num(5, 2), Total: schema.Num(5, 2)}},
		{DocumentType: "ORDER", ISIN: "IE000YU9K6K2", Date: "2024-06-17"}, // pending, must be skipped
	}

	var buf bytes.Buffer
	if err := WritePortfolioTransactions(&buf, txns, "en"); err != nil {
		t.Fatalf("WritePortfolioTransactions failed: %v", err)
	}

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected header + 2 rows (ORDER skipped), got %d lines: %v", len(lines), lines)
	}
	if !strings.Contains(lines[1], "Buy") || !strings.Contains(lines[1], "55") {
		t.Errorf("expected BUY row with Value=55 (gross+fee), got: %s", lines[1])
	}
	if !strings.Contains(lines[2], "Sell") || !strings.Contains(lines[2], "55") {
		t.Errorf("expected SELL row with Value=55 (gross-fee), got: %s", lines[2])
	}
}

func TestWritePortfolioTransactionsUsesFinalAmountWhenPresent(t *testing.T) {
	txns := []*schema.Transaction{
		{DocumentType: "CRYPTO", SecurityName: "BITCOIN", Date: "2024-06-15", Type: "BUY", Quantity: amt(0.01), GrossAmount: amt(500), Costs: &schema.Costs{Provision: schema.Num(10, 2), Total: schema.Num(10, 2)}, NetAmount: amt(-510)},
	}

	var buf bytes.Buffer
	if err := WritePortfolioTransactions(&buf, txns, "en"); err != nil {
		t.Fatalf("WritePortfolioTransactions failed: %v", err)
	}

	// flatex signs Endbetrag by cash direction (negative for a buy); PP takes
	// the magnitude and reads the direction from the Type column instead.
	if !strings.Contains(buf.String(), "510") || strings.Contains(buf.String(), "-510") {
		t.Errorf("expected Value to be NetAmount's magnitude (510), got: %s", buf.String())
	}
}

func TestWriteAccountTransactionsMapsTypes(t *testing.T) {
	txns := []*schema.Transaction{
		{DocumentType: "DIVIDEND", ISIN: "IE000YU9K6K2", Date: "2024-06-15", NetAmount: amt(10)},
		{DocumentType: "INTEREST", Date: "2024-06-16", NetAmount: amt(2)},
		{DocumentType: "ACCUMULATING", ISIN: "IE000YU9K6K2", Date: "2024-06-17", WithholdingTax: amt(3)},
		{DocumentType: "ACCUMULATING", ISIN: "IE000YU9K6K2", Date: "2024-06-18", WithholdingTax: amt(0)}, // no real cash movement, must be skipped
	}

	var buf bytes.Buffer
	if err := WriteAccountTransactions(&buf, txns, "en"); err != nil {
		t.Fatalf("WriteAccountTransactions failed: %v", err)
	}

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != 4 {
		t.Fatalf("expected header + 3 rows (zero-tax ACCUMULATING skipped), got %d: %v", len(lines), lines)
	}
	if !strings.Contains(lines[1], "Dividend") {
		t.Errorf("expected Dividend row, got: %s", lines[1])
	}
	if !strings.Contains(lines[2], "Interest") {
		t.Errorf("expected Interest row, got: %s", lines[2])
	}
	if !strings.Contains(lines[3], "Taxes") {
		t.Errorf("expected Taxes row, got: %s", lines[3])
	}
}

func TestValidLang(t *testing.T) {
	for _, lang := range []string{"en", "de"} {
		if !ValidLang(lang) {
			t.Errorf("ValidLang(%q) = false, want true", lang)
		}
	}
	for _, lang := range []string{"fr", "EN", "", "en-US"} {
		if ValidLang(lang) {
			t.Errorf("ValidLang(%q) = true, want false", lang)
		}
	}
}

// The two PP CSVs are disjoint: buy/sell documents belong in the portfolio
// file and must never appear as cash-account rows.
func TestWriteAccountTransactionsSkipsPortfolioDocumentTypes(t *testing.T) {
	txns := []*schema.Transaction{
		{DocumentType: "TRADE", ISIN: "IE000YU9K6K2", Date: "2024-06-15", Type: "BUY", GrossAmount: amt(50)},
		{DocumentType: "CRYPTO", ISIN: "X", Date: "2024-06-16", Type: "BUY", NetAmount: amt(100)},
		{DocumentType: "SAVINGSPLAN", ISIN: "Y", Date: "2024-06-17", Type: "BUY", GrossAmount: amt(25)},
		{DocumentType: "ORDER", ISIN: "Z", Date: "2024-06-18"},
	}

	var buf bytes.Buffer
	if err := WriteAccountTransactions(&buf, txns, "en"); err != nil {
		t.Fatalf("WriteAccountTransactions failed: %v", err)
	}

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != 1 {
		t.Errorf("expected header only, got %d lines: %v", len(lines), lines)
	}
}

func TestWriteAccountTransactionsUnknownLang(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteAccountTransactions(&buf, nil, "fr"); err == nil {
		t.Fatal("expected error for unknown lang, got nil")
	}
}

// The Note column carries the order number when there is one, falling back to
// the transaction number, so a PP import can be traced back to the document.
func TestNoteColumnPrefersOrderNumber(t *testing.T) {
	txns := []*schema.Transaction{
		{DocumentType: "TRADE", ISIN: "A", Date: "2024-06-15", Type: "BUY", OrderNumber: "ORD-1", TransactionNumber: "TXN-1"},
		{DocumentType: "TRADE", ISIN: "B", Date: "2024-06-16", Type: "BUY", TransactionNumber: "TXN-2"},
	}

	var buf bytes.Buffer
	if err := WritePortfolioTransactions(&buf, txns, "en"); err != nil {
		t.Fatalf("WritePortfolioTransactions failed: %v", err)
	}

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected header + 2 rows, got %d: %v", len(lines), lines)
	}
	if !strings.HasSuffix(lines[1], "ORD-1") {
		t.Errorf("expected order number in Note, got: %s", lines[1])
	}
	if !strings.HasSuffix(lines[2], "TXN-2") {
		t.Errorf("expected transaction number fallback in Note, got: %s", lines[2])
	}
}

func TestWritePortfolioTransactionsRejectsUnknownTradeType(t *testing.T) {
	txns := []*schema.Transaction{{DocumentType: "TRADE", ISIN: "X", Date: "2024-06-15", Type: "SPLIT"}}

	var buf bytes.Buffer
	if err := WritePortfolioTransactions(&buf, txns, "en"); err == nil {
		t.Fatal("expected error for unknown trade type, got nil")
	}
}

func TestWritePortfolioTransactionsGermanLang(t *testing.T) {
	txns := []*schema.Transaction{
		{DocumentType: "TRADE", ISIN: "IE000YU9K6K2", Date: "2024-06-15", Type: "BUY", Quantity: amt(1), GrossAmount: amt(50), Costs: &schema.Costs{Provision: schema.Num(5, 2), Total: schema.Num(5, 2)}},
		{DocumentType: "TRADE", ISIN: "IE000YU9K6K2", Date: "2024-06-16", Type: "SELL", Quantity: amt(1), GrossAmount: amt(60), Costs: &schema.Costs{Provision: schema.Num(5, 2), Total: schema.Num(5, 2)}},
	}

	var buf bytes.Buffer
	if err := WritePortfolioTransactions(&buf, txns, "de"); err != nil {
		t.Fatalf("WritePortfolioTransactions failed: %v", err)
	}

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if lines[0] != "Datum;Typ;Wert;Stück;ISIN;WKN;Wertpapiername;Gebühren;Steuern;Währung Bruttobetrag;Wechselkurs;Notiz" {
		t.Errorf("unexpected German header: %s", lines[0])
	}
	if !strings.Contains(lines[1], "Kauf") {
		t.Errorf("expected Kauf row, got: %s", lines[1])
	}
	if !strings.Contains(lines[2], "Verkauf") {
		t.Errorf("expected Verkauf row, got: %s", lines[2])
	}
}

func TestWritePortfolioTransactionsGermanUsesSemicolonDelimiter(t *testing.T) {
	txns := []*schema.Transaction{
		{DocumentType: "TRADE", ISIN: "IE000YU9K6K2", Date: "2024-06-15", Type: "BUY", Quantity: amt(1), GrossAmount: amt(50)},
	}

	var buf bytes.Buffer
	if err := WritePortfolioTransactions(&buf, txns, "de"); err != nil {
		t.Fatalf("WritePortfolioTransactions failed: %v", err)
	}
	if strings.Contains(buf.String(), ",") {
		t.Errorf("expected no commas in German-locale output (semicolon-delimited), got: %s", buf.String())
	}
}

// A savings-plan row prints no charge line, so the extracted transaction
// carries no costs block at all. PP is accounting for what the purchase cost,
// so the exporter derives the withheld fee from the gap between the cash that
// moved and the value of the shares, and rounds it to the cent a fee is
// actually charged in. Dropping it would understate the purchase.
func TestWritePortfolioTransactionsDerivesSavingsPlanFee(t *testing.T) {
	// 1,478695 shares at 134,2400 is 198.5000168 of stock against 200.00
	// settled: a withheld fee of 1.50.
	txns := []*schema.Transaction{{
		DocumentType: "SAVINGSPLAN", ISIN: "IE00B3RBWM25", Date: "2025-01-15", Type: "BUY",
		Quantity: amt(1.478695), Price: amt(134.24), NetAmount: amt(-200.00), NetCurrency: "EUR",
	}}

	var buf bytes.Buffer
	if err := WritePortfolioTransactions(&buf, txns, "en"); err != nil {
		t.Fatalf("WritePortfolioTransactions failed: %v", err)
	}
	row := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")[1]
	fields := strings.Split(row, ",")

	if got := fields[2]; got != "200" { // Value: the full cash movement
		t.Errorf("Value = %q, want 200: %s", got, row)
	}
	if got := fields[7]; got != "1.5" { // Fees: derived, rounded to the cent
		t.Errorf("Fees = %q, want 1.5: %s", got, row)
	}
	// The row has no Kurswert, so GrossCurrency is empty and the settlement
	// currency is what the column means.
	if got := fields[9]; got != "EUR" {
		t.Errorf("Currency Gross Amount = %q, want EUR: %s", got, row)
	}
	// No Devisenkurs is printed either, but PP needs a rate and a 0 breaks it.
	if got := fields[10]; got != "1" {
		t.Errorf("Exchange Rate = %q, want 1: %s", got, row)
	}
}

func TestWritePortfolioTransactionsGermanUsesCommaDecimalSeparator(t *testing.T) {
	txns := []*schema.Transaction{
		{DocumentType: "SAVINGSPLAN", ISIN: "IE00B3RBWM25", Date: "2025-01-15", Type: "BUY", Quantity: amt(1.478695), Price: amt(134.24), GrossAmount: amt(200.00)},
	}

	var buf bytes.Buffer
	if err := WritePortfolioTransactions(&buf, txns, "de"); err != nil {
		t.Fatalf("WritePortfolioTransactions failed: %v", err)
	}

	if strings.Contains(buf.String(), ".") {
		t.Errorf("expected no periods in German-locale output (comma-decimal), got: %s", buf.String())
	}
	if !strings.Contains(buf.String(), "1,478695") {
		t.Errorf("expected Shares column as comma-decimal 1,478695, got: %s", buf.String())
	}
}

func TestWriteAccountTransactionsGermanLang(t *testing.T) {
	txns := []*schema.Transaction{
		{DocumentType: "DIVIDEND", ISIN: "IE000YU9K6K2", Date: "2024-06-15", NetAmount: amt(10)},
		{DocumentType: "INTEREST", Date: "2024-06-16", NetAmount: amt(2)},
		{DocumentType: "ACCUMULATING", ISIN: "IE000YU9K6K2", Date: "2024-06-17", WithholdingTax: amt(3)},
	}

	var buf bytes.Buffer
	if err := WriteAccountTransactions(&buf, txns, "de"); err != nil {
		t.Fatalf("WriteAccountTransactions failed: %v", err)
	}

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if !strings.HasPrefix(lines[0], "Datum;Typ;Wert;ISIN;WKN;Wertpapiername;Steuern;Gebühren;Notiz") {
		t.Errorf("unexpected German header: %s", lines[0])
	}
	if !strings.Contains(lines[1], "Dividende") {
		t.Errorf("expected Dividende row, got: %s", lines[1])
	}
	if !strings.Contains(lines[2], "Zinsen") {
		t.Errorf("expected Zinsen row, got: %s", lines[2])
	}
	if !strings.Contains(lines[3], "Steuern") {
		t.Errorf("expected Steuern row, got: %s", lines[3])
	}
}

func TestWritePortfolioTransactionsUnknownLang(t *testing.T) {
	txns := []*schema.Transaction{{DocumentType: "TRADE", ISIN: "X", Date: "2024-06-15", Type: "BUY"}}
	var buf bytes.Buffer
	if err := WritePortfolioTransactions(&buf, txns, "fr"); err == nil {
		t.Fatal("expected error for unknown lang, got nil")
	}
}
