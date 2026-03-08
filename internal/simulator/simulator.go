// Package simulator implements a Trinity Study-style asset withdrawal simulation
// using historical S&P 500 returns.
package simulator

import "fmt"

// YearResult holds the state of the portfolio at the end of a simulation year.
type YearResult struct {
	Year              int     // Calendar year
	YearIndex         int     // 0-based index within the simulation
	StartBalance      float64 // Portfolio value at start of year
	Withdrawal        float64 // Amount withdrawn this year
	MarketReturn      float64 // S&P 500 return for this year
	EndBalance        float64 // Portfolio value at end of year (after withdrawal and return)
	CumulativeInflate float64 // Cumulative inflation factor from start
}

// PeriodResult holds the outcome of a single rolling-period simulation.
type PeriodResult struct {
	StartYear    int          // First year of this simulation period
	EndYear      int          // Last year of this simulation period
	Success      bool         // True if portfolio survived the entire period
	FinalBalance float64      // Portfolio value at the end
	Years        []YearResult // Year-by-year breakdown
}

// SimulationResult holds the aggregate results across all rolling periods.
type SimulationResult struct {
	Strategy     Strategy
	Periods      []PeriodResult
	SuccessCount int
	TotalCount   int
	SuccessRate  float64 // 0.0 - 1.0
}

// Run executes the Trinity Study-style simulation.
// It tests the given strategy across all possible rolling periods in the historical data.
func Run(s Strategy, returns []AnnualReturn) (*SimulationResult, error) {
	if s.PeriodYears <= 0 {
		return nil, fmt.Errorf("simulator: period must be positive, got %d", s.PeriodYears)
	}
	if len(returns) < s.PeriodYears {
		return nil, fmt.Errorf("simulator: insufficient data: need %d years, have %d", s.PeriodYears, len(returns))
	}

	totalPeriods := len(returns) - s.PeriodYears + 1
	result := &SimulationResult{
		Strategy: s,
		Periods:  make([]PeriodResult, 0, totalPeriods),
	}

	for start := 0; start < totalPeriods; start++ {
		pr := simulatePeriod(s, returns[start:start+s.PeriodYears])
		result.Periods = append(result.Periods, pr)
		if pr.Success {
			result.SuccessCount++
		}
	}

	result.TotalCount = totalPeriods
	if totalPeriods > 0 {
		result.SuccessRate = float64(result.SuccessCount) / float64(totalPeriods)
	}

	return result, nil
}

func simulatePeriod(s Strategy, returns []AnnualReturn) PeriodResult {
	balance := s.InitialPortfolio
	baseWithdrawal := s.InitialPortfolio * s.WithdrawalRate
	prevWithdrawal := baseWithdrawal
	cumulativeInflation := 1.0

	years := make([]YearResult, 0, len(returns))
	success := true

	for i, ar := range returns {
		startBalance := balance

		// Determine withdrawal amount
		var withdrawal float64
		if i == 0 {
			withdrawal = baseWithdrawal
		} else {
			withdrawal = s.Rule.AdjustWithdrawal(baseWithdrawal, prevWithdrawal, s.InflationRate, returns[i-1].Return, i)
		}

		// Cannot withdraw more than available
		if withdrawal > balance {
			withdrawal = balance
		}

		// Withdraw at the beginning of the year
		balance -= withdrawal

		// Apply market return on remaining balance
		balance *= (1 + ar.Return)

		cumulativeInflation *= (1 + s.InflationRate)

		yr := YearResult{
			Year:              ar.Year,
			YearIndex:         i,
			StartBalance:      startBalance,
			Withdrawal:        withdrawal,
			MarketReturn:      ar.Return,
			EndBalance:        balance,
			CumulativeInflate: cumulativeInflation,
		}
		years = append(years, yr)

		prevWithdrawal = withdrawal

		// Check if portfolio is depleted
		if balance <= 0 {
			success = false
			// Fill remaining years as zero
			for j := i + 1; j < len(returns); j++ {
				years = append(years, YearResult{
					Year:              returns[j].Year,
					YearIndex:         j,
					StartBalance:      0,
					Withdrawal:        0,
					MarketReturn:      returns[j].Return,
					EndBalance:        0,
					CumulativeInflate: cumulativeInflation * (1 + s.InflationRate),
				})
				cumulativeInflation *= (1 + s.InflationRate)
			}
			break
		}
	}

	return PeriodResult{
		StartYear:    returns[0].Year,
		EndYear:      returns[len(returns)-1].Year,
		Success:      success,
		FinalBalance: balance,
		Years:        years,
	}
}

// RunMultipleRates runs simulations for a range of withdrawal rates and returns all results.
func RunMultipleRates(baseStrategy Strategy, rates []float64, returns []AnnualReturn) ([]*SimulationResult, error) {
	results := make([]*SimulationResult, 0, len(rates))
	for _, rate := range rates {
		s := baseStrategy
		s.WithdrawalRate = rate
		r, err := Run(s, returns)
		if err != nil {
			return nil, fmt.Errorf("simulator: rate %.1f%%: %w", rate*100, err)
		}
		results = append(results, r)
	}
	return results, nil
}
