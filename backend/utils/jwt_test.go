package utils

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

func TestGenerateAndParseJWT(t *testing.T) {
	username := "alice"
	sessionID := "sess123"

	token, jti, exp, err := GenerateJWT(username, sessionID)
	require.NoError(t, err)
	require.NotEmpty(t, token)
	require.NotEmpty(t, jti)
	require.WithinDuration(t, time.Now().Add(1*time.Hour), exp, 5*time.Second)

	// Parse the token
	claims, err := ParseJWT(token)
	require.NoError(t, err)
	require.Equal(t, username, claims.Username)
	require.Equal(t, sessionID, claims.SessionID)
	require.Equal(t, jti, claims.ID)
	require.WithinDuration(t, exp, claims.ExpiresAt.Time, 5*time.Second)
}

func TestParseJWT_InvalidToken(t *testing.T) {
	_, err := ParseJWT("invalid.token.here")
	require.Error(t, err)
}

func TestParseJWT_ExpiredToken(t *testing.T) {
	// generate a token that expires immediately
	exp := time.Now().Add(-time.Minute)
	claims := Claims{
		Username:  "bob",
		SessionID: "sess456",
		RegisteredClaims: jwt.RegisteredClaims{
			Audience:  jwt.ClaimStrings{"chords-access"},
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			ID:        "jti123",
		},
	}
	tokenObj := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, err := tokenObj.SignedString(signingKey())
	require.NoError(t, err)

	_, err = ParseJWT(token)
	require.Error(t, err)
	require.True(t, errors.Is(err, jwt.ErrTokenExpired))
}

func TestJWTRequiresRuntimeKey(t *testing.T) {
	t.Setenv("JWT_SECRET", "")
	_, _, _, err := GenerateJWT("alice", "session")
	require.Error(t, err)
	_, err = SignCommand(CommandClaims{})
	require.Error(t, err)
}
func TestJWTRejectsWrongAlgorithmAndMissingExpiry(t *testing.T) {
	for _, method := range []jwt.SigningMethod{jwt.SigningMethodHS256, jwt.SigningMethodHS512} {
		claims := Claims{Username: "alice", SessionID: "session", RegisteredClaims: jwt.RegisteredClaims{Audience: jwt.ClaimStrings{"chords-access"}}}
		raw, err := jwt.NewWithClaims(method, claims).SignedString(signingKey())
		require.NoError(t, err)
		_, err = ParseJWT(raw)
		require.Error(t, err)
	}
	claims := CommandClaims{RegisteredClaims: jwt.RegisteredClaims{Audience: jwt.ClaimStrings{"chords-command"}, ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Minute))}}
	raw, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(signingKey())
	require.NoError(t, err)
	_, err = ParseCommand(raw)
	require.Error(t, err)
}
