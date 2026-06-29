package parser

import (
	"testing"

	"github.com/welworx/flatex-pdf-cli/internal/extractor"
)

// TestParseRouting tests that the Parse function routes TRADE documents correctly.
func TestParseRouting(t *testing.T) {
	text := "Kauf VANECK SPACE INNOVATORS E (IE000YU9K6K2/A3DP9J)\nAusgeführt : 1,058537 St. Kurswert : 50,00 EUR\nKurs : 47,235000 EUR Provision : 0,00 EUR\nDevisenkurs : 1,000000\nAusführungsdatum : 15.06.2026"
	doc := &extractor.ExtractedDocument{
		Filename:     "trade.pdf",
		Text:         text,
		DocumentType: "TRADE",
	}

	_, err := Parse(doc)
	if err != nil {
		t.Errorf("Parse should successfully route TRADE document, got error: %v", err)
	}
}

// TestParseDividendRouting tests that the Parse function routes DIVIDEND documents correctly.
func TestParseDividendRouting(t *testing.T) {
	text := "Nr.4684511050 VANGUARD FTSE ALL-WLD UCI (IE00B3RBWM25/A1JX52)\nSt. : 78,70 Bruttoausschüttung\npro Stück : 0,5459180 USD\nExtag : 18.12.2025 Bruttoausschüttung : 42,96 USD\nValuta : 01.01.2026\n*Einbeh. Steuer : 5,39 EUR\nDevisenkurs : 1,175000\nEndbetrag : 31,17 EUR"
	doc := &extractor.ExtractedDocument{
		Filename:     "dividend.pdf",
		Text:         text,
		DocumentType: "DIVIDEND",
	}

	_, err := Parse(doc)
	if err != nil {
		t.Errorf("Parse should successfully route DIVIDEND document, got error: %v", err)
	}
}

// TestParseAccumulatingRouting tests that the Parse function routes ACCUMULATING documents correctly.
func TestParseAccumulatingRouting(t *testing.T) {
	text := "Nr.4684511050 XTRACKERS IE00 (IE00B5L8K969/A2H514)\nSt. : 4,75 pro Stück : -0,572 USD\nExtag : 15.06.2026 Bruttothesaurierung : -2,72 USD\nValuta : 30.06.2026\nEinbeh. Steuer : 0,00 EUR\nDevisenkurs : 1,080000"
	doc := &extractor.ExtractedDocument{
		Filename:     "thesaurierung.pdf",
		Text:         text,
		DocumentType: "ACCUMULATING",
	}

	_, err := Parse(doc)
	if err != nil {
		t.Errorf("Parse should successfully route ACCUMULATING document, got error: %v", err)
	}
}

// TestParseTradeBuy tests parsing a BUY trade confirmation.
func TestParseTradeBuy(t *testing.T) {
	text := "Kauf VANECK SPACE INNOVATORS E (IE000YU9K6K2/A3DP9J)\nAusgeführt : 1,058537 St. Kurswert : 50,00 EUR\nKurs : 47,235000 EUR Provision : 0,00 EUR\nDevisenkurs : 1,000000\nAusführungsdatum : 15.06.2026"
	doc := &extractor.ExtractedDocument{
		Filename:     "trade_buy.pdf",
		Text:         text,
		DocumentType: "TRADE",
	}

	tx, err := ParseTrade(doc)
	if err != nil {
		t.Fatalf("ParseTrade failed: %v", err)
	}

	// Verify core fields
	if tx.Type != "BUY" {
		t.Errorf("expected Type=BUY, got %s", tx.Type)
	}
	if tx.ISIN != "IE000YU9K6K2" {
		t.Errorf("expected ISIN=IE000YU9K6K2, got %s", tx.ISIN)
	}
	if tx.WKN != "A3DP9J" {
		t.Errorf("expected WKN=A3DP9J, got %s", tx.WKN)
	}
	if tx.Quantity != 1.058537 {
		t.Errorf("expected Quantity=1.058537, got %f", tx.Quantity)
	}
	if tx.Price != 47.235 {
		t.Errorf("expected Price=47.235, got %f", tx.Price)
	}
	if tx.PriceCurrency != "EUR" {
		t.Errorf("expected PriceCurrency=EUR, got %s", tx.PriceCurrency)
	}
	if tx.GrossValue != 50.00 {
		t.Errorf("expected GrossValue=50.00, got %f", tx.GrossValue)
	}
	if tx.Provision != 0.00 {
		t.Errorf("expected Provision=0.00, got %f", tx.Provision)
	}
}

