package main

import (
	"fmt"
	"time"
)

func SeedStocks(r *Repo) error {
	listing := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	entries := []struct {
		Ticker      string
		Company     string
		Sector      string
		LastPrice   int64
		TotalShares int64
	}{
		{"BBCA", "Bank Central Asia Tbk", "Perbankan", 10_300, 123_306_376_986},
		{"BBRI", "Bank Rakyat Indonesia (Persero) Tbk", "Perbankan", 4_840, 163_714_439_506},
		{"BMRI", "Bank Mandiri (Persero) Tbk", "Perbankan", 7_025, 92_538_238_792},
		{"BBNI", "Bank Negara Indonesia (Persero) Tbk", "Perbankan", 5_350, 18_648_656_458},
		{"BRIS", "Bank Syariah Indonesia Tbk", "Perbankan", 2_870, 28_007_809_930},
		{"TLKM", "Telkom Indonesia (Persero) Tbk", "Telekomunikasi", 3_180, 99_062_216_600},
		{"EXCL", "XL Axiata Tbk", "Telekomunikasi", 2_310, 10_687_960_423},
		{"ISAT", "Indosat Tbk", "Telekomunikasi", 2_810, 8_062_692_952},
		{"ASII", "Astra International Tbk", "Industri Dasar", 4_630, 40_483_553_140},
		{"ADRO", "Adaro Energy Indonesia Tbk", "Pertambangan", 2_650, 31_985_962_000},
		{"PTBA", "Bukit Asam (Persero) Tbk", "Pertambangan", 3_200, 11_520_659_250},
		{"PGAS", "Perusahaan Gas Negara Tbk", "Energi", 1_590, 24_241_508_196},
		{"MEDC", "Medco Energi Internasional Tbk", "Energi", 1_350, 9_679_447_767},
		{"UNVR", "Unilever Indonesia Tbk", "Konsumer", 2_460, 38_150_000_000},
		{"ICBP", "Indofood CBP Sukses Makmur Tbk", "Konsumer", 10_725, 11_661_908_000},
		{"INDF", "Indofood Sukses Makmur Tbk", "Konsumer", 6_800, 8_780_426_500},
		{"HMSP", "HM Sampoerna Tbk", "Konsumer", 820, 116_318_076_900},
		{"MYOR", "Mayora Indah Tbk", "Konsumer", 2_640, 22_358_699_725},
		{"BSDE", "Bumi Serpong Damai Tbk", "Properti", 1_185, 19_246_696_192},
		{"SMRA", "Summarecon Agung Tbk", "Properti", 615, 14_426_781_680},
		{"JSMR", "Jasa Marga (Persero) Tbk", "Infrastruktur", 4_400, 6_800_000_000},
		{"GOTO", "GoTo Gojek Tokopedia Tbk", "Teknologi", 73, 1_190_684_447_928},
		{"BUKA", "Bukalapak.com Tbk", "Teknologi", 132, 104_081_765_731},
		{"DMMX", "Digital Mediatama Maxima Tbk", "Teknologi", 710, 4_285_600_000},
		{"SMGR", "Semen Indonesia (Persero) Tbk", "Material", 4_600, 5_931_520_000},
		{"INTP", "Indocement Tunggal Prakarsa Tbk", "Material", 9_275, 3_681_231_699},
	}

	for _, e := range entries {
		s := Stock{
			Ticker:      e.Ticker,
			CompanyName: e.Company,
			Sector:      e.Sector,
			ListingDate: listing,
			TotalShares: e.TotalShares,
			LastPrice:   e.LastPrice,
			PrevClose:   e.LastPrice - e.LastPrice/20,
		}
		if err := r.UpsertStock(s); err != nil {
			return fmt.Errorf("seed %s: %w", e.Ticker, err)
		}
	}
	return nil
}

type seedPort struct {
	Ticker string
	Qty    int64
	Price  int64
}

func SeedTraders(r *Repo) error {
	demos := []struct {
		Username string
		Name     string
		Cash     int64
		Port     []seedPort
	}{
		{
			"budi", "Budi Santoso", 500_000_000,
			[]seedPort{
				{"BBCA", 100, 10000},
				{"TLKM", 500, 3100},
				{"GOTO", 5000, 65},
			},
		},
		{
			"siti", "Siti Rahayu", 250_000_000,
			[]seedPort{
				{"ASII", 200, 4500},
				{"BMRI", 150, 6800},
			},
		},
		{
			"agus", "Agus Wijaya", 1_000_000_000,
			[]seedPort{
				{"BBRI", 1000, 4800},
				{"ICBP", 50, 10500},
				{"UNVR", 200, 2400},
			},
		},
		{
			"dewi", "Dewi Kusuma", 750_000_000,
			[]seedPort{
				{"PGAS", 1000, 1500},
				{"ADRO", 300, 2600},
				{"PTBA", 200, 3100},
			},
		},
	}
	for _, d := range demos {
		existing, err := r.GetTraderByUsername(d.Username)
		if err != nil {
			return err
		}
		if existing != nil {
			continue
		}
		t := &Trader{Username: d.Username, FullName: d.Name, CashBalance: d.Cash}
		if err := r.CreateTrader(t); err != nil {
			return fmt.Errorf("seed trader %s: %w", d.Username, err)
		}

		// Seed initial portfolio
		for _, p := range d.Port {
			tx, err := r.DB.Begin()
			if err != nil {
				return err
			}
			err = r.UpsertPortfolioAdd(tx, t.ID, p.Ticker, p.Qty, p.Price)
			if err == nil {
				tx.Commit()
			} else {
				tx.Rollback()
			}
		}
	}
	return nil
}
