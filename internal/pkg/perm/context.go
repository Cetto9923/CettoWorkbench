package perm

import "context"

// permissionContextKey is private so unrelated request values cannot collide
// with the authenticated capability set.
type permissionContextKey struct{}

// WithGranted attaches the authenticated capability snapshot to a request
// context for service-layer action derivation. Authorization itself remains in
// middleware and the service's object-level checks.
func WithGranted(ctx context.Context, granted map[string]bool) context.Context {
	copyOfGranted := make(map[string]bool, len(granted))
	for code, allowed := range granted {
		copyOfGranted[code] = allowed
	}
	return context.WithValue(ctx, permissionContextKey{}, copyOfGranted)
}

// HasAnyGranted reports whether the authenticated request has one of perms.
// A missing snapshot fails closed; super-admin handling belongs to middleware.
func HasAnyGranted(ctx context.Context, perms ...Permission) bool {
	granted, ok := ctx.Value(permissionContextKey{}).(map[string]bool)
	if !ok {
		return false
	}
	for _, p := range perms {
		if granted[p.String()] {
			return true
		}
	}
	return false
}
