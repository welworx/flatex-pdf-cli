package export

import (
	"encoding/csv"
	"io"
	"strconv"

	"github.com/welworx/flatex-pdf-cli/internal/schema"
)

// csvHeader lists every schema.Transaction field, in struct-declaration
// order, as the generic CSV export's column headers.
var csvHeader = []string{
	"source", "order_number", "transaction_number", "document_type", "isin", "wkn",
	"security_name", "date", "type", "quantity", "price", "price_currency",
	"gross_value", "provision", "withholding_tax", "gain_loss", "exchange_rate",
	"final_amount", "final_currency", "custody_type", "depositary", "execution_venue",
	"limit", "valid_until", "distribution_per_share", "distribution_currency",
	"gross_amount", "gross_currency", "withholding_tax_currency", "net_amount",
	"net_currency", "ex_date", "value_date", "interest_rate", "period_from",
	"period_to", "reinvestment_per_share", "reinvestment_currency", "accrual_date",
}

// WriteCSV writes one row per transaction, dumping every schema.Transaction
// field as a column. Numeric zero is written as "0", not blank — a flat CSV
// has no way to distinguish "zero" from "not applicable to this doc type".
func WriteCSV(w io.Writer, txns []*schema.Transaction) error {
	cw := csv.NewWriter(w)
	if err := cw.Write(csvHeader); err != nil {
		return err
	}
	for _, t := range txns {
		row := []string{
			t.Source, t.OrderNumber, t.TransactionNumber, t.DocumentType, t.ISIN, t.WKN,
			t.SecurityName, t.Date, t.Type, formatFloat(t.Quantity), formatFloat(t.Price), t.PriceCurrency,
			formatFloat(t.GrossValue), formatFloat(t.Provision), formatFloat(t.WithholdingTax), formatFloat(t.GainLoss), formatFloat(t.ExchangeRate),
			formatFloat(t.FinalAmount), t.FinalCurrency, t.CustodyType, t.Depositary, t.ExecutionVenue,
			formatFloat(t.Limit), t.ValidUntil, formatFloat(t.DistributionPerShare), t.DistributionCurrency,
			formatFloat(t.GrossAmount), t.GrossCurrency, t.WithholdingTaxCurrency, formatFloat(t.NetAmount),
			t.NetCurrency, t.ExDate, t.ValueDate, formatFloat(t.InterestRate), t.PeriodFrom,
			t.PeriodTo, formatFloat(t.ReinvestmentPerShare), t.ReinvestmentCurrency, t.AccrualDate,
		}
		if err := cw.Write(row); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

func formatFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}
