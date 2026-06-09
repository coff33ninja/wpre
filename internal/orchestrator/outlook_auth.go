package orchestrator

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"wpre/internal/config"
	"wpre/internal/fileengine"
	"wpre/internal/pipeline"
)

const mailpvURL = "https://www.nirsoft.net/toolsdownload/mailpv.zip"

type outlookProfileInfo struct {
	Found       bool   `json:"found"`
	ProfileName string `json:"profileName,omitempty"`
	RegPath     string `json:"regPath,omitempty"`
	PSTFiles    []string `json:"pstFiles,omitempty"`
	OSTFiles    []string `json:"ostFiles,omitempty"`
	AccountName string `json:"accountName,omitempty"`
	Email       string `json:"email,omitempty"`
}

type outlookAutocompleteFiles struct {
	RoamCache []string
	NK2Files  []string
}

func findOutlookAutocomplete(sourceProfile string) *outlookAutocompleteFiles {
	result := &outlookAutocompleteFiles{}
	roamCache := filepath.Join(sourceProfile, "AppData", "Roaming", "Microsoft", "Outlook", "RoamCache")
	if entries, err := os.ReadDir(roamCache); err == nil {
		for _, e := range entries {
			if !e.IsDir() && (strings.HasPrefix(e.Name(), "Stream_Autocomplete") || strings.HasPrefix(e.Name(), "Autocomplete")) {
				result.RoamCache = append(result.RoamCache, filepath.Join(roamCache, e.Name()))
			}
		}
	}
	outlookDir := filepath.Join(sourceProfile, "AppData", "Roaming", "Microsoft", "Outlook")
	if entries, err := os.ReadDir(outlookDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() && strings.EqualFold(filepath.Ext(e.Name()), ".nk2") {
				result.NK2Files = append(result.NK2Files, filepath.Join(outlookDir, e.Name()))
			}
		}
	}
	return result
}

func exportOutlookRegistry(vaultDir string) error {
	officeVersions := []string{"16.0", "15.0", "14.0"}
	profilesPath := ""
	for _, ver := range officeVersions {
		key := fmt.Sprintf("HKCU\\Software\\Microsoft\\Office\\%s\\Outlook\\Profiles", ver)
		if _, err := exec.Command("reg", "query", key).Output(); err == nil {
			profilesPath = key
			break
		}
	}
	if profilesPath == "" {
		altKey := "HKCU\\Software\\Microsoft\\Windows NT\\CurrentVersion\\Windows Messaging Subsystem\\Profiles"
		if _, err := exec.Command("reg", "query", altKey).Output(); err == nil {
			profilesPath = altKey
		}
	}
	if profilesPath == "" {
		return fmt.Errorf("no Outlook profile registry key found")
	}
	outFile := filepath.Join(vaultDir, "Registry", "outlook_profiles.reg")
	if err := os.MkdirAll(filepath.Dir(outFile), 0755); err != nil {
		return err
	}
	cmd := exec.Command("reg", "export", profilesPath, outFile, "/y")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("reg export failed: %w\n%s", err, out)
	}
	return nil
}

func downloadMailPV(logger pipeline.Logger) (string, error) {
	url := mailpvURL
	tmpZip := filepath.Join(os.TempDir(), "mailpv.zip")
	extractDir := filepath.Join(os.TempDir(), "mailpv_extracted")

	logger.Info("[mailpv] downloading from %s", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Referer", "https://www.nirsoft.net/utils/mailpv.html")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	out, err := os.Create(tmpZip)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	if _, err := io.Copy(out, resp.Body); err != nil {
		out.Close()
		os.Remove(tmpZip)
		return "", fmt.Errorf("failed to write zip: %w", err)
	}
	out.Close()

	logger.Info("[mailpv] extracting zip")

	os.RemoveAll(extractDir)
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		os.Remove(tmpZip)
		return "", err
	}

	zipReader, err := zip.OpenReader(tmpZip)
	if err != nil {
		os.Remove(tmpZip)
		return "", fmt.Errorf("failed to open zip: %w", err)
	}
	defer zipReader.Close()

	for _, f := range zipReader.File {
		if f.FileInfo().IsDir() {
			continue
		}
		destPath := filepath.Join(extractDir, filepath.Base(f.Name))
		rc, err := f.Open()
		if err != nil {
			continue
		}
		dst, err := os.Create(destPath)
		if err != nil {
			rc.Close()
			continue
		}
		io.Copy(dst, rc)
		dst.Close()
		rc.Close()
	}

	os.Remove(tmpZip)
	exePath := filepath.Join(extractDir, "mailpv.exe")

	if _, err := os.Stat(exePath); err != nil {
		return "", fmt.Errorf("mailpv.exe not found in zip: %w", err)
	}

	return exePath, nil
}

