# CLAUDE.md - NixOS Immich WebUI

## Project Overview

**Easy Immich Server** is a Go-based web application that provides an appliance-like experience for managing a NixOS host running Immich (a self-hosted photo and video backup solution). The project compiles into a single binary that serves a web interface for configuring and managing the entire system.

### Key Features
- Web-based NixOS configuration management
- Immich container lifecycle management (start/stop/update)
- USB backup functionality for photos and system configs
- Tailscale integration for remote access
- Email notification configuration
- System power management (reboot/poweroff)

### Current Status & Roadmap

- **Version**: Alpha development (v0.1.0-alpha.3 in progress)
- **Stage**: Active development, not production-ready
- **Main Branch**: `main`
- **Current Branch**: `v0.1.0-alpha.3/backups`

#### Roadmap

- **v0.1.0-alpha.1** (Complete)
  - Single configuration.nix file with stable Immich server config
  - Immich accessible at http://immich.local, admin UI at http://immich.local:8080
  - Web UI for viewing/modifying config, applying changes, minimal Immich container controls

- **v0.1.0-alpha.2** (Complete)
  - Email config via web UI
  - Embedded template files in binary
  - Power off/restart server from UI
  - Logging levels (Info, Error, Debug)
  - Basic USB backup (photos, config, DB dump)

- **v0.1.0-alpha.3** (In Progress - ~90% Complete)
  - ✅ JSON-based configuration management (replaces .nix template parsing)
  - ✅ Refactored modular architecture with clean package separation
  - ✅ Automatic rollback on nixos-rebuild failure
  - ✅ Input validation framework (timezone, time, email, Tailscale keys)
  - ✅ Concurrency safety with mutex protection
  - ✅ ML model selection support for Immich
  - ✅ Build tag system (dev/prod modes)
  - ⏳ Documentation updates for new features
  - ⏳ Backup system updates for new config locations

- **v0.1.0-beta.1** (Planned)
  - Mobile-first CSS/UI
  - Responsive UI with HTMX modals, progressive enhancement
  - Host system update button
  - GitHub binary releases

See `/docs/dev/todo.md` for detailed pending features.

## Project Structure

```
nixOS-immich-webui/
├── main.go                          # Slim initialization file (49 lines)
├── go.mod                          # Go module dependencies
├── internal/                       # Modular packages
│   ├── config/                     # Configuration management
│   │   ├── types.go               # ConfigVariables & data structures
│   │   ├── parser.go              # JSON parsing & file operations
│   │   ├── validation.go          # Input validation functions
│   │   ├── paths_dev.go           # Development build paths
│   │   └── paths_prod.go          # Production build paths
│   ├── handlers/                   # HTTP request handlers
│   │   ├── system.go              # System configuration endpoints
│   │   ├── immich.go              # Immich service management
│   │   └── backup.go              # Backup operations
│   ├── services/                   # Business logic services
│   │   └── backup.go              # Backup service implementation
│   ├── system/                     # System command operations
│   │   └── commands.go            # NixOS & Docker system commands
│   └── templates/                  # Embedded web templates
│       ├── embed.go               # Template embedding
│       └── web/                   # HTML templates
│           ├── index.html         # Main admin interface
│           ├── save.html          # Configuration confirmation page
│           ├── email_form.html    # Email configuration form fragment
│           ├── ml_form.html       # ML model selection form fragment
│           ├── backup_config.html # Backup configuration
│           ├── backup_dashboard.html # Backup status dashboard
│           ├── backup_form.html   # Backup form fragment
│           ├── backup_status.html # Backup status fragment
│           ├── disk_options.html  # Disk selection options fragment
│           └── no_disks.html      # No eligible disks message fragment
├── example/etc/nixos/             # Example NixOS configuration
│   ├── nixconfig.json             # JSON configuration file
│   ├── system.nix                 # System configuration module
│   ├── networking.nix             # Network configuration module
│   ├── immich.nix                 # Docker/Immich module
│   ├── remoteaccess.nix           # Tailscale VPN module
│   ├── zfs.nix                    # ZFS storage module
│   └── admin.nix                  # User packages module
├── docs/                          # Documentation
│   ├── dev/                       # Development docs
│   │   ├── todo.md                # TODO items
│   │   ├── features.md            # Feature roadmap
│   │   ├── configuration.md       # Configuration architecture
│   │   ├── environment.md         # Environment assumptions
│   │   ├── backups.md             # Backup functionality docs
│   │   └── considerations.md      # Development considerations
│   └── setup/                     # Setup documentation
│       ├── system.md              # System setup
│       ├── storage.md             # Storage configuration
│       └── remote-access.md       # Remote access setup
└── test/                          # Test configurations
    └── nixos/                     # Test NixOS configs
        └── nixconfig.json         # Test JSON configuration
```

