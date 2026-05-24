# BEI Exchange Simulator — Web Edition

Terminal UI yang telah diubah menjadi **web application** dengan real-time data via WebSocket.

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

# 3. Jalankan web server
go run .

# 4. Buka browser
open http://localhost:8080
```

Setelah aplikasi berjalan, klik tombol **SEED** di header untuk mengisi data demo (26 saham BEI + 4 trader).

## Fitur Web

| Section | Deskripsi |
|---------|-----------|
| **Market Watch** | Tabel saham real-time dengan flash animasi saat harga berubah |
| **Order Book** | Bid/Ask depth dengan bar visualisasi, spread info, recent trades |
| **Portfolio** | P/L per saham dengan bar indicator, summary total aset |
| **Orders** | Riwayat order dengan cancel button |
| **Traders** | Daftar trader dengan login langsung |

## Arsitektur Web

```
main.go           ← Entry point
server.go         ← HTTP server + WebSocket hub + REST API
db.go             ← Koneksi PostgreSQL
engine.go         ← Matching engine (price-time priority)
model.go          ← Domain types
repo.go           ← Repository layer (SQL queries)
seed.go           ← Data demo BEI blue-chips
styles.go         ← Number formatting helpers
templates/
  index.html      ← Single-page app (Go HTML template + Tailwind + vanilla JS)
schema.sql        ← DDL PostgreSQL
compose.yaml      ← Docker Compose
Makefile          ← Command shortcuts
```

## WebSocket Events

Server broadcast events ke semua connected clients:

| Event | Data | Trigger |
|-------|------|---------|
| `stocks` | `[]Stock` | Setiap 2 detik (auto-refresh) + setelah order match |
| `trades:{ticker}` | `[]Trade` | Setelah order match untuk ticker tersebut |
| `orderbook:{ticker}` | `*OrderBook` | Setelah order match untuk ticker tersebut |
| `trader:{id}` | `*Trader` | Setelah order submit / cancel (update cash) |

## REST API

| Method | Path | Deskripsi |
|--------|------|-----------|
| GET | `/api/stocks` | Semua saham |
| GET | `/api/orderbook?ticker=BBCA` | Order book untuk ticker |
| GET | `/api/trades?ticker=BBCA` | 20 trade terakhir |
| GET | `/api/portfolio?trader_id=UUID` | Portfolio trader |
| GET | `/api/orders?trader_id=UUID` | 50 order terakhir trader |
| GET | `/api/traders` | Semua trader |
| POST | `/api/order/submit` | Submit order baru |
| POST | `/api/order/cancel` | Batalkan order |
| POST | `/api/seed` | Seed data demo |

## Environment Variables

| Variable | Default | Keterangan |
|----------|---------|------------|
| `DB_HOST` | `localhost` | Host PostgreSQL |
| `DB_PORT` | `5432` | Port PostgreSQL |
| `DB_USER` | `postgres` | Username |
| `DB_PASS` | `postgres` | Password |
| `DB_NAME` | `exchange` | Nama database |
| `HTTP_ADDR` | `:8080` | Alamat HTTP server |

## Design System

- **Font Display**: Bebas Neue (headers, ticker, labels)
- **Font Mono**: Space Mono (data, numbers, code)
- **Warna Accent**: `#E63329` (merah brutal)
- **Background**: `#F5F0E8` (krem/paper)
- **Border**: 3-4px solid hitam dengan box-shadow offset (brutalist)

## Requirements

- Go 1.21+
- Docker & Docker Compose (untuk PostgreSQL)
