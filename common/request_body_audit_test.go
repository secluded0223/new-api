package common

import (
	"net/http/httptest"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAttachRequestBodyAuditSanitizesAdminOnlyContent(t *testing.T) {
	previousEnabled := RequestBodyLogEnabled
	RequestBodyLogEnabled = true
	t.Cleanup(func() { RequestBodyLogEnabled = previousEnabled })

	requestBody := []byte(`{"model":"gpt-test","messages":[{"role":"user","content":"hello"}],"api_key":"secret-key","metadata":{"access-token":"secret-token","refresh_token":"refresh-secret"},"image":"data:image/png;base64,AAAA","input_audio":{"data":"` + strings.Repeat("A", 2048) + `"}}`)
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Set(KeyRequestBody, requestBody)
	other := map[string]interface{}{"model_ratio": 1.0}

	AttachRequestBodyAudit(context, other)

	adminInfo, ok := other["admin_info"].(map[string]interface{})
	require.True(t, ok)
	storedBody, ok := adminInfo["request_body"].(string)
	require.True(t, ok)
	assert.Contains(t, storedBody, `"content":"hello"`)
	assert.NotContains(t, storedBody, "secret-key")
	assert.NotContains(t, storedBody, "secret-token")
	assert.NotContains(t, storedBody, "refresh-secret")
	assert.Contains(t, storedBody, `"api_key":"[redacted]"`)
	assert.Contains(t, storedBody, `"image":"[binary data omitted]"`)
	assert.Contains(t, storedBody, `"data":"[binary data omitted]"`)
}

func TestAttachRequestBodyAuditCompactsCodexStyleInput(t *testing.T) {
	previousEnabled := RequestBodyLogEnabled
	RequestBodyLogEnabled = true
	t.Cleanup(func() { RequestBodyLogEnabled = previousEnabled })

	requestBody := []byte(`{
		"client_metadata":{"x-codex-turn-metadata":"private session data"},
		"include":["reasoning.encrypted_content"],
		"model":"gpt-test",
		"input":[
			{"role":"user","content":[{"type":"input_text","text":"old user message"}],"id":"msg_old"},
			{"role":"assistant","content":[{"type":"output_text","text":"assistant context"}],"id":"msg_assistant"},
			{"role":"user","content":[{"type":"input_text","text":"current user message"}],"id":"msg_current"}
		]
	}`)
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Set(KeyRequestBody, requestBody)
	other := map[string]interface{}{}

	AttachRequestBodyAudit(context, other)

	adminInfo := other["admin_info"].(map[string]interface{})
	storedBody := adminInfo["request_body"].(string)
	assert.Contains(t, storedBody, `"text":"current user message"`)
	assert.NotContains(t, storedBody, "old user message")
	assert.NotContains(t, storedBody, "assistant context")
	assert.NotContains(t, storedBody, "private session data")
	assert.NotContains(t, storedBody, "reasoning.encrypted_content")
	assert.NotContains(t, storedBody, `"id":"msg_current"`)
	assert.Equal(t, true, adminInfo["request_body_truncated"])
}

func TestAttachRequestBodyAuditKeepsCurrentInputFromLargeRequest(t *testing.T) {
	previousEnabled := RequestBodyLogEnabled
	RequestBodyLogEnabled = true
	t.Cleanup(func() { RequestBodyLogEnabled = previousEnabled })

	requestBody := []byte(`{
		"model":"gpt-test",
		"padding":"` + strings.Repeat("x", 1<<20+128) + `",
		"input":[
			{"role":"user","content":[{"type":"input_text","text":"current request"}]}
		]
	}`)
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Set(KeyRequestBody, requestBody)
	other := map[string]interface{}{}

	AttachRequestBodyAudit(context, other)

	adminInfo := other["admin_info"].(map[string]interface{})
	storedBody, ok := adminInfo["request_body"].(string)
	require.True(t, ok)
	assert.Contains(t, storedBody, `"text":"current request"`)
	assert.NotEqual(t, "too_large", adminInfo["request_body_omitted_reason"])
}

func TestAttachRequestBodyAuditHonorsDisabledSetting(t *testing.T) {
	previousEnabled := RequestBodyLogEnabled
	RequestBodyLogEnabled = false
	t.Cleanup(func() { RequestBodyLogEnabled = previousEnabled })

	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Set(KeyRequestBody, []byte(`{"messages":[{"content":"private"}]}`))
	other := map[string]interface{}{}

	AttachRequestBodyAudit(context, other)

	assert.NotContains(t, other, "admin_info")
}

func TestAttachRequestBodyAuditBoundsStoredContent(t *testing.T) {
	previousEnabled := RequestBodyLogEnabled
	RequestBodyLogEnabled = true
	t.Cleanup(func() { RequestBodyLogEnabled = previousEnabled })

	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	requestBody := []byte(`{"prompt":"` + strings.Repeat("界", RequestBodyAuditMaxStoredBytes) + `"}`)
	context.Set(KeyRequestBody, requestBody)
	other := map[string]interface{}{}

	AttachRequestBodyAudit(context, other)

	adminInfo := other["admin_info"].(map[string]interface{})
	storedBody := adminInfo["request_body"].(string)
	assert.LessOrEqual(t, len(storedBody), RequestBodyAuditMaxStoredBytes)
	assert.True(t, utf8.ValidString(storedBody))
	assert.Equal(t, true, adminInfo["request_body_truncated"])
}

func TestAttachRequestBodyAuditRejectsOversizedSource(t *testing.T) {
	previousEnabled := RequestBodyLogEnabled
	RequestBodyLogEnabled = true
	t.Cleanup(func() { RequestBodyLogEnabled = previousEnabled })

	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Set(KeyRequestBody, make([]byte, RequestBodyAuditMaxSourceBytes+1))
	other := map[string]interface{}{}

	AttachRequestBodyAudit(context, other)

	adminInfo := other["admin_info"].(map[string]interface{})
	assert.Equal(t, "too_large", adminInfo["request_body_omitted_reason"])
	assert.NotContains(t, adminInfo, "request_body")
}
