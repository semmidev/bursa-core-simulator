package main

import (
	"fmt"
)

func FmtRupiah(v int64) string {
	s := fmt.Sprintf("%d", v)
	var out []byte
	for i := len(s) - 1; i >= 0; i-- {
		out = append([]byte{s[i]}, out...)
		if (len(s)-i)%3 == 0 && i != 0 && s[i-1] != '-' {
			out = append([]byte{'.'}, out...)
		}
	}
	return "Rp " + string(out)
}

func FmtNumber(v int64) string {
	s := fmt.Sprintf("%d", v)
	var out []byte
	for i := len(s) - 1; i >= 0; i-- {
		out = append([]byte{s[i]}, out...)
		if (len(s)-i)%3 == 0 && i != 0 && s[i-1] != '-' {
			out = append([]byte{'.'}, out...)
		}
	}
	return string(out)
}

func FmtBillions(v int64) string {
	b := float64(v) / 1_000_000_000
	return fmt.Sprintf("%.2f M", b)
}

func FmtPercent(change int64, prev int64) string {
	if prev <= 0 {
		return "0.00%"
	}
	pct := float64(change) / float64(prev) * 100
	if pct >= 0 {
		return fmt.Sprintf("+%.2f%%", pct)
	}
	return fmt.Sprintf("%.2f%%", pct)
}
