package usecase

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/shunsuke-suzuki-zeroboard/sedori/internal/domain"
)

// ArbitrageUseCase orchestrates searching products across sites,
// comparing prices, and notifying about arbitrage opportunities.
type ArbitrageUseCase struct {
	searchers []domain.ProductSearcher
	notifier  domain.Notifier
	keywords  []string
	minDiff   int
	minRate   float64
}

// NewArbitrageUseCase creates a new ArbitrageUseCase.
func NewArbitrageUseCase(
	searchers []domain.ProductSearcher,
	notifier domain.Notifier,
	keywords []string,
	minDiff int,
	minRate float64,
) *ArbitrageUseCase {
	return &ArbitrageUseCase{
		searchers: searchers,
		notifier:  notifier,
		keywords:  keywords,
		minDiff:   minDiff,
		minRate:   minRate,
	}
}

// Execute runs a single search-compare-notify cycle.
func (u *ArbitrageUseCase) Execute(ctx context.Context) error {
	log.Println("Starting search cycle...")

	productsByKeyword := u.searchAll(ctx)

	diffs := u.findArbitrage(productsByKeyword)

	if len(diffs) == 0 {
		log.Println("No arbitrage opportunities found in this cycle.")
		return nil
	}

	log.Printf("Found %d arbitrage opportunities!", len(diffs))
	for _, d := range diffs {
		fmt.Printf("  [%s] Buy: %s ¥%d -> Sell: %s ¥%d (diff: ¥%d, %.1f%%)\n",
			d.Keyword, d.BuyFrom.Site, d.BuyFrom.Price, d.SellAt.Site, d.SellAt.Price, d.PriceDiff, d.ProfitRate)
	}

	if err := u.notifier.Notify(diffs); err != nil {
		return fmt.Errorf("usecase: failed to notify: %w", err)
	}
	log.Println("Notification sent successfully!")

	return nil
}

// searchAll searches all keywords on all sites concurrently.
func (u *ArbitrageUseCase) searchAll(ctx context.Context) map[string][]domain.Product {
	productsByKeyword := make(map[string][]domain.Product)
	var mu sync.Mutex

	var wg sync.WaitGroup
	for _, keyword := range u.keywords {
		for _, s := range u.searchers {
			wg.Add(1)
			go func(kw string, sc domain.ProductSearcher) {
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

	return productsByKeyword
}

// findArbitrage finds price arbitrage opportunities across products from different sites.
func (u *ArbitrageUseCase) findArbitrage(productsByKeyword map[string][]domain.Product) []domain.PriceDiff {
	var diffs []domain.PriceDiff

	for keyword, products := range productsByKeyword {
		// Group by site
		bySite := make(map[domain.Site][]domain.Product)
		for _, p := range products {
			bySite[p.Site] = append(bySite[p.Site], p)
		}

		// Find cheapest product per site
		cheapest := make(map[domain.Site]domain.Product)
		for site, siteProducts := range bySite {
			min := siteProducts[0]
			for _, p := range siteProducts[1:] {
				if p.Price < min.Price {
					min = p
				}
			}
			cheapest[site] = min
		}

		// Find most expensive product per site (potential sell price)
		mostExpensive := make(map[domain.Site]domain.Product)
		for site, siteProducts := range bySite {
			max := siteProducts[0]
			for _, p := range siteProducts[1:] {
				if p.Price > max.Price {
					max = p
				}
			}
			mostExpensive[site] = max
		}

		// Compare all site pairs for arbitrage
		for buySite, buyProduct := range cheapest {
			for sellSite, sellProduct := range mostExpensive {
				if buySite == sellSite {
					continue
				}

				diff := sellProduct.Price - buyProduct.Price
				if diff < u.minDiff {
					continue
				}

				rate := float64(diff) / float64(buyProduct.Price) * 100
				if rate < u.minRate {
					continue
				}

				diffs = append(diffs, domain.PriceDiff{
					Keyword:    keyword,
					BuyFrom:    buyProduct,
					SellAt:     sellProduct,
					PriceDiff:  diff,
					ProfitRate: rate,
				})
			}
		}
	}

	return diffs
}
