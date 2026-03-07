package scraper

import (
	"context"

	"github.com/shunsuke-suzuki-zeroboard/sedori/internal/model"
)

// Scraper is the interface for fetching product listings from an EC site.
type Scraper interface {
	// Search searches for products matching the given keyword.
	Search(ctx context.Context, keyword string) ([]model.Product, error)
	// Site returns which site this scraper handles.
	Site() model.Site
}
