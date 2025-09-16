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

### Current Status
- **Version**: Alpha development (v0.1.0-alpha.2 completed)
- **Stage**: Active development, not production-ready
- **Main Branch**: `main`
- **Current Branch**: `claude-vibe`

## Project Structure

```
nixOS-immich-webui/
├── main.go                          # Single-file Go application (monolithic)
├── go.mod                          # Go module dependencies
├── internal/templates/             # Embedded templates
│   ├── nixos/configuration.nix     # NixOS config template
│   └── web/                       # HTML templates
│       ├── index.html             # Main admin interface
│       └── save.html              # Configuration confirmation page
├── docs/                          # Documentation
│   ├── dev/                       # Development docs
│   │   ├── todo.md                # TODO items
│   │   ├── features.md            # Feature roadmap
│   │   ├── environment.md         # Environment assumptions
│   │   ├── backups.md             # Backup functionality docs
│   │   └── considerations.md      # Development considerations
│   └── setup/                     # Setup documentation
│       ├── system.md              # System setup
│       ├── storage.md             # Storage configuration
│       └── remote-access.md       # Remote access setup
└── test/                          # Test configurations
    └── nixos/configuration.nix    # Test NixOS config
```

## Technology Stack

### Backend
- **Language**: Go 1.23.3
- **HTTP Server**: Standard library `net/http`
- **Templating**: `html/template` and `text/template`
- **File Embedding**: `embed` package for templates
- **Logging**: `log/slog` for structured logging

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

### Building
```bash
go build -o nixos-immich-webui .
```

### Running (Development)
```bash
./nixos-immich-webui
# Server starts at http://localhost:8000
```

### Environment Setup
The application expects:
1. NixOS system with ZFS pool named "tank"
2. Binary placed in `/root/`
3. Immich docker-compose setup in `/root/immich-app/`
4. Tank datasets: `tank/pgdata` and `tank/immich`

### Development Mode
- File paths are currently set to `test/` directory for safety
- Templates are embedded in binary using `//go:embed`
- Debug logging can be enabled by uncommenting `slog.SetLogLoggerLevel(slog.LevelDebug)`

## Key Components

### Configuration Management
- **NixConfig struct**: Defines all modifiable NixOS settings
- **ImmichConfig struct**: Manages Immich-specific configuration
- **Template processing**: Uses Go templates with embedded files
- **File operations**: Safe config file switching with backups

### Web Interface Routes
```go
GET  /{$}           # Main admin panel
POST /save          # Save configuration
POST /apply         # Apply NixOS configuration
GET  /status        # Immich service status
POST /start         # Start Immich service
POST /stop          # Stop Immich service
POST /update        # Update Immich containers
POST /email         # Configure email settings
GET  /disks         # List eligible USB disks
POST /backup        # Start USB backup
POST /poweroff      # System poweroff
POST /reboot        # System reboot
```

### System Integration Functions
- **NixOS management**: `switchConfig()`, `applyChanges()`
- **Docker management**: `immichService()`, `updateImmichContainer()`
- **Backup operations**: `backupToUSB()`, `getEligibleDisks()`
- **File operations**: `CopyFile()`, configuration parsing functions

## Development Workflow

### Frontend Development Philosophy
When adding new UI features, follow the progressive enhancement pattern:

1. **HTML First**: Create fully functional forms and links
2. **Add HTMX**: Enhance with `hx-*` attributes for better UX
3. **Fallback Testing**: Ensure functionality works with JavaScript disabled
4. **Error Handling**: Provide meaningful feedback for both modes

### Common Development Tasks

#### 1. Adding New Configuration Options
1. Update `NixConfig` or `ImmichConfig` struct in `main.go:27-44`
2. Add parsing logic in `loadCurrentConfig()` function
3. Update NixOS template at `internal/templates/nixos/configuration.nix`
4. Modify web form in `internal/templates/web/index.html`
5. Update form handling in `handleSave()` function

#### 2. Adding New Web Routes
1. Define handler function following pattern: `handleName(w http.ResponseWriter, r *http.Request)`
2. Add route to mux in `main()` function at `main.go:990-1004`
3. Add any necessary HTML templates
4. Update frontend with progressive enhancement (HTMX)

#### 3. Adding HTMX-Enhanced Features
1. Build functional HTML form/interface first
2. Add HTMX attributes for enhancement:
   - `hx-get`, `hx-post` for AJAX requests
   - `hx-target` for partial page updates
   - `hx-trigger` for event handling
   - `hx-confirm` for user confirmations
3. Test both with and without JavaScript enabled
4. Ensure server responses work for both traditional and HTMX requests

#### 4. Testing Changes
```bash
# Build and test
go build -o nixos-immich-webui .
./nixos-immich-webui

# Check functionality at http://localhost:8000
# Test with NixOS test configuration in test/ directory
# Test with JavaScript disabled for progressive enhancement
```

### Code Organization Notes
- **Current**: Single monolithic `main.go` file (1013 lines)
- **Planned**: Refactor into separate modules/packages
- **Templates**: Embedded at compile time, parsed at runtime
- **Logging**: Structured logging with debug/info/error levels

## Important Constants and Paths

```go
const nixDir string = "test/nixos/"          # Development: test/, Production: "/etc/nixos/"
const immichDir string = "/root/immich-app/" # Immich docker-compose location
const tankImmich string = "test/tank/immich/" # Immich config JSON location
```

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

### Template Execution
```go
tmpl, err := htmltemplate.ParseFS(templates, "internal/templates/web/file.html")
if err != nil {
    slog.Error("| Error rendering template |", "err", err)
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
}
tmpl.Execute(w, data)
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

### Near-term (v0.1.0-alpha.3)
- Refactor monolithic `main.go` into separate modules
- Add proper documentation and setup guides
- Improve error handling and rollback mechanisms

### Medium-term (v0.1.0-beta.1)
- **Mobile-first CSS implementation**
- **Enhanced HTMX integration** for:
  - Real-time backup progress indicators
  - Modal dialogs for confirmations
  - Live system status updates
  - Progressive form validation
- **Ensure progressive enhancement** throughout the interface
- Host system update functionality
- GitHub release automation

### Long-term Features
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

## Contributing Guidelines

### Code Style
- Follow Go standard formatting
- Use structured logging with `slog`
- Include error handling for all operations
- Add debug logging for function entry/exit
- Use descriptive variable names

### Frontend Guidelines
- **Progressive enhancement first**: Build without JavaScript, enhance with HTMX
- **Semantic HTML**: Use proper form elements and semantic markup
- **Accessibility**: Ensure keyboard navigation and screen reader compatibility
- **Mobile-first**: Design for mobile devices first, enhance for larger screens

### Testing
- Test against NixOS test configuration
- Verify backup functionality with test USB drives
- Check all web routes and form submissions
- **Test without JavaScript** to ensure progressive enhancement
- Validate NixOS configuration syntax

### Documentation
- Update this CLAUDE.md file for significant changes
- Document new environment requirements
- Add setup instructions for new features
- Update roadmap and todo items
- Document HTMX enhancement patterns

---

*This documentation reflects the current state of the project as of the latest commit. The project is in active alpha development with frequent changes expected. HTMX integration follows progressive enhancement principles to ensure robust functionality across all client capabilities.*