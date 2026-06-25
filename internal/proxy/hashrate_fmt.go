package proxy

import (
	"fmt"
	"strings"
)

// FormatHashrateMHs converts a raw MH/s hashrate to a formatted string with units.
func FormatHashrateMHs(hs float64, unit string) string {
	if unit == "" {
		if hs >= 1000000000 {
			hs = hs / 1000000000
			unit = "PH/s"
		} else if hs >= 1000000 {
			hs = hs / 1000000
			unit = "TH/s"
		} else if hs >= 1000 {
			hs = hs / 1000
			unit = "GH/s"
		} else {
			unit = "MH/s"
		}
	} else {
		switch strings.ToUpper(unit) {
		case "PH/S", "PH":
			hs = hs / 1000000000
		case "TH/S", "TH":
			hs = hs / 1000000
		case "GH/S", "GH":
			hs = hs / 1000
		}
	}
	return fmt.Sprintf("%.2f %s", hs, unit)
}
