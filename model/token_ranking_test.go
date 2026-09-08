package model

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestGetUserTokenUsageRankingScopesAndSortsUsage(t *testing.T) {
	previousDB, previousLogDB := DB, LOG_DB
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	DB, LOG_DB = db, db
	require.NoError(t, db.AutoMigrate(&Log{}, &Token{}))
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() {
		DB, LOG_DB = previousDB, previousLogDB
		_ = sqlDB.Close()
	})

	now := time.Now().Unix()
	eightDaysAgo := time.Now().AddDate(0, 0, -8).Unix()
	require.NoError(t, db.Create(&[]Token{
		{Id: 1, UserId: 7, Name: "Alpha", Key: "key-alpha"},
		{Id: 2, UserId: 7, Name: "Beta", Key: "key-beta"},
		{Id: 3, UserId: 8, Name: "Other user", Key: "key-other"},
	}).Error)
	require.NoError(t, db.Create(&[]Log{
		{UserId: 7, TokenId: 1, TokenName: "Alpha", PromptTokens: 80, CompletionTokens: 20, Quota: 100, CreatedAt: now, Type: LogTypeConsume},
		{UserId: 7, TokenId: 2, TokenName: "Beta", PromptTokens: 40, CompletionTokens: 10, Quota: 50, CreatedAt: now, Type: LogTypeConsume},
		{UserId: 7, TokenId: 2, TokenName: "Beta", PromptTokens: 10, CompletionTokens: 0, Quota: 10, CreatedAt: eightDaysAgo, Type: LogTypeConsume},
		{UserId: 7, TokenId: 1, TokenName: "Alpha", PromptTokens: 999, CompletionTokens: 999, Quota: 1, CreatedAt: now, Type: LogTypeRefund},
		{UserId: 8, TokenId: 3, TokenName: "Other user", PromptTokens: 900, CompletionTokens: 900, Quota: 1, CreatedAt: now, Type: LogTypeConsume},
	}).Error)

	ranking, err := GetUserTokenUsageRanking(7)
	require.NoError(t, err)
	require.Len(t, ranking.Daily, 2)
	assert.Equal(t, 1, ranking.Daily[0].TokenID)
	assert.EqualValues(t, 100, ranking.Daily[0].Tokens)
	assert.EqualValues(t, 1, ranking.Daily[0].Requests)
	assert.Equal(t, 2, ranking.Daily[1].TokenID)
	require.Len(t, ranking.Weekly, 2)
	assert.EqualValues(t, 50, ranking.Weekly[1].Tokens)
	require.Len(t, ranking.Monthly, 2)
	require.Len(t, ranking.Total, 2)
	assert.EqualValues(t, 60, ranking.Total[1].Tokens)
	assert.EqualValues(t, 100, ranking.Total[0].Quota)
}
