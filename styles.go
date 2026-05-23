package main

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ── Palette ───────────────────────────────────────────────────

var (
	ColorBg       = lipgloss.Color("#0D1117")
	ColorSurface  = lipgloss.Color("#161B22")
	ColorBorder   = lipgloss.Color("#30363D")
	ColorMuted    = lipgloss.Color("#6E7681")
	ColorText     = lipgloss.Color("#E6EDF3")
	ColorAccent   = lipgloss.Color("#58A6FF")
	ColorGreen    = lipgloss.Color("#3FB950")
	ColorRed      = lipgloss.Color("#F85149")
	ColorYellow   = lipgloss.Color("#D29922")
	ColorOrange   = lipgloss.Color("#E3B341")
	ColorPurple   = lipgloss.Color("#BC8CFF")
	ColorCyan     = lipgloss.Color("#39C5CF")
	ColorDimGreen = lipgloss.Color("#1B4332")
	ColorDimRed   = lipgloss.Color("#3B1C1C")
)

// ── Base Styles ───────────────────────────────────────────────

var (
	StyleBase = lipgloss.NewStyle().
			Background(ColorBg).
			Foreground(ColorText)

	StyleTitle = lipgloss.NewStyle().
			Foreground(ColorAccent).
			Bold(true)

	StyleSubtitle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	StyleSuccess = lipgloss.NewStyle().Foreground(ColorGreen)
	StyleError   = lipgloss.NewStyle().Foreground(ColorRed)
	StyleWarning = lipgloss.NewStyle().Foreground(ColorYellow)
	StyleMuted   = lipgloss.NewStyle().Foreground(ColorMuted)
	StyleBold    = lipgloss.NewStyle().Bold(true).Foreground(ColorText)
	StyleAccent  = lipgloss.NewStyle().Foreground(ColorAccent)
	StylePurple  = lipgloss.NewStyle().Foreground(ColorPurple)
	StyleCyan    = lipgloss.NewStyle().Foreground(ColorCyan)

	StyleBorderBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder)

	StyleCard = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Background(ColorSurface).
			Padding(0, 1)

	StyleHeader = lipgloss.NewStyle().
			Background(ColorSurface).
			Foreground(ColorAccent).
			Bold(true).
			Padding(0, 2)

	StyleFooter = lipgloss.NewStyle().
			Foreground(ColorMuted).
			Background(ColorSurface).
			Padding(0, 1)

	StyleActiveTab = lipgloss.NewStyle().
			Foreground(ColorBg).
			Background(ColorAccent).
			Bold(true).
			Padding(0, 2)

	StyleInactiveTab = lipgloss.NewStyle().
				Foreground(ColorMuted).
				Background(ColorSurface).
				Padding(0, 2)

	StyleSelectedRow = lipgloss.NewStyle().
				Background(lipgloss.Color("#1C2128")).
				Foreground(ColorAccent).
				Bold(true)

	StyleTableHeader = lipgloss.NewStyle().
				Foreground(ColorMuted).
				Bold(true)
)

// ── Price colour helpers ──────────────────────────────────────

func PriceStyle(price, prev int64) lipgloss.Style {
	switch {
	case price > prev:
		return lipgloss.NewStyle().Foreground(ColorGreen).Bold(true)
	case price < prev:
		return lipgloss.NewStyle().Foreground(ColorRed).Bold(true)
	default:
		return lipgloss.NewStyle().Foreground(ColorText)
	}
}

func ChangeStyle(change int64) lipgloss.Style {
	if change > 0 {
		return lipgloss.NewStyle().Foreground(ColorGreen)
	} else if change < 0 {
		return lipgloss.NewStyle().Foreground(ColorRed)
	}
	return lipgloss.NewStyle().Foreground(ColorMuted)
}

func ChangeArrow(change int64) string {
	if change > 0 {
		return lipgloss.NewStyle().Foreground(ColorGreen).Render("▲")
	} else if change < 0 {
		return lipgloss.NewStyle().Foreground(ColorRed).Render("▼")
	}
	return lipgloss.NewStyle().Foreground(ColorMuted).Render("─")
}

func BidStyle() lipgloss.Style { return lipgloss.NewStyle().Foreground(ColorGreen) }
func AskStyle() lipgloss.Style { return lipgloss.NewStyle().Foreground(ColorRed) }
func BidBgStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(ColorGreen).Background(ColorDimGreen)
}
func AskBgStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(ColorRed).Background(ColorDimRed)
}

// ── Number formatting ─────────────────────────────────────────

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

// ── Box / layout helpers ──────────────────────────────────────

func HR(width int) string {
	return StyleMuted.Render(strings.Repeat("─", width))
}

func SectionTitle(title string) string {
	return StyleTitle.Render("▌ "+title) + "\n" +
		StyleMuted.Render(strings.Repeat("─", len(title)+4))
}

// Pad pads string to given visual width (no colour codes counted)
func Pad(s string, width int) string {
	vis := lipgloss.Width(s)
	if vis >= width {
		return s
	}
	return s + strings.Repeat(" ", width-vis)
}

// TruncStr truncates to max visual characters
func TruncStr(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-1]) + "…"
}

// Banner ASCII art
const BannerText = `    ▄▄▄     ▄▄▄▄▄▄▄   ▄▄▄▄▄▄        ▄▄▄▄▄▄▄                                                   ▄▄▄▄▄                    ▄▄
   ██▀▀█▄  █▀██▀▀▀   █▀ ██         █▀██▀▀▀               █▄                                  ██▀▀▀▀█▄                   ██       █▄
   ██ ▄█▀    ██         ██           ██                  ██          ▄        ▄▄             ▀██▄  ▄▀ ▀▀ ▄              ██      ▄██▄      ▄
   ██▀▀█▄    ████       ██           ████  ▀██ ██▀ ▄███▀ ████▄ ▄▀▀█▄ ████▄ ▄████ ▄█▀█▄         ▀██▄▄  ██ ███▄███▄ ██ ██ ██ ▄▀▀█▄ ██ ▄███▄ ████▄
 ▄ ██  ▄█    ██         ██   ▀▀▀▀    ██      ███   ██    ██ ██ ▄█▀██ ██ ██ ██ ██ ██▄█▀ ▀▀▀▀  ▄   ▀██▄ ██ ██ ██ ██ ██ ██ ██ ▄█▀██ ██ ██ ██ ██
 ▀██████▀    ▀█████   ▄▄██▄▄         ▀█████▄██ ██▄▄▀███▄▄██ ██▄▀█▄██▄██ ▀█▄▀████▄▀█▄▄▄       ▀██████▀▄██▄██ ██ ▀█▄▀██▀█▄██▄▀█▄██▄██▄▀███▀▄█▀
                                                                              ██
                                                                            ▀▀▀
`
