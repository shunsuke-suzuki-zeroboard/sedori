package notifier

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shunsuke-suzuki-zeroboard/sedori/internal/domain"
)

func TestNotify_Success(t *testing.T) {
	var received lineMessage
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("unexpected Authorization header: %s", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("unexpected Content-Type: %s", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ln := newTestLINENotifier("test-token", "U1234", srv.URL)

	diffs := []domain.PriceDiff{
		{
			Keyword: "テスト商品",
			BuyFrom: domain.Product{
				Title: "商品A",
				Price: 1000,
				URL:   "https://example.com/buy",
				Site:  domain.SiteMercari,
			},
			SellAt: domain.Product{
				Title: "商品A",
				Price: 2000,
				URL:   "https://example.com/sell",
				Site:  domain.SiteAmazon,
			},
			PriceDiff:  1000,
			ProfitRate: 100.0,
		},
	}

	if err := ln.Notify(context.Background(), diffs); err != nil {
		t.Fatalf("Notify failed: %v", err)
	}

	if received.To != "U1234" {
		t.Errorf("expected to=U1234, got %s", received.To)
	}
	if len(received.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(received.Messages))
	}
}

func TestNotify_EmptyDiffs(t *testing.T) {
	ln := NewLINENotifier("token", "user")

	if err := ln.Notify(context.Background(), nil); err != nil {
		t.Fatalf("Notify with empty diffs should not error: %v", err)
	}
}

func TestNotify_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"message":"Invalid channel access token"}`))
	}))
	defer srv.Close()

	ln := newTestLINENotifier("bad-token", "U1234", srv.URL)

	diffs := []domain.PriceDiff{
		{
			Keyword: "テスト",
			BuyFrom: domain.Product{Price: 100, Site: domain.SiteMercari},
			SellAt:  domain.Product{Price: 200, Site: domain.SiteAmazon},
		},
	}

	err := ln.Notify(context.Background(), diffs)
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
}

func TestSendText_Success(t *testing.T) {
	var received lineMessage
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&received)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ln := newTestLINENotifier("token", "U5678", srv.URL)

	if err := ln.SendText(context.Background(), "テストメッセージ"); err != nil {
		t.Fatalf("SendText failed: %v", err)
	}

	if received.To != "U5678" {
		t.Errorf("expected to=U5678, got %s", received.To)
	}
}

func TestSendText_ContextCanceled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ln := newTestLINENotifier("token", "U1234", srv.URL)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := ln.SendText(ctx, "test")
	if err == nil {
		t.Fatal("expected error for canceled context")
	}
}

func TestFormatMessage(t *testing.T) {
	ln := NewLINENotifier("token", "user")

	diffs := []domain.PriceDiff{
		{
			Keyword: "Switch",
			BuyFrom: domain.Product{
				Price: 30000,
				URL:   "https://example.com/buy",
				Site:  domain.SiteMercari,
			},
			SellAt: domain.Product{
				Price: 35000,
				URL:   "https://example.com/sell",
				Site:  domain.SiteAmazon,
			},
			PriceDiff:  5000,
			ProfitRate: 16.7,
		},
	}

	msg := ln.formatMessage(diffs)

	checks := []string{
		"せどりチャンス発見",
		"Switch",
		"メルカリ",
		"Amazon",
		"¥30000",
		"¥35000",
		"¥5000",
		"16.7%",
	}
	for _, want := range checks {
		if !contains(msg, want) {
			t.Errorf("formatMessage missing %q in output:\n%s", want, msg)
		}
	}
}

func TestSiteLabel(t *testing.T) {
	tests := []struct {
		site domain.Site
		want string
	}{
		{domain.SiteAmazon, "Amazon"},
		{domain.SiteRakuten, "楽天市場"},
		{domain.SiteMercari, "メルカリ"},
		{domain.Site("unknown"), "unknown"},
	}
	for _, tt := range tests {
		if got := siteLabel(tt.site); got != tt.want {
			t.Errorf("siteLabel(%s) = %s, want %s", tt.site, got, tt.want)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// newTestLINENotifier creates a LINENotifier that sends to a test server instead of the real LINE API.
func newTestLINENotifier(token, userID, testURL string) *LINENotifier {
	ln := NewLINENotifier(token, userID)
	ln.baseURL = testURL
	return ln
}
