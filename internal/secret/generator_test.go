package secret

import (
	"strings"
	"testing"
)

func TestPasswordWithOptionsIncludesRequiredGroups(t *testing.T) {
	value, err := PasswordWithOptions(32, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(value) != 32 {
		t.Fatalf("password length = %d", len(value))
	}
	if !containsAny(value, Lowercase) || !containsAny(value, Uppercase) || !containsAny(value, Digits) || !containsAny(value, Symbols) {
		t.Fatalf("password missing required character group: %q", value)
	}
}

func TestPasswordWithOptionsWithoutSymbols(t *testing.T) {
	value, err := PasswordWithOptions(32, false)
	if err != nil {
		t.Fatal(err)
	}
	if containsAny(value, Symbols) {
		t.Fatalf("password contains symbols: %q", value)
	}
}

func TestJWTWithEnvFormat(t *testing.T) {
	value, err := JWTWithFormat(8, "env", "jwt-secret")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(value, "JWT_SECRET=") {
		t.Fatalf("jwt env value = %q", value)
	}
}

func TestAPIKeyPrefix(t *testing.T) {
	value, err := APIKey("orot", 16)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(value, "orot_") {
		t.Fatalf("api key = %q", value)
	}
	if len(strings.TrimPrefix(value, "orot_")) != 16 {
		t.Fatalf("api key length = %d", len(strings.TrimPrefix(value, "orot_")))
	}
}

func TestEnvLineRejectsInvalidKey(t *testing.T) {
	_, err := EnvLine("1bad", "value")
	if err == nil {
		t.Fatal("expected invalid key error")
	}
}

func containsAny(value string, alphabet string) bool {
	for _, char := range value {
		if strings.ContainsRune(alphabet, char) {
			return true
		}
	}
	return false
}
