package utils

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func GetRootProjectDir() (string, error) {
	workingDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	current := workingDir
	for {
		goModPath := filepath.Join(current, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return current, nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			return "", errors.New("reached filesystem root without finding go.mod")
		}
		current = parent
	}
}