// TestParseTradeIdentifiers verifies extraction of the order number
// (Auftragsnummer), transaction number (Transaktion-Nr.) and execution venue
// (Ausf.platz/-art) from a trade confirmation.
func TestParseTradeIdentifiers(t *testing.T) {
	text := "Auftragsnummer 999888777/1\n" +
		"Ausf.platz/-artXETRA\n" +
		"Wertpapierabrechnung Kauf GLOBAL X COPPER MINERS ET (IE0003Z9E2Y3/A3C7FZ)\n" +
		"Handelstag 30.01.2026\n" +
		"Ausgeführt: 35 St.Kurswert: 2.034,20 EUR\n" +
		"Kurs: 58,120000 EURProvision: 0,00 EUR\n" +
		"Devisenkurs: 1,000000\n" +
		"Details dazu finden Sie im Steuerreport unter der Transaktion-Nr.: 8887776665.\n" +
		"Die Verrechnung der Endbeträge erfolgt über Ihr Konto Nr.: 31022213999"
	doc := &extractor.ExtractedDocument{Filename: "trade.pdf", Text: text, DocumentType: "TRADE"}

	tx, err := ParseTrade(doc)
	if err != nil {
		t.Fatalf("ParseTrade failed: %v", err)
	}
	if tx.OrderNumber != "999888777/1" {
		t.Errorf("OrderNumber = %q, want 999888777/1", tx.OrderNumber)
	}
	if tx.TransactionNumber != "8887776665" {
		t.Errorf("TransactionNumber = %q, want 8887776665", tx.TransactionNumber)
	}
	if tx.ExecutionVenue != "XETRA" {
		t.Errorf("ExecutionVenue = %q, want XETRA", tx.ExecutionVenue)
	}
}

// TestParseCrypto tests parsing a Sammelabrechnung Kryptowerte (crypto settlement).
func TestParseCrypto(t *testing.T) {
	// Layout mirrors gxpdf extraction of the real doc (two columns merged per line).
	text := "Sammelabrechnung (Kauf/-verkauf Kryptowerte)\n" +
		"Ihr Verwahrkonto bei Tangany GmbH: 44000000041\n" +
		"Inhaber: Dr. Stefan Berger\n" +
		"Nr.999000111/1    Kauf                           BITCOIN\n" +
		"Ordervolumen: 0,014 St. Handelsplatz: Tradias\n" +
		"davon ausgef.: 0,014 St. Schlusstag: 29.01.2026, 16:00 Uhr\n" +
		"Kurs: 72.462,2200 EUR Kurswert: 1.014,47 EUR\n" +
		"Devisenkurs: Provision: 5,07 EUR\n" +
		"Bew-Faktor: 1,0000\n" +
		"Verwahrart: Kryptoverwahrung\n" +
		"Kryptoverwahrer: Tangany GmbH **Einbeh. Steuer: 0,00 EUR\n" +
		"Gewinn/Verlust: 0,00 EUR\n" +
		"Valuta: 30.01.2026 Endbetrag: -1.019,54 EUR\n" +
		"** Transaktion-Nr.: 4400000044\n" +
		"Die Verrechnung der Endbeträge erfolgt über Ihr Konto Nr.: 44000000042"
	doc := &extractor.ExtractedDocument{Filename: "krypto.pdf", Text: text, DocumentType: "CRYPTO"}

	tx, err := ParseCrypto(doc)
	if err != nil {
		t.Fatalf("ParseCrypto failed: %v", err)
	}
	checks := []struct {
		name string
		got  interface{}
		want interface{}
	}{
		{"DocumentType", tx.DocumentType, "CRYPTO"},
		{"Type", tx.Type, "BUY"},
		{"SecurityName", tx.SecurityName, "BITCOIN"},
		{"OrderNumber", tx.OrderNumber, "999000111/1"},
		{"TransactionNumber", tx.TransactionNumber, "4400000044"},
		{"Quantity", tx.Quantity, 0.014},
		{"Price", tx.Price, 72462.22},
		{"GrossValue", tx.GrossValue, 1014.47},
		{"Provision", tx.Provision, 5.07},
		{"FinalAmount", tx.FinalAmount, -1019.54},
		{"Date", tx.Date, "2026-01-29"},
		{"ValueDate", tx.ValueDate, "2026-01-30"},
		{"CustodyType", tx.CustodyType, "Kryptoverwahrung"},
		{"Depositary", tx.Depositary, "Tangany GmbH"},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %v, want %v", c.name, c.got, c.want)
		}
	}
}

