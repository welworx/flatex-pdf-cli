package parser

import "testing"

// TestCountryISO2 covers the Lagerland translation, including the reason it
// matches a prefix: gxpdf merges the Lagerland column with the one to its
// right, so the raw field text runs straight into the next label.
func TestCountryISO2(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"plain name", "Großbritannien", "GB"},
		{"trailing column padding", "Großbritannien          Bemessungs-", "GB"},
		{"merged with the next column", "GroßbritannienBemessungsgrundlage: 0,00 EUR", "GB"},
		{"ss spelling", "Grossbritannien", "GB"},
		{"multi-word name", "Vereinigtes Königreich", "GB"},
		{"umlaut", "Österreich", "AT"},
		{"already short", "USA", "US"},
		{"empty", "", ""},
		{"unknown name yields nothing rather than a guess", "Absurdistan", ""},
		{"label leftovers do not match", "Bemessungsgrundlage: 0,00 EUR", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := countryISO2(tc.in); got != tc.want {
				t.Errorf("countryISO2(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// Every code in the table must be a two-letter uppercase ISO 3166-1 alpha-2
// code — a lowercase or three-letter entry would flow straight into the JSON.
func TestCountryCodesAreISO2(t *testing.T) {
	for name, code := range germanCountryISO2 {
		if len(code) != 2 {
			t.Errorf("%q maps to %q, want a 2-letter code", name, code)
			continue
		}
		for _, r := range code {
			if r < 'A' || r > 'Z' {
				t.Errorf("%q maps to %q, want uppercase A-Z", name, code)
				break
			}
		}
	}
}

// countryNamesByLength must be ordered longest-first, so that adding a name
// which starts with an existing one cannot shadow the longer match.
func TestCountryNamesOrderedLongestFirst(t *testing.T) {
	if len(countryNamesByLength) != len(germanCountryISO2) {
		t.Fatalf("ordered list has %d names, map has %d",
			len(countryNamesByLength), len(germanCountryISO2))
	}
	for i := 1; i < len(countryNamesByLength); i++ {
		if len(countryNamesByLength[i-1]) < len(countryNamesByLength[i]) {
			t.Fatalf("names not ordered longest-first at %d: %q before %q",
				i, countryNamesByLength[i-1], countryNamesByLength[i])
		}
	}
}
