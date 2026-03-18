package stream

import (
	"testing"
)

func TestNormalizeArguments_StringToNumber(t *testing.T) {
	args := map[string]any{"x": "42", "y": "3.14"}
	types := map[string]string{"x": "number", "y": "number"}

	got := NormalizeArguments(args, types)

	if v, ok := got["x"].(float64); !ok || v != 42 {
		t.Errorf("x = %v (%T), want 42 (float64)", got["x"], got["x"])
	}
	if v, ok := got["y"].(float64); !ok || v != 3.14 {
		t.Errorf("y = %v (%T), want 3.14 (float64)", got["y"], got["y"])
	}
}

func TestNormalizeArguments_StringToNumber_Scientific(t *testing.T) {
	args := map[string]any{"charge": "1e-9"}
	types := map[string]string{"charge": "number"}

	got := NormalizeArguments(args, types)

	if v, ok := got["charge"].(float64); !ok || v != 1e-9 {
		t.Errorf("charge = %v (%T), want 1e-9 (float64)", got["charge"], got["charge"])
	}
}

func TestNormalizeArguments_NumberAlreadyCorrect(t *testing.T) {
	args := map[string]any{"x": float64(5)}
	types := map[string]string{"x": "number"}

	got := NormalizeArguments(args, types)

	if v, ok := got["x"].(float64); !ok || v != 5 {
		t.Errorf("x = %v (%T)", got["x"], got["x"])
	}
}

func TestNormalizeArguments_StringToBool(t *testing.T) {
	args := map[string]any{"flag": "true"}
	types := map[string]string{"flag": "boolean"}

	got := NormalizeArguments(args, types)

	if v, ok := got["flag"].(bool); !ok || !v {
		t.Errorf("flag = %v (%T), want true (bool)", got["flag"], got["flag"])
	}
}

func TestNormalizeArguments_BoolAlreadyCorrect(t *testing.T) {
	args := map[string]any{"flag": true}
	types := map[string]string{"flag": "boolean"}

	got := NormalizeArguments(args, types)

	if v, ok := got["flag"].(bool); !ok || !v {
		t.Errorf("flag = %v (%T)", got["flag"], got["flag"])
	}
}

func TestNormalizeArguments_NumberToString(t *testing.T) {
	args := map[string]any{"id": float64(123)}
	types := map[string]string{"id": "string"}

	got := NormalizeArguments(args, types)

	if v, ok := got["id"].(string); !ok || v != "123" {
		t.Errorf("id = %v (%T), want '123' (string)", got["id"], got["id"])
	}
}

func TestNormalizeArguments_BoolToString(t *testing.T) {
	args := map[string]any{"flag": true}
	types := map[string]string{"flag": "string"}

	got := NormalizeArguments(args, types)

	if v, ok := got["flag"].(string); !ok || v != "true" {
		t.Errorf("flag = %v (%T), want 'true' (string)", got["flag"], got["flag"])
	}
}

func TestNormalizeArguments_StringToArray(t *testing.T) {
	args := map[string]any{"items": "[1, 2, 3]"}
	types := map[string]string{"items": "array"}

	got := NormalizeArguments(args, types)

	arr, ok := got["items"].([]any)
	if !ok {
		t.Fatalf("items = %T, want []any", got["items"])
	}
	if len(arr) != 3 {
		t.Errorf("len = %d, want 3", len(arr))
	}
}

func TestNormalizeArguments_ArrayAlreadyCorrect(t *testing.T) {
	args := map[string]any{"items": []any{1.0, 2.0}}
	types := map[string]string{"items": "array"}

	got := NormalizeArguments(args, types)

	arr, ok := got["items"].([]any)
	if !ok || len(arr) != 2 {
		t.Errorf("items = %v (%T)", got["items"], got["items"])
	}
}

func TestNormalizeArguments_StringToObject(t *testing.T) {
	args := map[string]any{"config": `{"key": "value"}`}
	types := map[string]string{"config": "object"}

	got := NormalizeArguments(args, types)

	obj, ok := got["config"].(map[string]any)
	if !ok {
		t.Fatalf("config = %T, want map[string]any", got["config"])
	}
	if obj["key"] != "value" {
		t.Errorf("key = %v", obj["key"])
	}
}

func TestNormalizeArguments_UnknownProperty(t *testing.T) {
	args := map[string]any{"x": "hello", "extra": float64(99)}
	types := map[string]string{"x": "string"}

	got := NormalizeArguments(args, types)

	if got["extra"] != float64(99) {
		t.Errorf("extra = %v, want 99 (unchanged)", got["extra"])
	}
}

func TestNormalizeArguments_UnparseableString(t *testing.T) {
	args := map[string]any{"x": "not-a-number"}
	types := map[string]string{"x": "number"}

	got := NormalizeArguments(args, types)

	if got["x"] != "not-a-number" {
		t.Errorf("x = %v, want 'not-a-number' (unchanged)", got["x"])
	}
}

func TestNormalizeArguments_NilArgs(t *testing.T) {
	got := NormalizeArguments(nil, map[string]string{"x": "number"})
	if got != nil {
		t.Errorf("got %v, want nil", got)
	}
}

func TestNormalizeArguments_EmptyTypes(t *testing.T) {
	args := map[string]any{"x": "42"}
	got := NormalizeArguments(args, nil)

	if got["x"] != "42" {
		t.Errorf("x = %v, want '42' (unchanged)", got["x"])
	}
}

func TestNormalizeArguments_IntegerType(t *testing.T) {
	args := map[string]any{"count": "5"}
	types := map[string]string{"count": "integer"}

	got := NormalizeArguments(args, types)

	if v, ok := got["count"].(float64); !ok || v != 5 {
		t.Errorf("count = %v (%T), want 5 (float64)", got["count"], got["count"])
	}
}
