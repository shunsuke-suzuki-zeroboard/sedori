package notifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/shunsuke-suzuki-zeroboard/sedori/internal/domain"
)

// LINENotifier sends notifications via LINE Messaging API.
type LINENotifier struct {
	channelAccessToken string
	userID             string
	client             *http.Client
}

func NewLINENotifier(channelAccessToken, userID string) *LINENotifier {
	return &LINENotifier{
		channelAccessToken: channelAccessToken,
		userID:             userID,
		client:             &http.Client{Timeout: 10 * time.Second},
	}
}

type lineMessage struct {
	To       string        `json:"to"`
	Messages []interface{} `json:"messages"`
}

type lineTextMessage struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// Notify sends price diff notifications to LINE.
func (l *LINENotifier) Notify(diffs []domain.PriceDiff) error {
	if len(diffs) == 0 {
		return nil
	}

	text := l.formatMessage(diffs)
	return l.send(text)
}

// SendText sends a simple text message to LINE (useful for testing).
func (l *LINENotifier) SendText(text string) error {
	return l.send(text)
}

// send pushes a text message to the configured LINE user.
func (l *LINENotifier) send(text string) error {
	msg := lineMessage{
		To: l.userID,
		Messages: []interface{}{
			lineTextMessage{
				Type: "text",
				Text: text,
			},
		},
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("line: failed to marshal message: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, "https://api.line.me/v2/bot/message/push", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("line: failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+l.channelAccessToken)

	resp, err := l.client.Do(req)
	if err != nil {
		return fmt.Errorf("line: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("line: unexpected status code: %d, body: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func (l *LINENotifier) formatMessage(diffs []domain.PriceDiff) string {
	var sb strings.Builder
	sb.WriteString("🔔 せどりチャンス発見!\n\n")

	for i, d := range diffs {
		if i > 0 {
			sb.WriteString("\n---\n\n")
		}
		sb.WriteString(fmt.Sprintf("📦 %s\n", d.Keyword))
		sb.WriteString(fmt.Sprintf("  仕入れ: %s ¥%d\n", siteLabel(d.BuyFrom.Site), d.BuyFrom.Price))
		sb.WriteString(fmt.Sprintf("  → %s\n", d.BuyFrom.URL))
		sb.WriteString(fmt.Sprintf("  販売: %s ¥%d\n", siteLabel(d.SellAt.Site), d.SellAt.Price))
		sb.WriteString(fmt.Sprintf("  → %s\n", d.SellAt.URL))
		sb.WriteString(fmt.Sprintf("  💰 価格差: ¥%d (利益率: %.1f%%)\n", d.PriceDiff, d.ProfitRate))
	}

	return sb.String()
}

func siteLabel(site domain.Site) string {
	switch site {
	case domain.SiteAmazon:
		return "Amazon"
	case domain.SiteRakuten:
		return "楽天市場"
	case domain.SiteMercari:
		return "メルカリ"
	default:
		return string(site)
	}
}
