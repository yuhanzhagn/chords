package identity

import "context"

// Principal is a value copied from verified authentication, never JSON-bound.
type Principal struct {
	userID              uint
	username, sessionID string
}

func New(userID uint, username, sessionID string) Principal {
	return Principal{userID, username, sessionID}
}
func (p Principal) UserID() uint      { return p.userID }
func (p Principal) Username() string  { return p.username }
func (p Principal) SessionID() string { return p.sessionID }
func (p Principal) Valid() bool       { return p.userID != 0 && p.username != "" && p.sessionID != "" }

type contextKey struct{}

func WithContext(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, contextKey{}, p)
}
func FromContext(ctx context.Context) Principal {
	p, _ := ctx.Value(contextKey{}).(Principal)
	return p
}
