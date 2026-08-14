package schema

import (
	"encoding/json"
	"testing"
)

// The whole point of Decimal: a JSON number keeps the digits the document
// printed. Marshalling through float64 collapsed all of these.
func TestDecimalMarshalsAtPrintedScale(t *testing.T) {
	cases := []struct {
		name  string
		value float64
		scale int
		want  string
	}{
		{"Kurs to six places", 110, 6, "110.000000"},
		{"Kurswert to the cent", 1540, 2, "1540.00"},
		{"whole-share execution", 14, 0, "14"},
		{"commission keeps its zero", 5.9, 2, "5.90"},
		{"a charge of nothing", 0, 2, "0.00"},
		{"negative settlement", -1019.54, 2, "-1019.54"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := json.Marshal(Num(tc.value, tc.scale))
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			if string(got) != tc.want {
				t.Errorf("Num(%v, %d) marshals as %s, want %s", tc.value, tc.scale, got, tc.want)
			}
		})
	}
}

// Computed floors at the cent without ever rounding: a value this package
// supplies has no printed precision to inherit, but it is still money.
func TestComputedFloorsAtTwoPlacesWithoutRounding(t *testing.T) {
	for _, tc := range []struct {
		value float64
		want  string
	}{
		{8.4, "8.40"},            // padded up to the cent
		{0, "0.00"},              // a charge of nothing is still 0,00 EUR
		{1, "1.00"},              // the exchange rate supplied for a EUR document
		{1.4999832, "1.4999832"}, // NOT rounded to 1.50
		{198.5000168, "198.5000168"},
	} {
		if got := Computed(tc.value).String(); got != tc.want {
			t.Errorf("Computed(%v) = %s, want %s", tc.value, got, tc.want)
		}
	}
}

// Deriving one amount from others must be exact. In float64 these identities
// all miss: 5.90+0.00+2.51 gives 8.409999999999999, and the old code hid that
// by snapping the result back to the cent — which also destroyed the genuine
// extra places in a share value.
func TestArithmeticIsExact(t *testing.T) {
	if got := Sum(Num(5.90, 2), Num(0, 2), Num(2.51, 2)).String(); got != "8.41" {
		t.Errorf("Sum(5.90, 0.00, 2.51) = %s, want 8.41", got)
	}

	// 1,478695 shares at 134,2400 — a six-place quantity times a four-place
	// price, so the product genuinely has ten places and keeps the seven it
	// needs. Rounding this to 198.50 is what produced a phantom 1.50 charge.
	shareValue := Mul(Num(1.478695, 6), Num(134.24, 4))
	if got := shareValue.String(); got != "198.5000168" {
		t.Errorf("Mul(1.478695, 134.2400) = %s, want 198.5000168", got)
	}

	charge := Sub(Num(200, 2), shareValue)
	if got := charge.String(); got != "1.4999832" {
		t.Errorf("200.00 - 198.5000168 = %s, want 1.4999832", got)
	}

	// The whole point: the parts add back up to the settled amount exactly.
	if got := Sum(shareValue, charge).String(); got != "200.00" {
		t.Errorf("share value plus charge = %s, want 200.00", got)
	}
}

// The output must survive a decode/encode round trip, so a consumer that reads
// the JSON and writes it back does not quietly restate the precision.
func TestDecimalRoundTripsThroughJSON(t *testing.T) {
	for _, want := range []string{"110.000000", "1540.00", "14", "-1019.54"} {
		var d Decimal
		if err := json.Unmarshal([]byte(want), &d); err != nil {
			t.Fatalf("Unmarshal(%s): %v", want, err)
		}
		got, err := json.Marshal(d)
		if err != nil {
			t.Fatalf("Marshal: %v", err)
		}
		if string(got) != want {
			t.Errorf("round trip of %s produced %s", want, got)
		}
	}
}

// A whole struct marshals as valid JSON — a hand-written MarshalJSON that
// emitted a stray quote or empty output would still pass the tests above.
func TestDecimalEmbedsAsValidJSONNumber(t *testing.T) {
	q, p := Num(14, 0), Num(110, 6)
	data, err := json.Marshal(&Transaction{
		DocumentType: "TRADE", ISIN: "X", Date: "2024-12-18",
		Quantity: &q, Price: &p,
	})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var back map[string]any
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatalf("output is not valid JSON: %s (%v)", data, err)
	}
	if back["price"] != 110.0 {
		t.Errorf("price decoded as %v, want 110: %s", back["price"], data)
	}
}
