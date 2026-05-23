# BEI Exchange Simulator

Terminal UI simulator untuk Bursa Efek Indonesia (BEI), dibangun dengan [Bubble Tea](https://github.com/charmbracelet/bubbletea) dan PostgreSQL.

```
██████╗ ███████╗██╗    ███████╗██╗  ██╗ ██████╗██╗  ██╗ █████╗ ███╗   ██╗ ██████╗ ███████╗
██╔══██╗██╔════╝██║    ██╔════╝╚██╗██╔╝██╔════╝██║  ██║██╔══██╗████╗  ██║██╔════╝ ██╔════╝
██████╔╝█████╗  ██║    █████╗   ╚███╔╝ ██║     ███████║███████║██╔██╗ ██║██║  ███╗█████╗
██╔══██╗██╔══╝  ██║    ██╔══╝   ██╔██╗ ██║     ██╔══██║██╔══██║██║╚██╗██║██║   ██║██╔══╝
██████╔╝███████╗██║    ███████╗██╔╝ ██╗╚██████╗██║  ██║██║  ██║██║ ╚████║╚██████╔╝███████╗
╚═════╝ ╚══════╝╚═╝    ╚══════╝╚═╝  ╚═╝ ╚═════╝╚═╝  ╚═╝╚═╝  ╚═╝╚═╝  ╚═══╝ ╚═════╝ ╚══════╝
```

## Quick Start

```bash
# 1. Jalankan PostgreSQL
docker compose up -d postgres

# 2. Install dependencies
go mod download

# 3. Jalankan aplikasi
go run .

# atau pakai Makefile
make run
```

Setelah aplikasi berjalan, tekan **S** untuk mengisi data demo (26 saham BEI + 4 trader).

## Fitur

| Tab | Shortcut | Deskripsi |
|-----|----------|-----------|
| Market | `1` | Real-time market watch — auto-refresh 3 detik |
| Order Book | `2` | Bid/ask depth dengan visualisasi bar |
| Trade | `3` | Form order multi-langkah (Limit/Market) |
| Portfolio | `4` | Portofolio + P/L per saham |
| Orders | `5` | Riwayat order + cancel |
| Traders | `6` | Daftar semua trader |

## Keyboard Shortcuts

```
1-6        Pindah tab
↑↓ / k/j   Navigasi
⏎           Pilih / konfirmasi
L           Login (pilih trader)
o           Logout
b           Buy saham terpilih
s           Sell saham terpilih
S           Seed data demo
r           Refresh data
c           Cancel order (di tab Orders)
q           Keluar
```

## Arsitektur

```
app.go            ← Bubble Tea UI (styles, app model, views)
db.go             ← koneksi PostgreSQL
engine.go         ← matching engine (price-time priority)
main.go           ← entry point
model.go          ← domain types (Stock, Order, Trade, ...)
repo.go           ← repository layer (semua query SQL)
seed.go           ← data demo BEI blue-chips
styles.go         ← Bubble Tea UI styles
schema.sql        ← DDL idempotent
compose.yaml      ← Docker Compose configuration
Makefile          ← Command shortcuts
```

## Environment Variables

| Variable | Default | Keterangan |
|----------|---------|------------|
| `DB_HOST` | `localhost` | Host PostgreSQL |
| `DB_PORT` | `5432` | Port PostgreSQL |
| `DB_USER` | `postgres` | Username |
| `DB_PASS` | `postgres` | Password |
| `DB_NAME` | `exchange` | Nama database |

## Konsep Matching Engine

- **Order Book** — State in-memory di PostgreSQL; Bids diurutkan harga turun, Asks harga naik
- **Price-Time Priority** — Order dengan harga terbaik dieksekusi lebih dulu; jika sama, yang lebih lama masuk didahulukan
- **Limit Order** — Masuk antrean, tunggu counterpart cocok
- **Market Order** — Eksekusi instan di harga terbaik yang tersedia, bisa partial fill
- **Atomik** — Setiap match dalam satu database transaction (ACID)

## Saham yang Tersedia (Seed)

BBCA, BBRI, BMRI, BBNI, BRIS, TLKM, EXCL, ISAT, ASII, ADRO, PTBA, PGAS, MEDC, UNVR, ICBP, INDF, HMSP, MYOR, BSDE, SMRA, JSMR, GOTO, BUKA, DMMX, SMGR, INTP

## Requirements

- Go 1.21+
- Docker & Docker Compose (untuk PostgreSQL)
