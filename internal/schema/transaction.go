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

	// Unitemised is a charge the document does not print as a line item,
	// recovered as the gap between the amount settled and the value of the
	// shares. Savings-plan settlements are the known case: a row buys
	// (Betrag - charge) / Kurs shares and never names the charge. It is kept
	// separate from the fields above because those carry a label the document
	// actually printed, and this one does not: it is computed, not read.
	// A pointer, unlike the printed charges above, because omitempty does not
	// apply to a struct: only nil keeps it out of a settlement that has none.
	Unitemised *Decimal `json:"unitemised,omitempty"`

	Total Decimal `json:"total"` // Provision + Eigene + Fremde Spesen + Unitemised

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

	// Date is the economically relevant date of the transaction: the trade
	// date (Handelstag/Schlusstag/Buchtag) for anything that moves a
	// position, the value date (Valuta) for pure cash events (dividend,
	// interest, accumulation). OrderDate and ValueDate carry the other two
	// so a consumer can pick a different convention without re-parsing. All
	// three are declared together so they are emitted together: ValueDate
	// used to sit among the dividend fields and surfaced at the far end of a
	// trade object, a long way from the two dates it belongs with.
	Date      string `json:"date"`
	OrderDate string `json:"order_date,omitempty"` // Auftragsdatum — when the order was placed
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
