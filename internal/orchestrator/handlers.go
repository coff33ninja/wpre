package orchestrator

import (
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"wpre/internal/config"
	"wpre/internal/fileengine"
	"wpre/internal/pipeline"
)

func handleBootstrap(ctx *pipeline.Context) error {
	ctx.Logger.Info("[bootstrap] checking admin privileges")
	if !isAdmin() {
		return fmt.Errorf("admin privileges required — run as administrator")
	}

	cfg := ctx.Config.(*config.Config)
	ctx.Logger.Info("[bootstrap] config loaded: vault=%s", cfg.Data.VaultRoot)

	ctx.Data.DryRun = cfg.Safety.DryRun
	ctx.Data.VaultRoot = cfg.Data.VaultRoot
	ctx.Data.SafeFolders = cfg.Data.SafeFolders
	ctx.Data.ExcludePatterns = cfg.Data.ExcludePatterns

	if !ctx.Data.DryRun {
		vaultPaths := []string{
			cfg.Data.VaultRoot,
			filepath.Join(cfg.Data.VaultRoot, "UserData"),
			filepath.Join(cfg.Data.VaultRoot, "Cloud"),
			filepath.Join(cfg.Data.VaultRoot, "AppData", "Roaming"),
			filepath.Join(cfg.Data.VaultRoot, "AppData", "Local"),
			filepath.Join(cfg.Data.VaultRoot, "Browsers"),
			filepath.Join(cfg.Data.VaultRoot, "Outlook"),
			filepath.Join(cfg.Data.VaultRoot, "Registry"),
		}
		for _, p := range vaultPaths {
			if err := os.MkdirAll(p, 0755); err != nil {
				return fmt.Errorf("failed to create vault path %s: %w", p, err)
			}
		}
	}

	if !ctx.Data.DryRun {
		ctx.Logger.Info("[bootstrap] vault directories created")
	}
	return nil
}

func handleProfileScan(ctx *pipeline.Context) error {
	ctx.Logger.Info("[profile_scan] running profile scanner")

	tmpFile := filepath.Join(os.TempDir(), "wpre_scan_result.json")
	if ctx.RunPS != nil {
		if _, err := ctx.RunPS("ps/profile_scanner.ps1", "-OutputPath", tmpFile); err != nil {
			return fmt.Errorf("profile scanner failed: %w", err)
		}
	} else {
		return fmt.Errorf("PS executor not available")
	}

	data, err := os.ReadFile(tmpFile)
	if err != nil {
		return fmt.Errorf("failed to read scan result: %w", err)
	}
	os.Remove(tmpFile)

	var scan pipeline.ProfileScanResult
	if err := json.Unmarshal(data, &scan); err != nil {
		return fmt.Errorf("failed to parse scan result: %w", err)
	}

	ctx.Data.ScanResults = &scan
	ctx.Logger.Info("[profile_scan] found %d profiles, OneDrive running=%v",
		len(scan.Profiles), scan.OneDrive.ProcessRunning)

	currentUser, _ := user.Current()
	for _, p := range scan.Profiles {
		ctx.Logger.Info("[profile_scan]   profile: sid=%s path=%s exists=%v", p.SID, p.Path, p.Exists)
	}
	_ = currentUser
	return nil
}

func handleBackupExisting(ctx *pipeline.Context) error {
	ctx.Logger.Info("[backup_existing] backing up existing profile data (stub)")
	return nil
}