// TestParseOrderConfirmation tests parsing a Sammelauftragsbestätigung, which
// lists multiple pending orders and must yield one transaction per order.
func TestParseOrderConfirmation(t *testing.T) {
	// Layout mirrors gxpdf extraction of the real doc. Bezeichnung and venue are
	// not always space-separated (see order[1] "…MINERS ETXETRA").
	text := "Sammelauftragsbestätigung\n" +
		"Ihre Depotnummer:33000000031\n" +
		"Depotinhaber:Dr. Lukas Hofer\n" +
		"Auftrags-Nr ISIN Bezeichnung Ausf.platz/-art\n" +
		"WKN Geschäftsart/Auftr.DatumStücke/Nominale\n" +
		"330000111 XFC000A2YY6Q BITCOIN Tradias\n" +
		"992668 Kauf vom 28.01.2026 0,014 St.\n" +
		"Gültig bis: 28.02.2026\n" +
		"Limit: 72.500,000 EUR\n" +
		"330000222 IE0003Z9E2Y3 GLOBAL X COPPER MINERS ETXETRA\n" +
		"A3C7FZ Kauf vom 28.01.2026 35,00 St.\n" +
		"Gültig bis: 27.02.2026\n" +
		"Limit: 59,500 EUR\n"
	doc := &extractor.ExtractedDocument{Filename: "order.pdf", Text: text, DocumentType: "ORDER"}

	txs, err := ParseOrderConfirmation(doc)
	if err != nil {
		t.Fatalf("ParseOrderConfirmation failed: %v", err)
	}
	if len(txs) != 2 {
		t.Fatalf("expected 2 orders, got %d", len(txs))
	}

	a := txs[0]
	if a.OrderNumber != "330000111" || a.ISIN != "XFC000A2YY6Q" || a.SecurityName != "BITCOIN Tradias" ||
		a.WKN != "992668" || a.Type != "BUY" ||
		a.Date != "2026-01-28" || a.Quantity != 0.014 || a.ValidUntil != "2026-02-28" ||
		a.Limit != 72500.0 || a.DocumentType != "ORDER" {
		t.Errorf("order[0] mismatch: %+v", a)
	}

	b := txs[1]
	if b.OrderNumber != "330000222" || b.ISIN != "IE0003Z9E2Y3" || b.SecurityName != "GLOBAL X COPPER MINERS ETXETRA" ||
		b.WKN != "A3C7FZ" || b.Type != "BUY" ||
		b.Quantity != 35.0 || b.ValidUntil != "2026-02-27" || b.Limit != 59.5 {
		t.Errorf("order[1] mismatch: %+v", b)
	}
}

