// Package access defines route access requirements and helpers to build
// chi middleware chains that enforce them.
//
// Actual authn/authz logic is stubbed; replace the middleware bodies with
// real implementations when you add an identity layer.
package access

import (
	"net/http"
)

// Access describes the authorization requirements for a route or menu item.
type Access struct {
	// Public routes require no authentication or authorization.
	Public bool
	// RequireAuthn enforces that the request has a valid authenticated identity.
	// Ignored when Public is true.
	RequireAuthn bool
	// Permissions is an optional list of permission strings that must all be
	// present on the identity.  Ignored when Public is true.
	Permissions []string
	// Roles is an optional list of roles the identity must hold.
	// Ignored when Public is true.
	Roles []string
}

// Public returns an Access that allows anyone, no auth required.
func Public() Access { return Access{Public: true} }

// Private returns an Access that requires authentication (no additional authz).
func Private() Access { return Access{RequireAuthn: true} }

// Perm returns an Access requiring authentication plus the listed permissions.
func Perm(perms ...string) Access {
	return Access{RequireAuthn: true, Permissions: perms}
}

// Role returns an Access requiring authentication plus the listed roles.
func Role(roles ...string) Access {
	return Access{RequireAuthn: true, Roles: roles}
}

// Middleware returns a slice of http.Handler-wrapping middleware that enforces
// the access requirements.  Add real authn/authz logic here.
func Middleware(a Access) []func(http.Handler) http.Handler {
	if a.Public {
		return nil
	}
	var chain []func(http.Handler) http.Handler
	if a.RequireAuthn {
		chain = append(chain, authnMiddleware)
	}
	if len(a.Permissions) > 0 || len(a.Roles) > 0 {
		chain = append(chain, authzMiddleware(a))
	}
	return chain
}

// authnMiddleware is a stub that would validate a session/JWT and populate
// the identity in the context.  Replace with real logic.
func authnMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO: validate session / JWT, set identity in context
		next.ServeHTTP(w, r)
	})
}

// authzMiddleware is a stub that would check permissions/roles on the identity
// stored in the context.  Replace with real logic.
func authzMiddleware(_ Access) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// TODO: check permissions / roles from context identity
			next.ServeHTTP(w, r)
		})
	}
}
