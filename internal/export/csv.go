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
	"security_name", "date", "order_date", "type", "quantity", "price", "price_currency",
	"gross_value", "provision", "own_expenses", "foreign_expenses", "unitemised", "total_costs",
	"fee_courtage", "fee_trading", "fee_settlement", "fee_closing_notes",
	"fee_ls_allocation", "fee_financial_transaction_tax", "fee_other",
	"withholding_tax", "gain_loss", "exchange_rate",
	"final_amount", "final_currency", "custody_type", "depositary", "deposit_country",
	"execution_venue",
	"limit", "valid_until", "distribution_per_share", "distribution_currency",
	"gross_amount", "gross_currency", "withholding_tax_currency", "net_amount",
	"net_currency", "ex_date", "value_date", "interest_rate", "period_from",
	"period_to", "reinvestment_per_share", "reinvestment_currency", "accrual_date",
}

// WriteCSV writes one row per transaction, dumping every schema.Transaction
// field as a column. Numeric zero is written as "0", not blank — a flat CSV
// has no way to distinguish "zero" from "not applicable to this doc type".
// The cost columns are the exception: they are left blank when the document
// carried no charge block, so a genuine 0,00 EUR fee stays readable as one.
func WriteCSV(w io.Writer, txns []*schema.Transaction) error {
	cw := csv.NewWriter(w)
	if err := cw.Write(csvHeader); err != nil {
		return err
	}
	for _, t := range txns {
		row := []string{
			t.Source, t.OrderNumber, t.TransactionNumber, t.DocumentType, t.ISIN, t.WKN,
			t.SecurityName, t.Date, t.OrderDate, t.Type, formatFloat(t.Quantity), formatFloat(t.Price), t.PriceCurrency,
			formatFloat(t.GrossValue),
		}
		row = append(row, costColumns(t.Costs)...)
		row = append(row,
			formatFloatPtr(t.WithholdingTax), formatFloatPtr(t.GainLoss), formatFloat(t.ExchangeRate),
			formatFloat(t.FinalAmount), t.FinalCurrency, t.CustodyType, t.Depositary, t.DepositCountry, t.ExecutionVenue,
			formatFloat(t.Limit), t.ValidUntil, formatFloat(t.DistributionPerShare), t.DistributionCurrency,
			formatFloat(t.GrossAmount), t.GrossCurrency, t.WithholdingTaxCurrency, formatFloat(t.NetAmount),
			t.NetCurrency, t.ExDate, t.ValueDate, formatFloat(t.InterestRate), t.PeriodFrom,
			t.PeriodTo, formatFloat(t.ReinvestmentPerShare), t.ReinvestmentCurrency, t.AccrualDate,
		)
		if err := cw.Write(row); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

// costColumns renders the 12 cost columns, all blank when the document had no
// charge block and all zero-filled when it had one but itemised no Gebühren.
func costColumns(c *schema.Costs) []string {
	if c == nil {
		return make([]string, 12)
	}
	f := c.Fees
	if f == nil {
		f = &schema.Fees{}
	}
	return []string{
		formatFloat(c.Provision), formatFloat(c.OwnExpenses),
		formatFloat(c.ForeignExpenses), formatFloat(c.Unitemised), formatFloat(c.Total),
		formatFloat(f.Courtage), formatFloat(f.TradingFee), formatFloat(f.Settlement),
		formatFloat(f.ClosingNotes), formatFloat(f.LSAllocation),
		formatFloat(f.FinancialTransactionTax), formatFloat(f.Other),
	}
}

func formatFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

// formatFloatPtr renders an optional amount, leaving the cell empty when the
// document did not state one. Same convention costColumns uses for an absent
// cost block, and it keeps a genuine 0 distinct from a missing value.
func formatFloatPtr(p *float64) string {
	if p == nil {
		return ""
	}
	return formatFloat(*p)
}
