package auth

import (
	"context"
	"database/sql"
	"errors"
	"log"

	"github.com/martin3zra/forge/foundation"
	"github.com/martin3zra/forge/database"
)

type ContextUserID struct{}

// CredentialResolver retrieves an identity by an arbitrary column/value pair.
// The application registers it so auth never needs to know the user schema.
type CredentialResolver func(db *sql.DB, column string, value any) (foundation.Authenticatable, error)

// PasswordResolver returns the stored password hash for a user id.
type PasswordResolver func(db *sql.DB, userID int) (string, error)

// UserDecoder rebuilds the authenticated identity from the request context.
type UserDecoder func(ctx context.Context) foundation.Authenticatable

var (
	credentialResolver CredentialResolver
	passwordResolver   PasswordResolver
	userDecoder        UserDecoder
)

// SetCredentialResolver registers the application's identity lookup.
func SetCredentialResolver(r CredentialResolver) { credentialResolver = r }

// SetPasswordResolver registers the application's password lookup.
func SetPasswordResolver(r PasswordResolver) { passwordResolver = r }

// SetUserDecoder registers the application's context decoder.
func SetUserDecoder(d UserDecoder) { userDecoder = d }

type Auth struct {
	db       *sql.DB
	Hashable foundation.Hash
}

func NewAuth(ctx context.Context) *Auth {
	db := ctx.Value(database.ConnectionKey{}).(*sql.DB)

	if db == nil {
		panic("database connection need to be set.")
	}
	return &Auth{
		db:       db,
		Hashable: foundation.NewHashable(),
	}
}

func (a *Auth) Authenticate(username, password string) (foundation.Authenticatable, error) {
	user, err := a.attempt("email", username)
	if err != nil {
		log.Printf("error authenticating user: %s\n", err)
		return nil, err
	}

	if !a.EnsureIsCurrentPassword(user.GetAuthPassword(), password) {
		log.Printf("error invalid password")
		return nil, errors.New("error invalid password")
	}

	return user, nil
}

func (a *Auth) LoginUsingId(id int) (foundation.Authenticatable, error) {
	user, err := a.attempt("id", id)
	if err != nil {
		log.Printf("error authenticating user: %s\n", err)
		return nil, err
	}

	return user, nil
}

func (a *Auth) EnsureIsCurrentPassword(hashed, password string) bool {
	return a.Hashable.Check(password, hashed)
}

func (a *Auth) attempt(column string, value any) (foundation.Authenticatable, error) {
	if credentialResolver == nil {
		return nil, errors.New("auth: credential resolver not registered")
	}
	return credentialResolver(a.db, column, value)
}

func (a *Auth) GetCurrentPassword(userId int) (string, error) {
	if passwordResolver == nil {
		return "", errors.New("auth: password resolver not registered")
	}
	return passwordResolver(a.db, userId)
}

// User retrieves the currently authenticated identity from context.
func User(ctx context.Context) foundation.Authenticatable {
	if userDecoder == nil {
		return nil
	}
	return userDecoder(ctx)
}

// ID retrieves the currently authenticated user's id.
func ID(ctx context.Context) int {
	user := User(ctx)
	if user == nil {
		return 0
	}
	return user.GetAuthIdentifier()
}
