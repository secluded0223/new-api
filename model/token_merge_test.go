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

func TestAllocateTokenQuotaDistributesOnlyThePositiveDifference(t *testing.T) {
	require.NoError(t, DB.AutoMigrate(&User{}, &Token{}))
	require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&Token{}).Error)
	require.NoError(t, DB.Unscoped().Where("id = ?", 501).Delete(&User{}).Error)
	t.Cleanup(func() {
		require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&Token{}).Error)
		require.NoError(t, DB.Unscoped().Where("id = ?", 501).Delete(&User{}).Error)
	})

	require.NoError(t, DB.Create(&User{Id: 501, Username: "balance-user", Password: "password", Quota: 3000, UsedQuota: 1000}).Error)
	require.NoError(t, DB.Create(&[]Token{
		{Id: 511, UserId: 501, Key: "balance-a", Quota: 1000, RemainQuota: 900, UsedQuota: 100},
		{Id: 512, UserId: 501, Key: "balance-b", Quota: 1000, RemainQuota: 800, UsedQuota: 200},
	}).Error)

	require.NoError(t, AllocateTokenQuota(501, []TokenQuotaAllocation{
		{TokenID: 511, Quota: 1000},
		{TokenID: 512, Quota: 1000},
	}))

	var first, second Token
	require.NoError(t, DB.First(&first, 511).Error)
	require.NoError(t, DB.First(&second, 512).Error)
	assert.Equal(t, 2000, first.Quota)
	assert.Equal(t, 1900, first.RemainQuota)
	assert.Equal(t, 100, first.UsedQuota)
	assert.Equal(t, 2000, second.Quota)
	assert.Equal(t, 1800, second.RemainQuota)
	assert.Equal(t, 200, second.UsedQuota)
}

func TestAllocateTokenQuotaAllowsPartialDifference(t *testing.T) {
	require.NoError(t, DB.AutoMigrate(&User{}, &Token{}))
	require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&Token{}).Error)
	require.NoError(t, DB.Unscoped().Where("id = ?", 502).Delete(&User{}).Error)
	t.Cleanup(func() {
		require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&Token{}).Error)
		require.NoError(t, DB.Unscoped().Where("id = ?", 502).Delete(&User{}).Error)
	})

	require.NoError(t, DB.Create(&User{Id: 502, Username: "partial-balance-user", Password: "password", Quota: 2500, UsedQuota: 500}).Error)
	require.NoError(t, DB.Create(&Token{Id: 521, UserId: 502, Key: "partial-balance", Quota: 1000, RemainQuota: 1000}).Error)

	require.NoError(t, AllocateTokenQuota(502, []TokenQuotaAllocation{{TokenID: 521, Quota: 500}}))

	var token Token
	require.NoError(t, DB.First(&token, 521).Error)
	assert.Equal(t, 1500, token.Quota)
	assert.Equal(t, 1500, token.RemainQuota)
}
