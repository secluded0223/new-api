package controller

import (
	"errors"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
)

type adminTokenMergeRequest struct {
	SourceID int `json:"source_id" binding:"required"`
	TargetID int `json:"target_id" binding:"required"`
}

func getAdminTokenTarget(c *gin.Context) (int, error) {
	userID, err := strconv.Atoi(c.Param("id"))
	if err != nil || userID <= 0 {
		return 0, errors.New("invalid user id")
	}
	target, err := model.GetUserById(userID, false)
	if err != nil {
		return 0, err
	}
	if !canManageTargetRole(c.GetInt("role"), target.Role) {
		return 0, errors.New("insufficient permission for target user")
	}
	return userID, nil
}

func GetAdminUserTokens(c *gin.Context) {
	userID, err := getAdminTokenTarget(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	limit := operation_setting.GetMaxUserTokens()
	if limit <= 0 || limit > 1000 {
		limit = 1000
	}
	tokens, err := model.GetAllUserTokens(userID, 0, limit)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, buildMaskedTokenResponses(tokens))
}

func MergeAdminUserTokenConsumption(c *gin.Context) {
	userID, err := getAdminTokenTarget(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	var request adminTokenMergeRequest
	if err := c.ShouldBindJSON(&request); err != nil || request.SourceID == request.TargetID {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	if err := model.MergeTokenConsumption(userID, request.SourceID, request.TargetID); err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAuditFor(c, userID, "token.merge", map[string]interface{}{
		"source_id": request.SourceID,
		"target_id": request.TargetID,
	})
	common.ApiSuccess(c, nil)
}
