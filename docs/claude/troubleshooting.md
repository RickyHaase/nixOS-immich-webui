# Troubleshooting — NixOS Immich WebUI

This document helps you triage and resolve the most common problems you may encounter while developing, running, or operating `nixOS-immich-webui`. It assumes you have access to the host where the binary runs and basic familiarity with Go and NixOS administration. If you need more help, include the information listed in "What to include in a bug report" when you ask.

---

## Quick triage checklist (read this first)
1. Are you running the binary in the correct mode?
   - Dev mode: uses `test/` folders (safe). Production: uses system paths (`/etc/nixos`, `/tank`).
2. Is the binary running and reachable on `localhost:8000`? Check processes and logs.
3. Look at the binary logs (stdout/stderr or systemd journal). Enable `--debug` to get more detail.
4. Check `backup-history.json` for backup errors (path depends on build mode).
5. If a UI interaction doesn’t behave as expected, test the same action with JavaScript disabled to confirm progressive-enhancement fallback.

---

## How to run for debugging

- Run in development mode (uses `test/` directories):
```/dev/null/commands.md#L1-5
go run -tags dev .
```

- Build and run a local binary (production-like):
```/dev/null/commands.md#L6-10
go build -o nixos-immich-webui .
./nixos-immich-webui --debug
```

- Enable verbose debug output from the application (if available):
```/dev/null/commands.md#L11-14
./nixos-immich-webui --debug
# or set environment variables your runtime honors (see main.go)
```

---

## Collect useful logs & artifacts
When you report an issue, gather these artifacts first — they answer ~90% of problems:

- Application stdout/stderr or systemd journal for the service:
```/dev/null/commands.md#L15-18
# If you started the binary directly:
journalctl --since "10 minutes ago" --no-pager -u nixos-immich-webui.service
# Or check the shell output where you ran the binary.
```

- Backup history file (production path depends on build tag; default near binary):
  - Path examples:
    - `test/backup-history.json` (dev)
    - `./backup-history.json` (production)
- Last successful/failed `nixos-rebuild` output (if applicable).
- `internal/templates/web/*.html` fragment(s) involved in the UI flow.
- Browser console output and network trace for UI issues (HTMX requests).

---

## Common issues & how to fix them

### 1) "Templates not found" or blank pages
Symptoms: UI returns an error page or fragment doesn't render.

Likely causes:
- `//go:embed` not including expected template path, or you ran `go build` from the wrong working directory.
- Template file path changed but embed logic not updated.

Checks and fixes:
- Confirm templates exist at `internal/templates/web/`.
- Build from the project root so `embed` sees the files (or re-run `go generate` if applicable).
- Confirm embed file (`internal/templates/embed.go`) contains the correct paths and rebuild.

Relevant files:
- `internal/templates/embed.go`
- `internal/templates/web/*`

### 2) Permission denied (file writes, nixos-rebuild, mounts)
Symptoms: `permission denied` when writing `nixconfig.json`, creating backup files, mounting disks, running `nixos-rebuild`.

Cause:
- The application requires elevated privileges for system operations (mount, apply, power) in production.

Checks and fixes:
- Verify file ownership & modes:
```/dev/null/commands.md#L19-22
ls -l /etc/nixos/nixconfig.json
ls -l /tank/immich-config/
```
- Run operations as root or use the privilege separation pattern (run the main web process as unprivileged user and use a restricted helper/systemd unit for privileged commands).
- Ensure the backup destination disk is writeable and not mounted read-only.

### 3) Backup fails to mount or disk not eligible
Symptoms: Backup state goes to `error` quickly; history entry shows mount failure.

Checks and fixes:
- Verify disk presence and partition type (`exFAT` expected):
```/dev/null/commands.md#L23-27
lsblk -f
blkid
udisksctl monitor
```
- Attempt manual mount to replicate the failure.
- Check that `udisksctl` (or configured mount helper) is installed and works for automatic mounting.
- Ensure the disk has sufficient free space.

Look at `backup-history.json` and application logs for the specific mount error message.

### 4) Rsync progress or percent parsing incorrect
Symptoms: UI shows no progress, or progress stuck/incorrect.