## Technology Stack

### Backend
- **Language**: Go 1.23.3
- **HTTP Server**: Standard library `net/http`
- **Templating**: `html/template` (text/template removed in alpha.3)
- **File Embedding**: `embed` package for templates
- **Logging**: `log/slog` for structured logging
- **Concurrency**: Package-level mutexes for thread-safe config operations
- **Validation**: Custom validation framework for user inputs

### Frontend - Progressive Enhancement Strategy
- **Base Layer**: Semantic HTML forms with full functionality without JavaScript
- **Enhancement Layer**: HTMX 2.0.4 for dynamic interactions and reduced page reloads
- **Styling**: Vanilla CSS (mobile-first approach planned)
- **Progressive Enhancement Philosophy**:
  - All core functionality works without JavaScript
  - HTMX enhances UX with AJAX requests, partial page updates, and real-time status updates
  - Graceful degradation ensures accessibility and robustness

### HTMX Integration Pattern
The application follows a progressive enhancement model where:

1. **Base Functionality**: Traditional form submissions and page navigation
2. **HTMX Enhancement**: Added via `hx-*` attributes for:
   - Status polling (`hx-get="/status" hx-trigger="load, every 10s"`)
   - Form submissions with partial page updates (`hx-post="/email" hx-target="#email-form"`)
   - Dynamic content loading (`hx-get="/disks" hx-trigger="load"`)
   - Confirmation dialogs (`hx-confirm="Are you sure..."`)

Example from `index.html`:
```html
<!-- Works without JS as regular form -->
<form id="email-form" action="/email" method="post">
    <!-- HTMX enhances with partial updates -->
    <button type="submit" hx-post="/email" hx-target="#email-form">Submit</button>
</form>
```

### System Integration
- **OS**: NixOS (declarative Linux distribution)
- **Container Runtime**: Docker with docker-compose
- **Reverse Proxy**: Caddy
- **File System**: ZFS (required for tank pool)
- **Service Discovery**: Avahi (mDNS)
- **VPN**: Tailscale integration

## Build and Development

### Build Modes

The application supports two build modes controlled by Go build tags:

#### Development Build (uses test/ directories)
```bash
go run -tags dev .
# OR
go build -tags dev -o nixos-immich-webui .
```

Paths used in dev mode:
- NixOS config: `test/nixos/`
- Immich config: `test/tank/immich-config/`
- Immich data: `test/tank/immich/`

#### Production Build (uses system paths)
```bash
go run .
# OR
go build -o nixos-immich-webui .
```

Paths used in production mode:
- NixOS config: `/etc/nixos/`
- Immich config: `/tank/immich-config/`
- Immich data: `/tank/immich/`

### Runtime Flags
```bash
./nixos-immich-webui --debug    # Enable debug logging
```

Server starts at http://localhost:8000 in both modes.

### Environment Setup
The application expects:
1. NixOS system with ZFS pool named "tank"
2. Binary placed in `/root/`
3. Immich docker-compose setup in `/tank/immich-config/`
4. Tank datasets: `tank/pgdata` and `tank/immich`
5. nixconfig.json in `/etc/nixos/` (see example in `example/etc/nixos/`)

## Key Components

### Configuration Management
- **config package**: Centralized configuration management with JSON-based approach
  - **ConfigVariables struct**: Defines all modifiable NixOS settings in JSON format
  - **ImmichConfig struct**: Manages Immich-specific configuration including ML models
  - **JSON processing**: Uses standard JSON marshaling/unmarshaling with `builtins.fromJSON`
  - **File operations**: Atomic config file switching with `.old` backups using os.Rename
  - **Concurrency safety**: Package-level mutexes (`nixConfigMu`, `immichConfigMu`)
  - **Input validation**: Comprehensive validation for timezone, time, email, Tailscale keys
  - **Automatic rollback**: RollbackConfigJSON() restores .old backup on nixos-rebuild failure
  - **ML model management**: Centralized ValidMLModels map with validation helpers
  - **Build-specific paths**: Separate dev/prod paths using Go build tags
