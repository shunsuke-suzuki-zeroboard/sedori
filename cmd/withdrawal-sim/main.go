// Command withdrawal-sim runs a Trinity Study-style asset withdrawal simulation.
//
// Usage:
//
//	withdrawal-sim [flags]
//	  -portfolio   初期資産額（円、デフォルト: 100000000 = 1億円）
//	  -rate        取り崩し率（%、デフォルト: 4.0）
//	  -rates       複数の取り崩し率をカンマ区切りで指定（例: 3.0,3.5,4.0,4.5,5.0）
//	  -inflation   年間物価上昇率（%、デフォルト: 2.0）
//	  -period      シミュレーション期間（年、デフォルト: 30）
//	  -rule        取り崩しルール: fixed, inflation, floor-ceiling, guardrail（デフォルト: inflation）
//	  -floor       フロア・シーリングルールの下限倍率（デフォルト: 0.8）
//	  -ceiling     フロア・シーリングルールの上限倍率（デフォルト: 1.2）
//	  -detail      詳細な年次結果を表示（開始年を指定、例: 1966）
//	  -worst       最悪ケースの詳細を表示
package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/shunsuke-suzuki-zeroboard/sedori/internal/simulator"
)

func main() {
	portfolio := flag.Float64("portfolio", 100_000_000, "初期資産額（円）")
	rate := flag.Float64("rate", 4.0, "取り崩し率（%）")
	ratesStr := flag.String("rates", "", "複数の取り崩し率をカンマ区切りで指定（例: 3.0,3.5,4.0,4.5,5.0）")
	inflation := flag.Float64("inflation", 2.0, "年間物価上昇率（%）")
	period := flag.Int("period", 30, "シミュレーション期間（年）")
	rule := flag.String("rule", "inflation", "取り崩しルール: fixed, inflation, floor-ceiling, guardrail")
	floor := flag.Float64("floor", 0.80, "フロア・シーリングの下限倍率")
	ceiling := flag.Float64("ceiling", 1.20, "フロア・シーリングの上限倍率")
	detailYear := flag.Int("detail", 0, "指定した開始年の詳細を表示")
	worst := flag.Bool("worst", false, "最悪ケースの詳細を表示")
	flag.Parse()

	withdrawalRule := selectRule(*rule, *floor, *ceiling)

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("  資産取り崩しシミュレーター（トリニティスタディ方式）")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Printf("  S&P 500 データ: %d年〜%d年（%d年分）\n",
		simulator.SP500Returns[0].Year,
		simulator.SP500Returns[len(simulator.SP500Returns)-1].Year,
		len(simulator.SP500Returns))
	fmt.Println()

	if *ratesStr != "" {
		rates, err := parseRates(*ratesStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "エラー: %v\n", err)
			os.Exit(1)
		}
		runMultipleRates(*portfolio, rates, *inflation, *period, withdrawalRule)
		return
	}

	s := simulator.Strategy{
		InitialPortfolio: *portfolio,
		WithdrawalRate:   *rate / 100,
		InflationRate:    *inflation / 100,
		PeriodYears:      *period,
		Rule:             withdrawalRule,
	}

	result, err := simulator.Run(s, simulator.SP500Returns)
	if err != nil {
		fmt.Fprintf(os.Stderr, "エラー: %v\n", err)
		os.Exit(1)
	}

	printSummary(result)

	if *detailYear > 0 {
		printDetail(result, *detailYear)
	}

	if *worst {
		printWorstCase(result)
	}
}

func selectRule(name string, floor, ceiling float64) simulator.WithdrawalRule {
	switch name {
	case "fixed":
		return simulator.FixedAmountRule()
	case "inflation":
		return simulator.InflationAdjustedRule()
	case "floor-ceiling":
		return simulator.FloorCeilingRule(floor, ceiling)
	case "guardrail":
		return simulator.GuardrailRule(0.20, 0.20, 0.10, 0.10)
	default:
		fmt.Fprintf(os.Stderr, "警告: 不明なルール '%s'、インフレ調整を使用します\n", name)
		return simulator.InflationAdjustedRule()
	}
}

func parseRates(s string) ([]float64, error) {
	parts := strings.Split(s, ",")
	rates := make([]float64, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		v, err := strconv.ParseFloat(p, 64)
		if err != nil {
			return nil, fmt.Errorf("取り崩し率のパースに失敗: '%s': %w", p, err)
		}
		rates = append(rates, v/100)
	}
	return rates, nil
}

