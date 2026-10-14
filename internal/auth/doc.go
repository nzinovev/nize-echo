// Package auth handles passwords (argon2id), server-side sessions, login rate
// limiting and CSRF tokens. Admin sessions are one kind of a generic user
// session, so reader accounts can be added later. Implemented in A-01.
package auth
