package connection

import "fmt"

func HashrateFormat(hs float64) string {
	m := 1000000.0
	g := m * 1000
	t := g * 1000

	if hs > t {
		return fmt.Sprintf("%.2f", hs/t) + " TH/s"
	}
	if hs > g {
		return fmt.Sprintf("%.2f", hs/g) + " GH/s"
	}
	if hs > m {
		return fmt.Sprintf("%.2f", hs/m) + " MH/s"
	}

	return "0 H/s"
}
