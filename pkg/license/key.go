package license

import "crypto/ed25519"

// The vendor's Ed25519 public key. It only verifies licenses; codes can only be
// created with the matching private key, which is never in this repository.
// The key is stored in masked pieces so it is not a single searchable value in
// the binary. To change it, run `go run ./tools/license keygen` and replace the
// pieces (every license issued with the old key stops working).
var keyParts = [4][8]byte{
	{0x8a, 0x09, 0x21, 0x30, 0xe3, 0xc3, 0x6a, 0x9e},
	{0x80, 0x61, 0x47, 0x7c, 0xe4, 0xcb, 0x09, 0x97},
	{0xfe, 0xe8, 0xfa, 0x8b, 0x94, 0x1a, 0x02, 0xef},
	{0x34, 0xba, 0xe8, 0xc6, 0xa5, 0xa8, 0x0a, 0x2d},
}

// PublicKey returns the vendor public key.
func PublicKey() ed25519.PublicKey {
	key := make(ed25519.PublicKey, 0, ed25519.PublicKeySize)
	for i, part := range keyParts {
		for j, value := range part {
			n := i*8 + j
			key = append(key, value^byte(n*73+41))
		}
	}
	return key
}
