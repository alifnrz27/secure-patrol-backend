package helper

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"strings"
)

func ExtractToken(authToken string, removePkcs bool) (string, error) {
	textParts := strings.Split(authToken, "-")
	if len(textParts) < 2 {
		return "", errors.New("auth token not valid")
	}

	iv := textParts[0]
	encryptedText := textParts[1]
	key := os.Getenv("MTSEL_SECRET_KEY")
	// key := "disable_service_for_testing_purpose_only_1234567890"

	bKey := []byte(key)
	bIV, err := hex.DecodeString(iv)
	if err != nil {
		return "", err
	}

	cipherTextDecoded, err := hex.DecodeString(encryptedText)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(bKey)
	if err != nil {
		return "", err
	}

	mode := cipher.NewCBCDecrypter(block, bIV)
	mode.CryptBlocks([]byte(cipherTextDecoded), []byte(cipherTextDecoded))

	if removePkcs == true {
		cipherTextDecoded, err = Pkcs7UnpadCBC(cipherTextDecoded, block.BlockSize())
		if err != nil {
			return "", err
		}
	}

	decryptedToken := string(cipherTextDecoded)

	return decryptedToken, nil
}

func DecryptMsisdn(msisdn string) (string, error) {
	// cipherAlgorithm := "aes-256-cbc"
	cipherPassword := os.Getenv("MTSEL_SECRET_KEY")

	if cipherPassword == "" {
		return "", nil
	}

	textParts := strings.Split(msisdn, "-")
	if len(textParts) < 2 {
		return "", nil
	}

	iv, err := hex.DecodeString(textParts[0])
	if err != nil {
		return "", err
	}

	encryptedText, err := hex.DecodeString(strings.Join(textParts[1:], ":"))
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher([]byte(cipherPassword))
	if err != nil {
		return "", err
	}

	mode := cipher.NewCBCDecrypter(block, iv)
	decrypted := make([]byte, len(encryptedText))
	mode.CryptBlocks(decrypted, encryptedText)

	// Remove possible padding
	padLen := int(decrypted[len(decrypted)-1])
	if padLen > 0 && padLen <= len(decrypted) {
		decrypted = decrypted[:len(decrypted)-padLen]
	}

	return string(decrypted), nil
}

func ExtractTokenGCM(encryptedData string, removePkcs bool) (string, error) {
	cipherPassword := os.Getenv("MTSEL_SECRET_KEY_GCM")
	// cipherPassword := "abc"
	parts := strings.Split(encryptedData, "-")
	if len(parts) != 3 {
		return "", errors.New("invalid encrypted data format")
	}

	iv, err := hex.DecodeString(parts[0])
	if err != nil {
		return "", err
	}

	encryptedText, err := hex.DecodeString(parts[1])
	if err != nil {
		return "", err
	}

	authTag, err := hex.DecodeString(parts[2])
	if err != nil {
		return "", err
	}

	key := []byte(cipherPassword)

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	// gcm, err := cipher.NewGCM(block)
	// if err != nil {
	// 	return "", err
	// }
	// ⭐ FIX: use custom nonce size (match Node.js IV length)
	gcm, err := cipher.NewGCMWithNonceSize(block, len(iv))
	if err != nil {
		return "", err
	}

	// In Go AES-GCM, authTag is appended to ciphertext
	ciphertextWithTag := append(encryptedText, authTag...)

	plaintext, err := gcm.Open(nil, iv, ciphertextWithTag, nil)
	if err != nil {
		return "", err
	}

	if removePkcs == true {
		plaintext = Pkcs7UnpadGCM(plaintext)
	}

	return string(plaintext), nil
}

func Pkcs7UnpadGCM(data []byte) []byte {
	if len(data) == 0 {
		return data
	}

	padLen := int(data[len(data)-1])

	// invalid size
	if padLen == 0 || padLen > len(data) {
		return data
	}

	// check all padding bytes match
	for i := len(data) - padLen; i < len(data); i++ {
		if int(data[i]) != padLen {
			return data
		}
	}

	return data[:len(data)-padLen]
}

func Pkcs7UnpadCBC(data []byte, blockSize int) ([]byte, error) {
	if len(data) == 0 {
		return nil, errors.New("pkcs7: empty data")
	}

	if len(data)%blockSize != 0 {
		return nil, errors.New("pkcs7: data not multiple of block size")
	}

	padLen := int(data[len(data)-1])

	// padding length must be valid
	if padLen == 0 || padLen > blockSize || padLen > len(data) {
		return nil, errors.New("pkcs7: invalid padding length")
	}

	// verify all padding bytes
	for i := len(data) - padLen; i < len(data); i++ {
		if data[i] != byte(padLen) {
			return nil, errors.New("pkcs7: invalid padding content")
		}
	}

	return data[:len(data)-padLen], nil
}

// ============================== TEMPLATE JS AUTH ==============================
// const Crypto = require('crypto')
// const Moment = require('moment');

// const decryptMsisdn = (msisdn) => {
//   try {
//     const cipherAlgorithm =  'aes-256-cbc';
//     const cipherPassword = os.Getenv("MTSEL_SECRET_KEY");

//     const textParts = msisdn.split('-');
//     const iv = Buffer.from(textParts.shift(), 'hex');
//     const encryptedText = Buffer.from(textParts.join(':'), 'hex');
//     const decipher = Crypto.createDecipheriv(cipherAlgorithm, Buffer.from(cipherPassword), iv);
//     let decrypted = decipher.update(encryptedText);

//     decrypted = Buffer.concat([decrypted, decipher.final()]);

//     return decrypted.toString();
//   } catch (err) {
//     return null;
//   }
// };

// const decryptCustParams = (custParam) => {
//   try {
//     const cipherAlgorithm =  'aes-256-cbc';
//     const cipherPassword = os.Getenv("MTSEL_SECRET_KEY");

//     const textParts = custParam.split('-');
//     const iv = Buffer.from(textParts.shift(), 'hex');
//     const encryptedText = Buffer.from(textParts.join(':'), 'hex');
//     const decipher = Crypto.createDecipheriv(cipherAlgorithm, Buffer.from(cipherPassword), iv);
//     let decrypted = decipher.update(encryptedText);

//     decrypted = Buffer.concat([decrypted, decipher.final()]);

//     return decrypted.toString();
//   } catch (err) {
//     return null;
//   }
// };

// let custParam = 'b1d838593724504510c88a6914ea46dc-2fbe0071d4dec0f2ea26ffe680704aecc210968205dd86026b1317140d8a99246770a46ab81b4ecf5a203dd9751bb8c9aOde29c80b6cad462051f4accd83b883a447dd2ae9b4f06d7afbbce96a47b0c5697602b9c6cf7ebac4c66faebb0fc85d217268d368fa4071872f9e014f025b5ecbe8bd81cfe0323bfc38dc29b3b3d261e8498cb608075beaa9c2f38a4b219298'
// console.log(custParam)
// let decCustParam = decryptCustParams(custParam)
// console.log(decCustParam)
// let decMsisdn = decryptMsisdn(decCustParam.split('|')[0])
// console.log(decMsisdn)
// ============================== TEMPLATE JS AUTH ==============================

func GenerateMsisdnHash(msisdn string) string {
	hash := sha256.Sum256([]byte(msisdn))
	return hex.EncodeToString(hash[:])
}
