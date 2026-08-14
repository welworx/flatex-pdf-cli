package schema

import (
	"fmt"
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
// It is a float64 carrying a scale, not an arbitrary-precision decimal: no
// arithmetic here is exact, and every cross-check in the parser still runs in
// float64 with an explicit tolerance. The scale governs formatting only.
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

// Computed builds an amount this package worked out rather than read. Such a
// value has no printed precision to inherit, so it is given at least two
// places — the cent, the smallest unit any of these documents settles in —
// and more only when an input it was derived from carried more.
func Computed(value float64, scale int) Decimal {
	if scale < minComputedScale {
		scale = minComputedScale
	}
	return Num(value, scale)
}

// minComputedScale is the floor for a value the parser derives: currency here
// is always settled to the cent, so a computed 8.4 is a 8.40 that lost its
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
