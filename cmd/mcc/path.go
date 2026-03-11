package main

import (
	"os"
	"path/filepath"
)

func executableDir() (string, error) {
	if cwd, err := os.Getwd(); err == nil {
		return cwd, nil
	}

	exePath, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Dir(exePath), nil
}
