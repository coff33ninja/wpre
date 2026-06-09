# WPRE — Windows Profile Reconstruction Engine

A deterministic multi-stage Windows user environment reconstruction system.
Single binary. No cloud dependency. Identity is disposable. Data is permanent.

```text
[Scanner] → [Harvester] → [Rebuilder] → [Injector] → [Validator]
      ↓            ↓
  PS1/PY       Go File Engine
  modules      (retry + hash + integrity)
```

---

## Quick start

```powershell
# Build the binary (requires Go 1.22+)
.\scripts\build.ps1

# Run a safety check first — scans profiles without making changes
.\wpre.exe --dry-run

# Run full pipeline (requires admin)
.\wpre.exe

# Resume from a specific stage if something failed
.\wpre.exe --resume onedrive_extract
```

**Requirements:**
- Windows 10/11 (not tested on Server)
- Administrator privileges
- Go 1.22+ (to build from source)
- PowerShell 5.1+ (bundled with Windows)

---

## Usage

```
.\wpre.exe [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--config` | `wpre.yaml` | Path to YAML config file |
| `--dry-run` | `false` | Scan + simulate, no changes |
| `--safe-mode` | `false` | Skip dangerous operations |
| `--resume` | `""` | Resume from stage name |

### Stage names (for --resume)

```
bootstrap  profile_scan  backup_existing  temp_profile_create
data_harvest  onedrive_extract  app_capture  new_profile_create
data_inject  first_login_init  profile_cleanup  final_validation
```

---

## Pipeline stages

| # | Stage | What it does | External calls |
|---|-------|-------------|----------------|
| 1 | **bootstrap** | Admin check, config load, create vault dirs | — |
| 2 | **profile_scan** | Detect existing user profiles + OneDrive state | `profile_scanner.ps1` |
| 3 | **backup_existing** | Full profile backup (stub) | — |
| 4 | **temp_profile_create** | Create `WPRE_TempAdmin`, set auto-login | `temp_profile.ps1` |
| 5 | **data_harvest** | Copy Desktop/Documents/Downloads/ etc. to vault | Go file engine |
| 6 | **onedrive_extract** | Pause sync → hydrate → copy → sign out | `onedrive_state.ps1` |
| 7 | **app_capture** | Browser auth data (cookies, logins, sessions) → vault | Go file engine |
| 8 | **new_profile_create** | Create `NewUser` target profile | `new_profile.ps1` |
| 9 | **data_inject** | Copy vault data into target profile | Go file engine |
| 10 | **first_login_init** | Prepare first-login state (stub) | — |
| 11 | **profile_cleanup** | Delete temp user + auto-login registry | `cleanup.ps1` |
| 12 | **final_validation** | Walk vault, report file counts | Go file engine |
| 13 | **complete** | Final log | — |

---

## Configuration

Default config is embedded in the binary. Override with `wpre.yaml`:

```yaml
onedrive:
  enabled: true
  pause_before_harvest: true
  detach_after_harvest: true

browsers:
  chrome: true
  edge: true
  firefox: true
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
  # mailpv_path: "C:\\Tools\\mailpv.exe"   # optional NirSoft MailPV

data:
  vault_root: "C:\\MigrationVault"
  safe_folders:
    - Desktop
    - Documents
    - Downloads
    - Pictures
    - Videos
    - Music

logging:
  level: info

safety:
  dry_run: false
  require_admin: true
```

Full schema in `resources/embed.go` (`DefaultConfigYAML` constant) or `resources/migrations/default_config.yaml`.

---

## What gets harvested

### Safe — always migrated
- Desktop, Documents, Downloads, Pictures, Videos, Music
- OneDrive files (stripped of sync metadata)
- Browser bookmarks (via Python parsers)
- Outlook PST/OST files (via Python parsers)

### Configurable — browser auth data (opt-in via `browsers.include_*`)
- **Cookies** (`BrowserConfig.IncludeCookies` / `include_cookies`) — raw `Cookies` SQLite database from Chrome/Edge, `cookies.sqlite` from Firefox
- **Saved passwords** (`BrowserConfig.IncludePasswords` / `include_passwords`) — `Login Data` (Chrome/Edge), `logins.json` (Firefox)
- **Sessions** (`BrowserConfig.IncludeSessions` / `include_sessions`) — `Sessions/`, `Session Storage/` (Chrome/Edge), `sessionstore-backups/` (Firefox)

All three default to `true`. Disable individually in `wpre.yaml`.

