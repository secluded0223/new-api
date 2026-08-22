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
		cents := int64(common.QuotaRound(float64(log.Quota) / common.QuotaPerUnit * 100))
		if log.Type == LogTypeConsume {
			point.RevenueCents += cents
		} else {
			point.RevenueCents -= cents
		}
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
		point.ProfitCents = point.RevenueCents - point.ExpenseCents
		trend = append(trend, *point)
	}
	sort.Slice(trend, func(i, j int) bool { return trend[i].Date < trend[j].Date })
	revenueQuota := revenue.Consume - revenue.Refund
	// QuotaPerUnit represents one unit of the default USD billing currency.
	revenueCents := common.QuotaRound(float64(revenueQuota) / common.QuotaPerUnit * 100)
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
