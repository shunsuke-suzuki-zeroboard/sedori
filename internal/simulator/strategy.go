package simulator

import "fmt"

// WithdrawalRule defines how the annual withdrawal amount is adjusted.
type WithdrawalRule struct {
	// Name is a human-readable label for this rule.
	Name string

	// AdjustWithdrawal calculates the withdrawal amount for the current year.
	// Parameters:
	//   baseWithdrawal: initial year's withdrawal amount (nominal)
	//   prevWithdrawal: previous year's actual withdrawal
	//   inflationRate:  annual inflation rate (e.g., 0.03 = 3%)
	//   yearReturn:     S&P 500 return for the previous year (e.g., 0.10 = +10%)
	//   yearIndex:      0-based index of the current simulation year
	AdjustWithdrawal func(baseWithdrawal, prevWithdrawal, inflationRate, yearReturn float64, yearIndex int) float64
}

// Strategy defines a complete withdrawal simulation configuration.
type Strategy struct {
	// InitialPortfolio is the starting portfolio value (e.g., 100_000_000 = 1億円).
	InitialPortfolio float64

	// WithdrawalRate is the initial annual withdrawal rate (e.g., 0.04 = 4%).
	WithdrawalRate float64

	// InflationRate is the assumed annual inflation rate (e.g., 0.03 = 3%).
	InflationRate float64

	// PeriodYears is the simulation period in years (e.g., 30).
	PeriodYears int

	// Rule defines how withdrawals are adjusted each year.
	Rule WithdrawalRule
}

// String returns a human-readable description of the strategy.
func (s Strategy) String() string {
	return fmt.Sprintf("初期資産: ¥%.0f / 取り崩し率: %.1f%% / インフレ率: %.1f%% / 期間: %d年 / ルール: %s",
		s.InitialPortfolio, s.WithdrawalRate*100, s.InflationRate*100, s.PeriodYears, s.Rule.Name)
}

// InflationAdjustedRule returns a rule that increases withdrawal by inflation each year.
func InflationAdjustedRule() WithdrawalRule {
	return WithdrawalRule{
		Name: "インフレ調整（毎年物価上昇率分を増額）",
		AdjustWithdrawal: func(baseWithdrawal, prevWithdrawal, inflationRate, yearReturn float64, yearIndex int) float64 {
			return prevWithdrawal * (1 + inflationRate)
		},
	}
}

// FixedAmountRule returns a rule that keeps the withdrawal amount constant (no adjustment).
func FixedAmountRule() WithdrawalRule {
	return WithdrawalRule{
		Name: "定額（毎年同額を取り崩し）",
		AdjustWithdrawal: func(baseWithdrawal, prevWithdrawal, inflationRate, yearReturn float64, yearIndex int) float64 {
			return baseWithdrawal
		},
	}
}

// FloorCeilingRule returns a rule that adjusts for inflation but caps increases and
// floors decreases relative to the base withdrawal.
// After a down year (negative return), withdrawal is not increased.
// After an up year, withdrawal increases by inflation but is capped at ceiling * base.
// Withdrawal never falls below floor * base.
func FloorCeilingRule(floor, ceiling float64) WithdrawalRule {
	return WithdrawalRule{
		Name: fmt.Sprintf("フロア・シーリング（下限: %.0f%% / 上限: %.0f%%）", floor*100, ceiling*100),
		AdjustWithdrawal: func(baseWithdrawal, prevWithdrawal, inflationRate, yearReturn float64, yearIndex int) float64 {
			var w float64
			if yearReturn < 0 {
				// 下落年: 取り崩し額を据え置き
				w = prevWithdrawal
			} else {
				// 上昇年: インフレ調整
				w = prevWithdrawal * (1 + inflationRate)
			}
			// フロア・シーリング適用（初期取り崩し額のインフレ調整値を基準）
			inflatedBase := baseWithdrawal
			for i := 0; i < yearIndex; i++ {
				inflatedBase *= (1 + inflationRate)
			}
			minW := inflatedBase * floor
			maxW := inflatedBase * ceiling
			if w < minW {
				w = minW
			}
			if w > maxW {
				w = maxW
			}
			return w
		},
	}
}

// GuardrailRule returns a rule based on the Guyton-Klinger guardrails approach.
// If the current withdrawal rate exceeds the initial rate by more than upperGuard,
// the withdrawal is cut by cutRate.
// If the current withdrawal rate falls below the initial rate by more than lowerGuard,
// the withdrawal is raised by raiseRate.
func GuardrailRule(upperGuard, lowerGuard, cutRate, raiseRate float64) WithdrawalRule {
	return WithdrawalRule{
		Name: fmt.Sprintf("ガードレール（上限+%.0f%% / 下限-%.0f%% / カット%.0f%% / 増額%.0f%%）",
			upperGuard*100, lowerGuard*100, cutRate*100, raiseRate*100),
		AdjustWithdrawal: func(baseWithdrawal, prevWithdrawal, inflationRate, yearReturn float64, yearIndex int) float64 {
			// まずインフレ調整
			w := prevWithdrawal * (1 + inflationRate)
			return w
		},
	}
}

// GuardrailAdjust applies guardrail logic after withdrawal is determined.
// This is called separately with the current portfolio value.
func GuardrailAdjust(withdrawal, portfolio, initialRate, upperGuard, lowerGuard, cutRate, raiseRate float64) float64 {
	if portfolio <= 0 {
		return withdrawal
	}
	currentRate := withdrawal / portfolio
	upperThreshold := initialRate * (1 + upperGuard)
	lowerThreshold := initialRate * (1 - lowerGuard)

	if currentRate > upperThreshold {
		withdrawal *= (1 - cutRate)
	} else if currentRate < lowerThreshold {
		withdrawal *= (1 + raiseRate)
	}
	return withdrawal
}
