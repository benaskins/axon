package stream

import "bytes"

// --- Filter actions ---

// FilterAction is returned by StreamFilter.Write and Flush to signal what happened.
type FilterAction interface{ filterAction() }

type ContinueAction struct{}
type ToolCallAction struct{ Calls []ToolCall }
type KillAction struct{ Reason string }

func (ContinueAction) filterAction() {}
func (ToolCallAction) filterAction() {}
func (KillAction) filterAction()     {}

// ToolCall is a provider-agnostic tool call representation.
type ToolCall struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

// --- Matcher interface ---

type MatchResult int

const (
	NoMatch      MatchResult = iota
	PartialMatch             // keep buffering, pattern might be forming
	FullMatch                // pattern confirmed, act on it
)

// Matcher scans a buffer and reports whether a pattern is present.
// prevTail contains the last N chars of previously emitted text for cross-boundary matching.
type Matcher interface {
	Scan(buf []byte, prevTail string) MatchResult
	Name() string
}

// Extractable is implemented by matchers that produce data on FullMatch.
type Extractable interface {
	Extract(buf []byte) FilterAction
}

// --- StreamFilter ---

const (
	DefaultMaxBuffer = 200
	defaultOverlap   = 20
)

// StreamFilter sits between the model output stream and the SSE sender.
// It maintains a small lookahead buffer, runs matchers against it, and
// emits tokens from the trailing edge with a fixed delay.
type StreamFilter struct {
	buf       bytes.Buffer
	emitFunc  func(string)
	matchers  []Matcher
	maxBuffer int
	overlap   int
	prevTail  string // last N chars of previously emitted text for boundary matching
}

// NewStreamFilter creates a filter with the given emit function, matchers, and buffer size.
func NewStreamFilter(emitFunc func(string), matchers []Matcher, maxBuffer int) *StreamFilter {
	if maxBuffer <= 0 {
		maxBuffer = DefaultMaxBuffer
	}
	return &StreamFilter{
		emitFunc:  emitFunc,
		matchers:  matchers,
		maxBuffer: maxBuffer,
		overlap:   defaultOverlap,
	}
}

// Write feeds a token into the filter. Returns an action if a matcher triggers.
func (f *StreamFilter) Write(token string) FilterAction {
	f.buf.WriteString(token)
	return f.scan()
}

// Flush drains the remaining buffer. Call after the model stream completes.
func (f *StreamFilter) Flush() FilterAction {
	if f.buf.Len() == 0 {
		return ContinueAction{}
	}

	// Final scan — a matcher might match the remaining content
	bufBytes := f.buf.Bytes()
	for _, m := range f.matchers {
		result := m.Scan(bufBytes, f.prevTail)
		if result == FullMatch {
			if ext, ok := m.(Extractable); ok {
				action := ext.Extract(bufBytes)
				f.buf.Reset()
				return action
			}
			f.buf.Reset()
			return KillAction{Reason: m.Name()}
		}
		// PartialMatch on flush → treat as no match, emit as text
	}

	// Emit everything remaining
	f.emit(f.buf.String())
	f.buf.Reset()
	return ContinueAction{}
}

// scan runs matchers and emits/holds buffer content accordingly.
func (f *StreamFilter) scan() FilterAction {
	bufBytes := f.buf.Bytes()

	for _, m := range f.matchers {
		result := m.Scan(bufBytes, f.prevTail)
		switch result {
		case FullMatch:
			if ext, ok := m.(Extractable); ok {
				action := ext.Extract(bufBytes)
				f.buf.Reset()
				return action
			}
			// Non-extractable full match = kill
			f.buf.Reset()
			return KillAction{Reason: m.Name()}

		case PartialMatch:
			// Hold everything — don't emit until resolved
			return ContinueAction{}
		}
	}

	// No matches — emit overflow beyond maxBuffer
	if f.buf.Len() > f.maxBuffer {
		emitN := f.buf.Len() - f.maxBuffer
		toEmit := string(bufBytes[:emitN])
		f.emit(toEmit)

		// Compact: keep only the tail
		remaining := make([]byte, f.maxBuffer)
		copy(remaining, bufBytes[emitN:])
		f.buf.Reset()
		f.buf.Write(remaining)
	}

	return ContinueAction{}
}

// emit sends text to the client and saves the tail for overlap matching.
func (f *StreamFilter) emit(s string) {
	if s == "" {
		return
	}
	f.emitFunc(s)

	// Keep overlap for cross-boundary matching
	if len(s) >= f.overlap {
		f.prevTail = s[len(s)-f.overlap:]
	} else {
		// Combine with existing tail, trim to overlap size
		combined := f.prevTail + s
		if len(combined) > f.overlap {
			f.prevTail = combined[len(combined)-f.overlap:]
		} else {
			f.prevTail = combined
		}
	}
}

// PrevTail returns the overlap text for external inspection.
func (f *StreamFilter) PrevTail() string {
	return f.prevTail
}
