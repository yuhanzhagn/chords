package handler

// ContextKey is used for middleware context keys.
type ContextKey = string

const (
	// RequestContextKey stores the original *http.Request in Context.Values.
	RequestContextKey ContextKey = "http_request"
	// JWTClaimsContextKey stores validated JWT claims in Context.Values and Context.Context.
	JWTClaimsContextKey ContextKey = "jwt_claims"
)
