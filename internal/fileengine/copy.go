package fileengine

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

type CopyOptions struct {
	SourceDir      string
	DestDir        string
	RetryCount     int
	RetryDelay     time.Duration
	PreserveTimestamps bool
	Overwrite      bool
	ExcludePatterns []string
	OnProgress     func(copied, total int, path string)
}

type CopyResult struct {
	FilesCopied int
	FilesFailed int
	BytesCopied int64
	Errors      []CopyError
}

type CopyError struct {
	Path  string `json:"path"`
	Error string `json:"error"`
}

func DefaultCopyOptions() CopyOptions {
	return CopyOptions{
		RetryCount:        3,
		RetryDelay:        2 * time.Second,
		PreserveTimestamps: true,
		Overwrite:         true,
		ExcludePatterns:   []string{},
	}
}

func CopyFile(src, dst string, opts CopyOptions) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source: %w", err)
	}
	defer sourceFile.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("failed to create destination dir: %w", err)
	}

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination: %w", err)
	}
	defer destFile.Close()

	written, err := io.Copy(destFile, sourceFile)
	if err != nil {
		return fmt.Errorf("failed to copy data: %w", err)
	}

	if opts.PreserveTimestamps {
		srcInfo, err := os.Stat(src)
		if err == nil {
			os.Chtimes(dst, srcInfo.ModTime(), srcInfo.ModTime())
		}
	}

	_ = written
	return nil
}

func CopyWithRetry(src, dst string, opts CopyOptions) error {
	var lastErr error
	for i := 0; i <= opts.RetryCount; i++ {
		if i > 0 {
			time.Sleep(opts.RetryDelay)
		}
		if err := CopyFile(src, dst, opts); err != nil {
			lastErr = err
			continue
		}
		return nil
	}
	return fmt.Errorf("copy failed after %d retries: %w", opts.RetryCount, lastErr)
}

func CopyTree(opts CopyOptions) (*CopyResult, error) {
	result := &CopyResult{}

	err := filepath.Walk(opts.SourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, _ := filepath.Rel(opts.SourceDir, path)
		if relPath == "." {
			return nil
		}

		destPath := filepath.Join(opts.DestDir, relPath)

		if info.IsDir() {
			return os.MkdirAll(destPath, info.Mode())
		}

		if opts.OnProgress != nil {
			opts.OnProgress(result.FilesCopied+result.FilesFailed+1, 0, relPath)
		}

		if err := CopyWithRetry(path, destPath, opts); err != nil {
			result.FilesFailed++
			result.Errors = append(result.Errors, CopyError{
				Path:  relPath,
				Error: err.Error(),
			})
			return nil
		}

		result.FilesCopied++
		return nil
	})

	return result, err
}
