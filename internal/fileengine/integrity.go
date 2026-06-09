package fileengine

import (
	"fmt"
	"os"
	"path/filepath"
)

type IntegrityReport struct {
	SourceRoot string `json:"source_root"`
	DestRoot   string `json:"dest_root"`
	TotalFiles int    `json:"total_files"`
	Verified   int    `json:"verified"`
	Failed     int    `json:"failed"`
	Missing    int    `json:"missing"`
	SizeMatch  bool   `json:"size_match"`
	Errors     []IntegrityError `json:"errors,omitempty"`
}

type IntegrityError struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

func VerifyCopyIntegrity(sourceRoot, destRoot string) (*IntegrityReport, error) {
	report := &IntegrityReport{
		SourceRoot: sourceRoot,
		DestRoot:   destRoot,
	}

	err := filepath.Walk(sourceRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		report.TotalFiles++

		relPath, _ := filepath.Rel(sourceRoot, path)
		destPath := filepath.Join(destRoot, relPath)

		destInfo, err := os.Stat(destPath)
		if os.IsNotExist(err) {
			report.Missing++
			report.Errors = append(report.Errors, IntegrityError{
				Path:    relPath,
				Message: "destination file missing",
			})
			return nil
		}

		if info.Size() != destInfo.Size() {
			report.Failed++
			report.Errors = append(report.Errors, IntegrityError{
				Path:    relPath,
				Message: fmt.Sprintf("size mismatch: source=%d, dest=%d", info.Size(), destInfo.Size()),
			})
			return nil
		}

		report.Verified++
		return nil
	})

	report.SizeMatch = report.Failed == 0 && report.Missing == 0

	return report, err
}

func VerifyFileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func VerifyFileSize(path string, expectedSize int64) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.Size() == expectedSize
}
