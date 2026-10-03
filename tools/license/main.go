// Command license is the vendor tool to create Secure Patrol licenses. It is
// NOT part of the server binary or image, and the private key must never be
// placed in this repository, on a customer server, or in a frontend.
//
//	go run ./tools/license keygen -out ~/.secure-patrol/license_private.key
//	go run ./tools/license issue -key ~/.secure-patrol/license_private.key \
//	    -install-id SP-XXXX-XXXX-XXXX-XXXX -customer "PT Contoh" -units 5 -app-clients 3 \
//	    -expires 2027-10-01 -grace 14
//	go run ./tools/license decode <code>
//
// The private key can also be given as LICENSE_PRIVATE_KEY (hex of the 32 byte seed).
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"secure-patrol-backend/pkg/license"
	"strings"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	switch os.Args[1] {
	case "keygen":
		keygen(os.Args[2:])
	case "issue":
		issue(os.Args[2:])
	case "decode":
		decode(os.Args[2:])
	default:
		usage()
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: license keygen|issue|decode (see the comment in tools/license/main.go)")
	os.Exit(2)
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func keygen(args []string) {
	flags := flag.NewFlagSet("keygen", flag.ExitOnError)
	out := flags.String("out", "", "file to write the private key seed to (required, must not exist)")
	flags.Parse(args)
	if *out == "" {
		flags.Usage()
		os.Exit(2)
	}

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		fail("keygen: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(*out), 0o700); err != nil {
		fail("keygen: %v", err)
	}
	file, err := os.OpenFile(*out, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		fail("keygen: %v (an existing key is never overwritten)", err)
	}
	defer file.Close()
	if _, err := fmt.Fprintln(file, hex.EncodeToString(priv.Seed())); err != nil {
		fail("keygen: %v", err)
	}

	fmt.Printf("Private key written to %s (keep it secret, back it up).\n", *out)
	fmt.Printf("Public key (put it in pkg/license/key.go): %s\n", hex.EncodeToString(pub))
}

func loadPrivateKey(path string) ed25519.PrivateKey {
	seedHex := os.Getenv("LICENSE_PRIVATE_KEY")
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			fail("read private key: %v", err)
		}
		seedHex = string(data)
	}
	seed, err := hex.DecodeString(strings.TrimSpace(seedHex))
	if err != nil || len(seed) != ed25519.SeedSize {
		fail("private key must be the hex of a 32 byte Ed25519 seed (use -key or LICENSE_PRIVATE_KEY)")
	}
	return ed25519.NewKeyFromSeed(seed)
}

func parseDate(value string) time.Time {
	jakarta, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		jakarta = time.FixedZone("WIB", 7*3600)
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02"} {
		if t, err := time.ParseInLocation(layout, value, jakarta); err == nil {
			return t
		}
	}
	fail("invalid date %q, use YYYY-MM-DD or RFC3339", value)
	return time.Time{}
}

func issue(args []string) {
	flags := flag.NewFlagSet("issue", flag.ExitOnError)
	keyFile := flags.String("key", "", "private key file (or LICENSE_PRIVATE_KEY)")
	lid := flags.String("lid", "", "license id (default LIC-<date>-<random>)")
	customer := flags.String("customer", "", "customer name (required)")
	installID := flags.String("install-id", "", "install id shown by `secure-patrol-backend license-info` (required)")
	units := flags.Int("units", 0, "maximum active units (required)")
	appClients := flags.Int("app-clients", 0, "maximum active app clients / app keys (required)")
	expires := flags.String("expires", "", "expiry date YYYY-MM-DD (end of that day is not included) or RFC3339 (required)")
	grace := flags.Int("grace", 14, "grace days after expiry")
	issued := flags.String("issued", "", "issue date (default now); YYYY-MM-DD or RFC3339")
	flags.Parse(args)

	if *customer == "" || *installID == "" || *units == 0 || *appClients == 0 || *expires == "" {
		flags.Usage()
		os.Exit(2)
	}
	if *lid == "" {
		suffix := make([]byte, 3)
		rand.Read(suffix)
		*lid = fmt.Sprintf("LIC-%s-%s", time.Now().Format("20060102"), strings.ToUpper(hex.EncodeToString(suffix)))
	}

	issuedAt := time.Now().Truncate(time.Second)
	if *issued != "" {
		issuedAt = parseDate(*issued)
	}

	code, err := license.Sign(license.Payload{
		LicenseID:     *lid,
		Customer:      *customer,
		InstallID:     strings.TrimSpace(*installID),
		MaxUnits:      *units,
		MaxAppClients: *appClients,
		IssuedAt:      issuedAt,
		ExpiresAt:     parseDate(*expires),
		GraceDays:     *grace,
	}, loadPrivateKey(*keyFile))
	if err != nil {
		fail("issue: %v", err)
	}
	fmt.Println(code)
}

func decode(args []string) {
	if len(args) != 1 {
		fail("usage: license decode <code>")
	}
	payload, err := license.Decode(args[0])
	if err != nil {
		fail("decode: %v", err)
	}
	out, _ := json.MarshalIndent(payload, "", "  ")
	fmt.Println(string(out))
	fmt.Fprintln(os.Stderr, "(content only; the signature is checked by the server)")
}
