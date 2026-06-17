package foundation

import "database/sql"

// Authenticatable is the contract a user identity must satisfy to participate in
// authentication. The concrete user type (and its persistence) lives in the
// application layer, not in foundation.
type Authenticatable interface {
	GetAuthIdentifier() int
	GetAuthIdentifierName() string
	GetAuthPassword() string
	GetRole() string
	SetRole(role string)
}

// MustVerifyPassword is implemented by identities that can be forced to rotate
// their password.
type MustVerifyPassword interface {
	HasNotChangedPassword() bool
	MarkPasswordAsChanged(db *sql.DB, password string) error
}
