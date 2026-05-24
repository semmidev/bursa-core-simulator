package main

import (
	"fmt"
	"math"
	"strings"
)

func FmtRupiah(v int64) string {
	if v < 0 {
		return "Rp -" + fmtInt(-v)
	}
	return "Rp " + fmtInt(v)
}

func FmtNumber(v int64) string { return fmtInt(v) }

func fmtInt(v int64) string {
	s := fmt.Sprintf("%d", v)
	n := len(s)
	var b strings.Builder
	for i, c := range s {
		if i > 0 && (n-i)%3 == 0 {
			b.WriteRune('.')
		}
		b.WriteRune(c)
	}
	return b.String()
}

func FmtBillions(v int64) string {
	b := float64(v) / 1_000_000_000
	if math.Abs(b) >= 1000 {
		return fmt.Sprintf("%.1f T", b/1000)
	}
	return fmt.Sprintf("%.2f B", b)
}

func FmtPercent(v float64) string {
	sign := "+"
	if v < 0 {
		sign = ""
	}
	return fmt.Sprintf("%s%.2f%%", sign, v)
}
