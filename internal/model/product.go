package model

import "time"

// Site represents an EC site.
type Site string

const (
	SiteAmazon  Site = "amazon"
	SiteRakuten Site = "rakuten"
	SiteMercari Site = "mercari"
)

// Product represents a product listing on an EC site.
type Product struct {
	Title     string
	Price     int
	URL       string
	Site      Site
	ImageURL  string
	Condition string // new, used, etc.
	FetchedAt time.Time
}

// PriceDiff represents a price arbitrage opportunity between two sites.
type PriceDiff struct {
	Keyword   string
	BuyFrom   Product
	SellAt    Product
	PriceDiff int
	ProfitRate float64 // percentage
}
