package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestReorderTokenMovesOwnedTokenAndNormalizesOrder(t *testing.T) {
	require.NoError(t, DB.AutoMigrate(&Token{}))
	require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&Token{}).Error)
	t.Cleanup(func() {
		require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&Token{}).Error)
	})

	tokens := []Token{
		{Id: 201, UserId: 20, Key: "reorder-a", SortOrder: 0},
		{Id: 202, UserId: 20, Key: "reorder-b", SortOrder: 1},
		{Id: 203, UserId: 20, Key: "reorder-c", SortOrder: 2},
		{Id: 204, UserId: 21, Key: "reorder-other-user", SortOrder: 0},
	}
	require.NoError(t, DB.Create(&tokens).Error)

	require.NoError(t, ReorderToken(201, 203, 20, false))

	var ordered []Token
	require.NoError(t, DB.Where("user_id = ?", 20).Order("sort_order").Find(&ordered).Error)
	require.Len(t, ordered, 3)
	assert.Equal(t, []int{202, 203, 201}, []int{ordered[0].Id, ordered[1].Id, ordered[2].Id})
	assert.Equal(t, []int64{0, 1, 2}, []int64{ordered[0].SortOrder, ordered[1].SortOrder, ordered[2].SortOrder})

	err := ReorderToken(201, 204, 20, true)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}
