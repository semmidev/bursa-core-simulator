package main

import (
	"fmt"
	"os"
)

func main() {
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
	srv := NewServer(r, e)
	srv.Run()
}
