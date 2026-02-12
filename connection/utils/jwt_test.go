package utils

import (
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
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			ID:        "jti123",
		},
	}
	tokenObj := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, err := tokenObj.SignedString(secret)
	require.NoError(t, err)

	_, err = ParseJWT(token)
	require.Error(t, err)
	//require.True(t, parsedClaims.ExpiresAt.Time.Before(time.Now()))
}
