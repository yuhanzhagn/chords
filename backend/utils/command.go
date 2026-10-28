package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"github.com/golang-jwt/jwt/v5"
	"time"
)

// CommandClaims authorizes one exact serialized event for a short interval.
type CommandClaims struct {
	Username  string `json:"username"`
	SessionID string `json:"session_id"`
	UserID    uint   `json:"user_id"`
	RoomID    uint   `json:"room_id"`
	Digest    string `json:"digest"`
	jwt.RegisteredClaims
}

func EventDigest(payload []byte) string { h := sha256.Sum256(payload); return hex.EncodeToString(h[:]) }
func SignCommand(c CommandClaims) (string, error) {
	if err := ValidateJWTConfig(); err != nil {
		return "", err
	}
	c.RegisteredClaims = jwt.RegisteredClaims{Audience: jwt.ClaimStrings{"chords-command"}, ExpiresAt: jwt.NewNumericDate(time.Now().Add(30 * time.Second))}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(signingKey())
}
func ParseCommand(raw string) (*CommandClaims, error) {
	c := &CommandClaims{}
	token, err := jwt.ParseWithClaims(raw, c, func(*jwt.Token) (any, error) {
		if err := ValidateJWTConfig(); err != nil {
			return nil, err
		}
		return signingKey(), nil
	}, jwt.WithValidMethods([]string{"HS256"}), jwt.WithAudience("chords-command"), jwt.WithExpirationRequired())
	if err != nil || !token.Valid {
		return nil, errors.New("invalid command authorization")
	}
	return c, nil
}
