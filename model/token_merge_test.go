package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestMergeTokenConsumptionMovesCurrentAndTotalUsage(t *testing.T) {
	require.NoError(t, DB.AutoMigrate(&Token{}))
	require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&Token{}).Error)
	t.Cleanup(func() {
		require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&Token{}).Error)
	})

	require.NoError(t, DB.Create(&[]Token{
		{Id: 301, UserId: 30, Key: "merge-source", Quota: 1000, RemainQuota: 200, UsedQuota: 800, TotalUsedQuota: 1200, Status: common.TokenStatusEnabled},
		{Id: 302, UserId: 30, Key: "merge-target", Quota: 2000, RemainQuota: 1500, UsedQuota: 500, TotalUsedQuota: 700, Status: common.TokenStatusEnabled},
		{Id: 303, UserId: 31, Key: "merge-other-user", TotalUsedQuota: 900},
	}).Error)

	require.NoError(t, MergeTokenConsumption(30, 301, 302))

	var source, target Token
	require.NoError(t, DB.First(&source, 301).Error)
	require.NoError(t, DB.First(&target, 302).Error)
	assert.Equal(t, int64(0), source.TotalUsedQuota)
	assert.Equal(t, common.TokenStatusDisabled, source.Status)
	assert.Equal(t, 1000, source.Quota)
	assert.Equal(t, 1000, source.RemainQuota)
	assert.Equal(t, 0, source.UsedQuota)
	assert.Equal(t, int64(1900), target.TotalUsedQuota)
	assert.Equal(t, 2000, target.Quota)
	assert.Equal(t, 700, target.RemainQuota)
	assert.Equal(t, 1300, target.UsedQuota)

	assert.Error(t, MergeTokenConsumption(30, 301, 303))
}

func TestUpdateTokenUsedQuotaRecalculatesRemainingWithoutChangingTotal(t *testing.T) {
	require.NoError(t, DB.AutoMigrate(&Token{}))
	require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&Token{}).Error)
	t.Cleanup(func() {
		require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&Token{}).Error)
	})

	require.NoError(t, DB.Create(&Token{Id: 401, UserId: 40, Key: "used-quota", Quota: 1000, RemainQuota: 700, UsedQuota: 300, TotalUsedQuota: 1500, Status: common.TokenStatusEnabled}).Error)
	require.NoError(t, UpdateTokenUsedQuota(40, 401, 600))

	var token Token
	require.NoError(t, DB.First(&token, 401).Error)
	assert.Equal(t, 600, token.UsedQuota)
	assert.Equal(t, 400, token.RemainQuota)
	assert.Equal(t, int64(1500), token.TotalUsedQuota)
	assert.Equal(t, common.TokenStatusEnabled, token.Status)
}

func TestUpdateTokenUsedQuotaRejectsOverQuota(t *testing.T) {
	require.NoError(t, DB.AutoMigrate(&Token{}))
	require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&Token{}).Error)
	t.Cleanup(func() {
		require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&Token{}).Error)
	})
	require.NoError(t, DB.Create(&Token{Id: 402, UserId: 40, Key: "used-quota-limit", Quota: 1000}).Error)
	assert.Error(t, UpdateTokenUsedQuota(40, 402, 1001))
}
