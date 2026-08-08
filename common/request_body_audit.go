package common

import (
	"bytes"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
)

const (
	RequestBodyAuditMaxSourceBytes = 8 << 20
	RequestBodyAuditMaxStoredBytes = 16 << 10
	requestBodyAuditMaxStringBytes = 2 << 10
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
	"instructions":      {},
}

var requestBodyAuditUserRequestMarkers = []string{
	"## My request for Codex:",
	"## My request:",
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
				delete(typedValue, key)
				*truncated = true
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
	if len(values) == 0 {
		return values
	}

	selectedIndex := len(values) - 1
	for index := len(values) - 1; index >= 0; index-- {
		if requestBodyAuditMessageRole(values[index]) == "user" {
			selectedIndex = index
			break
		}
	}
	if len(values) > 1 {
		*truncated = true
	}
	return []interface{}{sanitizeRequestBodyAuditMessage(values[selectedIndex], truncated)}
}

func sanitizeRequestBodyAuditMessage(value interface{}, truncated *bool) interface{} {
	message, ok := value.(map[string]interface{})
	if !ok {
		return sanitizeRequestBodyAuditValue(value, "", truncated)
	}

	compact := make(map[string]interface{})
	for _, key := range []string{"role", "type", "name", "content", "text", "prompt"} {
		if child, exists := message[key]; exists {
			if key == "content" || key == "text" || key == "prompt" {
				child = sanitizeRequestBodyAuditUserContent(child, truncated)
			}
			compact[key] = sanitizeRequestBodyAuditValue(child, key, truncated)
		}
	}
	if len(compact) == 0 {
		return sanitizeRequestBodyAuditValue(value, "", truncated)
	}
	return compact
}

func sanitizeRequestBodyAuditUserContent(value interface{}, truncated *bool) interface{} {
	switch typedValue := value.(type) {
	case string:
		sanitizedValue := typedValue
		markerIndex := -1
		markerLength := 0
		for _, marker := range requestBodyAuditUserRequestMarkers {
			if index := strings.LastIndex(sanitizedValue, marker); index > markerIndex {
				markerIndex = index
				markerLength = len(marker)
			}
		}
		if markerIndex >= 0 {
			sanitizedValue = sanitizedValue[markerIndex+markerLength:]
		} else {
			const contextStart = "<in-app-browser-context"
			const contextEnd = "</in-app-browser-context>"
			for {
				startIndex := strings.Index(sanitizedValue, contextStart)
				if startIndex < 0 {
					break
				}
				endOffset := strings.Index(sanitizedValue[startIndex:], contextEnd)
				if endOffset < 0 {
					sanitizedValue = sanitizedValue[:startIndex]
					break
				}
				endIndex := startIndex + endOffset + len(contextEnd)
				sanitizedValue = sanitizedValue[:startIndex] + sanitizedValue[endIndex:]
			}
		}
		sanitizedValue = strings.TrimSpace(sanitizedValue)
		if sanitizedValue != typedValue {
			*truncated = true
		}
		return sanitizedValue
	case []interface{}:
		for index, child := range typedValue {
			typedValue[index] = sanitizeRequestBodyAuditUserContent(child, truncated)
		}
	case map[string]interface{}:
		for key, child := range typedValue {
			typedValue[key] = sanitizeRequestBodyAuditUserContent(child, truncated)
		}
	}
	return value
}

func requestBodyAuditMessageRole(value interface{}) string {
	message, ok := value.(map[string]interface{})
	if !ok {
		return ""
	}
	role, ok := message["role"].(string)
	if !ok {
		return ""
	}
	return strings.ToLower(role)
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
