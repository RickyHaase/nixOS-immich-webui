# Backup Deep Dive

This document is the authoritative deep-dive on the USB backup subsystem implemented in `nixOS-immich-webui`. It covers the architecture, data models, workflow, progress parsing, HTMX polling pattern, eligibility rules, history management, verification options, error handling, tests, and where to find or change relevant code.

Audience: contributors who will modify backup logic, handlers that surface backup status, or operators who want to understand how the backup system behaves.

---

## Goals & Principles

- Reliable, resumable, verifiable backups of:
  - NixOS configuration files and Immich config/database (essential).
  - Photo library (large dataset).
- Non-blocking UI: backups run asynchronously while the web UI remains responsive.
- Observable progress: surface percent + file counts + current file via HTMX-driven polling.
- Safety: prevent concurrent backups, ensure safe unmount, and provide rollback/clear error state.
- Minimal host assumptions: prefer portable commands (rsync, udisksctl) and exFAT for USB compatibility.

---

## Key Data Structures (conceptual)

- `BackupState`
  - `InProgress` (bool) — whether a backup is active
  - `Status` (string) — one of `idle`, `mounting`, `configs`, `library`, `unmounting`, `complete`, `error`
  - `CurrentStep` (string) — human text describing current action
  - `ProgressPercent` (int) — 0..100 overall progress
  - `TotalFiles` (int64) — expected total file count (from rsync)
  - `ProcessedFiles` (int64) — processed files so far
  - `CurrentFile` (string) — last observed file path
  - `StartTime` (time) — when the operation started
  - `ErrorMessage` (string) — non-empty when `Status == "error"`
  - `VerifyChecksum` (bool) — whether `--checksum` was requested

- `BackupHistoryEntry`
  - `Timestamp`
  - `Status` (`success`|`failed`)
  - `DurationSec`
  - `FilesBackedUp`
  - `TotalSizeMB`
  - `DiskUsed` (identifier)
  - `BackupType` (currently `usb`)
  - `ErrorMessage`
  - `VerifiedChecksum` (bool)

The canonical types are defined in `internal/services/backup.go` and persisted to the JSON file referenced by `BackupHistoryFile` (see `internal/config/paths_*`).

---

## High-level Backup Workflow

1. User requests a backup via the UI (`POST /backup`). The handler validates request and disk selection, then asks `BackupService` to start an asynchronous job. If a backup is already running, the handler returns an error message.
2. `BackupService` launches a goroutine and sets `BackupState` to `mounting`.
3. The service mounts the disk (ideally via `udisksctl` or a wrapper); only exFAT partitions are eligible. On failure, set error state and record history entry.
4. Create a destination directory `immich-server-backup/YYYY-MM-DD_HHMMSS/`. Inside create `config/` and `library/`.
5. Config backup phase:
   - Gather essential files: `nixconfig.json`, `immich-config.json`, DB dump (`pg_dump` or compressed export).
   - Zip these into `config/config-<timestamp>.zip` with an `essential/` and `supplemental/` layout.
   - Mark progress 0–10% for this phase.
6. Library backup phase:
   - Use `rsync -a --info=progress2 --delete` to copy `/tank/immich/library` to destination `library/`.
   - If `VerifyChecksum` is true, include `--checksum` (slower).
   - Map rsync progress to global progress range 10–95%.
   - Parse `--info=progress2` output in real time to extract bytes/percent and `to-chk` counts.
7. Unmount disk safely.
8. Update `backup-history.json` and set `Status = complete` or `error`. Allow the UI to poll final status and briefly show a success message.

---

## Rsync Progress Parsing (practical)

Rsync's `--info=progress2` prints lines like:
"123,456,789  45%  123.45MB/s    0:12:34 (xfr#123, to-chk=456/789)"
- Extract the numeric percent (45%).
- Also parse `to-chk=456/789` to compute processed files = total - remaining.

Strategy:
- Read rsync stdout/stderr line-by-line.
- Use regex to find percentage and `to-chk` tokens.
- Maintain two-phase mapping:
  - Configs: 0–10% (deterministic steps)
  - Library: map rsync-provided percentage linearly into 10–95%
  - Finalization/unmount: 95–100%

