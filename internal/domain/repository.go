package domain

import "context"

// ProductSearcher searches for products on a specific EC site.
type ProductSearcher interface {
	Search(ctx context.Context, keyword string) ([]Product, error)
	Site() Site
}

// Notifier sends arbitrage opportunity notifications.
type Notifier interface {
	Notify(diffs []PriceDiff) error
}
