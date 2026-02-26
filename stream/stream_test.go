package stream

import (
	"fmt"
	"strings"
	"testing"
)

// collector gathers emitted text for assertions.
type collector struct {
	chunks []string
}

func (c *collector) emit(s string) {
	c.chunks = append(c.chunks, s)
}

func (c *collector) all() string {
	return strings.Join(c.chunks, "")
}

// --- StreamFilter buffer mechanics ---

func TestStreamFilter_PassthroughSmallTokens(t *testing.T) {
	c := &collector{}
	f := NewStreamFilter(c.emit, nil, 200)

	f.Write("Hello ")
	f.Write("world")
	f.Flush()

	if c.all() != "Hello world" {
		t.Errorf("expected 'Hello world', got %q", c.all())
	}
}

func TestStreamFilter_BufferDelaysEmission(t *testing.T) {
	c := &collector{}
	f := NewStreamFilter(c.emit, nil, 20)

	// Write more than maxBuffer to trigger emission
	f.Write("abcdefghijklmnopqrstuvwxyz") // 26 chars, maxBuffer=20

	// Should have emitted overflow (6 chars)
	if len(c.chunks) == 0 {
		t.Fatal("expected emission when buffer exceeds max")
	}
	emitted := c.all()
	if len(emitted) != 6 {
		t.Errorf("expected 6 chars emitted, got %d: %q", len(emitted), emitted)
	}
	if emitted != "abcdef" {
		t.Errorf("expected 'abcdef', got %q", emitted)
	}

	// Flush gets the rest
	f.Flush()
	if c.all() != "abcdefghijklmnopqrstuvwxyz" {
		t.Errorf("expected full alphabet, got %q", c.all())
	}
}

func TestStreamFilter_FlushEmptyBuffer(t *testing.T) {
	c := &collector{}
	f := NewStreamFilter(c.emit, nil, 200)

	action := f.Flush()
	if _, ok := action.(ContinueAction); !ok {
		t.Errorf("expected ContinueAction for empty flush, got %T", action)
	}
	if len(c.chunks) != 0 {
		t.Errorf("expected no emissions, got %d", len(c.chunks))
	}
}

func TestStreamFilter_MultipleSmallWrites(t *testing.T) {
	c := &collector{}
	f := NewStreamFilter(c.emit, nil, 50)

	// Write many small tokens
	for i := 0; i < 20; i++ {
		f.Write(fmt.Sprintf("tok%d ", i))
	}
	f.Flush()

	expected := ""
	for i := 0; i < 20; i++ {
		expected += fmt.Sprintf("tok%d ", i)
	}
	if c.all() != expected {
		t.Errorf("expected all tokens concatenated, got %q", c.all())
	}
}

func TestStreamFilter_PrevTailOverlap(t *testing.T) {
	c := &collector{}
	f := NewStreamFilter(c.emit, nil, 10)

	// Force emission by exceeding buffer
	f.Write("0123456789ABCDEFGHIJ") // 20 chars, maxBuffer=10
	// Should emit first 10 chars
	if f.PrevTail() == "" {
		t.Fatal("expected prevTail to be set after emission")
	}
	// prevTail should be the last overlap chars of emitted text
	if len(f.PrevTail()) > defaultOverlap {
		t.Errorf("prevTail too long: %d", len(f.PrevTail()))
	}
}

// --- ToolCallMatcher ---

func TestToolCallMatcher_SingleObject(t *testing.T) {
	m := NewToolCallMatcher()
	buf := []byte(`{"name": "web_search", "arguments": {"query": "golang"}}`)

	if r := m.Scan(buf, ""); r != FullMatch {
		t.Fatalf("expected FullMatch, got %v", r)
	}

	action := m.Extract(buf)
	tc, ok := action.(ToolCallAction)
	if !ok {
		t.Fatalf("expected ToolCallAction, got %T", action)
	}
	if len(tc.Calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(tc.Calls))
	}
	if tc.Calls[0].Name != "web_search" {
		t.Errorf("expected 'web_search', got %q", tc.Calls[0].Name)
	}
	if tc.Calls[0].Arguments["query"] != "golang" {
		t.Errorf("expected query 'golang', got %v", tc.Calls[0].Arguments["query"])
	}
}

