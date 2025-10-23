# Security Guide — NixOS Immich WebUI

This document summarizes the current security posture, immediate mitigations, recommended hardening steps, and operational practices for deploying and maintaining the `nixOS-immich-webui` service. It is intended for operators and contributors who will deploy or change the system.

This project context (short)
- The server binary binds to `localhost:8000` by default and is commonly fronted by `Caddy` (reverse proxy) for external access.
- It performs privileged operations on the host (apply NixOS configurations, mount/unmount disks, perform DB dumps, control containers, poweroff/reboot).
- In current alpha state the admin UI is unauthenticated; the service typically runs as root to perform system-level operations.
- Frontend uses server-side HTML rendering and HTMX for progressive enhancement.

High-level security goals
- Minimize attack surface exposed to networks.
- Enforce strong authentication and authorization for admin operations.
- Protect secrets and configuration files from accidental disclosure.
- Limit privilege where possible and audit privileged actions.
- Ensure backups and sensitive data are stored and moved securely.

Immediate mitigations (high-impact, low-effort)
1. Keep the service bound to loopback by default.
   - Do not expose `:8000` to the network. Put a reverse proxy in front (e.g., `Caddy`) and only proxy required routes.
2. Add a basic auth gate in the reverse proxy for the admin UI.
   - Use Caddy’s `basicauth` or an OIDC plugin; generate a hashed password via `caddy hash-password`.
3. Run in development mode (`-tags dev`) only on non-production machines.
4. Limit access to the machine at the network layer (firewall rules, SSH port restrictions).
5. Ensure the binary and config files have strict filesystem permissions:
   - Owner: `root`, Mode: `0600` for secrets and `0640` for config.
6. Add structured logging and monitor logs (success/fail events for `nixos-rebuild`, backups, mounts).

Authentication & access control
- Short-term: Protect the UI with reverse-proxy auth (basic auth or OIDC). This is faster to roll out.
- Medium-term: Implement strong authentication inside the app (or via proxy):
  - OIDC integration (Cloudflare Tunnel, Auth0, Keycloak) or Tailscale ACLs for admin access.
  - Support admin users and role-based access for potentially multi-admin environments.
- Principle of least privilege:
  - Limit what the web service account (if not root) can execute via a minimal set of allowed sudoers commands or systemd units.
  - Consider splitting privileged helpers into separate, audited binaries or systemd units that expose well-defined, minimal interfaces.

Reverse proxy / TLS
- Always front the app with a TLS-terminating reverse proxy in production. Caddy is a good default:
  - It provides automatic TLS via Let’s Encrypt and supports authentication modules.
- Example Caddyfile (illustrative — hash passwords with `caddy hash-password` first):
  ```
  example.com {
      basicauth /admin* {
          admin JDJhJDE0J...
      }
      reverse_proxy localhost:8000
  }
  ```
- Ensure the proxy only forwards the necessary headers and that `X-Forwarded-For` is handled correctly.
- Use HSTS and modern TLS ciphers (Caddy defaults are good).

Transport & network-level protections
- Use local-only binding for the binary; rely on the reverse proxy for public exposure.
- If remote admin access is required, prefer VPN/tunnel (Tailscale) or Cloudflare Tunnel + OIDC — avoid exposing the admin UI directly to the open internet.
- Configure firewall rules (nftables/iptables) to restrict inbound connections to necessary ports.

Secrets management
- Do not embed plain-text secrets (TLS keys, API tokens, Tailscale keys) into repo or templates.
- Recommended approaches:
  - Use NixOS secrets management, systemd `EnvironmentFile`, or a secrets manager.
  - For deployment on NixOS: use declarative secrets with restricted file permissions and populate at runtime.
  - When using Caddy, store hashed passwords and use secret stores where supported.
- Rotate keys and credentials regularly and revoke ones that are leaked or unused.

Privilege separation and runtime user
- Running the whole application as root increases blast radius. Options:
  - Keep the main HTTP server running as an unprivileged user and run privileged operations via:
    - Restricted `sudo` rules (very narrow commands), or
    - Systemd services that run with higher privileges and accept requests from the web service via socket or well-defined files, or
    - A small, audited privileged helper executable with a minimal API surface (setuid is risky; prefer systemd helpers).
  - Example pattern:
    - `nixos-immich-webui` runs as `immich-admin` (unprivileged).
    - Privileged actions like `nixos-rebuild` are performed by `systemd` service `nixos-immich-helper@.service` which accepts a validated request (e.g., via a temporary file or socket) and runs under root.
- When privileged helpers exist, audit their inputs and ensure they validate everything (no shell interpolation of user-provided strings).

Input validation & command execution
- Sanitize and validate all inputs coming from HTTP forms:
  - Disk identifiers (whitelist device names discovered from the system).
  - File names and paths (do not allow `..` or arbitrary absolute paths).
  - Numeric limits and flags (checkboxes like verify checksums).
- Avoid passing user input into shell commands directly. Use `exec.Command` with argument arrays and do not use `sh -c` unless input is fully validated and escaped.
- Use strong validation functions already in the codebase (timezone, email, Tailscale keys), and expand coverage for disk/device validation.

