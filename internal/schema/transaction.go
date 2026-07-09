package schema

// DocumentMetadata holds metadata about the parsed document.
type DocumentMetadata struct {
	DepotNumber   string `json:"depot_number"`
	DepotHolder   string `json:"depot_holder"`
	AccountNumber string `json:"account_number,omitempty"`
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
	Date              string `json:"date"`

	// TRADE fields
	Type           string  `json:"type,omitempty"`
	Quantity       float64 `json:"quantity,omitempty"`
	Price          float64 `json:"price,omitempty"`
	PriceCurrency  string  `json:"price_currency,omitempty"`
	GrossValue     float64 `json:"gross_value,omitempty"`
	Provision      float64 `json:"provision,omitempty"`
	WithholdingTax float64 `json:"withholding_tax,omitempty"`
	GainLoss       float64 `json:"gain_loss,omitempty"`
	ExchangeRate   float64 `json:"exchange_rate,omitempty"`
	FinalAmount    float64 `json:"final_amount,omitempty"`
	FinalCurrency  string  `json:"final_currency,omitempty"`
	CustodyType    string  `json:"custody_type,omitempty"`
	Depositary     string  `json:"depositary,omitempty"`
	ExecutionVenue string  `json:"execution_venue,omitempty"` // Ausf.platz/-art

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
