package middlewares

import (
	"connection/internal/handler"
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

var mainstreamJWTAlgs = []string{
	jwt.SigningMethodHS256.Alg(),
	jwt.SigningMethodHS384.Alg(),
	jwt.SigningMethodHS512.Alg(),
	jwt.SigningMethodRS256.Alg(),
	jwt.SigningMethodRS384.Alg(),
	jwt.SigningMethodRS512.Alg(),
	jwt.SigningMethodES256.Alg(),
	jwt.SigningMethodES384.Alg(),
	jwt.SigningMethodES512.Alg(),
	jwt.SigningMethodEdDSA.Alg(),
}

// JWTAuthOptions configures JWTAuthMiddleware.
type JWTAuthOptions[C jwt.Claims] struct {
	// NewClaims must return a fresh claims instance for each parse call.
	NewClaims func() C
	// Keyfunc resolves signature validation keys based on token headers.
	Keyfunc jwt.Keyfunc
	// AllowedAlgorithms restricts accepted signing algorithms.
	// When empty, mainstreamJWTAlgs is used.
	AllowedAlgorithms []string
	// ClaimsContextKey sets where validated claims are stored.
	// When empty, handler.JWTClaimsContextKey is used.
	ClaimsContextKey string
}

// KeyfuncByAlgorithm returns a keyfunc that resolves keys by JWT "alg".
func KeyfuncByAlgorithm(keys map[string]any) jwt.Keyfunc {
	return func(token *jwt.Token) (any, error) {
		if token == nil || token.Method == nil {
			return nil, errors.New("token signing method is missing")
		}
		alg := token.Method.Alg()
		key, ok := keys[alg]
		if !ok {
			return nil, fmt.Errorf("no verification key configured for alg %q", alg)
		}
		return key, nil
	}
}

// JWTAuthMiddleware validates a JWT from the websocket upgrade request.
func JWTAuthMiddleware[C jwt.Claims](opts JWTAuthOptions[C]) handler.Middleware {
	allowedAlgorithms := opts.AllowedAlgorithms
	if len(allowedAlgorithms) == 0 {
		allowedAlgorithms = mainstreamJWTAlgs
	}

	claimsContextKey := opts.ClaimsContextKey
	if claimsContextKey == "" {
		claimsContextKey = handler.JWTClaimsContextKey
	}

	return func(next handler.HandlerFunc) handler.HandlerFunc {
		return func(c *handler.Context) error {
			if c == nil {
				return errors.New("context is required")
			}
			if opts.NewClaims == nil {
				return errors.New("jwt middleware is misconfigured: NewClaims is required")
			}
			if opts.Keyfunc == nil {
				return errors.New("jwt middleware is misconfigured: Keyfunc is required")
			}

			req, err := requestFromContext(c)
			if err != nil {
				return err
			}

			tokenString, err := bearerToken(req)
			if err != nil {
				return err
			}

			claims := opts.NewClaims()
			token, err := jwt.ParseWithClaims(tokenString, claims, opts.Keyfunc, jwt.WithValidMethods(allowedAlgorithms))
			if err != nil {
				return fmt.Errorf("jwt validation failed: %w", err)
			}
			if !token.Valid {
				return errors.New("jwt token is invalid")
			}

			if c.Values == nil {
				c.Values = make(map[string]any)
			}
			c.Values[claimsContextKey] = claims

			if c.Context == nil {
				c.Context = context.Background()
			}
			c.Context = context.WithValue(c.Context, claimsContextKey, claims)

			return next(c)
		}
	}
}

func requestFromContext(c *handler.Context) (*http.Request, error) {
	if c.Values == nil {
		return nil, errors.New("missing request context")
	}
	raw, ok := c.Values[handler.RequestContextKey]
	if !ok {
		return nil, errors.New("missing request context")
	}
	req, ok := raw.(*http.Request)
	if !ok || req == nil {
		return nil, errors.New("invalid request context")
	}
	return req, nil
}

func bearerToken(req *http.Request) (string, error) {
	authHeader := req.Header.Get("Authorization")
	parts := strings.Fields(authHeader)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", errors.New("missing or invalid Authorization header")
	}
	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", errors.New("missing bearer token")
	}
	return token, nil
}
