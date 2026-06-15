package adif

import (
	"fmt"
	"regexp"
	"strings"
)

var bandEdges = []struct {
	lo, hi float64
	name   string
}{
	{1800000, 2000000, "160M"},
	{3500000, 4000000, "80M"},
	{5350000, 5450000, "60M"},
	{7000000, 7300000, "40M"},
	{10100000, 10150000, "30M"},
	{14000000, 14350000, "20M"},
	{18068000, 18168000, "17M"},
	{21000000, 21450000, "15M"},
	{24890000, 24990000, "12M"},
	{28000000, 29700000, "10M"},
	{50000000, 54000000, "6M"},
	{144000000, 148000000, "2M"},
	{430000000, 440000000, "70CM"},
}

func FreqToBand(freqHz string) string {
	var f float64
	if _, err := fmt.Sscanf(freqHz, "%f", &f); err != nil {
		return ""
	}
	for _, e := range bandEdges {
		if f >= e.lo && f <= e.hi {
			return e.name
		}
	}
	return ""
}

func MakeField(name, value string) string {
	return fmt.Sprintf("<%s:%d>%s", name, len(value), value)
}

func MakeRecord(fields map[string]string) string {
	var parts []string
	for k, v := range fields {
		if v != "" {
			parts = append(parts, MakeField(k, v))
		}
	}
	parts = append(parts, "<EOR>")
	return strings.Join(parts, " ")
}

var fieldRe = regexp.MustCompile(`(?i)<([A-Z_]+):(\d+)>([^<]+)`)

func ExtractField(adifStr, field string) string {
	for _, m := range fieldRe.FindAllStringSubmatch(adifStr, -1) {
		if strings.EqualFold(m[1], field) {
			return strings.TrimSpace(m[3])
		}
	}
	return ""
}

func ParseLocalKeys(content string) map[[3]string]struct{} {
	keys := make(map[[3]string]struct{})
	records := regexp.MustCompile(`(?i)<eor>\s*`).Split(content, -1)
	for _, rec := range records {
		call := ExtractField(rec, "CALL")
		qd := ExtractField(rec, "QSO_DATE")
		to := ExtractField(rec, "TIME_ON")
		if call != "" && qd != "" {
			key := [3]string{strings.ToUpper(call), qd, to}
			keys[key] = struct{}{}
		}
	}
	return keys
}

func ParseLocalRecords(content string) map[[3]string]string {
	records := make(map[[3]string]string)
	parts := regexp.MustCompile(`(?i)<eor>\s*`).Split(content, -1)
	for _, rec := range parts {
		rec = strings.TrimSpace(rec)
		if rec == "" {
			continue
		}
		call := ExtractField(rec, "CALL")
		qd := ExtractField(rec, "QSO_DATE")
		to := ExtractField(rec, "TIME_ON")
		if call != "" && qd != "" {
			key := [3]string{strings.ToUpper(call), qd, to}
			records[key] = rec + " <EOR>"
		}
	}
	return records
}

func Header() string {
	return "<ADIF_VER:5>3.1.0\n<PROGRAMID:8>wlgate\n<EOH>\n"
}
