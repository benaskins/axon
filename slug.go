package axon

import (
	"fmt"
	"regexp"
)

// ValidSlug matches lowercase alphanumeric slugs with hyphens between words.
// Examples: "my-agent", "chat", "deploy-gate-v2"
var ValidSlug = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// ValidateSlug returns an error if s is not a valid slug.
func ValidateSlug(s string) error {
	if !ValidSlug.MatchString(s) {
		return fmt.Errorf("invalid slug %q: must be lowercase alphanumeric with hyphens between words", s)
	}
	return nil
}
