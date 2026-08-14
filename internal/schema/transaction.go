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
	Provision       float64 `json:"provision"`        // Provision
	OwnExpenses     float64 `json:"own_expenses"`     // Eigene Spesen
	ForeignExpenses float64 `json:"foreign_expenses"` // Fremde Spesen

	// Unitemised is a charge the document does not print as a line item,
	// recovered as the gap between the amount settled and the value of the
	// shares. Savings-plan settlements are the known case: a row buys
	// (Betrag - charge) / Kurs shares and never names the charge. It is kept
	// separate from the fields above because those carry a label the document
	// actually printed, and this one does not: it is computed, not read.
	Unitemised float64 `json:"unitemised,omitempty"`

	Total float64 `json:"total"` // Provision + Eigene + Fremde Spesen + Unitemised

	// Fees itemises ForeignExpenses ("* Enthalten sind folgende Gebühren").
	// Its components sum to ForeignExpenses, so they are already counted in
	// Total and must not be added on top of it.
	Fees *Fees `json:"fees,omitempty"`
}

// Fees is the itemised breakdown of Costs.ForeignExpenses.
type Fees struct {
	Courtage                float64 `json:"courtage"`
	TradingFee              float64 `json:"trading_fee"`               // Tradinggebühr
	Settlement              float64 `json:"settlement"`                // Regulierung
	ClosingNotes            float64 `json:"closing_notes"`             // Schlussnoten
	LSAllocation            float64 `json:"ls_allocation"`             // LS-Umlegung
	FinancialTransactionTax float64 `json:"financial_transaction_tax"` // Finanztransaktionssteuer
	Other                   float64 `json:"other"`                     // Sonstige
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
	// so a consumer can pick a different convention without re-parsing.
	Date      string `json:"date"`
	OrderDate string `json:"order_date,omitempty"` // Auftragsdatum — when the order was placed

	// TRADE fields
	Type           string  `json:"type,omitempty"`
	Quantity       float64 `json:"quantity,omitempty"`
	Price          float64 `json:"price,omitempty"`
	PriceCurrency  string  `json:"price_currency,omitempty"`
	GrossValue     float64 `json:"gross_value,omitempty"`
	// Pointers so that a document printing "0,00 EUR" stays distinguishable
	// from one that prints no such line at all. A Verkauf sold at cost has a
	// genuine 0,00 Gewinn/Verlust; with a plain float64 that value was dropped
	// by omitempty and read back as "field absent". Same nil-means-absent
	// convention as Costs below.
	WithholdingTax *float64 `json:"withholding_tax,omitempty"`
	GainLoss       *float64 `json:"gain_loss,omitempty"`
	ExchangeRate   float64 `json:"exchange_rate,omitempty"`
	FinalAmount    float64 `json:"final_amount,omitempty"`
	FinalCurrency  string  `json:"final_currency,omitempty"`
	CustodyType    string  `json:"custody_type,omitempty"`
	Depositary     string  `json:"depositary,omitempty"`
	DepositCountry string  `json:"deposit_country,omitempty"` // Lagerland, as an ISO 3166-1 alpha-2 code
	ExecutionVenue string  `json:"execution_venue,omitempty"` // Ausf.platz/-art
	Costs          *Costs  `json:"costs,omitempty"`

	// ORDER fields (Sammelauftragsbestätigung — pending orders)
	Limit      float64 `json:"limit,omitempty"`       // Limit price
	ValidUntil string  `json:"valid_until,omitempty"` // Gültig bis

	// DIVIDEND fields
	DistributionPerShare   float64 `json:"distribution_per_share,omitempty"`
	DistributionCurrency   string  `json:"distribution_currency,omitempty"`
	GrossAmount            float64 `json:"gross_amount,omitempty"`
	GrossCurrency          string  `json:"gross_currency,omitempty"`
	WithholdingTaxCurrency string  `json:"withholding_tax_currency,omitempty"`
	NetAmount              float64 `json:"net_amount,omitempty"`
	NetCurrency            string  `json:"net_currency,omitempty"`
	ExDate                 string  `json:"ex_date,omitempty"`
	ValueDate              string  `json:"value_date,omitempty"`

	// INTEREST fields
	InterestRate float64 `json:"interest_rate,omitempty"`
	PeriodFrom   string  `json:"period_from,omitempty"`
	PeriodTo     string  `json:"period_to,omitempty"`

	// ACCUMULATING fields
	ReinvestmentPerShare float64 `json:"reinvestment_per_share,omitempty"`
	ReinvestmentCurrency string  `json:"reinvestment_currency,omitempty"`
	AccrualDate          string  `json:"accrual_date,omitempty"`
}

// Amount returns the value p points at, or 0 when p is nil. Use it wherever an
// absent optional amount should behave as zero (arithmetic, formatting) while
// the field itself stays nil-means-absent for callers that care.
func Amount(p *float64) float64 {
	if p == nil {
		return 0
	}
	return *p
}

// TotalCosts returns the transaction's total charge, or 0 when the document
// carried no cost block.
func (t *Transaction) TotalCosts() float64 {
	if t.Costs == nil {
		return 0
	}
	return t.Costs.Total
}
