package crypto_aux

// SPDX-LICENSE-IDENTIFIER: 3-Clauses-BSD

import "encoding/base64"

// Base64Deserialize turns bytes series of public key into base64 string
func Base64Deserialize(inpByte []byte) string {
	return base64.StdEncoding.EncodeToString(inpByte)
}

// Base64Serialize turns base64 string of public key into bytes and attach an error if any
func Base64Serialize(inpStr string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(inpStr)
}
