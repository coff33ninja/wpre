# Windows Profile Reconstruction Engine (WPRE)

> A deterministic multi-stage user environment reconstruction system  
> Go orchestrator + PowerShell actuators + Python data processors  
> Identity is disposable. Data is permanent. State is rebuildable.

---

## Table of Contents

1. [Core Philosophy](#1-core-philosophy)
2. [System Architecture](#2-system-architecture)
3. [State Machine Pipeline](#3-state-machine-pipeline)
4. [Module Reference](#4-module-reference)
5. [Data Model](#5-data-model)
6. [OneDrive Harvest Module](#6-onedrive-harvest-module)
7. [Browser Module](#7-browser-module)
8. [Outlook Module](#8-outlook-module)
9. [Dual-Profile Lifecycle](#9-dual-profile-lifecycle)
10. [Engineering Constraints](#10-engineering-constraints)
11. [Repo Layout](#11-repo-layout)
12. [Implementation Roadmap](#12-implementation-roadmap)
13. [Verification & Safety](#13-verification--safety)

---

## 1. Core Philosophy

```
┌──────────────────────────────────────────────────────────────┐
│  Identity is disposable — don't preserve broken login states │
│  Data is sacred — everything user-created must survive       │
│  Cloud is just storage — never a source of authority         │
│  State is rebuildable — system can be reconstructed cleanly  │
└──────────────────────────────────────────────────────────────┘
```

**What this is NOT:**
- Not a repair tool
- Not a profile copier
- Not a cloud sync tool
- Not a Microsoft account manipulator

**What this IS:**
- A deterministic user environment reconstruction engine
- A staged migration hypervisor for Windows identity
- A technician-grade data extraction + profile rebuild system

---

## 2. System Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    GO ORCHESTRATOR (single binary)               │
│                                                                  │
│  ┌─────────────┐  ┌──────────────┐  ┌────────────────────────┐ │
│  │ State Engine │  │ Module Loader│  │ Pipeline Controller    │ │
│  │ (state.go)   │  │ (loader.go) │  │ (pipeline.go)          │ │
│  └──────┬───────┘  └──────┬───────┘  └───────────┬────────────┘ │
│         │                 │                       │              │
│         ▼                 ▼                       ▼              │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                  Execution Layer                         │    │
│  │  ┌────────────┐  ┌──────────┐  ┌───────────────────┐   │    │
│  │  │ PS1 Runner │  │ PY Runner│  │ File Engine       │   │    │
│  │  │ (ps1.go)   │  │ (py.go)  │  │ (copy/hash/dedupe)│   │    │
│  │  └─────┬──────┘  └────┬─────┘  └────────┬──────────┘   │    │
│  └────────┼──────────────┼──────────────────┼──────────────┘    │
└───────────┼──────────────┼──────────────────┼───────────────────┘
            │              │                  │
            ▼              ▼                  ▼
   ┌──────────────┐ ┌──────────────┐ ┌──────────────────┐
   │  PowerShell  │ │   Python     │ │  Go Native       │
   │  Modules     │ │  Modules     │ │  FS + Registry   │
   │              │ │              │ │  + Syscalls      │
   ├──────────────┤ ├──────────────┤ ├──────────────────┤
   │ • User mgmt  │ │ • SQLite par │ │ • File copy      │
   │ • SID ops    │ │ • PST insp   │ │ • Hashing        │
   │ • Permissions│ │ • Dedup      │ │ • Dir walk       │
   │ • Registry   │ │ • Analysis   │ │ • Logging        │
   │ • OneDrive   │ │ • Reporting  │ │ • Rollback       │
   └──────────────┘ └──────────────┘ └──────────────────┘
```

### Language Roles

| Layer | Language | Responsibility |
|-------|----------|----------------|
| Orchestrator | **Go** | State machine, pipeline control, module dispatch, logging, rollback, reboot handling. Single static binary. |
| System Control | **PowerShell** | User creation, SID operations, folder permissions, registry edits, OneDrive state checks. Bundled as `.ps1` resources embedded in Go binary. |
| Data Intelligence | **Python** | SQLite parsing (browser history), Outlook PST inspection, structured file analysis, deduplication, reporting. Bundled as `.py` resources or compiled to standalone. |

---

## 3. State Machine Pipeline

```
 BOOTSTRAP
     │
     ▼
 PROFILE_SCAN ──────────────────────────────────────────┐
     │                                                   │
     ▼                                                   │
 BACKUP_EXISTING (optional: restore path skip)           │
     │                                                   │
     ▼                                                   │
 TEMP_PROFILE_CREATE                                     │
     │                                                   │
     ▼                                                   │
 DATA_HARVEST ─────────────────────┐                     │
     │                             │                     │
     ▼                             ▼                     │
 ONEDRIVE_EXTRACT          APP_STATE_CAPTURE             │
     │                             │                     │
     └──────────┬──────────────────┘                     │
                ▼                                        │
         NEW_PROFILE_CREATE                              │
                │                                        │
                ▼                                        │
         DATA_INJECTION                                  │
                │                                        │
                ▼                                        │
         FIRST_LOGIN_INIT                                │
                │                                        │
                ▼                                        │
         OLD_PROFILE_CLEANUP ◄───────────────────────────┘
                │
                ▼
         FINAL_VALIDATION
                │
                ▼
            COMPLETE
```

### State Definitions

```go
type PipelineState int

const (
    StateBootstrap         PipelineState = iota // 0  - init checks
    StateProfileScan                            // 1  - detect existing profiles
    StateBackupExisting                         // 2  - optional full backup
    StateTempProfileCreate                      // 3  - create temp admin
    StateDataHarvest                            // 4  - copy safe user data
    StateOneDriveExtract                        // 5  - harvest OneDrive
    StateAppCapture                             // 6  - browsers, Outlook, apps
    StateNewProfileCreate                       // 7  - build target profile
    StateDataInject                             // 8  - inject harvested data
    StateFirstLoginInit                         // 9  - prep first login
    StateProfileCleanup                         // 10 - delete temp profile
    StateFinalValidation                        // 11 - verify integrity
    StateComplete                               // 12 - done
)
```

### Transition Rules

| From | To | Condition |
|------|----|-----------|
| Bootstrap | ProfileScan | Admin privileges confirmed |
| ProfileScan | BackupExisting | `--backup` flag or `restore mode` |
| ProfileScan | TempProfileCreate | No backup requested |
| TempProfileCreate | DataHarvest | Temp admin logged in + verified |
| DataHarvest | OneDriveExtract | All user folders staged |
| DataHarvest | AppCapture | No OneDrive detected |
| OneDriveExtract | AppCapture | OneDrive harvest complete |
| AppCapture | NewProfileCreate | All app states captured |
| NewProfileCreate | DataInject | Target profile SID confirmed |
| DataInject | FirstLoginInit | All data written + verified |
| FirstLoginInit | ProfileCleanup | First login detected + ready |
| ProfileCleanup | FinalValidation | Temp profile removed |
| FinalValidation | Complete | All checks pass |
| *Any* | *Bootstrap* | Error → rollback → restart |

---

## 4. Module Reference

### 4.1 Go Core Modules (built-in)

```
orchestrator/
├── main.go              # Entry point, flag parsing
├── state.go             # Pipeline state machine
├── pipeline.go          # Pipeline controller, transition logic
├── loader.go            # Module discovery + execution dispatch
├── executor_ps.go       # PowerShell runner
├── executor_py.go       # Python runner
├── file_engine.go       # Copy, hash, deduplication, integrity
├── registry.go          # Registry read/write (syscall)
├── logging.go           # Structured logging + audit trail
├── rollback.go          # Rollback manager (state snapshots)
├── reboot.go            # Reboot orchestration + resume
└── config.go            # JSON/YAML config loader
```

### 4.2 PowerShell Modules (embedded resources)

```
modules/ps/
├── profile_scanner.ps1       # Detect existing profiles + state
├── temp_profile.ps1          # Create/destroy temp admin profile
├── new_profile.ps1           # Create target clean profile
├── sid_manager.ps1           # SID lookup, mapping, cleanup
├── folder_permissions.ps1    # ACL transfer utilities
├── registry_snapshot.ps1     # Registry hive dump + restore
├── onedrive_state.ps1        # OneDrive detection + pause + detach
├── user_folder_map.ps1       # Known folder resolution (FOLDERID_*)
└── cleanup.ps1               # Profile removal, temp file cleanup
```

### 4.3 Python Modules (embedded or bundled)

```
modules/py/
├── chrome_parser.py          # Chrome User Data extraction
├── edge_parser.py            # Edge User Data extraction
├── firefox_parser.py         # Firefox profile parsing
├── outlook_parser.py         # PST/OST detection + metadata
├── deduplicator.py           # File dedup by hash
├── app_detector.py           # Detect installed apps + configs
├── history_parser.py         # Browser history SQLite parsing (optional)
└── report_generator.py       # Final HTML/JSON migration report
```

---

## 5. Data Model

### 5.1 Data Classification

```
🟢 SAFE — always migrate
    Desktop, Documents, Downloads
    Pictures, Videos, Music
    Browser bookmarks
    App configs (non-auth)
    VS Code / dev configs
    SSH keys (explicit option)

🟡 SEMI-SAFE — selective / flag-gated
    AppData\Roaming (filtered)
    AppData\Local  (filtered)
    Outlook PST/OST files
    Browser history
    Fonts
    VPN configs

🔴 NEVER — strip / discard
    Windows credentials
    Browser cookies / login sessions
    Auth tokens / encrypted caches
    OneDrive sync metadata
    Registry SAM/SECURITY hives
    Old profile SID refs
```

### 5.2 Migration Vault Structure

```
C:\MigrationVault\
├── UserData\                  # Safe user files (Documents, Desktop, etc.)
│   ├── Desktop\
│   ├── Documents\
│   ├── Downloads\
│   ├── Pictures\
│   ├── Videos\
│   └── Music\
├── Cloud\                    # OneDrive harvest (structure preserved)
│   ├── [OneDrive folder root]\
│   └── manifest.json         # Harvest metadata
├── AppData\                  # Filtered app configs
│   ├── Roaming\
│   └── Local\
├── Browsers\                 # Browser data (bookmarks, clean state)
│   ├── Chrome\
│   ├── Edge\
│   └── Firefox\
├── Outlook\                  # PST/OST files
│   └── [detected files]\
├── Registry\                 # Selective registry hives
│   └── HKCU_apps.reg
├── Manifest.json             # Master migration manifest
├── FileIndex.json            # All harvested files with SHA256
└── Audit.log                 # Full operation log
```

### 5.3 Manifest Schema

```json
{
  "version": "1.0.0",
  "source_profile": "S-1-5-21-...",
  "source_user": "OldUser",
  "harvest_timestamp": "2026-06-09T14:30:00Z",
  "stages_completed": ["scan", "harvest", "onedrive", "app_capture"],
  "target_profile": "S-1-5-21-...",
  "target_user": "NewUser",
  "data": {
    "files_total": 15234,
    "files_copied": 15230,
    "files_error": 4,
    "size_bytes": 85899345920
  },
  "onedrive": {
    "detected": true,
    "harvested": true,
    "folders": ["OneDrive - Company"],
    "placeholders_resolved": 12,
    "integrity_verified": true
  },
  "apps": {
    "chrome": true,
    "edge": true,
    "firefox": false,
    "outlook_pst": ["archive.pst"]
  },
  "errors": [
    {"stage": "harvest", "file": "locked.docx", "error": "access_denied"}
  ]
}
```

---

## 6. OneDrive Harvest Module

### 6.1 Design Principle

> OneDrive is **harvested, not integrated**.  
> It is treated as a local folder cache, not a sync service.

### 6.2 State Machine

```
DETECT
  │
  ▼
STABILIZE ───┐
  │          │ pause sync
  ▼          │ wait for queue drain
FLUSH        │ verify file hydration
  │          │
  ▼          │
HARVEST ◄────┘
  │
  ▼
DECOUPLE
  │
  ▼
VERIFY
```

### 6.3 Detailed Steps

| Step | Action | PowerShell / Go |
|------|--------|-----------------|
| **DETECT** | Check OneDrive.exe running, sync client installed, known folders redirected | `Get-Process OneDrive*`, `Test-Path $env:USERPROFILE\OneDrive*` |
| **STABILIZE** | Pause sync, wait for queue drain, detect active file locks | `OneDrive.exe /pause`, check `HKCU\Software\Microsoft\OneDrive\Accounts\*` |
| **FLUSH** | Detect placeholder files (`FILE_ATTRIBUTE_RECALL_ON_DATA_ACCESS`), force hydration | `attrib` check, `fsutil` or Go `syscall.GetFileAttributes` |
| **HARVEST** | Copy all files to `MigrationVault\Cloud\`, strip `.db/.ini/.dat` sync junk | Go file engine with retry + hash verify |
| **DECOUPLE** | Sign out or unlink device, remove sync association, leave files intact | `OneDrive.exe /signout`, remove registry association |
| **VERIFY** | Compare file counts, check for missing files, validate integrity | Go hash manifest compare |

### 6.4 Critical: Placeholder Detection

```go
const (
    FILE_ATTRIBUTE_RECALL_ON_DATA_ACCESS = 0x00400000
    FILE_ATTRIBUTE_PINNED                = 0x00080000
    FILE_ATTRIBUTE_UNPINNED              = 0x00100000
)

func isPlaceholder(path string) bool {
    attrs, err := syscall.GetFileAttributes(syscall.StringToUTF16Ptr(path))
    if err != nil {
        return false
    }
    return attrs&FILE_ATTRIBUTE_RECALL_ON_DATA_ACCESS != 0
}
```

### 6.5 What Gets Stripped

| File Pattern | Reason |
|-------------|--------|
| `*.sync.db` | OneDrive sync database |
| `*.sync.id` | Sync identity marker |
| `desktop.ini` | Local folder metadata |
| `*.ini` (OneDrive dirs) | Config junk |
| `*.dat` (sync cache) | Transient state |
| `*.odl` | OneDrive log files |
| `Personal Delta*` | Delta sync markers |

---

## 7. Browser Module

### 7.1 Supported Browsers

| Browser | Profile Path | Key Data |
|---------|-------------|----------|
| Chrome | `%LOCALAPPDATA%\Google\Chrome\User Data\Default` | Bookmarks, Extensions, History (opt) |
| Edge | `%LOCALAPPDATA%\Microsoft\Edge\User Data\Default` | Bookmarks, Extensions, History (opt) |
| Firefox | `%APPDATA%\Mozilla\Firefox\Profiles\*.default-release` | Bookmarks, preferences, extensions |

### 7.2 Extraction Rules

```
🟢 SAFE — always extract
    Bookmarks (JSON)
    Extensions list
    Preferences (non-auth keys)
    Local State (non-auth)
    Fonts, spell-check dicts

🟡 OPTIONAL — flag-gated
    History (SQLite — contains URLs only, no creds)
    Saved passwords (NEVER — encrypted anyway)
    Cookies (NEVER)

🔴 STRIP
    Login Data SQLite
    Cookies SQLite
    Network persistent state
    Origin-bound tokens
    Local Storage (app-dependent, usually junk)
```

### 7.3 Chrome/Edge Bookmark Path

```json
// User Data/Default/Bookmarks
{
  "roots": {
    "bookmark_bar": { ... },
    "other": { ... },
    "synced": { ... }
  }
}
```

➡ Copy to: `MigrationVault\Browsers\Chrome\Bookmarks.json`

---

## 8. Outlook Module

### 8.1 Three Layers

```
Layer 1: PST/OST Detection ──► Safe file copy to MigrationVault\Outlook\
Layer 2: Profile Registry    ──► Optional recreation (.reg export)
Layer 3: Account Re-init     ──► ALWAYS requires clean login (no identity preserved)
```

### 8.2 What We Do

| Action | Method |
|--------|--------|
| Detect PST/OST files | Scan `%USERPROFILE%\Documents\Outlook Files`, registry `HKCU\Software\Microsoft\Office\16.0\Outlook\Search`, known locations |
| Copy PST/OST | File engine with retry — these files can be large (2-50 GB) |
| Export Outlook profile registry | `HKCU\Software\Microsoft\Office\*\Outlook\Profiles` |
| Verify integrity (scanpst.exe) | Optional flag — calls Microsoft's ScanPST tool |

### 8.3 What We DON'T Do

- ❌ Restore email accounts automatically
- ❌ Preserve cached credentials
- ❌ Import old OST as active store
- ❌ Touch Exchange/IMAP configs

---

## 9. Dual-Profile Lifecycle

This is the core innovation — using a **temporary admin profile** as an isolation layer.

```
PHASE 1: BOOTSTRAP
  ┌──────────────────────────────────────────────┐
  │  Running as: CURRENT USER (temp admin)        │
  │  Goal: detect state, create temp profile      │
  └──────────────────────────────────────────────┘

PHASE 2: TEMP ADMIN PROFILE
  ┌──────────────────────────────────────────────┐
  │  Running as: WPRE_TempAdmin (local admin)     │
  │  Goal: unlock files, harvest data             │
  │  Bypasses: file locks, permission issues      │
  └──────────────────────────────────────────────┘
         │
         ▼
PHASE 3: NEW TARGET PROFILE CREATED
  ┌──────────────────────────────────────────────┐
  │  Running as: WPRE_TempAdmin                    │
  │  Creates: NewCleanUser (target)               │
  │  Injects: harvested data into target          │
  └──────────────────────────────────────────────┘
         │
         ▼
PHASE 4: SWAP
  ┌──────────────────────────────────────────────┐
  │  1. Mark WPRE_TempAdmin for deletion          │
  │  2. Log out WPRE_TempAdmin                    │
  │  3. First login as NewCleanUser               │
  │  4. Complete profile initialization           │
  └──────────────────────────────────────────────┘
         │
         ▼
PHASE 5: CLEANUP
  ┌──────────────────────────────────────────────┐
  │  1. Remove WPRE_TempAdmin profile             │
  │  2. Delete WPRE_TempAdmin user                │
  │  3. Clean up MigrationVault (optional)        │
  │  4. Final validation report                   │
  └──────────────────────────────────────────────┘
```

### Temp Profile Creation (PowerShell)

```powershell
net user WPRE_TempAdmin <GeneratedPassword> /add
net localgroup administrators WPRE_TempAdmin /add

# Generate a secure disposable password
$password = [System.Web.Security.Membership]::GeneratePassword(16, 4)

# Auto-login registry for reboot
New-ItemProperty -Path "HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Winlogon" `
    -Name DefaultUserName -Value "WPRE_TempAdmin"
New-ItemProperty -Path "HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Winlogon" `
    -Name DefaultPassword -Value $password
New-ItemProperty -Path "HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Winlogon" `
    -Name AutoAdminLogon -Value "1"
```

---

## 10. Engineering Constraints

### 🔴 Hard Rules

| # | Rule | Rationale |
|---|------|-----------|
| 1 | Never modify live profile while in use | Corruption, locks, instability |
| 2 | Always stage data before injection | Never direct-overwrite into target |
| 3 | Always reboot between major phases | Windows aggressively locks state |
| 4 | Never trust OneDrive without hydration check | Placeholders destroy migration integrity |
| 5 | Never preserve auth state (cookies, tokens, creds) | Security boundary + clean slate |
| 6 | Always verify file integrity after copy | Silent data loss is worst failure mode |
| 7 | Always have rollback capability per stage | Partial migration is unrecoverable |

### 🟡 Soft Rules

| # | Rule | Rationale |
|---|------|-----------|
| 8 | Prefer file count verification over size | More reliable for mixed file types |
| 9 | Log EVERYTHING to structured audit file | Debugging without it is impossible |
| 10 | Default to skip, not include (AppData) | Reduces bloat, avoids edge cases |

---

## 11. Repo Layout

```
wpre/
├── DESIGN.md                 # This document
├── README.md                 # Project overview
├── go.mod                    # Go module
├── main.go                   # Entry point
│
├── cmd/
│   └── wpre/
│       └── main.go           # CLI entry
│
├── internal/
│   ├── state/
│   │   └── state.go          # Pipeline state machine
│   ├── pipeline/
│   │   └── pipeline.go       # Pipeline orchestrator
│   ├── executor/
│   │   ├── executor.go       # Module executor interface
│   │   ├── powershell.go     # PowerShell runner
│   │   └── python.go         # Python runner
│   ├── fileengine/
│   │   ├── copy.go           # Copy with retry + hash verify
│   │   ├── hash.go           # SHA256 hashing
│   │   ├── dedupe.go         # Deduplication by hash
│   │   └── integrity.go      # Integrity verification
│   ├── registry/
│   │   └── registry.go       # Windows registry access
│   ├── logging/
│   │   └── log.go            # Structured logging
│   ├── rollback/
│   │   └── rollback.go       # Stage-level rollback
│   ├── reboot/
│   │   └── reboot.go         # Reboot orchestration
│   └── config/
│       └── config.go         # YAML/JSON config
│
├── modules/
│   ├── ps/
│   │   ├── profile_scanner.ps1
│   │   ├── temp_profile.ps1
│   │   ├── new_profile.ps1
│   │   ├── sid_manager.ps1
│   │   ├── folder_permissions.ps1
│   │   ├── registry_snapshot.ps1
│   │   ├── onedrive_state.ps1
│   │   ├── user_folder_map.ps1
│   │   └── cleanup.ps1
│   └── py/
│       ├── chrome_parser.py
│       ├── edge_parser.py
│       ├── firefox_parser.py
│       ├── outlook_parser.py
│       ├── deduplicator.py
│       ├── app_detector.py
│       ├── history_parser.py
│       └── report_generator.py
│
├── resources/
│   ├── embed.go              # Go embed directives
│   └── migrations/
│       └── default_config.yaml
│
├── scripts/
│   ├── build.ps1             # Build single binary
│   └── test.ps1              # Test suite runner
│
└── test/
    ├── mocks/                # Mock Windows environment
    └── integration/          # Integration tests
```

---

## 12. Implementation Roadmap

### Phase 0: Foundation (MVP)

```
Goal: Working state machine + file engine + temp profile creation
Est: ~2 weeks
```

- [x] State machine skeleton (Go)
- [ ] File engine (copy with retry + hash + integrity)
- [ ] PowerShell runner (Go → PS1 bridge)
- [ ] Temp profile module (create + verify + destroy)
- [ ] Profile scanner (detect existing profiles)
- [ ] Configuration loader (YAML)
- [ ] Basic logging + audit trail
- [ ] Rollback shell (per-stage snapshots)
- [ ] Build system (`embed` PowerShell modules into binary)

**Deliverable**: Binary that can scan profiles, create temp admin, copy user data to vault.

### Phase 1: OneDrive Harvest

```
Goal: Safe OneDrive extraction + detach
Est: ~1 week
```

- [ ] OneDrive state detector (process + registry)
- [ ] Sync pauser (`OneDrive.exe /pause`)
- [ ] Queue drain waiter (poll sync status)
- [ ] Placeholder detector (Go `GetFileAttributes`)
- [ ] File hydrator (force full download)
- [ ] Harvest engine (copy + strip metadata)
- [ ] Integrity verifier (file count + hash check)
- [ ] Controlled detach (sign out, unlink, remove association)
- [ ] OneDrive manifest generation

**Deliverable**: OneDrive harvest module, end-to-end.

### Phase 2: Browser + App Capture

```
Goal: Chrome/Edge/Firefox extraction + Outlook PST
Est: ~2 weeks
```

- [ ] Chrome parser (bookmarks, extensions, history opt)
- [ ] Edge parser (same structure as Chrome)
- [ ] Firefox parser (profile.ini + places.sqlite)
- [ ] App detector (installed app configs)
- [ ] Outlook PST/OST detector
- [ ] PST file copier (large file support)
- [ ] Outlook profile registry export
- [ ] Deduplication engine
- [ ] App manifest generation

**Deliverable**: Browser + Outlook data extraction modules.

### Phase 3: Profile Reconstruction

```
Goal: Target profile creation + data injection + swap
Est: ~2 weeks
```

- [ ] New profile creator (PowerShell `New-LocalUser`)
- [ ] SID mapping and transfer
- [ ] Data injection engine (vault → target)
- [ ] Folder permission mirroring
- [ ] Registry key injection (safe subset)
- [ ] First-login initialization
- [ ] Temp profile cleanup
- [ ] Reboot orchestration
- [ ] Resume-from-reboot logic

**Deliverable**: Complete profile swap pipeline.

### Phase 4: Safety + Validation

```
Goal: Robust error handling, verification, reporting
Est: ~1 week
```

- [ ] Comprehensive rollback (per-stage + full)
- [ ] Integrity verification module (all files)
- [ ] Migration report (HTML + JSON)
- [ ] Error classification (warning / recoverable / fatal)
- [ ] Dry-run mode (scan + simulate, no writes)
- [ ] Safe mode (skip dangerous operations)
- [ ] Crash recovery (detect incomplete state + resume)

**Deliverable**: Production-grade safety systems.

### Phase 5: Polish + Distribution

```
Goal: Single binary distribution, GUI option
Est: ~2 weeks
```

- [ ] Windows cross-compilation (x64, arm64)
- [ ] Embedded Python runtime (if needed)
- [ ] CLI polish (flags, help, progress bars)
- [ ] Administrator privilege detection
- [ ] GUI wrapper (Tauri + React or native Win32)
- [ ] MSI installer (optional)
- [ ] Documentation

**Deliverable**: Distribution-ready tool.

---

## 13. Verification & Safety

### Stage Verification Checklist

| Stage | Verification | Method |
|-------|-------------|--------|
| ProfileScan | Profiles detected match registry + FS | Cross-reference SAM + `C:\Users` |
| TempProfileCreate | User exists, is admin, can log in | `net user`, `net localgroup`, test login |
| DataHarvest | File count + hash match | Compare manifest |
| OneDriveExtract | No placeholders remain, all files verified | `attrib` + hash manifest |
| AppCapture | All targeted app data present | Count-based verification |
| NewProfileCreate | SID created, folders initialized, no errors | Registry check |
| DataInject | Files present in target, permissions correct | ACL check + hash |
| FirstLoginInit | User can log in, folders populate | Test login |
| ProfileCleanup | Temp user deleted, no orphaned refs | SAM scan |
| FinalValidation | All stages verified, report generated | Automate |

### Rollback Triggers

| Condition | Action |
|-----------|--------|
| Stage timeout exceeded | Restart stage (up to 3x), then abort |
| File copy failure (access denied) | Log, skip, continue (unless critical) |
| OneDrive placeholders unhydratable | Warn, skip dehydrated files, continue |
| Temp profile creation fails | Abort — cannot proceed without isolation |
| Target profile creation fails | Rollback to pre-profile state |
| Integrity check failure (any stage) | Retry failed items, abort if persistent |
| Reboot interrupted | Resume from last checkpoint |

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Complete — all stages successful |
| 1 | Complete with warnings — non-critical errors |
| 2 | Aborted — unrecoverable error, rollback applied |
| 3 | Partial — some stages complete, manual intervention needed |
| 4 | Validation failure — data integrity compromised |
| 99 | Internal error — bug or unexpected state |

---

## Appendix A: OneDrive State Machine Detail

```
                    ┌──────────┐
                    │  UNKNOWN │  No OneDrive detected
                    └────┬─────┘
                         │ OneDrive.exe running
                         ▼
                    ┌──────────┐
                    │  ACTIVE  │  Sync client running, signed in
                    └────┬─────┘
                         │ /pause
                         ▼
                    ┌──────────┐
                    │  PAUSED  │  Sync paused, queue may be non-empty
                    └────┬─────┘
                         │ drain complete
                         ▼
                    ┌──────────┐
                    │  DRAINED │  Queue empty, no pending operations
                    └────┬─────┘
                         │ hydration check
                         ▼
                    ┌──────────┐
                    │  HYDRATED│  All files fully local (no placeholders)
                    └────┬─────┘
                         │ copy to vault
                         ▼
                    ┌──────────┐
                    │ HARVESTED│  Files copied, manifest created
                    └────┬─────┘
                         │ /signout
                         ▼
                    ┌──────────┐
                    │ DETACHED │  Association removed, files remain
                    └──────────┘
```

## Appendix B: Config Schema (YAML)

```yaml
# wpre.yaml — default config
version: "1.0"

pipeline:
  stages:
    - profile_scan
    - temp_profile_create
    - data_harvest
    - onedrive_extract
    - app_capture
    - new_profile_create
    - data_inject
    - profile_cleanup
    - final_validation

  reboot_between_stages: true
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
  include_history: false     # opt-in
  include_extensions: true

outlook:
  enabled: true
  detect_pst: true
  detect_ost: true
  copy_pst: true
  export_profile_reg: true
  verify_with_scanpst: false  # opt-in, slow

data:
  vault_root: "C:\\MigrationVault"
  include_appdata_roaming: false  # opt-in
  include_appdata_local: false    # opt-in
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
    - "$RECYCLE.BIN"
    - "System Volume Information"

logging:
  level: info                 # debug | info | warn | error
  audit_file: "wpse-audit.log"
  report_format: "json"       # json | html

safety:
  dry_run: false
  safe_mode: false            # skip dangerous ops
  rollback_enabled: true
  integrity_check: true
  require_admin: true
```

---

*This is a living design document. As implementation progresses, update sections with real API decisions, edge cases discovered, and architectural corrections.*
