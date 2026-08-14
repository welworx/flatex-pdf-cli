package parser

import (
	"strings"
	"testing"

	"github.com/welworx/flatex-pdf-cli/internal/extractor"
	"github.com/welworx/flatex-pdf-cli/internal/schema"
)

// TestParseRoutesDocumentTypes covers the Parse dispatch table, which the
// per-parser tests bypass by calling parseX directly. Each case asserts the
// routed transactions come back tagged with the type they were dispatched on,
// so a mis-wired switch arm cannot pass.
//
// These use synthetic text to keep the dispatch assertions independent of the
// PDF fixtures; TestAllFixturesParse covers the real files end to end.
func TestParseRoutesDocumentTypes(t *testing.T) {
	crypto := "Sammelabrechnung (Kauf/-verkauf Kryptowerte)\n" +
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

	order := "Sammelauftragsbestätigung\n" +
		"Ihre Depotnummer:33000000031\n" +
		"Depotinhaber:Dr. Lukas Hofer\n" +
		"Auftrags-Nr ISIN Bezeichnung Ausf.platz/-art\n" +
		"WKN Geschäftsart/Auftr.DatumStücke/Nominale\n" +
		"330000111 XFC000A2YY6Q BITCOIN Tradias\n" +
		"992668 Kauf vom 28.01.2026 0,014 St.\n" +
		"Gültig bis: 28.02.2026\n" +
		"Limit: 72.500,000 EUR"

	trade := "Kauf VANECK SPACE INNOVATORS E (IE000YU9K6K2/A3DP9J)\nAusgeführt : 1,058537 St. Kurswert : 50,00 EUR\nKurs : 47,235000 EUR Provision : 0,00 EUR\nDevisenkurs : 1,000000\nAusführungsdatum : 15.06.2026"

	dividend := "Nr.4684511050 VANGUARD FTSE ALL-WLD UCI (IE00B3RBWM25/A1JX52)\nSt. : 78,70 Bruttoausschüttung\npro Stück : 0,5459180 USD\nExtag : 18.12.2025 Bruttoausschüttung : 42,96 USD\nValuta : 01.01.2026\n*Einbeh. Steuer : 5,39 EUR\nDevisenkurs : 1,175000\nEndbetrag : 31,17 EUR"

	accumulating := "Nr.4684511050 XTRACKERS IE00 (IE00B5L8K969/A2H514)\nSt. : 4,75 pro Stück : -0,572 USD\nExtag : 15.06.2026 Bruttothesaurierung : -2,72 USD\nValuta : 30.06.2026\nEinbeh. Steuer : 0,00 EUR\nDevisenkurs : 1,080000"

	interest := "ISIN: IE00B3RBWM25\nBruttobetrag : 25,50 EUR\nEinbeh. KESt : 3,40 EUR\nEndbetrag : 22,10 EUR\nZinssatz : 2,5%\nZinsperiode : 01.01.2026 bis 31.03.2026\nValuta : 15.04.2026"

	savingsPlan := "Sammelabrechnung aus\n" +
		"Auftrags-Nr:0005500055\n" +
		"ISIN: IE00B3RBWM25\n" +
		"K/V Buchtag Valuta Stücke/Nom.Ausf.-Kurs Betrag\n" +
		"Kauf 15.01.2025 17.01.2025 1,478695 134,2400 EUR 200,00 EUR\n"

	cases := []struct {
		docType string
		text    string
	}{
		{"TRADE", trade},
		{"DIVIDEND", dividend},
		{"ACCUMULATING", accumulating},
		{"INTEREST", interest},
		{"SAVINGSPLAN", savingsPlan},
		{"CRYPTO", crypto},
		{"ORDER", order},
	}

	for _, tc := range cases {
		t.Run(tc.docType, func(t *testing.T) {
			doc := &extractor.ExtractedDocument{
				Filename:     strings.ToLower(tc.docType) + ".pdf",
				Text:         tc.text,
				DocumentType: tc.docType,
			}

			txs, err := Parse(doc)
			if err != nil {
				t.Fatalf("Parse failed to route %s: %v", tc.docType, err)
			}
			if len(txs) == 0 {
				t.Fatalf("expected at least one %s transaction", tc.docType)
			}
			for i, tx := range txs {
				if tx.DocumentType != tc.docType {
					t.Errorf("txs[%d].DocumentType = %s, want %s", i, tx.DocumentType, tc.docType)
				}
			}
		})
	}
}

// An unrecognised document type must surface as an error; that is what lets a
// batch run skip the file instead of emitting a silently empty result.
func TestParseUnknownDocumentTypeErrors(t *testing.T) {
	doc := &extractor.ExtractedDocument{
		Filename:     "mystery.pdf",
		Text:         "some flatex text",
		DocumentType: "STEUERBESCHEINIGUNG",
	}

	txs, err := Parse(doc)
	if err == nil {
		t.Fatal("expected an error for an unknown document type, got nil")
	}
	if txs != nil {
		t.Errorf("expected no transactions, got %d", len(txs))
	}
	if !strings.Contains(err.Error(), "STEUERBESCHEINIGUNG") {
		t.Errorf("expected the error to name the document type, got: %v", err)
	}
}

