# Development To-Do Items

This document tracks implementation details and candidate features that supplement the main roadmap in README.md. The README roadmap is the source of truth for version priorities and deliverables.

**Purpose**:
- Implementation details for current roadmap items
- Candidate features not yet promoted to main roadmap
- Research tasks and technical investigations

Items marked with checkboxes indicate completion status.

## Completed Items
- [x] Split README into different pages in /docs/dev/ to keep things organized.
- [x] Although the work has been done, only a monolithic Nix config file has been committed to the repo as it was manageable in one template and one page in the UI (so far).
  - ~~Create a set of .nix files that define the desired state of the server, organized and compartmentalized logically.~~
    - ~~configuration.nix with only the imports and default timezone, language, and regional settings configured during install.~~
    - ~~system.nix with the systemd service for the web server.~~
    - ~~Additional .nix files as needed to hold the configs that will be modifiable via the web interface.~~
    - ~~admin.nix for an advanced admin to have a file that won't be touched, allowing them to modify with any additional configs they may want.~~
      - ~~This could be modifiable in an advanced option to enable/disable SSH without overwriting the entire file, just updating the relevant string.~~
- [x] Convert the .nix file into a template.
- [x] Build a web page that contains inputs to modify the necessary parts of the server config.
- [x] Structure the web server to read the existing config corresponding to each webpage on load, save the .tmp file on save, alert when leaving without applying, and copy to .nix and run a rebuild on reload.
## Implementation Details for Current Roadmap

### v0.1.0-alpha.3 Details
- [ ] Rebuild nix config files parsing (depends on finalized config structure)
- [ ] Determine optimal Immich config storage directory structure
- [ ] Update backup functionality for new config file locations
- [ ] Validate documentation accuracy post-refactor

### v0.1.0-beta.1 Details
- [ ] Integrate Immich UI styling for visual consistency (see Frontend & UI section below)
- [ ] Evaluate HTMX modal patterns for responsive enhancement
- [ ] Progressive enhancement testing strategy

## Candidate Features (Not Yet in Roadmap)

### Ready for Roadmap Promotion
- [ ] Add HTMX and CSS libraries into source instead of CDNs
- [ ] Caddy basic auth implementation
- [ ] Auto-rollback if nixos-rebuild fails

### Research/Investigation Needed

#### Core System
- [ ] Auto-rollback implementation strategy
  - Auto-rollback if no web requests accepted within 60s of config apply
  - Optional manual rollback to previous config
  - Integration with systemd service monitoring
- [x] ~~Figure out how to embed templates into the binary~~ (completed in alpha.2)
- [x] ~~Parse templates at initialization instead of runtime~~ (completed with refactor)
- [x] ~~Re-organize code into multiple files~~ (completed in alpha.3)
- [ ] Internal backup failsafe strategy
  - Backup server config to data disk, photos to boot disk
  - Configurable re-encoding based on boot disk capacity
  - Supplimental to existing USB backup system

#### Frontend & UI
- [ ] Integrate Immich UI styling for visual consistency (beta.1 roadmap item)
  - Extract CSS classes and design tokens from @immich/ui Svelte components
  - Create utility CSS classes matching Immich's design system (using Tailwind patterns)
  - Apply styling to existing HTML forms and HTMX elements
  - Maintain accessibility and progressive enhancement (non-JS functionality)
  - Consider extracting: Card components, form styling, button variants, color scheme, typography
- [ ] Email notification system design
  - Admin password reset mechanism
  - System status notifications
  - Backup completion/failure alerts
- [ ] ZFS dataset validation and setup guidance
  - Check for required tank datasets on startup
  - Display setup guide link if missing
  - Graceful degradation when storage not configured

#### Container & Infrastructure
- [ ] Podman compatibility assessment
  - Docker-compose equivalent functionality
  - NixOS integration differences
  - Migration path from Docker
- [ ] Cockpit integration options
  - Read-only server status integration
  - Admin.nix inclusion vs web UI integration
  - Resource usage impact

#### Remote Access
- [ ] Enhanced Tailscale management
  - Start/stop/restart tailscale service
  - Sign out functionality
  - Tailscale serve configuration for Immich
  - Status monitoring and troubleshooting
- [ ] Cloudflare Tunnel integration
  - OIDC setup documentation
  - Split-connection app configuration
  - Zero Trust policy examples
- [ ] Pangolin VPS integration
  - Self-hosted VPS setup documentation
  - Configuration templates
  - Security considerations

### Future Consideration
- [ ] Unit testing framework implementation (moved to alpha.4 roadmap)
- [ ] Advanced backup scheduling with progress tracking
- [ ] Multi-user admin role management
- [ ] API endpoint versioning strategy
- [ ] Immich API integration beyond basic container management

---

**Note**: Items in "Ready for Roadmap Promotion" are candidates for inclusion in the main roadmap. Items in "Research/Investigation Needed" require further planning before roadmap inclusion.
