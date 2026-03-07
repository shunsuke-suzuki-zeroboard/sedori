package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/shunsuke-suzuki-zeroboard/sedori/internal/comparator"
	"github.com/shunsuke-suzuki-zeroboard/sedori/internal/config"
	"github.com/shunsuke-suzuki-zeroboard/sedori/internal/model"
	"github.com/shunsuke-suzuki-zeroboard/sedori/internal/notifier"
	"github.com/shunsuke-suzuki-zeroboard/sedori/internal/scraper"
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

	scrapers := buildScrapers(cfg)
	if len(scrapers) == 0 {
		log.Fatal("No scrapers enabled. Enable at least one site in config.")
	}

	log.Printf("Starting sedori monitor with %d scrapers, interval=%dm", len(scrapers), cfg.IntervalMinutes)
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
	run(ctx, cfg, scrapers, lineNotifier)

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
			run(ctx, cfg, scrapers, lineNotifier)
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
	diffs := []model.PriceDiff{
		{
			Keyword: "Nintendo Switch",
			BuyFrom: model.Product{
				Title: "Nintendo Switch 本体",
				Price: 29800,
				URL:   "https://example.com/buy",
				Site:  model.SiteMercari,
			},
			SellAt: model.Product{
				Title: "Nintendo Switch 本体",
				Price: 35000,
				URL:   "https://example.com/sell",
				Site:  model.SiteAmazon,
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

func buildScrapers(cfg *config.Config) []scraper.Scraper {
	var scrapers []scraper.Scraper

	if cfg.Amazon.Enabled {
		scrapers = append(scrapers, scraper.NewAmazonScraper(
			cfg.Amazon.AccessKey,
			cfg.Amazon.SecretKey,
			cfg.Amazon.PartnerTag,
			cfg.Amazon.Marketplace,
		))
		log.Println("Amazon scraper enabled")
	}

	if cfg.Rakuten.Enabled {
		scrapers = append(scrapers, scraper.NewRakutenScraper(cfg.Rakuten.ApplicationID))
		log.Println("Rakuten scraper enabled")
	}

	if cfg.Mercari.Enabled {
		scrapers = append(scrapers, scraper.NewMercariScraper())
		log.Println("Mercari scraper enabled")
	}

	return scrapers
}

func run(ctx context.Context, cfg *config.Config, scrapers []scraper.Scraper, lineNotifier *notifier.LINENotifier) {
	log.Println("Starting search cycle...")

	productsByKeyword := make(map[string][]model.Product)
	var mu sync.Mutex

	var wg sync.WaitGroup
	for _, keyword := range cfg.Keywords {
		for _, s := range scrapers {
			wg.Add(1)
			go func(kw string, sc scraper.Scraper) {
				defer wg.Done()

				products, err := sc.Search(ctx, kw)
				if err != nil {
					log.Printf("Error searching %s on %s: %v", kw, sc.Site(), err)
					return
				}

				mu.Lock()
				productsByKeyword[kw] = append(productsByKeyword[kw], products...)
				mu.Unlock()

				log.Printf("Found %d products for '%s' on %s", len(products), kw, sc.Site())
			}(keyword, s)
		}
	}
	wg.Wait()

	// Compare prices across sites
	diffs := comparator.Compare(productsByKeyword, cfg.MinPriceDiff, cfg.MinProfitRate)

	if len(diffs) == 0 {
		log.Println("No arbitrage opportunities found in this cycle.")
		return
	}

	log.Printf("Found %d arbitrage opportunities!", len(diffs))
	for _, d := range diffs {
		fmt.Printf("  [%s] Buy: %s ¥%d -> Sell: %s ¥%d (diff: ¥%d, %.1f%%)\n",
			d.Keyword, d.BuyFrom.Site, d.BuyFrom.Price, d.SellAt.Site, d.SellAt.Price, d.PriceDiff, d.ProfitRate)
	}

	// Send LINE notification
	if err := lineNotifier.Notify(diffs); err != nil {
		log.Printf("Failed to send LINE notification: %v", err)
	} else {
		log.Println("LINE notification sent successfully!")
	}
}
