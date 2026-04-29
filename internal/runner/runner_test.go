package runner

import "testing"

func TestQuoteLeavesSafeValuesAlone(t *testing.T) {
	got := Quote("abc-_.:/@%+=,123")
	if got != "abc-_.:/@%+=,123" {
		t.Fatalf("Quote() = %q", got)
	}
}

func TestQuoteEscapesShellValues(t *testing.T) {
	got := Quote("hello world")
	if got != "'hello world'" {
		t.Fatalf("Quote() = %q", got)
	}
}
