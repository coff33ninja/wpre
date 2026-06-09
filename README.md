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
| 7 | **app_capture** | Browser/Outlook data extraction (stub) | Python parsers |
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

### Stripped — never preserved
- OneDrive sync database (`*.sync.db`, `*.odl`, etc.)
- Browser cookies, login sessions, auth tokens
- Windows credentials
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
