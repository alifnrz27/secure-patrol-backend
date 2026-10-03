package license

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func testPayload() Payload {
	issued := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	return Payload{
		LicenseID: "LIC-TEST-1", Customer: "PT Uji", InstallID: "SP-AAAA-BBBB-CCCC-DDDD",
		MaxUnits: 5, MaxAppClients: 3, IssuedAt: issued, ExpiresAt: issued.AddDate(1, 0, 0), GraceDays: 14,
	}
}

func TestSignVerifyRoundTrip(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	code, err := Sign(testPayload(), priv)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(code, "SPL1.") || strings.Count(code, ".") != 2 {
		t.Fatalf("unexpected format: %s", code)
	}
	got, err := Verify(code, pub)
	if err != nil {
		t.Fatal(err)
	}
	if got != testPayload() {
		t.Fatalf("payload changed: %+v", got)
	}
	if got.GraceUntil() != testPayload().ExpiresAt.AddDate(0, 0, 14) {
		t.Fatal("grace until")
	}
}

func TestTamperingIsRejected(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	code, _ := Sign(testPayload(), priv)
	parts := strings.Split(code, ".")

	// Raise the unit limit without re-signing.
	forged := testPayload()
	forged.MaxUnits = 1000
	_, otherPriv, _ := ed25519.GenerateKey(rand.Reader)
	forgedCode, _ := Sign(forged, otherPriv)
	forgedPayload := strings.Split(forgedCode, ".")[1]

	cases := map[string]string{
		"changed payload":       parts[0] + "." + forgedPayload + "." + parts[2],
		"signed with other key": forgedCode,
		"flipped signature":     parts[0] + "." + parts[1] + "." + flip(parts[2]),
	}
	for name, candidate := range cases {
		if _, err := Verify(candidate, pub); !errors.Is(err, ErrSignatureInvalid) {
			t.Errorf("%s: want ErrSignatureInvalid, got %v", name, err)
		}
	}

	for _, bad := range []string{"", "abc", "SPL2." + parts[1] + "." + parts[2], parts[0] + "." + parts[1], code + ".x"} {
		if _, err := Verify(bad, pub); !errors.Is(err, ErrMalformed) {
			t.Errorf("%q: want ErrMalformed, got %v", bad, err)
		}
	}
}

func TestValidate(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	for name, change := range map[string]func(*Payload){
		"no lid":           func(p *Payload) { p.LicenseID = " " },
		"no install id":    func(p *Payload) { p.InstallID = "" },
		"zero units":       func(p *Payload) { p.MaxUnits = 0 },
		"zero app clients": func(p *Payload) { p.MaxAppClients = 0 },
		"expiry before":    func(p *Payload) { p.ExpiresAt = p.IssuedAt },
		"negative grace":   func(p *Payload) { p.GraceDays = -1 },
	} {
		p := testPayload()
		change(&p)
		if _, err := Sign(p, priv); !errors.Is(err, ErrPayloadInvalid) {
			t.Errorf("%s: want ErrPayloadInvalid, got %v", name, err)
		}
	}
}

// A code signed by another implementation (the license generator) must verify
// as long as it follows the format: here the payload JSON is written by hand
// with a different key order and time zone.
func TestForeignEncoderCompatible(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	json := `{"max_units":2,"grace_days":7,"lid":"LIC-X","customer":"PT X","install_id":"SP-1111-2222-3333-4444",` +
		`"max_app_clients":1,"issued_at":"2026-10-01T08:00:00+07:00","expires_at":"2027-10-01T08:00:00+07:00","extra":"ignored"}`
	signed := "SPL1." + base64.RawURLEncoding.EncodeToString([]byte(json))
	code := signed + "." + base64.RawURLEncoding.EncodeToString(ed25519.Sign(priv, []byte(signed)))

	got, err := Verify(code, pub)
	if err != nil {
		t.Fatal(err)
	}
	if got.MaxUnits != 2 || got.MaxAppClients != 1 || got.ExpiresAt.UTC().Hour() != 1 {
		t.Fatalf("unexpected payload: %+v", got)
	}
}

func flip(s string) string {
	b := []byte(s)
	if b[0] == 'A' {
		b[0] = 'B'
	} else {
		b[0] = 'A'
	}
	return string(b)
}

func TestEmbeddedPublicKey(t *testing.T) {
	key := PublicKey()
	if len(key) != ed25519.PublicKeySize {
		t.Fatalf("public key has %d bytes", len(key))
	}
	const want = "a37b9a34ae55b5b6f1db443071152ee747eab11f493c6d5735f07b1a80c6bd2d"
	if got := fmt.Sprintf("%x", []byte(key)); got != want {
		t.Fatalf("public key = %s, want %s", got, want)
	}
}
