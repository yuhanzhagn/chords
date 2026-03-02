package middlewares

import (
	"connection/internal/handler"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type testClaims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func TestJWTAuthMiddleware_ValidHS256TokenInjectsClaims(t *testing.T) {
	secret := []byte("test-secret")
	token := mustSignToken(t, jwt.SigningMethodHS256, secret, &testClaims{
		Username: "alice",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-1 * time.Minute)),
		},
	})

	mw := JWTAuthMiddleware[*testClaims](JWTAuthOptions[*testClaims]{
		NewClaims: func() *testClaims { return &testClaims{} },
		Keyfunc: KeyfuncByAlgorithm(map[string]any{
			jwt.SigningMethodHS256.Alg(): secret,
		}),
	})

	req, _ := http.NewRequest(http.MethodGet, "/ws", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	ctx := &handler.Context{
		Context: context.Background(),
		Values: map[string]any{
			handler.RequestContextKey: req,
		},
	}

	h := mw(func(c *handler.Context) error {
		v, ok := c.Values[handler.JWTClaimsContextKey]
		if !ok {
			t.Fatal("claims were not injected into handler values")
		}
		claims, ok := v.(*testClaims)
		if !ok {
			t.Fatalf("unexpected claims value type: %T", v)
		}
		if claims.Username != "alice" {
			t.Fatalf("unexpected claim username: %s", claims.Username)
		}

		fromContext, ok := c.Context.Value(handler.JWTClaimsContextKey).(*testClaims)
		if !ok || fromContext.Username != "alice" {
			t.Fatalf("claims were not injected into context correctly: %#v", fromContext)
		}
		return nil
	})

	if err := h(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestJWTAuthMiddleware_RejectsExpiredToken(t *testing.T) {
	secret := []byte("test-secret")
	token := mustSignToken(t, jwt.SigningMethodHS256, secret, &testClaims{
		Username: "bob",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	})

	mw := JWTAuthMiddleware[*testClaims](JWTAuthOptions[*testClaims]{
		NewClaims: func() *testClaims { return &testClaims{} },
		Keyfunc: KeyfuncByAlgorithm(map[string]any{
			jwt.SigningMethodHS256.Alg(): secret,
		}),
	})

	req, _ := http.NewRequest(http.MethodGet, "/ws", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	ctx := &handler.Context{
		Values: map[string]any{
			handler.RequestContextKey: req,
		},
	}

	err := mw(func(_ *handler.Context) error { return nil })(ctx)
	if err == nil {
		t.Fatal("expected expired token error")
	}
}

func TestJWTAuthMiddleware_RejectsMissingAuthorizationHeader(t *testing.T) {
	secret := []byte("test-secret")
	mw := JWTAuthMiddleware[*testClaims](JWTAuthOptions[*testClaims]{
		NewClaims: func() *testClaims { return &testClaims{} },
		Keyfunc: KeyfuncByAlgorithm(map[string]any{
			jwt.SigningMethodHS256.Alg(): secret,
		}),
	})

	req, _ := http.NewRequest(http.MethodGet, "/ws", nil)
	ctx := &handler.Context{
		Values: map[string]any{
			handler.RequestContextKey: req,
		},
	}

	err := mw(func(_ *handler.Context) error { return nil })(ctx)
	if err == nil {
		t.Fatal("expected missing header error")
	}
}

func TestJWTAuthMiddleware_ValidRS256Token(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	token := mustSignToken(t, jwt.SigningMethodRS256, privateKey, &testClaims{
		Username: "carol",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-1 * time.Minute)),
		},
	})

	mw := JWTAuthMiddleware[*testClaims](JWTAuthOptions[*testClaims]{
		NewClaims: func() *testClaims { return &testClaims{} },
		Keyfunc: KeyfuncByAlgorithm(map[string]any{
			jwt.SigningMethodRS256.Alg(): &privateKey.PublicKey,
		}),
	})

	req, _ := http.NewRequest(http.MethodGet, "/ws", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	ctx := &handler.Context{
		Values: map[string]any{
			handler.RequestContextKey: req,
		},
	}

	if err := mw(func(_ *handler.Context) error { return nil })(ctx); err != nil {
		t.Fatalf("unexpected error for RS256 token: %v", err)
	}
}

func mustSignToken(t *testing.T, method jwt.SigningMethod, key any, claims jwt.Claims) string {
	t.Helper()

	token := jwt.NewWithClaims(method, claims)
	signed, err := token.SignedString(key)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return signed
}