- **handlers package**: HTTP endpoint handling with clean separation of concerns
- **services package**: Business logic services for complex operations
- **system package**: Low-level system command operations

### Web Interface Routes

#### SystemHandler Routes
```go
GET  /{$}           # Main admin panel (HandleRoot)
POST /save          # Save configuration (HandleSave)
POST /apply         # Apply NixOS configuration (HandleApply)
POST /poweroff      # System poweroff (HandlePoweroff)
POST /reboot        # System reboot (HandleReboot)
```

#### ImmichHandler Routes
```go
GET  /status        # Immich service status (HandleStatus)
POST /start         # Start Immich service (HandleStart)
POST /stop          # Stop Immich service (HandleStop)
POST /update        # Update Immich containers (HandleUpdate)
POST /email         # Configure email settings (HandleEmailPost)
POST /mlmodel       # Configure ML model selection (HandleMLModelPost)
```

#### BackupHandler Routes
```go
GET  /disks         # List eligible USB disks (HandleGetDisks)
POST /backup        # Start USB backup (HandleBackup)
GET  /backupstatus  # Backup operation status (HandleGetBackupStatus)
```

### Package Architecture

#### config package
- **Configuration management**:
  - `LoadCurrentConfigJSON()` - Thread-safe JSON config loading
  - `SaveConfigJSON()` - Save config to .tmp file
  - `SwitchConfigJSON()` - Atomic switch with .old backup (uses os.Rename)
  - `RollbackConfigJSON()` - Restore from .old backup on failure
  - `GetImmichConfig()` - Thread-safe Immich config reading
  - `SetImmichEmail()` - Update Immich email configuration
  - `SetMLModel()` - Update ML model selection
- **Data structures**: `ConfigVariables`, `ImmichConfig` structs
- **Validation functions** (validation.go):
  - `ValidateTimezone()` - IANA timezone validation
  - `ValidateTimeFormat()` - HH:MM format validation (00:00-23:59)
  - `ValidateTailscaleAuthKey()` - tskey- prefix and length validation
  - `ValidateEmail()` - Regex-based email validation
- **ML Model support**:
  - `ValidMLModels` map - Single source of truth for allowed models
  - `IsValidMLModel()` - Validation helper
  - `GetMLModelDisplayName()` - Display name helper
- **Concurrency safety**:
  - `nixConfigMu` - Protects nixconfig.json operations
  - `immichConfigMu` - Protects immich-config.json operations
- **Build-specific paths**:
  - `paths_dev.go` - Development paths (test/ directories)
  - `paths_prod.go` - Production paths (/etc/nixos/, /tank/)
- **Utility functions**: `ParseBool()`, `GetLowerUpper()`, `CopyFile()`

#### handlers package
- **SystemHandler**:
  - Configuration save/apply with validation
  - Automatic rollback on nixos-rebuild failure
  - System power management
  - Uses modular template fragments for HTMX responses
- **ImmichHandler**:
  - Service status, start/stop/update
  - Email configuration with validation
  - ML model selection with centralized validation
  - Uses template files instead of inline HTML
- **BackupHandler**:
  - USB backup operations
  - Disk management
  - Uses template fragments for dynamic content

#### services package
- **BackupService**: Business logic for backup operations

#### system package
- **NixOS management**: `ApplyChanges()` (nixos-rebuild switch), `RollbackConfigJSON()`
- **Docker management**: `ImmichService()`, `UpdateImmichContainer()`
- **System operations**: `PowerOff()`, `Reboot()`, `GetStatus()`
- **Backup operations**: `GetEligibleDisks()`
- **Note**: `SwitchConfigJSON()` moved to config package for better encapsulation

## Development Workflow

- **Architecture**: Modular package structure with clean separation of concerns
- **main.go**: Slim 49-line initialization file handling only routing and service initialization
- **internal packages**: 1041 total lines across specialized modules
- **Configuration**: JSON-based with `builtins.fromJSON` pattern across modular .nix files
- **Testing**: Manual testing via web UI and test configs; unit tests planned
- **Logging**: Structured logging with debug/info/error levels

### Frontend Development Philosophy
- Progressive enhancement: Build functional HTML forms first, then enhance with HTMX
- All core features must work without JavaScript
- HTMX attributes for AJAX, partial updates, modals, and confirmations
- Test with and without JavaScript enabled

