package export

import (
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"

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
			formatAmount(portfolioValue(t), lang),
			formatAmount(schema.Amount(t.Quantity), lang),
			t.ISIN,
			t.WKN,
			t.SecurityName,
			formatAmount(ppFees(t), lang),
			formatAmount(schema.Amount(t.WithholdingTax), lang),
			ppCurrency(t),
			formatAmount(ppExchangeRate(t), lang),
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
// buy/sell. TRADE and CRYPTO carry flatex's own computed settlement amount
// (Endbetrag/NetAmount), which is signed by cash direction — negative for a
// buy. PP wants the magnitude, with Buy/Sell carried by the Type column, so
// the sign is dropped here. SAVINGSPLAN rows have no Endbetrag, so fees are
// added back for a buy (more cash out) and subtracted for a sell (less cash
// in).
func portfolioValue(t *schema.Transaction) float64 {
	// Falls back only when the document states no Endbetrag at all. A stated
	// 0,00 is the settlement amount, not a missing one.
	if t.NetAmount != nil {
		return math.Abs(t.NetAmount.Float())
	}
	grossValue := schema.Amount(t.GrossAmount)
	if t.Type == "SELL" {
		return grossValue - t.TotalCosts()
	}
	return grossValue + t.TotalCosts()
}

// ppFees fills PP's Gebühren column. Most documents itemise their charges and
// the column is just their total.
//
// A savings-plan row does not: it prints Stücke, Kurs and Betrag, and the fee
// flatex withheld shows up only as the gap between the cash that moved and
// what the shares are worth. The extracted transaction therefore carries no
// charge for it — a figure no line of the document states is not something an
// extractor should report. PP is a different audience: it is accounting for
// what the purchase cost, and dropping the fee there would understate it. So
// the gap is derived here, at the point it is needed, and rounded to the cent
// because that is the unit a fee is actually charged in — the further places
// are an artefact of the quantity being printed to six decimals.
func ppFees(t *schema.Transaction) float64 {
	if t.Costs != nil || t.DocumentType != "SAVINGSPLAN" {
		return t.TotalCosts()
	}
	shareValue := schema.Amount(t.Quantity) * schema.Amount(t.Price)
	gap := math.Abs(schema.Amount(t.NetAmount)) - shareValue
	if t.Type == "SELL" {
		gap = -gap
	}
	if gap < 0 {
		return 0
	}
	return math.Round(gap*100) / 100
}

// ppCurrency fills PP's "Currency Gross Amount" column, falling back to the
// settlement currency when the document states no gross amount to qualify —
// a savings-plan row prints EUR against its Kurs and Betrag but has no
// Kurswert line, so GrossCurrency is empty and NetCurrency is what it means.
func ppCurrency(t *schema.Transaction) string {
	if t.GrossCurrency != "" {
		return t.GrossCurrency
	}
	return t.NetCurrency
}

// ppExchangeRate fills PP's Wechselkurs column. A document that settles in EUR
// prints no Devisenkurs, so the extracted transaction carries none — the JSON
// reports what the statement says and says nothing where it is silent. PP does
// need a number here, and a blank or zero rate breaks its valuation, so the
// implied 1 is supplied at the point it is actually required.
func ppExchangeRate(t *schema.Transaction) float64 {
	if t.ExchangeRate == nil {
		return 1
	}
	return t.ExchangeRate.Float()
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
			ppType, value = labels["DIVIDEND"], schema.Amount(t.NetAmount)
		case "INTEREST":
			ppType, value = labels["INTEREST"], schema.Amount(t.NetAmount)
		case "ACCUMULATING":
			if schema.Amount(t.WithholdingTax) == 0 {
				continue
			}
			ppType, value = labels["TAXES"], schema.Amount(t.WithholdingTax)
		default:
			continue
		}
		row := []string{
			t.Date, ppType, formatAmount(value, lang), t.ISIN, t.WKN, t.SecurityName,
			formatAmount(schema.Amount(t.WithholdingTax), lang), "0", note(t),
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

// formatAmount renders f for lang. PP's per-column numeric-format picker
// defaults to comma-decimal exactly when the OS locale's own decimal
// separator is comma — the same condition that makes German headers
// necessary — so German output swaps the period from formatFloat for a
// comma; PP's parser accepts a bare digit string with no thousands
// separator, so no grouping is added.
func formatAmount(f float64, lang string) string {
	s := ppFormat(f)
	if lang == "de" {
		s = strings.Replace(s, ".", ",", 1)
	}
	return s
}

// ppFormat renders a PP column value in shortest round-trip form. The CSV and
// JSON exports carry each amount at the precision its document printed it
// with, but PP is not that audience: it re-parses these columns and several of
// them are derived here rather than read (see portfolioValue), so there is no
// printed precision to honour. Padding them to the cent would also round a
// fractional share count — 1.478695 shares is not 1.48.
func ppFormat(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

func note(t *schema.Transaction) string {
	if t.OrderNumber != "" {
		return t.OrderNumber
	}
	return t.TransactionNumber
}
