package utils

import (
	"crypto/rand"
	"fmt"
)

const codeCharset = "abcdefghjkmnpqrstuvwxyz23456789"

func GenerateCode(length int) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("length must be greater than zero")
	}

	code := make([]byte, length)
	random := make([]byte, length)

	for i := 0; i < length; {
		if _, err := rand.Read(random); err != nil {
			return "", fmt.Errorf("generate secure random bytes: %w", err)
		}

		for _, b := range random {
			idx := int(b)
			if idx >= 256-(256%len(codeCharset)) {
				continue
			}

			code[i] = codeCharset[idx%len(codeCharset)]
			i++
			if i == length {
				break
			}
		}
	}

	return string(code), nil
}
