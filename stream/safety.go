package stream

import "regexp"

// ContentSafetyPattern is a named regex pattern for content filtering.
type ContentSafetyPattern struct {
	Name     string `json:"name"`
	Regex    string `json:"regex"`
	compiled *regexp.Regexp
}

// ContentSafetyMatcher scans for blocked patterns in the stream.
type ContentSafetyMatcher struct {
	patterns []ContentSafetyPattern
}

// NewContentSafetyMatcher creates a matcher from compiled patterns.
func NewContentSafetyMatcher(patterns []ContentSafetyPattern) *ContentSafetyMatcher {
	compiled := make([]ContentSafetyPattern, 0, len(patterns))
	for _, p := range patterns {
		re, err := regexp.Compile("(?i)" + p.Regex)
		if err != nil {
			continue // skip invalid patterns
		}
		compiled = append(compiled, ContentSafetyPattern{
			Name:     p.Name,
			Regex:    p.Regex,
			compiled: re,
		})
	}
	return &ContentSafetyMatcher{patterns: compiled}
}

func (m *ContentSafetyMatcher) Name() string { return "content_safety" }

func (m *ContentSafetyMatcher) Scan(buf []byte, prevTail string) MatchResult {
	// Prepend overlap from previously emitted text for boundary matching
	var scanTarget []byte
	if prevTail != "" {
		scanTarget = append([]byte(prevTail), buf...)
	} else {
		scanTarget = buf
	}

	for _, p := range m.patterns {
		if p.compiled.Match(scanTarget) {
			return FullMatch
		}
	}
	return NoMatch
}
