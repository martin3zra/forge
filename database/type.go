package database

type ConnectionKey struct{}

// DialectKey looks up the active playsql dialect name ("postgres", "mysql",
// "sqlite") from context, set once at boot alongside ConnectionKey. Code that
// builds SQL without going through playsql's Builder (e.g. validator's dynamic
// exists/unique rules) uses it to pick the right placeholder style.
type DialectKey struct{}

// PlaysqlKey looks up the *playsql.DB from context, set once at boot alongside
// ConnectionKey. Code that queries through playsql's Builder (e.g. forge/auth's
// credential/password resolvers) uses this instead of ConnectionKey.
type PlaysqlKey struct{}