func TestToolCallMatcher_Array(t *testing.T) {
	m := NewToolCallMatcher()
	buf := []byte(`[{"name": "current_time", "arguments": {}}]`)

	if r := m.Scan(buf, ""); r != FullMatch {
		t.Fatalf("expected FullMatch, got %v", r)
	}
	action := m.Extract(buf)
	tc, ok := action.(ToolCallAction)
	if !ok {
		t.Fatalf("expected ToolCallAction, got %T", action)
	}
	if len(tc.Calls) != 1 || tc.Calls[0].Name != "current_time" {
		t.Errorf("unexpected calls: %+v", tc.Calls)
	}
}

func TestToolCallMatcher_MarkdownFenced(t *testing.T) {
	m := NewToolCallMatcher()
	buf := []byte("```json\n{\"name\": \"fetch_page\", \"arguments\": {\"url\": \"https://example.com\", \"question\": \"test\"}}\n```")

	if r := m.Scan(buf, ""); r != FullMatch {
		t.Fatalf("expected FullMatch, got %v", r)
	}
	action := m.Extract(buf)
	tc, ok := action.(ToolCallAction)
	if !ok {
		t.Fatalf("expected ToolCallAction, got %T", action)
	}
	if tc.Calls[0].Name != "fetch_page" {
		t.Errorf("expected 'fetch_page', got %q", tc.Calls[0].Name)
	}
}

func TestToolCallMatcher_PartialJSON(t *testing.T) {
	m := NewToolCallMatcher()

	buf := []byte(`{"name": "web_search", "arguments": {"query": "go`)
	if r := m.Scan(buf, ""); r != PartialMatch {
		t.Errorf("expected PartialMatch for incomplete JSON, got %v", r)
	}
}

func TestToolCallMatcher_RegularText(t *testing.T) {
	m := NewToolCallMatcher()

	buf := []byte("This is just regular text about searching the web.")
	if r := m.Scan(buf, ""); r != NoMatch {
		t.Errorf("expected NoMatch for regular text, got %v", r)
	}
}

func TestToolCallMatcher_BraceInProse(t *testing.T) {
	m := NewToolCallMatcher()

	// Short brace — ambiguous, should hold briefly
	buf := []byte(`{`)
	if r := m.Scan(buf, ""); r != PartialMatch {
		t.Errorf("expected PartialMatch for lone brace, got %v", r)
	}

	// Longer text starting with brace but no "name" key
	buf = []byte(`{some random text that is definitely not JSON and is long enough}`)
	if r := m.Scan(buf, ""); r != NoMatch {
		t.Errorf("expected NoMatch for non-JSON brace text, got %v", r)
	}
}

func TestToolCallMatcher_OversizedNonJSON(t *testing.T) {
	m := NewToolCallMatcher()

	buf := []byte(`{"name": "web_search", "arguments": {"query": "` + strings.Repeat("x", toolCallMaxAccumulate) + `"}}`)
	if r := m.Scan(buf, ""); r != NoMatch {
		t.Errorf("expected NoMatch for oversized buffer, got %v", r)
	}
}

func TestToolCallMatcher_Empty(t *testing.T) {
	m := NewToolCallMatcher()
	if r := m.Scan([]byte(""), ""); r != NoMatch {
		t.Errorf("expected NoMatch for empty, got %v", r)
	}
}

// --- ContentSafetyMatcher ---

func TestContentSafetyMatcher_DirectMatch(t *testing.T) {
	patterns := []ContentSafetyPattern{
		{Name: "test_block", Regex: `blocked_word`},
	}
	m := NewContentSafetyMatcher(patterns)

	if r := m.Scan([]byte("This contains blocked_word here"), ""); r != FullMatch {
		t.Errorf("expected FullMatch, got %v", r)
	}
}

func TestContentSafetyMatcher_CaseInsensitive(t *testing.T) {
	patterns := []ContentSafetyPattern{
		{Name: "test_block", Regex: `blocked_word`},
	}
	m := NewContentSafetyMatcher(patterns)

	if r := m.Scan([]byte("This contains BLOCKED_WORD here"), ""); r != FullMatch {
		t.Errorf("expected FullMatch (case insensitive), got %v", r)
	}
}

func TestContentSafetyMatcher_NoMatch(t *testing.T) {
	patterns := []ContentSafetyPattern{
		{Name: "test_block", Regex: `blocked_word`},
	}
	m := NewContentSafetyMatcher(patterns)

	if r := m.Scan([]byte("This is perfectly safe text"), ""); r != NoMatch {
		t.Errorf("expected NoMatch, got %v", r)
	}
}

func TestContentSafetyMatcher_WithOverlap(t *testing.T) {
	patterns := []ContentSafetyPattern{
		{Name: "test_block", Regex: `blocked`},
	}
	m := NewContentSafetyMatcher(patterns)

	// Buffer contains the rest, prevTail contains start of blocked word
	if r := m.Scan([]byte("ked content here"), "bloc"); r != FullMatch {
		t.Errorf("expected FullMatch with overlap, got %v", r)
	}
}

