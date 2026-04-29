package secret

import "fmt"

func Password(length int) (string, error) {
	return PasswordWithOptions(length, true)
}

func PasswordWithOptions(length int, symbols bool) (string, error) {
	required := []string{Lowercase, Uppercase, Digits}
	alphabet := AlphaNumeric
	if symbols {
		required = append(required, Symbols)
		alphabet += Symbols
	}
	if length < len(required) {
		return "", fmt.Errorf("password length must be at least %d", len(required))
	}
	out := make([]byte, 0, length)
	for _, group := range required {
		value, err := RandomString(1, group)
		if err != nil {
			return "", err
		}
		out = append(out, value[0])
	}
	for len(out) < length {
		value, err := RandomString(1, alphabet)
		if err != nil {
			return "", err
		}
		out = append(out, value[0])
	}
	if err := ShuffleBytes(out); err != nil {
		return "", err
	}
	return string(out), nil
}
