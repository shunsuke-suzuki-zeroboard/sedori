package simulator

import (
	"math"
	"testing"
)

func TestRunFixedAmount_SimpleData(t *testing.T) {
	// 10% return every year, 4% withdrawal, no inflation, 3 years
	returns := []AnnualReturn{
		{2000, 0.10},
		{2001, 0.10},
		{2002, 0.10},
	}
	s := Strategy{
		InitialPortfolio: 1_000_000,
		WithdrawalRate:   0.04,
		InflationRate:    0.0,
		PeriodYears:      3,
		Rule:             FixedAmountRule(),
	}

	result, err := Run(s, returns)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.TotalCount != 1 {
		t.Errorf("expected 1 period, got %d", result.TotalCount)
	}
	if !result.Periods[0].Success {
		t.Error("expected success")
	}

	// Year 1: start=1000000, withdraw=40000, remain=960000, *1.10 = 1056000
	// Year 2: start=1056000, withdraw=40000, remain=1016000, *1.10 = 1117600
	// Year 3: start=1117600, withdraw=40000, remain=1077600, *1.10 = 1185360
	expected := 1_185_360.0
	if math.Abs(result.Periods[0].FinalBalance-expected) > 1 {
		t.Errorf("expected final balance ~%.0f, got %.0f", expected, result.Periods[0].FinalBalance)
	}
}

func TestRunInflationAdjusted(t *testing.T) {
	returns := []AnnualReturn{
		{2000, 0.10},
		{2001, 0.10},
		{2002, 0.10},
	}
	s := Strategy{
		InitialPortfolio: 1_000_000,
		WithdrawalRate:   0.04,
		InflationRate:    0.03,
		PeriodYears:      3,
		Rule:             InflationAdjustedRule(),
	}

	result, err := Run(s, returns)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Year 1: withdraw=40000, remain=960000, *1.10 = 1056000
	// Year 2: withdraw=40000*1.03=41200, remain=1014800, *1.10 = 1116280
	// Year 3: withdraw=41200*1.03=42436, remain=1073844, *1.10 = 1181228.4
	p := result.Periods[0]
	if !p.Success {
		t.Error("expected success")
	}

	expectedY2Withdrawal := 40000 * 1.03
	if math.Abs(p.Years[1].Withdrawal-expectedY2Withdrawal) > 0.01 {
		t.Errorf("year 2 withdrawal: expected %.2f, got %.2f", expectedY2Withdrawal, p.Years[1].Withdrawal)
	}

	expectedY3Withdrawal := expectedY2Withdrawal * 1.03
	if math.Abs(p.Years[2].Withdrawal-expectedY3Withdrawal) > 0.01 {
		t.Errorf("year 3 withdrawal: expected %.2f, got %.2f", expectedY3Withdrawal, p.Years[2].Withdrawal)
	}
}

func TestRunDepletion(t *testing.T) {
	// Portfolio depletes with high withdrawal and negative returns
	returns := []AnnualReturn{
		{2000, -0.50},
		{2001, -0.50},
		{2002, -0.50},
	}
	s := Strategy{
		InitialPortfolio: 1_000_000,
		WithdrawalRate:   0.50,
		InflationRate:    0.0,
		PeriodYears:      3,
		Rule:             FixedAmountRule(),
	}

	result, err := Run(s, returns)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Periods[0].Success {
		t.Error("expected failure (depletion)")
	}
	if result.SuccessRate != 0 {
		t.Errorf("expected 0%% success rate, got %.1f%%", result.SuccessRate*100)
	}
}

func TestRunRollingPeriods(t *testing.T) {
	returns := []AnnualReturn{
		{2000, 0.10},
		{2001, 0.10},
		{2002, 0.10},
		{2003, 0.10},
		{2004, 0.10},
	}
	s := Strategy{
		InitialPortfolio: 1_000_000,
		WithdrawalRate:   0.04,
		InflationRate:    0.0,
		PeriodYears:      3,
		Rule:             FixedAmountRule(),
	}

	result, err := Run(s, returns)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 5 years of data, 3-year period = 3 rolling periods (2000-2002, 2001-2003, 2002-2004)
	if result.TotalCount != 3 {
		t.Errorf("expected 3 rolling periods, got %d", result.TotalCount)
	}
}

func TestRunInsufficientData(t *testing.T) {
	returns := []AnnualReturn{
		{2000, 0.10},
	}
	s := Strategy{
		InitialPortfolio: 1_000_000,
		WithdrawalRate:   0.04,
		PeriodYears:      3,
		Rule:             FixedAmountRule(),
	}

	_, err := Run(s, returns)
	if err == nil {
		t.Error("expected error for insufficient data")
	}
}

func TestRunMultipleRates(t *testing.T) {
	returns := []AnnualReturn{
		{2000, 0.10},
		{2001, 0.10},
		{2002, 0.10},
	}
	base := Strategy{
		InitialPortfolio: 1_000_000,
		InflationRate:    0.0,
		PeriodYears:      3,
		Rule:             FixedAmountRule(),
	}
	rates := []float64{0.03, 0.04, 0.05}

	results, err := RunMultipleRates(base, rates, returns)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}
}

