package config

import (
	"os"
	"testing"
)

func TestLoad_ValidConfig(t *testing.T) {
	content := `{
		"keywords": ["Switch", "PS5"],
		"min_profit_rate": 15.0,
		"min_price_diff": 1000,
		"interval_minutes": 60,
		"line": {
			"channel_access_token": "test-token",
			"user_id": "U1234"
		},
		"amazon": {
			"enabled": true,
			"access_key": "ak",
			"secret_key": "sk",
			"partner_tag": "tag",
			"marketplace": "www.amazon.co.jp"
		},
		"rakuten": {
			"enabled": false,
			"application_id": "app-id"
		},
		"mercari": {
			"enabled": true
		}
	}`

	f, err := os.CreateTemp("", "config-*.json")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(f.Name())

	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
	f.Close()

	cfg, err := Load(f.Name())
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(cfg.Keywords) != 2 {
		t.Errorf("expected 2 keywords, got %d", len(cfg.Keywords))
	}
	if cfg.MinProfitRate != 15.0 {
		t.Errorf("expected min_profit_rate=15.0, got %f", cfg.MinProfitRate)
	}
	if cfg.MinPriceDiff != 1000 {
		t.Errorf("expected min_price_diff=1000, got %d", cfg.MinPriceDiff)
	}
	if cfg.IntervalMinutes != 60 {
		t.Errorf("expected interval_minutes=60, got %d", cfg.IntervalMinutes)
	}
	if cfg.LINE.ChannelAccessToken != "test-token" {
		t.Errorf("expected LINE token=test-token, got %s", cfg.LINE.ChannelAccessToken)
	}
	if cfg.LINE.UserID != "U1234" {
		t.Errorf("expected LINE user_id=U1234, got %s", cfg.LINE.UserID)
	}
	if !cfg.Amazon.Enabled {
		t.Error("expected Amazon enabled")
	}
	if cfg.Rakuten.Enabled {
		t.Error("expected Rakuten disabled")
	}
	if !cfg.Mercari.Enabled {
		t.Error("expected Mercari enabled")
	}
}

func TestLoad_Defaults(t *testing.T) {
	content := `{"keywords": ["test"], "line": {"channel_access_token": "t", "user_id": "u"}}`

	f, err := os.CreateTemp("", "config-defaults-*.json")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(f.Name())

	f.WriteString(content)
	f.Close()

	cfg, err := Load(f.Name())
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.IntervalMinutes != 30 {
		t.Errorf("expected default interval_minutes=30, got %d", cfg.IntervalMinutes)
	}
	if cfg.MinProfitRate != 10.0 {
		t.Errorf("expected default min_profit_rate=10.0, got %f", cfg.MinProfitRate)
	}
	if cfg.MinPriceDiff != 500 {
		t.Errorf("expected default min_price_diff=500, got %d", cfg.MinPriceDiff)
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/config.json")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	f, err := os.CreateTemp("", "config-invalid-*.json")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(f.Name())

	f.WriteString("{invalid json}")
	f.Close()

	_, err = Load(f.Name())
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
