package comparator

import (
	"github.com/shunsuke-suzuki-zeroboard/sedori/internal/model"
)

// Compare finds price arbitrage opportunities across products from different sites.
// It looks for cases where a product can be bought cheaply on one site and sold for more on another.
func Compare(productsByKeyword map[string][]model.Product, minPriceDiff int, minProfitRate float64) []model.PriceDiff {
	var diffs []model.PriceDiff

	for keyword, products := range productsByKeyword {
		// Group by site
		bySite := make(map[model.Site][]model.Product)
		for _, p := range products {
			bySite[p.Site] = append(bySite[p.Site], p)
		}

		// Find cheapest product per site
		cheapest := make(map[model.Site]model.Product)
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
		mostExpensive := make(map[model.Site]model.Product)
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
		sites := []model.Site{model.SiteAmazon, model.SiteRakuten, model.SiteMercari}
		for _, buySite := range sites {
			buyProduct, hasBuy := cheapest[buySite]
			if !hasBuy {
				continue
			}
			for _, sellSite := range sites {
				if buySite == sellSite {
					continue
				}
				sellProduct, hasSell := mostExpensive[sellSite]
				if !hasSell {
					continue
				}

				diff := sellProduct.Price - buyProduct.Price
				if diff < minPriceDiff {
					continue
				}

				rate := float64(diff) / float64(buyProduct.Price) * 100
				if rate < minProfitRate {
					continue
				}

				diffs = append(diffs, model.PriceDiff{
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
