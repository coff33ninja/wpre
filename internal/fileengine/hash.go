package fileengine

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type FileHash struct {
	Path      string `json:"path"`
	SHA256    string `json:"sha256"`
	SizeBytes int64  `json:"size_bytes"`
}

func HashFile(path string) (*FileHash, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return nil, fmt.Errorf("failed to hash file: %w", err)
	}

	return &FileHash{
		Path:      path,
		SHA256:    hex.EncodeToString(hasher.Sum(nil)),
		SizeBytes: info.Size(),
	}, nil
}

func HashTree(root string) ([]FileHash, error) {
	var results []FileHash
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		hash, err := HashFile(path)
		if err != nil {
			return fmt.Errorf("failed to hash %s: %w", path, err)
		}
		relPath, _ := filepath.Rel(root, path)
		hash.Path = relPath
		results = append(results, *hash)
		return nil
	})
	return results, err
}

func VerifyHash(path, expectedSHA256 string) (bool, error) {
	hash, err := HashFile(path)
	if err != nil {
		return false, err
	}
	return hash.SHA256 == expectedSHA256, nil
}
