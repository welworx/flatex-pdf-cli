package schema

// DocumentMetadata holds metadata about the parsed document.
type DocumentMetadata struct {
	DepotNumber   string `json:"depot_number"`
	DepotHolder   string `json:"depot_holder"`
	AccountNumber string `json:"account_number,omitempty"`
}

// Costs holds a settlement's charge block. It is emitted whenever the
// document carries that block, and its own fields are written out even when
// zero: an absent "costs" object means the document had no cost section,
// a 0.00 field means flatex charged nothing. Those are not the same thing,
// and collapsing them behind omitempty made a genuine "Provision: 0,00 EUR"
// look like a parse failure.
type Costs struct {
	Provision       Decimal `json:"provision"`        // Provision
	OwnExpenses     Decimal `json:"own_expenses"`     // Eigene Spesen
	ForeignExpenses Decimal `json:"foreign_expenses"` // Fremde Spesen

	// Total is the exact sum of the three charges above. It is the one figure
	// in this package that no line of the document carries, and it is kept
	// because it asserts nothing the printed lines do not already say.
	//
	// There was a fourth charge here, "unitemised", holding a savings plan's
	// fee recovered as the gap between what the row settled and what its
	// shares were worth. It has been removed: the Sammelabrechnung prints no
	// such figure, so reporting one meant this extractor inventing a number,
	// and reconstructing it from a six-decimal quantity gave it digits
	// (1.4999832 for a fee of 1,50) that were noise dressed as precision. The
	// gap is still computed — it is a good check that the columns landed where
	// they should — but it is a check now, not output.
	Total Decimal `json:"total"`

	// ForeignExpensesBreakdown itemises ForeignExpenses and nothing else. The
	// document marks the relationship with a footnote: the charge line reads
	// "* Fremde Spesen" and the starred note below it lists the components
	// ("* Enthalten sind folgende Gebühren"). They therefore sum to
	// ForeignExpenses and are already inside Total — adding them on top
	// double-counts. It was called "fees" before, which said nothing about
	// which charge it belonged to and invited exactly that mistake.
	ForeignExpensesBreakdown *FeeBreakdown `json:"foreign_expenses_breakdown,omitempty"`
}

// FeeBreakdown is the itemised breakdown of Costs.ForeignExpenses.
type FeeBreakdown struct {
	Courtage                Decimal `json:"courtage"`
	TradingFee              Decimal `json:"trading_fee"`               // Tradinggebühr
	Settlement              Decimal `json:"settlement"`                // Regulierung
	ClosingNotes            Decimal `json:"closing_notes"`             // Schlussnoten
	LSAllocation            Decimal `json:"ls_allocation"`             // LS-Umlegung
	FinancialTransactionTax Decimal `json:"financial_transaction_tax"` // Finanztransaktionssteuer
	Other                   Decimal `json:"other"`                     // Sonstige
}

