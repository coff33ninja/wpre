package fileengine

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type DedupEntry struct {
	SHA256   string `json:"sha256"`
	Size     int64  `json:"size"`
	Paths    []string `json:"paths"`
	KeptPath string `json:"kept_path,omitempty"`
}

type DedupIndex struct {
	Entries map[string]*DedupEntry
}

func NewDedupIndex() *DedupIndex {
	return &DedupIndex{
		Entries: make(map[string]*DedupEntry),
	}
}

func (di *DedupIndex) IndexFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return err
	}

	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return err
	}

	hash := hex.EncodeToString(hasher.Sum(nil))

	if entry, ok := di.Entries[hash]; ok {
		entry.Paths = append(entry.Paths, path)
	} else {
		di.Entries[hash] = &DedupEntry{
			SHA256: hash,
			Size:   info.Size(),
			Paths:  []string{path},
		}
	}
	return nil
}

func (di *DedupIndex) IndexTree(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		return di.IndexFile(path)
	})
}

func (di *DedupIndex) Duplicates() []*DedupEntry {
	var dups []*DedupEntry
	for _, entry := range di.Entries {
		if len(entry.Paths) > 1 {
			dups = append(dups, entry)
		}
	}
	return dups
}

func (di *DedupIndex) TotalUniqueSize() int64 {
	var total int64
	for _, entry := range di.Entries {
		total += entry.Size
	}
	return total
}

func (di *DedupIndex) TotalSize() int64 {
	var total int64
	for _, entry := range di.Entries {
		total += entry.Size * int64(len(entry.Paths))
	}
	return total
}

func (di *DedupIndex) Deduplicate(keepFirst bool) ([]string, error) {
	var removed []string
	for _, entry := range di.Entries {
		if len(entry.Paths) <= 1 {
			continue
		}
		paths := entry.Paths
		if keepFirst {
			entry.KeptPath = paths[0]
			paths = paths[1:]
		} else {
			entry.KeptPath = paths[0]
			paths = paths[1:]
		}
		for _, p := range paths {
			if err := os.Remove(p); err != nil {
				return removed, fmt.Errorf("failed to remove duplicate %s: %w", p, err)
			}
			removed = append(removed, p)
		}
	}
	return removed, nil
}
