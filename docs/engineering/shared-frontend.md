# Shared Frontend Capabilities

This document catalogs the shared client-side capabilities, network abstractions, and token conventions in Workbench.

> [!IMPORTANT]
> **Evidence status**: Source inspection and Node mock / Go handler tests cover only the bounded contracts listed below. Real authenticated browser acceptance is **RUNTIME ACCEPTANCE PENDING**; these capabilities are not yet certified. Legacy modules (such as `internal/module/user/` or legacy scripts in `ui.js`) are **not** certified as global golden references. Certification in this document is **capability-level, not file-level**: a capability listed below (for example `window.escapeHtml` in section 3) is certified as the canonical source **for that capability**, without certifying the file that currently provides it.

---

## 1. Network & Fetch Capabilities

Workbench maintains two primary client-side fetch abstractions in `web/static/js`:

### 1.1 `window.appFetch` (`web/static/js/app.js`)

- **Role**: Base shared network request wrapper.
- **Signature**: `appFetch(input, init)` -> `Promise<Response>`
- **Automatic Headers**:
  - `X-Requested-With: XMLHttpRequest`
  - `X-CSRF-Token`: Extracted from `<meta name="csrf-token" content="...">`.
- **Return Contract**: Returns a raw standard `Response` object.

```javascript
// Illustrative placeholder URL; replace using the actual route contract.
window.appFetch("/api/example", {
  method: "POST",
  headers: {
    "Content-Type": "application/json",
    "Accept": "application/json"
  },
  body: JSON.stringify({ key: "value" })
})
  .then(function (resp) {
    if (!resp.ok) {
      // Caller must inspect HTTP status and deserialize error payload
      return resp.json().then(function (errData) {
        throw new Error(errData.error || errData.message || "Request failed with status " + resp.status);
      });
    }
    return resp.json();
  })
  .then(function (data) {
    console.log("Success:", data);
  })
  .catch(function (err) {
    showToast(err.message, "error");
  });
```

> [!CAUTION]
> **Anti-Pattern (FalseReferenceRejected)**: Never assume `appFetch` automatically parses JSON or throws on 4xx/5xx responses. It returns the raw `Promise<Response>`. Callers must explicitly check `resp.ok` and handle serialization.

- **Automated Evidence (not browser acceptance)**:
  - `tests/e2e/csrf-tokens.spec.js` covers scheduleFetch fallback only; appFetch is source-inspected, not tested by this script.
  - `internal/server/http_chain_test.go` (server-side verification)

---

### 1.2 `window.scheduleFetch` (`web/static/js/schedule/schedulefetch.js`)

- **Role**: Specialized fetch wrapper for schedule and async dashboard workflows requiring automated session handling.
- **Signature**: `scheduleFetch(input, init)` -> `Promise<Response>`
- **Behavior Contract**:
  1. **CSRF Propagation**: Ensures `X-CSRF-Token` is attached even if falling back to vanilla `fetch`.
  2. **Session Expiry (401)**: When response status is 401, or if redirected to `/login`, immediately redirects window to `/login?redirect=<currentPath>` and rejects promise.
  3. **Preservation of Business Errors (403, 409, 500+)**: Strictly distinguishes permission denial (403), concurrent conflicts (409), and internal server errors (500+) from session expiry. These errors are returned to caller without redirecting.

Use `scheduleFetch(url, options)` with a route and payload confirmed in the
current module's route registration. It returns a raw Response: inspect status,
then parse the endpoint's envelope (`error`, with `message` only for documented
legacy endpoints). The former `/schedule/windows/1/tasks` PUT example was not a
registered route and must not be copied.


- **Automated Evidence (not browser acceptance)**:
  - `tests/e2e/auth-errors.spec.js` (preservation of 403/409/500+ and detection of 401)
  - `tests/e2e/csrf-tokens.spec.js` (header attachment fallback)

---

## 2. CSRF Token Propagation Protocol

To prevent CSRF vulnerabilities while maintaining compatibility across form PRG and AJAX writes:

1. **HTML Meta Tag**:
   Rendered in both `layout/base.html` and `layout/auth.html`:
   ```html
   <meta name="csrf-token" content="{{ .CSRFToken }}">
   ```
2. **Standard Form Submissions (PRG)**:
   Every POST/PUT/DELETE `<form>` must include a hidden CSRF token input:
   ```html
   <input type="hidden" name="csrf_token" value="{{ .CSRFToken }}">
   ```
3. **AJAX Mutations**:
   All non-safe HTTP methods (POST, PUT, PATCH, DELETE) must supply the token either via `window.appFetch` or in `X-CSRF-Token` header.
4. **Backend Verification**:
   `internal/middleware/csrf.go` intercepts requests before Gin handlers. Missing or invalid tokens return 403 Forbidden (`application/json` for AJAX, HTML for standard navigation).

---

## 3. UI Component Boundaries

- **Pagination**: Rendered server-side via `web/templates/components/pager.html`. Supports query preservation, jump controls, and page size selection.
- **Modal Dialogs**: `openModal(id)` and `closeModal(id)` in `app.js` toggle `.open` CSS class.
- **Form Loading States**: `bindFormLoading()` in `app.js` disables the submit button and applies a spinner, restoring it after 3 seconds; it does not cancel repeated submit events or provide server-side idempotency.
- **Action Confirmations**: Handled via `[data-confirm]` attribute and native confirmation.
- **HTML Escaping**: `window.escapeHtml(value)` — provided by `ui.js`, loaded globally by `layout/base.html` — is the **capability-level canonical source** for HTML escaping. Per the Evidence status note above, this capability-level certification does **not** extend to the `ui.js` file as a whole, which remains not certified as a global golden reference. Call `window.escapeHtml` directly; a pre-existing module-level compatibility API may survive only as a one-line delegation (`function escapeHtml(v) { return window.escapeHtml(v); }`) for real external consumers. Do not introduce a new shared utility layer, and do not relocate the implementation (for example into `app.js`).
