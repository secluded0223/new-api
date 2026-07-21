package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestBatchUpdateTokenGroupOnlyUpdatesOwnedTokens(t *testing.T) {
	require.NoError(t, DB.AutoMigrate(&Token{}))
	require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&Token{}).Error)
	t.Cleanup(func() {
		require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&Token{}).Error)
	})

	tokens := []Token{
		{Id: 101, UserId: 10, Key: "batch-group-owned-1", Group: "default", CrossGroupRetry: true},
		{Id: 102, UserId: 10, Key: "batch-group-owned-2", Group: "default", CrossGroupRetry: true},
		{Id: 103, UserId: 11, Key: "batch-group-other-user", Group: "default", CrossGroupRetry: true},
	}
	require.NoError(t, DB.Create(&tokens).Error)

	count, err := BatchUpdateTokenGroup([]int{101, 102, 103}, 10, "premium")
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	var updated []Token
	require.NoError(t, DB.Order("id").Find(&updated, []int{101, 102, 103}).Error)
	require.Len(t, updated, 3)
	assert.Equal(t, "premium", updated[0].Group)
	assert.False(t, updated[0].CrossGroupRetry)
	assert.Equal(t, "premium", updated[1].Group)
	assert.False(t, updated[1].CrossGroupRetry)
	assert.Equal(t, "default", updated[2].Group)
	assert.True(t, updated[2].CrossGroupRetry)
}

func TestBatchUpdateTokenGroupPreservesAutoRetry(t *testing.T) {
	require.NoError(t, DB.AutoMigrate(&Token{}))
	require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&Token{}).Error)
	t.Cleanup(func() {
		require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&Token{}).Error)
	})

	token := Token{
		Id:              104,
		UserId:          10,
		Key:             "batch-group-auto",
		Group:           "default",
		CrossGroupRetry: true,
	}
	require.NoError(t, DB.Create(&token).Error)

	count, err := BatchUpdateTokenGroup([]int{104}, 10, "auto")
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	var updated Token
	require.NoError(t, DB.First(&updated, 104).Error)
	assert.Equal(t, "auto", updated.Group)
	assert.True(t, updated.CrossGroupRetry)
}