func TestContentSafetyMatcher_InvalidRegex(t *testing.T) {
	patterns := []ContentSafetyPattern{
		{Name: "bad", Regex: `[invalid`},
		{Name: "good", Regex: `blocked`},
	}
	m := NewContentSafetyMatcher(patterns)

	// Bad pattern skipped, good pattern works
	if r := m.Scan([]byte("this is blocked"), ""); r != FullMatch {
		t.Errorf("expected FullMatch from valid pattern, got %v", r)
	}
	if len(m.patterns) != 1 {
		t.Errorf("expected 1 compiled pattern, got %d", len(m.patterns))
	}
}

// --- Integration: StreamFilter + ToolCallMatcher ---

func TestStreamFilter_ToolCallDetection(t *testing.T) {
	c := &collector{}
	f := NewStreamFilter(c.emit, []Matcher{NewToolCallMatcher()}, 200)

	// Feed tool call JSON token by token
	json := `{"name": "web_search", "arguments": {"query": "golang"}}`
	for _, ch := range json {
		action := f.Write(string(ch))
		if _, ok := action.(ToolCallAction); ok {
			tc := action.(ToolCallAction)
			if tc.Calls[0].Arguments["query"] != "golang" {
				t.Errorf("expected query 'golang', got %v", tc.Calls[0].Arguments["query"])
			}
			if c.all() != "" {
				t.Errorf("expected no emission for tool call JSON, got %q", c.all())
			}
			return
		}
	}

	// If we get here, check flush
	action := f.Flush()
	if tc, ok := action.(ToolCallAction); ok {
		if len(tc.Calls) != 1 || tc.Calls[0].Name != "web_search" {
			t.Errorf("unexpected tool call: %+v", tc.Calls)
		}
		if c.all() != "" {
			t.Errorf("expected no emission for tool call JSON, got %q", c.all())
		}
	} else {
		t.Fatalf("expected ToolCallAction, got %T", action)
	}
}

func TestStreamFilter_NormalTextThenToolCall(t *testing.T) {
	c := &collector{}
	f := NewStreamFilter(c.emit, []Matcher{NewToolCallMatcher()}, 20)

	// Write enough normal text to overflow the buffer and force emission
	f.Write("Let me search for that information now. ")

	if c.all() == "" {
		t.Fatal("expected normal text to be emitted before tool call")
	}

	// Now write tool call JSON
	toolJSON := `{"name": "web_search", "arguments": {"query": "test"}}`
	var gotToolCall bool
	for _, ch := range toolJSON {
		action := f.Write(string(ch))
		if _, ok := action.(ToolCallAction); ok {
			gotToolCall = true
			break
		}
	}
	if !gotToolCall {
		action := f.Flush()
		if _, ok := action.(ToolCallAction); ok {
			gotToolCall = true
		}
	}

	if !gotToolCall {
		t.Fatal("expected tool call to be detected")
	}

	emitted := c.all()
	if !strings.Contains(emitted, "Let me search") {
		t.Errorf("expected normal text in emitted output, got %q", emitted)
	}
	if strings.Contains(emitted, `"name"`) {
		t.Error("tool call JSON should not appear in emitted text")
	}
}

func TestStreamFilter_ContentSafetyKill(t *testing.T) {
	patterns := []ContentSafetyPattern{
		{Name: "blocked", Regex: `forbidden_phrase`},
	}
	safety := NewContentSafetyMatcher(patterns)
	c := &collector{}
	f := NewStreamFilter(c.emit, []Matcher{safety}, 200)
	_ = f // safety now receives prevTail via Scan arg

	action := f.Write("This contains a forbidden_phrase that should be blocked")

	if _, ok := action.(KillAction); !ok {
		t.Fatalf("expected KillAction, got %T", action)
	}
	if c.all() != "" {
		t.Errorf("expected no emission on kill, got %q", c.all())
	}
}

func TestStreamFilter_FalsePositiveBrace(t *testing.T) {
	c := &collector{}
	f := NewStreamFilter(c.emit, []Matcher{NewToolCallMatcher()}, 30)

	// Write text with a brace that's not a tool call
	f.Write("I found {some results} in the data")
	f.Flush()

	emitted := c.all()
	if !strings.Contains(emitted, "{some results}") {
		t.Errorf("expected brace text to pass through, got %q", emitted)
	}
}
