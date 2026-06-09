package resources

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed modules/ps
//go:embed modules/py
var Modules embed.FS

const DefaultConfigYAML = `version: "1.0"

pipeline:
  stages:
    - profile_scan
    - temp_profile_create
    - data_harvest
    - onedrive_extract
    - app_capture
    - new_profile_create
    - data_inject
    - final_validation
  max_retries_per_stage: 3
  timeout_minutes: 30

onedrive:
  enabled: true
  pause_before_harvest: true
  wait_for_drain_seconds: 120
  force_hydration: true
  strip_metadata: true
  detach_after_harvest: true

browsers:
  chrome: true
  edge: true
  firefox: true
  include_history: false
  include_cookies: true
  include_passwords: true
  include_sessions: true

outlook:
  enabled: true
  detect_pst: true
  copy_pst: true
  export_profile_reg: true
  backup_autocomplete: true
  generate_setup_guide: true
  mailpv_auto_download: true
  mailpv_path: ""

data:
  vault_root: "C:\\MigrationVault"
  safe_folders:
    - Desktop
    - Documents
    - Downloads
    - Pictures
    - Videos
    - Music
  exclude_patterns:
    - "*.tmp"
    - "*.log"
    - "Thumbs.db"

logging:
  level: info

safety:
  dry_run: false
  rollback_enabled: true
  integrity_check: true
  require_admin: true
`

func GetDefaultConfig() []byte {
	return []byte(DefaultConfigYAML)
}

func ExtractModules(targetDir string) error {
	catDirs := []string{"modules/ps", "modules/py"}
	for _, cat := range catDirs {
		err := fs.WalkDir(Modules, cat, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			relPath, _ := filepath.Rel("modules", path)
			targetPath := filepath.Join(targetDir, relPath)
			if d.IsDir() {
				return os.MkdirAll(targetPath, 0755)
			}
			data, err := Modules.ReadFile(path)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return err
			}
			return os.WriteFile(targetPath, data, 0755)
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func ModuleBasePath() (string, error) {
	tmpDir := filepath.Join(os.TempDir(), "wpre-modules")
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return "", err
	}
	if err := ExtractModules(tmpDir); err != nil {
		return "", err
	}
	return tmpDir, nil
}
