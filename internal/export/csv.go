package export

import (
	"encoding/csv"
	"io"

	"github.com/welworx/flatex-pdf-cli/internal/schema"
)

// csvHeader lists every schema.Transaction field, in struct-declaration
// order, as the generic CSV export's column headers.
var csvHeader = []string{
	"source", "order_number", "transaction_number", "document_type", "isin", "wkn",
	"security_name", "date", "order_date", "value_date", "execution_time", "type", "quantity", "price",
	"gross_amount", "gross_currency",
	"provision", "own_expenses", "foreign_expenses", "total_costs",
	"fee_courtage", "fee_trading", "fee_settlement", "fee_closing_notes",
	"fee_ls_allocation", "fee_financial_transaction_tax", "fee_other",
	"withholding_tax", "withholding_tax_currency", "gain_loss", "exchange_rate",
	"net_amount", "net_currency",
	"custody_type", "depositary", "deposit_country", "execution_venue",
	"limit", "valid_until",
	"distribution_per_share", "distribution_currency", "ex_date",
	"interest_rate", "period_from", "period_to",
	"reinvestment_per_share", "reinvestment_currency", "accrual_date",
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
			t.SecurityName, t.Date, t.OrderDate, t.ValueDate, t.ExecutionTime, t.Type,
			formatFloatPtr(t.Quantity), formatFloatPtr(t.Price),
			formatFloatPtr(t.GrossAmount), t.GrossCurrency,
		}
		row = append(row, costColumns(t.Costs)...)
		row = append(row,
			formatFloatPtr(t.WithholdingTax), t.WithholdingTaxCurrency,
			formatFloatPtr(t.GainLoss), formatFloatPtr(t.ExchangeRate),
			formatFloatPtr(t.NetAmount), t.NetCurrency,
			t.CustodyType, t.Depositary, t.DepositCountry, t.ExecutionVenue,
			formatFloatPtr(t.Limit), t.ValidUntil,
			formatFloatPtr(t.DistributionPerShare), t.DistributionCurrency, t.ExDate,
			formatFloatPtr(t.InterestRate), t.PeriodFrom, t.PeriodTo,
			formatFloatPtr(t.ReinvestmentPerShare), t.ReinvestmentCurrency, t.AccrualDate,
		)
		if err := cw.Write(row); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

// costColumns renders the 11 cost columns, all blank when the document had no
// charge block and all zero-filled when it had one but itemised no Gebühren.
func costColumns(c *schema.Costs) []string {
	if c == nil {
		return make([]string, 11)
	}
	f := c.ForeignExpensesBreakdown
	if f == nil {
		// Zero-filled as currency, so these cells read 0.00 like every other
		// charge column rather than the bare 0 a zero-value Decimal renders.
		z := schema.Computed(0)
		f = &schema.FeeBreakdown{
			Courtage: z, TradingFee: z, Settlement: z, ClosingNotes: z,
			LSAllocation: z, FinancialTransactionTax: z, Other: z,
		}
	}
	return []string{
		formatFloat(c.Provision), formatFloat(c.OwnExpenses),
		formatFloat(c.ForeignExpenses), formatFloat(c.Total),
		formatFloat(f.Courtage), formatFloat(f.TradingFee), formatFloat(f.Settlement),
		formatFloat(f.ClosingNotes), formatFloat(f.LSAllocation),
		formatFloat(f.FinancialTransactionTax), formatFloat(f.Other),
	}
}

// formatFloat renders an amount at the precision the document printed it with,
// so the CSV carries the same digits as the JSON.
func formatFloat(d schema.Decimal) string {
	return d.String()
}

// formatFloatPtr renders an optional amount, leaving the cell empty when the
// document did not state one. Same convention costColumns uses for an absent
// cost block, and it keeps a genuine 0 distinct from a missing value.
func formatFloatPtr(p *schema.Decimal) string {
	if p == nil {
		return ""
	}
	return p.String()
}