// TestParseDividend tests parsing a DIVIDEND statement.
func TestParseDividend(t *testing.T) {
	text := "Nr.4684511050 VANGUARD FTSE ALL-WLD UCI (IE00B3RBWM25/A1JX52)\nSt. : 78,70 Bruttoausschüttung\npro Stück : 0,5459180 USD\nExtag : 18.12.2025 Bruttoausschüttung : 42,96 USD\nValuta : 01.01.2026\n*Einbeh. Steuer : 5,39 EUR\nDevisenkurs : 1,175000\nEndbetrag : 31,17 EUR"
	doc := &extractor.ExtractedDocument{
		Filename:     "dividend.pdf",
		Text:         text,
		DocumentType: "DIVIDEND",
	}

	tx, err := ParseDividend(doc)
	if err != nil {
		t.Fatalf("ParseDividend failed: %v", err)
	}

	// Verify core fields
	if tx.DocumentType != "DIVIDEND" {
		t.Errorf("expected DocumentType=DIVIDEND, got %s", tx.DocumentType)
	}
	if tx.ISIN != "IE00B3RBWM25" {
		t.Errorf("expected ISIN=IE00B3RBWM25, got %s", tx.ISIN)
	}
	if tx.WKN != "A1JX52" {
		t.Errorf("expected WKN=A1JX52, got %s", tx.WKN)
	}
	if tx.Quantity != 78.70 {
		t.Errorf("expected Quantity=78.70, got %f", tx.Quantity)
	}
	if tx.DistributionPerShare != 0.5459180 {
		t.Errorf("expected DistributionPerShare=0.5459180, got %f", tx.DistributionPerShare)
	}
	if tx.DistributionCurrency != "USD" {
		t.Errorf("expected DistributionCurrency=USD, got %s", tx.DistributionCurrency)
	}
	if tx.GrossAmount != 42.96 {
		t.Errorf("expected GrossAmount=42.96, got %f", tx.GrossAmount)
	}
	if tx.GrossCurrency != "USD" {
		t.Errorf("expected GrossCurrency=USD, got %s", tx.GrossCurrency)
	}
	if tx.WithholdingTax != 5.39 {
		t.Errorf("expected WithholdingTax=5.39, got %f", tx.WithholdingTax)
	}
	if tx.WithholdingTaxCurrency != "EUR" {
		t.Errorf("expected WithholdingTaxCurrency=EUR, got %s", tx.WithholdingTaxCurrency)
	}
	if tx.NetAmount != 31.17 {
		t.Errorf("expected NetAmount=31.17, got %f", tx.NetAmount)
	}
	if tx.NetCurrency != "EUR" {
		t.Errorf("expected NetCurrency=EUR, got %s", tx.NetCurrency)
	}
	if tx.ExchangeRate != 1.175 {
		t.Errorf("expected ExchangeRate=1.175, got %f", tx.ExchangeRate)
	}
	if tx.ExDate != "2025-12-18" {
		t.Errorf("expected ExDate=2025-12-18, got %s", tx.ExDate)
	}
	if tx.ValueDate != "2026-01-01" {
		t.Errorf("expected ValueDate=2026-01-01, got %s", tx.ValueDate)
	}
}

// TestParseInterestRouting tests that the Parse function routes INTEREST documents correctly.
func TestParseInterestRouting(t *testing.T) {
	text := "ISIN: IE00B3RBWM25\nBruttobetrag : 25,50 EUR\nEinbeh. KESt : 3,40 EUR\nEndbetrag : 22,10 EUR\nZinssatz : 2,5%\nZinsperiode : 01.01.2026 bis 31.03.2026\nValuta : 15.04.2026"
	doc := &extractor.ExtractedDocument{
		Filename:     "interest.pdf",
		Text:         text,
		DocumentType: "INTEREST",
	}

	_, err := Parse(doc)
	if err != nil {
		t.Errorf("Parse should successfully route INTEREST document, got error: %v", err)
	}
}

// TestParseInterest tests parsing an INTEREST statement.
func TestParseInterest(t *testing.T) {
	text := "ISIN: IE00B3RBWM25\nBruttobetrag : 25,50 EUR\nEinbeh. KESt : 3,40 EUR\nEndbetrag : 22,10 EUR\nZinssatz : 2,5%\nZinsperiode : 01.01.2026 bis 31.03.2026\nValuta : 15.04.2026"
	doc := &extractor.ExtractedDocument{
		Filename:     "interest.pdf",
		Text:         text,
		DocumentType: "INTEREST",
	}

	tx, err := ParseInterest(doc)
	if err != nil {
		t.Fatalf("ParseInterest failed: %v", err)
	}

	// Verify core fields
	if tx.DocumentType != "INTEREST" {
		t.Errorf("expected DocumentType=INTEREST, got %s", tx.DocumentType)
	}
	if tx.ISIN != "IE00B3RBWM25" {
		t.Errorf("expected ISIN=IE00B3RBWM25, got %s", tx.ISIN)
	}
	if tx.GrossAmount != 25.50 {
		t.Errorf("expected GrossAmount=25.50, got %f", tx.GrossAmount)
	}
	if tx.GrossCurrency != "EUR" {
		t.Errorf("expected GrossCurrency=EUR, got %s", tx.GrossCurrency)
	}
	if tx.WithholdingTax != 3.40 {
		t.Errorf("expected WithholdingTax=3.40, got %f", tx.WithholdingTax)
	}
	if tx.WithholdingTaxCurrency != "EUR" {
		t.Errorf("expected WithholdingTaxCurrency=EUR, got %s", tx.WithholdingTaxCurrency)
	}
	if tx.NetAmount != 22.10 {
		t.Errorf("expected NetAmount=22.10, got %f", tx.NetAmount)
	}
	if tx.NetCurrency != "EUR" {
		t.Errorf("expected NetCurrency=EUR, got %s", tx.NetCurrency)
	}
	if tx.InterestRate != 2.5 {
		t.Errorf("expected InterestRate=2.5, got %f", tx.InterestRate)
	}
	if tx.PeriodFrom != "2026-01-01" {
		t.Errorf("expected PeriodFrom=2026-01-01, got %s", tx.PeriodFrom)
	}
	if tx.PeriodTo != "2026-03-31" {
		t.Errorf("expected PeriodTo=2026-03-31, got %s", tx.PeriodTo)
	}
	if tx.Date != "2026-04-15" {
		t.Errorf("expected Date=2026-04-15, got %s", tx.Date)
	}
}

