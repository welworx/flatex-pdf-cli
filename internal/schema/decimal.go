package schema

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"
)

// Decimal is an amount together with the number of decimal places the document
// printed it with, so the output reproduces the statement's own precision
// instead of Go's shortest round-trip form.
//
// The precision is information the document is telling you. "Kurs :
// 110,000000 EUR" states a price exact to six places; "Ausgeführt : 14 St."
// states a whole-share execution, not a quantity that happens to be round.
// Marshalled as a plain float64 both collapse — 110 and 14 — and a reader can
// no longer tell a price quoted to six places from one quoted to two, nor a
// 5,90 EUR commission from a 5,9 that was never printed that way.
//
// It stores a float64 and a scale. That is enough to hold any figure these
// documents print, but not to add two of them: 5.90 + 0.00 + 2.51 lands on
// 8.409999999999999 in float64. Deriving one amount from others therefore goes
// through Sum, Sub and Mul, which work in exact rationals and hand back a
// result with every digit it genuinely has — no rounding step, and none of the
// noise that would make one necessary.
//
// The parser's cross-checks still run in float64 against an explicit
// tolerance; they compare magnitudes rather than produce output.
type Decimal struct {
	value float64
	scale int
}

// Num builds an amount printed with scale decimal places. A negative scale is
// clamped to zero.
func Num(value float64, scale int) Decimal {
	if scale < 0 {
		scale = 0
	}
	return Decimal{value: value, scale: scale}
}

// Computed builds a currency amount this package supplies rather than reads.
// Such a value has no printed precision to inherit, so it is rendered with
// every digit it actually has and padded to at least two places — the cent,
// the smallest unit any of these documents settles in.
//
// Nothing is rounded away: the value is rendered in shortest round-trip form
// first, so a figure with more precision than two places keeps all of it. Use
// Sum, Sub and Mul to derive a value from others; passing the float64 result
// of your own arithmetic here would show its error rather than hide it, which
// is the point.
func Computed(value float64) Decimal {
	s := strconv.FormatFloat(value, 'f', -1, 64)
	scale := ScaleOf(s)
	if scale < minComputedScale {
		scale = minComputedScale
	}
	return Num(value, scale)
}

// minComputedScale is the floor for a currency value the parser derives: it is
// always settled to the cent, so a computed 8.4 is an 8.40 that lost its
// trailing zero, not a figure known to one place.
const minComputedScale = 2

// Float returns the underlying value for arithmetic.
func (d Decimal) Float() float64 { return d.value }

// Scale returns the number of decimal places the value is printed with.
func (d Decimal) Scale() int { return d.scale }

// String renders the value at its own precision.
func (d Decimal) String() string {
	return strconv.FormatFloat(d.value, 'f', d.scale, 64)
}

// MarshalJSON writes the value as a JSON number at its own precision, so
// 1.540,00 EUR emits as 1540.00 rather than 1540.
func (d Decimal) MarshalJSON() ([]byte, error) {
	return []byte(d.String()), nil
}

// UnmarshalJSON reads a JSON number back, recovering the scale from the digits
// actually present so a decode/encode round trip preserves the precision.
func (d *Decimal) UnmarshalJSON(b []byte) error {
	s := strings.TrimSpace(string(b))
	if s == "null" {
		return nil
	}
	s = strings.Trim(s, `"`) // tolerate a quoted number
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fmt.Errorf("decimal: %w", err)
	}
	*d = Num(v, ScaleOf(s))
	return nil
}

// rat returns the value as an exact rational, read from its own rendered
// digits so the rational is the decimal the document stated, not the binary
// approximation float64 stores it as.
func (d Decimal) rat() *big.Rat {
	r, ok := new(big.Rat).SetString(d.String())
	if !ok {
		// Unreachable: String always emits a plain decimal.
		return new(big.Rat)
	}
	return r
}

// Sum returns the exact sum of ds. A sum of decimals needs no more places than
// its widest input, so the result is exact and nothing is rounded.
//
// Doing this in float64 is what forced rounding before: 5.90 + 0.00 + 2.51
// lands on 8.409999999999999, which then had to be snapped back to 8.41. Here
// the addition is exact and the rendering is the true value.
func Sum(ds ...Decimal) Decimal {
	total, scale := new(big.Rat), 0
	for _, d := range ds {
		total.Add(total, d.rat())
		if d.Scale() > scale {
			scale = d.Scale()
		}
	}
	return fromRat(total, scale)
}

// Sub returns the exact difference a-b, at the wider of the two scales.
func Sub(a, b Decimal) Decimal {
	scale := a.Scale()
	if b.Scale() > scale {
		scale = b.Scale()
	}
	return fromRat(new(big.Rat).Sub(a.rat(), b.rat()), scale)
}

// Mul returns the exact product a*b. A product needs exactly the sum of its
// operands' scales, so 1.478695 shares at 134.2400 is carried to all ten
// places it genuinely has rather than snapped to the cent.
func Mul(a, b Decimal) Decimal {
	return fromRat(new(big.Rat).Mul(a.rat(), b.rat()), a.Scale()+b.Scale())
}

// fromRat renders r at scale places — exact, because every caller passes a
// scale the result is representable at — then drops trailing zeros down to the
// two places currency keeps. The float64 it parses back differs from the
// decimal by far less than the last digit rendered, so the digits are the
// exact ones.
func fromRat(r *big.Rat, scale int) Decimal {
	if scale < minComputedScale {
		scale = minComputedScale
	}
	s := r.FloatString(scale)
	for scale > minComputedScale && strings.HasSuffix(s, "0") {
		s = s[:len(s)-1]
		scale--
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return Decimal{}
	}
	return Num(v, scale)
}

// ScaleOf counts the digits after the decimal point of a number already
// normalised to Go syntax (see the parser's normalizeDecimal). Exponent forms
// are not produced by these documents and are not handled.
func ScaleOf(normalized string) int {
	i := strings.IndexByte(normalized, '.')
	if i < 0 {
		return 0
	}
	return len(normalized) - i - 1
}
