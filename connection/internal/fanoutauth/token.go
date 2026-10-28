// Package fanoutauth defines the dedicated worker-to-gateway credential.
package fanoutauth

import (
	"crypto/sha256"
	"crypto/subtle"
	"errors"
)

const Header = "X-Chords-Fanout-Token"

// Validate accepts visible ASCII without spaces so HTTP preserves the exact value.
func Validate(token string) error {
	if len(token) < 32 {
		return errors.New("FANOUT_TOKEN must contain at least 32 bytes of visible ASCII without spaces")
	}
	for i := range token {
		if token[i] < 33 || token[i] > 126 {
			return errors.New("FANOUT_TOKEN must contain only visible ASCII without spaces")
		}
	}
	return nil
}

func Matches(token, expected string) bool {
	a, b := sha256.Sum256([]byte(token)), sha256.Sum256([]byte(expected))
	return subtle.ConstantTimeCompare(a[:], b[:]) == 1
}
