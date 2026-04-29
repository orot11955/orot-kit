package secret

import (
	"fmt"
	"regexp"
	"strings"
)

var envKeyPattern = regexp.MustCompile(`^[A-Z_][A-Z0-9_]*$`)

func JWT(byteLength int) (string, error) {
	return JWTWithFormat(byteLength, "hex", "JWT_SECRET")
}

func APIKey(prefix string, length int) (string, error) {
	token, err := Token(length)
	if err != nil {
		return "", err
	}
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return token, nil
	}
	return prefix + "_" + token, nil
}

func JWTWithFormat(byteLength int, format string, key string) (string, error) {
	switch format {
	case "hex":
		return Hex(byteLength)
	case "base64":
		return Base64(byteLength)
	case "env":
		value, err := Hex(byteLength)
		if err != nil {
			return "", err
		}
		return EnvLine(key, value)
	default:
		return "", fmt.Errorf("unsupported jwt format: %s", format)
	}
}

func EnvLine(key string, value string) (string, error) {
	normalized := NormalizeEnvKey(key)
	if normalized == "" {
		return "", fmt.Errorf("env key is required")
	}
	if !envKeyPattern.MatchString(normalized) {
		return "", fmt.Errorf("invalid env key: %s", key)
	}
	return normalized + "=" + value, nil
}

func NormalizeEnvKey(key string) string {
	key = strings.TrimSpace(key)
	key = strings.ReplaceAll(key, "-", "_")
	key = strings.ReplaceAll(key, " ", "_")
	return strings.ToUpper(key)
}
