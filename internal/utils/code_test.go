package utils

import "testing"

func TestGenerateCodeReturnsCodeWithRequestedLength(t *testing.T) {
	testCases := []int{1, 8, 32, 128}

	for _, length := range testCases {
		code, err := GenerateCode(length)
		if err != nil {
			t.Fatalf("GenerateCode(%d) returned error: %v", length, err)
		}

		if len(code) != length {
			t.Fatalf("expected code length %d, got %d", length, len(code))
		}
	}
}

func TestGenerateCodeGeneratesRandomCodes(t *testing.T) {
	const length = 8

	first, err := GenerateCode(length)
	if err != nil {
		t.Fatalf("GenerateCode returned error: %v", err)
	}

	second, err := GenerateCode(length)
	if err != nil {
		t.Fatalf("GenerateCode returned error: %v", err)
	}

	if first == second {
		t.Fatalf("expected different codes, got %q and %q", first, second)
	}
}