// TestParseAccumulating tests parsing a ACCUMULATING (reinvestment) statement.
func TestParseAccumulating(t *testing.T) {
	text := "Nr.4684511050 XTRACKERS IE00 (IE00B5L8K969/A2H514)\nSt. : 4,75 pro Stück : -0,572 USD\nExtag : 15.06.2026 Bruttothesaurierung : -2,72 USD\nValuta : 30.06.2026\nEinbeh. Steuer : 0,00 EUR\nDevisenkurs : 1,080000"
	doc := &extractor.ExtractedDocument{
		Filename:     "thesaurierung.pdf",
		Text:         text,
		DocumentType: "ACCUMULATING",
	}

	tx, err := ParseAccumulating(doc)
	if err != nil {
		t.Fatalf("ParseAccumulating failed: %v", err)
	}

	// Verify core fields
	if tx.DocumentType != "ACCUMULATING" {
		t.Errorf("expected DocumentType=ACCUMULATING, got %s", tx.DocumentType)
	}
	if tx.ISIN != "IE00B5L8K969" {
		t.Errorf("expected ISIN=IE00B5L8K969, got %s", tx.ISIN)
	}
	if tx.WKN != "A2H514" {
		t.Errorf("expected WKN=A2H514, got %s", tx.WKN)
	}
	if tx.Quantity != 4.75 {
		t.Errorf("expected Quantity=4.75, got %f", tx.Quantity)
	}
	if tx.ReinvestmentPerShare != -0.572 {
		t.Errorf("expected ReinvestmentPerShare=-0.572, got %f", tx.ReinvestmentPerShare)
	}
	if tx.ReinvestmentCurrency != "USD" {
		t.Errorf("expected ReinvestmentCurrency=USD, got %s", tx.ReinvestmentCurrency)
	}
	if tx.GrossAmount != -2.72 {
		t.Errorf("expected GrossAmount=-2.72, got %f", tx.GrossAmount)
	}
	if tx.GrossCurrency != "USD" {
		t.Errorf("expected GrossCurrency=USD, got %s", tx.GrossCurrency)
	}
	if tx.WithholdingTax != 0.0 {
		t.Errorf("expected WithholdingTax=0.0, got %f", tx.WithholdingTax)
	}
	if tx.WithholdingTaxCurrency != "EUR" {
		t.Errorf("expected WithholdingTaxCurrency=EUR, got %s", tx.WithholdingTaxCurrency)
	}
	if tx.ExchangeRate != 1.08 {
		t.Errorf("expected ExchangeRate=1.08, got %f", tx.ExchangeRate)
	}
	if tx.ExDate != "2026-06-15" {
		t.Errorf("expected ExDate=2026-06-15, got %s", tx.ExDate)
	}
	if tx.ValueDate != "2026-06-30" {
		t.Errorf("expected ValueDate=2026-06-30, got %s", tx.ValueDate)
	}
}

