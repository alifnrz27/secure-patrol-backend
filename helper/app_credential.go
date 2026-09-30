package helper

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"os"
	"strings"
)

// App credential algorithm
//
// App ID  : sp_<platform>_<16 chars random base32>_<6 chars checksum>
//           e.g. sp_and_k3m9qx2ht5vbn4rw_w8xq2a
//           The checksum is an HMAC of the prefix with a key derived from APP_MASTER_SECRET,
//           so malformed or guessed IDs are rejected before touching the database.
// App Key : spk_<43 chars base64url of 32 random bytes>. Shown only once, stored
//           encrypted with AES-256-GCM using a key derived from APP_MASTER_SECRET.
// Request : every request is signed with
//           HMAC-SHA256(app_key, METHOD \n REQUEST_URI \n TIMESTAMP \n NONCE \n SHA256_HEX(BODY))
//           and sent as lowercase hex in the X-Signature header.

const (
	appIDPrefix      = "sp"
	appKeyPrefix     = "spk_"
	appIDRandomLen   = 16
	appIDChecksumLen = 6
)

var ErrAppCredentialInvalid = errors.New("app credential is invalid")

// AppPlatformCodes maps a platform to the short code embedded in the App ID.
var AppPlatformCodes = map[string]string{
	"android": "and",
	"ios":     "ios",
	"web":     "web",
	"server":  "srv",
}

var base32NoPad = base32.StdEncoding.WithPadding(base32.NoPadding)

func deriveAppKey(label string) []byte {
	mac := hmac.New(sha256.New, []byte(os.Getenv("APP_MASTER_SECRET")))
	mac.Write([]byte(label))
	return mac.Sum(nil)
}

func GenerateAppID(platform string) (string, error) {
	code, ok := AppPlatformCodes[platform]
	if !ok {
		return "", errors.New("platform is not supported")
	}

	buf := make([]byte, 10) // 10 bytes = 16 base32 chars
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	body := appIDPrefix + "_" + code + "_" + strings.ToLower(base32NoPad.EncodeToString(buf))
	return body + "_" + appIDChecksum(body), nil
}

// ValidateAppIDFormat checks the structure and checksum of an App ID without a database lookup.
func ValidateAppIDFormat(appID string) bool {
	parts := strings.Split(appID, "_")
	if len(parts) != 4 || parts[0] != appIDPrefix || len(parts[2]) != appIDRandomLen {
		return false
	}

	validPlatform := false
	for _, code := range AppPlatformCodes {
		if parts[1] == code {
			validPlatform = true
			break
		}
	}
	if !validPlatform {
		return false
	}

	body := strings.Join(parts[:3], "_")
	return hmac.Equal([]byte(appIDChecksum(body)), []byte(parts[3]))
}

func appIDChecksum(body string) string {
	mac := hmac.New(sha256.New, deriveAppKey("app-id-checksum"))
	mac.Write([]byte(body))
	return strings.ToLower(base32NoPad.EncodeToString(mac.Sum(nil)))[:appIDChecksumLen]
}

func GenerateAppKey() (string, error) {
	token, err := RandomURLToken(32)
	if err != nil {
		return "", err
	}
	return appKeyPrefix + token, nil
}

// AppKeyHint returns the last 4 characters of the key so admins can tell keys apart.
func AppKeyHint(appKey string) string {
	if len(appKey) < 4 {
		return ""
	}
	return appKey[len(appKey)-4:]
}

func EncryptAppKey(appKey string) (string, error) {
	gcm, err := appKeyCipher()
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}

	sealed := gcm.Seal(nonce, nonce, []byte(appKey), nil)
	return base64.RawStdEncoding.EncodeToString(sealed), nil
}

func DecryptAppKey(encrypted string) (string, error) {
	gcm, err := appKeyCipher()
	if err != nil {
		return "", err
	}

	data, err := base64.RawStdEncoding.DecodeString(encrypted)
	if err != nil || len(data) < gcm.NonceSize() {
		return "", ErrAppCredentialInvalid
	}

	plain, err := gcm.Open(nil, data[:gcm.NonceSize()], data[gcm.NonceSize():], nil)
	if err != nil {
		return "", ErrAppCredentialInvalid
	}

	return string(plain), nil
}

func appKeyCipher() (cipher.AEAD, error) {
	block, err := aes.NewCipher(deriveAppKey("app-key-encryption"))
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func BuildSignaturePayload(method string, requestURI string, timestamp string, nonce string, body []byte) string {
	return strings.Join([]string{
		strings.ToUpper(method),
		requestURI,
		timestamp,
		nonce,
		SHA256Hex(body),
	}, "\n")
}

func SignPayload(appKey string, payload string) string {
	mac := hmac.New(sha256.New, []byte(appKey))
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

func VerifyPayloadSignature(appKey string, payload string, signature string) bool {
	expected := SignPayload(appKey, payload)
	return hmac.Equal([]byte(expected), []byte(strings.ToLower(signature)))
}
