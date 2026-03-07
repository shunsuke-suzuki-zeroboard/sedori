package scraper

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/shunsuke-suzuki-zeroboard/sedori/internal/model"
)

// MercariScraper fetches products from Mercari's public search API.
type MercariScraper struct {
	client *http.Client
}

func NewMercariScraper() *MercariScraper {
	return &MercariScraper{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (m *MercariScraper) Site() model.Site {
	return model.SiteMercari
}

type mercariResponse struct {
	Items []struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		Price     int    `json:"price"`
		Status    string `json:"status"`
		Thumbnails []string `json:"thumbnails"`
		ItemCondition struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"item_condition"`
	} `json:"items"`
}

func (m *MercariScraper) Search(ctx context.Context, keyword string) ([]model.Product, error) {
	u := fmt.Sprintf(
		"https://api.mercari.jp/v2/entities:search?keyword=%s&limit=30&status=on_sale&sort=price&order=asc",
		url.QueryEscape(keyword),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("mercari: failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; SedoriBot/1.0)")

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("mercari: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("mercari: unexpected status code: %d", resp.StatusCode)
	}

	var result mercariResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("mercari: failed to decode response: %w", err)
	}

	products := make([]model.Product, 0, len(result.Items))
	for _, item := range result.Items {
		imageURL := ""
		if len(item.Thumbnails) > 0 {
			imageURL = item.Thumbnails[0]
		}
		products = append(products, model.Product{
			Title:     item.Name,
			Price:     item.Price,
			URL:       fmt.Sprintf("https://jp.mercari.com/item/%s", item.ID),
			Site:      model.SiteMercari,
			ImageURL:  imageURL,
			Condition: item.ItemCondition.Name,
			FetchedAt: time.Now(),
		})
	}

	return products, nil
}
