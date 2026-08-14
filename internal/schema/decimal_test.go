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

// Computed floors at the cent: a value this package worked out has no printed
// precision to inherit, but it is still money.
func TestComputedFloorsAtTwoPlaces(t *testing.T) {
	if got := Computed(8.4, 0).String(); got != "8.40" {
		t.Errorf("Computed(8.4, 0) = %s, want 8.40", got)
	}
	// An input that carried more places is not truncated back to two.
	if got := Computed(1.5, 6).String(); got != "1.500000" {
		t.Errorf("Computed(1.5, 6) = %s, want 1.500000", got)
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