### Common Development Tasks
- **Add config options**: Update ConfigVariables struct in `config/types.go`, modify handlers in `handlers/` package, update web forms and templates
- **Add routes**: Create handler methods in appropriate handler files, register in `main.go` routing
- **Add business logic**: Implement in `services/` package, consume from handlers
- **Add system operations**: Implement in `system/commands.go`, call from services or handlers
- **Add HTMX features**: Build HTML first in templates, add HTMX, test fallback
- **Testing**: Build, run, test via UI and test configs, verify progressive enhancement
- **Update NixOS modules**: Add new JSON fields to relevant .nix files using `vars.section.setting` pattern


## Important Constants and Paths

### config package constants

Path constants are now defined in separate files based on build tags:

#### Development Build (`-tags dev`)
From `internal/config/paths_dev.go`:
```go
const (
    NixDir     = "test/nixos/"              // NixOS configuration directory
    ImmichDir  = "test/tank/immich-config/" // Immich docker-compose directory
    TankImmich = "test/tank/immich/"        // Immich config JSON location
)
```

#### Production Build (default)
From `internal/config/paths_prod.go`:
```go
const (
    NixDir     = "/etc/nixos/"          // NixOS configuration directory
    ImmichDir  = "/tank/immich-config/" // Immich docker-compose directory
    TankImmich = "/tank/immich/"        // Immich config JSON location
)
```

#### Shared Constants
From `internal/config/parser.go`:
```go
const ConfigFile = "nixconfig.json"  // JSON configuration file name
```

### Key file locations
- **NixOS Configuration**: `nixconfig.json` (JSON-based, replaces template approach)
- **Rollback Backups**: `nixconfig.json.old`, `immich-config.json.old`
- **Temporary Files**: `nixconfig.json.tmp`, `immich-config.json.tmp`
- **NixOS modules**: Modular `.nix` files using `builtins.fromJSON` to read nixconfig.json

## Security Considerations

### Current Security Model
- **Local access only**: Server binds to `localhost:8000`
- **Reverse proxy**: Caddy provides external access at `:8080`
- **No authentication**: Currently no auth on admin interface
- **File permissions**: Runs as root for system management

### Planned Security Enhancements
- Caddy basic auth for admin panel
- OIDC integration for Cloudflare tunnel access
- Tailscale-only admin access option
- Config validation and sanitization

## Backup System

### USB Backup Features
- **Eligibility**: USB drives with exFAT partitions
- **Content**: Photos, system configs, database dumps, compose files
- **Process**: Mount → Backup → Unmount automatically
- **Format**: Configs zipped, photos synced with rsync

### Backup Contents
1. Latest Immich database dump
2. Current `immich-config.json`
3. NixOS configuration directory
4. Docker compose files
5. Full photo library (rsync with --delete)

## Common Patterns and Conventions

### Error Handling
```go
if err != nil {
    slog.Error("| Error description |", "err", err)
    http.Error(w, "User-friendly message", http.StatusInternalServerError)
    return
}
```

### Logging Pattern
```go
slog.Info("| Action description |", "key", value)
slog.Debug("functionName()", "param", paramValue)
slog.Error("| Error description |", "err", err)
```

### JSON Configuration Handling
```go
// Reading current configuration (config package)
func LoadCurrentConfigJSON() (*ConfigVariables, error) {
    configPath := NixDir + ConfigFile
    data, err := os.ReadFile(configPath)
    if err != nil {
        return nil, err
    }
    var config ConfigVariables
    return &config, json.Unmarshal(data, &config)
}

// Saving configuration (config package)
func SaveConfigJSON(cfg *ConfigVariables) error {
    data, err := json.MarshalIndent(cfg, "", "  ")
    if err != nil {
        return err
    }
    tmpPath := NixDir + ConfigFile + ".tmp"
    return os.WriteFile(tmpPath, data, 0644)
}

// Handler usage (handlers package)
func (h *SystemHandler) HandleSave(w http.ResponseWriter, r *http.Request) {
    cfgJSON := &config.ConfigVariables{}
    // ... populate from form data ...
    config.SaveConfigJSON(cfgJSON)
    system.SwitchConfigJSON()  // Backup and apply
}
```

### HTMX Response Patterns
```go
// For HTMX partial updates, return HTML fragments
htmlStr := `<div>Updated content</div>`
tmpl, _ := htmltemplate.New("t").Parse(htmlStr)
tmpl.Execute(w, data)

// For traditional form submissions, return full pages or redirects
http.Redirect(w, r, "/", http.StatusSeeOther)
```

