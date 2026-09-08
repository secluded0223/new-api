package model

import (
	"time"
)

// TokenUsageRankingItem is the aggregated usage for one API key.
type TokenUsageRankingItem struct {
	TokenID   int    `json:"token_id" gorm:"column:token_id"`
	TokenName string `json:"token_name" gorm:"column:token_name"`
	Tokens    int64  `json:"tokens" gorm:"column:tokens"`
	Quota     int64  `json:"quota" gorm:"column:quota"`
	Requests  int64  `json:"requests" gorm:"column:requests"`
}

// TokenUsageRanking contains the four periods shown on the token rankings page.
type TokenUsageRanking struct {
	Daily   []*TokenUsageRankingItem `json:"daily"`
	Weekly  []*TokenUsageRankingItem `json:"weekly"`
	Monthly []*TokenUsageRankingItem `json:"monthly"`
	Total   []*TokenUsageRankingItem `json:"total"`
}

func getTokenUsageRanking(userID int, startTimestamp int64) ([]*TokenUsageRankingItem, error) {
	rows := make([]*TokenUsageRankingItem, 0)
	query := LOG_DB.Table("logs").
		Select("token_id, MAX(token_name) AS token_name, COALESCE(SUM(prompt_tokens), 0) + COALESCE(SUM(completion_tokens), 0) AS tokens, COALESCE(SUM(quota), 0) AS quota, COUNT(*) AS requests").
		Where("user_id = ? AND type = ? AND token_id > ?", userID, LogTypeConsume, 0).
		Group("token_id").
		Order("tokens DESC, token_id ASC")
	if startTimestamp > 0 {
		query = query.Where("created_at >= ?", startTimestamp)
	}
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}

	// Prefer the current name when the log snapshot is empty. Deleted keys keep
	// their historical snapshot and are rendered by the client using their ID.
	tokenIDs := make([]int, 0, len(rows))
	for _, row := range rows {
		if row.TokenName == "" {
			tokenIDs = append(tokenIDs, row.TokenID)
		}
	}
	if len(tokenIDs) == 0 {
		return rows, nil
	}
	var tokens []struct {
		ID   int    `gorm:"column:id"`
		Name string `gorm:"column:name"`
	}
	if err := DB.Model(&Token{}).Select("id, name").Where("user_id = ? AND id IN ?", userID, tokenIDs).Find(&tokens).Error; err != nil {
		return nil, err
	}
	names := make(map[int]string, len(tokens))
	for _, token := range tokens {
		names[token.ID] = token.Name
	}
	for _, row := range rows {
		if row.TokenName == "" {
			row.TokenName = names[row.TokenID]
		}
	}
	return rows, nil
}

// GetUserTokenUsageRanking returns usage rankings scoped to one user.
func GetUserTokenUsageRanking(userID int) (TokenUsageRanking, error) {
	if userID <= 0 {
		return TokenUsageRanking{
			Daily:   make([]*TokenUsageRankingItem, 0),
			Weekly:  make([]*TokenUsageRankingItem, 0),
			Monthly: make([]*TokenUsageRankingItem, 0),
			Total:   make([]*TokenUsageRankingItem, 0),
		}, nil
	}

	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local).Unix()
	weekStart := time.Unix(todayStart, 0).In(time.Local).AddDate(0, 0, -6).Unix()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local).Unix()

	daily, err := getTokenUsageRanking(userID, todayStart)
	if err != nil {
		return TokenUsageRanking{}, err
	}
	weekly, err := getTokenUsageRanking(userID, weekStart)
	if err != nil {
		return TokenUsageRanking{}, err
	}
	monthly, err := getTokenUsageRanking(userID, monthStart)
	if err != nil {
		return TokenUsageRanking{}, err
	}
	total, err := getTokenUsageRanking(userID, 0)
	if err != nil {
		return TokenUsageRanking{}, err
	}
	return TokenUsageRanking{Daily: daily, Weekly: weekly, Monthly: monthly, Total: total}, nil
}
