package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	_ "github.com/lib/pq"
)

func main() {
	// Connect DB
	if err := Connect(); err != nil {
		fmt.Fprintf(os.Stderr, "✗ Koneksi database gagal: %v\n", err)
		fmt.Fprintf(os.Stderr, "\nPastikan PostgreSQL berjalan. Env yang digunakan:\n")
		fmt.Fprintf(os.Stderr, "  DB_HOST=%s  DB_PORT=%s  DB_USER=%s  DB_NAME=%s\n\n",
			envOr("DB_HOST", "localhost"),
			envOr("DB_PORT", "5432"),
			envOr("DB_USER", "postgres"),
			envOr("DB_NAME", "exchange"),
		)
		os.Exit(1)
	}

	r := NewRepo(DB)
	e := NewEngine(r)
	app := NewApp(r, e)

	p := tea.NewProgram(
		app,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