func printSummary(r *simulator.SimulationResult) {
	s := r.Strategy
	fmt.Println("【設定】")
	fmt.Printf("  初期資産:     %s\n", formatYen(s.InitialPortfolio))
	fmt.Printf("  取り崩し率:   %.1f%%（年間 %s）\n", s.WithdrawalRate*100, formatYen(s.InitialPortfolio*s.WithdrawalRate))
	fmt.Printf("  物価上昇率:   %.1f%%/年\n", s.InflationRate*100)
	fmt.Printf("  期間:         %d年\n", s.PeriodYears)
	fmt.Printf("  ルール:       %s\n", s.Rule.Name)
	fmt.Println()

	fmt.Println("【結果】")
	fmt.Printf("  成功率:       %.1f%%（%d / %d 期間）\n",
		r.SuccessRate*100, r.SuccessCount, r.TotalCount)
	fmt.Printf("  失敗率:       %.1f%%（%d / %d 期間）\n",
		(1-r.SuccessRate)*100, r.TotalCount-r.SuccessCount, r.TotalCount)
	fmt.Println()

	// 成功ケースの最終残高統計
	var minBalance, maxBalance, sumBalance float64
	var successCount int
	minBalance = math.MaxFloat64
	for _, p := range r.Periods {
		if p.Success {
			successCount++
			sumBalance += p.FinalBalance
			if p.FinalBalance < minBalance {
				minBalance = p.FinalBalance
			}
			if p.FinalBalance > maxBalance {
				maxBalance = p.FinalBalance
			}
		}
	}

	if successCount > 0 {
		fmt.Println("【成功ケースの最終残高】")
		fmt.Printf("  最小: %s\n", formatYen(minBalance))
		fmt.Printf("  平均: %s\n", formatYen(sumBalance/float64(successCount)))
		fmt.Printf("  最大: %s\n", formatYen(maxBalance))
		fmt.Println()
	}

	// 失敗ケースの一覧
	failedPeriods := make([]simulator.PeriodResult, 0)
	for _, p := range r.Periods {
		if !p.Success {
			failedPeriods = append(failedPeriods, p)
		}
	}
	if len(failedPeriods) > 0 {
		fmt.Println("【失敗した期間】")
		for _, p := range failedPeriods {
			depletedYear := findDepletionYear(p)
			fmt.Printf("  %d〜%d年（%d年目で枯渇）\n", p.StartYear, p.EndYear, depletedYear)
		}
		fmt.Println()
	}
}

func runMultipleRates(portfolio float64, rates []float64, inflation float64, period int, rule simulator.WithdrawalRule) {
	fmt.Println("【取り崩し率別 成功率一覧】")
	fmt.Println()
	fmt.Printf("  %-10s  %-12s  %-10s  %s\n", "取り崩し率", "年間取り崩し", "成功率", "バー")
	fmt.Println("  " + strings.Repeat("─", 60))

	for _, r := range rates {
		s := simulator.Strategy{
			InitialPortfolio: portfolio,
			WithdrawalRate:   r,
			InflationRate:    inflation / 100,
			PeriodYears:      period,
			Rule:             rule,
		}
		result, err := simulator.Run(s, simulator.SP500Returns)
		if err != nil {
			fmt.Fprintf(os.Stderr, "エラー (%.1f%%): %v\n", r*100, err)
			continue
		}

		bar := strings.Repeat("█", int(result.SuccessRate*40))
		fmt.Printf("  %-10s  %-12s  %5.1f%%     %s\n",
			fmt.Sprintf("%.1f%%", r*100),
			formatYen(portfolio*r),
			result.SuccessRate*100,
			bar)
	}
	fmt.Println()
}

func printDetail(r *simulator.SimulationResult, startYear int) {
	for _, p := range r.Periods {
		if p.StartYear == startYear {
			fmt.Printf("【%d年開始の詳細】\n", startYear)
			fmt.Printf("  結果: %s\n", boolToResult(p.Success))
			fmt.Printf("  最終残高: %s\n", formatYen(p.FinalBalance))
			fmt.Println()
			fmt.Printf("  %-6s  %14s  %14s  %8s  %14s\n", "年", "期首残高", "取り崩し", "騰落率", "期末残高")
			fmt.Println("  " + strings.Repeat("─", 64))
			for _, y := range p.Years {
				fmt.Printf("  %-6d  %14s  %14s  %+7.1f%%  %14s\n",
					y.Year,
					formatYen(y.StartBalance),
					formatYen(y.Withdrawal),
					y.MarketReturn*100,
					formatYen(y.EndBalance))
			}
			fmt.Println()
			return
		}
	}
	fmt.Printf("  開始年 %d のデータが見つかりません\n\n", startYear)
}

func printWorstCase(r *simulator.SimulationResult) {
	var worst *simulator.PeriodResult
	worstBalance := math.MaxFloat64

	for i := range r.Periods {
		p := &r.Periods[i]
		if p.FinalBalance < worstBalance {
			worstBalance = p.FinalBalance
			worst = p
		}
	}

	if worst == nil {
		return
	}

	fmt.Printf("【最悪ケース: %d年開始】\n", worst.StartYear)
	fmt.Printf("  結果: %s\n", boolToResult(worst.Success))
	fmt.Printf("  最終残高: %s\n", formatYen(worst.FinalBalance))
	fmt.Println()
	fmt.Printf("  %-6s  %14s  %14s  %8s  %14s\n", "年", "期首残高", "取り崩し", "騰落率", "期末残高")
	fmt.Println("  " + strings.Repeat("─", 64))
	for _, y := range worst.Years {
		fmt.Printf("  %-6d  %14s  %14s  %+7.1f%%  %14s\n",
			y.Year,
			formatYen(y.StartBalance),
			formatYen(y.Withdrawal),
			y.MarketReturn*100,
			formatYen(y.EndBalance))
	}
	fmt.Println()
}

func findDepletionYear(p simulator.PeriodResult) int {
	for _, y := range p.Years {
		if y.EndBalance <= 0 {
			return y.YearIndex + 1
		}
	}
	return len(p.Years)
}

func boolToResult(b bool) string {
	if b {
		return "成功（資産残存）"
	}
	return "失敗（資産枯渇）"
}

func formatYen(v float64) string {
	if v < 0 {
		return "-" + formatYen(-v)
	}
	s := fmt.Sprintf("%.0f", v)
	// Add comma separators
	n := len(s)
	if n <= 3 {
		return "¥" + s
	}
	var b strings.Builder
	b.WriteString("¥")
	remainder := n % 3
	if remainder > 0 {
		b.WriteString(s[:remainder])
		if n > remainder {
			b.WriteString(",")
		}
	}
	for i := remainder; i < n; i += 3 {
		b.WriteString(s[i : i+3])
		if i+3 < n {
			b.WriteString(",")
		}
	}
	return b.String()
}