func handleTempProfileCreate(ctx *pipeline.Context) error {
	ctx.Logger.Info("[temp_profile_create] creating temporary admin profile")
	if ctx.Data.DryRun {
		ctx.Logger.Info("[temp_profile_create] dry-run — skipping")
		return nil
	}

	password := generatePassword(16)
	ctx.Data.TempUsername = "WPRE_TempAdmin"
	ctx.Data.TempPassword = password

	tmpFile := filepath.Join(os.TempDir(), "wpre_temp_profile.json")
	if ctx.RunPS != nil {
		if _, err := ctx.RunPS("ps/temp_profile.ps1",
			"-Action", "create",
			"-Username", ctx.Data.TempUsername,
			"-Password", password,
			"-OutputPath", tmpFile,
		); err != nil {
			return fmt.Errorf("temp profile creation failed: %w", err)
		}
	} else {
		return fmt.Errorf("PS executor not available")
	}

	data, err := os.ReadFile(tmpFile)
	if err != nil {
		return fmt.Errorf("failed to read temp profile result: %w", err)
	}
	os.Remove(tmpFile)

	var result pipeline.TempProfileResult
	if err := json.Unmarshal(data, &result); err != nil {
		return fmt.Errorf("failed to parse temp profile result: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("temp profile creation failed: %s", result.Error)
	}

	ctx.Logger.Info("[temp_profile_create] created user: %s", ctx.Data.TempUsername)
	return nil
}

func handleDataHarvest(ctx *pipeline.Context) error {
	ctx.Logger.Info("[data_harvest] harvesting user data")
	if ctx.Data.DryRun {
		ctx.Logger.Info("[data_harvest] dry-run — skipping")
		return nil
	}

	if ctx.Data.ScanResults == nil {
		return fmt.Errorf("no scan results available — run profile_scan first")
	}

	cfg := ctx.Config.(*config.Config)
	sourceProfile := ""
	for _, p := range ctx.Data.ScanResults.Profiles {
		if !p.Exists {
			continue
		}
		if isSystemSID(p.SID) {
			continue
		}
		sourceProfile = p.Path
		break
	}
	if sourceProfile == "" {
		return fmt.Errorf("no existing user profile found to harvest from")
	}
	ctx.Logger.Info("[data_harvest] source profile: %s", sourceProfile)

	vaultUserData := filepath.Join(ctx.Data.VaultRoot, "UserData")
	for _, folder := range ctx.Data.SafeFolders {
		src := filepath.Join(sourceProfile, folder)
		if _, err := os.Stat(src); os.IsNotExist(err) {
			ctx.Logger.Warn("[data_harvest] folder not found: %s", src)
			continue
		}
		dst := filepath.Join(vaultUserData, folder)
		ctx.Logger.Info("[data_harvest] copying %s -> %s", src, dst)

		opts := fileengine.DefaultCopyOptions()
		opts.ExcludePatterns = cfg.Data.ExcludePatterns
		result, err := fileengine.CopyTree(fileengine.CopyOptions{
			SourceDir:      src,
			DestDir:        dst,
			RetryCount:     opts.RetryCount,
			RetryDelay:     opts.RetryDelay,
			PreserveTimestamps: true,
			Overwrite:      true,
			ExcludePatterns: opts.ExcludePatterns,
		})
		if err != nil {
			ctx.Logger.Error("[data_harvest] copy error for %s: %v", folder, err)
			continue
		}
		ctx.Logger.Info("[data_harvest]   %s: %d files copied, %d failed", folder, result.FilesCopied, result.FilesFailed)
	}

	ctx.Logger.Info("[data_harvest] user data harvest complete")
	return nil
}

func handleOneDriveExtract(ctx *pipeline.Context) error {
	ctx.Logger.Info("[onedrive_extract] extracting OneDrive data")
	if ctx.Data.DryRun {
		ctx.Logger.Info("[onedrive_extract] dry-run — skipping")
		return nil
	}

	cfg := ctx.Config.(*config.Config)
	if !cfg.OneDrive.Enabled {
		ctx.Logger.Info("[onedrive_extract] OneDrive harvest disabled in config — skipping")
		return nil
	}

	if ctx.Data.ScanResults == nil {
		return fmt.Errorf("no scan results available")
	}
	if !ctx.Data.ScanResults.OneDrive.ProcessRunning {
		ctx.Logger.Info("[onedrive_extract] OneDrive not running — skipping")
		return nil
	}

	ctx.Logger.Info("[onedrive_extract] step 1: pause sync")
	if ctx.RunPS != nil {
		if _, err := ctx.RunPS("ps/onedrive_state.ps1", "-Action", "pause"); err != nil {
			ctx.Logger.Warn("[onedrive_extract] pause failed (non-fatal): %v", err)
		}
	}

	ctx.Logger.Info("[onedrive_extract] step 2: detect placeholder files")
	placeholderFile := filepath.Join(os.TempDir(), "wpre_placeholders.json")
	if ctx.RunPS != nil {
		for _, folder := range ctx.Data.ScanResults.OneDrive.FolderPaths {
			pResult, err := ctx.RunPS("ps/onedrive_state.ps1",
				"-Action", "placeholders",
				"-OutputPath", placeholderFile,
				"--", folder,
			)
			if err != nil {
				ctx.Logger.Warn("[onedrive_extract] placeholder check failed for %s: %v", folder, err)
				continue
			}
			_ = pResult

			data, _ := os.ReadFile(placeholderFile)
			os.Remove(placeholderFile)

			var phResult pipeline.OneDrivePlaceholderResult
			if json.Unmarshal(data, &phResult) == nil && phResult.PlaceholderCount > 0 {
				ctx.Logger.Warn("[onedrive_extract] %d placeholders in %s — may need hydration", phResult.PlaceholderCount, folder)
			}
		}
	}

	ctx.Logger.Info("[onedrive_extract] step 3: harvest OneDrive folders to vault")
	for _, folder := range ctx.Data.ScanResults.OneDrive.FolderPaths {
		if _, err := os.Stat(folder); os.IsNotExist(err) {
			continue
		}
		dest := filepath.Join(ctx.Data.VaultRoot, "Cloud", filepath.Base(folder))
		ctx.Logger.Info("[onedrive_extract]   copying %s -> %s", folder, dest)
		opts := fileengine.DefaultCopyOptions()
		result, err := fileengine.CopyTree(fileengine.CopyOptions{
			SourceDir:      folder,
			DestDir:        dest,
			RetryCount:     3,
			RetryDelay:     opts.RetryDelay,
			PreserveTimestamps: true,
			Overwrite:      true,
			ExcludePatterns: []string{"*.sync.db", "*.sync.id", "desktop.ini", "*.odl", "Personal Delta*"},
		})
		if err != nil {
			ctx.Logger.Error("[onedrive_extract] copy error: %v", err)
			continue
		}
		ctx.Logger.Info("[onedrive_extract]   copied %d files, %d errors", result.FilesCopied, result.FilesFailed)
	}

	ctx.Logger.Info("[onedrive_extract] step 4: verify integrity")
	for _, folder := range ctx.Data.ScanResults.OneDrive.FolderPaths {
		dest := filepath.Join(ctx.Data.VaultRoot, "Cloud", filepath.Base(folder))
		report, err := fileengine.VerifyCopyIntegrity(folder, dest)
		if err != nil {
			ctx.Logger.Warn("[onedrive_extract] integrity check failed: %v", err)
			continue
		}
		ctx.Logger.Info("[onedrive_extract]   integrity: %d/%d verified, %d missing, %d size mismatch",
			report.Verified, report.TotalFiles, report.Missing, report.Failed)
	}

	if cfg.OneDrive.DetachAfterHarvest {
		ctx.Logger.Info("[onedrive_extract] step 5: sign out OneDrive")
		if ctx.RunPS != nil {
			if _, err := ctx.RunPS("ps/onedrive_state.ps1", "-Action", "signout"); err != nil {
				ctx.Logger.Warn("[onedrive_extract] signout failed (non-fatal): %v", err)
			}
		}
	}

	ctx.Logger.Info("[onedrive_extract] OneDrive extraction complete")
	return nil
}

func handleAppCapture(ctx *pipeline.Context) error {
	ctx.Logger.Info("[app_capture] capturing browser and app auth data")
	if ctx.Data.DryRun {
		ctx.Logger.Info("[app_capture] dry-run — skipping")
		return nil
	}
	if ctx.Data.ScanResults == nil {
		return fmt.Errorf("no scan results — run profile_scan first")
	}

	source := findSourceProfile(ctx)
	if source == "" {
		ctx.Logger.Warn("[app_capture] no source profile found — skipping browser auth")
		return nil
	}

	cfg := ctx.Config.(*config.Config)
	if !cfg.Browsers.IncludeCookies && !cfg.Browsers.IncludePasswords && !cfg.Browsers.IncludeSessions {
		ctx.Logger.Info("[app_capture] browser auth capture disabled in config — skipping")
		return nil
	}
	ctx.Logger.Info("[app_capture] harvesting browser auth from: %s", source)

	if err := harvestBrowserAuth(ctx, source); err != nil {
		ctx.Logger.Warn("[app_capture] harvestBrowserAuth error (non-fatal): %v", err)
		return nil
	}

	ctx.Logger.Info("[app_capture] browser auth capture complete")
	return nil
}

func handleNewProfileCreate(ctx *pipeline.Context) error {
	ctx.Logger.Info("[new_profile_create] creating target user profile")
	if ctx.Data.DryRun {
		ctx.Logger.Info("[new_profile_create] dry-run — skipping")
		return nil
	}
	cfg := ctx.Config.(*config.Config)

	password := generatePassword(20)
	tmpFile := filepath.Join(os.TempDir(), "wpre_new_profile.json")
	if ctx.RunPS != nil {
		if _, err := ctx.RunPS("ps/new_profile.ps1",
			"-Username", "NewUser",
			"-FullName", "Migrated User",
			"-Password", password,
			"-OutputPath", tmpFile,
		); err != nil {
			return fmt.Errorf("new profile creation failed: %w", err)
		}
	} else {
		return fmt.Errorf("PS executor not available")
	}

	data, err := os.ReadFile(tmpFile)
	if err != nil {
		return fmt.Errorf("failed to read new profile result: %w", err)
	}
	os.Remove(tmpFile)

	var result struct {
		Success     bool   `json:"success"`
		Username    string `json:"username"`
		SID         string `json:"sid"`
		ProfilePath string `json:"profilePath"`
		Error       string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return fmt.Errorf("failed to parse new profile result: %w", err)
	}
	if !result.Success {
		return fmt.Errorf("new profile creation failed: %s", result.Error)
	}

	ctx.Data.TargetUsername = result.Username
	ctx.Data.TargetSID = result.SID
	ctx.Logger.Info("[new_profile_create] created user: %s sid=%s", result.Username, result.SID)
	_ = cfg
	return nil
}

func handleDataInject(ctx *pipeline.Context) error {
	ctx.Logger.Info("[data_inject] injecting harvested data into target profile")
	if ctx.Data.DryRun {
		ctx.Logger.Info("[data_inject] dry-run — skipping")
		return nil
	}
	cfg := ctx.Config.(*config.Config)

	targetProfile := filepath.Join("C:\\Users", ctx.Data.TargetUsername)
	if err := os.MkdirAll(targetProfile, 0755); err != nil {
		return fmt.Errorf("failed to create target profile directory: %w", err)
	}
	knownFolders := []string{"Desktop", "Documents", "Downloads", "Pictures", "Videos", "Music"}
	for _, f := range knownFolders {
		os.MkdirAll(filepath.Join(targetProfile, f), 0755)
	}

	injectDirs := []struct{ src, dst string }{
		{filepath.Join(ctx.Data.VaultRoot, "UserData"), targetProfile},
		{filepath.Join(ctx.Data.VaultRoot, "Cloud"), filepath.Join(targetProfile, "OneDrive_Harvest")},
	}
	for _, d := range injectDirs {
		if _, err := os.Stat(d.src); os.IsNotExist(err) {
			ctx.Logger.Warn("[data_inject] source not found: %s", d.src)
			continue
		}
		opts := fileengine.DefaultCopyOptions()
		opts.ExcludePatterns = cfg.Data.ExcludePatterns
		result, err := fileengine.CopyTree(fileengine.CopyOptions{
			SourceDir:      d.src,
			DestDir:        d.dst,
			RetryCount:     3,
			RetryDelay:     opts.RetryDelay,
			PreserveTimestamps: true,
			Overwrite:      true,
			ExcludePatterns: opts.ExcludePatterns,
		})
		if err != nil {
			ctx.Logger.Error("[data_inject] injection error: %v", err)
			continue
		}
		ctx.Logger.Info("[data_inject] injected %s -> %s: %d files", d.src, d.dst, result.FilesCopied)
	}

	browserVault := filepath.Join(ctx.Data.VaultRoot, "Browsers")
	restoreBrowser := cfg.Browsers.IncludeCookies || cfg.Browsers.IncludePasswords || cfg.Browsers.IncludeSessions
	if _, err := os.Stat(browserVault); err == nil && restoreBrowser {
		ctx.Logger.Info("[data_inject] restoring browser auth data to new profile")

		targetLocal := filepath.Join(targetProfile, "AppData", "Local")
		targetRoaming := filepath.Join(targetProfile, "AppData", "Roaming")

		browserRestore := []struct {
			vaultSub string
			dstRoot  string
			subPath  string
		}{
			{"Chrome", targetLocal, filepath.Join("Google", "Chrome", "User Data", "Default")},
			{"Edge", targetLocal, filepath.Join("Microsoft", "Edge", "User Data", "Default")},
			{"Firefox", targetRoaming, filepath.Join("Mozilla", "Firefox", "Profiles")},
		}

		for _, br := range browserRestore {
			src := filepath.Join(browserVault, br.vaultSub)
			if _, err := os.Stat(src); os.IsNotExist(err) {
				continue
			}
			dst := filepath.Join(br.dstRoot, br.subPath)
			if err := os.MkdirAll(dst, 0755); err != nil {
				ctx.Logger.Warn("[data_inject] mkdir %s: %v", dst, err)
				continue
			}
			opts := fileengine.DefaultCopyOptions()
			result, err := fileengine.CopyTree(fileengine.CopyOptions{
				SourceDir:          src,
				DestDir:            dst,
				RetryCount:         3,
				RetryDelay:         opts.RetryDelay,
				PreserveTimestamps: true,
				Overwrite:          true,
			})
			if err != nil {
				ctx.Logger.Warn("[data_inject] browser restore %s: %v", br.vaultSub, err)
				continue
			}
			ctx.Logger.Info("[data_inject]   browser %s: %d files restored to %s", br.vaultSub, result.FilesCopied, dst)
		}
	}

	ctx.Logger.Info("[data_inject] data injection complete")
	return nil
}

func handleFirstLoginInit(ctx *pipeline.Context) error {
	ctx.Logger.Info("[first_login_init] preparing first login (stub)")
	return nil
}

func handleProfileCleanup(ctx *pipeline.Context) error {
	ctx.Logger.Info("[profile_cleanup] cleaning up temporary profile")
	if ctx.Data.DryRun {
		ctx.Logger.Info("[profile_cleanup] dry-run — skipping")
		return nil
	}

	if ctx.Data.TempUsername == "" {
		ctx.Logger.Info("[profile_cleanup] no temp profile to clean")
		return nil
	}

	tmpFile := filepath.Join(os.TempDir(), "wpre_cleanup.json")
	if ctx.RunPS != nil {
		if _, err := ctx.RunPS("ps/cleanup.ps1",
			"-Action", "profile",
			"-Username", ctx.Data.TempUsername,
			"-OutputPath", tmpFile,
		); err != nil {
			ctx.Logger.Warn("[profile_cleanup] temp profile cleanup error (non-fatal): %v", err)
		}
	}

	if ctx.RunPS != nil {
		if _, err := ctx.RunPS("ps/cleanup.ps1", "-Action", "autologin"); err != nil {
			ctx.Logger.Warn("[profile_cleanup] autologin cleanup error (non-fatal): %v", err)
		}
	}

	os.Remove(tmpFile)
	ctx.Logger.Info("[profile_cleanup] cleanup complete")
	return nil
}

func handleFinalValidation(ctx *pipeline.Context) error {
	ctx.Logger.Info("[final_validation] running validation checks")
	if ctx.Data.DryRun {
		ctx.Logger.Info("[final_validation] dry-run — skipping")
		return nil
	}

	vaultRoot := ctx.Data.VaultRoot
	if _, err := os.Stat(vaultRoot); os.IsNotExist(err) {
		return fmt.Errorf("vault root missing: %s", vaultRoot)
	}

	subdirs := []string{"UserData", "Cloud", "Browsers", "Outlook"}
	for _, dir := range subdirs {
		path := filepath.Join(vaultRoot, dir)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			ctx.Logger.Warn("[final_validation] optional directory missing: %s", path)
			continue
		}
		files, _ := fileengine.HashTree(path)
		ctx.Logger.Info("[final_validation] %s: %d files", dir, len(files))
	}

	ctx.Logger.Info("[final_validation] validation complete")
	return nil
}

func handleComplete(ctx *pipeline.Context) error {
	ctx.Logger.Info("[complete] === WPRE migration finished ===")
	ctx.Logger.Info("[complete] vault: %s", ctx.Data.VaultRoot)
	return nil
}