// Transaction represents a single transaction (trade, dividend, interest, or thesaurierung).
type Transaction struct {
	// Common fields
	Source            string `json:"source,omitempty"`
	OrderNumber       string `json:"order_number,omitempty"`       // Auftragsnummer
	TransactionNumber string `json:"transaction_number,omitempty"` // Transaktion-Nr.
	DocumentType      string `json:"document_type"`
	ISIN              string `json:"isin"`
	WKN               string `json:"wkn,omitempty"`
	SecurityName      string `json:"security_name,omitempty"` // Bezeichnung (e.g. crypto without ISIN)

	// One field per date the document prints, each present only when it does.
	//
	// There used to be a single "date" alongside these, holding whichever of
	// them mattered for the document type — the trade date on a trade, the
	// value date on a dividend. It read as an extracted field and was not one:
	// its meaning changed underneath a consumer depending on what it was
	// looking at. Picking a primary date is a decision for whoever needs one,
	// so the Portfolio Performance export makes it (see ppDate) and the
	// extracted data no longer pretends to.
	OrderDate string `json:"order_date,omitempty"` // Auftragsdatum — when the order was placed

	// TradeDate is when the trade executed: Handelstag on a confirmation,
	// Schlusstag on crypto, Ausführungsdatum on older layouts. "Trade date" is
	// the standard term for this — the counterpart to the value date, and what
	// T+2 counts from — and it is what all three German labels denote.
	TradeDate string `json:"trade_date,omitempty"`

	// BookingDate is a savings plan's Buchtag. It is deliberately not folded
	// into TradeDate: Buchtag is when the row was booked, and a
	// Sammelabrechnung states no Handelstag at all, so those rows carry no
	// confirmed execution day to report.
	BookingDate string `json:"booking_date,omitempty"`

	ValueDate string `json:"value_date,omitempty"` // Valuta — when the cash settled

	// ExecutionTime is the Ausführungszeit, as "HH:MM". The document prints it
	// as a bare local time with no date and no zone ("13:56 Uhr"), so it is
	// carried as one rather than folded into Date: the day it belongs to is
	// Date, and inventing a zone to build a timestamp would state more than
	// the statement does.
	ExecutionTime string `json:"execution_time,omitempty"`

	// Every amount and quantity below is a pointer, so a document printing
	// "0,00" stays distinguishable from one that prints no such line at all.
	// As plain float64 the two collapsed: omitempty dropped a genuine zero and
	// a consumer read it back as "field absent". A sale closed exactly at
	// cost, a 0% interest period and a fee-free settlement all really do state
	// a zero. Same nil-means-absent convention as Costs below; read one with
	// Amount where absent should behave as 0.
	//
	// They are Decimal rather than float64 so each keeps the precision its
	// line was printed with — see the type's own comment.

	// Position fields — what moved, and at what unit price.
	Type     string   `json:"type,omitempty"` // BUY or SELL
	Quantity *Decimal `json:"quantity,omitempty"`
	Price    *Decimal `json:"price,omitempty"` // Kurs, always in EUR

	// Settlement amounts. Every document that settles states a gross figure,
	// deductions, and the Endbetrag that actually moved, so these are one set
	// of fields shared by all of them rather than one set per document type.
	//
	// They used to be two: a trade wrote gross_value/final_amount/
	// price_currency and a dividend wrote gross_amount/net_amount/
	// gross_currency, for the same three quantities read off the same labels
	// — Endbetrag became "final_amount" on one document and "net_amount" on
	// the other. A consumer that handled trades and missed the dividend
	// spelling read a silent zero. price_currency was the worst of them: Kurs
	// is only ever parsed in EUR, and the field actually carried the currency
	// of Kurswert, i.e. of the gross amount, which is also how the Portfolio
	// Performance export has always used it.
	//
	// NetAmount is signed by cash direction — negative when money left the
	// account. GrossAmount, WithholdingTax and Costs are unsigned magnitudes;
	// which way they apply follows from Type.
	GrossAmount            *Decimal `json:"gross_amount,omitempty"`   // Kurswert / Bruttoausschüttung
	GrossCurrency          string   `json:"gross_currency,omitempty"` // currency of GrossAmount
	WithholdingTax         *Decimal `json:"withholding_tax,omitempty"`
	WithholdingTaxCurrency string   `json:"withholding_tax_currency,omitempty"`
	GainLoss               *Decimal `json:"gain_loss,omitempty"`
	// ExchangeRate is 1 when the document prints no Devisenkurs. That default
	// is a computed value, so it carries two places (1.00) where a stated rate
	// carries the six the document prints (1.000000).
	ExchangeRate *Decimal `json:"exchange_rate,omitempty"`
	NetAmount    *Decimal `json:"net_amount,omitempty"`   // Endbetrag
	NetCurrency  string   `json:"net_currency,omitempty"` // currency of NetAmount
	Costs        *Costs   `json:"costs,omitempty"`

	// Custody and execution
	CustodyType    string `json:"custody_type,omitempty"`
	Depositary     string `json:"depositary,omitempty"`
	DepositCountry string `json:"deposit_country,omitempty"` // Lagerland, as an ISO 3166-1 alpha-2 code
	ExecutionVenue string `json:"execution_venue,omitempty"` // Ausf.platz/-art

	// ORDER fields (Sammelauftragsbestätigung — pending orders)
	Limit      *Decimal `json:"limit,omitempty"`       // Limit price
	ValidUntil string   `json:"valid_until,omitempty"` // Gültig bis

	// DIVIDEND fields
	DistributionPerShare *Decimal `json:"distribution_per_share,omitempty"`
	DistributionCurrency string   `json:"distribution_currency,omitempty"`
	ExDate               string   `json:"ex_date,omitempty"`

	// INTEREST fields
	InterestRate *Decimal `json:"interest_rate,omitempty"`
	PeriodFrom   string   `json:"period_from,omitempty"`
	PeriodTo     string   `json:"period_to,omitempty"`

	// ACCUMULATING fields
	ReinvestmentPerShare *Decimal `json:"reinvestment_per_share,omitempty"`
	ReinvestmentCurrency string   `json:"reinvestment_currency,omitempty"`
	AccrualDate          string   `json:"accrual_date,omitempty"`
}

// Amount returns the value p points at, or 0 when p is nil. Use it wherever an
// absent optional amount should behave as zero (arithmetic, formatting) while
// the field itself stays nil-means-absent for callers that care.
func Amount(p *Decimal) float64 {
	if p == nil {
		return 0
	}
	return p.Float()
}

// TotalCosts returns the transaction's total charge, or 0 when the document
// carried no cost block.
func (t *Transaction) TotalCosts() float64 {
	if t.Costs == nil {
		return 0
	}
	return t.Costs.Total.Float()
}
