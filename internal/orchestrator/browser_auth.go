package orchestrator

import (
	"fmt"
	"os"
	"path/filepath"

	"wpre/internal/config"
	"wpre/internal/fileengine"
	"wpre/internal/pipeline"
)

type browserAuthFiles struct {
	Name       string
	VaultSub   string
	ProfileDir string
	Files      []string
	Dirs       []string
}

func resolveChromiumAuth(localAppData, vendor, product string) (*browserAuthFiles, error) {
	userData := filepath.Join(localAppData, vendor, product, "User Data")
	defaultDir := filepath.Join(userData, "Default")
	if _, err := os.Stat(defaultDir); err != nil {
		dirs, err := os.ReadDir(userData)
		if err != nil {
			return nil, fmt.Errorf("cannot read %s: %w", userData, err)
		}
		for _, d := range dirs {
			if d.IsDir() {
				profileDir := filepath.Join(userData, d.Name())
				if _, err := os.Stat(filepath.Join(profileDir, "Cookies")); err == nil {
					defaultDir = profileDir
					break
				}
			}
		}
		if _, err := os.Stat(defaultDir); err != nil {
			return nil, fmt.Errorf("no profile dir found in %s", userData)
		}
	}
	return &browserAuthFiles{
		Name:       product,
		VaultSub:   product,
		ProfileDir: defaultDir,
		Files: []string{
			"Cookies",
			"Network/Cookies",
			"Login Data",
			"Login Data For Account",
		},
		Dirs: []string{
			"Sessions",
			"Session Storage",
		},
	}, nil
}

func resolveFirefoxAuth(roamingAppData string) (*browserAuthFiles, error) {
	profilesDir := filepath.Join(roamingAppData, "Mozilla", "Firefox", "Profiles")
	entries, err := os.ReadDir(profilesDir)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		profileDir := filepath.Join(profilesDir, e.Name())
		if _, err := os.Stat(filepath.Join(profileDir, "cookies.sqlite")); err == nil {
			return &browserAuthFiles{
				Name:       "Firefox",
				VaultSub:   "Firefox",
				ProfileDir: profileDir,
				Files: []string{
					"cookies.sqlite",
					"logins.json",
					"signons.sqlite",
				},
				Dirs: []string{
					"sessionstore-backups",
				},
			}, nil
		}
	}
	return nil, fmt.Errorf("no Firefox profile with cookies found in %s", profilesDir)
}

func browserHarvestSelectors(cfg *config.BrowserConfig) map[string][]string {
	sel := map[string][]string{}
	if cfg.Chrome {
		sel["chrome"] = nil
	}
	if cfg.Edge {
		sel["edge"] = nil
	}
	if cfg.Firefox {
		sel["firefox"] = nil
	}
	return sel
}

func harvestBrowserAuth(ctx *pipeline.Context, sourceProfile string) error {
	cfg := ctx.Config.(*config.Config)
	browsers := browserHarvestSelectors(&cfg.Browsers)
	if len(browsers) == 0 {
		ctx.Logger.Info("[browser_auth] no browsers enabled in config — skipping")
		return nil
	}

	localAppData := filepath.Join(sourceProfile, "AppData", "Local")
	roamingAppData := filepath.Join(sourceProfile, "AppData", "Roaming")
	vaultBrowsers := filepath.Join(ctx.Data.VaultRoot, "Browsers")

	type resolveFn func() (*browserAuthFiles, error)
	resolvers := map[string]resolveFn{}

	if _, ok := browsers["chrome"]; ok {
		resolvers["chrome"] = func() (*browserAuthFiles, error) {
			return resolveChromiumAuth(localAppData, "Google", "Chrome")
		}
	}
	if _, ok := browsers["edge"]; ok {
		resolvers["edge"] = func() (*browserAuthFiles, error) {
			return resolveChromiumAuth(localAppData, "Microsoft", "Edge")
		}
	}
	if _, ok := browsers["firefox"]; ok {
		resolvers["firefox"] = func() (*browserAuthFiles, error) {
			return resolveFirefoxAuth(roamingAppData)
		}
	}

	for name, resolve := range resolvers {
		info, err := resolve()
		if err != nil {
			ctx.Logger.Warn("[browser_auth] %s: %v — skipping", name, err)
			continue
		}
		ctx.Logger.Info("[browser_auth] found %s profile: %s", info.Name, info.ProfileDir)

		dest := filepath.Join(vaultBrowsers, info.VaultSub)
		if err := os.MkdirAll(dest, 0755); err != nil {
			ctx.Logger.Error("[browser_auth] failed to create vault dir for %s: %v", info.Name, err)
			continue
		}

		cpCfg := fileengine.DefaultCopyOptions()

		for _, rel := range info.Files {
			src := filepath.Join(info.ProfileDir, rel)
			if _, err := os.Stat(src); os.IsNotExist(err) {
				continue
			}
			dst := filepath.Join(dest, rel)
			if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
				ctx.Logger.Warn("[browser_auth] mkdir %s: %v", filepath.Dir(dst), err)
				continue
			}

			r, err := fileengine.CopyTree(fileengine.CopyOptions{
				SourceDir:          src,
				DestDir:            dst,
				RetryCount:         cpCfg.RetryCount,
				RetryDelay:         cpCfg.RetryDelay,
				PreserveTimestamps: true,
				Overwrite:          true,
			})
			if err != nil {
				ctx.Logger.Warn("[browser_auth] copy %s: %v", rel, err)
				continue
			}
			ctx.Logger.Info("[browser_auth]   %s/%s: %d files copied", info.Name, rel, r.FilesCopied)
		}

		for _, rel := range info.Dirs {
			src := filepath.Join(info.ProfileDir, rel)
			if _, err := os.Stat(src); os.IsNotExist(err) {
				continue
			}
			dst := filepath.Join(dest, rel)
			r, err := fileengine.CopyTree(fileengine.CopyOptions{
				SourceDir:          src,
				DestDir:            dst,
				RetryCount:         cpCfg.RetryCount,
				RetryDelay:         cpCfg.RetryDelay,
				PreserveTimestamps: true,
				Overwrite:          true,
			})
			if err != nil {
				ctx.Logger.Warn("[browser_auth] copy dir %s: %v", rel, err)
				continue
			}
			ctx.Logger.Info("[browser_auth]   %s/%s: %d files copied", info.Name, rel, r.FilesCopied)
		}

		ctx.Logger.Info("[browser_auth] %s auth data saved to vault", info.Name)
	}

	return nil
}

func findSourceProfile(ctx *pipeline.Context) string {
	if ctx.Data.ScanResults == nil {
		return ""
	}
	for _, p := range ctx.Data.ScanResults.Profiles {
		if !p.Exists {
			continue
		}
		if isSystemSID(p.SID) {
			continue
		}
		return p.Path
	}
	return ""
}
