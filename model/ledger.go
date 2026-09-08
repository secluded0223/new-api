package model

import (
	"sort"
	"time"

	"github.com/QuantumNous/new-api/common"
)

// LedgerExpense stores manually entered operating expenses in minor currency units.
type LedgerExpense struct {
	Id         int    `json:"id"`
	OccurredAt int64  `json:"occurred_at" gorm:"index"`
	Category   string `json:"category" gorm:"size:64;index"`
	Amount     int64  `json:"amount"`
	Currency   string `json:"currency" gorm:"size:8;default:'USD'"`
	Note       string `json:"note" gorm:"size:255"`
	CreatedAt  int64  `json:"created_at"`
}

type LedgerExpenseInput struct {
	OccurredAt int64  `json:"occurred_at"`
	Category   string `json:"category"`
	Amount     int64  `json:"amount"`
	Currency   string `json:"currency"`
	Note       string `json:"note"`
}

type LedgerSummary struct {
	RevenueQuota int64              `json:"revenue_quota"`
	RevenueCents int64              `json:"revenue_cents"`
	Expenses     int64              `json:"expenses"`
	Profit       int64              `json:"profit"`
	Currency     string             `json:"currency"`
	ExpensesList []LedgerExpense    `json:"expenses_list"`
	Trend        []LedgerTrendPoint `json:"trend"`
}

type LedgerTrendPoint struct {
	Date         string `json:"date"`
	RevenueCents int64  `json:"revenue_cents"`
	ExpenseCents int64  `json:"expense_cents"`
	ProfitCents  int64  `json:"profit_cents"`
}

type LedgerMonthlyPoint struct {
	Month        int   `json:"month"`
	RevenueCents int64 `json:"revenue_cents"`
	ExpenseCents int64 `json:"expense_cents"`
	ProfitCents  int64 `json:"profit_cents"`
}

type LedgerMonthlySummary struct {
	Year         int                  `json:"year"`
	Currency     string               `json:"currency"`
	RevenueCents int64                `json:"revenue_cents"`
	Expenses     int64                `json:"expenses"`
	Profit       int64                `json:"profit"`
	Months       []LedgerMonthlyPoint `json:"months"`
}

// GetLedgerMonthlySummary returns a fixed twelve-month view for the given local calendar year.
func GetLedgerMonthlySummary(year int) (LedgerMonthlySummary, error) {
	start := time.Date(year, time.January, 1, 0, 0, 0, 0, time.Local)
	end := start.AddDate(1, 0, 0)
	monthQuotas := make([]int64, 12)
	var logs []struct {
		CreatedAt int64
		Type      int
		Quota     int
	}
	if err := LOG_DB.Table("logs").Select("created_at, type, quota").
		Where("created_at >= ? AND created_at < ? AND type IN (?, ?)", start.Unix(), end.Unix(), LogTypeConsume, LogTypeRefund).
		Find(&logs).Error; err != nil {
		return LedgerMonthlySummary{}, err
	}
	for _, log := range logs {
		month := time.Unix(log.CreatedAt, 0).In(time.Local).Month()
		index := int(month) - 1
		if index < 0 || index >= len(monthQuotas) {
			continue
		}
		if log.Type == LogTypeConsume {
			monthQuotas[index] += int64(log.Quota)
		} else {
			monthQuotas[index] -= int64(log.Quota)
		}
	}

	var expenses []LedgerExpense
	if err := DB.Where("occurred_at >= ? AND occurred_at < ?", start.Unix(), end.Unix()).Find(&expenses).Error; err != nil {
		return LedgerMonthlySummary{}, err
	}
	months := make([]LedgerMonthlyPoint, 12)
	var totalExpenses int64
	for i := range months {
		months[i].Month = i + 1
		months[i].RevenueCents = int64(common.QuotaRound(float64(monthQuotas[i]) / common.QuotaPerUnit * 100))
	}
	for _, expense := range expenses {
		if expense.Currency != "USD" && expense.Currency != "" {
			continue
		}
		month := time.Unix(expense.OccurredAt, 0).In(time.Local).Month()
		index := int(month) - 1
		if index < 0 || index >= len(months) {
			continue
		}
		months[index].ExpenseCents += expense.Amount
		totalExpenses += expense.Amount
	}

	var totalQuota int64
	for _, quota := range monthQuotas {
		totalQuota += quota
	}
	totalRevenue := int64(common.QuotaRound(float64(totalQuota) / common.QuotaPerUnit * 100))
	var monthlyRevenue int64
	lastRevenueMonth := -1
	for i := range months {
		monthlyRevenue += months[i].RevenueCents
		if monthQuotas[i] != 0 {
			lastRevenueMonth = i
		}
	}
	if remainder := totalRevenue - monthlyRevenue; remainder != 0 && lastRevenueMonth >= 0 {
		months[lastRevenueMonth].RevenueCents += remainder
	}
	for i := range months {
		months[i].ProfitCents = months[i].RevenueCents - months[i].ExpenseCents
	}
	return LedgerMonthlySummary{
		Year:         year,
		Currency:     "USD",
		RevenueCents: totalRevenue,
		Expenses:     totalExpenses,
		Profit:       totalRevenue - totalExpenses,
		Months:       months,
	}, nil
}

