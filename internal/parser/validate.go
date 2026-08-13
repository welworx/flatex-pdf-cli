package parser

import (
	"fmt"
	"math"

	"github.com/welworx/flatex-pdf-cli/internal/schema"
)

// relTolerance is the relative slack allowed when a check has to reconstruct an
// amount from a printed quantity. flatex prints trade quantities to six
// decimals (0,685401 St.) and dividend holdings to two (74,45 St.), so on those
// documents quantity x price reproduces the stated total to within about
// 0.01%. The bound is set two orders of magnitude looser than that so ordinary
// rounding never trips it, while a value that landed in the wrong column (off
// by a factor, not by a rounding step) still does.
const relTolerance = 0.005

// absTolerance is used where both sides are stated amounts rather than
// reconstructed ones, so the identity holds to the cent and no slack for
// rounding is warranted.
const absTolerance = 0.01

// maxUnitemisedShare bounds the charge that may be recovered from the gap
// between what a savings-plan row settled and what its shares are worth. The
// observed gap is a flat 1.50 EUR on a 200.00 EUR order, so the ceiling sits
// well above any plausible fee while still refusing to explain away a large
// discrepancy as one. A layout change that swapped the Kurs and Betrag columns
// produces a gap that is negative or many times the order, and that must fail
// loudly rather than be booked as a suspiciously large fee.
const maxUnitemisedShare = 0.05

// unitemisedCharge recovers the charge a settlement row does not print, as the
// difference between the cash that moved and the value of the shares. A
// purchase settles more than the shares are worth, a sale settles less, so the
// charge is positive either way.
func unitemisedCharge(tradeType string, settled, shareValue float64) (float64, error) {
	charge := roundCents(settled - shareValue)
	if tradeType == "SELL" {
		charge = roundCents(shareValue - settled)
	}
	if charge < 0 || charge > math.Abs(settled)*maxUnitemisedShare {
		return 0, fmt.Errorf(
			"settled amount %.2f and share value %.2f differ by %.2f, which is too large to be an unitemised charge: the statement layout may have changed",
			settled, shareValue, charge)
	}
	return charge, nil
}

// roundCents snaps a computed amount to whole cents. Reconstructing a share
// value from a printed quantity lands fractions of a cent off, and a currency
// amount carried at that precision only propagates noise into every figure
// derived from it.
func roundCents(v float64) float64 {
	return math.Round(v*100) / 100
}

// validate cross-checks amounts that a document states in more than one place.
//
// Strict field extraction already catches a layout change that moves or renames
// a label: the field is not found and the parse fails by name. What it cannot
// catch is a layout change that keeps every label and shifts the values behind
// them, so all fields parse and one of them is quietly wrong. These identities
// hold inside the document itself, so a value that landed in the wrong field
// breaks them. That turns a silently wrong parse back into a loud one.
func validate(txns []*schema.Transaction) error {
	for _, t := range txns {
		if err := validateTransaction(t); err != nil {
			return err
		}
	}
	return nil
}

func validateTransaction(t *schema.Transaction) error {
	switch t.DocumentType {
	case "TRADE":
		// Kurs is only ever extracted in EUR. When Kurswert carries a foreign
		// currency the two are related by Devisenkurs rather than directly, so
		// the identity does not apply.
		if t.PriceCurrency == "EUR" {
			if err := checkProduct("gross value", t.Quantity, t.Price, t.GrossValue); err != nil {
				return err
			}
		}
		return checkSettlement(t)

	case "CRYPTO":
		// Deliberately no quantity x price check here. Crypto quantities are
		// printed to three decimals (0,014 St.), so reconstructing the gross
		// value from them can be off by several percent on small positions:
		// the check would be either useless or a source of false alarms. The
		// settlement identity below covers these documents exactly instead.
		return checkSettlement(t)

	case "DIVIDEND":
		// Gross minus withholding tax is deliberately not checked against the
		// net amount: on the testdata dividends the gross and tax are in USD
		// while Endbetrag is in EUR, so the two sides are separated by a
		// conversion rather than by a subtraction.
		if t.DistributionCurrency == t.GrossCurrency {
			if err := checkProduct("gross amount", t.Quantity, t.DistributionPerShare, t.GrossAmount); err != nil {
				return err
			}
		}
	}
	// SAVINGSPLAN is checked at parse time instead, by unitemisedCharge. Its
	// rows are reconciled by construction once the unprinted charge is
	// recovered, so running the settlement identity over them here would only
	// confirm arithmetic this package just performed. The bound on the
	// recovered charge is the check that can actually fail.
	// ORDER carries no price, gross value or settlement total at all.
	return nil
}

// checkSettlement verifies the settlement line: Endbetrag is what actually left
// the account, and on a purchase it is the gross value plus the costs and taxes
// the same document itemises. Both sides are stated amounts, so this holds
// exactly, which makes it the strongest signal available that every figure
// landed in the field it belongs to.
func checkSettlement(t *schema.Transaction) error {
	// Restricted to purchases: on a sale the deductions reverse sign, and there
	// is no sell document in the fixtures to confirm the arrangement against.
	// Guessing at it would risk failing every future sale.
	if t.Type != "BUY" || t.FinalAmount == 0 || t.GrossValue == 0 {
		return nil
	}
	want := t.GrossValue + t.TotalCosts() + t.WithholdingTax
	got := math.Abs(t.FinalAmount)
	if math.Abs(want-got) <= absTolerance {
		return nil
	}
	return fmt.Errorf("settlement total %.2f does not equal gross %.2f plus costs %.2f plus tax %.2f (expected %.2f): the statement layout may have changed",
		got, t.GrossValue, t.TotalCosts(), t.WithholdingTax, want)
}

// checkProduct verifies that stated == a*b within tolerance. It is skipped when
// any operand is zero: a document that legitimately omits one of these figures
// is not this check's business, the per-field extraction errors already cover
// values that are genuinely missing.
func checkProduct(name string, a, b, stated float64) error {
	if a == 0 || b == 0 || stated == 0 {
		return nil
	}
	want := a * b
	scale := math.Abs(want)
	if scale == 0 {
		scale = math.Abs(stated)
	}
	if math.Abs(want-stated) <= scale*relTolerance {
		return nil
	}
	return fmt.Errorf("%s %.2f does not reconcile with the document's own figures (expected about %.2f): the statement layout may have changed", name, stated, want)
}
