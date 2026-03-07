package usecase

import (
	"context"
	"fmt"
	"testing"

	"github.com/shunsuke-suzuki-zeroboard/sedori/internal/domain"
)

// mockSearcher implements domain.ProductSearcher for testing.
type mockSearcher struct {
	site     domain.Site
	products []domain.Product
	err      error
}

func (m *mockSearcher) Search(_ context.Context, _ string) ([]domain.Product, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.products, nil
}

func (m *mockSearcher) Site() domain.Site {
	return m.site
}

// mockNotifier implements domain.Notifier for testing.
type mockNotifier struct {
	notified []domain.PriceDiff
	err      error
}

func (m *mockNotifier) Notify(_ context.Context, diffs []domain.PriceDiff) error {
	if m.err != nil {
		return m.err
	}
	m.notified = append(m.notified, diffs...)
	return nil
}

func TestExecute_FindsArbitrage(t *testing.T) {
	mercari := &mockSearcher{
		site: domain.SiteMercari,
		products: []domain.Product{
			{Title: "商品A", Price: 1000, Site: domain.SiteMercari},
		},
	}
	amazon := &mockSearcher{
		site: domain.SiteAmazon,
		products: []domain.Product{
			{Title: "商品A", Price: 2000, Site: domain.SiteAmazon},
		},
	}
	n := &mockNotifier{}

	uc := NewArbitrageUseCase(
		[]domain.ProductSearcher{mercari, amazon},
		n,
		[]string{"商品A"},
		500,  // minDiff
		10.0, // minRate
	)

	if err := uc.Execute(context.Background()); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if len(n.notified) == 0 {
		t.Fatal("expected at least 1 arbitrage notification")
	}

	found := false
	for _, d := range n.notified {
		if d.BuyFrom.Site == domain.SiteMercari && d.SellAt.Site == domain.SiteAmazon {
			if d.PriceDiff != 1000 {
				t.Errorf("expected price diff=1000, got %d", d.PriceDiff)
			}
			found = true
		}
	}
	if !found {
		t.Error("expected arbitrage from Mercari to Amazon")
	}
}

func TestExecute_NoArbitrageWhenBelowThreshold(t *testing.T) {
	mercari := &mockSearcher{
		site: domain.SiteMercari,
		products: []domain.Product{
			{Title: "商品B", Price: 1000, Site: domain.SiteMercari},
		},
	}
	amazon := &mockSearcher{
		site: domain.SiteAmazon,
		products: []domain.Product{
			{Title: "商品B", Price: 1100, Site: domain.SiteAmazon},
		},
	}
	n := &mockNotifier{}

	uc := NewArbitrageUseCase(
		[]domain.ProductSearcher{mercari, amazon},
		n,
		[]string{"商品B"},
		500,  // minDiff: 差額100円 < 500円
		10.0, // minRate
	)

	if err := uc.Execute(context.Background()); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if len(n.notified) != 0 {
		t.Errorf("expected no notifications, got %d", len(n.notified))
	}
}

func TestExecute_SearchError(t *testing.T) {
	failing := &mockSearcher{
		site: domain.SiteAmazon,
		err:  fmt.Errorf("network error"),
	}
	mercari := &mockSearcher{
		site: domain.SiteMercari,
		products: []domain.Product{
			{Title: "商品C", Price: 1000, Site: domain.SiteMercari},
		},
	}
	n := &mockNotifier{}

	uc := NewArbitrageUseCase(
		[]domain.ProductSearcher{failing, mercari},
		n,
		[]string{"商品C"},
		500,
		10.0,
	)

	// Should not fail even when one searcher errors
	if err := uc.Execute(context.Background()); err != nil {
		t.Fatalf("Execute should not fail when one searcher errors: %v", err)
	}
}

func TestExecute_NotifierError(t *testing.T) {
	mercari := &mockSearcher{
		site: domain.SiteMercari,
		products: []domain.Product{
			{Title: "商品D", Price: 1000, Site: domain.SiteMercari},
		},
	}
	amazon := &mockSearcher{
		site: domain.SiteAmazon,
		products: []domain.Product{
			{Title: "商品D", Price: 3000, Site: domain.SiteAmazon},
		},
	}
	n := &mockNotifier{err: fmt.Errorf("LINE API error")}

	uc := NewArbitrageUseCase(
		[]domain.ProductSearcher{mercari, amazon},
		n,
		[]string{"商品D"},
		500,
		10.0,
	)

	err := uc.Execute(context.Background())
	if err == nil {
		t.Fatal("expected error when notifier fails")
	}
}

func TestFindArbitrage_MultipleSites(t *testing.T) {
	uc := &ArbitrageUseCase{minDiff: 100, minRate: 5.0}

	products := map[string][]domain.Product{
		"test": {
			{Title: "A", Price: 500, Site: domain.SiteMercari},
			{Title: "B", Price: 800, Site: domain.SiteAmazon},
			{Title: "C", Price: 1200, Site: domain.SiteRakuten},
		},
	}

	diffs := uc.findArbitrage(products)

	if len(diffs) == 0 {
		t.Fatal("expected at least 1 arbitrage opportunity")
	}

	// Mercari(500) → Rakuten(1200) should be the best
	var bestDiff domain.PriceDiff
	for _, d := range diffs {
		if d.PriceDiff > bestDiff.PriceDiff {
			bestDiff = d
		}
	}

	if bestDiff.BuyFrom.Site != domain.SiteMercari {
		t.Errorf("expected buy from Mercari, got %s", bestDiff.BuyFrom.Site)
	}
	if bestDiff.SellAt.Site != domain.SiteRakuten {
		t.Errorf("expected sell at Rakuten, got %s", bestDiff.SellAt.Site)
	}
	if bestDiff.PriceDiff != 700 {
		t.Errorf("expected price diff=700, got %d", bestDiff.PriceDiff)
	}
}
