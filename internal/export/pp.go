package export

import (
	"encoding/csv"
	"fmt"
	"io"

	"github.com/welworx/flatex-pdf-cli/internal/schema"
)

// Column names match Portfolio Performance's documented CSV import fields
// (https://help.portfolio-performance.info/en/reference/file/import/csv-import/).
// PP's CSV import is column-mapping based, not fixed-header — using its
// documented names just helps the import wizard auto-recognize columns.
var portfolioHeader = []string{
	"Date", "Type", "Value", "Shares", "ISIN", "WKN", "Security Name",
	"Fees", "Taxes", "Currency Gross Amount", "Exchange Rate", "Note",
}

var accountHeader = []string{
	"Date", "Type", "Value", "ISIN", "WKN", "Security Name", "Taxes", "Fees", "Note",
}

// WritePortfolioTransactions writes the buy/sell CSV for PP's "Portfolio
// Transactions" import (TRADE, CRYPTO, SAVINGSPLAN document types). Pending
// ORDER confirmations are skipped — they have no executed Value/Shares yet.
func WritePortfolioTransactions(w io.Writer, txns []*schema.Transaction) error {
	cw := csv.NewWriter(w)
	if err := cw.Write(portfolioHeader); err != nil {
		return err
	}
	for _, t := range txns {
		switch t.DocumentType {
		case "TRADE", "CRYPTO", "SAVINGSPLAN":
		default:
			continue
		}
		ppType, err := ppTradeType(t.Type)
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

func ppTradeType(tradeType string) (string, error) {
	switch tradeType {
	case "BUY":
		return "Buy", nil
	case "SELL":
		return "Sell", nil
	default:
		return "", fmt.Errorf("unknown trade type %q", tradeType)
	}
}

// WriteAccountTransactions writes the cash-account CSV for PP's "Account
// Transactions" import (DIVIDEND, INTEREST, ACCUMULATING document types).
// ACCUMULATING entries with no withheld tax are skipped — flatex's
// Vorabpauschale notice is a phantom accrual with no real cash movement
// unless tax was actually withheld.
func WriteAccountTransactions(w io.Writer, txns []*schema.Transaction) error {
	cw := csv.NewWriter(w)
	if err := cw.Write(accountHeader); err != nil {
		return err
	}
	for _, t := range txns {
		var ppType string
		var value float64
		switch t.DocumentType {
		case "DIVIDEND":
			ppType, value = "Dividend", t.NetAmount
		case "INTEREST":
			ppType, value = "Interest", t.NetAmount
		case "ACCUMULATING":
			if t.WithholdingTax == 0 {
				continue
			}
			ppType, value = "Taxes", t.WithholdingTax
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

func note(t *schema.Transaction) string {
	if t.OrderNumber != "" {
		return t.OrderNumber
	}
	return t.TransactionNumber
}