Cause:
- The service parses `rsync --info=progress2` output. If rsync version or locale changes output formatting, parsing can break.

Checks and fixes:
- Reproduce the rsync command used by the service manually:
```/dev/null/commands.md#L28-32
rsync -a --info=progress2 --delete /tank/immich/library /path/to/test-dest/
```
- Inspect stdout format and compare against the parser (see `internal/services/backup.go`).
- Add unit tests around the parsing logic if you adjust regexes.

### 5) HTMX not enhancing (UX differences between JS/no-JS)
Symptoms: HTMX attributes appear in templates but client-side behaviour not active.

Checks and fixes:
- Confirm the HTMX library is loaded in the page (if using a CDN or local copy).
- Inspect browser console for script errors.
- Verify the element that should be enhanced has the `hx-*` attributes in the rendered HTML (server must return fragments containing those attributes after `POST /backup` etc.).
- If the server returns full pages on HTMX requests (missing `HX-Request` branching), update the handler to detect `HX-Request` and return the fragment.

Relevant files:
- `internal/templates/web/backup_status.html`
- `internal/handlers/backup.go`

### 6) Immich containers not starting / docker-compose issues
Symptoms: Immich service status shows stopped, start fails.

Checks and fixes:
- Check docker / docker-compose logs in the Immich config folder:
```/dev/null/commands.md#L33-36
cd /tank/immich-config/
docker-compose ps
docker-compose logs -f
```
- Ensure `ImmichDir` path is correct for your build mode.
- Confirm images are pulled and up to date.

### 7) nixos-rebuild or apply fails, rollback triggered
Symptoms: `ApplyChanges()` fails and rollback occurs.

Checks and fixes:
- Inspect `nixos-rebuild` output captured by the application logs.
- Reproduce the rebuild manually on the host to see full stderr:
```/dev/null/commands.md#L37-40
sudo nixos-rebuild switch -I nixos-config=/etc/nixos/configuration.nix
```
- Validate JSON config (`nixconfig.json`) fields and ensure `builtins.fromJSON` usage is correct in modular `.nix` files.

---

## Debugging tips for handlers & services

- Narrow the problem: create minimal requests (curl or a small test client) that hit the handler with the same form data.
- Add temporary debug logging in the handler to inspect input parsing and validation decisions.
- Keep handlers thin; most logic should be in `internal/services/` so you can test logic directly with unit tests.
- When troubleshooting HTMX flows, capture the request/response pair (network tab) to ensure the server returns exactly the fragment expected by the client.

---

## How to reproduce reliably (recommended workflow)
1. Switch to `-tags dev` and use `test/` fixture data.
2. Run `go run -tags dev .` and exercise the UI or handlers with the same inputs you saw in production.
3. For backups, create a small test dataset under `test/tank/immich/` and run backup flows so you can iterate quickly without affecting production disks.

---

## What to include in a bug report
When filing an issue, include:
- The minimal steps to reproduce.
- Which mode you ran in: `-tags dev` or production.
- Exact command you used to start the app.
- Application logs (last 200–500 lines).
- Relevant files: `backup-history.json`, template fragments used, and the handler name (e.g., `handlers/backup.go`).
- Browser console logs for UI issues.
- Git branch/commit or binary version.

Suggested template:
- OS / NixOS version:
- Binary/branch/commit:
- Run mode: dev / prod
- Steps to reproduce:
- Expected behavior:
- Actual behavior:
- Attached logs & files: (paste relevant excerpts)

---

## Helpful file locations (quick reference)
- `internal/handlers/` — HTTP handlers and HTMX handling
- `internal/services/` — business logic (backup orchestration)
- `internal/templates/web/` — UI templates and fragments
- `internal/config/paths_dev.go`, `internal/config/paths_prod.go` — path constants
- `example/etc/nixos/` — sample configurations
- `docs/claude/` — more in-depth docs (architecture, backups, htmx, security)

---

If you'd like, I can produce a short checklist or a minimal PR that converts a single handler/template pair into a more debuggable structure (for example: add clearer logging, unit tests for rsync parsing, and a small reproduction script). Tell me which specific issue you want prioritized and I'll provide the next steps.