## Future Development Plans

### Core System
- ✅ Auto-rollback if `nixos-rebuild` fails (completed in alpha.3)
- ✅ JSON configuration management (completed in alpha.3)
- ✅ Modular package architecture (completed in alpha.3)
- ✅ Input validation framework (completed in alpha.3)
- Add unit tests
- Internal backup failsafe (backup server config to data disk, photos to boot disk)
- Configuration sanitization (validation complete, sanitization pending)

### Frontend & UI
- Add HTMX and CSS libraries locally (not CDN)
- Email notifications and admin password reset
- Caddy basic auth

### Container & Infrastructure
- Test Podman as alternative to Docker

### Remote Access
- Tailscale start/stop/sign-out/serve integration
- Cloudflare Tunnel integration (OIDC, docs)
- Pangolin integration (basic, docs for self-hosted VPS)

### Immich Features
- ✅ ML model selection (3 models supported - completed in alpha.3)
- ✅ Email notification configuration (completed in alpha.2)
- Advanced Immich API integration

### Medium/Long-term
- Mobile-first CSS/UI
- Responsive UI with HTMX modals, progressive enhancement
- Host system update button
- GitHub binary releases
- Full Immich API integration
- Advanced backup scheduling with HTMX progress tracking
- Multiple remote access methods
- Setup/installation automation

## HTMX Development Guidelines

### Progressive Enhancement Checklist
- [ ] Base functionality works without JavaScript
- [ ] HTMX enhances UX without breaking core features
- [ ] Server endpoints handle both traditional and HTMX requests
- [ ] Error states are handled gracefully in both modes
- [ ] Form validation works on server-side first, enhanced with client-side

### HTMX Best Practices for This Project
1. **Always provide fallback**: Every HTMX-enhanced element should work without it
2. **Server-side rendering**: Return appropriate HTML for both traditional and HTMX requests
3. **Meaningful URLs**: All actions should have corresponding POST/GET endpoints
4. **Status indicators**: Use `hx-indicator` for long-running operations like backups
5. **Graceful degradation**: Test all functionality with JavaScript disabled

## Troubleshooting

### Common Issues
1. **Templates not found**: Ensure `//go:embed` directive is correct
2. **Permission denied**: Application needs root access for system management
3. **Service failures**: Check systemd service status and logs
4. **Build failures**: Verify Go 1.23+ and clean module cache
5. **HTMX not enhancing**: Check JavaScript console and HTMX attributes

### Debug Mode
Uncomment in `main()` to enable verbose logging:
```go
slog.SetLogLoggerLevel(slog.LevelDebug)
```

### File Paths
Development mode uses `test/` directories to prevent system modification during development.

## Testing

- Manual testing via web UI and test configurations
- Unit tests not yet implemented (planned for next alpha)
- Verify backup functionality with test USB drives
- Test all web routes and form submissions
- Always test with JavaScript disabled to ensure progressive enhancement

## Environment Setup

- Requires NixOS system with ZFS pool named "tank"
- Separate boot and storage drives recommended (SSD for storage)
- Manual setup: install NixOS, configure ZFS, create datasets, place files/binary in correct folders
- Immich docker-compose setup in `/tank/immich-config/`
- Tank datasets: `tank/pgdata` and `tank/immich`
- See `/docs/setup/environment.md` and `/docs/setup/storage.md` for details

## Security Considerations

### Current Security Model
- Local access only: server binds to `localhost:8000`
- Reverse proxy: Caddy provides external access at `:8080`
- No authentication: admin interface currently unauthenticated
- File permissions: runs as root for system management

### Planned Security Enhancements
- Caddy basic auth for admin panel
- Email notification and password reset for admin
- OIDC integration for Cloudflare tunnel access
- Tailscale-only admin access option
- Config validation and sanitization

---

*This documentation reflects the current state of the project as of the latest commit. The project is in active alpha development with frequent changes expected. HTMX integration follows progressive enhancement principles to ensure robust functionality across all client capabilities.*

- The example directory at the root of the project includes example configuration files that are stored on the system and are necessary for understanding how the whole server operates (nixOS and Immich servers) and thus what the program needs to do to manage them. The folder structure mimics that of the system (etc/nixos -> /etc/nixos and tank/immich-config -> zfs dataset tank/immich-config)
- Don't build and then run go binaries, just run them directly using `go run`