> ⚠️ **DPAPI limitation**: Browser cookies and saved passwords are encrypted with Windows DPAPI, tied to the original user's SID and login password. The raw database files will be copied to the vault and restored to the new profile, but the browser will **not be able to decrypt them** on the new profile. The file data is preserved for forensic/extraction purposes.

> ✅ **Recommended approach — use browser sync**: After migration, log into your browser account on the new profile to restore all data from cloud sync:
>
> | Browser | Sync account | Restores |
> |---------|-------------|----------|
> | Chrome | Google Account (`chrome://settings/syncSetup`) | Bookmarks, passwords, open tabs, history, extensions + settings, payment info, addresses |
> | Edge | Microsoft Account (`edge://settings/profiles`) | Passwords, favorites, collections, extensions, open tabs, history, settings |
> | Firefox | Firefox Account (`about:preferences#sync`) | Bookmarks, logins, open tabs, history, add-ons, preferences |
>
> Browser sync is the most reliable way to preserve usable auth state across profiles. No DPAPI issues, no manual export steps. Make sure sync is enabled on the **source machine** before migration, then sign in on the new profile after WPRE completes.

### Configurable — Outlook profile & autocomplete (opt-in via `outlook.*`)
- **Profile registry** (`OutlookConfig.ExportProfileReg` / `export_profile_reg`) — exports `HKCU\Software\Microsoft\Office\<ver>\Outlook\Profiles` to `VaultRoot/Outlook/Registry/outlook_profiles.reg` for account server names, mailbox config
- **Autocomplete cache** (`OutlookConfig.BackupAutocomplete` / `backup_autocomplete`) — copies `Stream_Autocomplete_*.dat` (RoamCache) and `*.NK2` files to `VaultRoot/Outlook/Autocomplete/`. These power Outlook's predictive-text address suggestions when composing emails
- **Setup guide** (`OutlookConfig.GenerateSetupGuide` / `generate_setup_guide`) — generates `WPRE_Outlook_Setup_Guide.txt` in the vault and places a copy on the new user's desktop with step-by-step instructions for re-adding accounts and re-attaching PST files
- **MailPV** (`OutlookConfig.MailPVPath` / `mailpv_path`) — optional path to NirSoft MailPV (http://www.nirsoft.net/utils/mailpv.html). If provided, WPRE runs `mailpv.exe /stext` to extract visible email account passwords. Results saved to `VaultRoot/Outlook/MailPV/`

> ⚠️ **Antivirus false positives**: NirSoft tools (MailPV, etc.) are commonly flagged by antivirus software as "hacktools" or "password recovery tools." These are legitimate system administration utilities. If using MailPV, add an exclusion for the `mailpv.exe` binary and the vault output directory, or disable real-time protection temporarily. WPRE itself is a Go binary and is not flagged by any major AV vendor.
>
> ✅ **Recommended approach — Microsoft Modern Auth / OAuth 2.0**: Even with MailPV, most Outlook accounts now use Modern Auth (OAuth 2.0) which does not store a reusable password locally. The registry export provides your server names and mailbox paths; you will still need to re-authenticate interactively on the new profile. Use the setup guide placed on your desktop.
>
> ✅ **Autocomplete is per-machine**: The migrated RoamCache/NK2 files will restore your address autocomplete history. If suggestions don't appear immediately after migration, close Outlook, delete the new RoamCache `.dat` files, and restart — Outlook will rebuild from the NK2 backup.

### Stripped — never preserved
- OneDrive sync database (`*.sync.db`, `*.odl`, etc.)
- Windows credentials (DPAPI-protected, machine-specific)
- Old profile SID references
- `Thumbs.db`, `*.tmp`, `*.log`

---

## Project structure

```
wpre/
├── main.go                        # Entry point
├── cmd/wpre/main.go               # CLI entry (same logic)
│
├── internal/
│   ├── state/state.go             # 13-stage pipeline enum
│   ├── pipeline/
│   │   ├── pipeline.go            # Stage orchestrator + rollback
│   │   └── databus.go             # Runtime data bus types
│   ├── orchestrator/
│   │   ├── orchestrator.go        # Flag parsing, module extraction, wiring
│   │   ├── handlers.go            # All 13 stage implementations
│   │   └── util.go                # Admin check, password gen, SID filter
│   ├── executor/
│   │   ├── executor.go            # Base command executor
│   │   ├── powershell.go          # PS1 runner with module resolution
│   │   └── python.go              # PY runner with module resolution
│   ├── fileengine/
│   │   ├── copy.go                # Copy with retry + progress
│   │   ├── hash.go                # SHA256 file hashing
│   │   ├── dedupe.go              # Dedup index
│   │   └── integrity.go           # Verify copy integrity
│   ├── registry/registry.go       # Windows registry access
│   ├── logging/log.go             # Structured audit logging
│   ├── rollback/rollback.go       # Per-stage snapshot manager
│   ├── reboot/reboot.go           # Reboot orchestration
│   └── config/config.go           # YAML config loader
│
├── resources/
│   ├── embed.go                   # go:embed + module extractor
│   └── modules/
│       ├── ps/                    # 9 PowerShell modules
│       │   ├── profile_scanner.ps1
│       │   ├── temp_profile.ps1
│       │   ├── new_profile.ps1
│       │   ├── sid_manager.ps1
│       │   ├── folder_permissions.ps1
│       │   ├── registry_snapshot.ps1
│       │   ├── onedrive_state.ps1
│       │   ├── user_folder_map.ps1
│       │   └── cleanup.ps1
│       └── py/                    # 8 Python modules
│           ├── chrome_parser.py
│           ├── edge_parser.py
│           ├── firefox_parser.py
│           ├── outlook_parser.py
│           ├── deduplicator.py
│           ├── app_detector.py
│           ├── history_parser.py
│           └── report_generator.py
│
├── scripts/
│   ├── build.ps1                  # Build single binary
│   └── test.ps1                   # Run Go tests
│
├── pyproject.toml                 # Python dependency management (uv)
├── .python-version                # Python 3.12
├── go.mod
└── DESIGN.md                      # Full system architecture document
```

---

## Building

```powershell
# Native build
.\scripts\build.ps1

# Cross-compile for x64 + arm64
.\scripts\build.ps1 -CrossCompile
```

The binary is self-contained — all PowerShell and Python modules are embedded via `go:embed` and extracted to a temp directory at runtime.

---

## Python modules (uv)

```powershell
# Sync Python environment
uv sync

# Run individual parsers manually
python resources/modules/py/chrome_parser.py <output_dir>
python resources/modules/py/outlook_parser.py <output_dir>
```

---

## Development

```powershell
# Test
.\scripts\test.ps1

# Quick iteration
go run . --dry-run

# Clean up test artifacts created during development
net user WPRE_TempAdmin /delete 2>$null
net user NewUser /delete 2>$null
Remove-Item -Path "C:\MigrationVault" -Recurse -Force 2>$null
```

### Adding a new stage

1. Add a new `Stage` constant to `internal/state/state.go`
2. Add it to the `order` slice in `internal/pipeline/pipeline.go`
3. Write the handler function in `internal/orchestrator/handlers.go`
4. Register it in `registerHandlers()` in `internal/orchestrator/orchestrator.go`
5. Add it to `stageFromString()` for resume support

---

## Cross-machine migration (backup & restore)

> **⚠️ Status: Not yet implemented — see limitations below.**

WPRE is currently designed for **same-machine** profile reconstruction. The pipeline harvests data from a source profile and rebuilds it locally on the same Windows installation.

For **cross-machine** migration (backup to external media → rebuild on another PC), these features still need work:

| Feature | Status | Notes |
|---------|--------|-------|
| User-specified vault path | ✅ Partial | `data.vault_root` in config.yaml works, but no CLI flag |
| Export manifest (JSON index of all harvested data) | ❌ Missing | Needed to validate restore completeness |
| Compress vault to archive (ZIP/tarball) | ❌ Missing | Large harvests need portable packaging |
| Cross-machine restore pipeline stage | ❌ Missing | `data_inject` assumes same machine — won't handle new SID, new drive letters, or missing registry keys |
| Selective restore (choose folders at restore time) | ❌ Missing | All-or-nothing copy |
| Hardware-independent rebuild (no HW-specific registry refs) | ❌ Missing | Registry snapshots contain machine-specific paths |

### Workaround

You can manually copy `C:\MigrationVault` (or your configured `vault_root`) to external media, then place it at the same path on the target machine before running WPRE. Stage 3 (`backup_existing`) and Stage 7 (`app_capture`) are stubs — real backup/restore logic has not been implemented yet.

---

## Design

See [`DESIGN.md`](DESIGN.md) for the full architecture document covering:

- Core philosophy (identity is disposable, data is sacred)
- Dual-profile lifecycle (temp admin isolation)
- OneDrive harvest module design (detect → stabilize → flush → harvest → decouple)
- Browser/Outlook data rules (what's safe vs stripped)
- Rollback system and verification checks
- 5-phase implementation roadmap
