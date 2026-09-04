# Frontend engineering

Templates render server-owned data. Browser JavaScript owns events, requests,
transient view state, and rendering. It MUST NOT be the only implementation of
permission checks, business filtering, totals, or state transitions.

Before page-local code, inspect shared support for HTML escaping; fetch/JSON/error
and authentication handling; toast/modal; pagination/dropdowns; form errors;
CSRF; and distinct loading/empty/error states. New code MUST reuse a matching
stable capability. If similar implementations have different contracts, keep
them local and record why. Extract only after behavior and API are stable.

`app.js`/shared UI files provide transport and primitives; `components/` owns
reusable widgets; `layout/` owns shell navigation; module directories own page
behavior. Page scripts SHOULD be capability-focused and MUST meet the size gate.

CSS tokens define design values, shared component CSS defines reusable widget
geometry/states, and page CSS composes/specializes a page. Do not copy tokens or
a full shared component into page CSS.

Internal/object navigation opens in the current page by default. A new window
requires an explicit product reason and safe `noopener` handling.

Loading, empty, and error are separate states. Errors must not render as empty,
stale rows must not remain after failed refresh, and retry should be visible when
possible. Fetch handling covers non-2xx, invalid payload, auth expiry, and user
feedback consistently.
