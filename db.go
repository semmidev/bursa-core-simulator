package main

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func Connect() error {
	dsn := dsn()
	d, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	d.SetMaxOpenConns(25)
	d.SetMaxIdleConns(10)
	d.SetConnMaxLifetime(5 * time.Minute)
	if err := d.Ping(); err != nil {
		return fmt.Errorf("ping: %w", err)
	}
	DB = d
	return nil
}

func dsn() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		envOr("DB_HOST", "localhost"),
		envOr("DB_PORT", "5432"),
		envOr("DB_USER", "postgres"),
		envOr("DB_PASS", "postgres"),
		envOr("DB_NAME", "exchange"),
		envOr("DB_SSLMODE", "disable"),
	)
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

