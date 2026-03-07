package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config holds the application configuration.
type Config struct {
	Keywords         []string      `json:"keywords"`
	MinProfitRate    float64       `json:"min_profit_rate"`    // minimum profit rate (%) to trigger notification
	MinPriceDiff     int           `json:"min_price_diff"`     // minimum price difference (yen) to trigger notification
	IntervalMinutes  int           `json:"interval_minutes"`   // monitoring interval
	LINE             LINEConfig    `json:"line"`
	Amazon           AmazonConfig  `json:"amazon"`
	Rakuten          RakutenConfig `json:"rakuten"`
	Mercari          MercariConfig `json:"mercari"`
}

type LINEConfig struct {
	ChannelAccessToken string `json:"channel_access_token"`
	UserID             string `json:"user_id"`
}

type AmazonConfig struct {
	Enabled     bool   `json:"enabled"`
	AccessKey   string `json:"access_key"`
	SecretKey   string `json:"secret_key"`
	PartnerTag  string `json:"partner_tag"`
	Marketplace string `json:"marketplace"`
}

type RakutenConfig struct {
	Enabled       bool   `json:"enabled"`
	ApplicationID string `json:"application_id"`
}

type MercariConfig struct {
	Enabled bool `json:"enabled"`
}

// Load reads and parses the config file.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if cfg.IntervalMinutes <= 0 {
		cfg.IntervalMinutes = 30
	}
	if cfg.MinProfitRate <= 0 {
		cfg.MinProfitRate = 10.0
	}
	if cfg.MinPriceDiff <= 0 {
		cfg.MinPriceDiff = 500
	}

	return &cfg, nil
}
