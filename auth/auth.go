package auth

import (
	"context"
	"errors"
	"log"

	"github.com/martin3zra/forge/database"
	"github.com/martin3zra/forge/foundation"
	"github.com/martin3zra/playsql"
)

type ContextUserID struct{}

// CredentialResolver retrieves an identity by an arbitrary column/value pair.
// The application registers it so auth never needs to know the user schema.
type CredentialResolver func(db *playsql.DB, column string, value any) (foundation.Authenticatable, error)

// PasswordResolver returns the stored password hash for a user id.
type PasswordResolver func(db *playsql.DB, userID int) (string, error)

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
	db       *playsql.DB
	Hashable foundation.Hash
}

func NewAuth(ctx context.Context) *Auth {
	db := ctx.Value(database.PlaysqlKey{}).(*playsql.DB)

	if db == nil {
		panic("database connection need to be set.")
	}
	return &Auth{
		db:       db,
		Hashable: foundation.NewHashable(),
	}
}

func (a *Auth) Authenticate(username, password string) (foundation.Authenticatable, error) {
	return a.AuthenticateBy("email", username, password)
}

// AuthenticateBy is Authenticate generalized to an arbitrary lookup column —
// a host app that lets a user log in with, say, either an email or a
// username picks the column itself and calls this instead. The registered
// CredentialResolver already accepts any column (see SetCredentialResolver);
// this is just the exported entry point that lets a caller pass one.
func (a *Auth) AuthenticateBy(column, identifier, secret string) (foundation.Authenticatable, error) {
	user, err := a.attempt(column, identifier)
	if err != nil {
		log.Printf("error authenticating user: %s\n", err)
		return nil, err
	}

	if !a.EnsureIsCurrentPassword(user.GetAuthPassword(), secret) {
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
