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
	requestBodyAuditMaxStringBytes = 2 << 10
	requestBodyAuditMaxListItems   = 12
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

var requestBodyAuditOmittedKeys = map[string]struct{}{
	"client_metadata":   {},
	"encrypted_content": {},
	"include":           {},
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

	truncated := false
	sanitizedBody, err := Marshal(sanitizeRequestBodyAuditValue(payload, "", &truncated))
	if err != nil {
		return
	}

	if len(sanitizedBody) > RequestBodyAuditMaxStoredBytes {
		adminInfo["request_body"] = truncateUTF8Bytes(string(sanitizedBody), RequestBodyAuditMaxStoredBytes)
		adminInfo["request_body_truncated"] = true
		return
	}
	adminInfo["request_body"] = string(sanitizedBody)
	if truncated {
		adminInfo["request_body_truncated"] = true
	}
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

func sanitizeRequestBodyAuditValue(value interface{}, key string, truncated *bool) interface{} {
	switch typedValue := value.(type) {
	case map[string]interface{}:
		for key, child := range typedValue {
			normalizedKey := strings.ReplaceAll(strings.ToLower(key), "-", "_")
			if _, omitted := requestBodyAuditOmittedKeys[normalizedKey]; omitted {
				typedValue[key] = "[omitted]"
				continue
			}
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
			typedValue[key] = sanitizeRequestBodyAuditValue(child, normalizedKey, truncated)
		}
	case []interface{}:
		if key == "messages" || key == "input" {
			return sanitizeRequestBodyAuditList(typedValue, truncated)
		}
		for index, child := range typedValue {
			typedValue[index] = sanitizeRequestBodyAuditValue(child, key, truncated)
		}
	case string:
		lowerValue := strings.ToLower(typedValue)
		if strings.HasPrefix(lowerValue, "data:") && strings.Contains(lowerValue, ";base64,") {
			return "[binary data omitted]"
		}
		if len(typedValue) > requestBodyAuditMaxStringBytes {
			*truncated = true
			return truncateUTF8Bytes(typedValue, requestBodyAuditMaxStringBytes) + "...[truncated]"
		}
	}
	return value
}

func sanitizeRequestBodyAuditList(values []interface{}, truncated *bool) []interface{} {
	startIndex := 0
	if len(values) > requestBodyAuditMaxListItems {
		startIndex = len(values) - requestBodyAuditMaxListItems
		*truncated = true
	}

	result := make([]interface{}, 0, len(values)-startIndex+1)
	if startIndex > 0 {
		result = append(result, map[string]interface{}{
			"_omitted_previous_items": startIndex,
		})
	}

	for _, value := range values[startIndex:] {
		result = append(result, sanitizeRequestBodyAuditMessage(value, truncated))
	}
	return result
}

func sanitizeRequestBodyAuditMessage(value interface{}, truncated *bool) interface{} {
	message, ok := value.(map[string]interface{})
	if !ok {
		return sanitizeRequestBodyAuditValue(value, "", truncated)
	}

	compact := make(map[string]interface{})
	for _, key := range []string{"role", "type", "name", "content", "text", "prompt"} {
		if child, exists := message[key]; exists {
			compact[key] = sanitizeRequestBodyAuditValue(child, key, truncated)
		}
	}
	if len(compact) == 0 {
		return sanitizeRequestBodyAuditValue(value, "", truncated)
	}
	return compact
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
