package model

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestGetLedgerSummaryCombinesConsumptionRefundsAndExpenses(t *testing.T) {
	previousDB, previousLogDB := DB, LOG_DB
	previousMainDatabaseType, previousLogDatabaseType := common.MainDatabaseType(), common.LogDatabaseType()
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	DB, LOG_DB = db, db
	require.NoError(t, db.AutoMigrate(&Log{}, &LedgerExpense{}))
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() {
		DB, LOG_DB = previousDB, previousLogDB
		common.SetDatabaseTypes(previousMainDatabaseType, previousLogDatabaseType)
		_ = sqlDB.Close()
	})

	firstDay := time.Date(2026, time.August, 21, 12, 0, 0, 0, time.Local).Unix()
	secondDay := time.Date(2026, time.August, 22, 12, 0, 0, 0, time.Local).Unix()
	quotaPerDollar := int(common.QuotaPerUnit)
	require.NoError(t, db.Create(&[]Log{
		{CreatedAt: firstDay, Type: LogTypeConsume, Quota: 2 * quotaPerDollar},
		{CreatedAt: firstDay, Type: LogTypeRefund, Quota: quotaPerDollar / 2},
		{CreatedAt: secondDay, Type: LogTypeConsume, Quota: quotaPerDollar},
	}).Error)
	require.NoError(t, db.Create(&[]LedgerExpense{
		{OccurredAt: firstDay, Category: "upstream_recharge", Amount: 75, Currency: "USD"},
		{OccurredAt: secondDay, Category: "upstream_recharge", Amount: 125, Currency: "USD"},
	}).Error)

	summary, err := GetLedgerSummary(firstDay-1, secondDay+1)
	require.NoError(t, err)

	assert.EqualValues(t, 2*quotaPerDollar+quotaPerDollar/2, summary.RevenueQuota)
	assert.EqualValues(t, 250, summary.RevenueCents)
	assert.EqualValues(t, 200, summary.Expenses)
	assert.EqualValues(t, 50, summary.Profit)
	require.Len(t, summary.ExpensesList, 2)
	require.Len(t, summary.Trend, 2)
	assert.Equal(t, time.Unix(firstDay, 0).Format("2006-01-02"), summary.Trend[0].Date)
	assert.EqualValues(t, 150, summary.Trend[0].RevenueCents)
	assert.EqualValues(t, 75, summary.Trend[0].ExpenseCents)
	assert.EqualValues(t, 75, summary.Trend[0].ProfitCents)
	assert.Equal(t, time.Unix(secondDay, 0).Format("2006-01-02"), summary.Trend[1].Date)
	assert.EqualValues(t, 100, summary.Trend[1].RevenueCents)
	assert.EqualValues(t, 125, summary.Trend[1].ExpenseCents)
	assert.EqualValues(t, -25, summary.Trend[1].ProfitCents)
}

func TestGetLedgerSummarySupportsUnboundedStartForTotalView(t *testing.T) {
	previousDB, previousLogDB := DB, LOG_DB
	previousMainDatabaseType, previousLogDatabaseType := common.MainDatabaseType(), common.LogDatabaseType()
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	DB, LOG_DB = db, db
	require.NoError(t, db.AutoMigrate(&Log{}, &LedgerExpense{}))
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() {
		DB, LOG_DB = previousDB, previousLogDB
		common.SetDatabaseTypes(previousMainDatabaseType, previousLogDatabaseType)
		_ = sqlDB.Close()
	})

	occurredAt := time.Date(2025, time.January, 2, 12, 0, 0, 0, time.Local).Unix()
	require.NoError(t, db.Create(&Log{
		CreatedAt: occurredAt,
		Type:      LogTypeConsume,
		Quota:     int(common.QuotaPerUnit),
	}).Error)
	require.NoError(t, db.Create(&LedgerExpense{
		OccurredAt: occurredAt,
		Category:   "upstream_recharge",
		Amount:     40,
		Currency:   "USD",
	}).Error)

	summary, err := GetLedgerSummary(-1, occurredAt+1)
	require.NoError(t, err)
	assert.EqualValues(t, 100, summary.RevenueCents)
	assert.EqualValues(t, 40, summary.Expenses)
	assert.EqualValues(t, 60, summary.Profit)
	require.Len(t, summary.Trend, 1)
}

func TestGetLedgerSummaryKeepsTrendRevenueConsistentWithTotalRounding(t *testing.T) {
	previousDB, previousLogDB := DB, LOG_DB
	previousMainDatabaseType, previousLogDatabaseType := common.MainDatabaseType(), common.LogDatabaseType()
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	DB, LOG_DB = db, db
	require.NoError(t, db.AutoMigrate(&Log{}, &LedgerExpense{}))
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() {
		DB, LOG_DB = previousDB, previousLogDB
		common.SetDatabaseTypes(previousMainDatabaseType, previousLogDatabaseType)
		_ = sqlDB.Close()
	})

	firstDay := time.Date(2026, time.August, 23, 12, 0, 0, 0, time.Local).Unix()
	secondDay := time.Date(2026, time.August, 24, 12, 0, 0, 0, time.Local).Unix()
	// Each entry is 0.4 cents. Rounding by day would report zero, while the
	// aggregate is 0.8 cents and must report one cent.
	subCentQuota := int(common.QuotaPerUnit * 4 / 1000)
	require.NoError(t, db.Create(&[]Log{
		{CreatedAt: firstDay, Type: LogTypeConsume, Quota: subCentQuota},
		{CreatedAt: secondDay, Type: LogTypeConsume, Quota: subCentQuota},
	}).Error)

	summary, err := GetLedgerSummary(firstDay-1, secondDay+1)
	require.NoError(t, err)
	assert.EqualValues(t, 1, summary.RevenueCents)
	require.Len(t, summary.Trend, 2)
	var trendRevenue int64
	for _, point := range summary.Trend {
		trendRevenue += point.RevenueCents
	}
	assert.EqualValues(t, summary.RevenueCents, trendRevenue)
}
