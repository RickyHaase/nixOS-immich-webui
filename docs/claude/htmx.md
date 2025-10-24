# HTMX Guidelines for nixOS-immich-webui

Purpose
-------
This document codifies HTMX usage and progressive-enhancement patterns for the project. It targets contributors who will:
- Add or modify UI interactions (forms, status panels, modals).
- Implement HTMX-aware handlers in `internal/handlers`.
- Keep the UI functional without JavaScript and improve UX with HTMX.

Principles
----------
- Progressive enhancement first: every interactive element must work as a standard HTML form or link without JavaScript.
- Small fragments: return concise HTML fragments for HTMX requests (not full pages).
- Idempotency and safety: POSTs for state changes, GETs only for safe reads; handlers must validate inputs server-side.
- Poll responsibly: use event-driven polling (start/stop) to avoid unnecessary load.
- Accessibility: keep semantic markup and explicit labels; ensure fallback flows exist.

Detecting HTMX requests (server-side)
-------------------------------------
HTMX sets `HX-Request: true` in the request headers. Use this to decide whether to return:
- A full page (normal request), or
- A fragment (for HTMX partial updates).

Example detection (pseudo-Go):
```/dev/null/htmx_handler_example.go#L1-40
if r.Header.Get("HX-Request") == "true" {
    // render fragment template used by HTMX
} else {
    // render full page template
}
```

Common HTMX attributes used in this project
-------------------------------------------
- `hx-get` / `hx-post`: request method and URL.
- `hx-target`: CSS selector where response content should be inserted.
- `hx-swap`: how to replace content (e.g., `outerHTML`, `innerHTML`, `afterbegin`).
- `hx-trigger`: what triggers request (`click`, `load`, `every 2s`, `after 3s`).
- `hx-indicator`: CSS selector for a spinner during requests.
- `hx-vals`: push small JSON values with the request.
- `hx-include`: include other inputs in the request.
- `hx-confirm`: simple confirmation UI before request.
- `hx-headers`: add custom headers if needed (use sparingly).
- `hx-push-url`: update browser URL in history for navigable changes.

Key HTMX patterns in this repo
------------------------------
1. Progressive-enhanced form (works without JS)
```/dev/null/htmx_form_example.html#L1-24
<form id="email-form" action="/email" method="post">
  <label for="email">Admin email</label>
  <input id="email" name="email" type="email" required>
  <button type="submit" 
          hx-post="/email" 
          hx-target="#email-form" 
          hx-swap="outerHTML"
          hx-indicator="#ind">Save</button>
  <span id="ind" class="spinner" hidden></span>
</form>
```

2. Event-driven backup polling (only poll while active)
```/dev/null/htmx_backup_example.html#L1-28
<!-- Trigger backup and start polling status -->
<form id="backup-form" action="/backup" method="post">
  <select name="disk">...</select>
  <label><input type="checkbox" name="verify"> Verify checksum</label>
  <button type="submit"
          hx-post="/backup"
          hx-target="#backup-panel"
          hx-swap="outerHTML">Start Backup</button>
</form>

<!-- /backup handler responds with a fragment that includes polling -->
<div id="backup-panel" hx-get="/backupstatus" hx-trigger="load, every 2s" hx-swap="outerHTML">
  <!-- live status fragment rendered by server -->
</div>
```
Notes:
- The `/backup` handler returns an HTMX fragment that contains the `hx-get` polling attributes.
- When the server returns a final state (idle/complete/error), the fragment should not include `hx-trigger` so polling stops.

Server response strategies
--------------------------
- For HTMX requests return only the fragment needed to replace `hx-target`.
- For non-HTMX requests, render the full page.
- For state-changing POSTs prefer returning a fragment of the updated state (instead of redirect), except when a full redirect is desired; in that case return `303 See Other`.
- Use `HTTP 204 No Content` sparingly for "no UI change" responses but be careful: HTMX will not swap content on 204.

