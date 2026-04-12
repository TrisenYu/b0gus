package crypto_aux

// SPDX-LICENSE-IDENTIFIER: 3-Clauses-BSD

import "encoding/base64"

// Base64Convert turns bytes series into a base64 string
func Base64Convert(inpByte []byte) string {
	return base64.StdEncoding.EncodeToString(inpByte)
}

// Base64Recover recovers a base64 string as its original bytes and attach an error if any
func Base64Recover(inpStr string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(inpStr)
}