func TestExtractFloatGermanNumbers(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  float64
	}{
		// German format: '.' thousands, ',' decimal
		{"de plain decimal", "Betrag : 72,95 EUR", 72.95},
		{"de thousands separator", "Betrag : 2.034,20 EUR", 2034.20},
		{"de thousands with trailing space", "Betrag : 2.034,20  EUR", 2034.20},
		{"de millions", "Betrag : 1.234.567,89 EUR", 1234567.89},
		{"de negative thousands", "Betrag : -1.500,00 EUR", -1500.00},
		// English format: ',' thousands, '.' decimal
		{"en plain decimal", "Betrag : 72.95 EUR", 72.95},
		{"en thousands separator", "Betrag : 2,034.20 EUR", 2034.20},
		{"en millions", "Betrag : 1,234,567.89 EUR", 1234567.89},
		{"en negative thousands", "Betrag : -1,500.00 EUR", -1500.00},
		// no separators
		{"integer no decimals", "Betrag : 50 EUR", 50},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := extractFloat(tc.input, `Betrag : (-?[\d.,]+)`)
			if err != nil {
				t.Fatalf("extractFloat(%q) returned error: %v", tc.input, err)
			}
			if got != tc.want {
				t.Errorf("extractFloat(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

// TestParseSparplan tests parsing a Sammelabrechnung aus (annual Sparplan settlement).
// The text mirrors gxpdf output: K/V, Buchtag, Valuta, Stücke/Nom., Ausf.-Kurs, Betrag.
func TestParseSparplan(t *testing.T) {
	text := "Sammelabrechnung aus\n" +
		"Ihre Depotnummer: 31022213800\n" +
		"Auftrags-Nr:0003207723\n" +
		"ISIN: IE00B3RBWM25\n" +
		"K/V Buchtag Valuta Stücke/Nom.Ausf.-Kurs Betrag\n" +
		"Kauf 15.01.2025 17.01.2025 1,478695 134,2400 EUR 200,00 EUR\n" +
		"Verkauf 17.02.2025 19.02.2025 1,436948 138,1400 EUR 198,50 EUR\n"
	doc := &extractor.ExtractedDocument{
		Filename:     "sparplan.pdf",
		Text:         text,
		DocumentType: "SPARPLAN",
	}

	txs, err := ParseSparplan(doc)
	if err != nil {
		t.Fatalf("ParseSparplan failed: %v", err)
	}
	if len(txs) != 2 {
		t.Fatalf("expected 2 transactions, got %d", len(txs))
	}

	a := txs[0]
	if a.DocumentType != "SPARPLAN" {
		t.Errorf("DocumentType = %q, want SPARPLAN", a.DocumentType)
	}
	if a.ISIN != "IE00B3RBWM25" {
		t.Errorf("ISIN = %q, want IE00B3RBWM25", a.ISIN)
	}
	if a.OrderNumber != "0003207723" {
		t.Errorf("OrderNumber = %q, want 0003207723", a.OrderNumber)
	}
	if a.Type != "BUY" {
		t.Errorf("Type = %q, want BUY", a.Type)
	}
	if a.Date != "2025-01-15" {
		t.Errorf("Date = %q, want 2025-01-15", a.Date)
	}
	if a.Quantity != 1.478695 {
		t.Errorf("Quantity = %f, want 1.478695", a.Quantity)
	}
	if a.Price != 134.24 {
		t.Errorf("Price = %f, want 134.24", a.Price)
	}
	if a.PriceCurrency != "EUR" {
		t.Errorf("PriceCurrency = %q, want EUR", a.PriceCurrency)
	}
	if a.GrossValue != 200.00 {
		t.Errorf("GrossValue = %f, want 200.00", a.GrossValue)
	}

	b := txs[1]
	if b.Type != "SELL" {
		t.Errorf("Type = %q, want SELL", b.Type)
	}
	if b.Date != "2025-02-17" {
		t.Errorf("Date = %q, want 2025-02-17", b.Date)
	}
	if b.GrossValue != 198.50 {
		t.Errorf("GrossValue = %f, want 198.50", b.GrossValue)
	}
}

// TestParseSparplanRouting verifies Parse() routes SPARPLAN documents correctly.
func TestParseSparplanRouting(t *testing.T) {
	text := "Sammelabrechnung aus\n" +
		"Auftrags-Nr:0003207723\n" +
		"ISIN: IE00B3RBWM25\n" +
		"K/V Buchtag Valuta Stücke/Nom.Ausf.-Kurs Betrag\n" +
		"Kauf 15.01.2025 17.01.2025 1,478695 134,2400 EUR 200,00 EUR\n"
	doc := &extractor.ExtractedDocument{
		Filename:     "sparplan.pdf",
		Text:         text,
		DocumentType: "SPARPLAN",
	}

	txs, err := Parse(doc)
	if err != nil {
		t.Fatalf("Parse routing failed: %v", err)
	}
	if len(txs) != 1 {
		t.Errorf("expected 1 transaction via routing, got %d", len(txs))
	}
}
