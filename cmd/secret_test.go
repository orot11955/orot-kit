package cmd

import "testing"

func TestSecretCommandPreviewJWTEnv(t *testing.T) {
	got := secretCommandPreview("jwt", secretOptions{length: 64, format: "env", envKey: "jwt-secret"})
	want := "kit internal secret jwt --length 64 --format env --key JWT_SECRET"
	if got != want {
		t.Fatalf("preview = %q, want %q", got, want)
	}
}

func TestSecretCommandPreviewPasswordNoSymbols(t *testing.T) {
	got := secretCommandPreview("password", secretOptions{length: 24, symbols: false})
	want := "kit internal secret password --length 24 --symbols=false"
	if got != want {
		t.Fatalf("preview = %q, want %q", got, want)
	}
}

func TestSecretCommandPreviewAPIKey(t *testing.T) {
	got := secretCommandPreview("api-key", secretOptions{length: 32, prefix: "orot"})
	want := "kit internal secret api-key --length 32 --prefix orot"
	if got != want {
		t.Fatalf("preview = %q, want %q", got, want)
	}
}