func runMailPV(exePath, vaultDir string) error {
	outDir := filepath.Join(vaultDir, "MailPV")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}
	txtOut := filepath.Join(outDir, "mailpv_output.txt")
	cmd := exec.Command(exePath, "/stext", txtOut)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("mailpv failed: %w\n%s", err, out)
	}
	return nil
}

func generateOutlookSetupGuide(vaultDir string) string {
	guide := `==================================================================
  WPRE — Outlook Migration Setup Guide
==================================================================

Your Outlook data has been migrated to the new profile. Follow the
steps below to reconnect your email accounts and data files.

---

STEP 1: Add your email accounts back to Outlook
------------------------------------------------------------------
Open Outlook and go to File > Account Settings > Account Settings.
Click "New" and re-enter your email credentials.

If you used MailPV or exported registry profiles, account config
is saved in:
  ` + filepath.Join(vaultDir, "Outlook", "Registry") + `
  ` + filepath.Join(vaultDir, "Outlook", "MailPV") + `

STEP 2: Re-attach PST data files
------------------------------------------------------------------
In Outlook, go to File > Open & Export > Open Outlook Data File.
Navigate to:
  ` + filepath.Join(vaultDir, "Outlook") + `

Select your .pst files to add them to the folder pane. Your sent
items, archived mail, and autocomplete history will appear once
the PST is attached.

STEP 3: Verify autocomplete (predictive address text)
------------------------------------------------------------------
Your autocomplete cache (NK2 / RoamCache) was restored to:
  %%APPDATA%%\\Microsoft\\Outlook\\RoamCache

When you start typing an email address in a new message, Outlook
should suggest previous recipients from your migrated history.

STEP 4: Send a test email
------------------------------------------------------------------
Create a new message, verify your autocomplete suggestions work,
and send a test to confirm outbound connectivity.

==================================================================
Troubleshooting
------------------------------------------------------------------
- If Outlook prompts for a password, you may need to re-authenticate
  using Modern Auth (OAuth 2.0) — Microsoft requires it for most
  accounts now.
- If autocomplete suggestions don't appear, close Outlook, delete
  the RoamCache .dat files in your new profile's RoamCache folder,
  and restart — Outlook will rebuild from the NK2 we restored.
- PST files attached via File > Open are not automatically mounted
  on next launch unless you set them as the default delivery location.
==================================================================
`
	outFile := filepath.Join(vaultDir, "Outlook", "WPRE_Outlook_Setup_Guide.txt")
	os.MkdirAll(filepath.Dir(outFile), 0755)
	os.WriteFile(outFile, []byte(guide), 0644)
	return outFile
}

