package stream

import (
	"bytes"
	"encoding/json"
	"strings"
)

const toolCallMaxAccumulate = 2048 // give up after this many bytes

// ToolCallMatcher detects tool call JSON emitted as text content.
type ToolCallMatcher struct{}

func NewToolCallMatcher() *ToolCallMatcher { return &ToolCallMatcher{} }
func (m *ToolCallMatcher) Name() string    { return "tool_call" }

func (m *ToolCallMatcher) Scan(buf []byte, _ string) MatchResult {
	trimmed := bytes.TrimSpace(buf)
	if len(trimmed) == 0 {
		return NoMatch
	}

	// Strip markdown code fence prefix if present
	trimmed = stripCodeFencePrefix(trimmed)

	// Must start with { or [ to be a candidate
	if trimmed[0] != '{' && trimmed[0] != '[' {
		return NoMatch
	}

	// Too large without closing — give up
	if len(trimmed) > toolCallMaxAccumulate {
		return NoMatch
	}

	// Try to parse as complete JSON with a "name" field
	if tryParseToolCallJSON(trimmed) {
		return FullMatch
	}

	// Looks like it could be JSON starting — partial match
	if looksLikeToolCallStart(trimmed) {
		return PartialMatch
	}

	return NoMatch
}

func (m *ToolCallMatcher) Extract(buf []byte) FilterAction {
	trimmed := bytes.TrimSpace(buf)
	trimmed = stripCodeFencePrefix(trimmed)
	trimmed = stripCodeFenceSuffix(trimmed)
	trimmed = bytes.TrimSpace(trimmed)

	calls := parseToolCallJSON(trimmed)
	if len(calls) > 0 {
		return ToolCallAction{Calls: calls}
	}
	return ContinueAction{}
}

// stripCodeFencePrefix removes ```json\n or ``` prefix.
func stripCodeFencePrefix(b []byte) []byte {
	if !bytes.HasPrefix(b, []byte("```")) {
		return b
	}
	rest := b[3:]
	if idx := bytes.IndexByte(rest, '\n'); idx >= 0 {
		return bytes.TrimSpace(rest[idx+1:])
	}
	return bytes.TrimSpace(rest)
}

// stripCodeFenceSuffix removes trailing ```.
func stripCodeFenceSuffix(b []byte) []byte {
	if bytes.HasSuffix(b, []byte("```")) {
		return bytes.TrimSpace(b[:len(b)-3])
	}
	return b
}

// looksLikeToolCallStart checks if partial JSON looks like a tool call forming.
func looksLikeToolCallStart(b []byte) bool {
	s := string(b)
	// Single object starting
	if strings.HasPrefix(s, "{") {
		if len(s) < 10 {
			return true // too short to tell, hold
		}
		return strings.Contains(s, `"name"`)
	}
	// Array starting
	if strings.HasPrefix(s, "[") {
		if len(s) < 12 {
			return true
		}
		return strings.Contains(s, `"name"`)
	}
	return false
}

// tryParseToolCallJSON checks if the buffer contains complete, valid tool call JSON.
func tryParseToolCallJSON(b []byte) bool {
	b = stripCodeFenceSuffix(b)
	b = bytes.TrimSpace(b)

	// Single object
	var single struct {
		Name string `json:"name"`
	}
	if json.Unmarshal(b, &single) == nil && single.Name != "" {
		return true
	}

	// Array
	var arr []struct {
		Name string `json:"name"`
	}
	if json.Unmarshal(b, &arr) == nil && len(arr) > 0 && arr[0].Name != "" {
		return true
	}

	return false
}

// parseToolCallJSON extracts tool calls from JSON bytes.
func parseToolCallJSON(b []byte) []ToolCall {
	// Single object
	var single struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	if json.Unmarshal(b, &single) == nil && single.Name != "" {
		return []ToolCall{{
			Name:      single.Name,
			Arguments: single.Arguments,
		}}
	}

	// Array
	var arr []struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	if json.Unmarshal(b, &arr) == nil && len(arr) > 0 {
		var calls []ToolCall
		for _, item := range arr {
			if item.Name == "" {
				continue
			}
			calls = append(calls, ToolCall{
				Name:      item.Name,
				Arguments: item.Arguments,
			})
		}
		return calls
	}

	return nil
}
