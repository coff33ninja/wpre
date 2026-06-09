package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Version  string        `yaml:"version"`
	Pipeline PipelineConfig `yaml:"pipeline"`
	OneDrive OneDriveConfig `yaml:"onedrive"`
	Browsers BrowserConfig  `yaml:"browsers"`
	Outlook  OutlookConfig  `yaml:"outlook"`
	Data     DataConfig     `yaml:"data"`
	Logging  LoggingConfig  `yaml:"logging"`
	Safety   SafetyConfig   `yaml:"safety"`
}

type PipelineConfig struct {
	Stages              []string `yaml:"stages"`
	RebootBetweenStages bool     `yaml:"reboot_between_stages"`
	MaxRetriesPerStage  int      `yaml:"max_retries_per_stage"`
	TimeoutMinutes      int      `yaml:"timeout_minutes"`
}

type OneDriveConfig struct {
	Enabled               bool   `yaml:"enabled"`
	PauseBeforeHarvest    bool   `yaml:"pause_before_harvest"`
	WaitForDrainSeconds   int    `yaml:"wait_for_drain_seconds"`
	ForceHydration        bool   `yaml:"force_hydration"`
	StripMetadata         bool   `yaml:"strip_metadata"`
	DetachAfterHarvest    bool   `yaml:"detach_after_harvest"`
}

type BrowserConfig struct {
	Chrome            bool `yaml:"chrome"`
	Edge              bool `yaml:"edge"`
	Firefox           bool `yaml:"firefox"`
	IncludeHistory    bool `yaml:"include_history"`
	IncludeExtensions bool `yaml:"include_extensions"`
	IncludeCookies    bool `yaml:"include_cookies"`
	IncludePasswords  bool `yaml:"include_passwords"`
	IncludeSessions   bool `yaml:"include_sessions"`
}

type OutlookConfig struct {
	Enabled            bool   `yaml:"enabled"`
	DetectPST          bool   `yaml:"detect_pst"`
	DetectOST          bool   `yaml:"detect_ost"`
	CopyPST            bool   `yaml:"copy_pst"`
	ExportProfileReg   bool   `yaml:"export_profile_reg"`
	VerifyWithScanpst  bool   `yaml:"verify_with_scanpst"`
	BackupAutocomplete bool   `yaml:"backup_autocomplete"`
	GenerateSetupGuide bool   `yaml:"generate_setup_guide"`
	MailPVPath         string `yaml:"mailpv_path"`
}

type DataConfig struct {
	VaultRoot           string   `yaml:"vault_root"`
	IncludeAppDataRoaming bool   `yaml:"include_appdata_roaming"`
	IncludeAppDataLocal bool     `yaml:"include_appdata_local"`
	SafeFolders         []string `yaml:"safe_folders"`
	ExcludePatterns     []string `yaml:"exclude_patterns"`
}

type LoggingConfig struct {
	Level        string `yaml:"level"`
	AuditFile    string `yaml:"audit_file"`
	ReportFormat string `yaml:"report_format"`
}

type SafetyConfig struct {
	DryRun          bool `yaml:"dry_run"`
	SafeMode        bool `yaml:"safe_mode"`
	RollbackEnabled bool `yaml:"rollback_enabled"`
	IntegrityCheck  bool `yaml:"integrity_check"`
	RequireAdmin    bool `yaml:"require_admin"`
}

func Default() *Config {
	return &Config{
		Version: "1.0",
		Pipeline: PipelineConfig{
			Stages: []string{
				"profile_scan", "temp_profile_create", "data_harvest",
				"onedrive_extract", "app_capture", "new_profile_create",
				"data_inject", "profile_cleanup", "final_validation",
			},
			RebootBetweenStages: true,
			MaxRetriesPerStage:  3,
			TimeoutMinutes:      30,
		},
		OneDrive: OneDriveConfig{
			Enabled:             true,
			PauseBeforeHarvest:  true,
			WaitForDrainSeconds: 120,
			ForceHydration:      true,
			StripMetadata:       true,
			DetachAfterHarvest:  true,
		},
		Browsers: BrowserConfig{
			Chrome:            true,
			Edge:              true,
			Firefox:           true,
			IncludeHistory:    false,
			IncludeExtensions: true,
			IncludeCookies:    true,
			IncludePasswords:  true,
			IncludeSessions:   true,
		},
		Outlook: OutlookConfig{
			Enabled:            true,
			DetectPST:          true,
			DetectOST:          true,
			CopyPST:            true,
			ExportProfileReg:   true,
			VerifyWithScanpst:  false,
			BackupAutocomplete: true,
			GenerateSetupGuide: true,
			MailPVPath:         "",
		},
		Data: DataConfig{
			VaultRoot:            "C:\\MigrationVault",
			IncludeAppDataRoaming: false,
			IncludeAppDataLocal:  false,
			SafeFolders:          []string{"Desktop", "Documents", "Downloads", "Pictures", "Videos", "Music"},
			ExcludePatterns:      []string{"*.tmp", "*.log", "Thumbs.db", "$RECYCLE.BIN", "System Volume Information"},
		},
		Logging: LoggingConfig{
			Level:        "info",
			AuditFile:    "wpre-audit.log",
			ReportFormat: "json",
		},
		Safety: SafetyConfig{
			DryRun:          false,
			SafeMode:        false,
			RollbackEnabled: true,
			IntegrityCheck:  true,
			RequireAdmin:    true,
		},
	}
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	cfg := Default()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return cfg, nil
}

func (c *Config) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
