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

// austrianKEStRate is the Austrian Kapitalertragsteuer rate on realised capital
// gains. flatex computes the withheld tax itself and this package only extracts
// it — the rate is never used to recompute the figure, only as a ceiling on it.
//
// The withheld amount can legitimately fall far below the rate and this must
// not be flagged: it is netted against the Verluststeuertopf (the corpus's only
// sale withholds 24.51 on a gain of 403.97, i.e. 6.07%), and Altbestand bought
// before the regime took effect is exempt entirely, so 0,00 on a real gain is
// valid too. What cannot happen is withholding *more* than the rate allows on
// the stated gain, which is what a value that landed in the wrong column looks
// like.
const austrianKEStRate = 0.275

// maxUnitemisedShare bounds the charge that may be recovered from the gap
// between what a savings-plan row settled and what its shares are worth. The
// observed gap is a flat 1.50 EUR on a 200.00 EUR order, so the ceiling sits
// well above any plausible fee while still refusing to explain away a large
// discrepancy as one. A layout change that swapped the Kurs and Betrag columns
// produces a gap that is negative or many times the order, and that must fail
// loudly rather than be booked as a suspiciously large fee.
const maxUnitemisedShare = 0.05

// checkSavingsPlanRow verifies that a Sammelabrechnung row's columns landed
// where they belong. A row prints Stücke, Ausf.-Kurs and Betrag; the shares are
// worth slightly less than a purchase settled, or slightly more than a sale
// settled, because flatex withholds a fee it never prints. That gap is
// therefore small and one-signed, and a layout change that swapped the Kurs
// and Betrag columns makes it neither.
//
// The gap is computed and thrown away. It used to be reported as
// costs.unitemised, which meant emitting a fee no line of the document carries
// — and one whose digits were an artefact of the quantity being printed to six
// places rather than precision the statement has.
func checkSavingsPlanRow(tradeType string, settled, shareValue schema.Decimal) error {
	charge := schema.Sub(settled, shareValue)
	if tradeType == "SELL" {
		charge = schema.Sub(shareValue, settled)
	}
	// The lower bound allows a cent of slack rather than demanding a
	// non-negative gap: rebuilt from a six-decimal quantity, a row that really
	// charged nothing computes to -0.00000328. That is noise in the last
	// places, not a negative fee, and it is far below the swapped-column error
	// this bound exists to catch.
	if v := charge.Float(); v < -absTolerance || v > math.Abs(settled.Float())*maxUnitemisedShare {
		return fmt.Errorf(
			"settled amount %s and share value %s differ by %s, which is not a plausible withheld charge: the statement layout may have changed",
			settled, shareValue, charge)
	}
	return nil
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
	if t.ISIN != "" && !isinChecksumValid(t.ISIN) {
		return fmt.Errorf("ISIN %q fails its check digit: the statement layout may have changed", t.ISIN)
	}
	switch t.DocumentType {
	case "TRADE":
		// Kurs is only ever extracted in EUR. When Kurswert carries a foreign
		// currency the two are related by Devisenkurs rather than directly, so
		// the identity does not apply.
		if t.GrossCurrency == "EUR" {
			if err := checkProduct("gross value", t.Quantity, t.Price, t.GrossAmount); err != nil {
				return err
			}
		}
		if err := checkWithholdingTax(t); err != nil {
			return err
		}
		return checkSettlement(t)

	case "CRYPTO":
		// Deliberately no quantity x price check here. Crypto quantities are
		// printed to three decimals (0,014 St.), so reconstructing the gross
		// value from them can be off by several percent on small positions:
		// the check would be either useless or a source of false alarms. The
		// settlement identity below covers these documents exactly instead.
		if err := checkWithholdingTax(t); err != nil {
			return err
		}
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
	// Skipped only when the document states no such figure. A stated 0,00 is
	// checked like any other value: it used to disable this check silently,
	// which let an impossible settlement through.
	if (t.Type != "BUY" && t.Type != "SELL") || t.NetAmount == nil || t.GrossAmount == nil {
		return nil
	}
	grossValue := schema.Amount(t.GrossAmount)
	// A purchase adds the deductions to what you pay; a sale subtracts them
	// from what you receive. Confirmed against verkauf_sample_1: gross 1540.00
	// less costs 8.41 less KESt 24.51 equals the stated Endbetrag of 1507.08.
	tax := schema.Amount(t.WithholdingTax)
	sign := 1.0
	verb := "plus"
	if t.Type == "SELL" {
		sign, verb = -1.0, "less"
	}
	want := grossValue + sign*(t.TotalCosts()+tax)
	got := math.Abs(schema.Amount(t.NetAmount))
	if math.Abs(want-got) <= absTolerance {
		return nil
	}
	return fmt.Errorf("settlement total %.2f does not equal gross %.2f %s costs %.2f %s tax %.2f (expected %.2f): the statement layout may have changed",
		got, grossValue, verb, t.TotalCosts(), verb, tax, want)
}

// checkWithholdingTax bounds the withheld KESt against the gain the same
// document states. Austrian KESt is levied on the Gewinn/Verlust, so on a gain
// the two are related and a tax that exceeds the statutory share of that gain
// means one of the two figures did not land in the field it belongs to.
//
// On a loss nothing is checked. A realised loss can refund tax already withheld
// earlier in the year (Verluststeuertopf), so the withheld amount is negative
// and bounded by that year's prior withholdings — a figure this document does
// not carry and this package cannot reconstruct.
func checkWithholdingTax(t *schema.Transaction) error {
	if t.GainLoss == nil || t.WithholdingTax == nil {
		return nil
	}
	gain, tax := t.GainLoss.Float(), t.WithholdingTax.Float()
	if gain <= 0 {
		return nil
	}
	if tax < 0 {
		return fmt.Errorf("withheld tax %.2f is negative on a gain of %.2f: a refund arises from a loss, not a gain, so the statement layout may have changed", tax, gain)
	}
	if ceiling := gain*austrianKEStRate + absTolerance; tax > ceiling {
		return fmt.Errorf("withheld tax %.2f exceeds %.1f%% of the stated gain %.2f (at most %.2f): the statement layout may have changed",
			tax, austrianKEStRate*100, gain, ceiling)
	}
	return nil
}

// checkProduct verifies that stated == a*b within tolerance. It is skipped when
// the document states no value for one of the operands: a document that
// legitimately omits one of these figures is not this check's business, the
// per-field extraction errors already cover values that are genuinely missing.
// A stated 0,00 is a value and is checked.
func checkProduct(name string, aP, bP, statedP *schema.Decimal) error {
	if aP == nil || bP == nil || statedP == nil {
		return nil
	}
	a, b, stated := aP.Float(), bP.Float(), statedP.Float()
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
