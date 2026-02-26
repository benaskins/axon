package axon_test

import (
	"testing"

	"github.com/benaskins/axon"
)

func TestValidSlug(t *testing.T) {
	valid := []string{"chat", "deploy-gate", "my-agent-v2", "a", "a1", "abc-123"}
	for _, s := range valid {
		if !axon.ValidSlug.MatchString(s) {
			t.Errorf("expected %q to be valid", s)
		}
	}

	invalid := []string{"", "Chat", "deploy_gate", "-leading", "trailing-", "double--hyphen", "UPPER", "has space"}
	for _, s := range invalid {
		if axon.ValidSlug.MatchString(s) {
			t.Errorf("expected %q to be invalid", s)
		}
	}
}

func TestValidateSlug(t *testing.T) {
	if err := axon.ValidateSlug("my-agent"); err != nil {
		t.Errorf("expected valid slug, got error: %v", err)
	}

	if err := axon.ValidateSlug("Invalid"); err == nil {
		t.Error("expected error for invalid slug")
	}
}
