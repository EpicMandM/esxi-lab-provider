package service

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

const (
	defaultPasswordLength = 16
	passwordChars         = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
)

// GeneratePassword generates a random password of specified length
func GeneratePassword(length int) (string, error) {
	if length <= 0 {
		length = defaultPasswordLength
	}

	password := make([]byte, length)
	charsLength := big.NewInt(int64(len(passwordChars)))

	for i := range password {
		randomIndex, err := rand.Int(rand.Reader, charsLength)
		if err != nil {
			return "", fmt.Errorf("failed to generate random number: %w", err)
		}
		password[i] = passwordChars[randomIndex.Int64()]
	}

	return string(password), nil
}
