package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"

	"github.com/stretchr/testify/require"
)

// TestFormatUserLogsStripsAdminInfo verifies request audit and quota saturation
// fields are removed from non-admin log views with the whole admin_info object.
func TestFormatUserLogsStripsAdminInfo(t *testing.T) {
	other := common.MapToJsonStr(map[string]interface{}{
		"model_price": 0.004,
		"admin_info": map[string]interface{}{
			"request_body": `{"messages":[{"content":"private"}]}`,
			"quota_saturation": map[string]interface{}{
				"op":      "QuotaFromDecimal",
				"kind":    "overflow",
				"clamped": common.MaxQuota,
			},
		},
	})
	logs := []*Log{{Other: other}}

	formatUserLogs(logs, 0)

	parsed, err := common.StrToMap(logs[0].Other)
	require.NoError(t, err)
	_, hasAdminInfo := parsed["admin_info"]
	require.False(t, hasAdminInfo, "admin_info (and nested quota_saturation) must be stripped for non-admin views")
	// Non-admin billing fields remain visible.
	require.Contains(t, parsed, "model_price")
}

func TestFormatAdminLogsStripsRequestBodyForNonRootAdmin(t *testing.T) {
	other := common.MapToJsonStr(map[string]interface{}{
		"admin_info": map[string]interface{}{
			"request_body":                `{"messages":[{"content":"private"}]}`,
			"request_body_truncated":      true,
			"request_body_omitted_reason": "too_large",
			"usage_billing_path":          "local",
			"local_count_tokens":          true,
		},
	})
	logs := []*Log{{Other: other}}

	formatAdminLogs(logs, common.RoleAdminUser)

	parsed, err := common.StrToMap(logs[0].Other)
	require.NoError(t, err)
	adminInfo := parsed["admin_info"].(map[string]interface{})
	require.NotContains(t, adminInfo, "request_body")
	require.NotContains(t, adminInfo, "request_body_truncated")
	require.NotContains(t, adminInfo, "request_body_omitted_reason")
	require.Equal(t, "local", adminInfo["usage_billing_path"])
	require.Equal(t, true, adminInfo["local_count_tokens"])
}

func TestFormatAdminLogsKeepsRequestBodyForRoot(t *testing.T) {
	other := common.MapToJsonStr(map[string]interface{}{
		"admin_info": map[string]interface{}{
			"request_body": `{"messages":[{"content":"private"}]}`,
		},
	})
	logs := []*Log{{Other: other}}

	formatAdminLogs(logs, common.RoleRootUser)

	parsed, err := common.StrToMap(logs[0].Other)
	require.NoError(t, err)
	adminInfo := parsed["admin_info"].(map[string]interface{})
	require.Contains(t, adminInfo, "request_body")
}
