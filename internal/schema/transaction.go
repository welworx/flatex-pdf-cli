package schema

import "time"

// DocumentMetadata holds metadata about the parsed document.
type DocumentMetadata struct {
	Source       string    `json:"source,omitempty"`
	DocNumber    string    `json:"doc_number"`
	DocumentType string    `json:"document_type"`
	Date         time.Time `json:"date"`
	ParsedAt     time.Time `json:"parsed_at,omitempty"`
}

// Transaction represents a single transaction (trade, dividend, interest, or thesaurierung).
type Transaction struct {
	// Common fields
	Source       string    `json:"source,omitempty"`
	DocNumber    string    `json:"doc_number"`
	DocumentType string    `json:"document_type"`
	ISIN         string    `json:"isin"`
	WKN          string    `json:"wkn,omitempty"`
	Date         time.Time `json:"date"`

	// TRADE fields
	Type            string  `json:"type,omitempty"`
	Quantity        float64 `json:"quantity,omitempty"`
	Price           float64 `json:"price,omitempty"`
	PriceCurrency   string  `json:"price_currency,omitempty"`
	GrossValue      float64 `json:"gross_value,omitempty"`
	Provision       float64 `json:"provision,omitempty"`
	OwnCosts        float64 `json:"own_costs,omitempty"`
	ThirdPartyCosts float64 `json:"third_party_costs,omitempty"`
	WithholdingTax  float64 `json:"withholding_tax,omitempty"`
	GainLoss        float64 `json:"gain_loss,omitempty"`
	ExchangeRate    float64 `json:"exchange_rate,omitempty"`
	FinalAmount     float64 `json:"final_amount,omitempty"`
	FinalCurrency   string  `json:"final_currency,omitempty"`
	CustodyType     string  `json:"custody_type,omitempty"`
	Depositary      string  `json:"depositary,omitempty"`
	Country         string  `json:"country,omitempty"`

	// DIVIDEND fields
	DistributionPerShare   float64   `json:"distribution_per_share,omitempty"`
	DistributionCurrency   string    `json:"distribution_currency,omitempty"`
	GrossAmount            float64   `json:"gross_amount,omitempty"`
	GrossCurrency          string    `json:"gross_currency,omitempty"`
	WithholdingTaxCurrency string    `json:"withholding_tax_currency,omitempty"`
	NetAmount              float64   `json:"net_amount,omitempty"`
	NetCurrency            string    `json:"net_currency,omitempty"`
	ExDate                 time.Time `json:"ex_date,omitempty"`
	ValueDate              time.Time `json:"value_date,omitempty"`

	// INTEREST fields
	InterestRate float64   `json:"interest_rate,omitempty"`
	PeriodFrom   time.Time `json:"period_from,omitempty"`
	PeriodTo     time.Time `json:"period_to,omitempty"`

	// THESAURIERUNG fields
	ReinvestmentPerShare float64   `json:"reinvestment_per_share,omitempty"`
	ReinvestmentCurrency string    `json:"reinvestment_currency,omitempty"`
	AccrualDate          time.Time `json:"accrual_date,omitempty"`
}
