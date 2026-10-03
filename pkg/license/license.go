// Package license encodes and verifies Secure Patrol license codes.
//
// A license code is
//
//	SPL1.<payload>.<signature>
//
// where payload is the base64url (no padding) JSON of Payload and signature is
// the base64url (no padding) Ed25519 signature of the ASCII text "SPL1.<payload>".
// Only the vendor holds the private key and can create codes; the backend only
// has the public key, so a customer cannot create or change a license. The same
// format is implemented by the license generator (see docs/LICENSE_SPEC.md).
package license

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Prefix identifies the format version.
const Prefix = "SPL1"

const maxGraceDays = 365

var (
	ErrMalformed        = errors.New("license code is malformed")
	ErrSignatureInvalid = errors.New("license code signature is invalid")
	ErrPayloadInvalid   = errors.New("license code content is invalid")
)

var b64 = base64.RawURLEncoding

// Payload is the content of a license.
type Payload struct {
	LicenseID     string    `json:"lid"`
	Customer      string    `json:"customer"`
	InstallID     string    `json:"install_id"`
	MaxUnits      int       `json:"max_units"`
	MaxAppClients int       `json:"max_app_clients"`
	IssuedAt      time.Time `json:"issued_at"`
	ExpiresAt     time.Time `json:"expires_at"`
	GraceDays     int       `json:"grace_days"`
}

// GraceUntil is the moment the license stops working.
func (p Payload) GraceUntil() time.Time {
	return p.ExpiresAt.Add(time.Duration(p.GraceDays) * 24 * time.Hour)
}

// Validate checks the fields every license must have.
func (p Payload) Validate() error {
	var problems []string
	if strings.TrimSpace(p.LicenseID) == "" {
		problems = append(problems, "lid is required")
	}
	if strings.TrimSpace(p.InstallID) == "" {
		problems = append(problems, "install_id is required")
	}
	if p.MaxUnits < 1 {
		problems = append(problems, "max_units must be at least 1")
	}
	if p.MaxAppClients < 1 {
		problems = append(problems, "max_app_clients must be at least 1")
	}
	if p.IssuedAt.IsZero() || p.ExpiresAt.IsZero() {
		problems = append(problems, "issued_at and expires_at are required")
	} else if !p.ExpiresAt.After(p.IssuedAt) {
		problems = append(problems, "expires_at must be after issued_at")
	}
	if p.GraceDays < 0 || p.GraceDays > maxGraceDays {
		problems = append(problems, fmt.Sprintf("grace_days must be between 0 and %d", maxGraceDays))
	}
	if len(problems) > 0 {
		return fmt.Errorf("%w: %s", ErrPayloadInvalid, strings.Join(problems, "; "))
	}
	return nil
}

// Sign creates a license code. It is used by the vendor tools only.
func Sign(payload Payload, privateKey ed25519.PrivateKey) (string, error) {
	if err := payload.Validate(); err != nil {
		return "", err
	}
	if len(privateKey) != ed25519.PrivateKeySize {
		return "", errors.New("private key must be an Ed25519 private key")
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	signed := Prefix + "." + b64.EncodeToString(data)
	signature := ed25519.Sign(privateKey, []byte(signed))
	return signed + "." + b64.EncodeToString(signature), nil
}

// Verify checks the signature of a code with the public key and returns its payload.
func Verify(code string, publicKey ed25519.PublicKey) (Payload, error) {
	var payload Payload

	parts := strings.Split(strings.TrimSpace(code), ".")
	if len(parts) != 3 || parts[0] != Prefix {
		return payload, ErrMalformed
	}

	signature, err := b64.DecodeString(parts[2])
	if err != nil || len(signature) != ed25519.SignatureSize {
		return payload, ErrMalformed
	}
	if len(publicKey) != ed25519.PublicKeySize ||
		!ed25519.Verify(publicKey, []byte(parts[0]+"."+parts[1]), signature) {
		return payload, ErrSignatureInvalid
	}

	data, err := b64.DecodeString(parts[1])
	if err != nil {
		return payload, ErrMalformed
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return payload, ErrMalformed
	}
	return payload, payload.Validate()
}

// Decode returns the payload without checking the signature, for display in
// vendor tools only. Never use it to decide what a customer may use.
func Decode(code string) (Payload, error) {
	var payload Payload
	parts := strings.Split(strings.TrimSpace(code), ".")
	if len(parts) != 3 || parts[0] != Prefix {
		return payload, ErrMalformed
	}
	data, err := b64.DecodeString(parts[1])
	if err != nil {
		return payload, ErrMalformed
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return payload, ErrMalformed
	}
	return payload, nil
}
