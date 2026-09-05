# Shared Frontend Capabilities

This document catalogs the shared client-side scripts, fetch abstractions, and UI patterns across Workbench templates.

## 1. Network / Fetch Layers

Workbench provides two primary shared fetch wrappers in `web/static/js`:

### `window.appFetch` (`web/static/js/app.js`)
- **Purpose**: Base fetch wrapper for application requests.
- **Headers injected**:
  - `X-Requested-With: XMLHttpRequest`
  - `X-CSRF-Token`: Extracted from `<meta name="csrf-token" content="...">`.
- **Return Contract**: Returns a raw standard `Promise<Response>`.
  > [!IMPORTANT]
  > `appFetch` does **not** parse or standardize error envelopes. Callers must inspect `response.ok`, `response.status`, and deserialize JSON or text themselves.

### `window.scheduleFetch` (`web/static/js/schedule/schedulefetch.js`)
- **Purpose**: Specialized wrapper for schedule and async component workflows.
- **Session Expiry Handling**:
  - Intercepts 401 status and redirects to `/login?redirect=<currentPath>`.
  - Recognizes redirect responses targeting `/login`.
  - **Does NOT redirect** on 403 (forbidden), 409 (conflict), or 500+ (server error), preserving business error messages.
- **CSRF Token**: Automatically extracts token from meta and injects `X-CSRF-Token` header if not already present.

## 2. CSRF Token Conventions

- **HTML Meta Tag**:
  Both `web/templates/layout/base.html` and `web/templates/layout/auth.html` render:
  ```html
  <meta name="csrf-token" content="{{ .CSRFToken }}">
  ```
- **HTML Form Submissions**:
  POST/PUT/DELETE forms must render a hidden token input:
  ```html
  <input type="hidden" name="csrf_token" value="{{ .CSRFToken }}">
  ```
- **JavaScript Callers**:
  AJAX / `fetch` mutations must retrieve the token via `getCsrfToken()` and attach the `X-CSRF-Token` header.

## 3. Shared UI Components

- **Pagination**: `web/templates/components/pager.html` provides standard pagination controls with page size selector and jump input.
- **Modal Dialogs**: Controlled via `openModal(id)` and `closeModal(id)` with backdrop support.
- **Submit Loading States**: `bindFormLoading()` automatically disables submit buttons and displays loading spinners on form submission.
- **Action Confirmations**: Elements with `data-confirm="Message"` automatically prompt a confirmation modal or alert before proceeding.
