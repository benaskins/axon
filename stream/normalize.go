package stream

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// NormalizeArguments coerces tool call argument values to match the
// JSON Schema types declared in the tool's parameter schema. LLMs
// sometimes return numbers as strings, bools as strings, etc.
// Unknown or missing properties are left unchanged.
//
// types maps property names to their JSON Schema type strings
// (e.g. "number", "boolean", "string", "array", "object").
func NormalizeArguments(args map[string]any, types map[string]string) map[string]any {
	if len(args) == 0 || len(types) == 0 {
		return args
	}

	out := make(map[string]any, len(args))
	for k, v := range args {
		typ, ok := types[k]
		if !ok {
			out[k] = v
			continue
		}
		out[k] = coerce(v, typ)
	}
	return out
}

// coerce attempts to convert v to the target JSON Schema type.
// If conversion fails, the original value is returned unchanged.
func coerce(v any, targetType string) any {
	switch targetType {
	case "number", "integer":
		return coerceToNumber(v)
	case "boolean":
		return coerceToBool(v)
	case "string":
		return coerceToString(v)
	case "array":
		return coerceToArray(v)
	case "object":
		return coerceToObject(v)
	default:
		return v
	}
}

func coerceToNumber(v any) any {
	switch n := v.(type) {
	case float64, float32, int, int64:
		return v
	case string:
		if f, err := strconv.ParseFloat(n, 64); err == nil {
			return f
		}
	}
	return v
}

func coerceToBool(v any) any {
	switch b := v.(type) {
	case bool:
		return v
	case string:
		if parsed, err := strconv.ParseBool(b); err == nil {
			return parsed
		}
	}
	return v
}

func coerceToString(v any) any {
	switch v.(type) {
	case string:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}

func coerceToArray(v any) any {
	switch v := v.(type) {
	case []any:
		return v
	case string:
		var arr []any
		if json.Unmarshal([]byte(v), &arr) == nil {
			return arr
		}
	}
	return v
}

func coerceToObject(v any) any {
	switch v := v.(type) {
	case map[string]any:
		return v
	case string:
		var obj map[string]any
		if json.Unmarshal([]byte(v), &obj) == nil {
			return obj
		}
	}
	return v
}
