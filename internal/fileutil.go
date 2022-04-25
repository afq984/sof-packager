package internal

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// sha256sum returns the sha256 hex digest of a file
func sha256sum(path string) (string, error) {
	r, err := os.Open(path)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// copyFile copies src to dst
func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("os.MkdirAll(%s): %s", filepath.Dir(dst), err)
	}

	w, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("os.Create(%s): %s", dst, err)
	}
	defer w.Close()

	r, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("os.Open(%s): %s", src, err)
	}
	defer r.Close()

	if _, err := io.Copy(w, r); err != nil {
		return err
	}

	return nil
}
