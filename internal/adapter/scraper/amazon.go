package scraper

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/shunsuke-suzuki-zeroboard/sedori/internal/domain"
)

// AmazonScraper fetches products using the Amazon Product Advertising API (PA-API 5.0).
type AmazonScraper struct {
	accessKey   string
	secretKey   string
	partnerTag  string
	marketplace string
	client      *http.Client
}

func NewAmazonScraper(accessKey, secretKey, partnerTag, marketplace string) *AmazonScraper {
	if marketplace == "" {
		marketplace = "www.amazon.co.jp"
	}
	return &AmazonScraper{
		accessKey:   accessKey,
		secretKey:   secretKey,
		partnerTag:  partnerTag,
		marketplace: marketplace,
		client:      &http.Client{Timeout: 10 * time.Second},
	}
}

func (a *AmazonScraper) Site() domain.Site {
	return domain.SiteAmazon
}

type amazonSearchRequest struct {
	Keywords    string   `json:"Keywords"`
	Resources   []string `json:"Resources"`
	PartnerTag  string   `json:"PartnerTag"`
	PartnerType string   `json:"PartnerType"`
	Marketplace string   `json:"Marketplace"`
	ItemCount   int      `json:"ItemCount"`
	SortBy      string   `json:"SortBy"`
}

type amazonSearchResponse struct {
	SearchResult struct {
		Items []struct {
			ASIN       string `json:"ASIN"`
			DetailPage string `json:"DetailPageURL"`
			ItemInfo   struct {
				Title struct {
					DisplayValue string `json:"DisplayValue"`
				} `json:"Title"`
			} `json:"ItemInfo"`
			Offers struct {
				Listings []struct {
					Price struct {
						Amount   float64 `json:"Amount"`
						Currency string  `json:"Currency"`
					} `json:"Price"`
					Condition struct {
						Value string `json:"Value"`
					} `json:"Condition"`
				} `json:"Listings"`
			} `json:"Offers"`
			Images struct {
				Primary struct {
					Medium struct {
						URL string `json:"URL"`
					} `json:"Medium"`
				} `json:"Primary"`
			} `json:"Images"`
		} `json:"Items"`
	} `json:"SearchResult"`
}

func (a *AmazonScraper) Search(ctx context.Context, keyword string) ([]domain.Product, error) {
	payload := amazonSearchRequest{
		Keywords:    keyword,
		Resources:   []string{"ItemInfo.Title", "Offers.Listings.Price", "Offers.Listings.Condition", "Images.Primary.Medium"},
		PartnerTag:  a.partnerTag,
		PartnerType: "Associates",
		Marketplace: a.marketplace,
		ItemCount:   10,
		SortBy:      "Price:LowToHigh",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("amazon: failed to marshal request: %w", err)
	}

	host := "webservices.amazon.co.jp"
	endpoint := fmt.Sprintf("https://%s/paapi5/searchitems", host)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("amazon: failed to create request: %w", err)
	}

	now := time.Now().UTC()
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	req.Header.Set("Host", host)
	req.Header.Set("X-Amz-Date", now.Format("20060102T150405Z"))
	req.Header.Set("X-Amz-Target", "com.amazon.paapi5.v1.ProductAdvertisingAPIv1.SearchItems")
	req.Header.Set("Content-Encoding", "amz-1.0")

	a.signRequest(req, body, now)

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("amazon: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("amazon: unexpected status code: %d", resp.StatusCode)
	}

	var result amazonSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("amazon: failed to decode response: %w", err)
	}

	products := make([]domain.Product, 0, len(result.SearchResult.Items))
	for _, item := range result.SearchResult.Items {
		if len(item.Offers.Listings) == 0 {
			continue
		}
		listing := item.Offers.Listings[0]
		products = append(products, domain.Product{
			Title:     item.ItemInfo.Title.DisplayValue,
			Price:     int(listing.Price.Amount),
			URL:       item.DetailPage,
			Site:      domain.SiteAmazon,
			ImageURL:  item.Images.Primary.Medium.URL,
			Condition: listing.Condition.Value,
			FetchedAt: time.Now(),
		})
	}

	return products, nil
}

func (a *AmazonScraper) signRequest(req *http.Request, payload []byte, now time.Time) {
	dateStamp := now.Format("20060102")
	region := "us-west-2"
	service := "ProductAdvertisingAPI"

	credentialScope := fmt.Sprintf("%s/%s/%s/aws4_request", dateStamp, region, service)

	mac := hmac.New(sha256.New, []byte("AWS4"+a.secretKey))
	mac.Write([]byte(dateStamp))
	dateKey := mac.Sum(nil)

	mac = hmac.New(sha256.New, dateKey)
	mac.Write([]byte(region))
	regionKey := mac.Sum(nil)

	mac = hmac.New(sha256.New, regionKey)
	mac.Write([]byte(service))
	serviceKey := mac.Sum(nil)

	mac = hmac.New(sha256.New, serviceKey)
	mac.Write([]byte("aws4_request"))
	signingKey := mac.Sum(nil)

	payloadHash := sha256.Sum256(payload)
	payloadHashHex := hex.EncodeToString(payloadHash[:])

	canonicalHeaders := fmt.Sprintf("content-encoding:amz-1.0\ncontent-type:application/json; charset=UTF-8\nhost:%s\nx-amz-date:%s\nx-amz-target:com.amazon.paapi5.v1.ProductAdvertisingAPIv1.SearchItems\n",
		req.Host, req.Header.Get("X-Amz-Date"))
	signedHeaders := "content-encoding;content-type;host;x-amz-date;x-amz-target"

	canonicalRequest := fmt.Sprintf("POST\n/paapi5/searchitems\n\n%s\n%s\n%s",
		canonicalHeaders, signedHeaders, payloadHashHex)

	canonicalRequestHash := sha256.Sum256([]byte(canonicalRequest))
	stringToSign := fmt.Sprintf("AWS4-HMAC-SHA256\n%s\n%s\n%s",
		now.Format("20060102T150405Z"), credentialScope, hex.EncodeToString(canonicalRequestHash[:]))

	mac = hmac.New(sha256.New, signingKey)
	mac.Write([]byte(stringToSign))
	signature := hex.EncodeToString(mac.Sum(nil))

	authHeader := fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		a.accessKey, credentialScope, signedHeaders, signature)
	req.Header.Set("Authorization", authHeader)
}
