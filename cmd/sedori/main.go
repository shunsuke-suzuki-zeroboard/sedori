package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/shunsuke-suzuki-zeroboard/sedori/internal/adapter/notifier"
	"github.com/shunsuke-suzuki-zeroboard/sedori/internal/adapter/scraper"
	"github.com/shunsuke-suzuki-zeroboard/sedori/internal/domain"
	"github.com/shunsuke-suzuki-zeroboard/sedori/internal/infrastructure/config"
	"github.com/shunsuke-suzuki-zeroboard/sedori/internal/usecase"
)

func main() {
	configPath := flag.String("config", "config.json", "path to config file")
	once := flag.Bool("once", false, "run once and exit (no periodic monitoring)")
	testLine := flag.Bool("test-line", false, "send a test notification to LINE and exit")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	lineNotifier := notifier.NewLINENotifier(cfg.LINE.ChannelAccessToken, cfg.LINE.UserID)

	if *testLine {
		runTestLine(lineNotifier)
		return
	}

	searchers := buildSearchers(cfg)
	if len(searchers) == 0 {
		log.Fatal("No scrapers enabled. Enable at least one site in config.")
	}

	uc := usecase.NewArbitrageUseCase(
		searchers,
		lineNotifier,
		cfg.Keywords,
		cfg.MinPriceDiff,
		cfg.MinProfitRate,
	)

	log.Printf("Starting sedori monitor with %d scrapers, interval=%dm", len(searchers), cfg.IntervalMinutes)
	log.Printf("Keywords: %v", cfg.Keywords)
	log.Printf("Min profit rate: %.1f%%, Min price diff: ¥%d", cfg.MinProfitRate, cfg.MinPriceDiff)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("Shutting down...")
		cancel()
	}()

	// Run immediately
	if err := uc.Execute(ctx); err != nil {
		log.Printf("Error in search cycle: %v", err)
	}

	if *once {
		return
	}

	// Periodic monitoring
	ticker := time.NewTicker(time.Duration(cfg.IntervalMinutes) * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := uc.Execute(ctx); err != nil {
				log.Printf("Error in search cycle: %v", err)
			}
		}
	}
}

func runTestLine(ln *notifier.LINENotifier) {
	log.Println("Sending test notification to LINE...")

	// 1) Simple text test
	if err := ln.SendText("🔔 せどり通知ツール: テスト送信です。このメッセージが届いていれば設定は正常です。"); err != nil {
		log.Fatalf("Test failed: %v", err)
	}
	log.Println("Simple text test: OK")

	// 2) Sample arbitrage notification
	diffs := []domain.PriceDiff{
		{
			Keyword: "Nintendo Switch",
			BuyFrom: domain.Product{
				Title: "Nintendo Switch 本体",
				Price: 29800,
				URL:   "https://example.com/buy",
				Site:  domain.SiteMercari,
			},
			SellAt: domain.Product{
				Title: "Nintendo Switch 本体",
				Price: 35000,
				URL:   "https://example.com/sell",
				Site:  domain.SiteAmazon,
			},
			PriceDiff:  5200,
			ProfitRate: 17.4,
		},
	}
	if err := ln.Notify(diffs); err != nil {
		log.Fatalf("Notification test failed: %v", err)
	}
	log.Println("Arbitrage notification test: OK")
	log.Println("All LINE tests passed!")
}

func buildSearchers(cfg *config.Config) []domain.ProductSearcher {
	var searchers []domain.ProductSearcher

	if cfg.Amazon.Enabled {
		searchers = append(searchers, scraper.NewAmazonScraper(
			cfg.Amazon.AccessKey,
			cfg.Amazon.SecretKey,
			cfg.Amazon.PartnerTag,
			cfg.Amazon.Marketplace,
		))
		log.Println("Amazon scraper enabled")
	}

	if cfg.Rakuten.Enabled {
		searchers = append(searchers, scraper.NewRakutenScraper(cfg.Rakuten.ApplicationID))
		log.Println("Rakuten scraper enabled")
	}

	if cfg.Mercari.Enabled {
		searchers = append(searchers, scraper.NewMercariScraper())
		log.Println("Mercari scraper enabled")
	}

	return searchers
}
