package common

import (
	"bytes"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
)

const (
	RequestBodyAuditMaxSourceBytes = 1 << 20
	RequestBodyAuditMaxStoredBytes = 16 << 10
)

var requestBodyAuditSensitiveKeys = map[string]struct{}{
	"access_token":  {},
	"api_key":       {},
	"apikey":        {},
	"authorization": {},
	"client_secret": {},
	"password":      {},
	"private_key":   {},
	"secret":        {},
	"token":         {},
}

var requestBodyAuditBinaryKeys = map[string]struct{}{
	"audio":    {},
	"b64_json": {},
	"data":     {},
	"file":     {},
	"image":    {},
}

func AttachRequestBodyAudit(c *gin.Context, other map[string]interface{}) {
	if !RequestBodyLogEnabled || c == nil || other == nil {
		return
	}

	requestBody, bodySize, found := cachedRequestBody(c)
	if !found || bodySize == 0 {
		return
	}

	adminInfo := requestBodyAuditAdminInfo(other)
	if bodySize > RequestBodyAuditMaxSourceBytes {
		adminInfo["request_body_omitted_reason"] = "too_large"
		return
	}

	trimmedBody := bytes.TrimSpace(requestBody)
	if len(trimmedBody) == 0 || (trimmedBody[0] != '{' && trimmedBody[0] != '[') {
		return
	}

	var payload interface{}
	if err := Unmarshal(trimmedBody, &payload); err != nil {
		adminInfo["request_body_omitted_reason"] = "invalid_json"
		return
	}

	sanitizedBody, err := Marshal(sanitizeRequestBodyAuditValue(payload))
	if err != nil {
		return
	}

	if len(sanitizedBody) > RequestBodyAuditMaxStoredBytes {
		adminInfo["request_body"] = truncateUTF8Bytes(string(sanitizedBody), RequestBodyAuditMaxStoredBytes)
		adminInfo["request_body_truncated"] = true
		return
	}
	adminInfo["request_body"] = string(sanitizedBody)
}

func cachedRequestBody(c *gin.Context) ([]byte, int64, bool) {
	if storageValue, exists := c.Get(KeyBodyStorage); exists {
		if storage, ok := storageValue.(BodyStorage); ok && storage != nil {
			bodySize := storage.Size()
			if bodySize > RequestBodyAuditMaxSourceBytes {
				return nil, bodySize, true
			}
			requestBody, err := storage.Bytes()
			return requestBody, bodySize, err == nil
		}
	}

	if bodyValue, exists := c.Get(KeyRequestBody); exists {
		if requestBody, ok := bodyValue.([]byte); ok {
			return requestBody, int64(len(requestBody)), true
		}
	}
	return nil, 0, false
}

func requestBodyAuditAdminInfo(other map[string]interface{}) map[string]interface{} {
	if adminInfo, ok := other["admin_info"].(map[string]interface{}); ok {
		return adminInfo
	}
	adminInfo := make(map[string]interface{})
	other["admin_info"] = adminInfo
	return adminInfo
}

func sanitizeRequestBodyAuditValue(value interface{}) interface{} {
	switch typedValue := value.(type) {
	case map[string]interface{}:
		for key, child := range typedValue {
			normalizedKey := strings.ReplaceAll(strings.ToLower(key), "-", "_")
			_, sensitive := requestBodyAuditSensitiveKeys[normalizedKey]
			if sensitive || strings.HasSuffix(normalizedKey, "_token") ||
				strings.HasSuffix(normalizedKey, "_secret") ||
				strings.HasSuffix(normalizedKey, "_password") ||
				strings.HasSuffix(normalizedKey, "_api_key") {
				typedValue[key] = "[redacted]"
				continue
			}
			if stringValue, ok := child.(string); ok && len(stringValue) > 1024 {
				if _, binary := requestBodyAuditBinaryKeys[normalizedKey]; binary {
					typedValue[key] = "[binary data omitted]"
					continue
				}
			}
			typedValue[key] = sanitizeRequestBodyAuditValue(child)
		}
	case []interface{}:
		for index, child := range typedValue {
			typedValue[index] = sanitizeRequestBodyAuditValue(child)
		}
	case string:
		lowerValue := strings.ToLower(typedValue)
		if strings.HasPrefix(lowerValue, "data:") && strings.Contains(lowerValue, ";base64,") {
			return "[binary data omitted]"
		}
	}
	return value
}

func truncateUTF8Bytes(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	for limit > 0 && !utf8.ValidString(value[:limit]) {
		limit--
	}
	return value[:limit]
}