// one propagates a parser failure instead of wrapping a nil transaction into a
// one-element slice.
func TestParseSurfacesParserErrorForKnownType(t *testing.T) {
	doc := &extractor.ExtractedDocument{
		Filename:     "empty-crypto.pdf",
		Text:         "Sammelabrechnung (Kauf/-verkauf Kryptowerte)\n",
		DocumentType: "CRYPTO",
	}

	txs, err := Parse(doc)
	if err == nil {
		t.Fatal("expected an error from the CRYPTO parser, got nil")
	}
	if txs != nil {
		t.Errorf("expected no transactions on error, got %d", len(txs))
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

	tx, err := parseTrade(doc)
	if err != nil {
		t.Fatalf("parseTrade failed: %v", err)
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
	if schema.Amount(tx.Quantity) != 1.058537 {
		t.Errorf("expected Quantity=1.058537, got %f", schema.Amount(tx.Quantity))
	}
	if schema.Amount(tx.Price) != 47.235 {
		t.Errorf("expected Price=47.235, got %f", schema.Amount(tx.Price))
	}
	if tx.GrossCurrency != "EUR" {
		t.Errorf("expected GrossCurrency=EUR, got %s", tx.GrossCurrency)
	}
	if schema.Amount(tx.GrossAmount) != 50.00 {
		t.Errorf("expected GrossAmount=50.00, got %f", schema.Amount(tx.GrossAmount))
	}
	if tx.Costs == nil {
		t.Fatal("expected a cost block, got nil")
	}
	if tx.Costs.Provision.Float() != 0.00 {
		t.Errorf("expected Provision=0.00, got %f", tx.Costs.Provision.Float())
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
		"Die Verrechnung der Endbeträge erfolgt über Ihr Konto Nr.: 55000000999"
	doc := &extractor.ExtractedDocument{Filename: "trade.pdf", Text: text, DocumentType: "TRADE"}

	tx, err := parseTrade(doc)
	if err != nil {
		t.Fatalf("parseTrade failed: %v", err)
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

	tx, err := parseCrypto(doc)
	if err != nil {
		t.Fatalf("parseCrypto failed: %v", err)
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
		{"Quantity", schema.Amount(tx.Quantity), 0.014},
		{"Price", schema.Amount(tx.Price), 72462.22},
		{"GrossAmount", schema.Amount(tx.GrossAmount), 1014.47},
		{"Provision", tx.Costs.Provision.Float(), 5.07},
		{"TotalCosts", tx.TotalCosts(), 5.07},
		{"NetAmount", schema.Amount(tx.NetAmount), -1019.54},
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

	txs, err := parseOrderConfirmation(doc)
	if err != nil {
		t.Fatalf("parseOrderConfirmation failed: %v", err)
	}
	if len(txs) != 2 {
		t.Fatalf("expected 2 orders, got %d", len(txs))
	}

	a := txs[0]
	if a.OrderNumber != "330000111" || a.ISIN != "XFC000A2YY6Q" || a.SecurityName != "BITCOIN Tradias" ||
		a.WKN != "992668" || a.Type != "BUY" ||
		a.Date != "2026-01-28" || schema.Amount(a.Quantity) != 0.014 || a.ValidUntil != "2026-02-28" ||
		schema.Amount(a.Limit) != 72500.0 || a.DocumentType != "ORDER" {
		t.Errorf("order[0] mismatch: %+v", a)
	}

	b := txs[1]
	if b.OrderNumber != "330000222" || b.ISIN != "IE0003Z9E2Y3" || b.SecurityName != "GLOBAL X COPPER MINERS ETXETRA" ||
		b.WKN != "A3C7FZ" || b.Type != "BUY" ||
		schema.Amount(b.Quantity) != 35.0 || b.ValidUntil != "2026-02-27" || schema.Amount(b.Limit) != 59.5 {
		t.Errorf("order[1] mismatch: %+v", b)
	}
}

// TestParseRefundedWithholdingTax covers a negative "Einbeh. KESt"/"Einbeh.
// Steuer". Austrian KESt is levied on the Gewinn/Verlust, so a realised loss
// refunds tax already withheld earlier in the year out of the
// Verluststeuertopf and the document states the amount with a minus sign.
// Every document type has to accept it: the extraction patterns once required
// a leading digit, which made the whole statement fail to parse rather than
// yield a negative figure.
func TestParseRefundedWithholdingTax(t *testing.T) {
	cases := []struct {
		docType string
		text    string
		want    float64
	}{
		{
			"TRADE",
			"Kauf VANECK SPACE INNOVATORS E (IE000YU9K6K2/A3DP9J)\nAusgeführt : 1,058537 St. Kurswert : 50,00 EUR\nKurs : 47,235000 EUR Provision : 0,00 EUR\nDevisenkurs : 1,000000\nGewinn/Verlust: -120,00 EUR **Einbeh. KESt : -33,00 EUR\nAusführungsdatum : 15.06.2026",
			-33.00,
		},
		{
			"DIVIDEND",
			"Nr.4684511050 VANGUARD FTSE ALL-WLD UCI (IE00B3RBWM25/A1JX52)\nSt. : 78,70 Bruttoausschüttung\npro Stück : 0,5459180 USD\nExtag : 18.12.2025 Bruttoausschüttung : 42,96 USD\nValuta : 01.01.2026\n*Einbeh. Steuer : -5,39 EUR\nDevisenkurs : 1,175000\nEndbetrag : 31,17 EUR",
			-5.39,
		},
		{
			"INTEREST",
			"ISIN: IE00B3RBWM25\nBruttobetrag : 25,50 EUR\nEinbeh. KESt : -3,40 EUR\nEndbetrag : 28,90 EUR\nZinssatz : 2,5%\nZinsperiode : 01.01.2026 bis 31.03.2026\nValuta : 15.04.2026",
			-3.40,
		},
	}

	for _, tc := range cases {
		t.Run(tc.docType, func(t *testing.T) {
			doc := &extractor.ExtractedDocument{Filename: "x.pdf", Text: tc.text, DocumentType: tc.docType}
			txs, err := Parse(doc)
			if err != nil {
				t.Fatalf("a refunded tax must parse, got: %v", err)
			}
			if got := schema.Amount(txs[0].WithholdingTax); got != tc.want {
				t.Errorf("withholding tax = %.2f, want %.2f", got, tc.want)
			}
		})
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

	tx, err := parseDividend(doc)
	if err != nil {
		t.Fatalf("parseDividend failed: %v", err)
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
	if schema.Amount(tx.Quantity) != 78.70 {
		t.Errorf("expected Quantity=78.70, got %f", schema.Amount(tx.Quantity))
	}
	if schema.Amount(tx.DistributionPerShare) != 0.5459180 {
		t.Errorf("expected DistributionPerShare=0.5459180, got %f", schema.Amount(tx.DistributionPerShare))
	}
	if tx.DistributionCurrency != "USD" {
		t.Errorf("expected DistributionCurrency=USD, got %s", tx.DistributionCurrency)
	}
	if schema.Amount(tx.GrossAmount) != 42.96 {
		t.Errorf("expected GrossAmount=42.96, got %f", schema.Amount(tx.GrossAmount))
	}
	if tx.GrossCurrency != "USD" {
		t.Errorf("expected GrossCurrency=USD, got %s", tx.GrossCurrency)
	}
	if schema.Amount(tx.WithholdingTax) != 5.39 {
		t.Errorf("expected WithholdingTax=5.39, got %f", schema.Amount(tx.WithholdingTax))
	}
	if tx.WithholdingTaxCurrency != "EUR" {
		t.Errorf("expected WithholdingTaxCurrency=EUR, got %s", tx.WithholdingTaxCurrency)
	}
	if schema.Amount(tx.NetAmount) != 31.17 {
		t.Errorf("expected NetAmount=31.17, got %f", schema.Amount(tx.NetAmount))
	}
	if tx.NetCurrency != "EUR" {
		t.Errorf("expected NetCurrency=EUR, got %s", tx.NetCurrency)
	}
	if schema.Amount(tx.ExchangeRate) != 1.175 {
		t.Errorf("expected ExchangeRate=1.175, got %f", schema.Amount(tx.ExchangeRate))
	}
	if tx.ExDate != "2025-12-18" {
		t.Errorf("expected ExDate=2025-12-18, got %s", tx.ExDate)
	}
	if tx.ValueDate != "2026-01-01" {
		t.Errorf("expected ValueDate=2026-01-01, got %s", tx.ValueDate)
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

	tx, err := parseInterest(doc)
	if err != nil {
		t.Fatalf("parseInterest failed: %v", err)
	}

	// Verify core fields
	if tx.DocumentType != "INTEREST" {
		t.Errorf("expected DocumentType=INTEREST, got %s", tx.DocumentType)
	}
	if tx.ISIN != "IE00B3RBWM25" {
		t.Errorf("expected ISIN=IE00B3RBWM25, got %s", tx.ISIN)
	}
	if schema.Amount(tx.GrossAmount) != 25.50 {
		t.Errorf("expected GrossAmount=25.50, got %f", schema.Amount(tx.GrossAmount))
	}
	if tx.GrossCurrency != "EUR" {
		t.Errorf("expected GrossCurrency=EUR, got %s", tx.GrossCurrency)
	}
	if schema.Amount(tx.WithholdingTax) != 3.40 {
		t.Errorf("expected WithholdingTax=3.40, got %f", schema.Amount(tx.WithholdingTax))
	}
	if tx.WithholdingTaxCurrency != "EUR" {
		t.Errorf("expected WithholdingTaxCurrency=EUR, got %s", tx.WithholdingTaxCurrency)
	}
	if schema.Amount(tx.NetAmount) != 22.10 {
		t.Errorf("expected NetAmount=22.10, got %f", schema.Amount(tx.NetAmount))
	}
	if tx.NetCurrency != "EUR" {
		t.Errorf("expected NetCurrency=EUR, got %s", tx.NetCurrency)
	}
	if schema.Amount(tx.InterestRate) != 2.5 {
		t.Errorf("expected InterestRate=2.5, got %f", schema.Amount(tx.InterestRate))
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

	tx, err := parseAccumulating(doc)
	if err != nil {
		t.Fatalf("parseAccumulating failed: %v", err)
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
	if schema.Amount(tx.Quantity) != 4.75 {
		t.Errorf("expected Quantity=4.75, got %f", schema.Amount(tx.Quantity))
	}
	if schema.Amount(tx.ReinvestmentPerShare) != -0.572 {
		t.Errorf("expected ReinvestmentPerShare=-0.572, got %f", schema.Amount(tx.ReinvestmentPerShare))
	}
	if tx.ReinvestmentCurrency != "USD" {
		t.Errorf("expected ReinvestmentCurrency=USD, got %s", tx.ReinvestmentCurrency)
	}
	if schema.Amount(tx.GrossAmount) != -2.72 {
		t.Errorf("expected GrossAmount=-2.72, got %f", schema.Amount(tx.GrossAmount))
	}
	if tx.GrossCurrency != "USD" {
		t.Errorf("expected GrossCurrency=USD, got %s", tx.GrossCurrency)
	}
	if schema.Amount(tx.WithholdingTax) != 0.0 {
		t.Errorf("expected WithholdingTax=0.0, got %f", schema.Amount(tx.WithholdingTax))
	}
	if tx.WithholdingTaxCurrency != "EUR" {
		t.Errorf("expected WithholdingTaxCurrency=EUR, got %s", tx.WithholdingTaxCurrency)
	}
	if schema.Amount(tx.ExchangeRate) != 1.08 {
		t.Errorf("expected ExchangeRate=1.08, got %f", schema.Amount(tx.ExchangeRate))
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
		name string
		// wantText is the value as it must be rendered back out: the digits
		// the input printed, not Go's shortest round-trip form. "50" must not
		// become "50.00", and "110,000000" must not collapse to "110".
		input, wantText string
		want            float64
	}{
		// German format: '.' thousands, ',' decimal
		{"de plain decimal", "Betrag : 72,95 EUR", "72.95", 72.95},
		{"de thousands separator", "Betrag : 2.034,20 EUR", "2034.20", 2034.20},
		{"de thousands with trailing space", "Betrag : 2.034,20  EUR", "2034.20", 2034.20},
		{"de millions", "Betrag : 1.234.567,89 EUR", "1234567.89", 1234567.89},
		{"de negative thousands", "Betrag : -1.500,00 EUR", "-1500.00", -1500.00},
		// English format: ',' thousands, '.' decimal
		{"en plain decimal", "Betrag : 72.95 EUR", "72.95", 72.95},
		{"en thousands separator", "Betrag : 2,034.20 EUR", "2034.20", 2034.20},
		{"en millions", "Betrag : 1,234,567.89 EUR", "1234567.89", 1234567.89},
		{"en negative thousands", "Betrag : -1,500.00 EUR", "-1500.00", -1500.00},
		// no separators
		{"integer no decimals", "Betrag : 50 EUR", "50", 50},
		// the precision a Kurs line states, which shortest-form marshalling
		// used to throw away
		{"six decimal places", "Betrag : 110,000000 EUR", "110.000000", 110},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := extractFloat(tc.input, `Betrag : (-?[\d.,]+)`)
			if err != nil {
				t.Fatalf("extractFloat(%q) returned error: %v", tc.input, err)
			}
			if got.Float() != tc.want {
				t.Errorf("extractFloat(%q) = %v, want %v", tc.input, got.Float(), tc.want)
			}
			if got.String() != tc.wantText {
				t.Errorf("extractFloat(%q) renders as %q, want %q", tc.input, got.String(), tc.wantText)
			}
		})
	}
}

// TestParseSavingsPlan tests parsing a Sammelabrechnung aus (annual savings-plan settlement).
// The text mirrors gxpdf output: K/V, Buchtag, Valuta, Stücke/Nom., Ausf.-Kurs, Betrag.
func TestParseSavingsPlan(t *testing.T) {
	text := "Sammelabrechnung aus\n" +
		"Ihre Depotnummer: 55000000051\n" +
		"Auftrags-Nr:0005500055\n" +
		"ISIN: IE00B3RBWM25\n" +
		"K/V Buchtag Valuta Stücke/Nom.Ausf.-Kurs Betrag\n" +
		"Kauf 15.01.2025 17.01.2025 1,478695 134,2400 EUR 200,00 EUR\n" +
		"Verkauf 17.02.2025 19.02.2025 1,436948 138,1400 EUR 198,50 EUR\n"
	doc := &extractor.ExtractedDocument{
		Filename:     "savingsplan.pdf",
		Text:         text,
		DocumentType: "SAVINGSPLAN",
	}

	txs, err := parseSavingsPlan(doc)
	if err != nil {
		t.Fatalf("parseSavingsPlan failed: %v", err)
	}
	if len(txs) != 2 {
		t.Fatalf("expected 2 transactions, got %d", len(txs))
	}

	a := txs[0]
	if a.DocumentType != "SAVINGSPLAN" {
		t.Errorf("DocumentType = %q, want SAVINGSPLAN", a.DocumentType)
	}
	if a.ISIN != "IE00B3RBWM25" {
		t.Errorf("ISIN = %q, want IE00B3RBWM25", a.ISIN)
	}
	if a.OrderNumber != "0005500055" {
		t.Errorf("OrderNumber = %q, want 0005500055", a.OrderNumber)
	}
	if a.Type != "BUY" {
		t.Errorf("Type = %q, want BUY", a.Type)
	}
	if a.Date != "2025-01-15" {
		t.Errorf("Date = %q, want 2025-01-15", a.Date)
	}
	if schema.Amount(a.Quantity) != 1.478695 {
		t.Errorf("Quantity = %f, want 1.478695", schema.Amount(a.Quantity))
	}
	if schema.Amount(a.Price) != 134.24 {
		t.Errorf("Price = %f, want 134.24", schema.Amount(a.Price))
	}
	if a.GrossCurrency != "EUR" {
		t.Errorf("GrossCurrency = %q, want EUR", a.GrossCurrency)
	}
	// GrossAmount is the value of the shares (Stücke x Kurs), not the Betrag
	// column: the 200.00 settled buys 198.50 worth of shares, and the 1.50
	// difference is a charge the Sammelabrechnung never prints as a line item.
	if schema.Amount(a.GrossAmount) != 198.50 {
		t.Errorf("GrossAmount = %f, want 198.50", schema.Amount(a.GrossAmount))
	}
	if a.Costs == nil {
		t.Fatal("Costs = nil, want the derived charge")
	}
	if schema.Amount(a.Costs.Unitemised) != 1.50 {
		t.Errorf("Costs.Unitemised = %f, want 1.50", schema.Amount(a.Costs.Unitemised))
	}
	if a.Costs.Total.Float() != 1.50 {
		t.Errorf("Costs.Total = %f, want 1.50", a.Costs.Total.Float())
	}
	// Buys move cash out, matching NetAmount on a trade confirmation.
	if schema.Amount(a.NetAmount) != -200.00 {
		t.Errorf("NetAmount = %f, want -200.00", schema.Amount(a.NetAmount))
	}

	b := txs[1]
	if b.Type != "SELL" {
		t.Errorf("Type = %q, want SELL", b.Type)
	}
	if b.Date != "2025-02-17" {
		t.Errorf("Date = %q, want 2025-02-17", b.Date)
	}
	// 1,436948 x 138,14 is 198.50, exactly what this row settled, so there is
	// no gap and no charge to recover.
	if schema.Amount(b.GrossAmount) != 198.50 {
		t.Errorf("GrossAmount = %f, want 198.50", schema.Amount(b.GrossAmount))
	}
	if schema.Amount(b.Costs.Unitemised) != 0 {
		t.Errorf("Costs.Unitemised = %f, want 0", schema.Amount(b.Costs.Unitemised))
	}
	if schema.Amount(b.NetAmount) != 198.50 {
		t.Errorf("NetAmount = %f, want 198.50", schema.Amount(b.NetAmount))
	}
}

// TestSavingsPlanChargeBound asserts that a settlement gap too large to be a
// fee fails the parse instead of being booked as a suspiciously large charge.
// This is the check that a layout change actually trips: once the charge is
// recovered the row reconciles by construction, so the bound is the only part
// that can fail.
func TestSavingsPlanChargeBound(t *testing.T) {
	tests := []struct {
		name    string
		row     string
		wantErr bool
	}{
		{
			name: "ordinary charge",
			row:  "Kauf 15.01.2025 17.01.2025 1,478695 134,2400 EUR 200,00 EUR\n",
		},
		{
			name:    "shares worth more than was settled",
			row:     "Kauf 15.01.2025 17.01.2025 1,478695 134,2400 EUR 150,00 EUR\n",
			wantErr: true,
		},
		{
			name:    "gap far too large to be a fee",
			row:     "Kauf 15.01.2025 17.01.2025 1,478695 134,2400 EUR 900,00 EUR\n",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			text := "Sammelabrechnung aus\nAuftrags-Nr:0005500055\nISIN: IE00B3RBWM25\n" +
				"K/V Buchtag Valuta Stücke/Nom.Ausf.-Kurs Betrag\n" + tc.row
			doc := &extractor.ExtractedDocument{Filename: "x.pdf", Text: text, DocumentType: "SAVINGSPLAN"}
			_, err := parseSavingsPlan(doc)
			if tc.wantErr && err == nil {
				t.Fatal("expected the parse to fail, got no error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected the parse to succeed, got %v", err)
			}
		})
	}
}

// runMissingFieldCases asserts that removing each given substring from base
// causes the parser to return an error, verifying the "field not found"
// guard clauses that well-formed fixtures never exercise.
func runMissingFieldCases(t *testing.T, docType string, base string, cases []struct {
	name   string
	remove string
}, parse func(doc *extractor.ExtractedDocument) error) {
	t.Helper()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			text := strings.Replace(base, c.remove, "", 1)
			if text == base {
				t.Fatalf("substring %q not found in base text", c.remove)
			}
			doc := &extractor.ExtractedDocument{Filename: "x.pdf", Text: text, DocumentType: docType}
			if err := parse(doc); err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestParseTradeMissingRequiredFields(t *testing.T) {
	base := "Kauf VANECK SPACE INNOVATORS E (IE000YU9K6K2/A3DP9J)\nAusgeführt : 1,058537 St. Kurswert : 50,00 EUR\nKurs : 47,235000 EUR Provision : 0,00 EUR\nDevisenkurs : 1,000000\nAusführungsdatum : 15.06.2026"
	cases := []struct {
		name   string
		remove string
	}{
		{"missing ISIN", "(IE000YU9K6K2/A3DP9J)"},
		{"missing date", "Ausführungsdatum : 15.06.2026"},
		{"missing quantity", "Ausgeführt : 1,058537 St."},
		{"missing price", "Kurs : 47,235000 EUR"},
		{"missing gross value", "Kurswert : 50,00 EUR"},
	}
	runMissingFieldCases(t, "TRADE", base, cases, func(doc *extractor.ExtractedDocument) error {
		_, err := parseTrade(doc)
		return err
	})
}

func TestParseCryptoMissingRequiredFields(t *testing.T) {
	base := "Sammelabrechnung (Kauf/-verkauf Kryptowerte)\n" +
		"Nr.999000111/1    Kauf                           BITCOIN\n" +
		"Ordervolumen: 0,014 St. Handelsplatz: Tradias\n" +
		"davon ausgef.: 0,014 St. Schlusstag: 29.01.2026, 16:00 Uhr\n" +
		"Kurs: 72.462,2200 EUR Kurswert: 1.014,47 EUR\n" +
		"Devisenkurs: Provision: 5,07 EUR\n" +
		"Valuta: 30.01.2026 Endbetrag: -1.019,54 EUR\n"
	cases := []struct {
		name   string
		remove string
	}{
		{"missing order line", "Nr.999000111/1    Kauf                           BITCOIN"},
		{"missing security name", "Kauf                           BITCOIN"},
		{"missing trade date", "Schlusstag: 29.01.2026, 16:00 Uhr"},
		{"missing quantity", "davon ausgef.: 0,014 St."},
		{"missing price", "Kurs: 72.462,2200 EUR"},
		{"missing gross value", "Kurswert: 1.014,47 EUR"},
	}
	runMissingFieldCases(t, "CRYPTO", base, cases, func(doc *extractor.ExtractedDocument) error {
		_, err := parseCrypto(doc)
		return err
	})
}

func TestParseOrderConfirmationNoOrders(t *testing.T) {
	doc := &extractor.ExtractedDocument{Filename: "order.pdf", Text: "no matching order blocks here", DocumentType: "ORDER"}
	if _, err := parseOrderConfirmation(doc); err == nil {
		t.Error("expected error when no order blocks match, got nil")
	}
}

func TestParseDividendMissingRequiredFields(t *testing.T) {
	base := "Nr.4684511050 VANGUARD FTSE ALL-WLD UCI (IE00B3RBWM25/A1JX52)\nSt. : 78,70 Bruttoausschüttung\npro Stück : 0,5459180 USD\nExtag : 18.12.2025 Bruttoausschüttung : 42,96 USD\nValuta : 01.01.2026\n*Einbeh. Steuer : 5,39 EUR\nDevisenkurs : 1,175000\nEndbetrag : 31,17 EUR"
	cases := []struct {
		name   string
		remove string
	}{
		{"missing ISIN", "(IE00B3RBWM25/A1JX52)"},
		{"missing value date", "Valuta : 01.01.2026"},
		{"missing quantity", "St. : 78,70 Bruttoausschüttung"},
		{"missing distribution per share", "pro Stück : 0,5459180 USD"},
		{"missing gross amount", "Bruttoausschüttung : 42,96 USD"},
		{"missing withholding tax", "Einbeh. Steuer : 5,39 EUR"},
		{"missing net amount", "Endbetrag : 31,17 EUR"},
	}
	runMissingFieldCases(t, "DIVIDEND", base, cases, func(doc *extractor.ExtractedDocument) error {
		_, err := parseDividend(doc)
		return err
	})
}

func TestParseInterestMissingRequiredFields(t *testing.T) {
	base := "ISIN: IE00B3RBWM25\nBruttobetrag : 25,50 EUR\nEinbeh. KESt : 3,40 EUR\nEndbetrag : 22,10 EUR\nZinssatz : 2,5%\nZinsperiode : 01.01.2026 bis 31.03.2026\nValuta : 15.04.2026"
	cases := []struct {
		name   string
		remove string
	}{
		{"missing ISIN", "IE00B3RBWM25"},
		{"missing value date", "Valuta : 15.04.2026"},
		{"missing gross amount", "Bruttobetrag : 25,50 EUR"},
		{"missing withholding tax", "Einbeh. KESt : 3,40 EUR"},
		{"missing net amount", "Endbetrag : 22,10 EUR"},
		{"missing interest rate", "Zinssatz : 2,5%"},
	}
	runMissingFieldCases(t, "INTEREST", base, cases, func(doc *extractor.ExtractedDocument) error {
		_, err := parseInterest(doc)
		return err
	})
}

func TestParseAccumulatingMissingRequiredFields(t *testing.T) {
	base := "Nr.4684511050 XTRACKERS IE00 (IE00B5L8K969/A2H514)\nSt. : 4,75 pro Stück : -0,572 USD\nExtag : 15.06.2026 Bruttothesaurierung : -2,72 USD\nValuta : 30.06.2026\nEinbeh. Steuer : 0,00 EUR\nDevisenkurs : 1,080000"
	cases := []struct {
		name   string
		remove string
	}{
		{"missing ISIN", "(IE00B5L8K969/A2H514)"},
		{"missing value date", "Valuta : 30.06.2026"},
		{"missing quantity", "St. : 4,75 "},
		{"missing reinvestment per share", "Stück : -0,572 USD"},
		{"missing gross amount", "Bruttothesaurierung : -2,72 USD"},
	}
	runMissingFieldCases(t, "ACCUMULATING", base, cases, func(doc *extractor.ExtractedDocument) error {
		_, err := parseAccumulating(doc)
		return err
	})
}

func TestParseSavingsPlanMissingRequiredFields(t *testing.T) {
	t.Run("missing ISIN", func(t *testing.T) {
		text := "Sammelabrechnung aus\nAuftrags-Nr:0005500055\n" +
			"Kauf 15.01.2025 17.01.2025 1,478695 134,2400 EUR 200,00 EUR\n"
		doc := &extractor.ExtractedDocument{Filename: "sp.pdf", Text: text, DocumentType: "SAVINGSPLAN"}
		if _, err := parseSavingsPlan(doc); err == nil {
			t.Error("expected error when ISIN missing, got nil")
		}
	})

	t.Run("no rows found", func(t *testing.T) {
		text := "Sammelabrechnung aus\nISIN: IE00B3RBWM25\n"
		doc := &extractor.ExtractedDocument{Filename: "sp.pdf", Text: text, DocumentType: "SAVINGSPLAN"}
		if _, err := parseSavingsPlan(doc); err == nil {
			t.Error("expected error when no table rows match, got nil")
		}
	})
}

// TestAllFixturesParse runs every committed PDF fixture through the real
// extract-and-parse path and checks the identifiers that redaction is most
// likely to disturb.
//
// The values it asserts are the ones a PDF writer can silently move: PyMuPDF
// appends replacement text as a new content stream, so a re-redacted fixture
// renders correctly while every replaced identifier drops out of its slot in
// stream order. That failure is invisible to a test built on synthetic text,
// which is why this one reads the files.
func TestAllFixturesParse(t *testing.T) {
	cases := []struct {
		file              string
		docType           string
		wantTransactions  int
		orderNumber       string
		transactionNumber string
		depotNumber       string
		depotHolder       string
		date              string // txs[0].Date — the trade/value date, not the letter date
		orderDate         string
		valueDate         string
		executionTime     string
		depositCountry    string
		tradeType         string
		securityName      string
		gainLoss          *schema.Decimal
		withholdingTax    *schema.Decimal
	}{
		{
			file: "trade_sample_1.pdf", docType: "TRADE", wantTransactions: 1,
			orderNumber: "700000011/1", transactionNumber: "7000000011",
			depotNumber: "11000000011", depotHolder: "Mustermann, Max",
			// Letter date is 16.09.2025; Handelstag is 15.09.2025.
			date: "2025-09-15", orderDate: "2025-09-15", valueDate: "2025-09-17",
			depositCountry: "GB", securityName: "L&G GOLD MINING ETF",
		},
		{
			file: "trade_sample_2.pdf", docType: "TRADE", wantTransactions: 1,
			orderNumber: "800000022/1", transactionNumber: "7000000022",
			depotNumber: "22000000021", depotHolder: "Beispiel, Erika",
			// Auftragsdatum 28.01. and Valuta 03.02. straddle Handelstag 30.01.
			date: "2026-01-30", orderDate: "2026-01-28", valueDate: "2026-02-03",
			// Lagerland runs into the next column in gxpdf's output here.
			depositCountry: "GB",
			// Older layout prints "Nr. 800000022/1" with a space after the dot.
			securityName: "GLOBAL X COPPER MINERS ET",
		},
		{
			file: "trade_sample_3.pdf", docType: "TRADE", wantTransactions: 1,
			orderNumber: "880000088/1", transactionNumber: "8800000088",
			depotNumber: "88000000081", depotHolder: "Steiner, Felix",
			// Letter date, Auftragsdatum and Handelstag are all 15.01.2025.
			date: "2025-01-15", orderDate: "2025-01-15", valueDate: "2025-01-17",
			// This is the older layout, whose mono body font is StandardEncoding:
			// gxpdf decodes it as WinAnsi, so "Großbritannien" arrives as
			// "Groûbritannien" and yields no country unless extractTextFromPDF
			// repairs it. Asserting GB here is what pins that fix.
			depositCountry: "GB",
			tradeType:      "BUY",
			securityName:   "VANGUARD FTSE ALL-WLD UCI",
		},
		{
			// The only sale in the corpus. It is what pins the SELL side of
			// checkSettlement (deductions come off the proceeds, not onto
			// them) and the only fixture with a non-zero Gewinn/Verlust.
			file: "verkauf_sample_1.pdf", docType: "TRADE", wantTransactions: 1,
			orderNumber: "990000099/1", transactionNumber: "9900000099",
			depotNumber: "99000000091", depotHolder: "Wallner, Sophie",
			date: "2024-12-18", orderDate: "2024-12-17", valueDate: "2024-12-20",
			executionTime:  "13:56",
			depositCountry: "GB", tradeType: "SELL",
			securityName: "VANGUARD S&P 500 ETF",
			gainLoss:     amt(403.97), withholdingTax: amt(24.51),
		},
		{
			file: "krypto_sample_1.pdf", docType: "CRYPTO", wantTransactions: 1,
			orderNumber: "660000111/1", transactionNumber: "6600000066",
			date: "2026-01-29", valueDate: "2026-01-30", executionTime: "16:00",
		},
		{
			file: "orderbestaetigung_sample_1.pdf", docType: "ORDER", wantTransactions: 2,
			orderNumber: "770000111",
			depotNumber: "77000000071", depotHolder: "Hofer, Lukas",
			date: "2026-01-28",
		},
		{
			file: "dividend_sample_1.pdf", docType: "DIVIDEND", wantTransactions: 1,
			depotNumber: "33000000031", depotHolder: "Österreicher, Johann",
			date: "2025-10-01", valueDate: "2025-10-01",
		},
		{
			file: "dividend_sample_2.pdf", docType: "DIVIDEND", wantTransactions: 1,
			depotNumber: "44000000041", depotHolder: "Gruber, Anna-Maria",
			date: "2025-10-01", valueDate: "2025-10-01",
		},
		{
			file: "sparplan_sample_1.pdf", docType: "SAVINGSPLAN", wantTransactions: 12,
			depotNumber: "55000000051", depotHolder: "Dr. Klaus Bergmann",
			date: "2025-01-15", valueDate: "2025-01-17", tradeType: "BUY",
		},
	}

	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			doc, err := extractor.ExtractPDF("../../testdata/" + tc.file)
			if err != nil {
				t.Fatalf("ExtractPDF: %v", err)
			}
			if doc.DocumentType != tc.docType {
				t.Errorf("document type = %q, want %q", doc.DocumentType, tc.docType)
			}
			if tc.depotNumber != "" && doc.DepotNumber != tc.depotNumber {
				t.Errorf("depot number = %q, want %q", doc.DepotNumber, tc.depotNumber)
			}
			if tc.depotHolder != "" && doc.DepotHolder != tc.depotHolder {
				t.Errorf("depot holder = %q, want %q", doc.DepotHolder, tc.depotHolder)
			}

			txs, err := Parse(doc)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if len(txs) != tc.wantTransactions {
				t.Fatalf("got %d transactions, want %d", len(txs), tc.wantTransactions)
			}
			if tc.orderNumber != "" && txs[0].OrderNumber != tc.orderNumber {
				t.Errorf("order number = %q, want %q", txs[0].OrderNumber, tc.orderNumber)
			}
			if tc.transactionNumber != "" && txs[0].TransactionNumber != tc.transactionNumber {
				t.Errorf("transaction number = %q, want %q",
					txs[0].TransactionNumber, tc.transactionNumber)
			}
			for _, d := range []struct{ name, got, want string }{
				{"date", txs[0].Date, tc.date},
				{"order date", txs[0].OrderDate, tc.orderDate},
				{"value date", txs[0].ValueDate, tc.valueDate},
				{"execution time", txs[0].ExecutionTime, tc.executionTime},
				{"deposit country", txs[0].DepositCountry, tc.depositCountry},
				{"trade type", txs[0].Type, tc.tradeType},
				{"security name", txs[0].SecurityName, tc.securityName},
			} {
				if d.want != "" && d.got != d.want {
					t.Errorf("%s = %q, want %q", d.name, d.got, d.want)
				}
			}
			for _, d := range []struct {
				name      string
				got, want *schema.Decimal
			}{
				{"gain/loss", txs[0].GainLoss, tc.gainLoss},
				{"withholding tax", txs[0].WithholdingTax, tc.withholdingTax},
			} {
				if d.want == nil {
					continue
				}
				if d.got == nil {
					t.Errorf("%s = nil, want %.2f", d.name, d.want.Float())
				} else if d.got.Float() != d.want.Float() {
					t.Errorf("%s = %.2f, want %.2f", d.name, d.got.Float(), d.want.Float())
				}
			}
		})
	}
}

// tradeHeader is the header block of a real flatex trade confirmation, in the
// order gxpdf yields it. The letter date ("Graz, …") precedes all three
// transaction dates, which is what made a first-date-wins scan pick it up.
const tradeHeader = "             Graz, 16.09.2025\n" +
	"Auftragsdatum      15.09.2025\n" +
	"Handelstag         12.09.2025\n" +
	"Ausführungszeit    00:00 Uhr\n" +
	"Valuta             17.09.2025\n" +
	"Auftragsnummer     700000011/1\n"

const tradeBody = "Nr.700000011/1     Kauf              L&G GOLD MINING ETF (IE00B3CNHG25/A0Q8HZ)\n" +
	"Ausgeführt    :        0,685401 St.     Kurswert      :              50,00 EUR\n" +
	"Kurs          :       72,950000 EUR     Provision     :               0,00 EUR\n"

// TestParseTradeUsesHandelstagNotLetterDate pins which of the four dates on a
// trade confirmation becomes Date. Handelstag dates the position change and is
// what Portfolio Performance imports; the letter date, Auftragsdatum and
// Valuta are all distinct here so a regression cannot pass by coincidence.
func TestParseTradeUsesHandelstagNotLetterDate(t *testing.T) {
	doc := &extractor.ExtractedDocument{
		Filename: "trade.pdf", Text: tradeHeader + tradeBody, DocumentType: "TRADE",
	}

	tx, err := parseTrade(doc)
	if err != nil {
		t.Fatalf("parseTrade failed: %v", err)
	}
	if tx.Date != "2025-09-12" {
		t.Errorf("Date = %q, want the Handelstag 2025-09-12", tx.Date)
	}
	if tx.OrderDate != "2025-09-15" {
		t.Errorf("OrderDate = %q, want the Auftragsdatum 2025-09-15", tx.OrderDate)
	}
	if tx.ValueDate != "2025-09-17" {
		t.Errorf("ValueDate = %q, want the Valuta 2025-09-17", tx.ValueDate)
	}
}

// A trade with no Handelstag falls back down the chain rather than reaching
// for the letter date, and fails outright when no transaction date exists.
// The Handelstag-wins case itself is pinned by
// TestParseTradeUsesHandelstagNotLetterDate.
func TestParseTradeDateFallback(t *testing.T) {
	cases := []struct {
		name   string
		header string
		want   string // "" means parseTrade must fail
	}{
		{
			"falls back to Auftragsdatum",
			"             Graz, 16.09.2025\nAuftragsdatum      15.09.2025\n", "2025-09-15",
		},
		{
			"falls back to Schlusstag",
			"             Graz, 16.09.2025\nSchlusstag: 11.09.2025, 16:00 Uhr\n", "2025-09-11",
		},
		{"letter date alone is not a trade date", "             Graz, 16.09.2025\n", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc := &extractor.ExtractedDocument{
				Filename: "trade.pdf", Text: tc.header + tradeBody, DocumentType: "TRADE",
			}
			tx, err := parseTrade(doc)
			if tc.want == "" {
				if err == nil {
					t.Fatalf("expected an error, got Date=%q", tx.Date)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseTrade failed: %v", err)
			}
			if tx.Date != tc.want {
				t.Errorf("Date = %q, want %q", tx.Date, tc.want)
			}
		})
	}
}

// TestParseTradeCosts covers the charge block: Provision, both Spesen lines,
// and the itemised Gebühren that make up Fremde Spesen. The layout is the one
// from trade_sample_2 — Tradinggebühr + Regulierung sum to Fremde Spesen,
// which must not then be added on top of it.
func TestParseTradeCosts(t *testing.T) {
	text := tradeHeader +
		"Nr.800000022/1     Kauf       GLOBAL X COPPER MINERS ET (IE0003Z9E2Y3/A3C7FZ)\n" +
		"Ausgeführt: 35 St.Kurswert: 2.034,20 EUR\n" +
		"Kurs: 58,120000 EURProvision: 1,50 EUR\n" +
		"Devisenkurs: 1,000000 Eigene Spesen: 0,90 EUR\n" +
		"*Fremde Spesen: 3,00 EUR\n" +
		"Gewinn/Verlust: 0,00 EUR**Einbeh. KESt: 0,25 EUR\n" +
		"Endbetrag: -2.039,85 EUR\n" +
		"* Enthalten sind folgende Gebühren: Courtage: 0,00 EUR\n" +
		"Tradinggebühr: 0,50 EUR\n" +
		"Regulierung: 2,50 EUR\n" +
		"Schlussnoten: 0,00 EUR\n" +
		"LS-Umlegung: 0,00 EUR\n" +
		"Finanztransaktionssteuer: 0,00 EUR\n" +
		"Sonstige: 0,00 EUR\n"
	doc := &extractor.ExtractedDocument{Filename: "trade.pdf", Text: text, DocumentType: "TRADE"}

	tx, err := parseTrade(doc)
	if err != nil {
		t.Fatalf("parseTrade failed: %v", err)
	}
	if tx.Costs == nil {
		t.Fatal("expected a cost block, got nil")
	}
	if tx.Costs.ForeignExpensesBreakdown == nil {
		t.Fatal("expected an itemised Gebühren breakdown, got nil")
	}
	checks := []struct {
		name string
		got  float64
		want float64
	}{
		{"Provision", tx.Costs.Provision.Float(), 1.50},
		{"Eigene Spesen", tx.Costs.OwnExpenses.Float(), 0.90},
		{"Fremde Spesen", tx.Costs.ForeignExpenses.Float(), 3.00},
		{"total", tx.Costs.Total.Float(), 5.40},
		{"Tradinggebühr", tx.Costs.ForeignExpensesBreakdown.TradingFee.Float(), 0.50},
		{"Regulierung", tx.Costs.ForeignExpensesBreakdown.Settlement.Float(), 2.50},
		{"Courtage", tx.Costs.ForeignExpensesBreakdown.Courtage.Float(), 0.00},
		{"Einbeh. KESt", schema.Amount(tx.WithholdingTax), 0.25},
		{"Endbetrag", schema.Amount(tx.NetAmount), -2039.85},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %v, want %v", c.name, c.got, c.want)
		}
	}
	// The breakdown itemises Fremde Spesen; double-counting it would give 8.40.
	sum := tx.Costs.ForeignExpensesBreakdown.Courtage.Float() + tx.Costs.ForeignExpensesBreakdown.TradingFee.Float() + tx.Costs.ForeignExpensesBreakdown.Settlement.Float() +
		tx.Costs.ForeignExpensesBreakdown.ClosingNotes.Float() + tx.Costs.ForeignExpensesBreakdown.LSAllocation.Float() +
		tx.Costs.ForeignExpensesBreakdown.FinancialTransactionTax.Float() + tx.Costs.ForeignExpensesBreakdown.Other.Float()
	if sum != tx.Costs.ForeignExpenses.Float() {
		t.Errorf("Gebühren sum to %v, want Fremde Spesen %v", sum, tx.Costs.ForeignExpenses.Float())
	}
}

// A document with no Provision line at all must leave Costs nil, so that
// "flatex charged nothing" stays distinguishable from "this document type has
// no charge block".
func TestParseCostsAbsentBlockIsNil(t *testing.T) {
	if c := extractCosts("Ertragsmitteilung\nEndbetrag : 22,43 EUR\n"); c != nil {
		t.Errorf("expected nil cost block, got %+v", c)
	}
	c := extractCosts("Provision     :               0,00 EUR\n")
	if c == nil {
		t.Fatal("expected a cost block for a document with a Provision line")
	}
	if c.Total.Float() != 0 {
		t.Errorf("total = %v, want 0", c.Total)
	}
	if c.ForeignExpensesBreakdown != nil {
		t.Errorf("expected no Gebühren breakdown without its heading, got %+v", c.ForeignExpensesBreakdown)
	}
}

// eurField and dateField must not cross a line break: a label whose value
// column is blank has to yield nothing rather than capture the next line.
func TestFieldHelpersStayOnTheirLine(t *testing.T) {
	if got, err := eurField("Provision     :\nEndbetrag : 50,00 EUR\n", `Provision`); err == nil {
		t.Errorf("eurField reached across the line break and returned %v", got)
	}
	if got := dateField("Handelstag\nValuta 17.09.2025\n", `Handelstag`); got != "" {
		t.Errorf("dateField reached across the line break and returned %q", got)
	}
}
