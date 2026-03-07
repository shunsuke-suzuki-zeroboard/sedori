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

// RakutenScraper fetches products from Rakuten Ichiba API.
type RakutenScraper struct {
	applicationID string
	client        *http.Client
}

func NewRakutenScraper(applicationID string) *RakutenScraper {
	return &RakutenScraper{
		applicationID: applicationID,
		client:        &http.Client{Timeout: 10 * time.Second},
	}
}

func (r *RakutenScraper) Site() model.Site {
	return model.SiteRakuten
}

type rakutenResponse struct {
	Items []struct {
		Item struct {
			ItemName     string `json:"itemName"`
			ItemPrice    int    `json:"itemPrice"`
			ItemURL      string `json:"itemUrl"`
			ImageURL     string `json:"mediumImageUrls"`
			Availability int    `json:"availability"`
		} `json:"Item"`
	} `json:"Items"`
}

func (r *RakutenScraper) Search(ctx context.Context, keyword string) ([]model.Product, error) {
	u := fmt.Sprintf(
		"https://app.rakuten.co.jp/services/api/IchibaItem/Search/20170706?applicationId=%s&keyword=%s&hits=30&sort=%%2BitemPrice",
		r.applicationID,
		url.QueryEscape(keyword),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("rakuten: failed to create request: %w", err)
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("rakuten: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("rakuten: unexpected status code: %d", resp.StatusCode)
	}

	var result rakutenResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("rakuten: failed to decode response: %w", err)
	}

	products := make([]model.Product, 0, len(result.Items))
	for _, item := range result.Items {
		products = append(products, model.Product{
			Title:     item.Item.ItemName,
			Price:     item.Item.ItemPrice,
			URL:       item.Item.ItemURL,
			Site:      model.SiteRakuten,
			ImageURL:  item.Item.ImageURL,
			Condition: "new",
			FetchedAt: time.Now(),
		})
	}

	return products, nil
}
