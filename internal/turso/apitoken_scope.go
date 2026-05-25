package turso

// Scope is a permission label carried by a group-scoped platform API token.
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

// IsValidScope reports whether s is a known scope label. Used to surface
// typos client-side instead of waiting for the platform 400.
func IsValidScope(s string) bool {
	for _, sc := range AllScopes {
		if string(sc) == s {
			return true
		}
	}
	return false
}
