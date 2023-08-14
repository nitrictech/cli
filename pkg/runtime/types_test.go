package runtime

import (
	"testing"
)

func TestNormalizeExtension(t *testing.T) {
	handler := "test.ts"

	result := normalizeFileName(handler)

	if result != "test" {
		t.Error("expected ", result, "to equal test")
	}
}

func TestNormalizeWithDots(t *testing.T) {
	handler := "test.api.ts"

	result := normalizeFileName(handler)

	if result != "test-api" {
		t.Error("expected ", result, "to equal test-api")
	}
}
