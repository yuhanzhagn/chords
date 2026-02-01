// utils/jwt.go
package utils

import (
    "time"

    "github.com/golang-jwt/jwt/v5"
)

var secret = []byte("CHANGE_ME")

type Claims struct {
    Username string `json:"Username"`
	SessionID string	
    jwt.RegisteredClaims
}

func GenerateJWT(username string, sessionID string) (token string, jti string, exp time.Time, err error) {
    exp = time.Now().Add(1 * time.Hour)
    jti = generateJTI()

    claims := Claims{
        Username: username,
		SessionID: sessionID,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(exp),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
            ID:        jti,
        },
    }

    tokenObj := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    token, err = tokenObj.SignedString(secret)
    return
}

func ParseJWT(tokenStr string) (*Claims, error) {
    tok, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
        return secret, nil
    })
    if err != nil {
        return nil, err
    }

    return tok.Claims.(*Claims), nil
}

func generateJTI() string {
    return jwt.NewNumericDate(time.Now()).String()
}

