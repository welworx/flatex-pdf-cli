package parser

import (
	"sort"
	"strings"
)

// germanCountryISO2 maps the German country names flatex prints in the
// "Lagerland" field to ISO 3166-1 alpha-2 codes. It covers the European and
// major global custody locations a flatex depot can reference; an unlisted
// name yields no code rather than a guess.
var germanCountryISO2 = map[string]string{
	"Deutschland":              "DE",
	"Österreich":               "AT",
	"Schweiz":                  "CH",
	"Liechtenstein":            "LI",
	"Großbritannien":           "GB",
	"Grossbritannien":          "GB",
	"Vereinigtes Königreich":   "GB",
	"Irland":                   "IE",
	"Luxemburg":                "LU",
	"Niederlande":              "NL",
	"Belgien":                  "BE",
	"Frankreich":               "FR",
	"Monaco":                   "MC",
	"Italien":                  "IT",
	"Spanien":                  "ES",
	"Portugal":                 "PT",
	"Griechenland":             "GR",
	"Malta":                    "MT",
	"Zypern":                   "CY",
	"Schweden":                 "SE",
	"Norwegen":                 "NO",
	"Dänemark":                 "DK",
	"Finnland":                 "FI",
	"Island":                   "IS",
	"Estland":                  "EE",
	"Lettland":                 "LV",
	"Litauen":                  "LT",
	"Polen":                    "PL",
	"Tschechien":               "CZ",
	"Tschechische Republik":    "CZ",
	"Slowakei":                 "SK",
	"Slowenien":                "SI",
	"Ungarn":                   "HU",
	"Kroatien":                 "HR",
	"Rumänien":                 "RO",
	"Bulgarien":                "BG",
	"Türkei":                   "TR",
	"Russland":                 "RU",
	"USA":                      "US",
	"Vereinigte Staaten":       "US",
	"Kanada":                   "CA",
	"Mexiko":                   "MX",
	"Brasilien":                "BR",
	"Japan":                    "JP",
	"China":                    "CN",
	"Hongkong":                 "HK",
	"Singapur":                 "SG",
	"Südkorea":                 "KR",
	"Taiwan":                   "TW",
	"Indien":                   "IN",
	"Israel":                   "IL",
	"Südafrika":                "ZA",
	"Australien":               "AU",
	"Neuseeland":               "NZ",
	"Jersey":                   "JE",
	"Guernsey":                 "GG",
	"Isle of Man":              "IM",
	"Bermuda":                  "BM",
	"Kaimaninseln":             "KY",
	"Cayman Islands":           "KY",
	"Britische Jungferninseln": "VG",
}

// countryNamesByLength lists the map's keys longest first, so that a name
// which starts with a shorter one cannot shadow it.
var countryNamesByLength = func() []string {
	names := make([]string, 0, len(germanCountryISO2))
	for n := range germanCountryISO2 {
		names = append(names, n)
	}
	sort.Slice(names, func(i, j int) bool {
		if len(names[i]) != len(names[j]) {
			return len(names[i]) > len(names[j])
		}
		return names[i] < names[j]
	})
	return names
}()

// countryISO2 reads a German country name off the front of s and returns its
// ISO 3166-1 alpha-2 code, or "" if s does not start with a known name.
//
// It matches a prefix rather than the whole string because gxpdf merges the
// Lagerland column with the one to its right: a real confirmation yields
// "GroßbritannienBemessungsgrundlage: 0,00 EUR" on one line, with no
// separator to split on. The country list is therefore also what tokenises
// the field, which is why an unknown name yields nothing at all — without a
// name to match, there is no reliable way to tell where the value ends.
func countryISO2(s string) string {
	s = strings.TrimSpace(s)
	for _, name := range countryNamesByLength {
		if strings.HasPrefix(s, name) {
			return germanCountryISO2[name]
		}
	}
	return ""
}