func TestSP500ReturnsData(t *testing.T) {
	if len(SP500Returns) < 90 {
		t.Errorf("expected at least 90 years of data, got %d", len(SP500Returns))
	}
	// Verify data is chronologically ordered
	for i := 1; i < len(SP500Returns); i++ {
		if SP500Returns[i].Year != SP500Returns[i-1].Year+1 {
			t.Errorf("data not sequential at index %d: %d -> %d",
				i, SP500Returns[i-1].Year, SP500Returns[i].Year)
		}
	}
}

func TestFloorCeilingRule(t *testing.T) {
	rule := FloorCeilingRule(0.8, 1.2)

	// After a down year, withdrawal should not increase
	w := rule.AdjustWithdrawal(100, 100, 0.03, -0.10, 1)
	if w != 100 {
		t.Errorf("after down year, expected 100, got %.2f", w)
	}

	// After an up year, withdrawal should increase by inflation
	w = rule.AdjustWithdrawal(100, 100, 0.03, 0.10, 1)
	if math.Abs(w-103) > 0.01 {
		t.Errorf("after up year, expected 103, got %.2f", w)
	}
}

func TestSkipDownYearRule(t *testing.T) {
	rule := SkipDownYearRule()

	// After a down year, withdrawal should be 0
	w := rule.AdjustWithdrawal(100, 100, 0.03, -0.10, 1)
	if w != 0 {
		t.Errorf("after down year, expected 0, got %.2f", w)
	}

	// After an up year, should return inflation-adjusted amount
	w = rule.AdjustWithdrawal(100, 100, 0.03, 0.10, 1)
	if math.Abs(w-103) > 0.01 {
		t.Errorf("after up year, expected 103, got %.2f", w)
	}

	// After skip (prevWithdrawal=0), should recalculate from base
	w = rule.AdjustWithdrawal(100, 0, 0.03, 0.10, 2)
	expected := 100 * 1.03 * 1.03 // base * (1+inflation)^2
	if math.Abs(w-expected) > 0.01 {
		t.Errorf("after skip, expected %.2f, got %.2f", expected, w)
	}
}

func TestSkipDownYearRuleSimulation(t *testing.T) {
	// 下落→上昇→上昇 の3年間
	returns := []AnnualReturn{
		{2000, -0.20}, // 下落年
		{2001, 0.30},  // 上昇年（前年下落なのでスキップ）
		{2002, 0.10},  // 上昇年（前年上昇なので取り崩す）
	}
	s := Strategy{
		InitialPortfolio: 1_000_000,
		WithdrawalRate:   0.04,
		InflationRate:    0.02,
		PeriodYears:      3,
		Rule:             SkipDownYearRule(),
	}

	result, err := Run(s, returns)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	p := result.Periods[0]

	// Year 1 (2000): first year, always withdraw base = 40000
	// remain = 960000, * 0.80 = 768000
	if math.Abs(p.Years[0].Withdrawal-40000) > 0.01 {
		t.Errorf("year 1 withdrawal: expected 40000, got %.2f", p.Years[0].Withdrawal)
	}

	// Year 2 (2001): previous year return was -0.20 → skip (withdrawal = 0)
	// remain = 768000, * 1.30 = 998400
	if p.Years[1].Withdrawal != 0 {
		t.Errorf("year 2 withdrawal: expected 0 (skip), got %.2f", p.Years[1].Withdrawal)
	}

	// Year 3 (2002): previous year return was +0.30 → withdraw
	// prevWithdrawal=0 なので base からインフレ調整で再計算
	if p.Years[2].Withdrawal == 0 {
		t.Error("year 3 withdrawal: should not be 0")
	}
	t.Logf("year-by-year: w1=%.0f w2=%.0f w3=%.0f final=%.0f",
		p.Years[0].Withdrawal, p.Years[1].Withdrawal, p.Years[2].Withdrawal, p.FinalBalance)
}

func TestSkipDownYearWithSP500(t *testing.T) {
	s := Strategy{
		InitialPortfolio: 100_000_000,
		WithdrawalRate:   0.04,
		InflationRate:    0.02,
		PeriodYears:      30,
		Rule:             SkipDownYearRule(),
	}
	result, err := Run(s, SP500Returns)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 下落時スキップルールはインフレ調整のみより成功率が高いはず
	t.Logf("skip-down rule: %.1f%% success (%d/%d)",
		result.SuccessRate*100, result.SuccessCount, result.TotalCount)
	if result.SuccessRate < 0.90 {
		t.Errorf("skip-down rule success rate unexpectedly low: %.1f%%", result.SuccessRate*100)
	}
}

func TestRunWithSP500Data(t *testing.T) {
	// Integration test with real S&P 500 data
	s := Strategy{
		InitialPortfolio: 100_000_000,
		WithdrawalRate:   0.04,
		InflationRate:    0.03,
		PeriodYears:      30,
		Rule:             InflationAdjustedRule(),
	}

	result, err := Run(s, SP500Returns)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 4% rule should have high success rate historically
	if result.SuccessRate < 0.80 {
		t.Errorf("4%% rule success rate unexpectedly low: %.1f%%", result.SuccessRate*100)
	}
	t.Logf("4%% rule with 3%% inflation over 30 years: %.1f%% success (%d/%d)",
		result.SuccessRate*100, result.SuccessCount, result.TotalCount)
}
