package export

import (
	"bytes"
	"strings"
	"testing"

	"github.com/welworx/flatex-pdf-cli/internal/schema"
)

func TestWritePortfolioTransactionsBuyAndSell(t *testing.T) {
	txns := []*schema.Transaction{
		{DocumentType: "TRADE", ISIN: "IE000YU9K6K2", Date: "2024-06-15", Type: "BUY", Quantity: 1, GrossValue: 50, Provision: 5},
		{DocumentType: "TRADE", ISIN: "IE000YU9K6K2", Date: "2024-06-16", Type: "SELL", Quantity: 1, GrossValue: 60, Provision: 5},
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
		{DocumentType: "CRYPTO", SecurityName: "BITCOIN", Date: "2024-06-15", Type: "BUY", Quantity: 0.01, GrossValue: 500, Provision: 10, FinalAmount: 512},
	}

	var buf bytes.Buffer
	if err := WritePortfolioTransactions(&buf, txns, "en"); err != nil {
		t.Fatalf("WritePortfolioTransactions failed: %v", err)
	}

	if !strings.Contains(buf.String(), "512") {
		t.Errorf("expected Value to use FinalAmount (512), got: %s", buf.String())
	}
}

func TestWriteAccountTransactionsMapsTypes(t *testing.T) {
	txns := []*schema.Transaction{
		{DocumentType: "DIVIDEND", ISIN: "IE000YU9K6K2", Date: "2024-06-15", NetAmount: 10},
		{DocumentType: "INTEREST", Date: "2024-06-16", NetAmount: 2},
		{DocumentType: "ACCUMULATING", ISIN: "IE000YU9K6K2", Date: "2024-06-17", WithholdingTax: 3},
		{DocumentType: "ACCUMULATING", ISIN: "IE000YU9K6K2", Date: "2024-06-18", WithholdingTax: 0}, // no real cash movement, must be skipped
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

func TestWritePortfolioTransactionsRejectsUnknownTradeType(t *testing.T) {
	txns := []*schema.Transaction{{DocumentType: "TRADE", ISIN: "X", Date: "2024-06-15", Type: "SPLIT"}}

	var buf bytes.Buffer
	if err := WritePortfolioTransactions(&buf, txns, "en"); err == nil {
		t.Fatal("expected error for unknown trade type, got nil")
	}
}

func TestWritePortfolioTransactionsGermanLang(t *testing.T) {
	txns := []*schema.Transaction{
		{DocumentType: "TRADE", ISIN: "IE000YU9K6K2", Date: "2024-06-15", Type: "BUY", Quantity: 1, GrossValue: 50, Provision: 5},
		{DocumentType: "TRADE", ISIN: "IE000YU9K6K2", Date: "2024-06-16", Type: "SELL", Quantity: 1, GrossValue: 60, Provision: 5},
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
		{DocumentType: "TRADE", ISIN: "IE000YU9K6K2", Date: "2024-06-15", Type: "BUY", Quantity: 1, GrossValue: 50},
	}

	var buf bytes.Buffer
	if err := WritePortfolioTransactions(&buf, txns, "de"); err != nil {
		t.Fatalf("WritePortfolioTransactions failed: %v", err)
	}
	if strings.Contains(buf.String(), ",") {
		t.Errorf("expected no commas in German-locale output (semicolon-delimited), got: %s", buf.String())
	}
}

func TestWritePortfolioTransactionsEnglishUsesCommaDelimiter(t *testing.T) {
	txns := []*schema.Transaction{
		{DocumentType: "TRADE", ISIN: "IE000YU9K6K2", Date: "2024-06-15", Type: "BUY", Quantity: 1, GrossValue: 50},
	}

	var buf bytes.Buffer
	if err := WritePortfolioTransactions(&buf, txns, "en"); err != nil {
		t.Fatalf("WritePortfolioTransactions failed: %v", err)
	}
	if !strings.Contains(buf.String(), ",") {
		t.Errorf("expected comma-delimited English output, got: %s", buf.String())
	}
}

func TestWriteAccountTransactionsGermanLang(t *testing.T) {
	txns := []*schema.Transaction{
		{DocumentType: "DIVIDEND", ISIN: "IE000YU9K6K2", Date: "2024-06-15", NetAmount: 10},
		{DocumentType: "INTEREST", Date: "2024-06-16", NetAmount: 2},
		{DocumentType: "ACCUMULATING", ISIN: "IE000YU9K6K2", Date: "2024-06-17", WithholdingTax: 3},
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
