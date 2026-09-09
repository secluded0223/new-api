package model

import (
	"sort"
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

func getTokenUsageRanking(userID int, tokens []*Token, startTimestamp int64, useCurrentTotal bool) ([]*TokenUsageRankingItem, error) {
	rows := make([]*TokenUsageRankingItem, 0, len(tokens))
	if len(tokens) == 0 {
		return rows, nil
	}

	tokenIDs := make([]int, 0, len(tokens))
	for _, token := range tokens {
		tokenIDs = append(tokenIDs, token.Id)
	}

	usageRows := make([]*TokenUsageRankingItem, 0, len(tokens))
	query := LOG_DB.Table("logs").
		Select("token_id, COALESCE(SUM(prompt_tokens), 0) + COALESCE(SUM(completion_tokens), 0) AS tokens, COALESCE(SUM(quota), 0) AS quota, COUNT(*) AS requests").
		Where("user_id = ? AND type = ? AND token_id IN ?", userID, LogTypeConsume, tokenIDs).
		Group("token_id")
	if startTimestamp > 0 {
		query = query.Where("created_at >= ?", startTimestamp)
	}
	if err := query.Find(&usageRows).Error; err != nil {
		return nil, err
	}

	usageByTokenID := make(map[int]*TokenUsageRankingItem, len(usageRows))
	for _, row := range usageRows {
		usageByTokenID[row.TokenID] = row
	}
	for _, token := range tokens {
		row := usageByTokenID[token.Id]
		if row == nil {
			row = &TokenUsageRankingItem{TokenID: token.Id}
		}
		row.TokenName = token.Name
		if useCurrentTotal {
			row.Quota = token.TotalUsedQuota
		}
		rows = append(rows, row)
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].Quota != rows[j].Quota {
			return rows[i].Quota > rows[j].Quota
		}
		if rows[i].Tokens != rows[j].Tokens {
			return rows[i].Tokens > rows[j].Tokens
		}
		return rows[i].TokenID < rows[j].TokenID
	})
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
	var tokens []*Token
	if err := DB.Select("id, name, total_used_quota").Where("user_id = ?", userID).Find(&tokens).Error; err != nil {
		return TokenUsageRanking{}, err
	}

	daily, err := getTokenUsageRanking(userID, tokens, todayStart, false)
	if err != nil {
		return TokenUsageRanking{}, err
	}
	weekly, err := getTokenUsageRanking(userID, tokens, weekStart, false)
	if err != nil {
		return TokenUsageRanking{}, err
	}
	monthly, err := getTokenUsageRanking(userID, tokens, monthStart, false)
	if err != nil {
		return TokenUsageRanking{}, err
	}
	total, err := getTokenUsageRanking(userID, tokens, 0, true)
	if err != nil {
		return TokenUsageRanking{}, err
	}
	return TokenUsageRanking{Daily: daily, Weekly: weekly, Monthly: monthly, Total: total}, nil
}
