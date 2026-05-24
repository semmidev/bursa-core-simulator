# BursaCore ⚡

BursaCore (sebelumnya BEI Exchange Simulator) adalah simulator **Matching Engine Bursa Saham** yang ditulis menggunakan bahasa Go. Proyek ini mendemonstrasikan mekanisme perdagangan saham real-time berbasis *Price-Time Priority*, lengkap dengan antarmuka web interaktif yang cantik, cepat, dan responsif.

![BursaCore](https://img.shields.io/badge/Status-Active-brightgreen.svg) ![Go Version](https://img.shields.io/badge/Go-1.21%2B-blue.svg) ![PostgreSQL](https://img.shields.io/badge/Database-PostgreSQL-336791.svg)

---

## Fitur Utama

- **Matching Engine Super Cepat**: Implementasi *Price-Time Priority* klasik yang langsung mempertemukan order Beli dan Jual secara *atomic*.
- **Aturan ARA & ARB Otomatis**: Dilengkapi dengan hard-coded validation rules bursa lokal:
  - Harga Rp 50 - Rp 200: Max naik/turun 35% sehari.
  - Harga Rp 200 - Rp 5.000: Max naik/turun 25% sehari.
  - Harga > Rp 5.000: Max naik/turun 20% sehari.
- **Real-Time Data Streaming**: Menggunakan WebSockets untuk *auto-refresh* Order Book, Daftar Saham, Portfolio, dan Notifikasi secara *real-time* tanpa perlu reload manual.
- **Robust Order Execution**:
  - `LIMIT Order` dengan reservasi dana yang akurat.
  - `MARKET Order` dengan mekanisme *Fill-and-Kill* yang langsung melakukan *refund* jika likuiditas tidak mencukupi (termasuk partial-fills otomatis).
- **Desain UI Brutalist Premium**: Interface web yang clean, kontras tinggi (Light Mode), dan estetik. Dirancang sedemikian rupa untuk menonjolkan fungsi (Form follows Function) tanpa mengorbankan _visual elegance_. Menggunakan *Space Mono* dan *Inter* untuk kejelasan data finansial.
- **Auto-Seeded Database**: Data awal (26 saham *Blue-Chip* dan 4 trader simulator) disuntikkan secara otomatis via skema DDL ke PostgreSQL.

---

## Quick Start

1. **Jalankan PostgreSQL Database**
   ```bash
   docker compose up -d postgres
   ```

2. **Install Dependencies**
   ```bash
   go mod download
   ```

3. **Jalankan Web Server**
   ```bash
   go run .
   ```

4. **Buka Browser**
   Buka `http://localhost:8080`. Data awal sudah otomatis di-seed. Gunakan username `budi`, `siti`, `agus`, atau `dewi` untuk mencoba *login* dan *trading*.

---

## Arsitektur Aplikasi

```text
bursa-core/
├── main.go           # Entry point aplikasi
├── server.go         # Konfigurasi HTTP server + WebSocket hub + Routing
├── db.go             # Manajemen koneksi ke PostgreSQL
├── engine.go         # Otak aplikasi: Matching Engine (Price-Time Priority)
├── model.go          # Definisi struktur data (Domain types)
├── repo.go           # Repository pattern untuk eksekusi SQL queries
├── styles.go         # Helpers untuk number & currency formatting (Rupiah)
├── schema.sql        # Database schema DDL & Automatic Seeding mechanism
├── templates/        # Kumpulan file HTML (Go html/template) dengan Brutalist UI
└── static/           # Asset static (style.css, dll)
```

---

## WebSocket Channels

Server otomatis melakukan *broadcasting* event ke semua klien yang terhubung secara realtime:

| Event Name | Tipe Data | Deskripsi |
|---|---|---|
| `stocks` | `[]Stock` | Broadcast pembaruan seluruh tabel harga saham. |
| `trades:{ticker}` | `[]Trade` | Daftar transaksi sukses terakhir untuk suatu ticker. |
| `orderbook:{ticker}` | `*OrderBook` | Pembaruan susunan *Bid/Ask* kedalaman order book. |
| `trader:{id}` | `*Trader` | Sinkronisasi mutasi kas/portofolio setelah order/match. |

---

## Routes & Endpoints

Aplikasi ini menggunakan mode render *Server-Side Rendering* (SSR) murni dengan bantuan WebSocket untuk dinamisasi elemen.

| Method | Path | Deskripsi |
|---|---|---|
| `GET` | `/` | Redirects to `/market` |
| `GET` | `/market` | Live Market Watch |
| `GET` | `/orderbook` | Trade interface & Order Book depth |
| `GET` | `/portfolio` | Dashboard portofolio pribadi & P/L |
| `GET` | `/orders` | Riwayat order terbuka, sukses, dan dibatalkan |
| `GET` | `/traders` | Daftar simulasi trader (Login interface) |
| `POST` | `/order/submit`| Entry-point untuk Matching Engine memproses order baru |
| `POST` | `/order/cancel`| Tarik antrean order dari Engine |
| `POST` | `/login` / `/logout` | Manajemen Session Cookies |

---

## Design System

BursaCore dibangun dengan pedoman desain **Light Brutalism**:
- Latar krem redup (paper-like) `var(--bg-color)` untuk mereduksi ketegangan mata dengan *radial-gradient* dot pattern.
- Sudut elemen ditekuk tajam menggunakan batas _solid_ `3px` hitam (tanpa blur shadow konvensional).
- Warna aksen cerah: **Hijau Neon** (`#00e676`) dan **Merah Tajam** (`#ff3d00`) untuk indikator *up/down* yang mudah dilihat.
- Font tipografi menggunakan `Inter` untuk bacaan umum dan `Space Mono` untuk seluruh presisi data finansial.