Templates & fragment shape stability
------------------------------------
- Keep fragment HTML shapes stable. If you change the element IDs or wrapper structure, update both templates and client code (`hx-target` selectors).
- Put HTMX fragments in `internal/templates/web/` and name them clearly (e.g., `backup_status.html`, `email_form.html`).

Loading indicators and UX
-------------------------
- Use `hx-indicator` with a consistent spinner element. The indicator selector should exist in the fragment so it can be swapped.
- Prefer subtle inline indicators over global spinners to avoid jarring page changes.

Security and CSRF
-----------------
- HTMX makes POSTs like any other XHR; CSRF protections still apply.
- Recommended approaches:
  - Include a server-generated CSRF token as a hidden input in each form. Server validates token on POST.
  - For fetches that are not forms, consider adding the token via `hx-vals` or `hx-headers`.
  - Validate `Origin`/`Referer` headers on sensitive endpoints as a defense-in-depth.
- Do not rely solely on custom headers, since some proxies might strip them.

Server-side validation
----------------------
- Always validate and sanitize inputs server-side. HTMX is only a transport — clients can be manipulated.
- For file/disk identifiers (e.g., backup target), whitelist acceptable identifiers on the server.

Polling and rate limits
-----------------------
- Use HTMX `hx-trigger` to control frequency (e.g., `every 2s`) and rely on the server to return fragments that remove triggers when idle.
- Avoid very high polling frequencies for many concurrent clients. If necessary, implement server-side rate-limiting or exponential backoff.
- For large fleets or remote access, consider websocket/eventstream alternatives for statuses, but keep HTMX polling as the primary simple solution.

Error handling and user messaging
--------------------------------
- HTMX partial responses should include user-readable error messages inside the fragment.
- For fatal server errors return a fragment that renders an error block; include a clear next step.
- Log detailed errors server-side with `slog` and return sanitized messages to users.

Testing & progressive-enhancement checks
----------------------------------------
- Test every HTMX flow with JavaScript disabled — verify the fallback full-page flow works identically.
- Test with HTMX enabled and disabled in the browser to ensure parity.
- Unit-test server-side fragment rendering (templates) and handler logic where feasible.
- For polling, simulate long-running operations and verify polling stops on completion.

Best-practice checklist for contributors
---------------------------------------
- [ ] Does the UI element work without JavaScript (form action + method)?
- [ ] Does the handler detect `HX-Request` and return a fragment when appropriate?
- [ ] Is the HTMX target selector stable and specific (`#id` or `.class` on a known wrapper)?
- [ ] Are server-side validations in place for all inputs?
- [ ] Is CSRF protection present on POST endpoints?
- [ ] Is polling event-driven and does the fragment remove the trigger on completion?
- [ ] Are fragments small and focused (avoid returning whole pages)?
- [ ] Is the fragment stored under `internal/templates/web/` and named clearly?

References & files to inspect
----------------------------
- HTMX docs: https://htmx.org/
- Main pages:
  - `internal/templates/web/index.html` (home/backup/system dashboard)
  - `internal/templates/web/config.html` (configuration page)
- Templates used for HTMX fragments:
  - `internal/templates/web/backup_status.html`
  - `internal/templates/web/backup_dashboard.html`
  - `internal/templates/web/email_form.html`
  - `internal/templates/web/ml_form.html`
  - `internal/templates/web/oauth_form.html`
  - `internal/templates/web/save.html`
  - `internal/templates/web/apply_success.html`
- Handlers:
  - `internal/handlers/backup.go`
  - `internal/handlers/immich.go`
  - `internal/handlers/system.go`

Closing notes
-------------
HTMX is a powerful tool for incremental UX improvements. Keep server-side logic authoritative, maintain clear and small fragments, and preserve full non-JS flows. If you want, I can prepare a short example pull request that converts one UI form into an HTMX-enhanced progressive-enhancement flow (handler + fragment + test checklist).