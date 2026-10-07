package turso

import "slices"

// Scope is a permission label carried by a scoped platform API token.
// The vocabulary mirrors api/service/scope/scope.go on the platform side; if
// the platform adds or renames scopes, this file must follow.
type Scope string

const (
	ScopeRead             Scope = "read"
	ScopeDbCreate         Scope = "db:create"
	ScopeDbDelete         Scope = "db:delete"
	ScopeDbConfigure      Scope = "db:configure"
	ScopeDbMintToken      Scope = "db:mint-token"
	ScopeDbRotateCreds    Scope = "db:rotate-creds"
	ScopeGroupConfigure   Scope = "group:configure"
	ScopeGroupMintToken   Scope = "group:mint-token"
	ScopeGroupRotateCreds Scope = "group:rotate-creds"
	ScopeGroupCreate      Scope = "group:create"
	ScopeGroupDelete      Scope = "group:delete"
	ScopeOrgRead          Scope = "org:read"
)

// AllScopes is the canonical ordering used by --help output and the
// individual-scope validator. Presets ("read-only", "full-access") are
// accepted by the platform but not listed here — they're handled as
// separate CLI flags.
var AllScopes = []Scope{
	ScopeRead,
	ScopeDbCreate,
	ScopeDbDelete,
	ScopeDbConfigure,
	ScopeDbMintToken,
	ScopeDbRotateCreds,
	ScopeGroupConfigure,
	ScopeGroupMintToken,
	ScopeGroupRotateCreds,
}

// OrgLevelScopes may only be granted to organization-scoped tokens.
var OrgLevelScopes = []Scope{
	ScopeGroupCreate,
	ScopeGroupDelete,
	ScopeOrgRead,
}

// IsValidScope reports whether s is a known scope label. Used to surface
// typos client-side instead of waiting for the platform 400.
func IsValidScope(s string) bool {
	return slices.Contains(AllScopes, Scope(s)) || IsOrgLevelScope(s)
}

// IsOrgLevelScope reports whether s may only be granted to org-scoped tokens.
func IsOrgLevelScope(s string) bool {
	return slices.Contains(OrgLevelScopes, Scope(s))
}
