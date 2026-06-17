package i18n

import "strings"

func getNestedKey(data map[string]interface{}, key string) (interface{}, bool) {
	parts := strings.Split(key, ".")
	current := data

	for i, part := range parts {
		val, ok := current[part]
		if !ok {
			return nil, false
		}
		if i == len(parts)-1 {
			return val, true
		}
		next, ok := val.(map[string]interface{})
		if !ok {
			return nil, false
		}
		current = next
	}
	return nil, false
}

func flatten(input map[string]interface{}, prefix string) map[string]string {
	result := make(map[string]string)

	for key, value := range input {
		fullKey := prefix + "." + key
		switch val := value.(type) {
		case string:
			result[fullKey] = val
		case map[string]interface{}:
			nested := flatten(val, fullKey)
			for k, v := range nested {
				result[k] = v
			}
		}
	}

	return result
}
