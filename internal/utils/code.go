package utils

import (
	"crypto/rand"
	"math/big"
)

func GenerateCode(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	code := make([]byte, length)
	for i := range code {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		code[i] = charset[n.Int64()]
	}
	return string(code), nil
}