Edge cases:
- `--checksum` produces much slower output; percent advancement may stall — show "verifying" text and keep file counts fresh.
- If rsync exits with non-zero status, capture stderr and transition to `error`.

---

## HTMX Polling Pattern

Design goals: poll only while backup is active to reduce load.

- Start action response includes an HTMX snippet that triggers polling, e.g. `hx-trigger="load, every 2s"`.
- Poll endpoint: `GET /backupstatus` returns the small HTML fragment or JSON containing `BackupState`.
- When `Status` transitions to `idle` or `complete` or `error`, the returned fragment removes the `hx-trigger`, stopping automatic polling.
- On completion, server can include `hx-trigger="after 3s"` to refresh the UI once more (e.g., to show history).

Implementation notes:
- Handlers must detect HTMX requests and return fragments (partial templates) versus full page responses.

---

## Disk Eligibility & Mounting Rules

A disk is eligible if:
- Device path indicates USB (udev attributes or sysfs check).
- Contains an exFAT partition (cross-platform compatibility).
- Is mountable with `udisksctl` or the configured mount helper.
- Has at least X GB free (optionally configured) to hold the expected backup.

Failure modes:
- If mount fails, present user with a clear error: "Unable to mount disk: <reason>".
- If insufficient space, abort with `Status = error` and record history.

---

## History Management

- File: location determined by `BackupHistoryFile` in `internal/config/paths_*.go`.
- Format: JSON array of `BackupHistoryEntry` objects (pretty-printed).
- Retention: automatically prune to the last 100 entries on save.
- Writes must be atomic:
  - Write to `.tmp`, fsync, then `os.Rename` to final file.
- Reads are tolerant: if file missing, return empty history.

---

## Error Handling & Reporting

- All errors should:
  - Update `BackupState.Status = "error"` and `ErrorMessage`.
  - Log with structured logging `slog.Error` including details and stack where useful.
  - Append a `BackupHistoryEntry` with `Status = failed` and `ErrorMessage`.
- For user-facing messages, prefer short, actionable text (e.g., "Mount failed: permission denied").

---

## Testing & Troubleshooting

Testing recommendations:
- Unit-test parsing logic for rsync lines and mapping to progress.
- Integration tests (manual or CI with loopback mounts) for:
  - Full backup with small sample library.
  - `--checksum` path for correctness and performance.
  - Failure injection: simulate mount failure, rsync error, disk removal.

Troubleshooting quick checklist:
- Check service logs (binary) for structured messages.
- Verify `backup-history.json` for last recorded error details.
- Re-run the same command line used by the service manually (mount, rsync) to reproduce issues.

---

## Configuration & Where to Change Code

- Start point for handlers: `internal/handlers/backup.go`.
- Core orchestration and state: `internal/services/backup.go`.
- Disk discovery/mount helpers: `internal/system/commands.go`.
- Templates for status fragments: `internal/templates/web/backup_status.html` and `backup_dashboard.html`.
- History file path: `internal/config/paths_prod.go` and `paths_dev.go`.

When changing behavior:
- Update tests around rsync parsing.
- Keep UI fragment shape stable to avoid HTMX mismatches.
- Preserve atomic file operations and mutex protections.

---

## Security & Safety Notes

- The backup service performs privileged operations (mount, database dump). Code paths must validate inputs and sanitize disk identifiers.
- Runs as root in production; restrict web access with reverse proxy auth (Caddy) or network-level controls (Tailscale).
- Consider adding a dry-run or simulation mode for safer testing on production hosts.

---

## Quick Operator Commands (examples)

- Manually run a config-only backup flow (for debugging):
  - Mount disk, create folders, tar/zip the config files, unmount — mimic what service does.
- Test rsync parsing with a small copy and `--info=progress2` to inspect stdout formatting.

---

## Checklist Before Merging Changes

- [ ] New code includes unit tests for parsing & mapping.
- [ ] Backup concurrency protections remain intact.
- [ ] Atomic writes for history and temp files preserved.
- [ ] UI fragments updated alongside handler changes.
- [ ] Operation tested in `-tags dev` mode using `test/` directories.

---

This file is a living reference for the backup subsystem. For operational runbooks, refer to `docs/setup/storage.md` and `docs/claude/architecture.md` for broader context.