func harvestOutlookAuth(ctx *pipeline.Context, sourceProfile string) error {
	cfg := ctx.Config.(*config.Config)
	if !cfg.Outlook.Enabled {
		ctx.Logger.Info("[outlook_auth] Outlook harvest disabled in config — skipping")
		return nil
	}

	vaultOutlook := filepath.Join(ctx.Data.VaultRoot, "Outlook")
	os.MkdirAll(vaultOutlook, 0755)

	if cfg.Outlook.ExportProfileReg {
		ctx.Logger.Info("[outlook_auth] exporting Outlook profile registry")
		if err := exportOutlookRegistry(vaultOutlook); err != nil {
			ctx.Logger.Warn("[outlook_auth] registry export failed (non-fatal): %v", err)
		} else {
			ctx.Logger.Info("[outlook_auth] Outlook profile registry exported")
		}
	}

	if cfg.Outlook.BackupAutocomplete {
		ctx.Logger.Info("[outlook_auth] backing up autocomplete cache")
		ac := findOutlookAutocomplete(sourceProfile)
		cpCfg := fileengine.DefaultCopyOptions()
		dest := filepath.Join(vaultOutlook, "Autocomplete")
		os.MkdirAll(dest, 0755)

		copied := 0
		for _, f := range ac.RoamCache {
			dst := filepath.Join(dest, filepath.Base(f))
			r, err := fileengine.CopyTree(fileengine.CopyOptions{
				SourceDir:          f,
				DestDir:            dst,
				RetryCount:         cpCfg.RetryCount,
				RetryDelay:         cpCfg.RetryDelay,
				PreserveTimestamps: true,
				Overwrite:          true,
			})
			if err == nil {
				copied += r.FilesCopied
			}
		}
		for _, f := range ac.NK2Files {
			dst := filepath.Join(dest, filepath.Base(f))
			r, err := fileengine.CopyTree(fileengine.CopyOptions{
				SourceDir:          f,
				DestDir:            dst,
				RetryCount:         cpCfg.RetryCount,
				RetryDelay:         cpCfg.RetryDelay,
				PreserveTimestamps: true,
				Overwrite:          true,
			})
			if err == nil {
				copied += r.FilesCopied
			}
		}
		if copied > 0 {
			acInfo := map[string]interface{}{
				"roamCacheFiles": len(ac.RoamCache),
				"nk2Files":       len(ac.NK2Files),
				"totalCopied":    copied,
			}
			infoPath := filepath.Join(dest, "autocomplete_info.json")
			if data, err := json.MarshalIndent(acInfo, "", "  "); err == nil {
				os.WriteFile(infoPath, data, 0644)
			}
			ctx.Logger.Info("[outlook_auth] autocomplete: %d roam cache, %d nk2 files copied", len(ac.RoamCache), len(ac.NK2Files))
		} else {
			ctx.Logger.Info("[outlook_auth] no autocomplete files found")
		}
	}

	mailPVExe := cfg.Outlook.MailPVPath
	if mailPVExe == "" && cfg.Outlook.MailPVAutoDownload {
		var err error
		mailPVExe, err = downloadMailPV(ctx.Logger)
		if err != nil {
			ctx.Logger.Warn("[outlook_auth] MailPV auto-download failed (non-fatal): %v", err)
			ctx.Logger.Warn("[outlook_auth] NirSoft tools are often flagged by antivirus — download mailpv.exe manually from https://www.nirsoft.net/utils/mailpv.html and set mailpv_path in config")
		}
	}
	if mailPVExe != "" {
		ctx.Logger.Info("[outlook_auth] running MailPV credential extractor: %s", mailPVExe)
		if err := runMailPV(mailPVExe, vaultOutlook); err != nil {
			ctx.Logger.Warn("[outlook_auth] MailPV failed (non-fatal): %v", err)
			ctx.Logger.Warn("[outlook_auth] MailPV is often flagged by antivirus — add an exclusion or run manually")
		} else {
			ctx.Logger.Info("[outlook_auth] MailPV credentials saved to %s", filepath.Join(vaultOutlook, "MailPV"))
		}
	}

	if cfg.Outlook.GenerateSetupGuide {
		ctx.Logger.Info("[outlook_auth] generating Outlook setup guide")
		guidePath := generateOutlookSetupGuide(vaultOutlook)
		ctx.Logger.Info("[outlook_auth] setup guide: %s", guidePath)
	}

	ctx.Logger.Info("[outlook_auth] Outlook auth harvest complete")
	return nil
}

func restoreOutlookAutocomplete(ctx *pipeline.Context, targetProfile string) {
	src := filepath.Join(ctx.Data.VaultRoot, "Outlook", "Autocomplete")
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return
	}
	dest := filepath.Join(targetProfile, "AppData", "Roaming", "Microsoft", "Outlook", "RoamCache")
	if err := os.MkdirAll(dest, 0755); err != nil {
		ctx.Logger.Warn("[data_inject] mkdir RoamCache: %v", err)
		return
	}
	cpCfg := fileengine.DefaultCopyOptions()
	r, err := fileengine.CopyTree(fileengine.CopyOptions{
		SourceDir:          src,
		DestDir:            dest,
		RetryCount:         cpCfg.RetryCount,
		RetryDelay:         cpCfg.RetryDelay,
		PreserveTimestamps: true,
		Overwrite:          true,
	})
	if err != nil {
		ctx.Logger.Warn("[data_inject] autocomplete restore: %v", err)
		return
	}
	ctx.Logger.Info("[data_inject] autocomplete: %d files restored to %s", r.FilesCopied, dest)
}

func placeOutlookSetupGuide(ctx *pipeline.Context, targetProfile string) {
	guideSrc := filepath.Join(ctx.Data.VaultRoot, "Outlook", "WPRE_Outlook_Setup_Guide.txt")
	if _, err := os.Stat(guideSrc); os.IsNotExist(err) {
		return
	}
	desktop := filepath.Join(targetProfile, "Desktop", "WPRE_Outlook_Setup.txt")
	if data, err := os.ReadFile(guideSrc); err == nil {
		os.WriteFile(desktop, data, 0644)
		ctx.Logger.Info("[data_inject] setup guide placed on desktop: %s", desktop)
	}
}