func GetLedgerSummary(startTimestamp, endTimestamp int64) (LedgerSummary, error) {
	if startTimestamp == 0 {
		startTimestamp = time.Now().AddDate(0, 0, -30).Unix()
	}
	if endTimestamp == 0 {
		endTimestamp = time.Now().Unix()
	}
	var revenue struct {
		Consume int64
		Refund  int64
	}
	logQuery := LOG_DB.Table("logs").Select(
		"COALESCE(SUM(CASE WHEN type = ? THEN quota ELSE 0 END), 0) AS consume, COALESCE(SUM(CASE WHEN type = ? THEN quota ELSE 0 END), 0) AS refund",
		LogTypeConsume, LogTypeRefund,
	).Where("created_at <= ?", endTimestamp)
	if startTimestamp > 0 {
		logQuery = logQuery.Where("created_at >= ?", startTimestamp)
	}
	if err := logQuery.Scan(&revenue).Error; err != nil {
		return LedgerSummary{}, err
	}
	var expenses []LedgerExpense
	expenseQuery := DB.Where("occurred_at <= ?", endTimestamp)
	if startTimestamp > 0 {
		expenseQuery = expenseQuery.Where("occurred_at >= ?", startTimestamp)
	}
	if err := expenseQuery.Order("occurred_at DESC, id DESC").Find(&expenses).Error; err != nil {
		return LedgerSummary{}, err
	}
	var total int64
	for _, expense := range expenses {
		if expense.Currency == "USD" || expense.Currency == "" {
			total += expense.Amount
		}
	}
	trendMap := make(map[string]*LedgerTrendPoint)
	trendRevenueQuota := make(map[string]int64)
	var logs []struct {
		CreatedAt int64
		Type      int
		Quota     int
	}
	trendQuery := LOG_DB.Table("logs").Select("created_at, type, quota").Where("created_at <= ? AND type IN (?, ?)", endTimestamp, LogTypeConsume, LogTypeRefund)
	if startTimestamp > 0 {
		trendQuery = trendQuery.Where("created_at >= ?", startTimestamp)
	}
	if err := trendQuery.Find(&logs).Error; err != nil {
		return LedgerSummary{}, err
	}
	for _, log := range logs {
		date := time.Unix(log.CreatedAt, 0).Format("2006-01-02")
		point := trendMap[date]
		if point == nil {
			point = &LedgerTrendPoint{Date: date}
			trendMap[date] = point
		}
		if log.Type == LogTypeConsume {
			trendRevenueQuota[date] += int64(log.Quota)
		} else {
			trendRevenueQuota[date] -= int64(log.Quota)
		}
	}
	for date, quota := range trendRevenueQuota {
		trendMap[date].RevenueCents = int64(common.QuotaRound(float64(quota) / common.QuotaPerUnit * 100))
	}
	for _, expense := range expenses {
		date := time.Unix(expense.OccurredAt, 0).Format("2006-01-02")
		point := trendMap[date]
		if point == nil {
			point = &LedgerTrendPoint{Date: date}
			trendMap[date] = point
		}
		point.ExpenseCents += expense.Amount
	}
	trend := make([]LedgerTrendPoint, 0, len(trendMap))
	for _, point := range trendMap {
		trend = append(trend, *point)
	}
	sort.Slice(trend, func(i, j int) bool { return trend[i].Date < trend[j].Date })
	revenueQuota := revenue.Consume - revenue.Refund
	// QuotaPerUnit represents one unit of the default USD billing currency.
	revenueCents := common.QuotaRound(float64(revenueQuota) / common.QuotaPerUnit * 100)
	var trendRevenueCents int64
	for _, point := range trend {
		trendRevenueCents += point.RevenueCents
	}
	// Allocate the unavoidable sub-cent rounding remainder so the trend total
	// always matches the headline revenue for the selected range.
	if remainder := int64(revenueCents) - trendRevenueCents; remainder != 0 {
		for i := len(trend) - 1; i >= 0; i-- {
			if trendRevenueQuota[trend[i].Date] != 0 {
				trend[i].RevenueCents += remainder
				break
			}
		}
	}
	for i := range trend {
		trend[i].ProfitCents = trend[i].RevenueCents - trend[i].ExpenseCents
	}
	return LedgerSummary{
		RevenueQuota: revenueQuota,
		RevenueCents: int64(revenueCents),
		Expenses:     total,
		Profit:       int64(revenueCents) - total,
		Currency:     "USD",
		ExpensesList: expenses,
		Trend:        trend,
	}, nil
}