Cross-Site concerns (XSS, CSRF)
- XSS:
  - Use Go’s `html/template` for all HTML rendering — it auto-escapes variables. Audit template fragments for unsafe `html/template` usage (e.g., `template.HTML`).
  - Sanitize any user-provided text that ends up rendered as HTML (e.g., backup labels).
- CSRF:
  - Protect state-changing endpoints with CSRF tokens. Insert a server-generated token into every HTML form and validate it on POST.
  - Alternatively, require same-site cookies and/or ensure authenticated endpoints are not callable without token checks.
- Cookie settings:
  - Use `Secure`, `HttpOnly`, and `SameSite=Strict`/`Lax` as appropriate.
- Content Security Policy (CSP):
  - Add a strict CSP header to reduce the impact of XSS (disallow inline scripts/styles unless necessary and hashed).

HTMX-specific considerations
- HTMX requests are XHRs; treat them like other AJAX traffic:
  - Ensure CSRF token is present in HTMX requests (place token in forms or send with `hx-headers`/`hx-vals`).
  - Validate `HX-Request` header server-side only for response shaping — do not use it as a security boundary.

File system & backup security
- Backups may contain sensitive config files and DB dumps.
  - Encrypt backups at rest on external media (GPG or LUKS) or require physical handling policies.
  - Avoid writing unencrypted DB dumps to world-readable locations.
- When using exFAT on USB:
  - Understand exFAT does not support POSIX permissions — prefer encrypting the backup or using a filesystem that supports ownership/mode if you need to preserve them.
  - Require manual confirmation before writing to removable media and show destination/freespace to user.
- Backup history JSON files:
  - Store them with restrictive permissions (`0600`) and avoid embedding secrets there.

Container and runtime hardening
- When launching or controlling containers:
  - Prefer running containers with least privileges (drop capabilities, read-only filesystems where possible).
  - Avoid bind-mounting host-sensitive paths into containers unless necessary.
  - Consider Podman or Docker rootless modes if feasible.
- Keep third-party images up to date and scan images for vulnerabilities.

Logging, monitoring & alerting
- Structured logging with `slog` is in place — ensure logs include context (request IDs, user, operation).
- Centralize logs and monitor for:
  - Failed login attempts / repeated failed admin actions.
  - Unexpected `nixos-rebuild` failures.
  - Frequent mount/unmount errors or aborted backups.
- Implement alerting for critical failure modes (failed backups, repeated permission errors).

Auditing, testing and CI security
- Add static analysis and security checks to CI:
  - `gosec`, `staticcheck`, `govulncheck`.
  - Dependency vulnerability scanning (e.g., `govulncheck`, `dependabot`).
- Add unit tests for input validation and rsync parsing to reduce logic errors that can be abused.
- Periodically run a security review or pentest on privileged code paths (`internal/system`, backup helper logic).

Operational checklist before production
- [ ] Configure TLS + reverse proxy (Caddy) with authentication.
- [ ] Put firewall rules in place to restrict access.
- [ ] Move sensitive operations into restricted helpers or systemd units.
- [ ] Encrypt backups or the USB media used for backups.
- [ ] Run the service as an unprivileged user where possible.
- [ ] Enable and monitor structured logs and alerts.
- [ ] Apply regular updates to base OS, Go, and runtime components.
- [ ] Verify CSRF protections and XSS sanitization across templates.
- [ ] Audit file and directory permissions (`/etc/nixos`, `/tank/*`, `backup-history.json`).

Incident response and recovery
- Keep a known-good backup of core config files and a tested restore procedure.
- Have a rollback plan for config application failures (already implemented: automatic rollback on rebuild failure).
- Rotate credentials if a compromise is suspected.
- Preserve logs for forensic analysis and consider immutable log storage for critical systems.

References & where to change code
- Authentication / proxy examples: `docs/claude/security.md` (this file) and proxy configs in your deployment artifacts.
- Privileged helpers and system operations: `internal/system/commands.go`
- Config & path constants: `internal/config/paths_prod.go`, `internal/config/paths_dev.go`
- Templates (XSS surface): `internal/templates/web/*.html`
- Backup orchestration: `internal/services/backup.go`, `internal/handlers/backup.go`

Notes and trade-offs
- Running privileged operations via a single root-owned binary is convenient but risky. Prefer privilege separation even if it requires more initial engineering.
- Balancing usability (easy local access) vs. security (strong auth) is important: for single-admin home setups, Tailscale + machine-local binding may be a pragmatic compromise.
- Plan for gradual rollout: start with reverse-proxy auth, add OIDC/Tailscale integration, and then reduce runtime privileges.

If you want, I can:
- Produce a minimal `Caddyfile` example with `basicauth` and TLS that you can drop into `deploy/`.
- Propose a `systemd` helper scaffold to perform `nixos-rebuild` and mount actions with a well-defined IPC mechanism so the main web process can run unprivileged.
- Audit `internal/system` for risky shell usage patterns and propose concrete code-level hardening suggestions.