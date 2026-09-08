package controller

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

func GetLedger(c *gin.Context) {
	start, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	end, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	summary, err := model.GetLedgerSummary(start, end)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": summary})
}

func GetLedgerMonthly(c *gin.Context) {
	year := time.Now().Year()
	if value := c.Query("year"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1970 || parsed > 2100 {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "年份无效"})
			return
		}
		year = parsed
	}
	summary, err := model.GetLedgerMonthlySummary(year)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": summary})
}

func CreateLedgerExpense(c *gin.Context) {
	var input model.LedgerExpenseInput
	if err := common.DecodeJson(c.Request.Body, &input); err != nil {
		common.ApiError(c, err)
		return
	}
	if input.Amount <= 0 || input.Amount > 2_000_000_000 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "金额必须大于 0 且不超过 20,000,000"})
		return
	}
	input.Category = strings.TrimSpace(input.Category)
	if input.Category == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "请输入支出分类"})
		return
	}
	if len(input.Category) > 64 || len(input.Note) > 255 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "文本长度超过限制"})
		return
	}
	if input.Currency == "" {
		input.Currency = "USD"
	}
	if input.Currency != "USD" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "当前台账仅支持 USD"})
		return
	}
	if input.OccurredAt == 0 {
		input.OccurredAt = time.Now().Unix()
	}
	expense := &model.LedgerExpense{
		OccurredAt: input.OccurredAt,
		Category:   input.Category,
		Amount:     input.Amount,
		Currency:   input.Currency,
		Note:       strings.TrimSpace(input.Note),
		CreatedAt:  time.Now().Unix(),
	}
	if err := model.DB.Create(expense).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": expense})
}
