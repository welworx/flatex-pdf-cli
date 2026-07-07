package export

import (
	"encoding/csv"
	"fmt"
	"io"

	"github.com/welworx/flatex-pdf-cli/internal/schema"
)

// Column names match Portfolio Performance's documented CSV import fields
// (https://help.portfolio-performance.info/en/reference/file/import/csv-import/).
// German headers/labels are sourced from PP's own German locale resource
// files (messages_de.properties, labels_de.properties) — PP's CSV column
// auto-recognition is locale-sensitive with no English fallback, so a
// German-locale PP install needs German headers and German Type values to
// auto-map columns at all.
var portfolioHeader = map[string][]string{
	"en": {"Date", "Type", "Value", "Shares", "ISIN", "WKN", "Security Name", "Fees", "Taxes", "Currency Gross Amount", "Exchange Rate", "Note"},
	"de": {"Datum", "Typ", "Wert", "Stück", "ISIN", "WKN", "Wertpapiername", "Gebühren", "Steuern", "Währung Bruttobetrag", "Wechselkurs", "Notiz"},
}

// ValidLang reports whether lang is a supported -lang value for the pp
// export functions ("en" or "de").
func ValidLang(lang string) bool {
	_, ok := portfolioHeader[lang]
	return ok
}

var accountHeader = map[string][]string{
	"en": {"Date", "Type", "Value", "ISIN", "WKN", "Security Name", "Taxes", "Fees", "Note"},
	"de": {"Datum", "Typ", "Wert", "ISIN", "WKN", "Wertpapiername", "Steuern", "Gebühren", "Notiz"},
}

var tradeTypeLabel = map[string]map[string]string{
	"en": {"BUY": "Buy", "SELL": "Sell"},
	"de": {"BUY": "Kauf", "SELL": "Verkauf"},
}

var accountTypeLabel = map[string]map[string]string{
	"en": {"DIVIDEND": "Dividend", "INTEREST": "Interest", "TAXES": "Taxes"},
	"de": {"DIVIDEND": "Dividende", "INTEREST": "Zinsen", "TAXES": "Steuern"},
}

// WritePortfolioTransactions writes the buy/sell CSV for PP's "Portfolio
// Transactions" import (TRADE, CRYPTO, SAVINGSPLAN document types). Pending
// ORDER confirmations are skipped — they have no executed Value/Shares yet.
// lang selects the header row and Type vocabulary: "en" or "de".
func WritePortfolioTransactions(w io.Writer, txns []*schema.Transaction, lang string) error {
	header, ok := portfolioHeader[lang]
	if !ok {
		return fmt.Errorf("unknown lang %q (want en or de)", lang)
	}
	cw := csv.NewWriter(w)
	cw.Comma = csvDelimiter(lang)
	if err := cw.Write(header); err != nil {
		return err
	}
	for _, t := range txns {
		switch t.DocumentType {
		case "TRADE", "CRYPTO", "SAVINGSPLAN":
		default:
			continue
		}
		ppType, err := ppTradeType(lang, t.Type)
		if err != nil {
			return fmt.Errorf("%s %s: %w", t.DocumentType, t.Date, err)
		}
		row := []string{
			t.Date,
			ppType,
			formatFloat(portfolioValue(t)),
			formatFloat(t.Quantity),
			t.ISIN,
			t.WKN,
			t.SecurityName,
			formatFloat(t.Provision),
			formatFloat(t.WithholdingTax),
			t.PriceCurrency,
			formatFloat(t.ExchangeRate),
			note(t),
		}
		if err := cw.Write(row); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

// portfolioValue computes PP's "Value" column: the total cash movement of a
// buy/sell. CRYPTO already carries flatex's own computed settlement amount
// (FinalAmount); TRADE/SAVINGSPLAN only carry GrossValue, so fees are added
// back for a buy (more cash out) and subtracted for a sell (less cash in).
func portfolioValue(t *schema.Transaction) float64 {
	if t.FinalAmount != 0 {
		return t.FinalAmount
	}
	if t.Type == "SELL" {
		return t.GrossValue - t.Provision
	}
	return t.GrossValue + t.Provision
}

func ppTradeType(lang, tradeType string) (string, error) {
	labels, ok := tradeTypeLabel[lang]
	if !ok {
		return "", fmt.Errorf("unknown lang %q (want en or de)", lang)
	}
	label, ok := labels[tradeType]
	if !ok {
		return "", fmt.Errorf("unknown trade type %q", tradeType)
	}
	return label, nil
}

// WriteAccountTransactions writes the cash-account CSV for PP's "Account
// Transactions" import (DIVIDEND, INTEREST, ACCUMULATING document types).
// ACCUMULATING entries with no withheld tax are skipped — flatex's
// Vorabpauschale notice is a phantom accrual with no real cash movement
// unless tax was actually withheld. lang selects the header row and Type
// vocabulary: "en" or "de".
func WriteAccountTransactions(w io.Writer, txns []*schema.Transaction, lang string) error {
	header, ok := accountHeader[lang]
	if !ok {
		return fmt.Errorf("unknown lang %q (want en or de)", lang)
	}
	labels := accountTypeLabel[lang]
	cw := csv.NewWriter(w)
	cw.Comma = csvDelimiter(lang)
	if err := cw.Write(header); err != nil {
		return err
	}
	for _, t := range txns {
		var ppType string
		var value float64
		switch t.DocumentType {
		case "DIVIDEND":
			ppType, value = labels["DIVIDEND"], t.NetAmount
		case "INTEREST":
			ppType, value = labels["INTEREST"], t.NetAmount
		case "ACCUMULATING":
			if t.WithholdingTax == 0 {
				continue
			}
			ppType, value = labels["TAXES"], t.WithholdingTax
		default:
			continue
		}
		row := []string{
			t.Date, ppType, formatFloat(value), t.ISIN, t.WKN, t.SecurityName,
			formatFloat(t.WithholdingTax), "0", note(t),
		}
		if err := cw.Write(row); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

// csvDelimiter returns the field separator conventional for lang: German
// locale CSV uses semicolon, since comma is the German decimal separator —
// PP's own CSV import wizard defaults its delimiter picker to semicolon for
// this reason, so matching it removes a manual step for German-locale users.
func csvDelimiter(lang string) rune {
	if lang == "de" {
		return ';'
	}
	return ','
}

func note(t *schema.Transaction) string {
	if t.OrderNumber != "" {
		return t.OrderNumber
	}
	return t.TransactionNumber
}
