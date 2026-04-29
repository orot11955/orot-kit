package secret

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/big"
)

const (
	Lowercase = "abcdefghijklmnopqrstuvwxyz"
	Uppercase = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	Digits    = "0123456789"
	Symbols   = "!@#$%^&*()-_=+[]{}:,.?/"

	AlphaNumeric = Lowercase + Uppercase + Digits
)

func RandomBytes(length int) ([]byte, error) {
	if length <= 0 {
		return nil, fmt.Errorf("length must be greater than 0")
	}
	value := make([]byte, length)
	_, err := rand.Read(value)
	return value, err
}

func Hex(byteLength int) (string, error) {
	value, err := RandomBytes(byteLength)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(value), nil
}

func Base64(byteLength int) (string, error) {
	value, err := RandomBytes(byteLength)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func Token(length int) (string, error) {
	return RandomString(length, AlphaNumeric)
}

func RandomString(length int, alphabet string) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("length must be greater than 0")
	}
	if alphabet == "" {
		return "", fmt.Errorf("alphabet must not be empty")
	}
	out := make([]byte, length)
	max := big.NewInt(int64(len(alphabet)))
	for index := range out {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		out[index] = alphabet[n.Int64()]
	}
	return string(out), nil
}

func ShuffleBytes(values []byte) error {
	for index := len(values) - 1; index > 0; index-- {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(index+1)))
		if err != nil {
			return err
		}
		swap := int(n.Int64())
		values[index], values[swap] = values[swap], values[index]
	}
	return nil
}
