package cmd

import "testing"

func TestNormalizeFindPatternWrapsPlainText(t *testing.T) {
	got := normalizeFindPattern("nginx")
	if got != "*nginx*" {
		t.Fatalf("normalizeFindPattern() = %q", got)
	}
}

func TestNormalizeFindPatternKeepsGlob(t *testing.T) {
	got := normalizeFindPattern("*.go")
	if got != "*.go" {
		t.Fatalf("normalizeFindPattern() = %q", got)
	}
}
