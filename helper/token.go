package helper

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"time"
)

const tokenIssuer = "secure-patrol-backend"

var (
	ErrTokenInvalid = errors.New("token is invalid")
	ErrTokenExpired = errors.New("token is expired")
)

// AccessTokenClaims is the payload of the access token (JWT, HS256).
type AccessTokenClaims struct {
	Subject   int64  `json:"sub"`
	SessionID string `json:"sid"`
	Role      string `json:"role"`
	AppID     string `json:"aid"`
	Issuer    string `json:"iss"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
}

var jwtHeader = base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))

func GenerateAccessToken(claims AccessTokenClaims) (string, error) {
	claims.Issuer = tokenIssuer

	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	unsigned := jwtHeader + "." + base64.RawURLEncoding.EncodeToString(payload)
	return unsigned + "." + signJWT(unsigned), nil
}

func ParseAccessToken(token string) (AccessTokenClaims, error) {
	var claims AccessTokenClaims

	parts := strings.Split(token, ".")
	// The header is fixed, so "alg: none" or other algorithms are rejected here.
	if len(parts) != 3 || parts[0] != jwtHeader {
		return claims, ErrTokenInvalid
	}

	expected := signJWT(parts[0] + "." + parts[1])
	if !hmac.Equal([]byte(expected), []byte(parts[2])) {
		return claims, ErrTokenInvalid
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return claims, ErrTokenInvalid
	}

	if err := json.Unmarshal(payload, &claims); err != nil || claims.Issuer != tokenIssuer {
		return claims, ErrTokenInvalid
	}

	if time.Now().Unix() >= claims.ExpiresAt {
		return claims, ErrTokenExpired
	}

	return claims, nil
}

func signJWT(unsigned string) string {
	mac := hmac.New(sha256.New, []byte(os.Getenv("JWT_SECRET")))
	mac.Write([]byte(unsigned))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// RandomHex returns n random bytes encoded as hex.
func RandomHex(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// RandomURLToken returns n random bytes encoded as unpadded base64url.
func RandomURLToken(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func SHA256Hex(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}
